package main

import (
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/ScottBrooks/superspatial"

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

func (mm *MainMenuScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	engo.Input.RegisterButton("Up", engo.KeyArrowUp)
	engo.Input.RegisterButton("Down", engo.KeyArrowDown)
	engo.Input.RegisterButton("Left", engo.KeyArrowLeft)
	engo.Input.RegisterButton("Right", engo.KeyArrowRight)
	engo.Input.RegisterButton("Enter", engo.KeyEnter)

	common.SetBackground(color.White)
	rs := common.RenderSystem{}

	ss := superspatial.SelectionSystem{}
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
			Drawable:    mustLoadSprite("UI/Main_Menu/Start_BTN.png"),
			Scale:       engo.Point{1, 1},
			StartZIndex: 100,
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
			Drawable:    mustLoadSprite("UI/Main_Menu/Exit_BTN.png"),
			Scale:       engo.Point{1, 1},
			StartZIndex: 100,
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
		ss.Reset()
		engo.SetSceneByName("Game", false)
		mm.ConnectToSpatial()
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

func (*MainMenuScene) Type() string { return "Menu" }

func (*MainMenuScene) ConnectToSpatial() {
	log.Printf("Connecting to spatial")
}

func main() {
	rand.Seed(time.Now().Unix())
	var useGraphics bool
	displayEnv := os.Getenv("DISPLAY")
	if displayEnv != "" {
		useGraphics = true
	}

	opts := engo.RunOptions{
		Title:          "SuperSpatial",
		Width:          worldWidth,
		Height:         worldHeight,
		StandardInputs: true,
		HeadlessMode:   !useGraphics,
	}
	engo.RegisterScene(&ClientGameScene{})

	if useGraphics {
		engo.Run(opts, &MainMenuScene{})
	} else {
		engo.Run(opts, &ClientGameScene{})
	}
}
