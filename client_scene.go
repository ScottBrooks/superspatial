package superspatial

type ClientScene struct {
	ServerScene

	R common.RenderSystem
}

func (cs *ClientScene) Preload() {}
func (cs *ClientScene) Setup(u engo.Updater) {
	cs.ServerScene.Setup(u)

	w, _ := u.(*ecs.World)

	w.AddSystem(&cs.R)

}

func (cs *ClientScene) Type() string {
	return "Client"
}
