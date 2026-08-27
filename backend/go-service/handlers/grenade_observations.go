package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
	"math"
	"sort"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func utilityPlayer(player *common.Player, source string) utility.PlayerRef {
	if player == nil || player.SteamID64 == 0 {
		return utility.PlayerRef{
			Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		}
	}
	return utility.PlayerRef{
		ID: player.SteamID64, Name: player.Name, Side: getTeamSide(player),
		Status: utility.AvailabilityObserved, Source: source,
	}
}

func utilityRelation(actor, victim *common.Player) utility.Relation {
	if actor == nil || victim == nil || actor.SteamID64 == 0 || victim.SteamID64 == 0 {
		return utility.RelationUnknown
	}
	if actor.SteamID64 == victim.SteamID64 {
		return utility.RelationSelf
	}
	if actor.Team == victim.Team {
		return utility.RelationTeammate
	}
	return utility.RelationEnemy
}

func observeThrowView(player *common.Player) (observation utility.ViewObservation) {
	observation = utility.ViewObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
	if player == nil || player.Entity == nil {
		return observation
	}
	defer func() {
		if recover() != nil {
			observation = utility.ViewObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
		}
	}()
	return normalizedThrowView(float64(player.ViewDirectionX()), float64(player.ViewDirectionY()))
}

func normalizedThrowView(yaw, pitch float64) utility.ViewObservation {
	if math.IsNaN(yaw) || math.IsInf(yaw, 0) || math.IsNaN(pitch) || math.IsInf(pitch, 0) ||
		pitch < -90 || pitch > 90 {
		return utility.ViewObservation{
			Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		}
	}
	yaw = math.Mod(yaw+180, 360)
	if yaw < 0 {
		yaw += 360
	}
	yaw -= 180
	vector := viewVector(yaw, pitch)
	return utility.ViewObservation{
		Yaw: yaw, Pitch: pitch, Vector: vector,
		Status: utility.AvailabilityObserved, Source: utility.SourcePlayerView,
	}
}

func observeThrowerVelocity(
	ctx *models.DemoContext,
	player *common.Player,
	round int,
	tick int,
) utility.VelocityObservation {
	if player == nil {
		return utility.VelocityObservation{
			Status: utility.AvailabilityNotApplicable, Source: utility.SourceVelocityNotApplicable,
		}
	}
	estimate := ctx.PlayerMotion.ObservePlayer(player, round, tick, getTickRate(ctx))
	observation := utility.VelocityObservation{
		ObservedTick: estimate.ObservedTick, MeasurementWindowTicks: estimate.IntervalTicks,
		Status: utility.AvailabilityUnavailable, Source: string(estimate.Source),
	}
	if !estimate.Available {
		return observation
	}
	observation.Vector = toUtilityVector(estimate.Vector)
	observation.HorizontalSpeed = estimate.HorizontalSpeed()
	observation.Status = utility.AvailabilityObserved
	return observation
}

func observeThrowStance(player *common.Player) (observation utility.StanceObservation) {
	observation = utility.StanceObservation{
		Value: utility.StanceUnknown, Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
	}
	if player == nil || player.Entity == nil {
		return observation
	}
	defer func() {
		if recover() != nil {
			observation = utility.StanceObservation{
				Value: utility.StanceUnknown, Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
			}
		}
	}()
	stance := utility.StanceStanding
	switch {
	case player.IsAirborne():
		stance = utility.StanceAirborne
	case player.IsDucking() && player.IsWalking():
		stance = utility.StanceCrouchWalking
	case player.IsDucking():
		stance = utility.StanceCrouching
	case player.IsWalking():
		stance = utility.StanceWalking
	}
	return utility.StanceObservation{
		Value: stance, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerState,
	}
}

func observePlayerArea(ctx *models.DemoContext, player *common.Player) utility.StringObservation {
	if player == nil || player.Entity == nil {
		return unavailableArea()
	}
	if ctx.MapManager != nil {
		if area := ctx.MapManager.GetCallout(player.Position()); area != "" {
			return utility.StringObservation{
				Value: area, Status: utility.AvailabilityObserved, Source: utility.SourceMapCallout,
			}
		}
	}
	if area := player.LastPlaceName(); area != "" {
		return utility.StringObservation{
			Value: area, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerLastPlace,
		}
	}
	return unavailableArea()
}

