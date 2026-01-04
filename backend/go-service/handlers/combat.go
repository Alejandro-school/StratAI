package handlers

import (
	"cs2-demo-service/models"
	"fmt"
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterCombatHandlers registra handlers para eventos de combate
func RegisterCombatHandlers(ctx *models.DemoContext) {
	// PlayerHurt events for damage tracking
	ctx.Parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker == nil || e.Player == nil {
			return
		}
		attackerID := e.Attacker.SteamID64
		victimID := e.Player.SteamID64

		if _, ok := ctx.RoundDamage[attackerID]; !ok {
			ctx.RoundDamage[attackerID] = make(map[uint64]int)
		}
		ctx.RoundDamage[attackerID][victimID] += e.HealthDamage

		// FIX Bug 3: Track victim's HP BEFORE this damage event
		// e.Health is HP AFTER damage, so we add back the damage to get pre-damage HP
		// IMPORTANT: Cap at 100 because e.HealthDamage can include overkill damage
		preDamageHP := e.Health + e.HealthDamage
		if preDamageHP > 100 {
			preDamageHP = 100
		}
		ctx.LastKnownHealth[victimID] = preDamageHP

		// FIX: Track victim's active weapon WHILE ALIVE (for use in Kill events)
		// In Kill events, the victim is already dead and ActiveWeapon() returns nil
		if e.Player.IsAlive() {
			if activeWeapon := e.Player.ActiveWeapon(); activeWeapon != nil {
				ctx.LastActiveWeapon[victimID] = activeWeapon.String()
			}
		}
	})

	// Kill events
	ctx.Parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer == nil || e.Victim == nil {
			return
		}

		killEvent := models.KillEvent{
			Tick:              ctx.Parser.GameState().IngameTick(),
			Round:             ctx.CurrentRound,
			Killer:            e.Killer.Name,
			KillerSteamID:     e.Killer.SteamID64,
			Victim:            e.Victim.Name,
			VictimSteamID:     e.Victim.SteamID64,
			Weapon:            e.Weapon.String(),
			IsHeadshot:        e.IsHeadshot,
			IsWallbang:        e.IsWallBang(),
			PenetratedObjects: e.PenetratedObjects,
			AttackerBlind:     e.AttackerBlind,
			NoScope:           e.NoScope,
			ThroughSmoke:      e.ThroughSmoke,
			Distance:          e.Distance,
			KillerX:           float64(e.Killer.Position().X),
			KillerY:           float64(e.Killer.Position().Y),
			KillerZ:           float64(e.Killer.Position().Z),
			VictimX:           float64(e.Victim.Position().X),
			VictimY:           float64(e.Victim.Position().Y),
			VictimZ:           float64(e.Victim.Position().Z),
			// PHASE 1: Callouts
			KillerAreaName: getAreaName(ctx, e.Killer),
			VictimAreaName: getAreaName(ctx, e.Victim),
		}

		if e.Assister != nil {
			killEvent.Assister = e.Assister.Name
			killEvent.AssisterSteamID = e.Assister.SteamID64
		}

		// Attach Mechanics Data if available (within last 100 ticks ~800ms)
		if mech, ok := ctx.LastShotMechanics[e.Killer.SteamID64]; ok {
			if ctx.Parser.GameState().IngameTick()-mech.Tick < 100 {
				killEvent.CounterStrafeRating = mech.CounterStrafeRating
			}
		}

		// PHASE 1: Weapon state BEFORE (from SprayStartState if available, else PreviousWeaponState)
		// Prefer SprayStartState to capture the start of the duel/spray
		if startState, ok := ctx.SprayStartState[e.Killer.SteamID64]; ok && startState != nil {
			if startState.WeaponName == e.Weapon.String() {
				stateCopy := *startState
				killEvent.WeaponStateBefore = &stateCopy
			}
		}

		if killEvent.WeaponStateBefore == nil {
			if prevState, ok := ctx.PreviousWeaponState[e.Killer.SteamID64]; ok && prevState != nil {
				if prevState.WeaponName == e.Weapon.String() {
					stateCopy := *prevState
					killEvent.WeaponStateBefore = &stateCopy
				}
			}
		}

		// PHASE 1: Weapon state AFTER (Current state from entity)
		// We capture the state immediately after the kill (current tick)
		if activeWeapon := e.Killer.ActiveWeapon(); activeWeapon != nil {
			killEvent.WeaponStateAfter = &models.WeaponStateSnapshot{
				WeaponName:  activeWeapon.String(),
				AmmoInMag:   activeWeapon.AmmoInMagazine(),
				AmmoReserve: activeWeapon.AmmoReserve(),
				IsReloading: e.Killer.IsReloading,
				ZoomLevel:   int(activeWeapon.ZoomLevel()),
			}
		}

		// DEBUG: Print first 3 kills with weapon state
		if len(ctx.MatchData.Kills) < 3 {
			fmt.Printf("🔫 Kill #%d: %s con %s\n", len(ctx.MatchData.Kills)+1, e.Killer.Name, e.Weapon.String())
			if killEvent.WeaponStateBefore != nil {
				fmt.Printf("   BEFORE: mag=%d reserve=%d\n", killEvent.WeaponStateBefore.AmmoInMag, killEvent.WeaponStateBefore.AmmoReserve)
			}
			if killEvent.WeaponStateAfter != nil {
				fmt.Printf("   AFTER:  mag=%d reserve=%d\n", killEvent.WeaponStateAfter.AmmoInMag, killEvent.WeaponStateAfter.AmmoReserve)
			}
		}

		ctx.MatchData.Kills = append(ctx.MatchData.Kills, killEvent)

		// ADD TO TIMELINE
		AddKillToTimeline(ctx, ctx.Parser.GameState().IngameTick(), &killEvent)

		// Capture raw event for consolidated duel system
		captureRawKillEvent(ctx, &e)
	})

	// Player hurt events
	ctx.Parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Player == nil || e.Attacker == nil {
			return
		}

		damageEvent := models.DamageEvent{
			Tick:         ctx.Parser.GameState().IngameTick(),
			Round:        ctx.CurrentRound,
			Attacker:     e.Attacker.Name,
			Victim:       e.Player.Name,
			Weapon:       e.Weapon.String(),
			HealthDamage: e.HealthDamage,
			ArmorDamage:  e.ArmorDamage,
			HitGroup:     fmt.Sprintf("%v", e.HitGroup),
			VictimHealth: e.Health,
			// PHASE 1: Callouts
			AttackerAreaName: getAreaName(ctx, e.Attacker),
			VictimAreaName:   getAreaName(ctx, e.Player),
		}

		// PHASE 1: Weapon state at damage (from SprayStartState if available)
		if startState, ok := ctx.SprayStartState[e.Attacker.SteamID64]; ok && startState != nil {
			// Verify weapon match just in case
			if startState.WeaponName == e.Weapon.String() {
				stateSnapshot := *startState
				damageEvent.WeaponState = &stateSnapshot
			}
		}

		// Fallback: if no SprayStart or weapon mismatch, try PreviousWeaponState
		if damageEvent.WeaponState == nil {
			if prevState, ok := ctx.PreviousWeaponState[e.Attacker.SteamID64]; ok && prevState != nil {
				if prevState.WeaponName == e.Weapon.String() {
					stateSnapshot := *prevState
					damageEvent.WeaponState = &stateSnapshot
				}
			}
		}

		// PHASE 1: Current Weapon State (at moment of impact)
		if activeWeapon := e.Attacker.ActiveWeapon(); activeWeapon != nil {
			damageEvent.WeaponStateCurrent = &models.WeaponStateSnapshot{
				WeaponName:  activeWeapon.String(),
				AmmoInMag:   activeWeapon.AmmoInMagazine(),
				AmmoReserve: activeWeapon.AmmoReserve(),
				IsReloading: e.Attacker.IsReloading,
				ZoomLevel:   int(activeWeapon.ZoomLevel()),
			}
		}

		ctx.MatchData.Damage = append(ctx.MatchData.Damage, damageEvent)

		// ADD TO TIMELINE
		AddDamageToTimeline(ctx, ctx.Parser.GameState().IngameTick(), &damageEvent)

		// Capture raw event for consolidated duel system
		captureRawDamageEvent(ctx, &e)
	})

	// PHASE 1: Weapon fire events - for spray tracking only
	ctx.Parser.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil {
			return
		}

		sid := e.Shooter.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()

		// Detect start of spray
		lastFireTick, exists := ctx.LastCombatFireTick[sid]
		// If no last fire, or last fire was long ago (> 32 ticks = 0.5s)
		isNewSpray := !exists || (currentTick-lastFireTick > 32)

		if isNewSpray {
			// Capture state BEFORE this shot.
			// We use PreviousWeaponState because it holds the state from the end of the previous tick.
			// Since this is the first shot of a spray, the previous tick state is the "full ammo" state.
			if prevState, ok := ctx.PreviousWeaponState[sid]; ok && prevState != nil {
				// Only use previous state if weapon matches (avoids quick-switch issues)
				if prevState.WeaponName == e.Weapon.String() {
					stateCopy := *prevState
					ctx.SprayStartState[sid] = &stateCopy
				} else {
					// If weapon changed, we don't have a valid "Before" state from previous tick.
					ctx.SprayStartState[sid] = nil
				}
			}
		}

		// Track para spray detection
		ctx.LastCombatFireTick[sid] = currentTick
	})

	// PHASE 1: Weapon Reload events
	ctx.Parser.RegisterEventHandler(func(e events.WeaponReload) {
		if e.Player == nil {
			return
		}

		// Capture reload event
		// Note: WeaponReload event triggers when reload STARTS or ENDS?
		// demoinfocs documentation says: "WeaponReload is triggered when a player starts reloading."
		// So we can capture the state BEFORE the reload completes (low ammo).

		activeWeapon := e.Player.ActiveWeapon()
		weaponName := "unknown"
		ammoInMag := 0
		ammoReserve := 0

		if activeWeapon != nil {
			weaponName = activeWeapon.String()
			ammoInMag = activeWeapon.AmmoInMagazine()
			ammoReserve = activeWeapon.AmmoReserve()
		}

		reloadEvent := models.ReloadEvent{
			Tick:        ctx.Parser.GameState().IngameTick(),
			Round:       ctx.CurrentRound,
			Player:      e.Player.Name,
			SteamID:     e.Player.SteamID64,
			Weapon:      weaponName,
			AmmoBefore:  ammoInMag,
			AmmoReserve: ammoReserve,
			X:           float64(e.Player.Position().X),
			Y:           float64(e.Player.Position().Y),
			Z:           float64(e.Player.Position().Z),
		}

		ctx.MatchData.Reloads = append(ctx.MatchData.Reloads, reloadEvent)
	})

	// PHASE 1: FrameDone - Rotate weapon states AFTER all tick events
	// PreviousWeaponState = state from tick N-1 (for "before" in kills/damage)
	// LastWeaponState = state from tick N (for "after" in kills)
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		if !ctx.InRound {
			return
		}

		// For all alive players, rotate states
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			if player == nil || !player.IsAlive() {
				continue
			}

			sid := player.SteamID64
			if sid == 0 {
				continue
			}

			activeWeapon := player.ActiveWeapon()
			if activeWeapon == nil {
				continue
			}

			// FIX: Track active weapon name for use in kill events (victim weapon)
			ctx.LastActiveWeapon[sid] = activeWeapon.String()

			// Save current LastWeaponState as PreviousWeaponState (tick N-1)
			if lastState, ok := ctx.LastWeaponState[sid]; ok && lastState != nil {
				stateCopy := *lastState
				ctx.PreviousWeaponState[sid] = &stateCopy
			}

			// Update LastWeaponState with current tick values (tick N)
			ctx.LastWeaponState[sid] = &models.WeaponStateSnapshot{
				WeaponName:  activeWeapon.String(),
				AmmoInMag:   activeWeapon.AmmoInMagazine(),
				AmmoReserve: activeWeapon.AmmoReserve(),
				IsReloading: player.IsReloading,
				ZoomLevel:   int(activeWeapon.ZoomLevel()),
			}

			// Update PreviousPlayerPosition
			ctx.PreviousPlayerPosition[sid] = player.Position()
		}
	})
}

