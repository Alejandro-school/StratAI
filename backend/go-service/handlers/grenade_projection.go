package handlers

import (
	"sort"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
)

func refreshLegacyUtilityProjection(ctx *models.DemoContext) {
	if ctx == nil || ctx.Utilities == nil {
		return
	}
	throws := ctx.Utilities.Snapshot()
	legacy := make([]models.AI_GrenadeEvent, 0, len(throws))
	flashes := make([]models.FlashEvent, 0)
	hes := make([]models.HEEvent, 0)
	smokes := make([]models.SmokeEvent, 0)
	molotovs := make([]models.MolotovEvent, 0)
	timeline := make([]models.TimelineEvent, 0)

	for _, entry := range throws {
		legacy = append(legacy, projectLegacyUtility(entry))
		if event, ok := projectLegacyTimelineEvent(entry); ok {
			timeline = append(timeline, event)
			switch {
			case event.Flash != nil:
				flashes = append(flashes, *event.Flash)
			case event.HE != nil:
				hes = append(hes, *event.HE)
			case event.Smoke != nil:
				smokes = append(smokes, *event.Smoke)
			case event.Molotov != nil:
				molotovs = append(molotovs, *event.Molotov)
			}
		}
	}
	ctx.AI_GrenadeEvents = legacy
	if ctx.MatchData != nil {
		ctx.MatchData.Flashes = flashes
		ctx.MatchData.HEGrenades = hes
		ctx.MatchData.Smokes = smokes
		ctx.MatchData.Molotovs = molotovs
	}
	ctx.Timeline = replaceUtilityTimeline(ctx.Timeline, timeline, 0)
	ctx.CurrentRoundEvents = replaceUtilityTimeline(ctx.CurrentRoundEvents, timeline, ctx.CurrentRound)
	for index := range ctx.RoundTimelines {
		round := ctx.RoundTimelines[index].RoundNumber
		ctx.RoundTimelines[index].Events = replaceUtilityTimeline(
			ctx.RoundTimelines[index].Events,
			timeline,
			round,
		)
	}
}

func projectLegacyUtility(entry utility.Throw) models.AI_GrenadeEvent {
	event := models.AI_GrenadeEvent{
		Round: entry.Round, Type: legacyUtilityType(entry.Type),
		Thrower: entry.Actor.Name, ThrowerSteamID: entry.Actor.ID,
		ThrowerSide: legacyUtilitySide(entry.Actor.Side),
		DidBounce: entry.Trajectory.BounceStatus == utility.AvailabilityObserved &&
			entry.Trajectory.BounceCount > 0,
		Extinguished: entry.Lifecycle.EndReason == utility.EndReasonSmokeExtinguished,
	}
	if entry.Launch.Tick.Status == utility.AvailabilityObserved {
		event.TickThrow = entry.Launch.Tick.Tick
	}
	if entry.Launch.Position.Status == utility.AvailabilityObserved {
		event.StartPosition = legacyUtilityVector(entry.Launch.Position.Value)
	}
	if entry.Launch.View.Status == utility.AvailabilityObserved {
		event.ThrowViewVector = legacyUtilityVector(entry.Launch.View.Vector)
	}
	if entry.Launch.Area.Status == utility.AvailabilityObserved {
		event.ThrowerAreaName = entry.Launch.Area.Value
	}
	if entry.Lifecycle.Area.Status == utility.AvailabilityObserved {
		event.LandArea = entry.Lifecycle.Area.Value
	}
	if moment, ok := utilityEffectMoment(entry.Lifecycle); ok {
		event.TickExplode = moment.Tick
		if moment.PositionStatus == utility.AvailabilityObserved {
			event.EndPosition = legacyUtilityVector(moment.Position)
		}
	}
	if entry.Lifecycle.Duration.Status == utility.AvailabilityObserved {
		event.Duration = entry.Lifecycle.Duration.Value
	}
	projectLegacyFlashes(entry, &event)
	projectLegacyDamage(entry, &event)
	return event
}

