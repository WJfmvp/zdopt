package Timer

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"zdopt/ZdoptServer/ObjectPool"
)

// 定义错误类型
var (
	ErrKeyFramePoolNotRegistered = errors.New("keyframe pool not registered")
	ErrInvalidKeyFrameTime       = errors.New("invalid keyframe time (must be positive)")
	ErrNilKeyFrameAction         = errors.New("keyframe action cannot be nil")
	ErrKeyFrameDoubleRelease     = errors.New("attempted to release keyframe twice")
)

// 全局对象池管理器（带互斥锁）
var (
	ObjectPoolManager *ObjectPool.Manager
	poolInitOnce      sync.Once
	poolInitError     error
	poolName          = "keyframe_pool" // 定义池名称
)

// KeyFrame 结构体实现 ObjectBase 接口
type KeyFrame struct {
	Time      float32
	Action    func()
	IsTrigger bool
	mu        sync.Mutex // 为并发操作添加互斥锁
}

// OnGet 对象从池中取出时调用
func (kf *KeyFrame) OnGet() {
	kf.mu.Lock()
	defer kf.mu.Unlock()

	kf.IsTrigger = false
	kf.Action = nil // 清空旧回调
}

// OnRelease 对象放回池时调用
func (kf *KeyFrame) OnRelease() {
	kf.mu.Lock()
	defer kf.mu.Unlock()

	kf.Time = 0
	kf.Action = nil
	kf.IsTrigger = false
}

// Validate 验证关键帧有效性
func (kf *KeyFrame) Validate() error {
	if kf.Time <= 0 {
		return fmt.Errorf("%w: got %f", ErrInvalidKeyFrameTime, kf.Time)
	}
	if kf.Action == nil {
		return ErrNilKeyFrameAction
	}
	return nil
}

// InitKeyFramePool 初始化对象池（线程安全）
func InitKeyFramePool() error {
	poolInitOnce.Do(func() {
		// 1. 创建管理器
		ObjectPoolManager = ObjectPool.NewManager()

		// 2. 创建泛型对象池实例
		keyFramePool := ObjectPool.NewGenericObjectPool(
			func() ObjectPool.ObjectBase {
				return &KeyFrame{Time: 0.1} // 默认值
			},
		)

		// 3. 注册到管理器（需指定池名称）
		poolInitError = ObjectPool.RegisterPool(
			ObjectPoolManager,
			poolName,     // 使用预定义的池名称
			keyFramePool, // 传入实现了 Pool 接口的对象
		)
		if poolInitError != nil {
			log.Printf("KeyFrame pool initialization failed: %v", poolInitError)
		}
	})
	return poolInitError
}

// GetKeyFrame 安全获取关键帧对象
func GetKeyFrame(time float32, action func()) (*KeyFrame, error) {
	// 参数校验
	if time <= 0 {
		return nil, ErrInvalidKeyFrameTime
	}
	if action == nil {
		return nil, ErrNilKeyFrameAction
	}

	// 确保对象池已初始化
	if err := InitKeyFramePool(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyFramePoolNotRegistered, err)
	}

	// 从对象池获取（指定池名称和类型）
	pool, err := ObjectPool.GetPool(ObjectPoolManager, poolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}

	// 调用池的 GetObj 方法
	baseObj := pool.GetObj(
		func(ob ObjectPool.ObjectBase) { ob.OnGet() },     // 初始化回调
		func(ob ObjectPool.ObjectBase) { ob.OnRelease() }, // 释放回调
		func() ObjectPool.ObjectBase { // 工厂函数
			return &KeyFrame{Time: time}
		},
	)

	// 类型断言为 *KeyFrame
	kf, ok := baseObj.(*KeyFrame)
	if !ok {
		return nil, errors.New("type assertion to *KeyFrame failed")
	}

	// 配置对象
	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.Time = time
	kf.Action = action

	// 二次验证
	if err := kf.Validate(); err != nil {
		_ = ReleaseKeyFrame(kf) // 回收无效对象
		return nil, err
	}

	return kf, nil
}

// ReleaseKeyFrame 安全释放关键帧
func ReleaseKeyFrame(kf *KeyFrame) error {
	if kf == nil {
		return errors.New("cannot release nil keyframe")
	}

	kf.mu.Lock()
	defer kf.mu.Unlock()

	// 防止重复释放
	if kf.Time == 0 && kf.Action == nil {
		return ErrKeyFrameDoubleRelease
	}

	// 获取对象池
	pool, err := ObjectPool.GetPool(ObjectPoolManager, poolName)
	if err != nil {
		return fmt.Errorf("failed to get pool: %w", err)
	}

	// 调用池的 ReleaseObj 方法
	if err := pool.ReleaseObj(kf); err != nil {
		return fmt.Errorf("release failed: %w", err)
	}

	// 标记为已释放
	kf.Time = 0
	kf.Action = nil
	return nil
}
