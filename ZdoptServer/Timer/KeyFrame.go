package Timer

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"zdopt/ZdoptServer/Error"
	"zdopt/ZdoptServer/ObjectPool"
)

// 全局池管理相关变量
var (
	ObjectPoolManager *ObjectPool.Manager // 对象池管理器实例
	poolInitOnce      sync.Once           // 确保池只初始化一次
	poolName          = "keyframe_pool"   // 对象池注册名称
)

// KeyFrame 表示时间轴上的关键帧节点
type KeyFrame struct {
	Time      float32    // 触发时间（秒）
	Action    func()     // 触发时执行的回调
	IsTrigger bool       // 是否已触发状态
	mu        sync.Mutex // 保护并发访问
}

// OnGet 对象从池中取出时的初始化逻辑
func (kf *KeyFrame) OnGet() {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.IsTrigger = false // 重置触发状态
}

// OnRelease 对象放回池中的清理逻辑
func (kf *KeyFrame) OnRelease() {
	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.Time = 0
	kf.Action = nil // 清除旧回调防止内存泄漏
	kf.IsTrigger = false
}

// Validate 验证关键帧有效性
func (kf *KeyFrame) Validate() error {
	if kf.Time <= 0 {
		return fmt.Errorf("%w: got %f", Error.ErrInvalidKeyFrameTime, kf.Time)
	}
	if kf.Action == nil {
		return Error.ErrNilKeyFrameAction
	}
	return nil
}

// InitKeyFramePool 初始化关键帧对象池（线程安全）
func InitKeyFramePool() error {
	poolInitOnce.Do(func() {
		// 创建带热缓存的对象池
		ObjectPoolManager = ObjectPool.NewManager()
		keyFramePool := ObjectPool.NewGenericObjectPool(
			func() ObjectPool.ObjectBase {
				return &KeyFrame{Time: 0.1} // 默认时间值
			},
			100, // 热缓存容量，根据业务负载调整
		)

		// 注册到全局管理器
		if err := ObjectPool.RegisterPool(
			ObjectPoolManager,
			poolName,
			keyFramePool,
		); err != nil {
			log.Fatalf("KeyFrame pool initialization failed: %v", err)
		}
	})
	return nil
}

// GetKeyFrame 安全获取关键帧对象
func GetKeyFrame(time float32, action func()) (*KeyFrame, error) {
	// 前置参数校验
	if time <= 0 {
		return nil, Error.ErrInvalidKeyFrameTime
	}
	if action == nil {
		return nil, Error.ErrNilKeyFrameAction
	}

	// 延迟初始化对象池
	if err := InitKeyFramePool(); err != nil {
		return nil, fmt.Errorf("%w: %v", Error.ErrKeyFramePoolNotRegistered, err)
	}

	// 从管理器获取对象池
	pool, err := ObjectPool.GetPool(ObjectPoolManager, poolName)
	if err != nil {
		return nil, fmt.Errorf("pool acquisition failed: %w", err)
	}

	// 获取基础对象并进行类型断言
	baseObj := pool.GetObj(
		func(ob ObjectPool.ObjectBase) { ob.OnGet() },
		func(ob ObjectPool.ObjectBase) { ob.OnRelease() },
	)
	kf, ok := baseObj.(*KeyFrame)
	if !ok {
		return nil, errors.New("type assertion to *KeyFrame failed")
	}

	// 配置关键帧参数
	kf.mu.Lock()
	defer kf.mu.Unlock()
	kf.Time = time
	kf.Action = action

	// 二次验证防止无效配置
	if err := kf.Validate(); err != nil {
		_ = ReleaseKeyFrame(kf) // 回收无效对象
		return nil, err
	}

	return kf, nil
}

// ReleaseKeyFrame 安全释放关键帧对象
func ReleaseKeyFrame(kf *KeyFrame) error {
	if kf == nil {
		return errors.New("cannot release nil keyframe")
	}

	kf.mu.Lock()
	defer kf.mu.Unlock()

	// 防止重复释放
	if kf.Time == 0 {
		return Error.ErrKeyFrameDoubleRelease
	}

	// 获取对象池实例
	pool, err := ObjectPool.GetPool(ObjectPoolManager, poolName)
	if err != nil {
		return fmt.Errorf("pool acquisition failed: %w", err)
	}

	// 执行释放操作
	if err := pool.ReleaseObj(kf); err != nil {
		return fmt.Errorf("release operation failed: %w", err)
	}

	// 清理状态
	kf.Time = 0
	kf.Action = nil
	return nil
}
