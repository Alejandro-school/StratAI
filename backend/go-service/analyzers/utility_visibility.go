package analyzers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// A geometric possibility, never evidence of attention or human perception.
type UtilityPointVisibility struct {
	Available bool
	Visible   bool
	EyeSource string
}

func EstimateUtilityPointVisibility(ctx *models.DemoContext, observer *common.Player, point r3.Vector, smokes map[int64]r3.Vector) UtilityPointVisibility {
	result := UtilityPointVisibility{}
	if ctx == nil || ctx.MapManager == nil || !ctx.MapManager.IsLoaded() || observer == nil || !finiteUtilityPoint(point) {
		return result
	}
	pawn := observer.PlayerPawnEntity()
	if pawn == nil {
		return result
	}
	if _, ok := pawn.PropertyValue("m_angEyeAngles"); !ok {
		return result
	}
	if _, ok := pawn.PropertyValue("m_bIsScoped"); !ok {
		return result
	}
	result.Available = true
	if !observer.IsAlive() || observer.IsBlinded() {
		return result
	}
	eyes, native := playerstate.EyePosition(observer)
	if !finiteUtilityPoint(eyes) {
		result.Available = false
		return result
	}
	result.EyeSource = "stance_eye_proxy"
	if native {
		result.EyeSource = "native_eye_position"
	}
	yaw, pitch := playerstate.ViewAngles(observer)
	if math.IsNaN(float64(yaw)) || math.IsInf(float64(yaw), 0) || math.IsNaN(float64(pitch)) || math.IsInf(float64(pitch), 0) {
		result.Available = false
		return result
	}
	state := visibilityPlayerState{pitch: pitch, yaw: yaw, isScoped: observer.IsScoped()}
	if weapon := observer.ActiveWeapon(); weapon != nil {
		state.weaponType, state.zoomLevel = weapon.Type, int(weapon.ZoomLevel())
	}
	if !playerViewFrustum(state).contains(eyes, point) || smokeBlocksSight(smokes, eyes, point) {
		return result
	}
	result.Visible = ctx.MapManager.IsVisible(eyes, point)
	return result
}

func finiteUtilityPoint(p r3.Vector) bool {
	return !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsNaN(p.Z) &&
		!math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0) && !math.IsInf(p.Z, 0)
}
