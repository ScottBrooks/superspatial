package superspatial

import (
	"fmt"
	"github.com/ScottBrooks/sos"
)

type SpatialAdapter struct {
}

func (SpatialAdapter) OnDisconnect(sos.DisconnectOp)             {}
func (SpatialAdapter) OnFlagUpdate(sos.FlagUpdateOp)             {}
func (SpatialAdapter) OnLogMessage(sos.LogMessageOp)             {}
func (SpatialAdapter) OnMetrics(sos.MetricsOp)                   {}
func (SpatialAdapter) OnCriticalSection(sos.CriticalSectionOp)   {}
func (SpatialAdapter) OnAddEntity(sos.AddEntityOp)               {}
func (SpatialAdapter) OnRemoveEntity(sos.RemoveEntityOp)         {}
func (SpatialAdapter) OnReserveEntityId(sos.ReserveEntityIdOp)   {}
func (SpatialAdapter) OnReserveEntityIds(sos.ReserveEntityIdsOp) {}
func (SpatialAdapter) OnCreateEntity(sos.CreateEntityOp)         {}
func (SpatialAdapter) OnDeleteEntity(sos.DeleteEntityOp)         {}
func (SpatialAdapter) OnEntityQuery(sos.EntityQueryOp)           {}
func (SpatialAdapter) OnAddComponent(sos.AddComponentOp)         {}
func (SpatialAdapter) OnRemoveComponent(sos.RemoveComponentOp)   {}
func (SpatialAdapter) OnAuthorityChange(sos.AuthorityChangeOp)   {}
func (SpatialAdapter) OnComponentUpdate(sos.ComponentUpdateOp)   {}
func (SpatialAdapter) OnCommandRequest(sos.CommandRequestOp)     {}
func (SpatialAdapter) OnCommandResponse(sos.CommandResponseOp)   {}
func (SpatialAdapter) AllocComponent(ID sos.EntityID, CID sos.ComponentID) (interface{}, error) {
	return nil, fmt.Errorf("Unimplemented")
}
func (*SpatialAdapter) WorkerType() string { return "Client" }
