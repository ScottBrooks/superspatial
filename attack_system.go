package superspatial

import (
	"math"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
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

	pos := am.Pos
	vel := am.Vel
	angleRad := float64(mgl32.DegToRad(am.Angle))
	dir := mgl32.Vec3{float32(math.Cos(angleRad)), float32(math.Sin(angleRad)), 0}.Normalize()
	offset := dir.Mul(30)
	pos = pos.Add(offset)
	vel = vel.Add(dir.Mul(30))

	ent := Bullet{
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Pos:  ImprobablePosition{Coords: Coordinates{float64(pos[0]), 0, float64(pos[1])}},
		Meta: ImprobableMetadata{Name: "Bullet"},
		Bullet: BulletComponent{
			Pos:    pos,
			Vel:    vel,
			Damage: am.Damage,
			ShipID: am.ShipID,
		},

		BasicEntity: ecs.NewBasic(),
		CollisionComponent: common.CollisionComponent{
			Main:  1,
			Group: 2,
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{pos[0], pos[1]},
			Width:    16,
			Height:   16,
		},
	}
	reqID := as.SS.spatial.CreateEntity(ent)

	as.SS.OnCreateFunc[reqID] = func(ID sos.EntityID) {
		ent.ID = ID
		as.SS.Entities[ID] = &ent
		as.SS.ECS[ent.GetBasicEntity().ID()] = &ent
		as.Entities = append(as.Entities, &ent)

		as.SS.CollisionSystem.Add(&ent.BasicEntity, &ent.CollisionComponent, &ent.SpaceComponent)
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
