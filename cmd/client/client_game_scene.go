package main

import (
	"image/color"

	"github.com/ScottBrooks/sos"
	"github.com/ScottBrooks/superspatial"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

type ClientGameScene struct {
	spatial *sos.SpatialSystem
}

func (*ClientGameScene) Preload() {
}
func (*ClientGameScene) Type() string {
	return "Game"
}
func (cgs *ClientGameScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	common.SetBackground(color.White)
	rs := common.RenderSystem{}
	w.AddSystem(&rs)

	adapter := superspatial.SpatialAdapter{}

	cgs.spatial = sos.NewSpatialSystem(&adapter, "localhost", 7777, "")
}
