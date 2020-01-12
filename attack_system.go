package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/sos"
	"github.com/go-gl/mathgl/mgl32"
)

type AttackMessage struct {
	Pos   mgl32.Vec3
	Vel   mgl32.Vec3
	Angle float32

	Damage uint32
	ShipID sos.EntityID
}

func (AttackMessage) Type() string {
	return "AttackMessage"
}

type AttackSystem struct {
	SS *ServerScene

	Entities []*Bullet
}

func (as *AttackSystem) newBullet(am AttackMessage) {
	readAttrSet := []WorkerAttributeSet{
		{[]string{"position"}},
		{[]string{"client"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		1001: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"position"}}}},
		54:   WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"position"}}}},
	}

	ent := Bullet{
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Pos:  ImprobablePosition{Coords: Coordinates{float64(am.Pos[0]), float64(am.Pos[1]), 2}},
		Meta: ImprobableMetadata{Name: "Bullet"},
		Bullet: BulletComponent{
			Pos:    am.Pos,
			Vel:    am.Vel,
			Damage: am.Damage,
			ShipID: am.ShipID,
		},
	}
	reqID := as.SS.spatial.CreateEntity(ent)

	as.SS.OnCreateFunc[reqID] = func(ID sos.EntityID) {
		ent.ID = ID
		as.SS.Entities[ID] = &ent
		as.Entities = append(as.Entities, &ent)
	}
}

func (as *AttackSystem) New(w *ecs.World) {
	engo.Mailbox.Listen(AttackMessage{}.Type(), func(msg engo.Message) {
		am, ok := msg.(AttackMessage)
		if !ok {
			return
		}

		as.newBullet(am)

	})
}
func (as *AttackSystem) Remove(ecs.BasicEntity) {}
func (as *AttackSystem) Update(dt float32) {
	for _, ent := range as.Entities {
		ent.Update(dt)
		if ent.HasAuthority {
			as.SS.spatial.UpdateComponent(ent.ID, 1001, ent.Bullet)
			as.SS.spatial.UpdateComponent(ent.ID, 54, ent.Pos)
		}
	}
}
