package main

import (
	"image/color"
	"log"

	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"

	"github.com/ScottBrooks/sos"
)

type ClientGameScene struct {
	spatial *sos.SpatialSystem
}

func (cgs *ClientGameScene) Setup(u engo.Updater) {
	//w, _ := u.(*ecs.World)
	common.SetBackground(color.White)
	log.Printf("Connect to spatial here")

}

func (cgs *ClientGameScene) Preload() {
}
func (cgs *ClientGameScene) Type() string {
	return "Game"
}
