package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"
	"math"
	"sort"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

const (
	TrackingSampleRateHz = 2      // 2 times per second
	NearbyDistanceUnits  = 1000.0 // Units to consider a teammate "nearby"
)

// RegisterTrackingHandler registers the handler for AI tracking data
func RegisterTrackingHandler(ctx *models.DemoContext) {
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		// Only track during active rounds
		if !ctx.InRound {
			return
		}

		// Skip warmup/round 0
		if ctx.ActualRoundNumber == 0 {
			return
		}

		// CRITICAL: Skip freeze time (buy phase)
		if !ctx.FreezeTimeEnded {
			return
		}

		gameState := ctx.Parser.GameState()
		currentTick := gameState.IngameTick()
		tickRate := ctx.Parser.TickRate()

		// Calculate ticks per sample
		// e.g. 128 tickrate / 2 Hz = 64 ticks per sample
		ticksPerSample := int(tickRate / float64(TrackingSampleRateHz))
		if ticksPerSample == 0 {
			ticksPerSample = 32 // Fallback
		}

		// Check if it's time to sample
		if currentTick-ctx.LastTrackingTick < ticksPerSample {
			return
		}

		ctx.LastTrackingTick = currentTick
		clock := CaptureObjectiveClockSnapshot(ctx)
		objectiveState := ensureObjectiveTracker(ctx).Snapshot()
		phaseTimeRemaining := clock.PhaseTimeRemaining

		// Iterate over all participants
		for _, player := range gameState.Participants().Playing() {
			if player == nil {
				continue
			}

			// Calculate nearby teammates
			nearbyTeammates := 0
			if player.IsAlive() {
				for _, teammate := range gameState.Participants().TeamMembers(player.Team) {
					if teammate == nil || teammate.SteamID64 == player.SteamID64 || !teammate.IsAlive() {
						continue
					}

					dist := distance(player.Position().X, player.Position().Y, player.Position().Z,
						teammate.Position().X, teammate.Position().Y, teammate.Position().Z)

					if dist <= NearbyDistanceUnits {
						nearbyTeammates++
					}
				}
			}

			// Get Area Name
			areaName := player.LastPlaceName()
			if ctx.MapManager != nil {
				if callout := ctx.MapManager.GetCallout(player.Position()); callout != "" {
					areaName = callout
				}
			}

			motion := ctx.PlayerMotion.ObservePlayer(
				player,
				ctx.ActualRoundNumber,
				currentTick,
				tickRate,
			)
			weaponState := observeActiveWeapon(ctx, player, currentTick)
			hasObjectiveC4, isPlanting, isDefusing, nativeRoleDisagreement := reconcileObjectiveRole(
				player.SteamID64,
				player.IsAlive(),
				hasC4(player),
				player.IsPlanting,
				player.IsDefusing,
				objectiveState,
			)
			if nativeRoleDisagreement {
				ctx.ObjectiveNativeRoleDisagreements++
			}
			event := models.AI_TrackingEvent{
				Tick:          currentTick,
				PlayerSteamID: player.SteamID64,
				Team:          getTeamString(player.Team),
				Position: models.AI_Vector{
					X: player.Position().X,
					Y: player.Position().Y,
					Z: player.Position().Z,
				},
				AreaName:       areaName,
				ViewAngleYaw:   player.ViewDirectionX(), // Yaw = horizontal rotation
				ViewAnglePitch: player.ViewDirectionY(), // Pitch = vertical rotation
				VelocityVector: models.AI_Vector{
					X: motion.Vector.X,
					Y: motion.Vector.Y,
					Z: motion.Vector.Z,
				},
				VelocityAvailable:        motion.Available,
				VelocitySource:           string(motion.Source),
				VelocityMeasurementTicks: motion.IntervalTicks,
				IsWalking:                player.IsWalking(),
				IsDucking:                player.IsDucking(),
				ActiveWeapon:             weaponState.CurrentWeapon,
				ActiveWeaponStatus:       weaponState.Status,
				HasC4:                    hasObjectiveC4,
				HasDefuseKit:             player.HasDefuseKit(),
				IsPlanting:               isPlanting,
				IsDefusing:               isDefusing,
				Health:                   player.Health(),
				Armor:                    player.Armor(),
				NearbyTeammates:          nearbyTeammates,
				IsAlive:                  player.IsAlive(),
				RoundTimeRemaining:       clock.PhaseTimeRemaining,
				ObjectivePhase:           string(clock.Phase),
				PhaseTimeRemaining:       &phaseTimeRemaining,
				RoundClockRemaining:      clock.RoundClockRemaining,
				BombTimeRemaining:        clock.BombTimeRemaining,
			}
			if weaponState.CurrentWeapon == nil && weaponState.LastObservation != nil {
				lastWeapon := weaponState.LastObservation.Weapon
				lastTick := weaponState.LastObservation.Tick
				event.LastObservedActiveWeapon = &lastWeapon
				event.LastObservedActiveWeaponTick = &lastTick
			}

			// Store with round number for later grouping
			ctx.AI_TrackingEventsWithRound = append(ctx.AI_TrackingEventsWithRound, models.AI_TrackingEventWithRound{
				Round: ctx.ActualRoundNumber,
				Event: event,
			})
		}
	})
}

