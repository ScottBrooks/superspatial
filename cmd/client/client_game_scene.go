package main
import(
	"github.com/EngoEngine/engo"
)

type ClientGameScene struct {}

func (*ClientGameScene) Preload() {
}
func (*ClientGameScene) Type() string {
	return "Game"
}
func (*ClientGameScene) Setup(u engo.Updater) {
}


