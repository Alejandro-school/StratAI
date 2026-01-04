package handlers

import (
	"cs2-demo-service/models"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
)

// --- Helper Functions to reduce code duplication ---

// getTickRate returns the demo tick rate with fallback to 128
func getTickRate(ctx *models.DemoContext) float64 {
	if rate := ctx.Parser.TickRate(); rate > 0 {
		return rate
	}
	return 128.0
}

// ticksToMs converts tick delta to milliseconds using the demo's tick rate
func ticksToMs(ctx *models.DemoContext, deltaTicks int) float64 {
	tickRate := getTickRate(ctx)
	if deltaTicks == 0 {
		// Minimum 1 tick interval for "instant" events
		return 1000.0 / tickRate
	}
	return float64(deltaTicks) * (1000.0 / tickRate)
}

// calculatePlayerVelocity calculates 2D velocity with fallback logic
// Returns velocity in units/second
func calculatePlayerVelocity(ctx *models.DemoContext, player *common.Player) float64 {
	if player == nil {
		return 0
	}

	// Primary: Use entity velocity directly
	vel := player.Velocity()
	velocity := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y)
	if velocity > 0 {
		return velocity
	}

	steamID := player.SteamID64

	// Fallback 1: Calculate from 1-tick position difference
	if prevPos, ok := ctx.PreviousPlayerPosition[steamID]; ok {
		currPos := player.Position()
		dist := math.Sqrt(math.Pow(float64(currPos.X-prevPos.X), 2) + math.Pow(float64(currPos.Y-prevPos.Y), 2))
		tickRate := getTickRate(ctx)
		velocity = dist * tickRate
		if velocity > 0 {
			return velocity
		}
	}

	// Fallback 2: Calculate from sampled interval
	if lastPos, ok := ctx.LastPositions[steamID]; ok {
		currPos := player.Position()
		dist := math.Sqrt(math.Pow(float64(currPos.X-lastPos.X), 2) + math.Pow(float64(currPos.Y-lastPos.Y), 2))

		deltaTicks := ctx.Parser.GameState().IngameTick() - ctx.LastTick
		if deltaTicks > 0 {
			tickRate := getTickRate(ctx)
			timeSeconds := float64(deltaTicks) / tickRate
			if timeSeconds > 0 {
				velocity = dist / timeSeconds
			}
		}
	}

	return velocity
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

// --- NEW ENRICHMENT HELPERS ---

// getPlayerVelocityVector returns the 3D velocity components
func getPlayerVelocityVector(player *common.Player) (float64, float64, float64) {
	if player == nil {
		return 0, 0, 0
	}
	vel := player.Velocity()
	return vel.X, vel.Y, vel.Z
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
	gs := ctx.Parser.GameState()
	if gs == nil {
		return 0
	}

	tickRate := getTickRate(ctx)
	currentTick := gs.IngameTick()

	// CS2 standard round time: 115 seconds (1:55)
	const roundDuration = 115.0
	// C4 timer: 40 seconds
	const bombTimer = 40.0

	// If bomb is planted, use bomb timer
	if ctx.BombPlanted && ctx.BombTick > 0 {
		bombTicksElapsed := currentTick - ctx.BombTick
		bombTimeElapsed := float64(bombTicksElapsed) / tickRate
		bombTimeRemaining := bombTimer - bombTimeElapsed
		if bombTimeRemaining < 0 {
			bombTimeRemaining = 0
		}
		return math.Round(bombTimeRemaining*100) / 100 // Round to 2 decimals
	}

	// Calculate round time remaining based on freeze time end
	if ctx.FreezeTimeEndTick > 0 {
		roundTicksElapsed := currentTick - ctx.FreezeTimeEndTick
		roundTimeElapsed := float64(roundTicksElapsed) / tickRate
		roundTimeRemaining := roundDuration - roundTimeElapsed
		if roundTimeRemaining < 0 {
			roundTimeRemaining = 0
		}
		return math.Round(roundTimeRemaining*100) / 100 // Round to 2 decimals
	}

	// Fallback if freeze time end not tracked yet
	return roundDuration
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