func reconcileObjectiveRole(
	playerID uint64,
	playerAlive bool,
	nativeHasC4, nativePlanting, nativeDefusing bool,
	snapshot objective.Snapshot,
) (hasC4, isPlanting, isDefusing, nativeDisagreement bool) {
	validPlayer := playerID != 0 && playerAlive
	hasC4 = validPlayer && nativeHasC4
	isPlanting = validPlayer && snapshot.Phase == objective.PhasePlanting && snapshot.PlantingPlayer.SteamID == playerID
	isDefusing = validPlayer && snapshot.Phase == objective.PhaseDefusing && snapshot.Defuser.SteamID == playerID

	if validPlayer {
		switch snapshot.Phase {
		case objective.PhasePlanting:
			hasC4 = isPlanting
		case objective.PhasePlanted, objective.PhaseDefusing, objective.PhaseResolved:
			hasC4 = false
		case objective.PhasePreplant:
			switch snapshot.State {
			case objective.StateDropped:
				hasC4 = false
			case objective.StateCarried:
				hasC4 = snapshot.Carrier.SteamID == playerID
			}
		}
	}
	nativeDisagreement = nativeHasC4 != hasC4 || nativePlanting != isPlanting || nativeDefusing != isDefusing
	return hasC4, isPlanting, isDefusing, nativeDisagreement
}

func distance(x1, y1, z1, x2, y2, z2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2) + math.Pow(z2-z1, 2))
}

func hasC4(player *common.Player) bool {
	for _, w := range player.Weapons() {
		if w.Type == common.EqBomb {
			return true
		}
	}
	return false
}

func getPlayerWeapons(player *common.Player) []string {
	entries := make([]playerWeaponEntry, 0, len(player.Weapons()))
	for _, weapon := range player.Weapons() {
		if weapon == nil {
			continue
		}

		name := weapon.String()
		if name == "" {
			continue
		}

		entries = append(entries, playerWeaponEntry{
			name:     name,
			priority: weaponPriority(weapon),
			count:    weaponInventoryCount(weapon),
		})
	}

	return flattenPlayerWeapons(entries)
}

type playerWeaponEntry struct {
	name     string
	priority int
	count    int
}

func weaponPriority(weapon *common.Equipment) int {
	switch weapon.Class() {
	case common.EqClassRifle, common.EqClassSMG, common.EqClassHeavy:
		return 0
	case common.EqClassPistols:
		return 1
	case common.EqClassGrenade:
		return 2
	default:
		if weapon.Type == common.EqKnife {
			return 4
		}
		return 3
	}
}

func weaponInventoryCount(weapon *common.Equipment) int {
	if weapon.Class() != common.EqClassGrenade {
		return 1
	}

	count := weapon.AmmoInMagazine() + weapon.AmmoReserve()
	if count < 1 {
		return 1
	}
	return count
}

func flattenPlayerWeapons(entries []playerWeaponEntry) []string {
	counts := make(map[string]int, len(entries))
	priorities := make(map[string]int, len(entries))
	for _, entry := range entries {
		if entry.name == "" {
			continue
		}
		if entry.count < 1 {
			entry.count = 1
		}
		if entry.count > counts[entry.name] {
			counts[entry.name] = entry.count
		}
		if previous, ok := priorities[entry.name]; !ok || entry.priority < previous {
			priorities[entry.name] = entry.priority
		}
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Slice(names, func(left, right int) bool {
		leftPriority := priorities[names[left]]
		rightPriority := priorities[names[right]]
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return names[left] < names[right]
	})

	weapons := make([]string, 0, len(entries))
	for _, name := range names {
		for count := 0; count < counts[name]; count++ {
			weapons = append(weapons, name)
		}
	}
	return weapons
}

func getTeamString(team common.Team) string {
	if team == common.TeamCounterTerrorists {
		return "CT"
	} else if team == common.TeamTerrorists {
		return "T"
	}
	return ""
}
