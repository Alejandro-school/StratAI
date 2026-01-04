package handlers

import (
	"cs2-demo-service/models"
	"fmt"
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterGrenadeHandlers registra handlers de granadas
func RegisterGrenadeHandlers(ctx *models.DemoContext) {
	// Track bounces
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileBounce) {
		if e.Projectile == nil {
			return
		}
		ctx.GrenadeBounces[e.Projectile.Entity.ID()]++
	})

	// Flashbang explode
	ctx.Parser.RegisterEventHandler(func(e events.FlashExplode) {
		if e.Thrower == nil {
			return
		}

		tickExplode := ctx.Parser.GameState().IngameTick()
		flashKey := fmt.Sprintf("%d|%d|%d", ctx.CurrentRound, tickExplode, e.Thrower.SteamID64)
		if _, seen := ctx.ProcessedFlashExplodes[flashKey]; seen {
			return
		}
		ctx.ProcessedFlashExplodes[flashKey] = struct{}{}

		flashEvent := models.FlashEvent{
			Tick:    tickExplode,
			Round:   ctx.CurrentRound,
			Thrower: e.Thrower.Name,
			X:       float64(e.Position.X),
			Y:       float64(e.Position.Y),
			Z:       float64(e.Position.Z),
			Victims: []models.FlashVictim{},
			// PHASE 1: Callouts
			ThrowerAreaName: getThrowerArea(ctx, e.Thrower),
			// LandAreaName can be inferred from position if needed via MapManager
			EnemiesBlinded: 0,
			AlliesBlinded:  0,
		}

		blindedPlayers := []models.AI_BlindedPlayer{}

		// Recorrer jugadores flasheados usando FlashDuration nativo
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			if player.FlashDuration > 0 {
				// Convert team enum to string
				teamStr := "Unknown"
				if player.Team == common.TeamCounterTerrorists {
					teamStr = "CT"
				} else if player.Team == common.TeamTerrorists {
					teamStr = "T"
				}

				victim := models.FlashVictim{
					Name:     player.Name,
					Duration: player.FlashDuration,
					Team:     teamStr, // PHASE 1: CT o T
				}
				flashEvent.Victims = append(flashEvent.Victims, victim)

				// AI Blinded Player
				blindedPlayers = append(blindedPlayers, models.AI_BlindedPlayer{
					Name:     player.Name,
					Duration: float32(player.FlashDuration),
					Team:     teamStr,
					IsEnemy:  player.Team != e.Thrower.Team,
				})

				// PHASE 1: Team Flash Detection
				if player.Team == e.Thrower.Team {
					flashEvent.AlliesBlinded++
				} else {
					flashEvent.EnemiesBlinded++
				}
			}
		}

		ctx.MatchData.Flashes = append(ctx.MatchData.Flashes, flashEvent)

		// ADD TO TIMELINE
		AddGrenadeToTimeline(ctx, tickExplode, "flash", &flashEvent)

		// ADD TO AI MODEL
		landArea := findLandArea(ctx, r3.Vector{X: float64(e.Position.X), Y: float64(e.Position.Y), Z: float64(e.Position.Z)})

		// Find trajectory
		trajID, traj := findGrenadeTrajectory(ctx, e.Thrower.SteamID64, "Flashbang", r3.Vector{X: float64(e.Position.X), Y: float64(e.Position.Y), Z: float64(e.Position.Z)})

		startPos := models.AI_Vector{X: 0, Y: 0, Z: 0}
		tickThrow := 0
		didBounce := false
		if traj != nil {
			tickThrow = traj.TickThrow
			if len(traj.Positions) > 0 {
				p := traj.Positions[0]
				startPos = models.AI_Vector{X: p.X, Y: p.Y, Z: p.Z}

				// Heuristic: If landArea is empty and throw was short, use thrower area
				if landArea == "" {
					dist := math.Sqrt(math.Pow(p.X-float64(e.Position.X), 2) + math.Pow(p.Y-float64(e.Position.Y), 2) + math.Pow(p.Z-float64(e.Position.Z), 2))
					if dist < 600 {
						landArea = getThrowerArea(ctx, e.Thrower)
					}
				}
			}
			didBounce = ctx.GrenadeBounces[trajID] > 0
		}

		aiGrenadeEvent := models.AI_GrenadeEvent{
			Round:           ctx.CurrentRound,
			Type:            "Flashbang",
			Thrower:         e.Thrower.Name,
			TickThrow:       tickThrow,
			TickExplode:     tickExplode,
			ThrowerAreaName: getThrowerArea(ctx, e.Thrower),
			ThrowerSide:     getTeamSide(e.Thrower),
			LandArea:        landArea,
			StartPosition:   startPos,
			EndPosition: models.AI_Vector{
				X: float64(e.Position.X),
				Y: float64(e.Position.Y),
				Z: float64(e.Position.Z),
			},
			BlindedPlayers:  blindedPlayers,
			EnemiesBlinded:  flashEvent.EnemiesBlinded,
			AlliesBlinded:   flashEvent.AlliesBlinded,
			ThrowViewVector: anglesToVector(e.Thrower.ViewDirectionX(), e.Thrower.ViewDirectionY()),
			DidBounce:       didBounce,
		}

		ctx.AI_GrenadeEvents = append(ctx.AI_GrenadeEvents, aiGrenadeEvent)
	})

	// HE grenade explode
	ctx.Parser.RegisterEventHandler(func(e events.HeExplode) {
		if e.Thrower == nil {
			return
		}

		heEvent := models.HEEvent{
			Tick:    ctx.Parser.GameState().IngameTick(),
			Round:   ctx.CurrentRound,
			Thrower: e.Thrower.Name,
			X:       float64(e.Position.X),
			Y:       float64(e.Position.Y),
			Z:       float64(e.Position.Z),
			// PHASE 1: Callouts
			ThrowerAreaName: getThrowerArea(ctx, e.Thrower),
		}

		ctx.MatchData.HEGrenades = append(ctx.MatchData.HEGrenades, heEvent)

		// ADD TO TIMELINE
		AddGrenadeToTimeline(ctx, ctx.Parser.GameState().IngameTick(), "he", &heEvent)

		landArea := findLandArea(ctx, r3.Vector{X: float64(e.Position.X), Y: float64(e.Position.Y), Z: float64(e.Position.Z)})

		// Find trajectory
		trajID, traj := findGrenadeTrajectory(ctx, e.Thrower.SteamID64, "HE", r3.Vector{X: float64(e.Position.X), Y: float64(e.Position.Y), Z: float64(e.Position.Z)})

		startPos := models.AI_Vector{X: 0, Y: 0, Z: 0}
		tickThrow := 0
		didBounce := false
		if traj != nil {
			tickThrow = traj.TickThrow
			if len(traj.Positions) > 0 {
				p := traj.Positions[0]
				startPos = models.AI_Vector{X: p.X, Y: p.Y, Z: p.Z}

				// Heuristic: If landArea is empty and throw was short, use thrower area
				if landArea == "" {
					dist := math.Sqrt(math.Pow(p.X-float64(e.Position.X), 2) + math.Pow(p.Y-float64(e.Position.Y), 2) + math.Pow(p.Z-float64(e.Position.Z), 2))
					if dist < 600 {
						landArea = getThrowerArea(ctx, e.Thrower)
					}
				}
			}
			didBounce = ctx.GrenadeBounces[trajID] > 0
		}

		// ADD TO AI MODEL
		aiGrenadeEvent := models.AI_GrenadeEvent{
			Round:           ctx.CurrentRound,
			Type:            "HE",
			Thrower:         e.Thrower.Name,
			TickThrow:       tickThrow,
			TickExplode:     ctx.Parser.GameState().IngameTick(),
			ThrowerAreaName: getThrowerArea(ctx, e.Thrower),
			ThrowerSide:     getTeamSide(e.Thrower),
			LandArea:        landArea,
			StartPosition:   startPos,
			EndPosition: models.AI_Vector{
				X: float64(e.Position.X),
				Y: float64(e.Position.Y),
				Z: float64(e.Position.Z),
			},
			ThrowViewVector: anglesToVector(e.Thrower.ViewDirectionX(), e.Thrower.ViewDirectionY()),
			DidBounce:       didBounce,
		}
		ctx.AI_GrenadeEvents = append(ctx.AI_GrenadeEvents, aiGrenadeEvent)

		// Track for damage attribution (cleared at FrameDone)
		ctx.PendingHEs[e.Thrower.SteamID64] = len(ctx.AI_GrenadeEvents) - 1
	})

	// Smoke started
	ctx.Parser.RegisterEventHandler(func(e events.SmokeStart) {
		if e.Thrower == nil {
			return
		}

		smokeEvent := models.SmokeEvent{
			Tick:    ctx.Parser.GameState().IngameTick(),
			Round:   ctx.CurrentRound,
			Thrower: e.Thrower.Name,
			X:       float64(e.Position.X),
			Y:       float64(e.Position.Y),
			Z:       float64(e.Position.Z),
			// PHASE 1: Callouts
			ThrowerAreaName: getThrowerArea(ctx, e.Thrower),
		}

		ctx.MatchData.Smokes = append(ctx.MatchData.Smokes, smokeEvent)

		// ADD TO TIMELINE
		AddGrenadeToTimeline(ctx, ctx.Parser.GameState().IngameTick(), "smoke", &smokeEvent)

		landArea := findLandArea(ctx, r3.Vector{X: float64(e.Position.X), Y: float64(e.Position.Y), Z: float64(e.Position.Z)})

		// Find trajectory
		trajID, traj := findGrenadeTrajectory(ctx, e.Thrower.SteamID64, "Smoke", r3.Vector{X: float64(e.Position.X), Y: float64(e.Position.Y), Z: float64(e.Position.Z)})

		startPos := models.AI_Vector{X: 0, Y: 0, Z: 0}
		tickThrow := 0
		didBounce := false
		if traj != nil {
			tickThrow = traj.TickThrow
			if len(traj.Positions) > 0 {
				p := traj.Positions[0]
				startPos = models.AI_Vector{X: p.X, Y: p.Y, Z: p.Z}

				// Heuristic: If landArea is empty and throw was short, use thrower area
				if landArea == "" {
					dist := math.Sqrt(math.Pow(p.X-float64(e.Position.X), 2) + math.Pow(p.Y-float64(e.Position.Y), 2) + math.Pow(p.Z-float64(e.Position.Z), 2))
					if dist < 600 {
						landArea = getThrowerArea(ctx, e.Thrower)
					}
				}
			}
			didBounce = ctx.GrenadeBounces[trajID] > 0
		}

		// ADD TO AI MODEL
		aiGrenadeEvent := models.AI_GrenadeEvent{
			Round:           ctx.CurrentRound,
			Type:            "Smoke",
			Thrower:         e.Thrower.Name,
			TickThrow:       tickThrow,
			TickExplode:     ctx.Parser.GameState().IngameTick(),
			ThrowerAreaName: getThrowerArea(ctx, e.Thrower),
			ThrowerSide:     getTeamSide(e.Thrower),
			LandArea:        landArea,
			StartPosition:   startPos,
			EndPosition: models.AI_Vector{
				X: float64(e.Position.X),
				Y: float64(e.Position.Y),
				Z: float64(e.Position.Z),
			},
			ThrowViewVector: anglesToVector(e.Thrower.ViewDirectionX(), e.Thrower.ViewDirectionY()),
			DidBounce:       didBounce,
		}
		ctx.AI_GrenadeEvents = append(ctx.AI_GrenadeEvents, aiGrenadeEvent)
	})

	// Molotov/Incendiary started
	ctx.Parser.RegisterEventHandler(func(e events.InfernoStart) {
		// Try to find thrower from recent grenade projectiles
		var thrower *common.Player
		var throwerAreaName string

		// InfernoStart doesn't have thrower, but we can try to infer from Inferno struct
		if e.Inferno.Thrower() != nil {
			thrower = e.Inferno.Thrower()
			throwerAreaName = getThrowerArea(ctx, thrower)
		}

		// Get position from inferno entity
		infernoPos := e.Inferno.Entity.Position()

		molotovEvent := models.MolotovEvent{
			Tick:  ctx.Parser.GameState().IngameTick(),
			Round: ctx.CurrentRound,
			X:     float64(infernoPos.X),
			Y:     float64(infernoPos.Y),
			Z:     float64(infernoPos.Z),
		}

		if thrower != nil {
			molotovEvent.Thrower = thrower.Name
			molotovEvent.ThrowerAreaName = throwerAreaName
		}

		ctx.MatchData.Molotovs = append(ctx.MatchData.Molotovs, molotovEvent)

		landArea := findLandArea(ctx, infernoPos)

		// ADD TO AI MODEL
		if thrower != nil {
			// Find trajectory
			trajID, traj := findGrenadeTrajectory(ctx, thrower.SteamID64, "Molotov", infernoPos)

			startPos := models.AI_Vector{X: 0, Y: 0, Z: 0}
			tickThrow := 0
			didBounce := false
			if traj != nil {
				tickThrow = traj.TickThrow
				if len(traj.Positions) > 0 {
					p := traj.Positions[0]
					startPos = models.AI_Vector{X: p.X, Y: p.Y, Z: p.Z}

					// Heuristic: If landArea is empty and throw was short, use thrower area
					if landArea == "" {
						dist := math.Sqrt(math.Pow(p.X-float64(infernoPos.X), 2) + math.Pow(p.Y-float64(infernoPos.Y), 2) + math.Pow(p.Z-float64(infernoPos.Z), 2))
						if dist < 600 {
							landArea = getThrowerArea(ctx, thrower)
						}
					}
				}
				didBounce = ctx.GrenadeBounces[trajID] > 0
			}

			aiGrenadeEvent := models.AI_GrenadeEvent{
				Round:           ctx.CurrentRound,
				Type:            "Molotov",
				Thrower:         thrower.Name,
				TickThrow:       tickThrow,
				TickExplode:     ctx.Parser.GameState().IngameTick(),
				ThrowerAreaName: throwerAreaName,
				ThrowerSide:     getTeamSide(thrower),
				LandArea:        landArea,
				StartPosition:   startPos,
				EndPosition: models.AI_Vector{
					X: float64(infernoPos.X),
					Y: float64(infernoPos.Y),
					Z: float64(infernoPos.Z),
				},
				ThrowViewVector: anglesToVector(thrower.ViewDirectionX(), thrower.ViewDirectionY()),
				DidBounce:       didBounce,
			}
			ctx.AI_GrenadeEvents = append(ctx.AI_GrenadeEvents, aiGrenadeEvent)
			ctx.ActiveInfernos[e.Inferno.Entity.ID()] = len(ctx.AI_GrenadeEvents) - 1
		}
	})

	// Smoke Start - Check for extinguished infernos
	ctx.Parser.RegisterEventHandler(func(e events.SmokeStart) {
		smokePos := e.Position
		// Check all active infernos
		for _, idx := range ctx.ActiveInfernos {
			// Get inferno start position
			infernoPos := r3.Vector{
				X: ctx.AI_GrenadeEvents[idx].StartPosition.X,
				Y: ctx.AI_GrenadeEvents[idx].StartPosition.Y,
				Z: ctx.AI_GrenadeEvents[idx].StartPosition.Z,
			}

			// Distance check
			// CS2 Smoke radius is ~144 units.
			// If smoke detonates close to inferno origin, it extinguishes it.
			// Using 180 units as a safe threshold for interaction.
			dist := smokePos.Distance(infernoPos)
			if dist < 180.0 {
				ctx.AI_GrenadeEvents[idx].Extinguished = true
			}
		}
	})

	// Molotov/Incendiary expired
	ctx.Parser.RegisterEventHandler(func(e events.InfernoExpired) {
		if idx, exists := ctx.ActiveInfernos[e.Inferno.Entity.ID()]; exists {
			// Calculate duration
			startTick := ctx.AI_GrenadeEvents[idx].TickExplode
			currentTick := ctx.Parser.GameState().IngameTick()
			tickRate := ctx.Parser.TickRate()
			if tickRate > 0 {
				duration := float64(currentTick-startTick) / tickRate
				ctx.AI_GrenadeEvents[idx].Duration = duration
				// Debug log for Molotov duration
				// fmt.Printf("[MOLOTOV DEBUG] ID: %d, Duration: %f, Inferno: %+v\n",
				// 	e.Inferno.Entity.ID(), duration, e.Inferno)

				// Try to get m_nFireLifetime from entity property (CS2 specific)
				// This reflects the intended duration (e.g. 7s for Molotov, ~5.5s for Incendiary)
				// Note: This does not account for being extinguished by smoke early,
				// but fixes the issue where entity lifetime is ~20s.
				// ONLY use this if NOT extinguished. If extinguished, the actual duration (calculated above) is correct.
				if !ctx.AI_GrenadeEvents[idx].Extinguished {
					if prop := e.Inferno.Entity.Property("m_nFireLifetime"); prop != nil {
						lifetime := prop.Value().Float()
						if lifetime > 0 && lifetime < 20 {
							ctx.AI_GrenadeEvents[idx].Duration = float64(lifetime)
						}
					}
				}
			}
			delete(ctx.ActiveInfernos, e.Inferno.Entity.ID())
		}
	})

	// Player Hurt (Grenade Damage)
	ctx.Parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker == nil {
			return
		}

		// HE Grenade Damage
		if e.Weapon != nil && e.Weapon.Type == common.EqHE {
			if idx, exists := ctx.PendingHEs[e.Attacker.SteamID64]; exists {
				ctx.AI_GrenadeEvents[idx].DamageDealt += e.HealthDamage
				isEnemy := e.Player.Team != e.Attacker.Team
				if isEnemy {
					ctx.AI_GrenadeEvents[idx].EnemiesDamaged++
				} else {
					ctx.AI_GrenadeEvents[idx].AlliesDamaged++
				}
				isKill := e.HealthDamage >= e.Player.Health()
				if isKill { // Kill
					ctx.AI_GrenadeEvents[idx].Kills++
				}

				// Add to DamagedPlayers
				teamStr := "Unknown"
				if e.Player.Team == common.TeamCounterTerrorists {
					teamStr = "CT"
				} else if e.Player.Team == common.TeamTerrorists {
					teamStr = "T"
				}

				// Check if player already exists in list
				found := false
				for i, p := range ctx.AI_GrenadeEvents[idx].DamagedPlayers {
					if p.Name == e.Player.Name {
						ctx.AI_GrenadeEvents[idx].DamagedPlayers[i].Damage += e.HealthDamage
						if isKill {
							ctx.AI_GrenadeEvents[idx].DamagedPlayers[i].IsKill = true
						}
						found = true
						break
					}
				}
				if !found {
					ctx.AI_GrenadeEvents[idx].DamagedPlayers = append(ctx.AI_GrenadeEvents[idx].DamagedPlayers, models.AI_DamagedPlayer{
						Name:    e.Player.Name,
						Damage:  e.HealthDamage,
						Team:    teamStr,
						IsEnemy: isEnemy,
						IsKill:  isKill,
					})
				}
			}
		}

		// Molotov/Incendiary Damage
		if e.Weapon != nil && (e.Weapon.Type == common.EqMolotov || e.Weapon.Type == common.EqIncendiary) {
			// Find active inferno for this attacker
			// Since we don't have direct link, we iterate active infernos
			for _, idx := range ctx.ActiveInfernos {
				if ctx.AI_GrenadeEvents[idx].Thrower == e.Attacker.Name {
					ctx.AI_GrenadeEvents[idx].DamageDealt += e.HealthDamage
					isEnemy := e.Player.Team != e.Attacker.Team
					if isEnemy {
						ctx.AI_GrenadeEvents[idx].EnemiesDamaged++
					} else {
						ctx.AI_GrenadeEvents[idx].AlliesDamaged++
					}
					isKill := e.HealthDamage >= e.Player.Health()
					if isKill { // Kill
						ctx.AI_GrenadeEvents[idx].Kills++
					}

					// Add to DamagedPlayers
					teamStr := "Unknown"
					if e.Player.Team == common.TeamCounterTerrorists {
						teamStr = "CT"
					} else if e.Player.Team == common.TeamTerrorists {
						teamStr = "T"
					}

					// Check if player already exists in list
					found := false
					for i, p := range ctx.AI_GrenadeEvents[idx].DamagedPlayers {
						if p.Name == e.Player.Name {
							ctx.AI_GrenadeEvents[idx].DamagedPlayers[i].Damage += e.HealthDamage
							if isKill {
								ctx.AI_GrenadeEvents[idx].DamagedPlayers[i].IsKill = true
							}
							found = true
							break
						}
					}
					if !found {
						ctx.AI_GrenadeEvents[idx].DamagedPlayers = append(ctx.AI_GrenadeEvents[idx].DamagedPlayers, models.AI_DamagedPlayer{
							Name:    e.Player.Name,
							Damage:  e.HealthDamage,
							Team:    teamStr,
							IsEnemy: isEnemy,
							IsKill:  isKill,
						})
					}

					// Assume one active inferno per player usually, or attribute to all (rare)
					// Break after first match to avoid double counting if multiple infernos (unlikely)
					break
				}
			}
		}
	})

	// --- TRAJECTORY TRACKING ---

	// 1. Start tracking on throw
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		if e.Projectile == nil || e.Projectile.Thrower == nil {
			return
		}

		// Get thrower area name
		throwerAreaName := ""
		if e.Projectile.Thrower != nil {
			throwerAreaName = getThrowerArea(ctx, e.Projectile.Thrower)
		}

		// Iniciar tracking
		ctx.ActiveGrenadeTrajectories[e.Projectile.Entity.ID()] = &models.GrenadeTrajectoryEvent{
			GrenadeType:     e.Projectile.WeaponInstance.Type.String(),
			Thrower:         e.Projectile.Thrower.Name,
			ThrowerID:       e.Projectile.Thrower.SteamID64,
			TickThrow:       ctx.Parser.GameState().IngameTick(),
			ThrowerAreaName: throwerAreaName,
			Positions:       []models.XYZ{}, // Se llenará frame a frame
		}
	})

	// 2. Update positions every tick (or every N ticks)
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		// Clear Pending HEs (damage is instant)
		for k := range ctx.PendingHEs {
			delete(ctx.PendingHEs, k)
		}

		// Solo si hay granadas activas
		if len(ctx.ActiveGrenadeTrajectories) == 0 {
			return
		}

		// Iterar sobre proyectiles activos en el juego
		for _, proj := range ctx.Parser.GameState().GrenadeProjectiles() {
			// Si lo estamos trackeando
			if traj, exists := ctx.ActiveGrenadeTrajectories[proj.Entity.ID()]; exists {
				pos := proj.Position()
				traj.Positions = append(traj.Positions, models.XYZ{
					X: float64(pos.X),
					Y: float64(pos.Y),
					Z: float64(pos.Z),
				})
			}
		}
	})

	// 3. Finish tracking on destroy
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
		if e.Projectile == nil {
			return
		}

		traj, exists := ctx.ActiveGrenadeTrajectories[e.Projectile.Entity.ID()]
		if !exists {
			return
		}

		// Guardar posición final
		pos := e.Projectile.Position()
		traj.LandPosition = models.XYZ{
			X: float64(pos.X),
			Y: float64(pos.Y),
			Z: float64(pos.Z),
		}

		// Get land area name using MapManager if available
		traj.LandAreaName = findLandArea(ctx, pos)

		// Agregar evento a timeline
		event := models.TimelineEvent{
			Type:              "grenade_trajectory",
			Tick:              ctx.Parser.GameState().IngameTick(),
			Round:             ctx.CurrentRound,
			GrenadeTrajectory: traj,
		}
		AddTimelineEvent(ctx, event)

		// Move to Completed instead of deleting immediately
		// This allows explosion events (which might fire slightly later) to find the trajectory
		ctx.CompletedGrenadeTrajectories[e.Projectile.Entity.ID()] = traj
		delete(ctx.ActiveGrenadeTrajectories, e.Projectile.Entity.ID())
		delete(ctx.GrenadeBounces, e.Projectile.Entity.ID())
	})
}

