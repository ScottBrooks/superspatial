package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"

	log "github.com/sirupsen/logrus"

	"github.com/ScottBrooks/sos"
)

type MenuSprite struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
}

func mustLoadSprite(sprite string) *common.Texture {
	tex, err := common.LoadedSprite(sprite)
	if err != nil {
		panic(err)
	}

	return tex
}

type ClientScene struct {
	ServerScene

	CS        *common.CameraSystem
	R         common.RenderSystem
	Camera    common.EntityScroller
	PIS       PlayerInputSystem
	CPS       ClientPredictionSystem
	HS        HudSystem
	HUDPos    engo.Point
	EnergyBar MenuSprite

	EntToEcs map[sos.EntityID]uint64
	Ships    map[sos.EntityID]*ClientShip
	Bullets  map[sos.EntityID]*ClientBullet
}

type PlayerInputSystem struct {
	ID      sos.EntityID
	spatial *sos.SpatialSystem
}

func (pis *PlayerInputSystem) Remove(ecs.BasicEntity) {}
func (pis *PlayerInputSystem) Update(dt float32) {
	var p PlayerInputComponent

	p.Left = engo.Input.Button("Left").Down()
	p.Right = engo.Input.Button("Right").Down()
	p.Forward = engo.Input.Button("Up").Down()
	p.Back = engo.Input.Button("Down").Down()
	p.Attack = engo.Input.Button("Space").Down()
	if p.Attack {
		log.Printf("Set attack")
	}

	if pis.ID != 0 {
		pis.spatial.UpdateComponent(pis.ID, 1003, p)
	}

}

type ClientBullet struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent

	BulletComponent
}

func (cb *ClientBullet) Predict(dt float32) {
	cb.BulletComponent.Pos = cb.BulletComponent.Pos.Add(cb.BulletComponent.Vel.Mul(dt))

	cb.SpaceComponent.SetCenter(engo.Point{cb.BulletComponent.Pos[0], cb.BulletComponent.Pos[1]})
}

type ClientShip struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent

	ShipComponent
}

func (cs *ClientShip) Predict(dt float32) {

	cs.ShipComponent.Pos = cs.ShipComponent.Pos.Add(cs.ShipComponent.Vel.Mul(dt))
	//	aabb := engo.AABB{Max: engo.Point{2048, 1024}}
	//	cs.ShipComponent.Pos = clampToAABB(cs.ShipComponent.Pos, aabb)

	cs.SpaceComponent.SetCenter(engo.Point{cs.ShipComponent.Pos[0], cs.ShipComponent.Pos[1]})
	cs.SpaceComponent.Rotation = cs.ShipComponent.Angle
}

type Background struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
}

func (cs *ClientScene) Preload() {
	engo.Files.Load("Ships/Ship1/Ship1.png")
	engo.Files.Load("Backgrounds/stars.png")
	engo.Files.Load("Ships/Shots/Shot6/bullet.png")
	engo.Files.Load("UI/Upgrade/Health.png")
	engo.Files.Load("UI/Loading_Bar/Loading_Bar_2_1.png")
	engo.Files.Load("UI/Loading_Bar/Loading_Bar_2_2.png")
	engo.Files.Load("UI/Loading_Bar/Loading_Bar_2_3.png")

}

