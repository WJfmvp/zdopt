package Timer

import (
	"errors"
)

// Set 安全设置关键帧参数
func (kf *KeyFrame) Set(time float32, action func()) error {
	if time <= 0 {
		return errors.New("invalid keyframe time")
	}
	if action == nil {
		return errors.New("action cannot be nil")
	}

	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.Time = time
	kf.Action = action
	kf.IsTrigger = false
	return nil
}

// IsTriggered 检查是否已触发
func (kf *KeyFrame) IsTriggered() bool {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	return kf.IsTrigger
}

// Trigger 触发关键帧动作
func (kf *KeyFrame) Trigger() {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	if !kf.IsTrigger && kf.Action != nil {
		kf.Action()
		kf.IsTrigger = true
	}
}

// Reset 重置关键帧状态
func (kf *KeyFrame) Reset() {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.IsTrigger = false
}
