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
	SS *ServerScene
}

func (sps *SpatialPumpSystem) Remove(ecs.BasicEntity) {}
func (sps *SpatialPumpSystem) Update(dt float32) {
	sps.SS.spatial.Update(dt)

	for _, e := range sps.SS.Entities {
		switch ent := e.(type) {
		case *Ship:
			ent.UpdatePos(dt)
			//log.Printf("Got a ship: %+v", ent)
			sps.SS.spatial.UpdateComponent(ent.ID, 1000, ent.Ship)
			sps.SS.spatial.UpdateComponent(ent.ID, 54, ent.Pos)
		}
	}
}

type ServerScene struct {
	spatial         *sos.SpatialSystem
	phys            PhysicsSystem
	Entities        map[sos.EntityID]interface{}
	CurrentEntityID sos.EntityID
	WorkerTypeName  string

	InCritical   bool
	OnCreateFunc map[sos.RequestID]func(ID sos.EntityID)
}

func (*ServerScene) Preload() {}
func (ss *ServerScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	ss.spatial = sos.NewSpatialSystem(ss, "localhost", 7777, "")
	ss.Entities = map[sos.EntityID]interface{}{}
	ss.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}

	w.AddSystem(&ss.phys)
	w.AddSystem(&SpatialPumpSystem{ss})

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
	//ss.Entities[op.ID] = &SpatialEntity{ID: op.ID}
}
func (ServerScene) OnRemoveEntity(op sos.RemoveEntityOp)         {}
func (ServerScene) OnReserveEntityId(op sos.ReserveEntityIdOp)   {}
func (ServerScene) OnReserveEntityIds(op sos.ReserveEntityIdsOp) {}
func (ss *ServerScene) OnCreateEntity(op sos.CreateEntityOp) {
	log.Debugf("OnCreateEntity: %+v", op)
	fn, ok := ss.OnCreateFunc[op.RID]
	if ok {
		fn(op.ID)
	}
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
func (ss *ServerScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	log.Printf("OnAuthorityChange: %+v", op)
	if op.CID == 58 && op.Authority == 1 {
		e := ss.Entities[op.ID]
		log.Printf("E: %+v", e)
		s, ok := e.(*Ship)
		if !ok {
			log.Printf("UNable to cast :%+v to ship", e)
		}
		if ok {
			ss.spatial.UpdateComponent(s.ID, 58, s.Interest)
		}
	}
}
func (ss *ServerScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	ent, ok := ss.Entities[op.ID].(*Ship)
	if ok {
		switch op.CID {
		case 1003:
			ent.PIC = *op.Component.(*PlayerInputComponent)
		}
	}

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
func (ss *ServerScene) WorkerType() string { return ss.WorkerTypeName }

// OnClientConnect is called once a worker has connected, and we should create them an entity with a player input component.
func (ss *ServerScene) OnClientConnect(WorkerID string) {
	// Create entity,
	log.Printf("Creating client entity: %s", WorkerID)
	readAttrSet := []WorkerAttributeSet{
		{[]string{"position"}},
		{[]string{"client"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		1003: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"workerId:" + WorkerID}}}},
		1000: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"position"}}}},
		58:   WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"position"}}}},
		54:   WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"position"}}}},
	}
	relSphere := QBIRelativeSphereConstraint{Radius: 100}
	ent := Ship{ACL: ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl}, Pos: ImprobablePosition{Coords: Coordinates{0, 0, 2}}, Meta: ImprobableMetadata{Name: "Client"}, Interest: ImprobableInterest{
		Interest: map[uint32]ComponentInterest{
			1003: ComponentInterest{
				Queries: []QBIQuery{
					{Constraint: QBIConstraint{RelativeSphereConstraint: &relSphere}, ResultComponents: []uint32{1000, 54, 53}},
				},
			},
		},
	}, Mass: 1000.0}
	reqID := ss.spatial.CreateEntity(ent)
	ss.OnCreateFunc[reqID] = func(ID sos.EntityID) {
		ent.ID = ID
		ent.SetupQBI()
		ss.Entities[ID] = &ent
		log.Printf("Interest: %+v", ent.Interest)
	}

}
