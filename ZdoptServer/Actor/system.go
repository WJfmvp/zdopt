package Actor

// actor/system.go
import (
	"context"
	"sync"
	"time"
)

type System struct {
	groups        map[int]*Group
	actors        sync.Map
	ctx           context.Context
	cancel        context.CancelFunc
	FuncgroupLock sync.RWMutex
}

func NewSystem() *System {
	sxt, cancel := context.WithCancel(context.Background())
	return &System{
		groups: make(map[int]*Group),
		ctx:    sxt,
		cancel: cancel,
	}
}

// AddGroupActors 添加Actor组
func (s *System) AddGroupActors(groupID int, creators []func() Actor) {
	s.FuncgroupLock.Lock()
	defer s.FuncgroupLock.Unlock()

	g := s.getOrCreateGroup(groupID)
	for _, create := range creators {
		actor := create()
		actor.Init(s.ctx)
		g.AddActor(actor)
	}
}

// getOrCreateGroup 获取或创建组（双重检查锁）
func (s *System) getOrCreateGroup(id int) *Group {
	if g, ok := s.groups[id]; ok {
		return g
	}

	s.FuncgroupLock.Lock()
	defer s.FuncgroupLock.Unlock()

	if g, ok := s.groups[id]; ok {
		return g
	}

	g := NewGroup(id, 33*time.Millisecond)
	s.groups[id] = g
	go g.StartUpdate()
	return g
}

// Stop 停止整个系统
func (s *System) Stop() {
	s.cancel()
	s.FuncgroupLock.Lock()
	defer s.FuncgroupLock.Unlock()
	for _, g := range s.groups {
		g.mu.Lock()
		for _, a := range g.actors {
			a.Stop()
		}
		g.mu.Unlock()
	}
}
