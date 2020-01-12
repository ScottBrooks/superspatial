package superspatial

import "github.com/EngoEngine/ecs"

type ClientPredictionSystem struct {
	Entities []*ClientShip
}

func (cps *ClientPredictionSystem) Add(ship *ClientShip) {
	cps.Entities = append(cps.Entities, ship)
}
func (cps *ClientPredictionSystem) Remove(ecs.BasicEntity) {}
func (cps *ClientPredictionSystem) Update(dt float32) {

	for _, e := range cps.Entities {
		e.UpdatePos(dt)
	}
}
