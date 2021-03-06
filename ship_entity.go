package superspatial

import (
	"math"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/ScottBrooks/sos"
	"github.com/go-gl/mathgl/mgl32"
)

type Ship struct {
	ecs.BasicEntity
	common.SpaceComponent
	common.CollisionComponent

	ID       sos.EntityID
	PIC      PlayerInputComponent `sos:"1003"`
	ACL      ImprobableACL        `sos:"50"`
	Pos      ImprobablePosition   `sos:"54"`
	Meta     ImprobableMetadata   `sos:"53"`
	Interest ImprobableInterest   `sos:"58"`
	Ship     ShipComponent        `sos:"1000"`
	Worker   WorkerComponent      `sos:"1005"`

	Mass         float32
	AttackDamage uint32
	HasAuthority bool
}

func NewShip(sp mgl32.Vec2, clientWorkerID string) Ship {
	readAttrSet := []WorkerAttributeSet{
		{[]string{"position"}},
		{[]string{"client"}},
		{[]string{"balancer"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		cidPlayerInput:    WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"workerId:" + clientWorkerID}}}},
		cidShip:           WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidInterest:       WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidPosition:       WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidACL:            WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidWorkerBalancer: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
	}
	relConstraint := QBIRelativeBoxConstraint{
		Edge: EdgeLength{X: 1024 * 1.5, Y: 30000, Z: 768 * 1.5},
	}

	playerInputCID := uint32(cidPlayerInput)

	ship := Ship{
		Pos:  ImprobablePosition{Coords: Coordinates{float64(sp[0]), 0, float64(sp[1])}},
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Meta: ImprobableMetadata{Name: "Client"},
		Interest: ImprobableInterest{
			Interest: map[uint32]ComponentInterest{
				cidPlayerInput: ComponentInterest{
					Queries: []QBIQuery{
						{Constraint: QBIConstraint{RelativeBoxConstraint: &relConstraint}, ResultComponents: []uint32{cidShip, cidPosition, cidMetadata, cidWorkerBalancer, cidEffect}},
					},
				},
				cidShip: ComponentInterest{
					Queries: []QBIQuery{
						{Constraint: QBIConstraint{ComponentIDConstraint: &playerInputCID}, ResultComponents: []uint32{cidPlayerInput}},
					},
				},
			},
		},
		Mass:         1000.0,
		AttackDamage: 20,
		Ship: ShipComponent{
			Pos:    sp.Vec3(0),
			Radius: 32,
		},

		BasicEntity:        ecs.NewBasic(),
		CollisionComponent: common.CollisionComponent{Main: 1, Group: 1, Collides: 1 | 2},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{X: sp[0], Y: sp[1]},
			Width:    64,
			Height:   64,
		},
	}

	return ship
}

func clampToAABB(pos mgl32.Vec3, vel mgl32.Vec3, aabb engo.AABB) (mgl32.Vec3, mgl32.Vec3) {
	if pos[0] < aabb.Min.X {
		pos[0] = aabb.Min.X
		vel[0] *= -1
	}
	if pos[1] < aabb.Min.Y {
		pos[1] = aabb.Min.Y
		vel[1] *= -1
	}
	if pos[0] > aabb.Max.X {
		pos[0] = aabb.Max.X
		vel[0] *= -1
	}
	if pos[1] > aabb.Max.Y {
		pos[1] = aabb.Max.Y
		vel[1] *= -1
	}

	return pos, vel
}

func (s *Ship) Update(dt float32) {
	s.UpdatePos(dt)
}

func (s *Ship) UpdatePos(dt float32) {
	if s.PIC.Forward {
		angleRad := float64(mgl32.DegToRad(s.Ship.Angle))
		accel := mgl32.Vec3{float32(math.Cos(angleRad)), float32(math.Sin(angleRad)), 0}
		accel = accel.Mul(s.Mass).Mul(dt)
		s.Ship.Vel = s.Ship.Vel.Add(accel)

	}
	if s.PIC.Back {
		angleRad := float64(mgl32.DegToRad(s.Ship.Angle))
		accel := mgl32.Vec3{float32(math.Cos(angleRad)), float32(math.Sin(angleRad)), 0}
		accel = accel.Mul(s.Mass).Mul(dt)
		s.Ship.Vel = s.Ship.Vel.Sub(accel)
	}
	if s.PIC.Left {
		s.Ship.Angle -= 90.0 * dt
	}
	if s.PIC.Right {
		s.Ship.Angle += 90.0 * dt
	}

	vLen := s.Ship.Vel.Len()
	if vLen > 500 || vLen < -500 {
		s.Ship.Vel = s.Ship.Vel.Normalize().Mul(500)
	}

	s.Ship.Pos = s.Ship.Pos.Add(s.Ship.Vel.Mul(dt))
	s.Ship.Pos, s.Ship.Vel = clampToAABB(s.Ship.Pos, s.Ship.Vel, worldBounds)
	s.Pos.Coords.X = float64(s.Ship.Pos[0])
	s.Pos.Coords.Z = float64(s.Ship.Pos[1])

	s.SpaceComponent.Position.X = s.Ship.Pos[0]
	s.SpaceComponent.Position.Y = s.Ship.Pos[1]
	s.SpaceComponent.Rotation = s.Ship.Angle
}
