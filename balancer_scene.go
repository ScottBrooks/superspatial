package superspatial

import (
	"math"
	"math/bits"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"

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
	Killing        bool
	Process        *os.Process
}

type WorkerComponent struct {
	WorkerID int32
}

type balancedEntity struct {
	ID     sos.EntityID
	ACL    ImprobableACL      `sos:"50"`
	Pos    ImprobablePosition `sos:"54"`
	Worker WorkerComponent    `sos:"1005"`

	Client string
}

type BalancerScene struct {
	ServerScene

	WorkersAdjusting  bool
	TargetWorkerCount int
	WorldBounds       engo.AABB
	Workers           []balancedWorker
	Entities          map[sos.EntityID]*balancedEntity

	BotProcesses    []*os.Process
	WorkerProcesses []*os.Process
	Clients         map[sos.EntityID]string
}

func (*BalancerScene) Preload() {}
func (bs *BalancerScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)
	log = log.WithField("worker", bs.ServerScene.WorkerType())
	sos.SilenceLogs()

	bs.spatial = sos.NewSpatialSystem(bs, bs.ServerScene.Host, bs.ServerScene.Port, bs.ServerScene.WorkerID, nil)
	bs.Entities = map[sos.EntityID]*balancedEntity{}
	bs.Clients = map[sos.EntityID]string{}
	bs.ServerScene.OnCreateFunc = map[sos.RequestID]func(ID sos.EntityID){}

	log.Printf("New spatialsystem")

	w.AddSystem(&SpatialPumpSystem{&bs.ServerScene})
}
func (*BalancerScene) Type() string { return "Balancer" }

func (bs *BalancerScene) OnAuthorityChange(op sos.AuthorityChangeOp) {
	// Gained authority over the ACL: adjust acl for the correct worker
	if op.Authority == 1 && op.CID == cidACL {
		bs.checkEntityBounds()
	}
}

func (bs *BalancerScene) OnAddEntity(op sos.AddEntityOp) {
	if bs.Entities[op.ID] == nil {
		bs.Entities[op.ID] = &balancedEntity{ID: op.ID, Worker: WorkerComponent{-1}}
	} else {
		log.Printf("Already had entity: %d", op.ID)
	}
}

func (bs *BalancerScene) OnAddComponent(op sos.AddComponentOp) {
	impWorker, ok := op.Component.(*ImprobableWorker)
	if ok && (impWorker.WorkerType == "LauncherClient" || impWorker.WorkerType == "Bot") {
		bs.Clients[op.ID] = impWorker.WorkerType

		bs.updateWorkerProcesses()
		bs.CreateClientShip(impWorker.WorkerID)
	}
	if ok && impWorker.WorkerType == "Server" {
		bs.WorkersAdjusting = false

		ent := NewServerWorker()
		reqID := bs.spatial.CreateEntity(ent)
		bs.OnCreateFunc[reqID] = func(ID sos.EntityID) {
			ent.ID = ID
			log.Printf("Create complete")

			pid, err := strconv.Atoi(strings.TrimPrefix(impWorker.WorkerID, "Server_"))
			if err != nil {
				log.Printf("Expected to be able to turn worker id into a pid", err)
			}

			proc, err := os.FindProcess(pid)
			if err != nil {
				log.Printf("Expected to be able to find process: %v", err)
			}
			bs.Workers = append(bs.Workers, balancedWorker{WorkerID: impWorker.WorkerID, WorkerEntityID: op.ID, ID: ID, Process: proc})
			bs.updateWorkerProcesses()

			bs.checkEntityBounds()
		}
	}
	if op.CID == cidACL {
		bs.Entities[op.ID].ACL = *op.Component.(*ImprobableACL)
	}
	if op.CID == cidBullet {
		log.Printf("It's a bullet")
	}
}

func (bs *BalancerScene) OnRemoveComponent(op sos.RemoveComponentOp) {

	_, ok := bs.Clients[op.ID]
	if ok {
		delete(bs.Clients, op.ID)
	}

	if op.CID == cidWorker {
		toDelete := -1
		for i, w := range bs.Workers {
			if w.WorkerEntityID == op.ID {
				toDelete = i
			}
		}
		if toDelete != -1 {
			bs.spatial.Delete(bs.Workers[toDelete].ID)
			bs.Workers = append(bs.Workers[:toDelete], bs.Workers[toDelete+1:]...)
		}
		bs.updateWorkerProcesses()
	}

}

func (bs *BalancerScene) OnRemoveEntity(op sos.RemoveEntityOp) {
	if e := bs.Entities[op.ID]; e != nil {
		if e.Client != "" {
			bs.CreateClientShip(e.Client)
		}
		log.Printf("Removing entity: %d %+v", op.ID, e)
		delete(bs.Entities, op.ID)
	}

}

