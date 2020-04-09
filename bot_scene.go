package superspatial

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/sos"
)

type TrackedEntity struct {
	ID sos.EntityID

	Ship        ShipComponent
	Pos         ImprobablePosition
	PlayerInput PlayerInputComponent
}

type BotScene struct {
	ServerScene

	Entities map[sos.EntityID]*TrackedEntity

	BotAI BotAISystem
}

func (*BotScene) Preload() {}
func (bs *BotScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)
	log = log.WithField("worker", bs.ServerScene.WorkerType())
	bs.spatial = sos.NewSpatialSystem(bs, bs.ServerScene.Host, bs.ServerScene.Port, bs.ServerScene.WorkerID, nil)
	bs.ServerScene.Entities = map[sos.EntityID]interface{}{}
	bs.Entities = map[sos.EntityID]*TrackedEntity{}
	bs.ECS = map[uint64]interface{}{}
	bs.Clients = map[sos.EntityID][]sos.EntityID{}

	bs.ServerScene.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}

	bs.BotAI = BotAISystem{SS: &bs.ServerScene}

	log.Printf("New spatialsystem")

	w.AddSystem(&SpatialPumpSystem{&bs.ServerScene})
	w.AddSystem(&bs.BotAI)
}
func (*BotScene) Type() string { return "Bot" }

func (bs *BotScene) OnAddEntity(op sos.AddEntityOp) {
	bs.Entities[op.ID] = &TrackedEntity{ID: op.ID}
}

func (bs *BotScene) OnRemoveEntity(op sos.RemoveEntityOp) {
	if bs.Entities[op.ID] != nil {
		delete(bs.Entities, op.ID)
	}
}
func (bs *BotScene) OnCreateEntity(op sos.CreateEntityOp) {
	bs.ServerScene.OnCreateEntity(op)
}

func (bs *BotScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	ent, ok := bs.Entities[op.ID]

	if !ok {
		return
	}

	switch c := op.Component.(type) {
	case *ImprobablePosition:
		ent.Pos = *c
	case *ShipComponent:
		ent.Ship = *c
	}
}

func (bs *BotScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	if op.CID == cidPlayerInput && op.Authority == 1 {
		bs.BotAI.Ship = bs.Entities[op.ID]
	}

}

func (bs *BotScene) WorkerType() string { return bs.WorkerTypeName }