func (cs *ClientScene) Setup(u engo.Updater) {

	w, _ := u.(*ecs.World)
	var locatorParams *sos.WorkerLocatorParams
	host := cs.ServerScene.Host
	port := cs.ServerScene.Port
	if cs.ServerScene.PIT != "" {
		locatorParams = &sos.WorkerLocatorParams{
			ProjectName:     cs.ServerScene.ProjectName,
			CredentialsType: sos.WorkerLocatorPlayerIdentityCredentials,

			PlayerIdentityCredentials: sos.WorkerPlayerIdentityCredentials{
				PlayerIdentityToken: cs.ServerScene.PIT,
				LoginToken:          cs.ServerScene.LT,
			},
		}
		host = cs.ServerScene.Locator
		port = 0
	}
	log.Printf("LocatorParams: %+v", locatorParams)

	cs.spatial = sos.NewSpatialSystem(cs, host, port, cs.ServerScene.WorkerID, locatorParams)
	cs.Entities = map[sos.EntityID]interface{}{}
	cs.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}
	cs.EntToEcs = map[sos.EntityID]uint64{}
	cs.Ships = map[sos.EntityID]*ClientShip{}
	cs.Bullets = map[sos.EntityID]*ClientBullet{}

	cs.PIS.spatial = cs.ServerScene.spatial

	w.AddSystem(&cs.R)
	w.AddSystem(&cs.PIS)
	w.AddSystem(&SpatialPumpSystem{&cs.ServerScene})
	w.AddSystem(&cs.CPS)
	for _, sys := range w.Systems() {
		switch ent := sys.(type) {
		case *common.CameraSystem:
			log.Printf("Found a camera system: %+v", ent)
			cs.CS = ent
		}
	}
	// Once we have found our camera system, hook up our hud system
	cs.HS = HudSystem{Pos: &cs.HUDPos, Camera: cs.CS}
	w.AddSystem(&cs.HS)

	backgroundImage, err := common.LoadedSprite("Backgrounds/stars.png")
	if err != nil {
		log.Printf("Unable to load background image: %+v", err)
	}
	bg := &Background{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable: backgroundImage,
			Scale:    engo.Point{1, 1},
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{0, 0},
			Width:    2048,
			Height:   1024,
		},
	}
	bg.SetZIndex(0)

	worldBounds := engo.AABB{Max: engo.Point{2048, 1024}}

	cs.Camera.TrackingBounds = worldBounds
	cs.R.Add(&bg.BasicEntity, &bg.RenderComponent, &bg.SpaceComponent)
	w.AddSystem(&cs.Camera)

	cs.HUDPos.Set(0, 0)

	healthBar := MenuSprite{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable:    mustLoadSprite("UI/Upgrade/Health.png"),
			Scale:       engo.Point{1, 1},
			StartZIndex: 100,
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{10, 10},
			Width:    258.0,
			Height:   44.0,
		},
	}
	cs.R.Add(&healthBar.BasicEntity, &healthBar.RenderComponent, &healthBar.SpaceComponent)
	cs.HS.Add(&healthBar.BasicEntity, &healthBar.SpaceComponent, engo.Point{10, 10})

	cs.EnergyBar = MenuSprite{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable:    mustLoadSprite("UI/Loading_Bar/Loading_Bar_2_2.png"),
			Scale:       engo.Point{0.6, 1},
			StartZIndex: 100,
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{10, 10},
			Width:    890.0,
			Height:   40.0,
		},
	}
	cs.R.Add(&cs.EnergyBar.BasicEntity, &cs.EnergyBar.RenderComponent, &cs.EnergyBar.SpaceComponent)
	cs.HS.Add(&cs.EnergyBar.BasicEntity, &cs.EnergyBar.SpaceComponent, engo.Point{140, 10})

	engo.Mailbox.Listen(DeleteEntityMessage{}.Type(), func(m engo.Message) {
		delete, ok := m.(DeleteEntityMessage)
		log.Printf("Got delete message: %+v", delete)
		if ok {
			log.Printf("Ent: %+v", cs.Entities[delete.ID])
			ship, foundShip := cs.Ships[delete.ID]
			if foundShip {
				w.RemoveEntity(ship.BasicEntity)
			}
			bullet, foundBullet := cs.Bullets[delete.ID]
			if foundBullet {
				w.RemoveEntity(bullet.BasicEntity)
			}

		}
	})

}

func (cs *ClientScene) NewShip(s *ShipComponent) *ClientShip {

	ship := ClientShip{BasicEntity: ecs.NewBasic()}
	texture, err := common.LoadedSprite("Ships/Ship1/Ship1.png")
	if err != nil {
		log.Printf("UNable to load texture: %+v", err)
	}

	spawnPoint := engo.Point{s.Pos[0], s.Pos[1]}

	ship.RenderComponent = common.RenderComponent{
		Drawable: texture,
		Scale:    engo.Point{1, 1},
	}

	ship.SpaceComponent = common.SpaceComponent{
		Width:  texture.Width() * ship.RenderComponent.Scale.X,
		Height: texture.Height() * ship.RenderComponent.Scale.Y,
	}

	ship.SpaceComponent.SetCenter(spawnPoint)
	log.Printf("SC: %+v %+v", ship.SpaceComponent, spawnPoint)
	ship.RenderComponent.SetZIndex(10)
	cs.R.Add(&ship.BasicEntity, &ship.RenderComponent, &ship.SpaceComponent)

	cs.CPS.Add(&ship)

	return &ship
}

