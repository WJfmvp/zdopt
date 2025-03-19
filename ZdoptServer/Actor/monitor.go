package Actor

//monitor.go
import (
	"expvar"
	"log"
	"runtime"
	"sync"
	"time"
)

var (
	actorCount   = expvar.NewInt("actors.count")
	messageQueue = expvar.NewInt("actors.messages")
	//goroutineCount = expvar.NewInt("goroutines.count")
	workerMutex    sync.Mutex
	currentWorkers int
)

func startMonitor() {
	go func() {
		for range time.Tick(5 * time.Second) {
			//更新性能指标updateMetres()
			updateMetrice()
			//动态调整资源adjustResources()
			adjustResources()
		}
	}()
}

func updateMetrice() {
	//获取运行时状态
	numGoRoutines := runtime.NumGoroutine()
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	//设置监控指标
	actorCount.Set(int64(numGoRoutines))
	messageQueue.Set(int64(memStats.Mallocs - memStats.Frees))
}

func adjustResources() {
	// 获取当前的运行时状态
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	// 获取当前消息队列的大小和Actor数量
	numGoRoutines := runtime.NumGoroutine()
	numActors := actorCount.Value()     //假设 actorCount 是存储当前 Actor 数量的expver变量
	numMessages := messageQueue.Value() //假设 messageQueue 存储的是待处理的消息数量

	// 设定阈值和动态调整的策略
	const highLoadThreshold = 1000 // 高负载阈值，消息队列大于该值时触发扩容
	const lowLoadThreshold = 200   // 低负载阈值，消息队列小于该值时触发收缩
	const maxWorkers = 100         // 最大工作线程数
	const minWorkers = 5           // 最小工作线程数

	// 计算当前负载
	loadFacter := float64(numMessages) / float64(numGoRoutines)

	// 调整工作线程数量
	if loadFacter > 5.0 && numActors < maxWorkers { //如果负载数量很高且线程数未达到最大值
		// 增加工作线程
		newWorkers := numGoRoutines + 2
		if newWorkers > maxWorkers {
			newWorkers = maxWorkers
		}
		// 触发动态扩容
		for i := numGoRoutines; i <= newWorkers; i++ {
			go startMonitor() // 启动新的工作线程
		}
		log.Printf("Increasing worker: %d -> %d", numGoRoutines, newWorkers)
	} else if loadFacter < 1.0 && numActors > minWorkers { // 如果负载很低且线程数大于最小值
		// 减少工作线程
		newWorkers := numGoRoutines - 1
		if newWorkers < minWorkers {
			newWorkers = minWorkers
		}
		// 触发动态收缩，关闭不再需要的工作线程
		stopWorkers(numGoRoutines - newWorkers)
		log.Printf("Increasing worker: %d -> %d", numGoRoutines, newWorkers)
	}

	// 调整内存使用情况
	if memStats.Mallocs-memStats.Frees > 1024*1024*100 { // 如果分配的内存大于 100MB
		// 假设需要进行资源释放（收回未使用的对象池资源）
		cleanupMemory()
		log.Println("Memory cleanup triggered")
	}

	// 根据消息处理量调整资源
	if numMessages > highLoadThreshold {
		// 如果消息数量过高，可以触发额外资源的扩展
		log.Println("High load detected, expanding resources...")
		// 此处可以扩展额外的网络带宽、数据库连接池等
	}
}

// 停止不再需要的工作线程
func stopWorkers(count int) {
	workerMutex.Lock()
	defer workerMutex.Unlock()

	for i := 0; i < count; i++ {
		// 优雅关闭线程的实现
		log.Printf("Stopping worker %d", currentWorkers)
		currentWorkers--
		// 可以通过控制某种信号来中断工作线程的执行（使用 context、channel 或其他方式来通知工作线程退出）
	}
	log.Printf("Stopped workers, current count: %d", currentWorkers)
}

/*// 启动一个新的工作线程
func startWorker() {
	workerMutex.Lock()
	defer workerMutex.Unlock()
	currentWorkers++

	go func() {
		// 模拟工作线程的处理逻辑
		for {
			select {
			case <-time.After(10 * time.Second): // 每10秒进行一次任务
				// 执行实际任务
				log.Println("Worker executing task")
			}
		}
	}()
	log.Printf("Started new worker, current count: %d", currentWorkers)
}*/

// 清理内存，释放未使用的资源
func cleanupMemory() {
	// 释放未使用的内存或清理对象池等
	// 清理消息池或其他资源
	messagePool.New = nil
}
