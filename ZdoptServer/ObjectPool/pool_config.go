package ObjectPool

type Pool interface {
	GetObj(init func(ObjectBase), callback func(ObjectBase), factory func() ObjectBase) ObjectBase
	ReleaseObj(obj ObjectBase) error
}
