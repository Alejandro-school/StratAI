package playerstate

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func NativeVelocity(player *common.Player) (r3.Vector, bool) {
	if player == nil {
		return r3.Vector{}, false
	}
	pawn := player.PlayerPawnEntity()
	if pawn == nil {
		return r3.Vector{}, false
	}
	value, exists := pawn.PropertyValue("m_vecVelocity")
	if !exists {
		value, exists = pawn.PropertyValue("m_vecAbsVelocity")
	}
	if !exists {
		return r3.Vector{}, false
	}
	return propertyVector(value.Any)
}

func propertyVector(value any) (r3.Vector, bool) {
	switch vector := value.(type) {
	case [3]float32:
		return r3.Vector{X: float64(vector[0]), Y: float64(vector[1]), Z: float64(vector[2])}, true
	case []float32:
		if len(vector) < 3 {
			return r3.Vector{}, false
		}
		return r3.Vector{X: float64(vector[0]), Y: float64(vector[1]), Z: float64(vector[2])}, true
	default:
		return r3.Vector{}, false
	}
}

// ViewAngles returns the Source view angles with unambiguous names.
// demoinfocs exposes yaw as ViewDirectionX and pitch as ViewDirectionY.
func ViewAngles(player *common.Player) (yaw, pitch float32) {
	if player == nil {
		return 0, 0
	}
	return player.ViewDirectionX(), player.ViewDirectionY()
}

// EyePosition returns the native CS2 eye position when available. Older demos
// may not contain the pawn eye-offset fields, so a stance-aware fallback is
// used instead of returning the pawn's feet position.
func EyePosition(player *common.Player) (r3.Vector, bool) {
	if player == nil {
		return r3.Vector{}, false
	}
	if position, ok := player.PositionEyes(); ok {
		return position, true
	}

	position := player.Position()
	eyeHeight := 64.0
	if player.IsDucking() {
		eyeHeight = 46.0
	}
	position.Z += eyeHeight
	return position, false
}

func EquipmentID(equipment *common.Equipment) int64 {
	if equipment == nil || equipment.Entity == nil {
		return 0
	}
	return EquipmentEntityID(equipment.Entity.ID(), equipment.Entity.SerialNum())
}

func EquipmentEntityID(entityID, serialNumber int) int64 {
	if entityID <= 0 || serialNumber < 0 {
		return 0
	}
	combined := uint64(uint32(serialNumber))<<32 | uint64(uint32(entityID))
	return int64(combined & uint64(^uint64(0)>>1))
}
