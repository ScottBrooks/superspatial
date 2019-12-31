package superspatial

import (
	"fmt"
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/sos"
	"log"
	"os"
)

type Coordinates struct {
	X float64
	Y float64
	Z float64
}

type WorkerAttributeSet struct {
	Attribute []string `sos:"attribute"`
}

type WorkerRequirementSet struct {
	AttributeSet []WorkerAttributeSet `sos:"attribute_set"`
}

type ImprobablePosition struct {
	Coords Coordinates
}
type ImprobableWorker struct {
	WorkerID   string
	WorkerType string
	//Connection

}

type ImprobableACL struct {
	ReadAcl           WorkerRequirementSet            `sos:"read_acl"`
	ComponentWriteAcl map[uint32]WorkerRequirementSet `sos:"component_write_acl"`
}

type SpatialGame struct {
	EntityID sos.EntityID
}

type SpatialPumpSystem struct {
	spatial *sos.SpatialSystem
}

func (ss *SpatialPumpSystem) Remove(ecs.BasicEntity) {}
func (ss *SpatialPumpSystem) Update(dt float32) {
	ss.spatial.Update(dt)
}

type ServerScene struct {
	spatial  *sos.SpatialSystem
	phys     PhysicsSystem
	Entities map[sos.EntityID]interface{}
}

func (*ServerScene) Preload() {}
func (ss *ServerScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	ss.spatial = sos.NewSpatialSystem(ss, "localhost", 7777, "")

	w.AddSystem(&ss.phys)
	w.AddSystem(&SpatialPumpSystem{ss.spatial})

}
func (*ServerScene) Type() string { return "Server" }

func (ServerScene) OnDisconnect(op sos.DisconnectOp) {
	os.Exit(0)
}
func (ServerScene) OnFlagUpdate(op sos.FlagUpdateOp) {}
func (ServerScene) OnLogMessage(op sos.LogMessageOp) {
	log.Printf("Log: %+v", op)
}
func (ServerScene) OnMetrics(op sos.MetricsOp) {}
func (ServerScene) OnCriticalSection(op sos.CriticalSectionOp) {
	log.Printf("In Critical: %+v", op)
}
func (ServerScene) OnAddEntity(op sos.AddEntityOp) {
	log.Printf("OnAddEntity: %+v", op)
}
func (ServerScene) OnRemoveEntity(op sos.RemoveEntityOp)         {}
func (ServerScene) OnReserveEntityId(op sos.ReserveEntityIdOp)   {}
func (ServerScene) OnReserveEntityIds(op sos.ReserveEntityIdsOp) {}
func (ServerScene) OnCreateEntity(op sos.CreateEntityOp)         {}
func (ServerScene) OnDeleteEntity(op sos.DeleteEntityOp)         {}
func (ServerScene) OnEntityQuery(op sos.EntityQueryOp) {
	log.Printf("OnEntityQuery: %+v", op)
}
func (ServerScene) OnAddComponent(op sos.AddComponentOp) {
	log.Printf("OnAddCOmponent: %+v %+v", op, op.Component)
}
func (ServerScene) OnRemoveComponent(op sos.RemoveComponentOp) {
	log.Printf("OnRemoveComponent: %+v", op)
}
func (ServerScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	log.Printf("OnAuthorityChange: %+v", op)
}
func (ServerScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	log.Printf("COmpUpdate: %+v", op)
	log.Printf("Component: %+v", op.Component)
}
func (ServerScene) OnCommandRequest(op sos.CommandRequestOp) {
	log.Printf("OnCommandRequest: %+v", op)
}
func (ServerScene) OnCommandResponse(op sos.CommandResponseOp) {
	log.Printf("OnCommandResponse: %+v", op)
}
func (ss *ServerScene) AllocComponent(ID sos.EntityID, CID sos.ComponentID) (interface{}, error) {
	switch CID {
	case 50:
		return &ImprobableACL{ComponentWriteAcl: map[uint32]WorkerRequirementSet{}}, nil
	case 54:
		return &ImprobablePosition{}, nil
	case 60:
		return &ImprobableWorker{}, nil
	case 1002:
		return &SpatialGame{}, nil
	}
	return nil, fmt.Errorf("Unimplemented")
}
func (*ServerScene) WorkerType() string { return "Server" }
