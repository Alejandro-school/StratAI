package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"
	"cs2-demo-service/pkg/playerstate"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type playerVelocityObservation struct {
	Horizontal       *float64
	X                *float64
	Y                *float64
	Z                *float64
	Available        bool
	Source           string
	Observation      string
	MeasurementTicks *int
	ObservedTick     *int
}

type ObjectiveClockPhase string

const (
	ObjectiveClockPreplant ObjectiveClockPhase = "preplant"
	ObjectiveClockPlanting ObjectiveClockPhase = "planting"
	ObjectiveClockPlanted  ObjectiveClockPhase = "planted"
	ObjectiveClockDefusing ObjectiveClockPhase = "defusing"
	ObjectiveClockResolved ObjectiveClockPhase = "resolved"
)

// ObjectiveClockSnapshot keeps the round and bomb clocks separate while also
// exposing the phase-relevant value used by the existing combat contract.
type ObjectiveClockSnapshot struct {
	Phase               ObjectiveClockPhase
	PhaseTimeRemaining  float64
	RoundClockRemaining *float64
	BombTimeRemaining   *float64
}

// --- Helper Functions to reduce code duplication ---

// getTickRate returns the demo tick rate with a CS2 demo fallback.
func getTickRate(ctx *models.DemoContext) float64 {
	if rate := ctx.Parser.TickRate(); rate > 0 {
		return rate
	}
	return 64.0
}

// ticksToMs converts tick delta to milliseconds using the demo's tick rate
func ticksToMs(ctx *models.DemoContext, deltaTicks int) float64 {
	tickRate := getTickRate(ctx)
	return float64(deltaTicks) * (1000.0 / tickRate)
}

func isBombPlantedNow(ctx *models.DemoContext) bool {
	return ensureObjectiveTracker(ctx).Snapshot().IsPlantedNow
}

func observePlayerVelocity(ctx *models.DemoContext, player *common.Player) playerVelocityObservation {
	if player == nil {
		return playerVelocityObservation{
			Source:      string(playerstate.VelocitySourceNotApplicable),
			Observation: models.VelocityObservationUnavailable,
		}
	}

	motion := ctx.PlayerMotion.ObservePlayer(
		player,
		ctx.ActualRoundNumber,
		ctx.Parser.GameState().IngameTick(),
		getTickRate(ctx),
	)
	if motion.Available {
		return velocityObservationFromEstimate(motion, models.VelocityObservationCurrentTick)
	}
	if player.IsAlive() {
		return velocityObservationFromEstimate(motion, models.VelocityObservationUnavailable)
	}
	lastAliveMotion, ok := ctx.PlayerMotion.LastAlive(player.SteamID64, ctx.ActualRoundNumber)
	if !ok {
		return velocityObservationFromEstimate(motion, models.VelocityObservationUnavailable)
	}
	return velocityObservationFromEstimate(lastAliveMotion, models.VelocityObservationLastAlive)
}

func velocityObservationFromEstimate(
	estimate playerstate.MotionEstimate,
	observationKind string,
) playerVelocityObservation {
	observation := playerVelocityObservation{
		Available:   estimate.Available,
		Source:      string(estimate.Source),
		Observation: observationKind,
	}
	if !estimate.Available {
		observation.Observation = models.VelocityObservationUnavailable
		return observation
	}
	horizontal := estimate.HorizontalSpeed()
	x, y, z := estimate.Vector.X, estimate.Vector.Y, estimate.Vector.Z
	measurementTicks, observedTick := estimate.IntervalTicks, estimate.ObservedTick
	observation.Horizontal = &horizontal
	observation.X = &x
	observation.Y = &y
	observation.Z = &z
	observation.MeasurementTicks = &measurementTicks
	observation.ObservedTick = &observedTick
	return observation
}

// getPlayerTeamString returns "CT" or "T" based on player team
func getPlayerTeamString(player *common.Player) string {
	if player.Team == 2 { // Terrorists
		return "T"
	}
	return "CT"
}

// isScopedWeapon checks if the weapon has a scope (for no_scope relevance)
func isScopedWeapon(weapon string) bool {
	scopedWeapons := map[string]bool{
		"AWP":     true,
		"SSG 08":  true, // Scout
		"SCAR-20": true,
		"G3SG1":   true,
		"AUG":     true,
		"SG 553":  true,
		"SG 556":  true, // Alternative name
		"Scar-20": true, // Alternative casing
		"Scout":   true, // Alternative name
	}
	return scopedWeapons[weapon]
}