func projectLegacyFlashes(entry utility.Throw, event *models.AI_GrenadeEvent) {
	for _, effect := range entry.Flashes {
		if effect.Victim.Status == utility.AvailabilityObserved {
			duration := float32(0)
			if effect.Duration.Status == utility.AvailabilityObserved {
				duration = float32(effect.Duration.Value)
			}
			event.BlindedPlayers = append(event.BlindedPlayers, models.AI_BlindedPlayer{
				SteamID: effect.Victim.ID, Name: effect.Victim.Name, Duration: duration,
				Team:     legacyUtilitySide(effect.Victim.Side),
				IsEnemy:  effect.Relation == utility.RelationEnemy,
				IsSelf:   effect.Relation == utility.RelationSelf,
				Relation: string(effect.Relation),
			})
		}
		switch effect.Relation {
		case utility.RelationEnemy:
			event.EnemiesBlinded++
		case utility.RelationTeammate, utility.RelationSelf:
			event.AlliesBlinded++
		}
	}
}

func projectLegacyDamage(entry utility.Throw, event *models.AI_GrenadeEvent) {
	players := make(map[uint64]*models.AI_DamagedPlayer)
	for _, effect := range entry.Damage {
		damage := max(0, effect.HealthDamage)
		armorDamage := max(0, effect.ArmorDamage)
		event.DamageDealt += damage
		event.ArmorDamageDealt += armorDamage
		switch effect.Relation {
		case utility.RelationEnemy:
			event.EnemyDamage += damage
			event.EnemyArmorDamage += armorDamage
		case utility.RelationTeammate:
			event.FriendlyDamage += damage
			event.FriendlyArmorDamage += armorDamage
		case utility.RelationSelf:
			event.SelfDamage += damage
			event.SelfArmorDamage += armorDamage
		}
		if effect.Victim.Status != utility.AvailabilityObserved {
			continue
		}
		player := players[effect.Victim.ID]
		if player == nil {
			player = &models.AI_DamagedPlayer{
				SteamID: effect.Victim.ID, Name: effect.Victim.Name,
				Team:     legacyUtilitySide(effect.Victim.Side),
				IsEnemy:  effect.Relation == utility.RelationEnemy,
				IsSelf:   effect.Relation == utility.RelationSelf,
				Relation: string(effect.Relation),
			}
			players[effect.Victim.ID] = player
		}
		player.Damage += damage
		player.ArmorDamage += armorDamage
		player.IsKill = player.IsKill || effect.Kill
	}
	ids := make([]uint64, 0, len(players))
	for steamID := range players {
		ids = append(ids, steamID)
	}
	sort.Slice(ids, func(left, right int) bool { return ids[left] < ids[right] })
	for _, steamID := range ids {
		player := players[steamID]
		event.DamagedPlayers = append(event.DamagedPlayers, *player)
		hasDamage := player.Damage > 0 || player.ArmorDamage > 0
		switch utility.Relation(player.Relation) {
		case utility.RelationSelf:
			event.SelfDamaged = hasDamage
		case utility.RelationEnemy:
			if hasDamage {
				event.EnemiesDamaged++
			}
			if player.IsKill {
				event.Kills++
			}
		case utility.RelationTeammate:
			if hasDamage {
				event.AlliesDamaged++
			}
		}
	}
}

