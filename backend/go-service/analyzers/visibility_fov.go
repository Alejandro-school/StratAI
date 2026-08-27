package analyzers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

const (
	visibilitySampleStride    = 4
	canonicalAspectRatio      = 16.0 / 9.0
	sourceReferenceAspect     = 4.0 / 3.0
	unscopedReferenceFOV      = 90.0
	defaultScopedReferenceFOV = 40.0
)

type viewFrustum struct {
	forward           r3.Vector
	right             r3.Vector
	up                r3.Vector
	tanHalfHorizontal float64
	tanHalfVertical   float64
	circular          bool
}

type visibilityPlayerState struct {
	steamID    uint64
	team       common.Team
	alive      bool
	blinded    bool
	eyes       r3.Vector
	chest      r3.Vector
	head       r3.Vector
	pitch      float32
	yaw        float32
	speed      float64
	speedKnown bool
	weaponType common.EquipmentType
	isScoped   bool
	zoomLevel  int
}

func newViewFrustum(pitch, yaw float32, referenceHorizontalFOV float64) viewFrustum {
	verticalFOV := 2 * math.Atan(math.Tan(degreesToRadians(referenceHorizontalFOV)/2)/sourceReferenceAspect)
	horizontalFOV := 2 * math.Atan(math.Tan(verticalFOV/2)*canonicalAspectRatio)
	forward := anglesToR3Vector(pitch, yaw)
	yawRadians := degreesToRadians(float64(yaw))
	right := r3.Vector{X: -math.Sin(yawRadians), Y: math.Cos(yawRadians)}

	return viewFrustum{
		forward:           forward,
		right:             right,
		up:                forward.Cross(right),
		tanHalfHorizontal: math.Tan(horizontalFOV / 2),
		tanHalfVertical:   math.Tan(verticalFOV / 2),
	}
}

func (frustum viewFrustum) contains(origin, target r3.Vector) bool {
	toTarget := target.Sub(origin)
	depth := frustum.forward.Dot(toTarget)
	if depth <= 0 {
		return false
	}

	horizontal := math.Abs(frustum.right.Dot(toTarget) / depth)
	vertical := math.Abs(frustum.up.Dot(toTarget) / depth)
	if horizontal > frustum.tanHalfHorizontal || vertical > frustum.tanHalfVertical {
		return false
	}
	if !frustum.circular {
		return true
	}
	horizontal /= frustum.tanHalfHorizontal
	vertical /= frustum.tanHalfVertical
	return horizontal*horizontal+vertical*vertical <= 1
}

func visibilityStateFromPlayer(ctx *models.DemoContext, player *common.Player, tick int) visibilityPlayerState {
	eyes, _ := playerstate.EyePosition(player)
	chest := player.Position()
	chest.Z += 40
	motion := ctx.PlayerMotion.ObservePlayer(
		player,
		ctx.ActualRoundNumber,
		tick,
		ctx.Parser.TickRate(),
	)
	state := visibilityPlayerState{
		steamID:    player.SteamID64,
		team:       player.Team,
		alive:      player.IsAlive(),
		blinded:    player.IsBlinded(),
		eyes:       eyes,
		chest:      chest,
		head:       eyes,
		pitch:      player.ViewDirectionY(),
		yaw:        player.ViewDirectionX(),
		speed:      motion.HorizontalSpeed(),
		speedKnown: motion.Available,
		isScoped:   player.IsScoped(),
	}
	if weapon := player.ActiveWeapon(); weapon != nil {
		state.weaponType = weapon.Type
		state.zoomLevel = int(weapon.ZoomLevel())
	}
	return state
}

func playerViewFrustum(state visibilityPlayerState) viewFrustum {
	fov := unscopedReferenceFOV
	if state.isScoped {
		fov = scopedReferenceFOV(state.weaponType, state.zoomLevel)
	}
	frustum := newViewFrustum(state.pitch, state.yaw, fov)
	frustum.circular = state.isScoped && isSniperWeapon(state.weaponType)
	return frustum
}

func scopedReferenceFOV(weapon common.EquipmentType, zoomLevel int) float64 {
	if zoomLevel < 1 {
		zoomLevel = 1
	}
	switch weapon {
	case common.EqAWP:
		if zoomLevel >= 2 {
			return 10
		}
		return 40
	case common.EqSSG08, common.EqG3SG1, common.EqScar20:
		if zoomLevel >= 2 {
			return 15
		}
		return 40
	case common.EqAUG, common.EqSG553:
		return 45
	default:
		return defaultScopedReferenceFOV
	}
}

func isInsideVisibilityFrustum(shooter visibilityPlayerState, targets ...r3.Vector) bool {
	frustum := playerViewFrustum(shooter)
	for _, target := range targets {
		if frustum.contains(shooter.eyes, target) {
			return true
		}
	}
	return false
}

func firstSeenDataFromStates(shooter, enemy visibilityPlayerState, tick int) models.FirstSeenData {
	target := enemy.head
	if isSniperWeapon(shooter.weaponType) {
		target = enemy.chest
	}
	ideal := target.Sub(shooter.eyes).Normalize()
	real := anglesToR3Vector(shooter.pitch, shooter.yaw)
	idealPitch := -radiansToDegrees(math.Asin(ideal.Z))
	pitchError := math.Abs(idealPitch - normalizeAngle(float64(shooter.pitch)))
	idealYaw := radiansToDegrees(math.Atan2(ideal.Y, ideal.X))
	yawError := math.Abs(normalizeAngle(idealYaw - float64(shooter.yaw)))

	return models.FirstSeenData{
		Tick:                     tick,
		LastSeenTick:             tick,
		CrosshairPlacementError:  calculateAngle(ideal, real),
		PitchError:               pitchError,
		YawError:                 yawError,
		ShooterVelocity:          shooter.speed,
		ShooterVelocityAvailable: shooter.speedKnown,
	}
}

func isSniperWeapon(weapon common.EquipmentType) bool {
	return weapon == common.EqAWP || weapon == common.EqSSG08 || weapon == common.EqG3SG1 || weapon == common.EqScar20
}

func normalizeAngle(angle float64) float64 {
	for angle > 180 {
		angle -= 360
	}
	for angle < -180 {
		angle += 360
	}
	return angle
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}

func radiansToDegrees(value float64) float64 {
	return value * 180 / math.Pi
}
