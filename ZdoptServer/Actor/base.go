package Actor

//base.go
import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Actor 基础接口
type Actor interface {
	Init(ctx context.Context)
	Start()
	Stop()
	Update(delta time.Duration)
	Receive(msg interface{})
}
type MessageQueue struct {
	head    uint64
	tail    uint64
	buffer  []interface{}
	modMask uint64
}

type BaseActor struct {
	id       int64
	mailbox  chan interface{}
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	handlers sync.Map // map[string]HandlerFunc
	queue    *MessageQueue
}

// NewBaseActor 创建基础Actor
func NewBaseActor(size uint64) *BaseActor {
	return &BaseActor{
		queue:   NewMessageQueue(size),
		mailbox: make(chan interface{}, 1024),
	}
}

// Init 初始化Actor
func (a *BaseActor) Init(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.wg.Add(1)
	go a.processMessages()
}

// processMessages 消息处理主循环
func (a *BaseActor) processMessages() {
	defer a.wg.Done()
	const batchSize = 64
	msgs := make([]interface{}, 0, batchSize)

	for {
		select {
		case msg := <-a.mailbox:
			msgs = append(msgs, msg)
			if len(msgs) >= batchSize {
				a.batchHandle(msgs)
				msgs = msgs[:0]
			}
		case <-a.ctx.Done():
			a.batchHandle(msgs)
			return
		default:
			if len(msgs) > 0 {
				a.batchHandle(msgs)
				msgs = msgs[:0]
			}
			runtime.Gosched()
		}
	}
}

// batchHandle 批量消息处理
func (a *BaseActor) batchHandle(msgs []interface{}) {
	var wg sync.WaitGroup
	for _, msg := range msgs {
		wg.Add(1)
		go func(m interface{}) {
			defer wg.Done()
			if handler, ok := a.handlers.Load(getMessageType(m)); ok {
				handler.(func(interface{}))(m)
			}
		}(msg)
	}
	wg.Wait()
}

// getMessageType 消息类型获取
func getMessageType(msg interface{}) string {
	return reflect.TypeOf(msg).String()
}

func NewMessageQueue(size uint64) *MessageQueue {
	return &MessageQueue{
		buffer:  make([]interface{}, 0, size),
		modMask: (size - 1) & ^uint64(1),
	}
}

func (q *MessageQueue) Enqueue(msg interface{}) bool {
	tail := atomic.LoadUint64(&q.tail)
	next := tail + 1
	if (next & q.modMask) == atomic.LoadUint64(&q.head) {
		return false
	}
	q.buffer[tail&q.modMask] = msg
	atomic.StoreUint64(&q.tail, next)
	return true
}

func (q *MessageQueue) Dequeue() (interface{}, bool) {
	head := atomic.LoadUint64(&q.head)
	tail := atomic.LoadUint64(&q.tail)
	if head == tail {
		return nil, false
	}
	msg := q.buffer[head&q.modMask]
	atomic.StoreUint64(&q.head, head+1)
	return msg, true
}