func (bs *BalancerScene) OnDeleteEntity(op sos.DeleteEntityOp) {
	if e := bs.Entities[op.ID]; e != nil {
		log.Printf("Deleting entity: %d %+v", op.ID, e)
		delete(bs.Entities, op.ID)
	}
}
func (bs *BalancerScene) OnCreateEntity(op sos.CreateEntityOp) {
	log.Printf("Go create ent op: %+v", op)
	bs.ServerScene.OnCreateEntity(op)

	log.Printf("Workers: %+v", bs.Workers)
	for _, w := range bs.Workers {
		if w.ID == op.ID {
			if bs.TargetWorkerCount == len(bs.Workers) {
				bs.rebalanceAuthority()
			}
		}
	}

}
func (bs *BalancerScene) OnComponentUpdate(op sos.ComponentUpdateOp) {
	switch op.CID {
	case cidPosition:
		pos, ok := op.Component.(*ImprobablePosition)
		if ok {
			ent, ok := bs.Entities[op.ID]
			//log.Printf("Component update for: %d", op.ID)
			if ok {
				ent.Pos = *pos
			}
			bs.checkEntityBounds()
		}
	case cidACL:
		acl, ok := op.Component.(*ImprobableACL)
		if ok {
			ent, ok := bs.Entities[op.ID]
			if ok {
				log.Warnf("Updating ACL on entity: %d, %+v", op.ID, *acl)
				ent.ACL = *acl
			}
		}
	}
}

func (bs *BalancerScene) WorkerType() string { return bs.WorkerTypeName }

func aabbContains(aabb engo.AABB, pt Coordinates) bool {
	return aabb.Min.X <= float32(pt.X) && aabb.Max.X > float32(pt.X) && aabb.Min.Y <= float32(pt.Z) && aabb.Max.Y > float32(pt.Z)
}
func (bs *BalancerScene) adjustAcl(i int, e *balancedEntity, w balancedWorker) {
	log.Printf("Moving Entity %d from Worker: %d to %d", e.ID, e.Worker.WorkerID, i)
	e.Worker.WorkerID = int32(i)
	bs.spatial.UpdateComponent(e.ID, cidWorkerBalancer, e.Worker)

	workerID := "workerId:" + w.WorkerID

	// Update our ACL entries that varry per worker.
	for _, cid := range []uint32{cidShip, cidBullet, cidPosition, cidEffect} {
		if _, ok := e.ACL.ComponentWriteAcl[cid]; ok {
			e.ACL.ComponentWriteAcl[cid] = WorkerRequirementSet{[]WorkerAttributeSet{{[]string{workerID}}}}
		}
	}

	log.Printf("Updating ACL: %+v", e.ACL)

	bs.spatial.UpdateComponent(e.ID, cidACL, e.ACL)
}

func (bs *BalancerScene) checkEntityBounds() {
	for _, e := range bs.Entities {
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
					bs.adjustAcl(i, e, w)
				} else {
					//log.Printf("Worker %+v does not contain %v", w, e.Pos.Coords)
				}
			}
		}
	}
}

func (bs *BalancerScene) OnFlagUpdate(op sos.FlagUpdateOp) {
	log.Printf("Flag Update: %+v", op)
	if op.Key == "NUM_BOTS" {
		target, err := strconv.Atoi(op.Value)
		if err != nil {
			log.Printf("Error converting %s to int: %v", op.Value, err)
			return
		}

		delta := target - len(bs.BotProcesses)
		log.Printf("Bots delta: %d", delta)
		if delta > 0 {
			for i := 0; i < delta; i++ {
				bs.startBot()
			}
		}

		if delta < 0 {
			for i := delta; i < 0; i++ {
				// Kill off bots
				bs.stopBot()
			}
		}
	}
}

func (bs *BalancerScene) updateWorkerProcesses() {
	var numWorkers int
	for _, w := range bs.Workers {
		if !w.Killing {
			numWorkers++
		}
	}
	numClients := len(bs.Clients)

	reqWorkers := int(math.Pow(float64(bits.Len(uint(numClients))/2), 2))
	if reqWorkers <= 1 {
		reqWorkers++
	}

	if reqWorkers > numWorkers && !bs.WorkersAdjusting {
		log.Printf("We are requesting more workers: %d %d", reqWorkers, numWorkers)
		bs.TargetWorkerCount = reqWorkers
		bs.startWorker()
	}
	if reqWorkers < numWorkers && !bs.WorkersAdjusting {
		log.Printf("We are killing a worker: %d")
		bs.stopWorker()
	}
}

func (bs *BalancerScene) CreateClientShip(WorkerID string) {
	// Create entity,
	log.Printf("Creating client entity: %s", WorkerID)
	spawnPoint := mgl32.Vec2{rand.Float32() * bs.WorldBounds.Max.X, rand.Float32() * bs.WorldBounds.Max.Y}
	ent := NewShip(spawnPoint, WorkerID)

	reqID := bs.spatial.CreateEntity(ent)
	bs.OnCreateFunc[reqID] = func(ID sos.EntityID) {
		ent.ID = ID

		bs.Entities[ID] = &balancedEntity{ID: ID, Worker: WorkerComponent{-1}, Client: WorkerID, ACL: ent.ACL}
		log.Printf("Entity: %+v", bs.Entities[ID])
	}

}

