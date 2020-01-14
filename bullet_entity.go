package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo/common"
	"github.com/ScottBrooks/sos"
)

type Bullet struct {
	ecs.BasicEntity
	common.CollisionComponent
	common.SpaceComponent

	ID     sos.EntityID
	ACL    ImprobableACL      `sos:"50"`
	Meta   ImprobableMetadata `sos:"53"`
	Pos    ImprobablePosition `sos:"54"`
	Bullet BulletComponent    `sos:"1001"`

	HasAuthority bool
}

func (b *Bullet) Update(dt float32) {
	b.Bullet.Pos = b.Bullet.Pos.Add(b.Bullet.Vel.Mul(dt))
	b.Pos.Coords.X = float64(b.Bullet.Pos[0])
	b.Pos.Coords.Z = float64(b.Bullet.Pos[1])

	b.SpaceComponent.Position.X = b.Bullet.Pos[0]
	b.SpaceComponent.Position.Y = b.Bullet.Pos[1]
}
