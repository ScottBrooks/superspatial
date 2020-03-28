package superspatial

import (
	"fmt"
	"os"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/ScottBrooks/sos"
)

var worldBounds = engo.AABB{Max: engo.Point{2048, 1024}}
var logger = logrus.New()
var log = logrus.NewEntry(logger)

func init() {
	//logger.SetLevel(logrus.DebugLevel)

	logger.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	logrus.SetOutput(colorable.NewColorableStdout())
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
				sps.SS.spatial.UpdateComponent(ent.ID, cidShip, ent.Ship)
				sps.SS.spatial.UpdateComponent(ent.ID, cidPosition, ent.Pos)
			}
		case *Bullet:
			ent.Update(dt)
			if ent.HasAuthority {
				sps.SS.spatial.UpdateComponent(ent.ID, cidBullet, ent.Bullet)
				sps.SS.spatial.UpdateComponent(ent.ID, cidPosition, ent.Pos)
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
	Development bool
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

	log = log.WithField("worker", ss.WorkerType())

	ss.spatial = sos.NewSpatialSystem(ss, ss.Host, ss.Port, ss.WorkerID, nil)
	ss.Entities = map[sos.EntityID]interface{}{}
	ss.ECS = map[uint64]interface{}{}
	ss.Clients = map[sos.EntityID][]sos.EntityID{}
	ss.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}

	ss.Bounds = worldBounds

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
			log.Printf("Collision: %+v %+v %+v", collision, collision.Entity.SpaceComponent, collision.To.SpaceComponent)
			ship, foundShip := ss.ECS[collision.To.ID()].(*Ship)
			target, foundTarget := ss.ECS[collision.Entity.ID()]

			if foundShip && foundTarget && ship != target {
				log.Printf("---------- %d HIT %d --------", collision.Entity.ID(), collision.To.ID())

				switch t := target.(type) {
				case *Bullet:
					if t.Bullet.ShipID != ship.ID {
						log.Printf("Ship: %+v, Target: %+v ", ship.SpaceComponent.Position, t.SpaceComponent.Position)

						w.RemoveEntity(t.BasicEntity)
						engo.Mailbox.Dispatch(DeleteEntityMessage{ID: t.ID})
						log.Printf("Bullet hit ship: %+v", ship)
						ss.NewEffect(t.Bullet.Pos, 1, 300)
						delete(ss.ECS, t.BasicEntity.ID())
					}
				case *Ship:
					log.Printf("Ship: %+v, Target: %+v ", ship.SpaceComponent.Position, t.SpaceComponent.Position)
					w.RemoveEntity(t.BasicEntity)
					engo.Mailbox.Dispatch(DeleteEntityMessage{ID: t.ID})

					log.Printf("Ship hit ship: %+v", ship)
					ss.NewEffect(t.Ship.Pos, 1, 1000)
					delete(ss.ECS, t.BasicEntity.ID())
				}

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
	/*
		impWorker, ok := op.Component.(*ImprobableWorker)
		if ok && impWorker.WorkerType == "LauncherClient" {
			ss.OnClientConnect(op.ID, impWorker.WorkerID)
		}
	*/
	if op.CID == cidShip {
		log.Printf("Making a new ship")
		ent := NewShip(mgl32.Vec2{}, "")
		ent.ID = op.ID
		ss.Entities[op.ID] = &ent
		ss.ECS[ent.BasicEntity.ID()] = &ent
		ss.CollisionSystem.Add(&ent.BasicEntity, &ent.CollisionComponent, &ent.SpaceComponent)
	}
	if op.CID == cidBullet {
		log.Printf("Making a new bullet")
		ent := Bullet{}
		ent.ID = op.ID
		ss.Entities[op.ID] = &ent
		ss.ECS[ent.BasicEntity.ID()] = &ent
		ss.CollisionSystem.Add(&ent.BasicEntity, &ent.CollisionComponent, &ent.SpaceComponent)
	}
	if op.CID == cidEffect {
		go func() {
			time.Sleep(time.Duration(op.Component.(*EffectComponent).Expiry) * time.Millisecond)

			log.Printf("Deleting the effect after expiry")
			engo.Mailbox.Dispatch(DeleteEntityMessage{ID: op.ID})

		}()
	}
}

func (ss *ServerScene) OnRemoveComponent(op sos.RemoveComponentOp) {
	log.Debugf("OnRemoveComponent: %+v", op)
	if op.CID == cidWorker {
		ss.OnClientDisconnect(op.ID)
	}

	if op.CID == cidShip {
		ent, ok := ss.Entities[op.ID].(*Ship)
		if !ok {
			log.Printf("nota ship: %+v", ent)
		} else {
			ss.CollisionSystem.Remove(ent.BasicEntity)

		}
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
	if op.CID == cidShip {
		log.Printf("Authority change for ship: %+v", op)
		e := ss.Entities[op.ID]
		s, ok := e.(*Ship)
		if ok {
			s.HasAuthority = op.Authority == 1
		}
	}
	if op.CID == cidBullet {
		log.Printf("Authority change for bullet: %+v", op)
		e := ss.Entities[op.ID]
		b, ok := e.(*Bullet)
		if ok {
			b.HasAuthority = op.Authority == 1
		}
	}
}

func (ss *ServerScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	shipEnt, ok := ss.Entities[op.ID].(*Ship)
	if ok {
		switch op.CID {
		case cidShip:
			shipEnt.Ship = *op.Component.(*ShipComponent)
		case cidPlayerInput:
			shipEnt.PIC = *op.Component.(*PlayerInputComponent)
		}
	}
	bulletEnt, ok := ss.Entities[op.ID].(*Bullet)
	if ok {
		switch op.CID {
		case cidBullet:
			bulletEnt.Bullet = *op.Component.(*BulletComponent)
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
	case cidACL:
		return &ImprobableACL{ComponentWriteAcl: map[uint32]WorkerRequirementSet{}}, nil
	case cidPosition:
		return &ImprobablePosition{}, nil
	case cidWorker:
		return &ImprobableWorker{}, nil
	case cidShip:
		return &ShipComponent{}, nil
	case cidBullet:
		return &BulletComponent{}, nil
	case cidGame:
		return &SpatialGameComponent{}, nil
	case cidPlayerInput:
		return &PlayerInputComponent{}, nil
	case cidWorkerBalancer:
		return &WorkerComponent{}, nil
	case cidEffect:
		return &EffectComponent{}, nil
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
