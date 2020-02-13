package superspatial

import (
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/ScottBrooks/sos"
	"github.com/go-gl/mathgl/mgl32"
)

type balancedWorker struct {
	ID sos.EntityID

	ACL      ImprobableACL      `sos:"50"`
	Pos      ImprobablePosition `sos:"54"`
	Meta     ImprobableMetadata `sos:"53"`
	Interest ImprobableInterest `sos:"58"`

	WorkerID       string
	WorkerEntityID sos.EntityID
	AABB           engo.AABB
}

type WorkerComponent struct {
	WorkerID int32
}

type balancedEntity struct {
	ID     sos.EntityID
	Pos    ImprobablePosition `sos:"54"`
	Worker WorkerComponent    `sos:"1005"`

	Client string
}

type BalancerScene struct {
	ServerScene

	RequestingWorker bool
	WorldBounds      engo.AABB
	Workers          []balancedWorker
	Entities         map[sos.EntityID]*balancedEntity
}

func (*BalancerScene) Preload() {}
func (bs *BalancerScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)
	log = log.WithField("worker", bs.ServerScene.WorkerType())
	bs.spatial = sos.NewSpatialSystem(bs, bs.ServerScene.Host, bs.ServerScene.Port, bs.ServerScene.WorkerID, nil)
	bs.Entities = map[sos.EntityID]*balancedEntity{}
	bs.ServerScene.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}

	log.Printf("New spatialsystem")

	w.AddSystem(&SpatialPumpSystem{&bs.ServerScene})
}
func (*BalancerScene) Type() string { return "Balancer" }

func (bs *BalancerScene) OnAddComponent(op sos.AddComponentOp) {
	log.Debugf("OnAddCOmponent: %+v %+v", op, op.Component)
	impWorker, ok := op.Component.(*ImprobableWorker)
	if ok && impWorker.WorkerType == "LauncherClient" {
		if bs.needsMoreWorkers() {
			bs.startWorker()
			time.Sleep(2 * time.Second)
			bs.startWorker()
		}
		bs.OnClientConnect(op.ID, impWorker.WorkerID)
	}
	if ok && impWorker.WorkerType == "Server" {
		log.Printf("OMG WE STARTED A SERVER")
		bs.RequestingWorker = false

		ent := NewServerWorker()
		reqID := bs.spatial.CreateEntity(ent)
		bs.OnCreateFunc[reqID] = func(ID sos.EntityID) {
			ent.ID = ID
			log.Printf("Creaet complete")
			bs.Workers = append(bs.Workers, balancedWorker{WorkerID: impWorker.WorkerID, WorkerEntityID: op.ID, ID: ID})
		}
	}
}
func (bs *BalancerScene) OnCreateEntity(op sos.CreateEntityOp) {
	log.Printf("Go create ent op: %+v", op)
	bs.ServerScene.OnCreateEntity(op)

	log.Printf("Workers: %+v", bs.Workers)
	for _, w := range bs.Workers {
		if w.ID == op.ID {
			bs.rebalanceAuthority()
		}
	}

}
func (bs *BalancerScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	switch op.CID {
	case cidPosition:
		pos, ok := op.Component.(*ImprobablePosition)
		if ok {
			ent, ok := bs.Entities[op.ID]
			if ok {
				ent.Pos = *pos
			}
			bs.checkEntityBounds()
		}
	}
}

func (bs *BalancerScene) WorkerType() string { return bs.WorkerTypeName }

func aabbContains(aabb engo.AABB, pt Coordinates) bool {
	return aabb.Min.X <= float32(pt.X) && aabb.Max.X > float32(pt.X) && aabb.Min.Y <= float32(pt.Z) && aabb.Max.Y > float32(pt.Z)
}

func (bs *BalancerScene) checkEntityBounds() {
	for id, e := range bs.Entities {

		needsAdjustment := true
		if e.Worker.WorkerID >= 0 && int(e.Worker.WorkerID) < len(bs.Workers) {
			worker := bs.Workers[e.Worker.WorkerID]

			if aabbContains(worker.AABB, e.Pos.Coords) {
				needsAdjustment = false
				//log.Printf("Entity %d %+v is inside worker %+v", id, e.Pos.Coords, worker)
			}
		}
		if needsAdjustment {
			for i, w := range bs.Workers {
				if aabbContains(w.AABB, e.Pos.Coords) {
					log.Printf("Moving Entity %d %d from Worker: %d to %d", id, e.ID, e.Worker.WorkerID, i)
					e.Worker.WorkerID = int32(i)
					bs.spatial.UpdateComponent(e.ID, cidWorkerBalancer, e.Worker)

					workerID := "workerId:" + w.WorkerID
					clientWorkerID := "workerId:" + e.Client

					readAttrSet := []WorkerAttributeSet{
						{[]string{"position"}},
						{[]string{"client"}},
						{[]string{"balancer"}},
					}
					readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
					writeAcl := map[uint32]WorkerRequirementSet{
						cidShip:        WorkerRequirementSet{[]WorkerAttributeSet{{[]string{workerID}}}},
						cidPosition:    WorkerRequirementSet{[]WorkerAttributeSet{{[]string{workerID}}}},
						cidPlayerInput: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{clientWorkerID}}}},
						//cidBullet: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{workerID}}}},
						// Leave interest write authority for now so things keep working.  This should move to the balancer though
						cidACL:            WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
						cidInterest:       WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
						cidWorkerBalancer: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
					}

					acl := ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl}
					log.Printf("Updating ACL: %+v", acl)

					bs.spatial.UpdateComponent(e.ID, cidACL, acl)
				} else {
					log.Printf("Worker %+v does not contain %v", w, e.Pos.Coords)
				}
			}
		}

	}
}

