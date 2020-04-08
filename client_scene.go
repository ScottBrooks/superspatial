package superspatial

import (
	"bytes"
	"fmt"
	"image/color"
	"time"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"golang.org/x/image/font/gofont/gosmallcaps"

	"github.com/ScottBrooks/sos"
)

type MenuSprite struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
}

type ClientScene struct {
	ServerScene

	CS        *common.CameraSystem
	R         common.RenderSystem
	Anim      common.AnimationSystem
	Camera    common.EntityScroller
	PIS       PlayerInputSystem
	CPS       ClientPredictionSystem
	HS        HudSystem
	HUDPos    engo.Point
	Font      *common.Font
	Explosion *common.Animation

	EntToEcs map[sos.EntityID]uint64
	Ships    map[sos.EntityID]*ClientShip
	Effects  map[sos.EntityID]*ClientEffect
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

	if pis.ID != 0 {
		pis.spatial.UpdateComponent(pis.ID, cidPlayerInput, p)
	}

}

type Text struct {
	ecs.BasicEntity
	common.SpaceComponent
	common.RenderComponent
}

type ClientShip struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
	text Text

	ShipComponent
	WorkerComponent
}

func (cs *ClientShip) Predict(dt float32) {

	// Add 50% of our new position and 50% of our old position.
	newPos := cs.ShipComponent.Pos.Add(cs.ShipComponent.Vel.Mul(dt))
	avgPos := cs.ShipComponent.Pos.Mul(0.5).Add(newPos.Mul(0.5))

	cs.SpaceComponent.SetCenter(engo.Point{X: avgPos[0], Y: avgPos[1]})
	cs.SpaceComponent.Rotation = cs.ShipComponent.Angle - 90
	cs.text.SpaceComponent = cs.SpaceComponent
	cs.text.SpaceComponent.Rotation = 0
}

type Background struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
}

func (cs *ClientScene) Preload() {
	for _, asset := range []string{
		"Ships/ship-aqua.png",
		"Ships/ship-blue.png",
		"Ships/ship-green.png",
		"Ships/ship-orange.png",
		"Ships/ship-red.png",
		"Backgrounds/stars.png",
		"UI/Upgrade/Health.png",
		"UI/Loading_Bar/Loading_Bar_2_1.png",
		"UI/Loading_Bar/Loading_Bar_2_2.png",
		"UI/Loading_Bar/Loading_Bar_2_3.png",
		"Ships/Explosion/explosion.png",
	} {
		err := engo.Files.Load(asset)
		if err != nil {
			panic(err)
		}
	}
	err := engo.Files.LoadReaderData("go.ttf", bytes.NewReader(gosmallcaps.TTF))
	if err != nil {
		panic(err)
	}

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
	cs.Effects = map[sos.EntityID]*ClientEffect{}
	cs.Explosion = &common.Animation{Name: "explosion", Frames: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}}

	cs.PIS.spatial = cs.ServerScene.spatial

	w.AddSystem(&cs.R)
	w.AddSystem(&cs.PIS)
	w.AddSystem(&SpatialPumpSystem{&cs.ServerScene})
	w.AddSystem(&cs.CPS)
	w.AddSystem(&cs.Anim)
	for _, sys := range w.Systems() {
		switch ent := sys.(type) {
		case *common.CameraSystem:
			log.Printf("Found a camera system: %+v", ent)
			cs.CS = ent
		}
	}

	cs.Font = &common.Font{
		URL:  "go.ttf",
		FG:   color.White,
		Size: 24,
	}
	err := cs.Font.CreatePreloaded()
	if err != nil {
		log.Printf("Err preloading font: %+v", err)
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
			Scale:    engo.Point{X: 1, Y: 1},
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{X: 0, Y: 0},
			Width:    worldBounds.Max.X,
			Height:   worldBounds.Max.Y,
		},
	}
	bg.SetZIndex(0)

	cs.Camera.TrackingBounds = worldBounds
	cs.R.Add(&bg.BasicEntity, &bg.RenderComponent, &bg.SpaceComponent)
	w.AddSystem(&cs.Camera)

	cs.HUDPos.Set(0, 0)

	engo.Mailbox.Listen(DeleteEntityMessage{}.Type(), func(m engo.Message) {
		delete, ok := m.(DeleteEntityMessage)
		log.Printf("Got delete message: %+v", delete)
		if ok {
			log.Printf("Ent: %+v", cs.Entities[delete.ID])
			ship := cs.Ships[delete.ID]
			if ship != nil {
				w.RemoveEntity(ship.BasicEntity)
				w.RemoveEntity(ship.text.BasicEntity)
			}
			effect := cs.Effects[delete.ID]
			if effect != nil {
				w.RemoveEntity(effect.BasicEntity)
			}
		}
	})

}

