package handlers

import (
	"cmp"
	"math"
	"slices"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
)

func projectReplayCombatMarkers(
	events []models.ReplayEvent,
	ledger []combat.Event,
	round int,
) []models.ReplayEvent {
	projected := make([]models.ReplayEvent, 0, len(events)+len(ledger))
	for _, event := range events {
		if event.Type != "player_hurt" && event.Type != "kill" {
			projected = append(projected, event)
		}
	}
	for _, event := range ledger {
		if event.Round != round {
			continue
		}
		var marker models.ReplayEvent
		switch event.Type {
		case combat.EventPlayerHurt:
			marker = replayHurtMarker(event)
		case combat.EventKill:
			marker = replayKillMarker(event)
		default:
			continue
		}
		projected = append(projected, marker)
	}
	slices.SortFunc(projected, compareReplayEvents)
	return projected
}

func replayHurtMarker(event combat.Event) models.ReplayEvent {
	marker := replayCombatMarker(event, "player_hurt")
	if event.HealthDamageTaken != nil {
		marker.Damage = *event.HealthDamageTaken
	}
	if event.IsHeadshot != nil {
		marker.Headshot = *event.IsHeadshot
	}
	marker.DurationMS = 320
	return marker
}

func replayKillMarker(event combat.Event) models.ReplayEvent {
	marker := replayCombatMarker(event, "kill")
	if event.IsHeadshot != nil {
		marker.Headshot = *event.IsHeadshot
	}
	if event.PenetratedObjects != nil {
		marker.Wallbang = *event.PenetratedObjects > 0
	}
	if event.NoScope != nil {
		marker.NoScope = *event.NoScope
	}
	return marker
}

func replayCombatMarker(event combat.Event, eventType string) models.ReplayEvent {
	marker := models.ReplayEvent{
		ID: "combat-" + event.LocalID, Tick: event.Tick, Type: eventType,
		SourceEventIDs: []string{event.LocalID},
		KillerID:       event.Actor.ID, VictimID: event.Target.ID, AssisterID: event.Assister.ID,
		KillerName: event.Actor.Name, VictimName: event.Target.Name, AssisterName: event.Assister.Name,
		KillerTeam: event.Actor.Side, VictimTeam: event.Target.Side, AssisterTeam: event.Assister.Side,
	}
	if event.Weapon.Status == combat.AvailabilityObserved {
		marker.Weapon = event.Weapon.Name
	}
	if event.Actor.PositionStatus == combat.AvailabilityObserved && event.Actor.Position != nil {
		marker.KillerX = event.Actor.Position.X
		marker.KillerY = event.Actor.Position.Y
	}
	if event.Target.PositionStatus == combat.AvailabilityObserved && event.Target.Position != nil {
		marker.VictimX = event.Target.Position.X
		marker.VictimY = event.Target.Position.Y
	}
	return marker
}

func projectReplayCombatFrames(
	frames []models.ReplayFrame,
	ledger []combat.Event,
	round int,
) []models.ReplayFrame {
	fires := make([]combat.Event, 0)
	for _, event := range ledger {
		if event.Round == round && event.Type == combat.EventWeaponFire {
			fires = append(fires, event)
		}
	}
	for frameIndex := range frames {
		shots := make([]models.ReplayShot, 0)
		for _, fire := range fires {
			if fire.Tick > frames[frameIndex].Tick || frames[frameIndex].Tick-fire.Tick > ShotVisibilityTicks {
				continue
			}
			shots = append(shots, replayShot(fire))
		}
		slices.SortFunc(shots, func(left, right models.ReplayShot) int {
			if order := cmp.Compare(left.Tick, right.Tick); order != 0 {
				return order
			}
			return cmp.Compare(left.SourceEventID, right.SourceEventID)
		})
		frames[frameIndex].Shots = shots
	}
	return frames
}

func projectReplayCombatShots(ledger []combat.Event, round int) []models.ReplayShot {
	shots := make([]models.ReplayShot, 0)
	for _, event := range ledger {
		if event.Round == round && event.Type == combat.EventWeaponFire {
			shots = append(shots, replayShot(event))
		}
	}
	slices.SortFunc(shots, func(left, right models.ReplayShot) int {
		if order := cmp.Compare(left.Tick, right.Tick); order != 0 {
			return order
		}
		return cmp.Compare(left.SourceEventID, right.SourceEventID)
	})
	return shots
}

func replayShot(event combat.Event) models.ReplayShot {
	shot := models.ReplayShot{
		Tick: event.Tick, SourceEventID: event.LocalID, ShotID: event.ShotID,
		ShooterID: event.Actor.ID, Result: string(event.ShotResult),
		ResultStatus:   string(event.ShotResultStatus),
		PositionStatus: string(event.Actor.PositionStatus), PositionSource: event.Actor.PositionSource,
		EndpointStatus: string(combat.AvailabilityUnavailable), EndpointSource: combat.SourceUnavailable,
		Hit: event.ShotResult == combat.ShotHit,
	}
	if event.Weapon.Status == combat.AvailabilityObserved {
		shot.Weapon = event.Weapon.Name
	}
	if event.Actor.PositionStatus != combat.AvailabilityObserved || event.Actor.Position == nil ||
		event.ViewYaw == nil || event.ViewPitch == nil {
		return shot
	}
	yaw := *event.ViewYaw * math.Pi / 180
	pitch := *event.ViewPitch * math.Pi / 180
	if !finite(yaw) || !finite(pitch) {
		return shot
	}
	shot.FromX = event.Actor.Position.X
	shot.FromY = event.Actor.Position.Y
	shot.ToX = shot.FromX + math.Cos(yaw)*math.Cos(pitch)*1500
	shot.ToY = shot.FromY + math.Sin(yaw)*math.Cos(pitch)*1500
	shot.EndpointStatus = string(combat.AvailabilityDerived)
	shot.EndpointSource = "view_direction_projection"
	return shot
}
