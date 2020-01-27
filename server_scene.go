package superspatial

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/go-gl/mathgl/mgl32"
	log "github.com/sirupsen/logrus"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
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
			ent.Update(dt)
			if ent.HasAuthority {
				sps.SS.spatial.UpdateComponent(ent.ID, 1000, ent.Ship)
				sps.SS.spatial.UpdateComponent(ent.ID, 54, ent.Pos)
			}
		case *Bullet:
			ent.Update(dt)
			if ent.HasAuthority {
				sps.SS.spatial.UpdateComponent(ent.ID, 1001, ent.Bullet)
				sps.SS.spatial.UpdateComponent(ent.ID, 54, ent.Pos)
				if !sps.SS.InBounds(ent.Bullet.Pos) {
					log.Printf("Bullet is out of bounds: %d %+v", ent.ID, ent.Bullet.Pos)
					engo.Mailbox.Dispatch(DeleteEntityMessage{ID: ent.ID})
				}
			}
		}
	}
}

type DeleteEntityMessage struct {
	ID sos.EntityID
}

func (DeleteEntityMessage) Type() string {
	return "DeleteEntityMessage"
}

type ServerScene struct {
	Host        string
	Port        int
	WorkerID    string
	ProjectName string
	Locator     string
	PIT         string
	LT          string

	spatial  *sos.SpatialSystem
	phys     PhysicsSystem
	Entities map[sos.EntityID]interface{}
	ECS      map[uint64]interface{}
	// Clients maps the entity id of their client worker with the entities we want to delete when they disconnect(ship?)
	Clients         map[sos.EntityID][]sos.EntityID
	CurrentEntityID sos.EntityID
	WorkerTypeName  string

	InCritical   bool
	OnCreateFunc map[sos.RequestID]func(ID sos.EntityID)

	Bounds engo.AABB

	CollisionSystem common.CollisionSystem
}

func (*ServerScene) Preload() {}
func (ss *ServerScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	ss.spatial = sos.NewSpatialSystem(ss, ss.Host, ss.Port, ss.WorkerID, nil)
	ss.Entities = map[sos.EntityID]interface{}{}
	ss.ECS = map[uint64]interface{}{}
	ss.Clients = map[sos.EntityID][]sos.EntityID{}
	ss.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}

	ss.Bounds = engo.AABB{Max: engo.Point{2048, 1024}}

	ss.CollisionSystem = common.CollisionSystem{Solids: 1}

	w.AddSystem(&ss.phys)
	w.AddSystem(&SpatialPumpSystem{ss})
	w.AddSystem(&AttackSystem{SS: ss})
	w.AddSystem(&ss.CollisionSystem)

	engo.Mailbox.Listen(DeleteEntityMessage{}.Type(), func(msg engo.Message) {
		dem, ok := msg.(DeleteEntityMessage)
		if !ok {
			return
		}
		ss.spatial.Delete(dem.ID)
	})

	engo.Mailbox.Listen(common.CollisionMessage{}.Type(), func(msg engo.Message) {
		collision, ok := msg.(common.CollisionMessage)
		if ok {
			ship, foundShip := ss.ECS[collision.To.ID()].(*Ship)
			bullet, foundBullet := ss.ECS[collision.Entity.ID()].(*Bullet)
			log.Printf("FS: %v FB: %v", foundShip, foundBullet)
			if foundShip && foundBullet && bullet.Bullet.ShipID != ship.ID {
				w.RemoveEntity(bullet.BasicEntity)
				engo.Mailbox.Dispatch(DeleteEntityMessage{ID: bullet.ID})
				ship.TakeDamage(bullet.Bullet.Damage)

				log.Printf("Hit ship: %+v", ship)

			}

		}
	})

}
func (*ServerScene) Type() string { return "Server" }

