package Timer

import (
	"errors"
	"log"
	"zdopt/ZdoptServer/Error"
)

// Set 安全设置关键帧参数
func (kf *KeyFrame) Set(time float32, action func()) error {
	if time <= 0 {
		return errors.New("invalid keyframe time (must be positive)")
	}
	if action == nil {
		return Error.ErrNilKeyFrameAction
	}

	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.Time = time
	kf.Action = action
	kf.IsTrigger = false // 重置触发状态
	return nil
}

// IsTriggered 检查是否已触发（线程安全）
func (kf *KeyFrame) IsTriggered() bool {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	return kf.IsTrigger
}

// Trigger 执行关键帧动作（带panic保护）
func (kf *KeyFrame) Trigger() {
	kf.mu.Lock()
	defer kf.mu.Unlock()

	if kf.IsTrigger || kf.Action == nil {
		return
	}

	// 异步执行防止阻塞时间轴
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("KeyFrame action panic: %v", r)
			}
		}()
		kf.Action()
	}()
	kf.IsTrigger = true
}

// Reset 重置关键帧触发状态
func (kf *KeyFrame) Reset() {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.IsTrigger = false
}
