package superspatial

import (
	"github.com/ScottBrooks/sos"

	"github.com/go-gl/mathgl/mgl32"
)

type Effect struct {
	ID     sos.EntityID
	Meta   ImprobableMetadata `sos:"53"`
	ACL    ImprobableACL      `sos:"50"`
	Pos    ImprobablePosition `sos:"54"`
	Worker WorkerComponent    `sos:"1005"`
	Effect EffectComponent    `sos:"1006"`
}

func (ss *ServerScene) NewEffect(pos mgl32.Vec3, effect int, expiry int) {
	readAttrSet := []WorkerAttributeSet{
		{[]string{"client"}},
		{[]string{"position"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		cidEffect:         WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidPosition:       WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidACL:            WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidWorkerBalancer: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
	}

	log.Printf("Createing effects at pos: %+v", pos)
	ent := Effect{
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Pos:  ImprobablePosition{Coords: Coordinates{float64(pos[0]), 0, float64(pos[1])}},
		Meta: ImprobableMetadata{Name: "Effect"},
		Effect: EffectComponent{
			Pos:    pos,
			Id:     int32(effect),
			Expiry: int32(expiry),
		},
	}
	ss.spatial.CreateEntity(ent)

}
