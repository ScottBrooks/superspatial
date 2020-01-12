package superspatial

import "github.com/ScottBrooks/sos"

type Bullet struct {
	ID     sos.EntityID       `sos:"-"`
	ACL    ImprobableACL      `sos:"50"`
	Meta   ImprobableMetadata `sos:"53"`
	Pos    ImprobablePosition `sos:"54"`
	Bullet BulletComponent    `sos:"1001"`

	HasAuthority bool `sos:"-"`
}

func (b *Bullet) Update(dt float32) {
	b.Bullet.Pos = b.Bullet.Pos.Add(b.Bullet.Vel.Mul(dt))
	b.Pos.Coords.X = float64(b.Bullet.Pos[0])
	b.Pos.Coords.Z = float64(b.Bullet.Pos[1])

}
