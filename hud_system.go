package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

type HudElement struct {
	*ecs.BasicEntity
	*common.SpaceComponent
	Offset engo.Point
}

type HudSystem struct {
	Pos    *engo.Point
	Camera *common.CameraSystem

	Entities []HudElement
}

func (hs *HudSystem) Add(ent *ecs.BasicEntity, sc *common.SpaceComponent, offset engo.Point) {
	hs.Entities = append(hs.Entities, HudElement{ent, sc, offset})
}
func (hs *HudSystem) Remove(ecs.BasicEntity) {}
func (hs *HudSystem) Update(dt float32) {
	if hs.Camera == nil {
		return
	}

	for _, e := range hs.Entities {
		offset := e.Offset
		offset.Add(*hs.Pos).Add(engo.Point{X: hs.Camera.X(), Y: hs.Camera.Y()}).Add(e.Offset).Subtract(common.CameraBounds.Min)
		e.SpaceComponent.Position = offset
	}
}
