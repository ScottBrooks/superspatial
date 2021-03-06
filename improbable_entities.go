package superspatial

import (
	"github.com/ScottBrooks/sos"
	"github.com/go-gl/mathgl/mgl32"
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

type ImprobableMetadata struct {
	Name string
}

type ImprobableACL struct {
	ReadAcl           WorkerRequirementSet            `sos:"read_acl"`
	ComponentWriteAcl map[uint32]WorkerRequirementSet `sos:"component_write_acl"`
}

type SpatialEntity struct {
	ID sos.EntityID
}

func (SpatialEntity) Create()   {}
func (SpatialEntity) Complete() {}

type SpatialGameComponent struct {
	EntityID sos.EntityID
}

type ShipComponent struct {
	Pos    mgl32.Vec3
	Vel    mgl32.Vec3
	Angle  float32
	Radius float32
}

type PlayerInputComponent struct {
	Left    bool
	Right   bool
	Forward bool
	Back    bool
	Attack  bool
}

type EffectComponent struct {
	Id     int32
	Expiry int32
	Pos    mgl32.Vec3
}
