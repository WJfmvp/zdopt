package Actor

//group.go
import (
	"sync"
	"time"
)

// Group Actor管理组
type Group struct {
	id        int
	deltaTime time.Duration
	actors    []Actor
	index     uint64
	mu        sync.RWMutex
}

func NewGroup(id int, delta time.Duration) *Group {
	return &Group{
		id:        id,
		deltaTime: delta,
		actors:    make([]Actor, 0, 1024),
	}
}

// AddActor 线程安全的Actor增加
func (g *Group) AddActor(actor Actor) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.actors = append(g.actors, actor)
}

func (g *Group) StartUpdate() {
	ticker := time.NewTicker(g.deltaTime)
	defer ticker.Stop()

	for range ticker.C {
		g.mu.Lock()
		for _, actor := range g.actors {
			go actor.Update(g.deltaTime)
		}
		g.mu.Unlock()
	}
}