func anglesToVector(pitch, yaw float32) models.AI_Vector {
	// Convert to radians
	p := float64(pitch) * math.Pi / 180.0
	y := float64(yaw) * math.Pi / 180.0

	// CS coordinates: Z is up. Pitch is rotation around Y axis?
	// Usually:
	// x = cos(yaw) * cos(pitch)
	// y = sin(yaw) * cos(pitch)
	// z = -sin(pitch) // Pitch is usually negative looking up in Source engine?

	return models.AI_Vector{
		X: math.Cos(y) * math.Cos(p),
		Y: math.Sin(y) * math.Cos(p),
		Z: -math.Sin(p),
	}
}

// Helper to find the closest active grenade trajectory
func findGrenadeTrajectory(ctx *models.DemoContext, throwerID uint64, grenadeType string, pos r3.Vector) (int, *models.GrenadeTrajectoryEvent) {
	var bestID int
	var bestTraj *models.GrenadeTrajectoryEvent
	minDist := 1000.0 // Max distance threshold

	// Helper function to check a map of trajectories
	checkTrajectories := func(trajectories map[int]*models.GrenadeTrajectoryEvent) {
		for id, traj := range trajectories {
			if traj.ThrowerID != throwerID {
				continue
			}
			// Check type (approximate string match or exact)
			match := false
			if grenadeType == "Flashbang" && traj.GrenadeType == "Flashbang" {
				match = true
			}
			if grenadeType == "HE" && traj.GrenadeType == "HE Grenade" {
				match = true
			}
			if grenadeType == "Smoke" && traj.GrenadeType == "Smoke Grenade" {
				match = true
			}
			if (grenadeType == "Molotov" || grenadeType == "Incendiary") && (traj.GrenadeType == "Molotov" || traj.GrenadeType == "Incendiary Grenade") {
				match = true
			}

			if !match {
				continue
			}

			// Check distance to last known position
			if len(traj.Positions) > 0 {
				lastPos := traj.Positions[len(traj.Positions)-1]
				dist := math.Sqrt(math.Pow(lastPos.X-pos.X, 2) + math.Pow(lastPos.Y-pos.Y, 2) + math.Pow(lastPos.Z-pos.Z, 2))
				if dist < minDist {
					minDist = dist
					bestID = id
					bestTraj = traj
				}
			}
		}
	}

	// Check Active Trajectories
	checkTrajectories(ctx.ActiveGrenadeTrajectories)

	// Check Completed Trajectories (if not found or to find a better match)
	checkTrajectories(ctx.CompletedGrenadeTrajectories)

	return bestID, bestTraj
}

