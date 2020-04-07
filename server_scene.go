package superspatial

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
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

	CircleCollisionSystem CircleCollisionSystem
}

func angleDist(a float32, b float32) float32 {
	phi := float32(math.Mod(math.Abs(float64(a-b)), 360))
	if phi > 180 {
		return 360 - phi
	}
	return phi
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

	w.AddSystem(&ss.phys)
	w.AddSystem(&SpatialPumpSystem{ss})
	w.AddSystem(&AttackSystem{SS: ss})
	w.AddSystem(&ss.CircleCollisionSystem)

	engo.Mailbox.Listen(DeleteEntityMessage{}.Type(), func(msg engo.Message) {
		dem, ok := msg.(DeleteEntityMessage)
		if !ok {
			return
		}
		ss.spatial.Delete(dem.ID)
	})

	engo.Mailbox.Listen(CircleCollisionMessage{}.Type(), func(msg engo.Message) {
		collision, ok := msg.(CircleCollisionMessage)
		if ok {
			log.Printf("Collision: %+v %+v %+v", collision, collision.A.SpaceComponent, collision.B.SpaceComponent)
			shipA, foundShipA := ss.ECS[collision.A.ID()].(*Ship)
			shipB, foundShipB := ss.ECS[collision.B.ID()].(*Ship)

			if foundShipA && foundShipB && shipA != shipB {
				delta := shipA.Ship.Pos.Sub(shipB.Ship.Pos).Len()
				// Too far away, not a real hit
				if delta > 64 {
					return
				}
				log.Printf("---------- %d HIT %d --------: %+v", collision.A.ID(), collision.B.ID(), delta)

				vAngleA := mgl32.RadToDeg(float32(math.Atan2(float64(shipA.Ship.Vel[1]), float64(shipA.Ship.Vel[0]))))
				vAngleB := mgl32.RadToDeg(float32(math.Atan2(float64(shipB.Ship.Vel[1]), float64(shipB.Ship.Vel[0]))))

				dA := angleDist(shipA.Ship.Angle, vAngleA)
				dB := angleDist(shipB.Ship.Angle, vAngleB)

				if dA > 30 {
					dA = 30
				}
				if dB > 30 {
					dB = 30
				}

				attackA := (30 - dA) * shipA.Ship.Vel.Len()
				attackB := (30 - dB) * shipB.Ship.Vel.Len()
				log.Printf("A: Angle: %f VAngle: %f Delta: %v AttackA: %f", shipA.Ship.Angle, vAngleA, dA, attackA)
				log.Printf("B: Angle: %f VAngle: %f Delta: %v AttackB: %f", shipB.Ship.Angle, vAngleB, dB, attackB)

				var deadShip *Ship
				if attackB < attackA { // A attacks B
					deadShip = shipB
				} else if attackA < attackB { // B attacks A
					deadShip = shipA

				}

				if deadShip != nil {

					log.Printf("Ship: %+v, Target: %+v ", shipA.SpaceComponent.Position, shipB.SpaceComponent.Position)
					w.RemoveEntity(deadShip.BasicEntity)
					engo.Mailbox.Dispatch(DeleteEntityMessage{ID: deadShip.ID})

					log.Printf("Ship hit ship: %+v", deadShip)
					ss.NewEffect(deadShip.Ship.Pos, 1, 1000)
					delete(ss.ECS, deadShip.BasicEntity.ID())
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
		ss.CircleCollisionSystem.Add(&ent.BasicEntity, &ent.SpaceComponent, ent.Ship.Radius)
	}
	if op.CID == cidBullet {
		log.Printf("Making a new bullet")
		ent := Bullet{}
		ent.ID = op.ID
		ss.Entities[op.ID] = &ent
		ss.ECS[ent.BasicEntity.ID()] = &ent
		ss.CircleCollisionSystem.Add(&ent.BasicEntity, &ent.SpaceComponent, 1)
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
			ss.CircleCollisionSystem.Remove(ent.BasicEntity)

		}
	}
}

func (ss *ServerScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	if op.CID == 58 && op.Authority == 1 {
		e := ss.Entities[op.ID]
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
