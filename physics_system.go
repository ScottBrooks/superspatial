package superspatial

import (
	"github.com/EngoEngine/ecs"
)

type PhysicsSystem struct {
}

func (*PhysicsSystem) Remove(ecs.BasicEntity) {}

func (*PhysicsSystem) Update(dt float32) {
}
