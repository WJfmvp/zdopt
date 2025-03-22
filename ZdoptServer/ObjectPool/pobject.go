package ObjectPool

import (
	"sync"
)

// PObject 池化对象包装器
type PObject[T any] struct {
	date     T          // 实际存储的对象
	isUsing  bool       // 使用状态标志
	init     func(T)    // 取出时的初始化回调
	callback func(T)    // 放回时的清理回调
	mu       sync.Mutex // 状态锁
}

// NewPObject 创建新的池化对象
func NewPObject[T any](date T) *PObject[T] {
	return &PObject[T]{
		date:    date,
		isUsing: false,
	}
}

// GetObj 取出对象（加锁保证原子性）
func (p *PObject[T]) GetObj(init func(T), callback func(T)) (T, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isUsing {
		var zero T
		return zero, false
	}

	p.isUsing = true
	p.init = init
	p.callback = callback

	if p.init != nil {
		p.init(p.date) // 执行初始化回调
	}
	return p.date, true
}

// ReleaseObj 放回对象到池中
func (p *PObject[T]) ReleaseObj(obj T) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isUsing {
		return false
	}

	p.isUsing = false
	if p.callback != nil {
		p.callback(obj) // 执行清理回调
	}

	// 清空回调引用防止内存泄漏
	p.init = nil
	p.callback = nil
	return true
}