func projectLegacyTimelineEvent(entry utility.Throw) (models.TimelineEvent, bool) {
	moment, observed := utilityEffectMoment(entry.Lifecycle)
	if !observed {
		return models.TimelineEvent{}, false
	}
	position := utility.Vector{}
	if moment.PositionStatus == utility.AvailabilityObserved {
		position = moment.Position
	}
	base := models.TimelineEvent{Tick: moment.Tick, Round: entry.Round}
	switch entry.Type {
	case utility.TypeFlashbang:
		legacy := projectLegacyUtility(entry)
		victims := make([]models.FlashVictim, 0, len(legacy.BlindedPlayers))
		for _, victim := range legacy.BlindedPlayers {
			victims = append(victims, models.FlashVictim{
				Name: victim.Name, Duration: victim.Duration, Team: victim.Team,
			})
		}
		base.Type = "flash"
		base.Flash = &models.FlashEvent{
			Tick: moment.Tick, Round: entry.Round, Thrower: entry.Actor.Name,
			X: position.X, Y: position.Y, Z: position.Z, Victims: victims,
			ThrowerAreaName: legacy.ThrowerAreaName, LandAreaName: legacy.LandArea,
			EnemiesBlinded: legacy.EnemiesBlinded, AlliesBlinded: legacy.AlliesBlinded,
		}
	case utility.TypeHE:
		base.Type = "he"
		base.HE = &models.HEEvent{
			Tick: moment.Tick, Round: entry.Round, Thrower: entry.Actor.Name,
			X: position.X, Y: position.Y, Z: position.Z,
			ThrowerAreaName: observedString(entry.Launch.Area),
			LandAreaName:    observedString(entry.Lifecycle.Area),
		}
	case utility.TypeSmoke:
		base.Type = "smoke"
		base.Smoke = &models.SmokeEvent{
			Tick: moment.Tick, Round: entry.Round, Thrower: entry.Actor.Name,
			X: position.X, Y: position.Y, Z: position.Z,
			ThrowerAreaName: observedString(entry.Launch.Area),
			LandAreaName:    observedString(entry.Lifecycle.Area),
		}
	case utility.TypeMolotov, utility.TypeIncendiary:
		base.Type = "molotov"
		base.Molotov = &models.MolotovEvent{
			Tick: moment.Tick, Round: entry.Round, Thrower: entry.Actor.Name,
			X: position.X, Y: position.Y, Z: position.Z,
			ThrowerAreaName: observedString(entry.Launch.Area),
			LandAreaName:    observedString(entry.Lifecycle.Area),
		}
	default:
		return models.TimelineEvent{}, false
	}
	return base, true
}

func utilityEffectMoment(lifecycle utility.Lifecycle) (utility.TickPositionObservation, bool) {
	if lifecycle.EffectStart.Status == utility.AvailabilityObserved {
		return lifecycle.EffectStart, true
	}
	if lifecycle.Detonation.Status == utility.AvailabilityObserved {
		return lifecycle.Detonation, true
	}
	return utility.TickPositionObservation{}, false
}

func replaceUtilityTimeline(
	existing []models.TimelineEvent,
	utilityEvents []models.TimelineEvent,
	round int,
) []models.TimelineEvent {
	result := make([]models.TimelineEvent, 0, len(existing)+len(utilityEvents))
	for _, event := range existing {
		if !isUtilityTimelineType(event.Type) {
			result = append(result, event)
		}
	}
	for _, event := range utilityEvents {
		if round == 0 || event.Round == round {
			result = append(result, event)
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Round != result[right].Round {
			return result[left].Round < result[right].Round
		}
		if result[left].Tick != result[right].Tick {
			return result[left].Tick < result[right].Tick
		}
		return timelineTypeOrder(result[left].Type) < timelineTypeOrder(result[right].Type)
	})
	return result
}

func isUtilityTimelineType(value string) bool {
	return value == "flash" || value == "he" || value == "smoke" || value == "molotov"
}

func timelineTypeOrder(value string) int {
	switch value {
	case "round_start":
		return 0
	case "round_end":
		return 2
	default:
		return 1
	}
}

func legacyUtilityType(value utility.Type) string {
	switch value {
	case utility.TypeFlashbang:
		return "Flashbang"
	case utility.TypeSmoke:
		return "Smoke"
	case utility.TypeHE:
		return "HE"
	case utility.TypeMolotov:
		return "Molotov"
	case utility.TypeIncendiary:
		return "Incendiary"
	case utility.TypeDecoy:
		return "Decoy"
	default:
		return "Unknown"
	}
}

func legacyUtilitySide(value string) string {
	if value == "CT" || value == "T" {
		return value
	}
	return ""
}

func legacyUtilityVector(value utility.Vector) models.AI_Vector {
	return models.AI_Vector{X: value.X, Y: value.Y, Z: value.Z}
}

func observedString(value utility.StringObservation) string {
	if value.Status == utility.AvailabilityObserved {
		return value.Value
	}
	return ""
}
