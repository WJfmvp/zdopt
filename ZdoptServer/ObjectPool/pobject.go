package ObjectPool

import (
	"sync"
)

// PObject 结构体用于封装池中的对象
type PObject[T any] struct {
	date     T
	isUsing  bool
	init     func(T)
	callback func(T)
	mu       sync.Mutex
}

// NewPObject 创建PObject 实例
func NewPObject[T any](date T) *PObject[T] {
	return &PObject[T]{date: date, isUsing: false}
}

// GetObj 从池中获取对象
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
		p.init(p.date)
	}
	return p.date, true
}

// ReleaseObj 释放对象回收池
func (p *PObject[T]) ReleaseObj(obj T) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isUsing {
		return false
	}

	p.isUsing = false
	if p.callback != nil {
		p.callback(obj)
	}

	p.init = nil
	p.callback = nil
	return true
}
