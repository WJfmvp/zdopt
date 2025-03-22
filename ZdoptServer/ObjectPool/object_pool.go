package ObjectPool

import (
	"errors"
	"fmt"
	"sync"
)

// ObjectPool 强制泛型 T 必须实现 ObjectBase 接口
type ObjectPool[T ObjectBase] struct {
	pool     []*PObject[T]
	FreeList []*PObject[T]
	mu       sync.Mutex
}

// NewObjectPool 创建对象池（泛型 T 必须实现 ObjectBase）
func NewObjectPool[T ObjectBase]() *ObjectPool[T] {
	return &ObjectPool[T]{
		pool:     make([]*PObject[T], 0),
		FreeList: make([]*PObject[T], 0),
	}
}

// AddObj 添加对象到池中
func (op *ObjectPool[T]) AddObj(factory func() T) *PObject[T] {
	op.mu.Lock()
	defer op.mu.Unlock()

	obj := factory()
	pObj := NewPObject(obj)
	op.pool = append(op.pool, pObj)
	op.FreeList = append(op.FreeList, pObj)
	return pObj
}

// GetObj 从对象池获取对象（内部方法，保持泛型约束）
func (op *ObjectPool[T]) GetObj(init func(T), callback func(T), factory func() T) T {
	op.mu.Lock()
	defer op.mu.Unlock()

	if len(op.FreeList) > 0 {
		pObj := op.FreeList[0]
		op.FreeList = op.FreeList[1:]
		obj, _ := pObj.GetObj(init, callback)
		return obj
	}

	pObj := op.AddObj(factory)
	obj, _ := pObj.GetObj(init, callback)
	return obj
}

// ReleaseObj 释放对象
func (op *ObjectPool[T]) ReleaseObj(obj T) error {
	op.mu.Lock()
	defer op.mu.Unlock()

	var released bool
	for _, pObj := range op.pool {
		if pObj.ReleaseObj(obj) {
			op.FreeList = append(op.FreeList, pObj)
			released = true
			break
		}
	}

	if !released {
		return fmt.Errorf("object not found or already released: %v", obj)
	}
	return nil
}

// GetObjAdapter 实现 Pool 接口的适配器方法（显式类型转换)
func (op *ObjectPool[T]) GetObjAdapter(
	init func(ObjectBase),
	callback func(ObjectBase),
	factory func() ObjectBase,
) ObjectBase {
	// 类型转换步骤：
	// 1. 将 ObjectBase 工厂函数转换为 T 的工厂函数
	genericFactory := func() T {
		base := factory() // 获取 ObjectBase 实例
		return base.(T)   // 显式断言为 T（T 必须实现 ObjectBase）
	}
	// 2. 将 ObjectBase 回调函数适配为 T 的回调函数
	genericInit := func(t T) {
		init(t) // T 已实现 ObjectBase，直接传递
	}
	genericCallback := func(t T) {
		callback(t)
	}
	// 3. 调用内部 GetObj 方法并返回 ObjectBase
	return op.GetObj(genericInit, genericCallback, genericFactory)
}

func (op *ObjectPool[T]) ReleaseObjAdapter(obj ObjectBase) error {
	// 显式类型断言：ObjectBase -> T
	tObj, ok := obj.(T)
	if !ok {
		return errors.New("invalid object type")
	}
	err := op.ReleaseObj(tObj)
	if err != nil {
		return err
	}
	return nil
}
