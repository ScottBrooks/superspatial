package superspatial

import "github.com/EngoEngine/ecs"

type EffectSystem struct {
	Entities []*ClientEffect
}

func (es *EffectSystem) Add(ent *ClientEffect) {
	es.Entities = append(es.Entities, ent)
}
func (es *EffectSystem) Remove(ecs.BasicEntity) {}
func (es *EffectSystem) Update(dt float32) {
	for _, e := range es.Entities {
		if e.HasExpired() {

		}
	}
}
