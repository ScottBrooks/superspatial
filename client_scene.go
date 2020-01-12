package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"

	log "github.com/sirupsen/logrus"

	"github.com/ScottBrooks/sos"
)

type ClientScene struct {
	ServerScene

	R      common.RenderSystem
	Camera common.EntityScroller
	PIS    PlayerInputSystem
	CPS    ClientPredictionSystem

	EntToEcs map[sos.EntityID]uint64
	Ships    map[sos.EntityID]*ClientShip
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

type ClientShip struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent

	ShipComponent
}

func (cs *ClientShip) UpdatePos(dt float32) {

	cs.ShipComponent.Pos = cs.ShipComponent.Pos.Add(cs.ShipComponent.Vel.Mul(dt))
	//	aabb := engo.AABB{Max: engo.Point{2048, 1024}}
	//	cs.ShipComponent.Pos = clampToAABB(cs.ShipComponent.Pos, aabb)

	cs.SpaceComponent.Position = engo.Point{cs.ShipComponent.Pos[0], cs.ShipComponent.Pos[1]}
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

}
func (cs *ClientScene) Setup(u engo.Updater) {

	w, _ := u.(*ecs.World)

	cs.spatial = sos.NewSpatialSystem(cs, "localhost", 7777, "")
	cs.Entities = map[sos.EntityID]interface{}{}
	cs.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}
	cs.EntToEcs = map[sos.EntityID]uint64{}
	cs.Ships = map[sos.EntityID]*ClientShip{}

	cs.PIS.spatial = cs.ServerScene.spatial

	w.AddSystem(&cs.R)
	w.AddSystem(&cs.PIS)
	w.AddSystem(&SpatialPumpSystem{&cs.ServerScene})
	w.AddSystem(&cs.CPS)

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

	worldBounds := engo.AABB{Max: engo.Point{2048, 1024}}

	cs.Camera.TrackingBounds = worldBounds
	cs.R.Add(&bg.BasicEntity, &bg.RenderComponent, &bg.SpaceComponent)
	w.AddSystem(&cs.Camera)

}

func (cs *ClientScene) NewShip(s *ShipComponent) *ClientShip {

	ship := ClientShip{BasicEntity: ecs.NewBasic()}
	texture, err := common.LoadedSprite("Ships/Ship1/Ship1.png")
	if err != nil {
		log.Printf("UNable to load texture: %+v", err)
	}

	ship.RenderComponent = common.RenderComponent{
		Drawable: texture,
		Scale:    engo.Point{1, 1},
	}

	ship.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{s.Pos[0], s.Pos[1]},
		Width:    texture.Width() * ship.RenderComponent.Scale.X,
		Height:   texture.Height() * ship.RenderComponent.Scale.Y,
	}
	ship.SpaceComponent.SetCenter(engo.Point{texture.Width() / 2, texture.Height() / 2})
	ship.RenderComponent.SetZIndex(10)
	cs.R.Add(&ship.BasicEntity, &ship.RenderComponent, &ship.SpaceComponent)

	cs.CPS.Add(&ship)

	cs.Camera.SpaceComponent = &ship.SpaceComponent
	log.Printf("SC: %+v", cs.Camera)
	return &ship
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
