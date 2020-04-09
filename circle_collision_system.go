package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

type CircleEntity struct {
	*ecs.BasicEntity
	*common.SpaceComponent

	Radius float32
}

type CircleCollisionSystem struct {
	Entities []CircleEntity
}

type CircleCollisionMessage struct {
	A *CircleEntity
	B *CircleEntity
}

func (CircleCollisionMessage) Type() string {
	return "CircleCollisionMessage"
}

func (ccs *CircleCollisionSystem) Add(ent *ecs.BasicEntity, sc *common.SpaceComponent, radius float32) {
	ccs.Entities = append(ccs.Entities, CircleEntity{ent, sc, radius})
}
func (ccs *CircleCollisionSystem) Remove(ent ecs.BasicEntity) {
	idx := -1
	for i, e := range ccs.Entities {
		if ent.ID() == e.ID() {
			idx = i
		}
	}
	if idx != -1 {
		ccs.Entities = append(ccs.Entities[:idx], ccs.Entities[idx+1:]...)
	}
}
func (ccs *CircleCollisionSystem) Update(dt float32) {
	for _, a := range ccs.Entities {
		for _, b := range ccs.Entities {
			if a == b {
				continue
			}
			dist := a.Position.PointDistance(b.Position)
			if dist-a.Radius-b.Radius < 0 {
				engo.Mailbox.Dispatch(CircleCollisionMessage{
					A: &a,
					B: &b,
				})
			}
		}
	}
}