func (bs *BalancerScene) startWorker() {
	var cmd *exec.Cmd
	cmd = exec.Command("./server", "-host", bs.ServerScene.Host, "-port", strconv.Itoa(bs.ServerScene.Port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Printf("Error starting worker: %+v", err)
	}
	bs.WorkersAdjusting = true
}

func (bs *BalancerScene) stopWorker() {
	if len(bs.Workers) == 0 {
		log.Printf("No workers to stop")
		return
	}

	bs.Workers[0].Killing = true
	proc := bs.Workers[0].Process
	if proc == nil {
		log.Printf("Proc is nil for worker: %v", bs.Workers[0])
	}
	log.Printf("Killing worker: %+v", proc)
	proc.Kill()
	p, err := proc.Wait()
	log.Printf("P: %+v Err: %+v", p, err)
}

func (bs *BalancerScene) startBot() {
	var cmd *exec.Cmd
	cmd = exec.Command("./bot", "-host", bs.ServerScene.Host, "-port", strconv.Itoa(bs.ServerScene.Port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Printf("Error starting ot: %+v", err)
	}

	bs.BotProcesses = append(bs.BotProcesses, cmd.Process)
}

func (bs *BalancerScene) stopBot() {
	if len(bs.BotProcesses) == 0 {
		log.Printf("No bots to stop")
		return
	}

	proc := bs.BotProcesses[0]
	log.Printf("Killing bot: %+v", proc)
	proc.Kill()
	p, err := proc.Wait()
	log.Printf("P: %+v Err: %+v", p, err)
	bs.BotProcesses = bs.BotProcesses[1:]
}

// Simple split into vertical slices
func (bs *BalancerScene) rebalanceAuthority() {
	cellCount := int(math.Sqrt(float64(len(bs.Workers))))
	xSize := int(bs.WorldBounds.Max.X-bs.WorldBounds.Min.X) / cellCount
	ySize := int(bs.WorldBounds.Max.Y-bs.WorldBounds.Min.Y) / cellCount
	log.Printf("Rebalance auth: Cell Count: %d XS: %d YS: %d", cellCount, xSize, ySize)
	for y := 0; y < cellCount; y++ {
		for x := 0; x < cellCount; x++ {
			i := y*cellCount + x
			w := bs.Workers[i]
			bounds := engo.AABB{
				Min: engo.Point{float32(x * xSize), float32(y * ySize)},
				Max: engo.Point{float32(x*xSize + xSize), float32(y*ySize + ySize)},
			}
			bs.setWorkerACL(w.ID, w.WorkerID, bounds)
			log.Printf("Bounds[%d, %d]: %d %+v", x, y, i, bounds)

			bs.Workers[i].AABB = bounds
		}
	}
}

func (bs *BalancerScene) setWorkerACL(ID sos.EntityID, workerID string, bounds engo.AABB) {
	log.Printf("Setting WorkerACL for worker: %d %s %+v", ID, workerID, bounds)
	ShipCID := uint32(cidShip)
	BulletCID := uint32(cidBullet)
	PlayerInputCID := uint32(cidPlayerInput)
	EffectCID := uint32(cidEffect)

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
		Edge:   EdgeLength{X: float64(bounds.Max.X-bounds.Min.X) * 1.1, Y: 10000, Z: float64(bounds.Max.Y-bounds.Min.Y) * 1.1},
	}

	constraint := QBIConstraint{
		AndConstraint: []QBIConstraint{
			QBIConstraint{BoxConstraint: &boxConstraint},
			QBIConstraint{
				OrConstraint: []QBIConstraint{
					QBIConstraint{ComponentIDConstraint: &ShipCID},
					QBIConstraint{ComponentIDConstraint: &BulletCID},
					QBIConstraint{ComponentIDConstraint: &EffectCID},
					QBIConstraint{ComponentIDConstraint: &PlayerInputCID},
				},
			},
		},
	}

	interest := ImprobableInterest{
		Interest: map[uint32]ComponentInterest{

			cidPosition: ComponentInterest{
				Queries: []QBIQuery{
					{Constraint: constraint, ResultComponents: []uint32{cidShip, cidPosition, cidEffect, cidBullet, cidPlayerInput}},
				},
			},
			/*
				cidShip: ComponentInterest{
					Queries: []QBIQuery{
						{Constraint: constraint, ResultComponents: []uint32{cidShip, cidPosition, cidEffect, cidPlayerInput}},
					},
				},
				cidBullet: ComponentInterest{
					Queries: []QBIQuery{
						{Constraint: constraint, ResultComponents: []uint32{cidBullet, cidPosition}},
					},
				},
			*/
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
		{[]string{"balancer"}},
	}
	readAcl := WorkerRequirementSet{AttributeSet: readAttrSet}
	writeAcl := map[uint32]WorkerRequirementSet{
		cidACL:      WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidInterest: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
		cidPosition: WorkerRequirementSet{[]WorkerAttributeSet{{[]string{"balancer"}}}},
	}
	worker := balancedWorker{
		ACL:  ImprobableACL{ComponentWriteAcl: writeAcl, ReadAcl: readAcl},
		Meta: ImprobableMetadata{Name: "Server"},
		Pos:  ImprobablePosition{},
	}
	return worker
}
