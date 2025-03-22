package ObjectPool

import (
	"sync"
	"zdopt/ZdoptServer/Error"
)

// ObjectBase 对象池元素基础接口
type ObjectBase interface {
	OnGet()     // 取出时调用
	OnRelease() // 放回时调用
}

// Manager 多对象池管理器
type Manager struct {
	mu    sync.RWMutex    // 读写锁保护注册表
	pools map[string]Pool // 名称->对象池映射
}

// NewManager 创建新的池管理器
func NewManager() *Manager {
	return &Manager{
		pools: make(map[string]Pool),
	}
}

// GenericObjectPool 带热缓存的泛型对象池
type GenericObjectPool[T ObjectBase] struct {
	pool     sync.Pool // 基础对象池
	hotCache chan T    // 热缓存队列（提升高频访问性能）
}

// NewGenericObjectPool 创建带热缓存的对象池
func NewGenericObjectPool[T ObjectBase](factory func() T, cacheSize int) *GenericObjectPool[T] {
	pool := &GenericObjectPool[T]{
		pool: sync.Pool{
			New: func() any { return factory() },
		},
		hotCache: make(chan T, cacheSize),
	}

	// 异步预热缓存（不阻塞主流程）
	go func() {
		for i := 0; i < cacheSize; i++ {
			select {
			case pool.hotCache <- factory():
			default:
				break
			}
		}
	}()
	return pool
}

// GetObj 优先从热缓存获取对象
func (gop *GenericObjectPool[T]) GetObj(
	init func(ObjectBase),
	callback func(ObjectBase),
) ObjectBase {
	select {
	case obj := <-gop.hotCache:
		obj.OnGet()
		return obj
	default:
		obj := gop.pool.Get().(T)
		obj.OnGet()
		return obj
	}
}

// ReleaseObj 优先放回热缓存
func (gop *GenericObjectPool[T]) ReleaseObj(obj ObjectBase) error {
	select {
	case gop.hotCache <- obj.(T): // 尝试放回热缓存
	default:
		gop.pool.Put(obj) // 热缓存满时放回基础池
	}
	return nil
}

// RegisterPool 注册对象池到管理器
func RegisterPool(opm *Manager, name string, pool Pool) error {
	opm.mu.Lock()
	defer opm.mu.Unlock()

	if _, exists := opm.pools[name]; exists {
		return Error.ErrPoolAlreadyRegistered
	}
	opm.pools[name] = pool
	return nil
}

// GetPool 从管理器获取已注册的对象池
func GetPool(opm *Manager, name string) (Pool, error) {
	opm.mu.RLock()
	defer opm.mu.RUnlock()

	pool, exists := opm.pools[name]
	if !exists {
		return nil, Error.ErrPoolNotFound
	}
	return pool, nil
}
