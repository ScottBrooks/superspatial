package superspatial

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/sos"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type SpatialPumpSystem struct {
	spatial *sos.SpatialSystem
}

func (ss *SpatialPumpSystem) Remove(ecs.BasicEntity) {}
func (ss *SpatialPumpSystem) Update(dt float32) {
	ss.spatial.Update(dt)
}

type ServerScene struct {
	spatial         *sos.SpatialSystem
	phys            PhysicsSystem
	Entities        map[sos.EntityID]EntityLifecycle
	CurrentEntityID sos.EntityID

	InCritical bool
}

func (*ServerScene) Preload() {}
func (ss *ServerScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	ss.spatial = sos.NewSpatialSystem(ss, "localhost", 7777, "")
	ss.Entities = map[sos.EntityID]EntityLifecycle{}

	w.AddSystem(&ss.phys)
	w.AddSystem(&SpatialPumpSystem{ss.spatial})

}
func (*ServerScene) Type() string { return "Server" }

func (ServerScene) OnDisconnect(op sos.DisconnectOp) {
	os.Exit(0)
}
func (ServerScene) OnFlagUpdate(op sos.FlagUpdateOp) {}
func (ServerScene) OnLogMessage(op sos.LogMessageOp) {
	log.Debugf("Log: %+v", op)
}
func (ServerScene) OnMetrics(op sos.MetricsOp) {}
func (ss *ServerScene) OnCriticalSection(op sos.CriticalSectionOp) {
	log.Debugf("In Critical: %+v", op)
	ss.InCritical = op.In

}
func (ss *ServerScene) OnAddEntity(op sos.AddEntityOp) {
	log.Debugf("OnAddEntity: %+v", op)
	ss.Entities[op.ID] = &SpatialEntity{ID: op.ID}
}
func (ServerScene) OnRemoveEntity(op sos.RemoveEntityOp)         {}
func (ServerScene) OnReserveEntityId(op sos.ReserveEntityIdOp)   {}
func (ServerScene) OnReserveEntityIds(op sos.ReserveEntityIdsOp) {}
func (ServerScene) OnCreateEntity(op sos.CreateEntityOp) {
	log.Debugf("OnCreateEntity: %+v", op)
}
func (ss *ServerScene) OnDeleteEntity(op sos.DeleteEntityOp) {
	log.Debugf("Deleting %d from entities", op.ID)
	delete(ss.Entities, op.ID)
}
func (ServerScene) OnEntityQuery(op sos.EntityQueryOp) {
	log.Debugf("OnEntityQuery: %+v", op)
}
func (ss *ServerScene) OnAddComponent(op sos.AddComponentOp) {
	log.Debugf("OnAddCOmponent: %+v %+v", op, op.Component)
	impWorker, ok := op.Component.(*ImprobableWorker)
	if ok && impWorker.WorkerType == "Client" {
		ss.OnClientConnect(impWorker.WorkerID)
	}

}
func (ServerScene) OnRemoveComponent(op sos.RemoveComponentOp) {
	log.Debugf("OnRemoveComponent: %+v", op)
}
func (ServerScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	log.Printf("OnAuthorityChange: %+v", op)
}
func (ss *ServerScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
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
	case 1000:
		return &ShipComponent{}, nil
	case 1001:
		return &BulletComponent{}, nil
	case 1002:
		return &SpatialGameComponent{}, nil
	case 1003:
		return &PlayerInputComponent{}, nil
	}
	return nil, fmt.Errorf("Unimplemented")
}
func (*ServerScene) WorkerType() string { return "Server" }

// OnClientConnect is called once a worker has connected, and we should create them an entity with a player input component.
func (ss *ServerScene) OnClientConnect(WorkerID string) {
	// Create entity,
	log.Printf("Creating client entity: %s", WorkerID)
	readAttrSet := []WorkerAttributeSet{{[]string{"position"}}}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		1003: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"workerId:" + WorkerID}}}},
	}
	ent := struct {
		PIC  PlayerInputComponent `sos:"1003"`
		ACL  ImprobableACL        `sos:"50"`
		Pos  ImprobablePosition   `sos:"54"`
		Meta ImprobableMetadata   `sos:"53"`
		Ship ShipComponent        `sos:"1000"`
	}{ACL: ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl}, Pos: ImprobablePosition{Coords: Coordinates{0, 0, 2}}, Meta: ImprobableMetadata{Name: "Client"}}
	ss.spatial.CreateEntity(ent)
}
