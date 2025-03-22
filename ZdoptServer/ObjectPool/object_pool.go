package ObjectPool

import (
	"errors"
	"fmt"
	"sync"
)

// ObjectPool 泛型对象池实现
type ObjectPool[T ObjectBase] struct {
	pool     []*PObject[T]    // 所有创建的对象
	FreeList chan *PObject[T] // 空闲对象队列（缓冲通道提升性能）
	mu       sync.Mutex       // 保护pool切片操作
}

// NewObjectPool 创建指定初始大小的对象池
func NewObjectPool[T ObjectBase](size int) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool:     make([]*PObject[T], 0, size),
		FreeList: make(chan *PObject[T], size), // 缓冲通道减少锁竞争
	}
}

// AddObj 向池中添加新对象
func (op *ObjectPool[T]) AddObj(factory func() T) *PObject[T] {
	op.mu.Lock()
	defer op.mu.Unlock()

	obj := factory()
	pObj := NewPObject(obj)
	op.pool = append(op.pool, pObj)

	// 非阻塞尝试放入空闲队列
	select {
	case op.FreeList <- pObj:
	default:
	}
	return pObj
}

// GetObj 获取可用对象（优先从空闲队列获取）
func (op *ObjectPool[T]) GetObj(init func(T), callback func(T)) T {
	select {
	case pObj := <-op.FreeList: // 无锁快速路径
		obj, _ := pObj.GetObj(init, callback)
		return obj
	default: // 队列为空时创建新对象
		return op.AddObj(func() T {
			var zero T
			return zero
		}).date
	}
}

// ReleaseObj 释放对象回池
func (op *ObjectPool[T]) ReleaseObj(obj T) error {
	op.mu.Lock()
	defer op.mu.Unlock()

	// 线性搜索匹配对象（可优化为map查找）
	for _, pObj := range op.pool {
		if pObj.ReleaseObj(obj) {
			select {
			case op.FreeList <- pObj: // 非阻塞放回
			default: // 队列满时保持对象活跃
			}
			return nil
		}
	}
	return fmt.Errorf("object %v not found in pool", obj)
}

// GetObjAdapter 接口适配方法
func (op *ObjectPool[T]) GetObjAdapter(
	init func(ObjectBase),
	callback func(ObjectBase),
) ObjectBase {
	return op.GetObj(
		func(t T) { init(t) },
		func(t T) { callback(t) },
	)
}

// ReleaseObjAdapter 接口适配方法
func (op *ObjectPool[T]) ReleaseObjAdapter(obj ObjectBase) error {
	tObj, ok := obj.(T)
	if !ok {
		return errors.New("type mismatch")
	}
	return op.ReleaseObj(tObj)
}
