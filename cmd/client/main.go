package main

import (
	"image/color"
	"log"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

var (
	scrollSpeed float32 = 700
	worldWidth  int     = 800
	worldHeight int     = 600
)

type MainMenuScene struct{}

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

func (*MainMenuScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	engo.Input.RegisterButton("Up", engo.KeyArrowUp)
	engo.Input.RegisterButton("Down", engo.KeyArrowDown)
	engo.Input.RegisterButton("Left", engo.KeyArrowLeft)
	engo.Input.RegisterButton("Right", engo.KeyArrowRight)
	engo.Input.RegisterButton("Enter", engo.KeyEnter)

	common.SetBackground(color.White)
	rs := common.RenderSystem{}

	ss := SelectionSystem{}
	w.AddSystem(&rs)
	w.AddSystem(&ss)

	bg := MenuSprite{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable: mustLoadSprite("UI/Main_Menu/BG.png"),
			Scale:    engo.Point{1, 1},
		},
	}

	startBut := MenuSprite{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable: mustLoadSprite("UI/Main_Menu/Start_BTN.png"),
			Scale:    engo.Point{1, 1},
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{250, 100},
			Width:    410.0,
			Height:   121.0,
		},
	}
	exitBut := MenuSprite{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable: mustLoadSprite("UI/Main_Menu/Exit_BTN.png"),
			Scale:    engo.Point{1, 1},
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{250, 221},
			Width:    410.0,
			Height:   121.0,
		},
	}

	rs.Add(&bg.BasicEntity, &bg.RenderComponent, &bg.SpaceComponent)
	rs.Add(&startBut.BasicEntity, &startBut.RenderComponent, &startBut.SpaceComponent)
	rs.Add(&exitBut.BasicEntity, &exitBut.RenderComponent, &exitBut.SpaceComponent)

	ss.Add(&startBut.BasicEntity, &startBut.RenderComponent, func() {
		log.Printf("START")
	})
	ss.Add(&exitBut.BasicEntity, &exitBut.RenderComponent, func() {
		engo.Exit()
	})
}

func (*MainMenuScene) Preload() {
	assets := []string{
		"UI/Main_Menu/BG.png",
		"UI/Main_Menu/Start_BTN.png",
		"UI/Main_Menu/Exit_BTN.png",
	}
	for _, asset := range assets {
		err := engo.Files.Load(asset)
		if err != nil {
			log.Fatalf("Error loading asset: %v", err)
		}
	}
}

func (*MainMenuScene) Type() string { return "Game" }

type selectable struct {
	*ecs.BasicEntity
	*common.RenderComponent
	exec func()
}
type SelectionSystem struct {
	selectables []selectable
	current     int
}

func (*SelectionSystem) Remove(ecs.BasicEntity) {}

func (ss *SelectionSystem) Update(dt float32) {
	ss.selectables[ss.current].RenderComponent.Color = color.RGBA{255, 255, 255, 255}
	if engo.Input.Button("Up").JustReleased() {
		log.Printf("Pressed up")
		ss.current--
	}
	if engo.Input.Button("Down").JustReleased() {
		log.Printf("Pressed down")
		ss.current++
	}
	if engo.Input.Button("Enter").JustReleased() {
		ss.selectables[ss.current].exec()
	}
	if ss.current < 0 {
		ss.current = len(ss.selectables) - 1
	}
	if ss.current >= len(ss.selectables) {
		ss.current = 0
	}
	ss.selectables[ss.current].RenderComponent.Color = color.RGBA{255, 0, 0, 255}

}

func (ss *SelectionSystem) Add(e *ecs.BasicEntity, rc *common.RenderComponent, exec func()) {
	ss.selectables = append(ss.selectables, selectable{e, rc, exec})

}

func main() {
	opts := engo.RunOptions{
		Title:          "SuperSpatial",
		Width:          worldWidth,
		Height:         worldHeight,
		StandardInputs: true,
	}

	engo.Run(opts, &MainMenuScene{})
}
