package handlers

import (
	"cs2-demo-service/models"
	"fmt"
	"math"
	"sort"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// ============================================================================
// DUEL CONSOLIDATION LOGIC
// Groups multiple damage/kill events into single consolidated duels
// ============================================================================

// DuelTimeoutTicks is the maximum gap between events to consider them part of the same duel
// 10 seconds @ 64 tick = 640 ticks
const DuelTimeoutTicks = 640

// GrenadeGroupingTicks is the window to group grenade hits (grenade can tick damage over time)
const GrenadeGroupingTicks = 30

// CollateralGroupingTicks is the window to group collateral hits (same bullet, very close ticks)
const CollateralGroupingTicks = 5

// isGrenade checks if a weapon is a grenade type (no aiming required)
func isGrenade(weapon string) bool {
	return weapon == "HE Grenade" || weapon == "Incendiary Grenade" ||
		weapon == "Molotov" || weapon == "Flashbang" || weapon == "Smoke Grenade"
}

// isFireGrenade checks if a weapon is an incendiary/molotov (tick damage over time)
func isFireGrenade(weapon string) bool {
	return weapon == "Incendiary Grenade" || weapon == "Molotov"
}

// getEngagementType returns "peek" or "hold" based on velocity threshold
// Peek: actively moving (>100 u/s), Hold: stationary or jiggling (≤100 u/s)
func getEngagementType(velocity float64) string {
	const peekVelocityThreshold = 100.0
	if velocity > peekVelocityThreshold {
		return "peek"
	}
	return "hold"
}

// ConsolidateDuels groups raw combat events into consolidated duels
// Handles two cases:
// 1. Grenades: Group by attacker + weapon (multi-victim possible)
// 2. Normal weapons: Group by attacker-victim PAIR (traditional duel)
func ConsolidateDuels(ctx *models.DemoContext) {
	if len(ctx.RawCombatEvents) == 0 {
		return
	}

	// Sort events by tick
	sort.Slice(ctx.RawCombatEvents, func(i, j int) bool {
		return ctx.RawCombatEvents[i].Tick < ctx.RawCombatEvents[j].Tick
	})

	processed := make(map[int]bool)

	// PASS 1: Process GRENADE events (group by attacker + weapon, multi-victim)
	type grenadeGroup struct {
		AttackerID uint64
		Weapon     string
		Events     []models.RawCombatEvent
	}
	var grenadeGroups []grenadeGroup

	for i, event := range ctx.RawCombatEvents {
		if processed[i] || !isGrenade(event.Weapon) {
			continue
		}

		// Determine grouping window for grenades
		groupingWindow := GrenadeGroupingTicks
		if isFireGrenade(event.Weapon) {
			groupingWindow = 500 // ~7.8 seconds for incendiary/molotov
		}

		group := grenadeGroup{
			AttackerID: event.AttackerSteamID,
			Weapon:     event.Weapon,
			Events:     []models.RawCombatEvent{event},
		}
		processed[i] = true
		lastTick := event.Tick

		for j := i + 1; j < len(ctx.RawCombatEvents); j++ {
			if processed[j] {
				continue
			}
			other := ctx.RawCombatEvents[j]

			// Group same attacker + same grenade weapon using sliding window
			if other.AttackerSteamID == event.AttackerSteamID &&
				other.Weapon == event.Weapon &&
				(other.Tick-lastTick) <= groupingWindow {
				group.Events = append(group.Events, other)
				processed[j] = true
				lastTick = other.Tick
			}
		}

		grenadeGroups = append(grenadeGroups, group)
	}

	// Build grenade duels
	for _, group := range grenadeGroups {
		duel := buildMultiVictimDuel(ctx, group.Events, true)
		if duel != nil {
			ctx.AI_Duels = append(ctx.AI_Duels, *duel)
		}
	}

	// PASS 2: Process NORMAL WEAPON events (group by player PAIR - bidirectional)
	// All damage between two players forms ONE duel, winner determined by who got the kill
	duelGroups := make(map[string][]models.RawCombatEvent)

	for i, event := range ctx.RawCombatEvents {
		if processed[i] {
			continue // Skip already processed (grenades)
		}
		// Use canonical key: both directions grouped together
		key := makeDuelKey(event.AttackerSteamID, event.VictimSteamID)
		duelGroups[key] = append(duelGroups[key], event)
	}

	// Split each pair's events by timeout and build duels
	for _, events := range duelGroups {
		duels := splitByTimeout(events, DuelTimeoutTicks)
		for _, duelEvents := range duels {
			duel := buildMultiVictimDuel(ctx, duelEvents, false)
			if duel != nil {
				ctx.AI_Duels = append(ctx.AI_Duels, *duel)
			}
		}
	}

	// Clear the raw events buffer
	ctx.RawCombatEvents = ctx.RawCombatEvents[:0]
}

// makeDuelKey creates a canonical key for a pair of players
func makeDuelKey(id1, id2 uint64) string {
	if id1 < id2 {
		return fmt.Sprintf("%d_%d", id1, id2)
	}
	return fmt.Sprintf("%d_%d", id2, id1)
}

// splitByTimeout splits a sequence of events into multiple duels based on time gaps
func splitByTimeout(events []models.RawCombatEvent, timeoutTicks int) [][]models.RawCombatEvent {
	if len(events) == 0 {
		return nil
	}

	var result [][]models.RawCombatEvent
	currentDuel := []models.RawCombatEvent{events[0]}

	for i := 1; i < len(events); i++ {
		gap := events[i].Tick - events[i-1].Tick
		if gap > timeoutTicks {
			// Start a new duel
			result = append(result, currentDuel)
			currentDuel = []models.RawCombatEvent{events[i]}
		} else {
			currentDuel = append(currentDuel, events[i])
		}
	}

	// Don't forget the last duel
	result = append(result, currentDuel)
	return result
}

// buildMultiVictimDuel creates a consolidated AI_Duel from a group of events
// Handles:
// - Grenades: one attacker, multiple victims
// - Normal duels: two players fighting, winner determined by kill or total damage
func buildMultiVictimDuel(ctx *models.DemoContext, events []models.RawCombatEvent, isGrenadeEvent bool) *models.AI_Duel {
	if len(events) == 0 {
		return nil
	}

	firstEvent := events[0]

	// For normal weapons (not grenades), we need to handle bidirectional 1v1 duels
	// Collect all unique participants (both as attackers and victims)
	if !isGrenadeEvent {
		// Get all unique player IDs involved
		participantIDs := make(map[uint64]bool)
		for _, e := range events {
			participantIDs[e.AttackerSteamID] = true
			participantIDs[e.VictimSteamID] = true
		}

		// If exactly 2 participants, this is a 1v1 duel - need to determine winner/loser
		if len(participantIDs) == 2 {
			var player1ID, player2ID uint64
			i := 0
			for id := range participantIDs {
				if i == 0 {
					player1ID = id
				} else {
					player2ID = id
				}
				i++
			}

			// Calculate damage dealt BY each player (when they are attacker)
			p1DamageDealt := 0
			p2DamageDealt := 0
			p1GotKill := false
			p2GotKill := false

			for _, e := range events {
				if e.AttackerSteamID == player1ID {
					p1DamageDealt += e.Damage
					if e.IsKill {
						p1GotKill = true
					}
				} else if e.AttackerSteamID == player2ID {
					p2DamageDealt += e.Damage
					if e.IsKill {
						p2GotKill = true
					}
				}
			}

			// Determine winner: whoever got the kill, or if no kill, whoever dealt more damage
			var winnerID, loserID uint64
			if p1GotKill {
				winnerID = player1ID
				loserID = player2ID
			} else if p2GotKill {
				winnerID = player2ID
				loserID = player1ID
			} else if p1DamageDealt >= p2DamageDealt {
				winnerID = player1ID
				loserID = player2ID
			} else {
				winnerID = player2ID
				loserID = player1ID
			}

			// Aggregate stats for winner (as attacker) and loser (as victim)
			winnerStats := aggregateAttackerStats(events, winnerID)
			// Also get damage received by winner
			winnerAsVictim := aggregateVictimStats(events, winnerID)
			if winnerAsVictim.TotalDamage > 0 {
				winnerStats.HealthAfter = winnerAsVictim.HealthAfter
				winnerStats.ArmorAfter = winnerAsVictim.ArmorAfter
			}

			loserStats := aggregateVictimStats(events, loserID)
			// Also get damage dealt by loser
			loserAsAttacker := aggregateAttackerStats(events, loserID)

			// Determine outcome
			outcome := "damage"
			if p1GotKill || p2GotKill {
				outcome = "kill"
			}

			// Build exchanges array with ALL damage (both directions)
			var exchanges []models.AI_DuelExchange
			if len(events) > 1 {
				// Track last FirstSeenTick per attacker to detect new visibility windows
				// TTR/TTFD should only appear on the first hit of each visibility window
				// Grenades never have TTR/TTFD (they don't require aiming at the enemy)
				attackerLastSeenTick := make(map[string]int)
				for _, e := range events {
					ttr := 0.0
					ttfd := 0.0

					// Only set TTR/TTFD if:
					// 1. NOT a grenade (grenades don't require aiming)
					// 2. This is a NEW visibility window for this attacker
					if !isGrenade(e.Weapon) && e.FirstSeenTick > 0 {
						lastTick, hasPrevious := attackerLastSeenTick[e.AttackerName]
						if !hasPrevious || e.FirstSeenTick != lastTick {
							// This is either the first hit or a new visibility window (re-peek)
							ttr = e.TimeToReaction
							ttfd = e.TimeToDamage
						}
						attackerLastSeenTick[e.AttackerName] = e.FirstSeenTick
					}

					exchanges = append(exchanges, models.AI_DuelExchange{
						Tick:              e.Tick,
						Attacker:          e.AttackerName,
						Weapon:            e.Weapon,
						Damage:            e.Damage,
						Hitgroup:          e.Hitgroup,
						IsKill:            e.IsKill,
						TimeToReaction:    ttr,
						TimeToFirstDamage: ttfd,
					})
				}
			}

			// Get context from kill event or last event
			contextEvent := events[len(events)-1]
			for i := range events {
				if events[i].IsKill {
					contextEvent = events[i]
					break
				}
			}

			// Duration
			tickDiff := contextEvent.Tick - firstEvent.Tick
			durationMs := float64(tickDiff) * (1000.0 / 64.0)

			// Generate duel ID
			ctx.CombatEventCounter++
			duelID := fmt.Sprintf("duel_%d", ctx.CombatEventCounter)

			// Calculate average metrics per participant from exchanges
			// Winner avg
			var winnerTTRSum, winnerTTFDSum float64
			var winnerTTRCount, winnerTTFDCount int
			// Loser avg
			var loserTTRSum, loserTTFDSum float64
			var loserTTRCount, loserTTFDCount int
			for _, ex := range exchanges {
				if ex.Attacker == winnerStats.Name {
					if ex.TimeToReaction > 0 {
						winnerTTRSum += ex.TimeToReaction
						winnerTTRCount++
					}
					if ex.TimeToFirstDamage > 0 {
						winnerTTFDSum += ex.TimeToFirstDamage
						winnerTTFDCount++
					}
				} else if ex.Attacker == loserStats.Name {
					if ex.TimeToReaction > 0 {
						loserTTRSum += ex.TimeToReaction
						loserTTRCount++
					}
					if ex.TimeToFirstDamage > 0 {
						loserTTFDSum += ex.TimeToFirstDamage
						loserTTFDCount++
					}
				}
			}
			// Only calculate AVG if there are multiple exchanges (>1) for that player
			winnerAvgTTR := 0.0
			if winnerTTRCount > 1 {
				winnerAvgTTR = winnerTTRSum / float64(winnerTTRCount)
			}
			winnerAvgTTFD := 0.0
			if winnerTTFDCount > 1 {
				winnerAvgTTFD = winnerTTFDSum / float64(winnerTTFDCount)
			}
			loserAvgTTR := 0.0
			if loserTTRCount > 1 {
				loserAvgTTR = loserTTRSum / float64(loserTTRCount)
			}
			loserAvgTTFD := 0.0
			if loserTTFDCount > 1 {
				loserAvgTTFD = loserTTFDSum / float64(loserTTFDCount)
			}

			return &models.AI_Duel{
				DuelID:      duelID,
				Type:        "duel",
				Outcome:     outcome,
				VictimCount: 1,
				Round:       firstEvent.Round,
				TickStart:   firstEvent.Tick,
				TickEnd:     contextEvent.Tick,
				DurationMs:  durationMs,
				Attacker: models.AI_DuelParticipant{
					SteamID:               winnerStats.SteamID,
					Name:                  winnerStats.Name,
					Team:                  winnerStats.Team,
					MapArea:               winnerStats.MapArea,
					Weapon:                winnerStats.Weapon,
					TotalDamageDealt:      winnerStats.TotalDamage,
					Hits:                  winnerStats.Hits,
					Headshots:             winnerStats.Headshots,
					ShotsFired:            winnerStats.ShotsFired,
					HealthBefore:          winnerStats.HealthBefore,
					HealthAfter:           winnerStats.HealthAfter,
					ArmorBefore:           winnerStats.ArmorBefore,
					ArmorAfter:            winnerStats.ArmorAfter,
					EquipmentValue:        winnerStats.EquipmentValue,
					AmmoInMagazine:        winnerStats.AmmoInMagazine,
					AmmoReserve:           winnerStats.AmmoReserve,
					InitialCrosshairError: winnerStats.InitialCrosshairError,
					PitchError:            winnerStats.PitchError,
					YawError:              winnerStats.YawError,
					TimeToReaction:        winnerStats.TimeToReaction,
					TimeToFirstDamage:     winnerStats.TimeToFirstDamage,
					AvgTimeToReaction:     winnerAvgTTR,
					AvgTimeToFirstDamage:  winnerAvgTTFD,
					Velocity:              winnerStats.Velocity,
					EngagementType:        getEngagementType(winnerStats.Velocity),
					IsBlind:               winnerStats.IsBlind,
					IsDucking:             winnerStats.IsDucking,
					Position:              &winnerStats.Position, // [NEW]
				},
				Victims: []models.AI_DuelParticipant{
					{
						SteamID: loserStats.SteamID,
						Name:    loserStats.Name,
						Team:    loserStats.Team,
						MapArea: loserStats.MapArea,
						// FIX: Use victim's weapon from when they were damaged, with fallbacks
						Weapon: func() string {
							// Priority 1: loserAsAttacker.Weapon (if they attacked back)
							if loserAsAttacker.Weapon != "" {
								return loserAsAttacker.Weapon
							}
							// Priority 2: loserStats.Weapon (from when they were victim)
							if loserStats.Weapon != "" {
								return loserStats.Weapon
							}
							// Priority 3: tracked weapon from context
							if w, exists := ctx.LastActiveWeapon[loserStats.SteamID]; exists && w != "" {
								return w
							}
							return ""
						}(),
						TotalDamageDealt:      loserAsAttacker.TotalDamage, // Damage the loser dealt back
						Hits:                  loserAsAttacker.Hits,
						Headshots:             loserAsAttacker.Headshots,
						ShotsFired:            loserAsAttacker.ShotsFired,
						HealthBefore:          loserStats.HealthBefore,
						HealthAfter:           loserStats.HealthAfter,
						ArmorBefore:           loserStats.ArmorBefore,
						ArmorAfter:            loserStats.ArmorAfter,
						EquipmentValue:        loserStats.EquipmentValue,
						InitialCrosshairError: loserAsAttacker.InitialCrosshairError,
						PitchError:            loserAsAttacker.PitchError,
						YawError:              loserAsAttacker.YawError,
						TimeToReaction:        loserAsAttacker.TimeToReaction,
						TimeToFirstDamage:     loserAsAttacker.TimeToFirstDamage,
						AvgTimeToReaction:     loserAvgTTR,
						AvgTimeToFirstDamage:  loserAvgTTFD,
						Velocity:              loserStats.Velocity,
						EngagementType:        getEngagementType(loserStats.Velocity),
						IsBlind:               loserStats.IsBlind,
						IsDucking:             loserStats.IsDucking,
						Position:              &loserStats.Position, // [NEW]
					},
				},
				Exchanges: exchanges,
				Context: models.AI_DuelContext{
					Distance:           contextEvent.Distance,
					HeightDiff:         contextEvent.HeightDiff,
					IsTrade:            contextEvent.IsTrade,
					ThroughSmoke:       anyThroughSmoke(events),
					IsWallbang:         anyWallbang(events),
					PenetratedObjects:  maxPenetratedObjects(events),
					NoScope:            getNoScopePtr(winnerStats.Weapon, contextEvent.NoScope),
					ZoomLevel:          getZoomLevelPtr(winnerStats.Weapon, contextEvent.ZoomLevel),
					BombPlanted:        contextEvent.BombPlanted,
					AliveCT:            contextEvent.AliveCT,
					AliveT:             contextEvent.AliveT,
					IsOpeningKill:      contextEvent.IsOpeningKill,
					RoundTimeRemaining: contextEvent.RoundTimeRemaining,
				},
			}
		}
	}

	// Original logic for grenades and other cases (one clear attacker, multiple victims)
	attackerID := firstEvent.AttackerSteamID

	// Collect unique victims
	victimMap := make(map[uint64]*playerDuelStats)
	for _, e := range events {
		if e.VictimSteamID == 0 {
			continue
		}
		if _, exists := victimMap[e.VictimSteamID]; !exists {
			victimMap[e.VictimSteamID] = &playerDuelStats{
				SteamID: e.VictimSteamID,
			}
		}
	}

	// Aggregate stats for each victim
	var victims []models.AI_DuelParticipant
	totalDamageDealt := 0
	killCount := 0
	var lastKillEvent *models.RawCombatEvent

	for victimID := range victimMap {
		stats := aggregateVictimStats(events, victimID)
		if stats.HealthBefore == 0 && stats.TotalDamage == 0 {
			continue // Skip if no actual damage
		}

		// Calculate damage received by this victim (sum of all damage events where they are victim)
		damageReceived := 0
		hitsReceived := 0
		for i := range events {
			if events[i].VictimSteamID == victimID {
				damageReceived += events[i].Damage
				hitsReceived++
			}
		}

		// Check if this victim was killed
		wasKilled := false
		for i := range events {
			if events[i].VictimSteamID == victimID && events[i].IsKill {
				wasKilled = true
				killCount++
				lastKillEvent = &events[i]
				break
			}
		}

		victim := models.AI_DuelParticipant{
			SteamID:          stats.SteamID,
			Name:             stats.Name,
			Team:             stats.Team,
			MapArea:          stats.MapArea,
			Weapon:           stats.Weapon,
			TotalDamageDealt: stats.TotalDamage,
			DamageReceived:   damageReceived, // Damage this victim received from attacker
			Hits:             hitsReceived,   // Hits received, not dealt
			ShotsFired:       0,              // Victims don't fire in this context
			HealthBefore:     stats.HealthBefore,
			HealthAfter:      stats.HealthAfter,
			ArmorBefore:      stats.ArmorBefore,
			ArmorAfter:       stats.ArmorAfter,
			EquipmentValue:   stats.EquipmentValue,
			Velocity:         stats.Velocity,
			EngagementType:   getEngagementType(stats.Velocity),
			IsBlind:          stats.IsBlind,
			IsDucking:        stats.IsDucking,
			Position:         &stats.Position, // [NEW]
		}

		// FIX: If weapon is still empty, use tracked weapon from context
		if victim.Weapon == "" {
			if trackedWeapon, exists := ctx.LastActiveWeapon[victimID]; exists && trackedWeapon != "" {
				victim.Weapon = trackedWeapon
			}
		}

		// Add headshots if victim was headshotted
		if stats.Headshots > 0 {
			victim.Headshots = stats.Headshots
		}

		victims = append(victims, victim)
		totalDamageDealt += stats.TotalDamage

		// Mark if killed
		if wasKilled {
			victim.HealthAfter = 0
		}
	}

	if len(victims) == 0 {
		return nil
	}

	// Get attacker stats
	attackerStats := aggregateAttackerStats(events, attackerID)

	// Determine event type and outcome
	eventType := "duel"
	if isGrenadeEvent {
		eventType = "grenade" // Always grenade if it's a grenade (1 or more victims)
	} else if len(victims) > 1 {
		eventType = "collateral" // Collateral only for normal weapons with multiple victims
	}

	outcome := "damage"
	if killCount > 0 {
		if killCount > 1 {
			outcome = "multi_kill"
		} else {
			outcome = "kill"
		}
	}

	// Build exchanges array (only for non-grenade events)
	// Grenades don't have meaningful exchanges - damage info is in victims[].damage_received
	var exchanges []models.AI_DuelExchange
	if !isGrenadeEvent && len(events) > 1 {
		// Track last FirstSeenTick per attacker to detect new visibility windows
		// TTR/TTFD should only appear on the first hit of each visibility window
		attackerLastSeenTick := make(map[string]int)
		for _, e := range events {
			ttr := 0.0
			ttfd := 0.0

			// Only set TTR/TTFD if this is a NEW visibility window for this attacker
			if e.FirstSeenTick > 0 {
				lastTick, hasPrevious := attackerLastSeenTick[e.AttackerName]
				if !hasPrevious || e.FirstSeenTick != lastTick {
					// This is either the first hit or a new visibility window (re-peek)
					ttr = e.TimeToReaction
					ttfd = e.TimeToDamage
				}
				attackerLastSeenTick[e.AttackerName] = e.FirstSeenTick
			}

			exchanges = append(exchanges, models.AI_DuelExchange{
				Tick:              e.Tick,
				Attacker:          e.AttackerName,
				Weapon:            e.Weapon,
				Damage:            e.Damage,
				Hitgroup:          e.Hitgroup,
				IsKill:            e.IsKill,
				TimeToReaction:    ttr,
				TimeToFirstDamage: ttfd,
			})
		}
	}

	// Get context from last event
	contextEvent := events[len(events)-1]
	if lastKillEvent != nil {
		contextEvent = *lastKillEvent
	}

	// Duration in ms
	tickDiff := contextEvent.Tick - firstEvent.Tick
	durationMs := float64(tickDiff) * (1000.0 / 64.0)

	// Generate duel ID (will be reassigned later in exporter)
	ctx.CombatEventCounter++
	duelID := fmt.Sprintf("duel_%d", ctx.CombatEventCounter)

	// Calculate average metrics from exchanges (only if >1 exchanges)
	var ttrSum, ttfdSum float64
	var ttrCount, ttfdCount int
	for _, ex := range exchanges {
		if ex.TimeToReaction > 0 {
			ttrSum += ex.TimeToReaction
			ttrCount++
		}
		if ex.TimeToFirstDamage > 0 {
			ttfdSum += ex.TimeToFirstDamage
			ttfdCount++
		}
	}
	avgTTR := 0.0
	if ttrCount > 1 {
		avgTTR = ttrSum / float64(ttrCount)
	}
	avgTTFD := 0.0
	if ttfdCount > 1 {
		avgTTFD = ttfdSum / float64(ttfdCount)
	}

	duel := &models.AI_Duel{
		DuelID:      duelID,
		Type:        eventType,
		Outcome:     outcome,
		VictimCount: len(victims),
		Round:       firstEvent.Round,
		TickStart:   firstEvent.Tick,
		TickEnd:     contextEvent.Tick,
		DurationMs:  durationMs,
		Attacker: models.AI_DuelParticipant{
			SteamID:               attackerStats.SteamID,
			Name:                  attackerStats.Name,
			Team:                  attackerStats.Team,
			MapArea:               attackerStats.MapArea,
			Weapon:                attackerStats.Weapon,
			TotalDamageDealt:      totalDamageDealt,
			Hits:                  attackerStats.Hits,
			Headshots:             attackerStats.Headshots,
			ShotsFired:            attackerStats.ShotsFired,
			HealthBefore:          attackerStats.HealthBefore,
			HealthAfter:           attackerStats.HealthAfter,
			ArmorBefore:           attackerStats.ArmorBefore,
			ArmorAfter:            attackerStats.ArmorAfter,
			EquipmentValue:        attackerStats.EquipmentValue,
			AmmoInMagazine:        attackerStats.AmmoInMagazine,
			AmmoReserve:           attackerStats.AmmoReserve,
			InitialCrosshairError: attackerStats.InitialCrosshairError,
			PitchError:            attackerStats.PitchError,
			YawError:              attackerStats.YawError,
			TimeToReaction:        attackerStats.TimeToReaction,
			TimeToFirstDamage:     attackerStats.TimeToFirstDamage,
			AvgTimeToReaction:     avgTTR,
			AvgTimeToFirstDamage:  avgTTFD,
			Velocity:              attackerStats.Velocity,
			EngagementType:        getEngagementType(attackerStats.Velocity),
			IsBlind:               attackerStats.IsBlind,
			IsDucking:             attackerStats.IsDucking,
		},
		Victims:   victims,
		Exchanges: exchanges,
		Context: models.AI_DuelContext{
			Distance:           contextEvent.Distance,
			HeightDiff:         contextEvent.HeightDiff,
			IsTrade:            contextEvent.IsTrade,
			ThroughSmoke:       anyThroughSmoke(events),
			IsWallbang:         anyWallbang(events),
			PenetratedObjects:  maxPenetratedObjects(events),
			NoScope:            getNoScopePtr(attackerStats.Weapon, contextEvent.NoScope),
			ZoomLevel:          getZoomLevelPtr(attackerStats.Weapon, contextEvent.ZoomLevel),
			BombPlanted:        contextEvent.BombPlanted,
			AliveCT:            contextEvent.AliveCT,
			AliveT:             contextEvent.AliveT,
			IsOpeningKill:      contextEvent.IsOpeningKill,
			RoundTimeRemaining: contextEvent.RoundTimeRemaining,
		},
	}

	return duel
}

// aggregateAttackerStats collects attacker stats from events
func aggregateAttackerStats(events []models.RawCombatEvent, attackerID uint64) *playerDuelStats {
	stats := &playerDuelStats{SteamID: attackerID}
	first := true

	for _, e := range events {
		if e.AttackerSteamID == attackerID {
			stats.TotalDamage += e.Damage
			stats.Hits++
			if e.Hitgroup == "head" {
				stats.Headshots++
			}

			if first {
				stats.Name = e.AttackerName
				stats.Team = e.AttackerTeam
				stats.MapArea = e.AttackerMapArea
				stats.Weapon = e.Weapon
				stats.HealthBefore = e.AttackerHealth
				stats.ArmorBefore = e.AttackerArmor
				stats.EquipmentValue = e.AttackerEquipmentValue
				// Only set weapon/aiming-related metrics for non-grenades
				if !isGrenade(e.Weapon) {
					stats.AmmoInMagazine = e.AttackerAmmoInMagazine
					stats.AmmoReserve = e.AttackerAmmoReserve
					stats.InitialCrosshairError = e.CrosshairError
					stats.PitchError = e.PitchError
					stats.YawError = e.YawError
					stats.TimeToReaction = e.TimeToReaction
					stats.TimeToFirstDamage = e.TimeToDamage
					stats.ShotsFired = e.AttackerShotsFired
				}
				stats.Velocity = e.AttackerVelocity
				stats.IsBlind = e.AttackerIsBlind
				stats.IsDucking = e.AttackerIsDucking
				stats.Position = e.AttackerPosition
				first = false
			}
			stats.HealthAfter = e.AttackerHealth
			stats.ArmorAfter = e.AttackerArmor
		}
	}

	return stats
}

// aggregateVictimStats collects victim stats from events
func aggregateVictimStats(events []models.RawCombatEvent, victimID uint64) *playerDuelStats {
	stats := &playerDuelStats{SteamID: victimID}
	first := true

	for _, e := range events {
		if e.VictimSteamID == victimID {
			stats.TotalDamage += e.Damage
			stats.Hits++
			if e.Hitgroup == "head" {
				stats.Headshots++
			}

			if first {
				stats.Name = e.VictimName
				stats.Team = e.VictimTeam
				stats.MapArea = e.VictimMapArea
				stats.Weapon = e.VictimWeapon
				stats.HealthBefore = e.VictimHealthBefore
				stats.ArmorBefore = e.VictimArmorBefore
				stats.EquipmentValue = e.VictimEquipmentValue
				stats.Velocity = e.VictimVelocity
				stats.IsBlind = e.VictimIsBlind
				stats.IsDucking = e.VictimIsDucking
				stats.Position = e.VictimPosition
				stats.EnemiesVisible = e.EnemiesVisibleToVictim
				first = false
			}
			stats.HealthAfter = e.VictimHealthAfter
			stats.ArmorAfter = e.VictimArmorAfter
		}
	}

	return stats
}

// playerDuelStats holds aggregated stats for a player in a duel
type playerDuelStats struct {
	SteamID uint64
	Name    string
	Team    string
	MapArea string
	Weapon  string

	HealthBefore int
	HealthAfter  int
	ArmorBefore  int
	ArmorAfter   int

	TotalDamage int
	Hits        int
	Headshots   int

	EquipmentValue int
	AmmoInMagazine int
	AmmoReserve    int

	InitialCrosshairError float64
	PitchError            float64
	YawError              float64
	TimeToReaction        float64
	TimeToFirstDamage     float64

	Velocity   float64
	IsBlind    bool
	IsDucking  bool
	ShotsFired int

	EnemiesVisible int
	Position       models.AI_Vector // [NEW]
}

// aggregateDuelStats collects all stats for a player from the events
func aggregateDuelStats(events []models.RawCombatEvent, playerID uint64) *playerDuelStats {
	stats := &playerDuelStats{
		SteamID: playerID,
	}

	firstAsAttacker := true
	firstAsVictim := true

	for _, e := range events {
		if e.AttackerSteamID == playerID {
			// This player was the attacker in this event
			stats.TotalDamage += e.Damage
			stats.Hits++
			if e.Hitgroup == "head" {
				stats.Headshots++
			}

			if firstAsAttacker {
				stats.Name = e.AttackerName
				stats.Team = e.AttackerTeam
				stats.MapArea = e.AttackerMapArea
				stats.Weapon = e.Weapon
				stats.HealthBefore = e.AttackerHealth
				stats.ArmorBefore = e.AttackerArmor
				stats.EquipmentValue = e.AttackerEquipmentValue
				// Only set weapon-related metrics for non-grenades
				if !isGrenade(e.Weapon) {
					stats.AmmoInMagazine = e.AttackerAmmoInMagazine
					stats.AmmoReserve = e.AttackerAmmoReserve
					stats.InitialCrosshairError = e.CrosshairError
					stats.PitchError = e.PitchError
					stats.YawError = e.YawError
					stats.TimeToReaction = e.TimeToReaction
					stats.TimeToFirstDamage = e.TimeToDamage
					stats.ShotsFired = e.AttackerShotsFired
				}
				stats.Velocity = e.AttackerVelocity
				stats.IsBlind = e.AttackerIsBlind
				stats.IsDucking = e.AttackerIsDucking
				stats.Position = e.AttackerPosition
				firstAsAttacker = false
			}
			// Update "after" values from last event where they were attacker
			stats.HealthAfter = e.AttackerHealth
			stats.ArmorAfter = e.AttackerArmor

		} else if e.VictimSteamID == playerID {
			// This player was the victim in this event
			if firstAsVictim {
				if stats.Name == "" {
					stats.Name = e.VictimName
					stats.Team = e.VictimTeam
					stats.MapArea = e.VictimMapArea
				}
				// Capture weapon if not already set (player didn't attack)
				if stats.Weapon == "" {
					stats.Weapon = e.VictimWeapon
				}
				stats.HealthBefore = e.VictimHealthBefore
				stats.ArmorBefore = e.VictimArmorBefore
				stats.EquipmentValue = e.VictimEquipmentValue
				stats.Velocity = e.VictimVelocity
				stats.IsBlind = e.VictimIsBlind
				stats.IsDucking = e.VictimIsDucking
				stats.Position = e.VictimPosition
				stats.EnemiesVisible = e.EnemiesVisibleToVictim
				firstAsVictim = false
			}
			// Update "after" values from last event where they were victim
			stats.HealthAfter = e.VictimHealthAfter
			stats.ArmorAfter = e.VictimArmorAfter
		}
	}

	return stats
}

// anyThroughSmoke returns true if any event in the duel was through smoke
func anyThroughSmoke(events []models.RawCombatEvent) bool {
	for _, e := range events {
		if e.ThroughSmoke {
			return true
		}
	}
	return false
}

// anyWallbang returns true if any event in the duel was a wallbang
func anyWallbang(events []models.RawCombatEvent) bool {
	for _, e := range events {
		if e.IsWallbang {
			return true
		}
	}
	return false
}

// maxPenetratedObjects returns the max penetrated objects from all events
func maxPenetratedObjects(events []models.RawCombatEvent) int {
	max := 0
	for _, e := range events {
		if e.PenetratedObjects > max {
			max = e.PenetratedObjects
		}
	}
	return max
}

// ============================================================================
// RAW EVENT CAPTURE FUNCTIONS
// Capture kill/damage events for later consolidation
// ============================================================================

// hitgroupToString converts a demoinfocs HitGroup to a human-readable string
func hitgroupToString(hg events.HitGroup) string {
	switch hg {
	case events.HitGroupHead:
		return "head"
	case events.HitGroupChest:
		return "chest"
	case events.HitGroupStomach:
		return "stomach"
	case events.HitGroupLeftArm:
		return "left_arm"
	case events.HitGroupRightArm:
		return "right_arm"
	case events.HitGroupLeftLeg:
		return "left_leg"
	case events.HitGroupRightLeg:
		return "right_leg"
	case events.HitGroupNeck:
		return "neck"
	default:
		return "generic"
	}
}

// captureRawKillEvent captures a kill event for later consolidation into duels
func captureRawKillEvent(ctx *models.DemoContext, e *events.Kill) {
	if e.Killer == nil || e.Victim == nil {
		return
	}
	if ctx.ActualRoundNumber == 0 {
		return // Skip warmup
	}

	killerTeam := getPlayerTeamString(e.Killer)
	victimTeam := getPlayerTeamString(e.Victim)

	// Calculate distance and height diff
	heightDiff := calculateHeightDiff(e.Killer, e.Victim)

	// Get crosshair error, pitch/yaw errors, and time metrics from ReactionTime data
	crosshairError := 0.0
	pitchError := 0.0
	yawError := 0.0
	timeToReaction := 0.0 // Time from visibility to first shot (pure reaction)
	timeToDamage := 0.0   // Time from visibility to this damage (reaction + accuracy)
	firstSeenTick := 0    // Track which visibility window this hit belongs to
	killerID := e.Killer.SteamID64
	victimID := e.Victim.SteamID64
	currentTick := ctx.Parser.GameState().IngameTick()
	tickRate := ctx.Parser.TickRate()
	if tickRate == 0 {
		tickRate = 64
	}

	// Try to get metrics from ReactionTimes (populated by WeaponFire handler)
	if playerData, exists := ctx.MatchData.Players[killerID]; exists {
		for i := len(playerData.ReactionTimes) - 1; i >= 0; i-- {
			rt := playerData.ReactionTimes[i]
			if rt.EnemyID == victimID && (currentTick-rt.FirstShotTick) < 128 {
				crosshairError = rt.CrosshairPlacementError
				pitchError = rt.PitchError
				yawError = rt.YawError
				timeToReaction = float64(rt.ReactionTimeMs)
				timeToDamage = rt.TimeToDamage
				firstSeenTick = rt.FirstSeenTick
				break
			}
		}
	}

	// Fallback: If no ReactionTimeEvent, try to calculate directly from FirstSeenTick
	// This handles cases where WeaponFire didn't register but Kill event did
	if firstSeenMap, ok := ctx.EnemyFirstSeenTick[killerID]; ok {
		if firstSeenData, ok := firstSeenMap[victimID]; ok {
			deltaTicks := currentTick - firstSeenData.Tick
			if deltaTicks >= 0 && deltaTicks <= 320 { // Valid range: 0-2500ms
				// Capture FirstSeenTick for visibility window tracking
				if firstSeenTick == 0 {
					firstSeenTick = firstSeenData.Tick
				}
				// Calculate time to damage if not already set
				if timeToDamage == 0 {
					timeToDamage = float64(deltaTicks) * (1000.0 / tickRate)
				}
				// Calculate time to reaction if not already set
				if timeToReaction == 0 {
					timeToReaction = float64(deltaTicks) * (1000.0 / tickRate)
				}
				// Also capture crosshair metrics from firstSeenData if we didn't have them
				if crosshairError == 0 {
					crosshairError = firstSeenData.CrosshairPlacementError
					pitchError = firstSeenData.PitchError
					yawError = firstSeenData.YawError
				}
			}
		}
	}

	// Count alive players
	aliveCT, aliveT := countAlivePlayers(ctx)

	// Determine if opening kill
	isOpeningKill := isFirstKillOfRound(ctx)

	// Get victim HP before damage
	victimHPBefore := getVictimPreDamageHP(ctx, e.Victim.SteamID64)

	// Determine hitgroup
	hitgroup := "chest"
	if e.IsHeadshot {
		hitgroup = "head"
	}

	// Get map areas
	killerMapArea := getAreaName(ctx, e.Killer)
	victimMapArea := getAreaName(ctx, e.Victim)

	rawEvent := models.RawCombatEvent{
		Tick:   ctx.Parser.GameState().IngameTick(),
		Round:  ctx.ActualRoundNumber,
		IsKill: true,

		// Attacker state
		AttackerSteamID:        e.Killer.SteamID64,
		AttackerName:           e.Killer.Name,
		AttackerTeam:           killerTeam,
		AttackerMapArea:        killerMapArea,
		AttackerHealth:         e.Killer.Health(),
		AttackerArmor:          e.Killer.Armor(),
		AttackerEquipmentValue: e.Killer.EquipmentValueCurrent(),
		AttackerAmmoInMagazine: getAmmoInMagazine(e.Killer),
		AttackerAmmoReserve:    getAmmoReserve(e.Killer),
		AttackerVelocity:       calculatePlayerVelocity(ctx, e.Killer),
		AttackerIsBlind:        e.Killer.FlashDuration > 0,
		AttackerIsDucking:      isPlayerDucking(e.Killer),
		AttackerShotsFired:     getShotsFired(ctx, e.Killer.SteamID64),
		AttackerPosition:       models.AI_Vector{X: float64(e.Killer.Position().X), Y: float64(e.Killer.Position().Y), Z: float64(e.Killer.Position().Z)},

		// Victim state
		VictimSteamID:        e.Victim.SteamID64,
		VictimName:           e.Victim.Name,
		VictimTeam:           victimTeam,
		VictimMapArea:        victimMapArea,
		VictimWeapon:         getActiveWeaponName(ctx, e.Victim),
		VictimHealthBefore:   victimHPBefore,
		VictimHealthAfter:    0,
		VictimArmorBefore:    e.Victim.Armor(),
		VictimArmorAfter:     0,
		VictimEquipmentValue: e.Victim.EquipmentValueCurrent(),
		VictimVelocity:       calculatePlayerVelocity(ctx, e.Victim),
		VictimIsBlind:        e.Victim.FlashDuration > 0,
		VictimIsDucking:      isPlayerDucking(e.Victim),
		VictimPosition:       models.AI_Vector{X: float64(e.Victim.Position().X), Y: float64(e.Victim.Position().Y), Z: float64(e.Victim.Position().Z)},

		// Combat details
		Weapon:            e.Weapon.String(),
		Damage:            victimHPBefore,
		Hitgroup:          hitgroup,
		Distance:          float64(e.Distance),
		HeightDiff:        heightDiff,
		ThroughSmoke:      e.ThroughSmoke,
		IsWallbang:        e.IsWallBang(),
		PenetratedObjects: e.PenetratedObjects,
		IsHeadshot:        e.IsHeadshot,
		NoScope:           e.NoScope,
		ZoomLevel:         getZoomLevel(e.Killer),

		// Aim metrics
		CrosshairError: crosshairError,
		PitchError:     pitchError,
		YawError:       yawError,
		TimeToReaction: timeToReaction,
		TimeToDamage:   timeToDamage,
		FirstSeenTick:  firstSeenTick,

		// Round context
		BombPlanted:            ctx.BombPlanted,
		AliveCT:                aliveCT,
		AliveT:                 aliveT,
		EnemiesVisibleToVictim: countVisibleEnemies(ctx, e.Victim),
		RoundTimeRemaining:     calculateRoundTimeRemaining(ctx),
		IsTrade:                false,
		IsOpeningKill:          isOpeningKill,
	}

	ctx.RawCombatEvents = append(ctx.RawCombatEvents, rawEvent)
}

// captureRawDamageEvent captures a damage event for later consolidation into duels
func captureRawDamageEvent(ctx *models.DemoContext, e *events.PlayerHurt) {
	if e.Attacker == nil || e.Player == nil {
		return
	}
	if ctx.ActualRoundNumber == 0 {
		return // Skip warmup
	}
	if e.Health == 0 {
		return // Skip fatal damage (will be handled by kill event)
	}

	attackerTeam := getPlayerTeamString(e.Attacker)
	victimTeam := getPlayerTeamString(e.Player)

	// Calculate distance and height diff
	attackerPos := e.Attacker.Position()
	victimPos := e.Player.Position()
	dist := math.Sqrt(math.Pow(float64(attackerPos.X-victimPos.X), 2) +
		math.Pow(float64(attackerPos.Y-victimPos.Y), 2) +
		math.Pow(float64(attackerPos.Z-victimPos.Z), 2))
	heightDiff := attackerPos.Z - victimPos.Z

	// Get crosshair error, pitch/yaw errors, and time metrics from ReactionTime data
	crosshairError := 0.0
	pitchError := 0.0
	yawError := 0.0
	timeToReaction := 0.0 // Time from visibility to first shot (pure reaction)
	timeToDamage := 0.0   // Time from visibility to this damage (reaction + accuracy)
	firstSeenTick := 0    // Track which visibility window this hit belongs to
	attackerID := e.Attacker.SteamID64
	victimID := e.Player.SteamID64
	currentTick := ctx.Parser.GameState().IngameTick()
	tickRate := ctx.Parser.TickRate()
	if tickRate == 0 {
		tickRate = 64
	}

	// Try to get metrics from ReactionTimes (populated by WeaponFire handler)
	if playerData, exists := ctx.MatchData.Players[attackerID]; exists {
		for i := len(playerData.ReactionTimes) - 1; i >= 0; i-- {
			rt := playerData.ReactionTimes[i]
			if rt.EnemyID == victimID && (currentTick-rt.FirstShotTick) < 128 {
				crosshairError = rt.CrosshairPlacementError
				pitchError = rt.PitchError
				yawError = rt.YawError
				timeToReaction = float64(rt.ReactionTimeMs)
				timeToDamage = rt.TimeToDamage
				firstSeenTick = rt.FirstSeenTick
				break
			}
		}
	}

	// Fallback: If no ReactionTimeEvent, try to calculate directly from FirstSeenTick
	// This handles cases where PlayerHurt fires BEFORE WeaponFire in the same tick
	if firstSeenMap, ok := ctx.EnemyFirstSeenTick[attackerID]; ok {
		if firstSeenData, ok := firstSeenMap[victimID]; ok {
			deltaTicks := currentTick - firstSeenData.Tick
			if deltaTicks >= 0 && deltaTicks <= 320 { // Valid range: 0-2500ms
				// Capture FirstSeenTick for visibility window tracking
				if firstSeenTick == 0 {
					firstSeenTick = firstSeenData.Tick
				}

				// Calculate time to damage (from FirstSeen to this damage event)
				if timeToDamage == 0 {
					timeToDamage = float64(deltaTicks) * (1000.0 / tickRate)
				}

				// Calculate time to reaction (from FirstSeen to first shot, which is this tick)
				// Since we're in PlayerHurt, the shot that caused this damage was at or before currentTick
				// The most accurate is to use this delta as TimeToReaction since it's the first damage from this attacker
				if timeToReaction == 0 {
					timeToReaction = float64(deltaTicks) * (1000.0 / tickRate)
				}

				// Capture crosshair metrics from firstSeenData if we didn't have them
				if crosshairError == 0 {
					crosshairError = firstSeenData.CrosshairPlacementError
					pitchError = firstSeenData.PitchError
					yawError = firstSeenData.YawError
				}
			}
		}
	}

	// Count alive players
	aliveCT, aliveT := countAlivePlayers(ctx)

	// Calculate victim HP before/after
	victimHPBefore := e.Health + e.HealthDamage
	if victimHPBefore > 100 {
		victimHPBefore = 100
	}
	victimArmorBefore := e.Armor + e.ArmorDamage

	// Get map areas
	attackerMapArea := getAreaName(ctx, e.Attacker)
	victimMapArea := getAreaName(ctx, e.Player)

	rawEvent := models.RawCombatEvent{
		Tick:   ctx.Parser.GameState().IngameTick(),
		Round:  ctx.ActualRoundNumber,
		IsKill: false,

		// Attacker state
		AttackerSteamID:        e.Attacker.SteamID64,
		AttackerName:           e.Attacker.Name,
		AttackerTeam:           attackerTeam,
		AttackerMapArea:        attackerMapArea,
		AttackerHealth:         e.Attacker.Health(),
		AttackerArmor:          e.Attacker.Armor(),
		AttackerEquipmentValue: e.Attacker.EquipmentValueCurrent(),
		AttackerAmmoInMagazine: getAmmoInMagazine(e.Attacker),
		AttackerAmmoReserve:    getAmmoReserve(e.Attacker),
		AttackerVelocity:       calculatePlayerVelocity(ctx, e.Attacker),
		AttackerIsBlind:        e.Attacker.FlashDuration > 0,
		AttackerIsDucking:      isPlayerDucking(e.Attacker),
		AttackerShotsFired:     getShotsFired(ctx, e.Attacker.SteamID64),
		AttackerPosition:       models.AI_Vector{X: float64(e.Attacker.Position().X), Y: float64(e.Attacker.Position().Y), Z: float64(e.Attacker.Position().Z)},

		// Victim state
		VictimSteamID:        e.Player.SteamID64,
		VictimName:           e.Player.Name,
		VictimTeam:           victimTeam,
		VictimMapArea:        victimMapArea,
		VictimWeapon:         getActiveWeaponName(ctx, e.Player),
		VictimHealthBefore:   victimHPBefore,
		VictimHealthAfter:    e.Health,
		VictimArmorBefore:    victimArmorBefore,
		VictimArmorAfter:     e.Armor,
		VictimEquipmentValue: e.Player.EquipmentValueCurrent(),
		VictimVelocity:       calculatePlayerVelocity(ctx, e.Player),
		VictimIsBlind:        e.Player.FlashDuration > 0,
		VictimIsDucking:      isPlayerDucking(e.Player),
		VictimPosition:       models.AI_Vector{X: float64(e.Player.Position().X), Y: float64(e.Player.Position().Y), Z: float64(e.Player.Position().Z)},

		// Combat details
		Weapon:            e.Weapon.String(),
		Damage:            e.HealthDamage,
		Hitgroup:          hitgroupToString(e.HitGroup),
		Distance:          dist,
		HeightDiff:        heightDiff,
		ThroughSmoke:      isThroughSmoke(ctx, attackerPos, victimPos),
		IsWallbang:        false, // Not available in PlayerHurt
		PenetratedObjects: 0,
		IsHeadshot:        e.HitGroup == events.HitGroupHead,
		NoScope:           false,
		ZoomLevel:         getZoomLevel(e.Attacker),

		// Aim metrics
		CrosshairError: crosshairError,
		PitchError:     pitchError,
		YawError:       yawError,
		TimeToReaction: timeToReaction,
		TimeToDamage:   timeToDamage,
		FirstSeenTick:  firstSeenTick,

		// Round context
		BombPlanted:            ctx.BombPlanted,
		AliveCT:                aliveCT,
		AliveT:                 aliveT,
		EnemiesVisibleToVictim: countVisibleEnemies(ctx, e.Player),
		RoundTimeRemaining:     calculateRoundTimeRemaining(ctx),
		IsTrade:                false,
		IsOpeningKill:          false,
	}

	ctx.RawCombatEvents = append(ctx.RawCombatEvents, rawEvent)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// countAlivePlayers returns (aliveCT, aliveT)
func countAlivePlayers(ctx *models.DemoContext) (int, int) {
	aliveCT := 0
	aliveT := 0
	for _, player := range ctx.Parser.GameState().Participants().Playing() {
		if player.IsAlive() {
			if player.Team == common.TeamCounterTerrorists {
				aliveCT++
			} else if player.Team == common.TeamTerrorists {
				aliveT++
			}
		}
	}
	return aliveCT, aliveT
}

// isFirstKillOfRound checks if this is the first kill of the round
func isFirstKillOfRound(ctx *models.DemoContext) bool {
	// Check if any kills have been recorded this round
	for _, duel := range ctx.RawCombatEvents {
		if duel.IsKill && duel.Round == ctx.ActualRoundNumber {
			return false
		}
	}
	return true
}

// isThroughSmoke checks if a shot went through smoke
func isThroughSmoke(ctx *models.DemoContext, from, to r3.Vector) bool {
	for _, smokePos := range ctx.ActiveSmokes {
		// Simple check: if smoke center is within ~144 units of the line between shooter and target
		dist := distancePointToSegment(smokePos, from, to)
		if dist < 144 {
			return true
		}
	}
	return false
}

// distancePointToSegment calculates distance from point P to line segment AB
func distancePointToSegment(p, a, b r3.Vector) float64 {
	ab := b.Sub(a)
	ap := p.Sub(a)
	t := ap.Dot(ab) / ab.Dot(ab)
	if t < 0.0 {
		t = 0.0
	} else if t > 1.0 {
		t = 1.0
	}
	nearest := a.Add(ab.Mul(t))
	return p.Sub(nearest).Norm()
}

// countVisibleEnemies counts how many enemies are visible to the player at this moment
func countVisibleEnemies(ctx *models.DemoContext, player *common.Player) int {
	if player == nil || ctx.MapManager == nil {
		return 0
	}

	count := 0
	playerTeam := player.Team
	playerEyes := player.Position()
	playerEyes.Z += 64 // Eye level

	for _, other := range ctx.Parser.GameState().Participants().Playing() {
		if other == nil || !other.IsAlive() {
			continue
		}
		// Skip teammates
		if other.Team == playerTeam {
			continue
		}
		// Skip self
		if other.SteamID64 == player.SteamID64 {
			continue
		}

		// Check if enemy is visible (head or chest)
		enemyHead := other.Position()
		enemyHead.Z += 62 // Head level
		enemyChest := other.Position()
		enemyChest.Z += 40 // Chest level

		isVisible := ctx.MapManager.IsVisible(playerEyes, enemyHead)
		if !isVisible {
			isVisible = ctx.MapManager.IsVisible(playerEyes, enemyChest)
		}

		if isVisible {
			count++
		}
	}

	return count
}

// getVictimPreDamageHP returns the HP of the victim before the current combat encounter
func getVictimPreDamageHP(ctx *models.DemoContext, steamID uint64) int {
	if hp, exists := ctx.LastKnownHealth[steamID]; exists {
		return hp
	}
	return 100 // Default if never damaged
}

// getActiveWeaponName returns the name of the player's active weapon
// Uses multiple fallbacks to ensure weapon is captured even for dead players:
// 1. Current ActiveWeapon() (for alive players)
// 2. PreviousWeaponState (tracked from previous tick - best for victims)
// 3. LastWeaponState (current tick tracking)
// 4. LastActiveWeapon tracking
// 5. Player's inventory (rifles > pistols > knife)
func getActiveWeaponName(ctx *models.DemoContext, player *common.Player) string {
	if player == nil {
		return ""
	}

	steamID := player.SteamID64

	// Try current active weapon first (works for alive players)
	if weapon := player.ActiveWeapon(); weapon != nil {
		return weapon.String()
	}

	// Fallback 1: Use PreviousWeaponState (tracked from previous tick while alive)
	// This is the most reliable for victims since it's captured before death
	if prevState, exists := ctx.PreviousWeaponState[steamID]; exists && prevState != nil && prevState.WeaponName != "" {
		return prevState.WeaponName
	}

	// Fallback 2: Use LastWeaponState (current tick tracking)
	if lastState, exists := ctx.LastWeaponState[steamID]; exists && lastState != nil && lastState.WeaponName != "" {
		return lastState.WeaponName
	}

	// Fallback 3: Use LastActiveWeapon tracking
	if weaponName, exists := ctx.LastActiveWeapon[steamID]; exists && weaponName != "" {
		return weaponName
	}

	// Fallback 4: Check player's inventory (dead players still have their weapons)
	// Priority: Primary (rifles/SMGs) > Secondary (pistols) > Knife
	weapons := player.Weapons()
	var primary, secondary, knife *common.Equipment

	for _, w := range weapons {
		if w == nil {
			continue
		}
		switch w.Class() {
		case common.EqClassRifle, common.EqClassSMG, common.EqClassHeavy:
			if primary == nil {
				primary = w
			}
		case common.EqClassPistols:
			if secondary == nil {
				secondary = w
			}
		case common.EqClassEquipment:
			if w.Type == common.EqKnife && knife == nil {
				knife = w
			}
		}
	}

	if primary != nil {
		return primary.String()
	}
	if secondary != nil {
		return secondary.String()
	}
	if knife != nil {
		return knife.String()
	}

	return ""
}
