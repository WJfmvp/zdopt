package Timer

import (
	"errors"
	"fmt"
	"sync"
	"zdopt/ZdoptServer/Actor"
	"zdopt/ZdoptServer/Logs"
)

// 定义错误类型
var (
	ErrTimerAlreadyRunning    = errors.New("timer already running")
	ErrNoKeyFrames            = errors.New("no key frames added")
	ErrInvalidTimerParameters = errors.New("invalid timer parameters")
	ErrActorNotSet            = errors.New("actor not initialized")
)

// ZTimer 结构体表示一个定时器
type ZTimer struct {
	TimerId      int
	MyActorBase  *Actor.BaseActor
	_keyFrames   []*KeyFrame
	overCallback func(*ZTimer)
	logger       *Logs.ZLogger
	maxTimer     float32
	IsLoop       bool
	isRun        bool
	currentTimer float32
	OffsetTime   float32
	mu           sync.RWMutex // 读写锁保护并发访问
	stopChan     chan struct{}
}

// NewZTimer 创建定时器实例（带参数验证）
func NewZTimer(offsetTime float32) (*ZTimer, error) {
	if offsetTime <= 0 {
		return nil, fmt.Errorf("%w: offset time must be positive", ErrInvalidTimerParameters)
	}

	// 初始化日志器（处理错误）
	logger, err := Logs.NewZLogger("ZTimer", Logs.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return &ZTimer{
		OffsetTime: offsetTime,
		logger:     logger,
		_keyFrames: make([]*KeyFrame, 0),
		stopChan:   make(chan struct{}, 1),
	}, nil
}

// AddKeyFrame 增强版关键帧添加（带参数验证和状态检查）
func (zt *ZTimer) AddKeyFrame(time float32, action func()) error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if zt.isRun {
		return fmt.Errorf("cannot add keyframe: %w", ErrTimerAlreadyRunning)
	}

	if time <= 0 {
		return fmt.Errorf("%w: keyframe time must be positive", ErrInvalidTimerParameters)
	}

	if action == nil {
		return fmt.Errorf("%w: keyframe action cannot be nil", ErrInvalidTimerParameters)
	}

	kf, err := GetKeyFrame(time, action)
	if err != nil {
		return fmt.Errorf("failed to acquire keyframe: %w", err)
	}

	kf.Set(time, action)
	zt._keyFrames = append(zt._keyFrames, kf)

	zt.logger.Debug(fmt.Sprintf("KeyFrame added at %.2fs", time))
	return nil
}

// Start 增强版启动逻辑（带多重状态验证）
func (zt *ZTimer) Start(actor *Actor.BaseActor) error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if zt.isRun {
		return ErrTimerAlreadyRunning
	}

	if actor == nil {
		return ErrActorNotSet
	}

	if len(zt._keyFrames) == 0 {
		return ErrNoKeyFrames
	}

	// 初始化计时器状态
	zt.MyActorBase = actor
	zt.currentTimer = 0
	zt.isRun = true

	// 计算最大关键帧时间
	zt.maxTimer = 0
	for _, frame := range zt._keyFrames {
		if frame.Time > zt.maxTimer {
			zt.maxTimer = frame.Time
		}
	}

	// 调用 StartTimer
	if err := zt.StartTimer(); err != nil {
		zt.isRun = false // 回滚状态
		return fmt.Errorf("actor timer startup failed: %w", err)
	}

	zt.logger.Info(fmt.Sprintf("Timer %d started with %d keyframes", zt.TimerId, len(zt._keyFrames)))
	return nil
}

// Update 线程安全更新逻辑
// Update 增强版更新逻辑
func (zt *ZTimer) Update(deltaTime float32) {
	zt.mu.RLock()
	defer zt.mu.RUnlock()

	if !zt.isRun || deltaTime <= 0 {
		return
	}

	select {
	case <-zt.stopChan:
		_ = zt.StopTimer()
		return
	default:
	}

	zt.currentTimer += deltaTime

	// 处理定时器循环/终止
	if zt.currentTimer > zt.maxTimer+zt.OffsetTime {
		if zt.IsLoop {
			zt.currentTimer -= zt.maxTimer
			zt.resetKeyFrames()
			zt.logger.Debug("Timer loop reset")
		} else {
			go zt.safeStop()
		}
		return
	}

	// 触发关键帧
	for _, kf := range zt._keyFrames {
		if !kf.IsTriggered() && zt.currentTimer >= kf.Time-zt.OffsetTime {
			kf.Trigger()
			zt.logger.Debug(fmt.Sprintf("KeyFrame triggered at %.2fs", kf.Time))
		}
	}
}

// StartTimer 实现Actor接口的启动方法
func (zt *ZTimer) StartTimer() error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if zt.MyActorBase == nil {
		return errors.New("actor base not initialized")
	}

	// 实际启动逻辑（示例）
	zt.logger.Debug("Starting underlying timer mechanism")
	return nil
}

// StopTimer 完整停止方法
func (zt *ZTimer) StopTimer() error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	if !zt.isRun {
		return nil
	}

	zt.logger.Debug("Initiating stop sequence")

	zt.safeStop()

	// 释放资源
	if err := zt.releaseResources(); err != nil {
		return fmt.Errorf("resource release failed: %w", err)
	}

	zt.isRun = false
	zt.logger.Debug("Timer stopped successfully")
	return nil
}

// 私有辅助方法
// --------------------------

// safeStop 安全停止定时器
func (zt *ZTimer) safeStop() {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	select {
	case zt.stopChan <- struct{}{}:
		zt.logger.Debug("Stop signal sent")
	default:
		zt.logger.Warn("Stop channel is full")
	}
}

// resetKeyFrames 重置所有关键帧状态
func (zt *ZTimer) resetKeyFrames() {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	for _, kf := range zt._keyFrames {
		kf.Reset()
	}
	zt.logger.Debug("All keyframes reset")
}

// releaseResources 释放所有资源
func (zt *ZTimer) releaseResources() error {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	var errs []error
	for _, kf := range zt._keyFrames {
		if err := ReleaseKeyFrame(kf); err != nil {
			errs = append(errs, fmt.Errorf("keyframe release failed: %w", err))
		}
	}
	zt._keyFrames = nil

	if len(errs) > 0 {
		return fmt.Errorf("resource cleanup errors: %v", errs)
	}
	zt.logger.Debug("All resources released")
	return nil
}

// 状态查询方法
// --------------------------

func (zt *ZTimer) IsRunning() bool {
	zt.mu.RLock()
	defer zt.mu.RUnlock()
	return zt.isRun
}

func (zt *ZTimer) CurrentProgress() float32 {
	zt.mu.RLock()
	defer zt.mu.RUnlock()
	if zt.maxTimer == 0 {
		return 0
	}
	return zt.currentTimer / zt.maxTimer
}