// getNoScopePtr returns a pointer to the noScope value only for scoped weapons
// Returns nil for non-scoped weapons (field will be omitted from JSON)
func getNoScopePtr(weapon string, noScope bool) *bool {
	if isScopedWeapon(weapon) {
		return &noScope
	}
	return nil
}

// getZoomLevelPtr returns a pointer to the zoom level only for scoped weapons
// Returns nil for non-scoped weapons (field will be omitted from JSON)
func getZoomLevelPtr(weapon string, zoomLevel int) *int {
	if isScopedWeapon(weapon) {
		return &zoomLevel
	}
	return nil
}

// isPlayerAirborne checks if player is in the air (jumping/falling)
// Uses official Source engine FL_ONGROUND flag for 100% accuracy
func isPlayerAirborne(player *common.Player) bool {
	if player == nil {
		return false
	}
	// FL_ONGROUND = (1 << 0) = bit 0
	// If FL_ONGROUND is NOT set, player is airborne
	const FL_ONGROUND = 1 << 0
	return (player.Flags() & FL_ONGROUND) == 0
}

// isPlayerDucking checks if player is crouching
func isPlayerDucking(player *common.Player) bool {
	if player == nil {
		return false
	}
	return player.IsDucking()
}

// isPlayerWalking checks if player is shift-walking (slow movement)
func isPlayerWalking(player *common.Player) bool {
	if player == nil {
		return false
	}
	return player.IsWalking()
}

// getFlashDuration returns remaining flash blindness duration in seconds
func getFlashDuration(player *common.Player) float64 {
	if player == nil {
		return 0
	}
	return float64(player.FlashDuration)
}

// getAmmoInMagazine returns current ammo in the active weapon's magazine
// Returns 0 for grenades and utilities (not relevant)
func getAmmoInMagazine(player *common.Player) int {
	if player == nil {
		return 0
	}
	if weapon := player.ActiveWeapon(); weapon != nil {
		// Skip grenades and utilities - ammo not meaningful
		if weapon.Class() == common.EqClassGrenade || weapon.Class() == common.EqClassEquipment {
			return 0
		}
		return weapon.AmmoInMagazine()
	}
	return 0
}

// hasHelmet checks if player has head protection
func hasHelmet(player *common.Player) bool {
	if player == nil {
		return false
	}
	return player.HasHelmet()
}

// hasDefuser checks if CT player has defuse kit
func hasDefuser(player *common.Player) bool {
	if player == nil {
		return false
	}
	return player.HasDefuseKit()
}

// calculateRoundTimeRemaining calculates seconds left in the round
// Returns time remaining based on round phase:
// - If bomb planted: time until explosion (40s max)
// - If bomb not planted: time remaining in round (115s max standard)
func calculateRoundTimeRemaining(ctx *models.DemoContext) float64 {
	return CaptureObjectiveClockSnapshot(ctx).PhaseTimeRemaining
}

// CaptureObjectiveClockSnapshot returns causal timer values for the current
// parser tick. BombTimeRemaining is nil outside an active post-plant.
func CaptureObjectiveClockSnapshot(ctx *models.DemoContext) ObjectiveClockSnapshot {
	gs := ctx.Parser.GameState()
	if gs == nil {
		return ObjectiveClockSnapshot{Phase: ObjectiveClockPreplant}
	}

	tickRate := getTickRate(ctx)
	currentTick := gs.IngameTick()

	roundDuration := 115.0
	bombTimer := 40.0
	if rules := gs.Rules(); rules != nil {
		if duration, err := rules.RoundTime(); err == nil && duration > 0 {
			roundDuration = duration.Seconds()
		}
		if duration, err := rules.BombTime(); err == nil && duration > 0 {
			bombTimer = duration.Seconds()
		}
	}

	objectiveState := ensureObjectiveTracker(ctx).Snapshot()
	return buildObjectiveClockSnapshot(
		objectiveState,
		currentTick,
		ctx.FreezeTimeEndTick,
		tickRate,
		roundDuration,
		bombTimer,
	)
}

