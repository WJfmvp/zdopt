package Actor

//balancer.go
import (
	"golang.org/x/net/context"
	"runtime"
	"sync/atomic"
)

// Balancer 带动态扩容的工作负载均衡器
type Balancer struct {
	workers []*worker
	index   uint64
	ctx     context.Context
}

type worker struct {
	ch     chan func()
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBalancer 创建负载均衡器，自动匹配CPU核心数
func NewBalancer(ctx context.Context) *Balancer {
	numCPU := runtime.NumCPU()
	b := &Balancer{
		workers: make([]*worker, numCPU),
		ctx:     ctx,
	}
	// 初始化worker池
	for i := range b.workers {
		w := &worker{
			ch: make(chan func(), 1024),
		}
		w.ctx, w.cancel = context.WithCancel(ctx)
		go w.run()
		b.workers[i] = w
	}
	return b
}

// Submit 提交任务，使用轮询策略 + 动态扩容
func (b *Balancer) Submit(task func()) {
	if task == nil {
		return
	}

	//轮询选择worker
	idx := atomic.AddUint64(&b.index, 1) % uint64(len(b.workers))
	select {
	case b.workers[idx].ch <- task:
	default:
		//触发动态扩容
		newworker := b.expandWorkers()
		newworker.ch <- task // 重试提交
	}
}

// expandWorkers 扩容worker池（增加10%）
func (b *Balancer) expandWorkers() *worker {
	newSize := len(b.workers) + len(b.workers)/10
	if newSize > runtime.NumCPU()*10 {
		newSize = runtime.NumCPU() * 10
	}
	// 实际实现应考虑最大限制和收缩策略
	newWorker := &worker{
		ch: make(chan func(), 1024),
	}
	newWorker.ctx, newWorker.cancel = context.WithCancel(b.ctx)
	go newWorker.run()

	b.workers = append(b.workers, newWorker)
	return newWorker
}

// worker 执行循环
func (w *worker) run() {
	defer w.cancel()
	for {
		select {
		case task := <-w.ch:
			task()
		case <-w.ctx.Done():
			return
		}
	}
}