// Helper to find land area using closest player
func findLandArea(ctx *models.DemoContext, pos r3.Vector) string {
	if ctx.MapManager != nil {
		area := ctx.MapManager.GetCallout(pos)
		if area != "" {
			return area
		}
	}

	// Fallback: Find closest player
	minDist := 800.0 // Max radius (increased from 500)
	bestArea := ""

	for _, p := range ctx.Parser.GameState().Participants().All() {
		if !p.IsAlive() {
			continue
		}
		pPos := p.Position()
		dist := pPos.Distance(pos)
		if dist < minDist {
			minDist = dist
			bestArea = p.LastPlaceName()
		}
	}
	return bestArea
}

// Helper to find thrower side
func getTeamSide(player *common.Player) string {
	if player == nil {
		return ""
	}
	if player.Team == common.TeamCounterTerrorists {
		return "CT"
	}
	if player.Team == common.TeamTerrorists {
		return "T"
	}
	return ""
}

// Helper to find thrower area with MapManager priority
func getThrowerArea(ctx *models.DemoContext, thrower *common.Player) string {
	if thrower == nil {
		return ""
	}
	// Try MapManager first
	if ctx.MapManager != nil {
		pos := thrower.Position()
		area := ctx.MapManager.GetCallout(pos)
		if area != "" {
			return area
		}
	}
	// Fallback to parser's nav mesh
	return thrower.LastPlaceName()
}
