package superspatial

import (
	"github.com/ScottBrooks/sos"
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

type Ship struct {
	ID       sos.EntityID         `sos:"-"`
	PIC      PlayerInputComponent `sos:"1003"`
	ACL      ImprobableACL        `sos:"50"`
	Pos      ImprobablePosition   `sos:"54"`
	Meta     ImprobableMetadata   `sos:"53"`
	Interest ImprobableInterest   `sos:"58"`
	Ship     ShipComponent        `sos:"1000"`

	Mass float32 `sos:"-"`
}

func (s *Ship) UpdatePos(dt float32) {
	if s.PIC.Forward {
		angleRad := float64(mgl32.DegToRad(s.Ship.Angle))
		accel := mgl32.Vec3{float32(math.Cos(angleRad)), float32(math.Sin(angleRad)), 0}
		accel = accel.Mul(s.Mass).Mul(dt)
		s.Ship.Vel = s.Ship.Vel.Add(accel)

	}
	if s.PIC.Back {
	}
	if s.PIC.Left {
		s.Ship.Angle -= 5.0 * dt
	}
	if s.PIC.Right {
		s.Ship.Angle += 5.0 * dt
	}
	s.Ship.Pos = s.Ship.Pos.Add(s.Ship.Vel.Mul(dt))
	s.Pos.Coords.X = float64(s.Ship.Pos[0])
	s.Pos.Coords.Y = float64(s.Ship.Pos[1])
}

func (s *Ship) SetupQBI() {
	ID := int64(s.ID)
	relSphere := QBIRelativeSphereConstraint{Radius: 100}
	constraint := QBIConstraint{AndConstraint: []QBIConstraint{
		QBIConstraint{EntityIDConstraint: &ID},
		QBIConstraint{RelativeSphereConstraint: &relSphere},
	}}
	qbi := ImprobableInterest{
		Interest: map[uint32]ComponentInterest{
			1003: ComponentInterest{
				Queries: []QBIQuery{
					{Constraint: constraint, ResultComponents: []uint32{1000, 54}},
				},
			},
		},
	}

	s.Interest = qbi

}