func buildObjectiveClockSnapshot(
	objectiveState objective.Snapshot,
	currentTick, freezeTimeEndTick int,
	tickRate, roundDuration, bombTimer float64,
) ObjectiveClockSnapshot {
	if tickRate <= 0 {
		tickRate = 64
	}
	roundTimeRemaining := roundDuration
	if freezeTimeEndTick > 0 {
		roundTicksElapsed := max(0, currentTick-freezeTimeEndTick)
		roundTimeRemaining = roundDuration - float64(roundTicksElapsed)/tickRate
	}
	roundTimeRemaining = roundedNonNegativeClock(roundTimeRemaining)

	if objectiveState.Phase == objective.PhaseResolved {
		return ObjectiveClockSnapshot{
			Phase:              ObjectiveClockResolved,
			PhaseTimeRemaining: 0,
		}
	}
	clockPhase := ObjectiveClockPreplant
	switch objectiveState.State {
	case objective.StatePlanting:
		clockPhase = ObjectiveClockPlanting
	case objective.StatePlanted:
		clockPhase = ObjectiveClockPlanted
	case objective.StateDefusing:
		clockPhase = ObjectiveClockDefusing
	}

	if objectiveState.IsPlantedNow && objectiveState.PlantTick > 0 {
		bombTicksElapsed := max(0, currentTick-objectiveState.PlantTick)
		bombTimeElapsed := float64(bombTicksElapsed) / tickRate
		bombTimeRemaining := roundedNonNegativeClock(bombTimer - bombTimeElapsed)
		return ObjectiveClockSnapshot{
			Phase:              clockPhase,
			PhaseTimeRemaining: bombTimeRemaining,
			BombTimeRemaining:  &bombTimeRemaining,
		}
	}

	return ObjectiveClockSnapshot{
		Phase:               clockPhase,
		PhaseTimeRemaining:  roundTimeRemaining,
		RoundClockRemaining: &roundTimeRemaining,
	}
}

func roundedNonNegativeClock(seconds float64) float64 {
	if seconds < 0 {
		seconds = 0
	}
	return math.Round(seconds*100) / 100
}

// getCounterStrafeRating retrieves the counter-strafe quality score
func getCounterStrafeRating(ctx *models.DemoContext, steamID uint64) float64 {
	if mech, ok := ctx.LastShotMechanics[steamID]; ok {
		// Check if mechanics data is recent enough (within 100 ticks)
		if ctx.Parser.GameState().IngameTick()-mech.Tick < 100 {
			return mech.CounterStrafeRating
		}
	}
	return 0
}

// getShotsFired returns the spray bullet index from the active spray
func getShotsFired(ctx *models.DemoContext, steamID uint64) int {
	if spray, ok := ctx.CurrentSpray[steamID]; ok && spray != nil {
		return spray.ShotCount
	}
	return 0
}

// --- FINAL ENRICHMENT HELPERS ---

// getAmmoReserve returns total bullets remaining for the active weapon
// Returns 0 for grenades and utilities (not relevant)
func getAmmoReserve(player *common.Player) int {
	if player == nil {
		return 0
	}
	if weapon := player.ActiveWeapon(); weapon != nil {
		// Skip grenades and utilities - ammo not meaningful
		if weapon.Class() == common.EqClassGrenade || weapon.Class() == common.EqClassEquipment {
			return 0
		}
		return weapon.AmmoReserve()
	}
	return 0
}

// isPlayerScoped checks if player is scoped in (for snipers)
func isPlayerScoped(player *common.Player) bool {
	if player == nil {
		return false
	}
	if weapon := player.ActiveWeapon(); weapon != nil {
		return weapon.ZoomLevel() > 0
	}
	return false
}

// getZoomLevel returns the current zoom level (0=none, 1=first, 2=second)
func getZoomLevel(player *common.Player) int {
	if player == nil {
		return 0
	}
	if weapon := player.ActiveWeapon(); weapon != nil {
		return int(weapon.ZoomLevel())
	}
	return 0
}

// calculateHeightDiff returns the vertical distance between killer and victim
// Positive = killer above victim, Negative = killer below victim
func calculateHeightDiff(killer, victim *common.Player) float64 {
	if killer == nil || victim == nil {
		return 0
	}
	return killer.Position().Z - victim.Position().Z
}