func observePositionArea(ctx *models.DemoContext, position utility.Vector) utility.StringObservation {
	worldPosition := r3.Vector{X: position.X, Y: position.Y, Z: position.Z}
	if ctx.MapManager != nil {
		if area := ctx.MapManager.GetCallout(worldPosition); area != "" {
			return utility.StringObservation{
				Value: area, Status: utility.AvailabilityObserved, Source: utility.SourceMapCallout,
			}
		}
	}
	if ctx.Parser == nil || ctx.Parser.GameState() == nil {
		return unavailableArea()
	}
	closestDistance := math.Inf(1)
	closestID := uint64(math.MaxUint64)
	closestArea := ""
	for _, player := range ctx.Parser.GameState().Participants().All() {
		if player == nil || player.Entity == nil || !player.IsAlive() {
			continue
		}
		area := player.LastPlaceName()
		if area == "" {
			continue
		}
		distance := player.Position().Distance(worldPosition)
		if distance > 800 || distance > closestDistance ||
			(distance == closestDistance && player.SteamID64 >= closestID) {
			continue
		}
		closestDistance, closestID, closestArea = distance, player.SteamID64, area
	}
	if closestArea == "" {
		return unavailableArea()
	}
	return utility.StringObservation{
		Value: closestArea, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerLastPlace,
	}
}

func deterministicInfernoCenter(inferno *common.Inferno) (utility.Vector, utility.Availability) {
	if inferno == nil {
		return utility.Vector{}, utility.AvailabilityUnavailable
	}
	fires := inferno.Fires().List()
	positions := make([]utility.Vector, 0, len(fires))
	for _, fire := range fires {
		position := toUtilityVector(fire.Vector)
		if finiteUtilityVector(position) {
			positions = append(positions, position)
		}
	}
	if len(positions) == 0 {
		return utility.Vector{}, utility.AvailabilityUnavailable
	}
	sort.Slice(positions, func(left, right int) bool {
		return compareUtilityVector(positions[left], positions[right]) < 0
	})
	center := utility.Vector{}
	for _, position := range positions {
		center.X += position.X
		center.Y += position.Y
		center.Z += position.Z
	}
	count := float64(len(positions))
	center.X, center.Y, center.Z = center.X/count, center.Y/count, center.Z/count
	return center, utility.AvailabilityObserved
}

func infernoEntityID(inferno *common.Inferno) int {
	if inferno == nil || inferno.Entity == nil {
		return 0
	}
	return inferno.Entity.ID()
}

func grenadeEntityID(entityID int, equipment *common.Equipment) int {
	if entityID != 0 {
		return entityID
	}
	if equipment == nil || equipment.Entity == nil {
		return 0
	}
	return equipment.Entity.ID()
}

func (state *utilityHandlerState) smokeEffectKey(
	entityID int,
	equipment *common.Equipment,
	create bool,
) int64 {
	if id := grenadeEntityID(entityID, equipment); id != 0 {
		return int64(id)
	}
	if equipment == nil {
		return 0
	}
	if id, exists := state.equipmentEffects[equipment]; exists {
		return id
	}
	if !create {
		return 0
	}
	state.nextFallbackEffect--
	state.equipmentEffects[equipment] = state.nextFallbackEffect
	return state.nextFallbackEffect
}

func toUtilityVector(vector r3.Vector) utility.Vector {
	return utility.Vector{X: vector.X, Y: vector.Y, Z: vector.Z}
}

func finiteUtilityVector(vector utility.Vector) bool {
	return !math.IsNaN(vector.X) && !math.IsInf(vector.X, 0) &&
		!math.IsNaN(vector.Y) && !math.IsInf(vector.Y, 0) &&
		!math.IsNaN(vector.Z) && !math.IsInf(vector.Z, 0)
}

func viewVector(yaw, pitch float64) utility.Vector {
	yawRadians, pitchRadians := yaw*math.Pi/180, pitch*math.Pi/180
	return utility.Vector{
		X: math.Cos(yawRadians) * math.Cos(pitchRadians),
		Y: math.Sin(yawRadians) * math.Cos(pitchRadians),
		Z: -math.Sin(pitchRadians),
	}
}

func anglesToVector(yaw, pitch float32) models.AI_Vector {
	vector := viewVector(float64(yaw), float64(pitch))
	return models.AI_Vector{X: vector.X, Y: vector.Y, Z: vector.Z}
}

func findLandArea(ctx *models.DemoContext, position r3.Vector) string {
	return observePositionArea(ctx, toUtilityVector(position)).Value
}

func getThrowerArea(ctx *models.DemoContext, thrower *common.Player) string {
	return observePlayerArea(ctx, thrower).Value
}

func unavailableArea() utility.StringObservation {
	return utility.StringObservation{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
	}
}

func observedVector(
	value utility.Vector,
	status utility.Availability,
	source string,
) utility.VectorObservation {
	if status != utility.AvailabilityObserved {
		return utility.VectorObservation{
			Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		}
	}
	return utility.VectorObservation{Value: value, Status: status, Source: source}
}
