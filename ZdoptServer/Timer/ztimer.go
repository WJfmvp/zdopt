package Timer

import (
	"fmt"
	"sync"
	"time"
	"zdopt/ZdoptServer/Actor"
	"zdopt/ZdoptServer/Error"
	"zdopt/ZdoptServer/Logs"
)

type ZTimer struct {
	TimerId      int
	MyActorBase  *Actor.BaseActor
	_keyFrames   []*KeyFrame
	logger       *Logs.ZLogger
	maxTimer     float32
	IsLoop       bool
	isRun        bool
	currentTimer float32
	OffsetTime   float32
	mu           sync.RWMutex
	stopChan     chan struct{}
	updateTicker *time.Ticker
	asyncWG      sync.WaitGroup
	lastUpdate   time.Time
}

func NewZTimer(offsetTime float32) (*ZTimer, error) {
	if offsetTime <= 0 {
		return nil, fmt.Errorf("%w: offset time must be positive", Error.ErrInvalidTimerParameters)
	}

	logger, err := Logs.NewZLogger("ZTimer", Logs.Info)
	if err != nil {
		return nil, fmt.Errorf("logger initialization failed: %w", err)
	}

	return &ZTimer{
		OffsetTime:   offsetTime,
		logger:       logger,
		_keyFrames:   make([]*KeyFrame, 0),
		stopChan:     make(chan struct{}),
		updateTicker: time.NewTicker(16 * time.Millisecond),
		lastUpdate:   time.Now(), // 初始化时间记录
	}, nil
}

// AddKeyFrame 增强参数校验
func (zt *ZTimer) AddKeyFrame(time float32, action func()) error {
	if time <= 0 {
		return fmt.Errorf("%w: keyframe time must be positive", Error.ErrInvalidTimerParameters)
	}
	if action == nil {
		return Error.ErrNilKeyFrameAction
	}

	zt.mu.Lock()
	defer zt.mu.Unlock()

	if zt.isRun {
		return Error.ErrTimerAlreadyRunning
	}

	kf, err := GetKeyFrame(time, action)
	if err != nil {
		return fmt.Errorf("acquire keyframe failed: %w", err)
	}

	zt._keyFrames = append(zt._keyFrames, kf)
	zt.logger.Debug(fmt.Sprintf("Keyframe added at %.2fs", time))
	return nil
}

// Start 增加状态校验和资源预检查
func (zt *ZTimer) Start(actor *Actor.BaseActor) error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if zt.isRun {
		return Error.ErrTimerAlreadyRunning
	}
	if actor == nil {
		return Error.ErrActorNotSet
	}
	if len(zt._keyFrames) == 0 {
		return Error.ErrNoKeyFrames
	}

	// 初始化计时状态
	zt.MyActorBase = actor
	zt.currentTimer = 0
	zt.isRun = true
	zt.lastUpdate = time.Now()

	// 计算最大时间（带校验）
	zt.maxTimer = 0
	for _, f := range zt._keyFrames {
		if f.Time > zt.maxTimer {
			zt.maxTimer = f.Time
		}
	}
	if zt.maxTimer <= 0 {
		return fmt.Errorf("%w: invalid max timer value", Error.ErrInvalidTimerParameters)
	}

	// 启动异步循环
	zt.asyncWG.Add(1)
	go zt.asyncUpdateLoop()
	zt.logger.Info(fmt.Sprintf("Timer %d started with %d keyframes", zt.TimerId, len(zt._keyFrames)))
	return nil
}

// asyncUpdateLoop 时间循环（增加恢复机制）
func (zt *ZTimer) asyncUpdateLoop() {
	defer zt.asyncWG.Done()
	defer func() {
		if r := recover(); r != nil {
			zt.logger.Error(fmt.Sprintf("Update loop panic: %v", r))
		}
	}()

	for {
		select {
		case <-zt.updateTicker.C:
			now := time.Now()
			delta := float32(now.Sub(zt.lastUpdate).Seconds())
			zt.lastUpdate = now

			zt.mu.RLock()
			if zt.isRun {
				zt.Update(delta)
			}
			zt.mu.RUnlock()

		case <-zt.stopChan:
			zt.updateTicker.Stop()
			zt.logger.Debug("Update loop exited")
			return
		}
	}
}

// safeStop 安全停止方法（幂等操作）
func (zt *ZTimer) safeStop() {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if zt.isRun {
		zt.logger.Debug("Initiating safe stop")
		select {
		case zt.stopChan <- struct{}{}:
		default:
		}
		zt.isRun = false
	}
}

// Update 增加执行边界保护
func (zt *ZTimer) Update(deltaTime float32) {
	defer func() {
		if r := recover(); r != nil {
			zt.logger.Error(fmt.Sprintf("Update panic: %v", r))
		}
	}()

	zt.mu.RLock()
	defer zt.mu.RUnlock()

	if !zt.isRun || deltaTime <= 0 {
		return
	}

	zt.currentTimer += deltaTime

	// 循环逻辑（带阈值保护）
	if zt.currentTimer > zt.maxTimer+zt.OffsetTime {
		if zt.IsLoop {
			zt.currentTimer -= zt.maxTimer
			go zt.asyncResetKeyFrames()
			zt.logger.Debug("Timer loop reset")
		} else {
			go zt.safeStop()
		}
		return
	}

	// 并行触发（带错误收集）
	var wg sync.WaitGroup
	errChan := make(chan error, len(zt._keyFrames))

	for _, kf := range zt._keyFrames {
		if !kf.IsTriggered() && zt.currentTimer >= kf.Time-zt.OffsetTime {
			wg.Add(1)
			go func(f *KeyFrame) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errChan <- fmt.Errorf("keyframe panic: %v", r)
					}
				}()

				f.Trigger()
				zt.logger.Debug(fmt.Sprintf("Triggered at %.2fs", f.Time))
			}(kf)
		}
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 处理错误
	for err := range errChan {
		zt.logger.Error(err.Error())
	}
}

// asyncResetKeyFrames 带保护的异步重置
func (zt *ZTimer) asyncResetKeyFrames() {
	defer func() {
		if r := recover(); r != nil {
			zt.logger.Error(fmt.Sprintf("Reset panic: %v", r))
		}
	}()

	zt.mu.Lock()
	defer zt.mu.Unlock()

	for _, kf := range zt._keyFrames {
		kf.Reset()
	}
}

// StopTimer 完善的停止逻辑
func (zt *ZTimer) StopTimer() error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if !zt.isRun {
		return nil
	}

	zt.logger.Debug("Initiating stop sequence")
	zt.isRun = false

	// 关闭通道并等待
	close(zt.stopChan)
	zt.asyncWG.Wait()

	// 释放资源
	var errs []error
	for _, kf := range zt._keyFrames {
		if err := ReleaseKeyFrame(kf); err != nil {
			errs = append(errs, fmt.Errorf("release error: %w", err))
		}
	}
	zt._keyFrames = nil

	// 返回组合错误
	if len(errs) > 0 {
		return fmt.Errorf("stop completed with errors: %v", errs)
	}
	zt.logger.Info("Timer stopped successfully")
	return nil
}

func (zt *ZTimer) IsRunning() bool {
	zt.mu.RLock()
	defer zt.mu.RUnlock()
	return zt.isRun
}