func (ss *ServerScene) InBounds(pos mgl32.Vec3) bool {
	if pos[0] < ss.Bounds.Min.X {
		return false
	}
	if pos[0] > ss.Bounds.Max.X {
		return false
	}
	if pos[1] < ss.Bounds.Min.Y {
		return false
	}
	if pos[1] > ss.Bounds.Max.Y {
		return false
	}
	return true
}

func (ServerScene) OnDisconnect(op sos.DisconnectOp) {
	os.Exit(0)
}
func (ServerScene) OnFlagUpdate(op sos.FlagUpdateOp) {}
func (ServerScene) OnLogMessage(op sos.LogMessageOp) {
	log.Debugf("Log: %+v", op)
}
func (ServerScene) OnMetrics(op sos.MetricsOp) {}
func (ss *ServerScene) OnCriticalSection(op sos.CriticalSectionOp) {
	//log.Debugf("In Critical: %+v", op)
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
	if ok && impWorker.WorkerType == "LauncherClient" {
		ss.OnClientConnect(op.ID, impWorker.WorkerID)
	}
}

func (ss *ServerScene) OnRemoveComponent(op sos.RemoveComponentOp) {
	log.Debugf("OnRemoveComponent: %+v", op)
	if op.CID == 60 {
		ss.OnClientDisconnect(op.ID)
	}
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
	if op.CID == 1000 {
		log.Printf("Authority change for ship: %+v", op)
		e := ss.Entities[op.ID]
		s, ok := e.(*Ship)
		if ok {
			s.HasAuthority = op.Authority == 1
		}
	}
	if op.CID == 1001 {
		log.Printf("Authority change for bullet: %+v", op)
		e := ss.Entities[op.ID]
		b, ok := e.(*Bullet)
		if ok {
			b.HasAuthority = op.Authority == 1
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

func (ss *ServerScene) OnClientDisconnect(ID sos.EntityID) {
	ents, ok := ss.Clients[ID]
	if ok {
		for _, e := range ents {
			engo.Mailbox.Dispatch(DeleteEntityMessage{ID: e})
		}
		delete(ss.Clients, ID)
	}

}

// OnClientConnect is called once a worker has connected, and we should create them an entity with a player input component.
func (ss *ServerScene) OnClientConnect(ClientID sos.EntityID, WorkerID string) {
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
	spawnPoint := mgl32.Vec2{rand.Float32() * 2048, rand.Float32() * 1024}
	ent := Ship{
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Pos:  ImprobablePosition{Coords: Coordinates{float64(spawnPoint[0]), 0, float64(spawnPoint[1])}},
		Meta: ImprobableMetadata{Name: "Client"},
		Interest: ImprobableInterest{
			Interest: map[uint32]ComponentInterest{
				1003: ComponentInterest{
					Queries: []QBIQuery{
						{Constraint: QBIConstraint{RelativeSphereConstraint: &relSphere}, ResultComponents: []uint32{1000, 54, 53}},
					},
				},
			},
		},
		Mass:         1000.0,
		AttackDamage: 20,
		Ship: ShipComponent{
			Pos:           spawnPoint.Vec3(0),
			MaxEnergy:     100,
			CurrentEnergy: 100,
		},

		BasicEntity:        ecs.NewBasic(),
		CollisionComponent: common.CollisionComponent{Main: 1, Group: 1, Collides: 1 | 2},
		SpaceComponent: common.SpaceComponent{
			Position: engo.Point{spawnPoint[0], spawnPoint[1]},
			Width:    100,
			Height:   100,
		},
	}
	reqID := ss.spatial.CreateEntity(ent)
	ss.OnCreateFunc[reqID] = func(ID sos.EntityID) {
		ent.ID = ID
		ent.SetupQBI()
		ss.Entities[ID] = &ent
		ss.ECS[ent.BasicEntity.ID()] = &ent
		ss.Clients[ClientID] = []sos.EntityID{ID}
		ss.CollisionSystem.Add(&ent.BasicEntity, &ent.CollisionComponent, &ent.SpaceComponent)
	}
}