func (cs *ClientScene) NewShip(s *ShipComponent) *ClientShip {

	ship := ClientShip{BasicEntity: ecs.NewBasic()}
	texture, err := common.LoadedSprite("Ships/ship-aqua.png")
	if err != nil {
		log.Printf("UNable to load texture: %+v", err)
	}

	spawnPoint := engo.Point{X: s.Pos[0], Y: s.Pos[1]}

	ship.RenderComponent = common.RenderComponent{
		Drawable: texture,
		Scale:    engo.Point{X: 1, Y: 1},
	}

	ship.SpaceComponent = common.SpaceComponent{
		Width:  texture.Width() * ship.RenderComponent.Scale.X,
		Height: texture.Height() * ship.RenderComponent.Scale.Y,
	}

	ship.text = Text{BasicEntity: ecs.NewBasic()}
	ship.text.RenderComponent.Drawable = common.Text{
		Font: cs.Font,
		Text: "Ship",
	}
	ship.text.SpaceComponent = ship.SpaceComponent
	ship.text.RenderComponent.SetZIndex(11)

	ship.SpaceComponent.SetCenter(spawnPoint)
	ship.RenderComponent.SetZIndex(10)

	cs.R.Add(&ship.BasicEntity, &ship.RenderComponent, &ship.SpaceComponent)

	cs.R.Add(&ship.text.BasicEntity, &ship.text.RenderComponent, &ship.text.SpaceComponent)

	cs.CPS.Add(&ship)

	return &ship
}

func (cs *ClientScene) NewEffect(e *EffectComponent) *ClientEffect {
	log.Printf("Got a new effect: %v", e)

	spriteSheet := common.NewSpritesheetFromFile("Ships/Explosion/explosion.png", 128, 128)

	effect := ClientEffect{BasicEntity: ecs.NewBasic()}
	effect.AnimationComponent = common.NewAnimationComponent(spriteSheet.Drawables(), 0.1)
	effect.AnimationComponent.AddDefaultAnimation(cs.Explosion)
	effect.EffectComponent = *e

	switch e.Id {
	case 1:
		effect.RenderComponent = common.RenderComponent{
			Drawable: spriteSheet.Cell(0),
			Scale:    engo.Point{X: 1, Y: 1},
		}
		effect.SpaceComponent = common.SpaceComponent{
			Position: engo.Point{X: e.Pos[0], Y: e.Pos[1]},
			Width:    128 * effect.RenderComponent.Scale.X,
			Height:   128 * effect.RenderComponent.Scale.Y,
		}
		effect.SpaceComponent.SetCenter(engo.Point{X: e.Pos[0], Y: e.Pos[1]})
		// Effects go slightly behind ships
		effect.RenderComponent.SetZIndex(9)

		log.Printf("Space Component: %+v", effect.SpaceComponent)
	}

	cs.R.Add(&effect.BasicEntity, &effect.RenderComponent, &effect.SpaceComponent)
	cs.Anim.Add(&effect.BasicEntity, &effect.AnimationComponent, &effect.RenderComponent)

	return &effect
}

func (cs *ClientScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	cs.ServerScene.OnComponentUpdate(op)

	switch c := op.Component.(type) {
	case *ShipComponent:
		ship := cs.Ships[op.ID]
		ship.ShipComponent = *c
		if op.ID == cs.PIS.ID {
			ship, ok := cs.Ships[op.ID]
			if ok {
				cs.Camera.SpaceComponent = &ship.SpaceComponent
			}
		}
	case *WorkerComponent:
		ship, ok := cs.Ships[op.ID]
		if ok {
			if ship.WorkerComponent.WorkerID != c.WorkerID {
				ship.text.RenderComponent.Drawable = common.Text{
					Font: cs.Font,
					Text: fmt.Sprintf("Worker: %d", c.WorkerID),
				}
			}
			ship.WorkerComponent = *c
		}
	}
}

func (cs *ClientScene) OnAddComponent(op sos.AddComponentOp) {
	//cs.ServerScene.OnAddComponent(op)
	log.Printf("Client add componeont: %+v", op)

	switch c := op.Component.(type) {
	case *ShipComponent:
		ship := cs.NewShip(c)
		cs.EntToEcs[op.ID] = ship.ID()
		cs.Ships[op.ID] = ship
	case *EffectComponent:
		_, hasEffect := cs.EntToEcs[op.ID]
		if !hasEffect {
			effect := cs.NewEffect(c)
			cs.EntToEcs[op.ID] = effect.ID()
			cs.Effects[op.ID] = effect
		}
	}

}

func (cs *ClientScene) OnRemoveEntity(op sos.RemoveEntityOp) {
	cs.ServerScene.OnRemoveEntity(op)

	engo.Mailbox.Dispatch(DeleteEntityMessage{ID: op.ID})
}

func (cs *ClientScene) OnRemoveComponent(op sos.RemoveComponentOp) {
	cs.ServerScene.OnRemoveComponent(op)

}

func (cs *ClientScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	log.Printf("AUthChanged: %+v", op)
	if op.CID == cidPlayerInput && op.Authority == 1 {
		cs.PIS.ID = op.ID
	}
}

func (cs *ClientScene) Type() string {
	return "Client"
}

type ClientEffect struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
	common.AnimationComponent

	EffectComponent `sos:"1006"`
	CreatedAt       time.Time
}

func (ce *ClientEffect) HasExpired() bool {
	return time.Since(ce.CreatedAt) > time.Duration(ce.EffectComponent.Expiry)
}
