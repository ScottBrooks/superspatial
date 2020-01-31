package superspatial

import (
	"math"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/ScottBrooks/sos"
	"github.com/go-gl/mathgl/mgl32"
	log "github.com/sirupsen/logrus"
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

	Mass         float32
	AttackDamage uint32
	HasAuthority bool
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
	s.UpdateAttack(dt)
	s.UpdateCooldown(dt)
}

func (s *Ship) UpdateCooldown(dt float32) {
	s.Ship.Cooldown -= dt
	if s.Ship.Cooldown <= 0 {
		s.Ship.Cooldown = 0
	}
	s.Ship.CurrentEnergy += dt * s.Ship.ChargeRate
	if s.Ship.CurrentEnergy >= s.Ship.MaxEnergy {
		s.Ship.CurrentEnergy = s.Ship.MaxEnergy
	}
}

func (s *Ship) UpdateAttack(dt float32) {
	if s.PIC.Attack && s.Ship.Cooldown <= 0 && s.Ship.CurrentEnergy > float32(s.AttackDamage) {
		engo.Mailbox.Dispatch(AttackMessage{
			Pos:    s.Ship.Pos,
			Vel:    s.Ship.Vel,
			Angle:  s.Ship.Angle,
			ShipID: s.ID,
			Damage: s.AttackDamage,
		})

		s.Ship.Cooldown = 0.2
		s.Ship.CurrentEnergy -= float32(s.AttackDamage)
	}
}

func (s *Ship) UpdatePos(dt float32) {
	if s.PIC.Forward {
		angleRad := float64(mgl32.DegToRad(s.Ship.Angle))
		accel := mgl32.Vec3{float32(math.Cos(angleRad)), float32(math.Sin(angleRad)), 0}
		accel = accel.Mul(s.Mass).Mul(dt)
		s.Ship.Vel = s.Ship.Vel.Add(accel)
		s.Ship.CurrentEnergy -= dt * (s.Ship.ChargeRate / 2.0)

	}
	if s.PIC.Back {
		angleRad := float64(mgl32.DegToRad(s.Ship.Angle))
		accel := mgl32.Vec3{float32(math.Cos(angleRad)), float32(math.Sin(angleRad)), 0}
		accel = accel.Mul(s.Mass).Mul(dt)
		s.Ship.Vel = s.Ship.Vel.Sub(accel)
		s.Ship.CurrentEnergy -= dt * (s.Ship.ChargeRate / 2.0)
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

func (s *Ship) SetupQBI() {
	ID := int64(s.ID)
	ShipCID := uint32(1000)
	BulletCID := uint32(1001)
	//relConstraint := QBIRelativeSphereConstraint{Radius: 100}
	relConstraint := QBIRelativeBoxConstraint{
		Edge: EdgeLength{X: 800, Y: 30000, Z: 300},
	}
	constraint := QBIConstraint{
		OrConstraint: []QBIConstraint{
			// EntiyID is our entity id
			QBIConstraint{EntityIDConstraint: &ID},
			// OR
			QBIConstraint{
				// It's in our relative sphere AND it's ShipComponent or Bullet component
				AndConstraint: []QBIConstraint{
					QBIConstraint{RelativeBoxConstraint: &relConstraint},
					QBIConstraint{
						OrConstraint: []QBIConstraint{
							QBIConstraint{ComponentIDConstraint: &ShipCID},
							QBIConstraint{ComponentIDConstraint: &BulletCID},
						},
					},
				},
			},
		},
	}
	qbi := ImprobableInterest{
		Interest: map[uint32]ComponentInterest{
			1003: ComponentInterest{
				Queries: []QBIQuery{
					{Constraint: constraint, ResultComponents: []uint32{1000, 1001, 54}},
				},
			},
		},
	}

	s.Interest = qbi

}

func (s *Ship) TakeDamage(amount uint32) {
	log.Printf("I'm taking damage: %d", amount)
	s.Ship.CurrentEnergy -= float32(amount)

}