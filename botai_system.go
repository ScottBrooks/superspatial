package superspatial

import (
	"math/rand"
	"time"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

type BotAISystem struct {
	SS *ServerScene

	Ship *TrackedEntity

	NextTurnAt time.Time
	StopTurnAt time.Time
}

func (bas *BotAISystem) Add(ent *ecs.BasicEntity, sc *common.SpaceComponent, offset engo.Point) {
}
func (bas *BotAISystem) Remove(ecs.BasicEntity) {}
func (bas *BotAISystem) Update(dt float32) {
	if bas.Ship != nil {
		speed := bas.Ship.Ship.Vel.Len()
		if speed < 20 {
			bas.Ship.PlayerInput.Forward = true
		}

		now := time.Now()
		if bas.NextTurnAt.Sub(now) < 0 {
			bas.NextTurnAt = now.Add(time.Duration(rand.Intn(10)) * time.Second)
			bas.StopTurnAt = now.Add(2 * time.Second)

			coin := rand.Intn(2)
			if coin == 0 {
				log.Printf("Turning left)")
				bas.Ship.PlayerInput.Left = true
				bas.Ship.PlayerInput.Right = false
			} else {
				log.Printf("Turning right")
				bas.Ship.PlayerInput.Left = false
				bas.Ship.PlayerInput.Right = true
			}
			bas.Ship.PlayerInput.Forward = true
		}
		if bas.StopTurnAt.Sub(now) < 0 {
			bas.Ship.PlayerInput.Left = false
			bas.Ship.PlayerInput.Right = false
			bas.Ship.PlayerInput.Forward = false

		}

		// Disable AI updating
		//bas.SS.spatial.UpdateComponent(bas.Ship.ID, cidPlayerInput, bas.Ship.PlayerInput)
	}
}