// getVictimWeapon returns the victim's active weapon name
func getVictimWeapon(victim *common.Player) string {
	if victim.ActiveWeapon() != nil {
		return victim.ActiveWeapon().String()
	}
	return "Unknown"
}

// getAreaName returns the area/callout name for a player's position
func getAreaName(ctx *models.DemoContext, player *common.Player) string {
	if player == nil {
		return ""
	}
	// Try MapManager first
	if ctx.MapManager != nil {
		if callout := ctx.MapManager.GetCallout(player.Position()); callout != "" {
			return callout
		}
	}
	// Fallback to parser's nav mesh
	return player.LastPlaceName()
}

// --- Crosshair Calculation Helpers ---

func calculateCrosshairMetrics(shooter, victim *common.Player) (float64, float64, float64) {
	if shooter == nil || victim == nil {
		return 0, 0, 0
	}

	// Determinar objetivo (Cabeza vs Cuerpo) según arma
	targetPos := victim.Position()
	activeWeapon := shooter.ActiveWeapon()
	isSniper := false
	if activeWeapon != nil {
		wType := activeWeapon.Type
		if wType == common.EqAWP || wType == common.EqSSG08 || wType == common.EqG3SG1 || wType == common.EqScar20 {
			isSniper = true
		}
	}

	if isSniper {
		targetPos.Z += 40.0 // Altura aproximada pecho/estómago
	} else {
		targetPos.Z += 62.0 // Altura aproximada cabeza
	}

	playerEyePos := shooter.Position()
	playerEyePos.Z += 64.0 // Altura ojos

	// Vector Ideal (Desde ojos a objetivo)
	vecIdeal := r3.Vector{
		X: targetPos.X - playerEyePos.X,
		Y: targetPos.Y - playerEyePos.Y,
		Z: targetPos.Z - playerEyePos.Z,
	}.Normalize()

	// Vector Real (Hacia donde mira el jugador)
	// NOTE: ViewDirectionX is Yaw, ViewDirectionY is Pitch in demoinfocs-golang
	vecReal := anglesToR3Vector(shooter.ViewDirectionY(), shooter.ViewDirectionX())

	// Calcular ángulo total
	angleError := calculateAngle(vecIdeal, vecReal)

	// Calcular Pitch y Yaw Error
	idealPitch := float64(-math.Asin(vecIdeal.Z) * (180.0 / math.Pi))
	realPitch := float64(shooter.ViewDirectionY())
	pitchError := math.Abs(idealPitch - realPitch)

	idealYaw := float64(math.Atan2(vecIdeal.Y, vecIdeal.X) * (180.0 / math.Pi))
	realYaw := float64(shooter.ViewDirectionX())

	yawDiff := idealYaw - realYaw
	for yawDiff > 180 {
		yawDiff -= 360
	}
	for yawDiff < -180 {
		yawDiff += 360
	}
	yawError := math.Abs(yawDiff)

	return angleError, pitchError, yawError
}

func anglesToR3Vector(pitch, yaw float32) r3.Vector {
	p := float64(pitch) * math.Pi / 180.0
	y := float64(yaw) * math.Pi / 180.0
	sinP := math.Sin(p)
	cosP := math.Cos(p)
	sinY := math.Sin(y)
	cosY := math.Cos(y)
	return r3.Vector{X: cosP * cosY, Y: cosP * sinY, Z: -sinP}
}

func calculateAngle(v1, v2 r3.Vector) float64 {
	dot := v1.Dot(v2)
	if dot > 1.0 {
		dot = 1.0
	} else if dot < -1.0 {
		dot = -1.0
	}
	angleRad := math.Acos(dot)
	return angleRad * (180.0 / math.Pi)
}