func (bs *BalancerScene) needsMoreWorkers() bool {
	if !bs.RequestingWorker && len(bs.Workers) == 0 {
		bs.RequestingWorker = true

		return true
	}

	return false
}

func (bs *BalancerScene) OnClientConnect(ClientID sos.EntityID, WorkerID string) {
	// Create entity,
	log.Printf("Creating client entity: %s", WorkerID)
	spawnPoint := mgl32.Vec2{rand.Float32() * bs.WorldBounds.Max.X, rand.Float32() * bs.WorldBounds.Max.Y}
	ent := NewShip(spawnPoint, WorkerID)

	reqID := bs.spatial.CreateEntity(ent)
	bs.OnCreateFunc[reqID] = func(ID sos.EntityID) {
		ent.ID = ID
		ent.SetupQBI()

		bs.Entities[ID] = &balancedEntity{ID: ID, Worker: WorkerComponent{-1}, Client: WorkerID}
	}

}

func (bs *BalancerScene) startWorker() {
	bs.RequestingWorker = true
	var cmd *exec.Cmd
	if bs.ServerScene.Development {
		cmd = exec.Command("go", "run", "cmd/server/main.go")
	} else {
		cmd = exec.Command("./server", "-host", bs.ServerScene.Host, "-port", strconv.Itoa(bs.ServerScene.Port))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Printf("Error starting worker: %+v", err)
	}
}

// Simple split into vertical slices
func (bs *BalancerScene) rebalanceAuthority() {
	log.Printf("Rebalance auth")
	xSize := int(bs.WorldBounds.Max.X-bs.WorldBounds.Min.X) / len(bs.Workers)
	xOffset := 0
	for i, w := range bs.Workers {
		bounds := engo.AABB{Min: engo.Point{float32(xOffset), bs.WorldBounds.Min.Y}, Max: engo.Point{float32(xOffset + xSize), bs.WorldBounds.Max.Y}}
		bs.setWorkerACL(w.ID, w.WorkerID, bounds)

		bs.Workers[i].AABB = bounds
		xOffset += xSize
	}

}

func (bs *BalancerScene) setWorkerACL(ID sos.EntityID, workerID string, bounds engo.AABB) {
	log.Printf("Setting WorkerACL for worker: %d %s %+v", ID, workerID, bounds)
	ShipCID := uint32(cidShip)
	BulletCID := uint32(cidBullet)
	PlayerInputCID := uint32(cidPlayerInput)

	workerID = "workerId:" + workerID

	readAttrSet := []WorkerAttributeSet{
		{[]string{"position"}},
		{[]string{"client"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		// Leave interest write authority for now so things keep working.  This should move to the balancer though
		cidInterest: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidPosition: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
	}

	boxConstraint := QBIBoxConstraint{
		Center: Coordinates{X: float64(bounds.Min.X) + float64(bounds.Max.X-bounds.Min.X)/2, Y: 0, Z: float64(bounds.Min.Y) + float64(bounds.Max.Y-bounds.Min.Y)/2},
		Edge:   EdgeLength{X: float64(bounds.Max.X - bounds.Min.X), Y: 10000, Z: float64(bounds.Max.Y - bounds.Min.Y)},
	}

	constraint := QBIConstraint{
		AndConstraint: []QBIConstraint{
			QBIConstraint{BoxConstraint: &boxConstraint},
			QBIConstraint{
				OrConstraint: []QBIConstraint{
					QBIConstraint{ComponentIDConstraint: &ShipCID},
					QBIConstraint{ComponentIDConstraint: &BulletCID},
					QBIConstraint{ComponentIDConstraint: &PlayerInputCID},
				},
			},
		},
	}

	interest := ImprobableInterest{
		Interest: map[uint32]ComponentInterest{
			cidShip: ComponentInterest{
				Queries: []QBIQuery{
					{Constraint: constraint, ResultComponents: []uint32{cidShip, cidPosition, cidPlayerInput}},
				},
			},
			cidBullet: ComponentInterest{
				Queries: []QBIQuery{
					{Constraint: constraint, ResultComponents: []uint32{cidBullet, cidPosition}},
				},
			},
		},
	}
	acl := ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl}

	bs.spatial.UpdateComponent(ID, cidACL, acl)
	bs.spatial.UpdateComponent(ID, cidInterest, interest)
	pos := ImprobablePosition{Coords: boxConstraint.Center}
	bs.spatial.UpdateComponent(ID, cidPosition, pos)

}

func NewServerWorker() balancedWorker {

	readAttrSet := []WorkerAttributeSet{
		{[]string{"client"}},
		{[]string{"balancer"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		cidACL:      WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidInterest: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidPosition: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		/*
			cidBullet: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
			// Leave interest write authority for now so things keep working.  This should move to the balancer though
			cidPosition: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		*/
	}
	worker := balancedWorker{
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Meta: ImprobableMetadata{Name: "Server"},
		Pos:  ImprobablePosition{},
	}
	return worker
}
