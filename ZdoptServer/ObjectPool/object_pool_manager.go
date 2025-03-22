package ObjectPool

import (
	"errors"
	"sync"
)

var (
	ErrPoolAlreadyRegistered = errors.New("pool already registered")
	ErrPoolNotFound          = errors.New("pool not found")
)

type ObjectBase interface {
	OnGet()
	OnRelease()
}

// Manager 结构体用于管理多个对象池
type Manager struct {
	mu    sync.Mutex
	pools map[string]Pool
}

func NewManager() *Manager {
	return &Manager{
		pools: make(map[string]Pool),
	}
}

// GenericObjectPool 结构体用于封装泛型对象池
type GenericObjectPool[T ObjectBase] struct {
	pool sync.Pool
}

// NewGenericObjectPool 创建泛型对象池
func NewGenericObjectPool[T ObjectBase](factory func() T) *GenericObjectPool[T] {
	return &GenericObjectPool[T]{
		pool: sync.Pool{
			New: func() any {
				return factory()
			},
		},
	}
}

// GetObj 实现Pool接口
func (gop *GenericObjectPool[T]) GetObj(
	init func(ObjectBase),
	callback func(ObjectBase),
	factory func() ObjectBase,
) ObjectBase {
	obj := gop.pool.Get().(T)
	obj.OnGet()
	return obj
}

func (gop *GenericObjectPool[T]) ReleaseObj(obj ObjectBase) error {
	tObj, ok := obj.(T)
	if !ok {
		return errors.New("object is not T")
	}
	tObj.OnRelease()
	gop.pool.Put(tObj)
	return nil
}

// RegisterPool 注册和获取逻辑
func RegisterPool(opm *Manager, name string, pool Pool) error {
	opm.mu.Lock()
	defer opm.mu.Unlock()

	if _, exists := opm.pools[name]; exists {
		return ErrPoolAlreadyRegistered
	}
	opm.pools[name] = pool
	return nil
}

func GetPool(opm *Manager, name string) (Pool, error) {
	opm.mu.Lock()
	defer opm.mu.Unlock()

	pool, exists := opm.pools[name]
	if !exists {
		return nil, ErrPoolNotFound
	}
	return pool, nil
}
