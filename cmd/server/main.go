package main

import (
	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/superspatial"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	opts := engo.RunOptions{
		Title:        "SuperSpatial",
		HeadlessMode: true,
	}

	engo.Run(opts, &superspatial.ServerScene{})
}
