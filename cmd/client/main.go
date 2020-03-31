package main

import (
	"bytes"
	"flag"
	"image/color"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ScottBrooks/superspatial"
	"golang.org/x/image/font/gofont/gosmallcaps"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

var (
	scrollSpeed float32 = 700
	worldWidth  int     = 1024
	worldHeight int     = 768
)

type MainMenuScene struct {
	Font *common.Font
}

type Text struct {
	ecs.BasicEntity
	common.SpaceComponent
	common.RenderComponent
}
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
	engo.Input.RegisterButton("Space", engo.KeySpace)

	common.SetBackground(color.White)
	rs := common.RenderSystem{}

	ss := superspatial.SelectionSystem{}
	w.AddSystem(&rs)
	w.AddSystem(&ss)

	mm.Font = &common.Font{
		URL:  "go.ttf",
		FG:   color.White,
		Size: 64,
	}
	mm.Font.CreatePreloaded()

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
			Position: engo.Point{307, 326},
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
			Position: engo.Point{307, 500},
			Width:    410.0,
			Height:   121.0,
		},
	}
	title := Text{
		BasicEntity: ecs.NewBasic(),
		RenderComponent: common.RenderComponent{
			Drawable: common.Text{
				Font: mm.Font,
				Text: "Stabby Ships",
			},
			Scale:       engo.Point{1, 1},
			StartZIndex: 6,
		},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{307, 60},
			Width:    200,
			Height:   60,
		},
	}

	rs.Add(&bg.BasicEntity, &bg.RenderComponent, &bg.SpaceComponent)
	rs.Add(&startBut.BasicEntity, &startBut.RenderComponent, &startBut.SpaceComponent)
	rs.Add(&exitBut.BasicEntity, &exitBut.RenderComponent, &exitBut.SpaceComponent)
	rs.Add(&title.BasicEntity, &title.RenderComponent, &title.SpaceComponent)

	ss.Add(&startBut.BasicEntity, &startBut.RenderComponent, func() {
		ss.Reset()
		engo.SetSceneByName("Client", true)
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
	engo.Files.LoadReaderData("go.ttf", bytes.NewReader(gosmallcaps.TTF))
	for _, asset := range assets {
		err := engo.Files.Load(asset)
		if err != nil {
			log.Fatalf("Error loading asset: %v", err)
		}
	}
}

func (*MainMenuScene) Type() string { return "Menu" }

func main() {
	// Chdir to the directory our exe started in.
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat("assets"); os.IsNotExist(err) {
		log.Printf("Can't find assets folder, changing our directory to path of exe.")
		exePath := filepath.Dir(ex)
		err = os.Chdir(exePath)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := os.Stat("assets"); os.IsNotExist(err) {
			log.Fatal("Still can't find assets folder, quitting")
		}
	}

	locator := flag.String("locator", "locator.improbable.io", "locator address")
	pit := flag.String("playerIdentityToken", "", "player identity token")
	lt := flag.String("loginToken", "", "login token")
	project := flag.String("project", "", "project name")

	host := flag.String("host", "127.0.0.1", "receptionist host address")
	port := flag.Int("port", 7777, "receptionist port")
	workerID := flag.String("worker", "", "worker ID")
	flag.Parse()

	rand.Seed(time.Now().Unix())
	var useGraphics bool
	displayEnv := os.Getenv("DISPLAY")
	if displayEnv != "" || runtime.GOOS == "windows" {
		useGraphics = true
	}

	cs := superspatial.ClientScene{ServerScene: superspatial.ServerScene{WorkerTypeName: "LauncherClient", Host: *host, Port: *port, WorkerID: *workerID, Locator: *locator, PIT: *pit, LT: *lt, ProjectName: *project}}

	opts := engo.RunOptions{
		Title:          "SuperSpatial",
		Width:          worldWidth,
		Height:         worldHeight,
		StandardInputs: true,
		HeadlessMode:   !useGraphics,
		FPSLimit:       30,
	}
	engo.RegisterScene(&cs)

	if useGraphics {
		engo.Run(opts, &MainMenuScene{})
	} else {
		engo.Run(opts, &cs)
	}
}
