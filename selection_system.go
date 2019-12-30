package superspatial

import (
	"image/color"
	"log"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

type selectable struct {
	*ecs.BasicEntity
	*common.RenderComponent
	exec func()
}
type SelectionSystem struct {
	selectables []selectable
	current     int
}

func (ss *SelectionSystem) Reset() {
	ss.selectables = []selectable{}
	ss.current = 0
}

func (*SelectionSystem) Remove(ecs.BasicEntity) {}

func (ss *SelectionSystem) Update(dt float32) {
	if len(ss.selectables) == 0 {
		return
	}
	ss.selectables[ss.current].RenderComponent.Color = color.RGBA{255, 255, 255, 255}
	if engo.Input.Button("Up").JustReleased() {
		log.Printf("Pressed up")
		ss.current--
	}
	if engo.Input.Button("Down").JustReleased() {
		log.Printf("Pressed down")
		ss.current++
	}
	if engo.Input.Button("Enter").JustReleased() {
		ss.selectables[ss.current].exec()
		return
	}
	if ss.current < 0 {
		ss.current = len(ss.selectables) - 1
	}
	if ss.current >= len(ss.selectables) {
		ss.current = 0
	}
	ss.selectables[ss.current].RenderComponent.Color = color.RGBA{255, 0, 0, 255}

}

func (ss *SelectionSystem) Add(e *ecs.BasicEntity, rc *common.RenderComponent, exec func()) {
	ss.selectables = append(ss.selectables, selectable{e, rc, exec})

}


