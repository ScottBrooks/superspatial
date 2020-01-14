package superspatial

import "github.com/EngoEngine/ecs"

type Predictable interface {
	Predict(dt float32)
}
type ClientPredictionSystem struct {
	Entities []Predictable
}

func (cps *ClientPredictionSystem) Add(ent Predictable) {
	cps.Entities = append(cps.Entities, ent)
}
func (cps *ClientPredictionSystem) Remove(ecs.BasicEntity) {}
func (cps *ClientPredictionSystem) Update(dt float32) {

	for _, e := range cps.Entities {
		e.Predict(dt)
	}
}
