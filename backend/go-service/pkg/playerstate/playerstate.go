package playerstate

import (
	"hash/fnv"
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func Velocity(player *common.Player) r3.Vector {
	if player == nil {
		return r3.Vector{}
	}
	pawn := player.PlayerPawnEntity()
	if pawn == nil {
		return r3.Vector{}
	}
	value, exists := pawn.PropertyValue("m_vecVelocity")
	if !exists {
		value, exists = pawn.PropertyValue("m_vecAbsVelocity")
	}
	if !exists {
		return r3.Vector{}
	}
	return value.R3Vec()
}

func EquipmentID(equipment *common.Equipment) int64 {
	if equipment == nil {
		return 0
	}
	id := equipment.UniqueID2()
	hash := fnv.New64a()
	_, _ = hash.Write(id[:])
	return int64(hash.Sum64() & math.MaxInt64)
}
