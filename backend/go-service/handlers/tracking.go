package handlers

import (
	"cs2-demo-service/models"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
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

			// Create event
			event := models.AI_TrackingEvent{
				Tick:          currentTick,
				PlayerSteamID: player.SteamID64,
				Team:          getTeamString(player.Team),
				Position: models.AI_Vector{
					X: player.Position().X,
					Y: player.Position().Y,
					Z: player.Position().Z,
				},
				AreaName:           areaName,
				ViewAngleYaw:       player.ViewDirectionX(), // Yaw = horizontal rotation
				ViewAnglePitch:     player.ViewDirectionY(), // Pitch = vertical rotation
				VelocityLen:        math.Sqrt(player.Velocity().X*player.Velocity().X + player.Velocity().Y*player.Velocity().Y),
				IsWalking:          player.IsWalking(),
				IsDucking:          player.IsDucking(),
				ActiveWeapon:       getActiveWeapon(player),
				HasC4:              hasC4(player),
				Health:             player.Health(),
				Armor:              player.Armor(),
				NearbyTeammates:    nearbyTeammates,
				IsAlive:            player.IsAlive(),
				RoundTimeRemaining: calculateRoundTimeRemaining(ctx),
			}

			// Store with round number for later grouping
			ctx.AI_TrackingEventsWithRound = append(ctx.AI_TrackingEventsWithRound, models.AI_TrackingEventWithRound{
				Round: ctx.ActualRoundNumber,
				Event: event,
			})
		}
	})
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

func getActiveWeapon(player *common.Player) string {
	if player.ActiveWeapon() != nil {
		return player.ActiveWeapon().String()
	}
	return "Knife"
}

func getTeamString(team common.Team) string {
	if team == common.TeamCounterTerrorists {
		return "CT"
	} else if team == common.TeamTerrorists {
		return "T"
	}
	return ""
}
