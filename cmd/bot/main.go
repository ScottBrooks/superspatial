package main

import (
	"flag"
	"math/rand"
	"time"

	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/superspatial"
)

func main() {
	rand.Seed(time.Now().Unix())
	host := flag.String("host", "127.0.0.1", "receptionist host address")
	port := flag.Int("port", 7777, "receptionist port")
	workerID := flag.String("worker", "", "worker ID")
	development := flag.Bool("dev", true, "set to false if to try to fork ./server")
	flag.Parse()

	opts := engo.RunOptions{
		Title:        "SuperSpatial",
		HeadlessMode: true,
		FPSLimit:     30,
	}
	ss := superspatial.BotScene{ServerScene: superspatial.ServerScene{WorkerTypeName: "Bot", Host: *host, Port: *port, WorkerID: *workerID, Development: *development}}

	engo.Run(opts, &ss)
}