func (cs *ClientScene) NewBullet(b *BulletComponent) *ClientBullet {
	log.Printf("Got a new bullet:%+v", b)
	bullet := ClientBullet{BasicEntity: ecs.NewBasic()}
	texture, err := common.LoadedSprite("Ships/Shots/Shot6/bullet.png")
	if err != nil {
		log.Printf("Unable to load texture: %v", err)
	}

	bullet.BulletComponent = *b
	bullet.RenderComponent = common.RenderComponent{
		Drawable: texture,
		Scale:    engo.Point{1, 1},
	}

	bullet.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{b.Pos[0], b.Pos[1]},
		Width:    texture.Width() * bullet.RenderComponent.Scale.X,
		Height:   texture.Height() * bullet.RenderComponent.Scale.Y,
	}
	bullet.SpaceComponent.SetCenter(engo.Point{texture.Width() / 2, texture.Height() / 2})
	// Bullets go slightly behind ships
	bullet.RenderComponent.SetZIndex(9)
	cs.R.Add(&bullet.BasicEntity, &bullet.RenderComponent, &bullet.SpaceComponent)

	cs.CPS.Add(&bullet)
	log.Printf("Add bullet: %+v", bullet)

	return &bullet
}

func (cs *ClientScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	cs.ServerScene.OnComponentUpdate(op)

	if op.CID == 1000 {
		s, ok := op.Component.(*ShipComponent)
		if !ok {
			log.Printf("Expected ShipComponent but not found")
		}
		ship := cs.Ships[op.ID]
		ship.ShipComponent = *s
		if op.ID == cs.PIS.ID {
			ship, ok := cs.Ships[op.ID]
			if ok {
				cs.Camera.SpaceComponent = &ship.SpaceComponent
				energyPercent := float32(ship.ShipComponent.CurrentEnergy) / float32(ship.ShipComponent.MaxEnergy)

				log.Printf("SC: %+v %f", ship.ShipComponent, energyPercent)
				cs.EnergyBar.RenderComponent.Scale.X = energyPercent * 0.6
			}
		}

	}
	if op.CID == 1001 {
		b, ok := op.Component.(*BulletComponent)
		if !ok {
			log.Printf("Expected Bullet component but not found")
		}
		bullet := cs.Bullets[op.ID]
		bullet.BulletComponent = *b

	}
}

func (cs *ClientScene) OnAddComponent(op sos.AddComponentOp) {
	cs.ServerScene.OnAddComponent(op)

	if op.CID == 1000 {
		s, ok := op.Component.(*ShipComponent)
		if !ok {
			log.Printf("Expected ShipComponent but not found")
		}
		ship := cs.NewShip(s)
		cs.EntToEcs[op.ID] = ship.ID()
		cs.Ships[op.ID] = ship
	}
	if op.CID == 1001 {
		b, ok := op.Component.(*BulletComponent)
		if !ok {
			log.Printf("Expected BulletComponent but not found")
		}
		_, hasBullet := cs.EntToEcs[op.ID]
		if !hasBullet {
			bullet := cs.NewBullet(b)
			cs.EntToEcs[op.ID] = bullet.ID()
			cs.Bullets[op.ID] = bullet
		}

	}

}

func (cs *ClientScene) OnRemoveComponent(op sos.RemoveComponentOp) {
	cs.ServerScene.OnRemoveComponent(op)

	if op.CID == 1001 {
		engo.Mailbox.Dispatch(DeleteEntityMessage{ID: op.ID})
	}
}

func (cs *ClientScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	log.Printf("AUthChanged: %+v", op)
	if op.CID == 1003 && op.Authority == 1 {
		cs.PIS.ID = op.ID
	}
}

func (cs *ClientScene) Type() string {
	return "Client"
}
