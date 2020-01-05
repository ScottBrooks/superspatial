package superspatial

import ()

type EdgeLength struct {
	X float64
	Y float64
	Z float64
}

type QBIConstraint struct {
	SphereConstraint           *QBISphereConstraint
	CylinderConstraint         *QBICylinderConstraint
	BoxConstraint              *QBIBoxConstraint
	RelativeSphereConstraint   *QBIRelativeSphereConstraint
	RelativeCylinderConstraint *QBIRelativeCylinderConstraint
	RelativeBoxConstraint      *QBIRelativeBoxConstraint
	EntityIDConstraint         *int64
	ComponentIDConstraint      *uint32
	AndConstraint              []QBIConstraint
	OrConstraint               []QBIConstraint
}

type QBISphereConstraint struct {
	Center Coordinates
	Radius float64
}

type QBICylinderConstraint struct {
	Center Coordinates
	Radius float64
}

type QBIBoxConstraint struct {
	Center Coordinates
	Edge   EdgeLength
}

type QBIRelativeSphereConstraint struct {
	Radius float64
}

type QBIRelativeCylinderConstraint struct {
	Radius float64
}

type QBIRelativeBoxConstraint struct {
	Edge EdgeLength
}

type QBIQuery struct {
	Constraint       QBIConstraint
	FullSnapshot     *bool
	ResultComponents []uint32
	Frequency        *float32
}

type ComponentInterest struct {
	Queries []QBIQuery
}

type ImprobableInterest struct {
	Interest map[uint32]ComponentInterest
}
