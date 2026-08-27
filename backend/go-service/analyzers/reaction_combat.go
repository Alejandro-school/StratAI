package analyzers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type CombatReactionMetrics struct {
	CrosshairError float64
	PitchError     float64
	YawError       float64
	TimeToReaction float64
	TimeToDamage   float64
	FirstSeenTick  int
}

type CombatReactionEventKey struct {
	Round             int
	Tick              int
	AttackerSteamID   uint64
	VictimSteamID     uint64
	IsKill            bool
	Weapon            string
	Hitgroup          string
	Damage            int
	VictimHealthAfter int
	VictimArmorAfter  int
}

func ResolveCombatReactionMetrics(
	ctx *models.DemoContext,
	attackerID uint64,
	victimID uint64,
	currentTick int,
	priorEventLimit int,
) CombatReactionMetrics {
	metrics := reactionMetricsFromRecentShot(ctx, attackerID, victimID, currentTick)
	metrics = enrichReactionMetricsFromVisibility(ctx, metrics, attackerID, victimID, currentTick)
	if hasPriorVisibilityWindowDamage(ctx, attackerID, victimID, metrics.FirstSeenTick, priorEventLimit) {
		metrics.TimeToReaction = 0
		metrics.TimeToDamage = 0
	}
	return metrics
}

func recordFirstDamageReaction(
	ctx *models.DemoContext,
	attackerID uint64,
	victimID uint64,
	currentTick int,
) {
	if ctx.FirstDamageTick[attackerID] == nil {
		ctx.FirstDamageTick[attackerID] = make(map[uint64]int)
	}
	if _, exists := ctx.FirstDamageTick[attackerID][victimID]; !exists {
		ctx.FirstDamageTick[attackerID][victimID] = currentTick
	}

	firstSeenMap, exists := ctx.EnemyFirstSeenTick[attackerID]
	if !exists {
		return
	}
	firstSeen, exists := firstSeenMap[victimID]
	if !exists || firstSeen.FirstDamageTick != 0 {
		return
	}
	firstSeen.FirstDamageTick = currentTick
	firstSeenMap[victimID] = firstSeen

	deltaTicks := currentTick - firstSeen.Tick
	if deltaTicks < 0 || ctx.MatchData == nil {
		return
	}
	timeToDamage := float64(deltaTicks) * (1000.0 / combatReactionTickRate(ctx))
	playerData, exists := ctx.MatchData.Players[attackerID]
	if !exists || playerData == nil {
		return
	}
	for index := len(playerData.ReactionTimes) - 1; index >= 0; index-- {
		reaction := &playerData.ReactionTimes[index]
		if reaction.EnemyID != victimID ||
			reaction.FirstSeenTick != firstSeen.Tick ||
			reaction.TimeToDamage != 0 {
			continue
		}
		if currentTick-reaction.FirstShotTick <= int(2.5*combatReactionTickRate(ctx)) {
			reaction.TimeToDamage = timeToDamage
			return
		}
	}
}

// EnrichCapturedCombatReaction makes a captured combat fact converge with the
// reaction state finalized by the analyzer, regardless of callback order.
func EnrichCapturedCombatReaction(ctx *models.DemoContext, key CombatReactionEventKey) {
	rawIndex := findLatestCombatEvent(ctx.RawCombatEvents, key)
	if rawIndex < 0 {
		return
	}
	metrics := ResolveCombatReactionMetrics(
		ctx,
		key.AttackerSteamID,
		key.VictimSteamID,
		key.Tick,
		rawIndex,
	)
	applyCombatReactionMetrics(&ctx.RawCombatEvents[rawIndex], metrics)

	atomicIndex := findLatestCombatEvent(ctx.AI_CombatEvents, key)
	if atomicIndex >= 0 {
		applyCombatReactionMetrics(&ctx.AI_CombatEvents[atomicIndex], metrics)
	}
}

func CombatReactionKeyFromRaw(event models.RawCombatEvent) CombatReactionEventKey {
	return CombatReactionEventKey{
		Round:             event.Round,
		Tick:              event.Tick,
		AttackerSteamID:   event.AttackerSteamID,
		VictimSteamID:     event.VictimSteamID,
		IsKill:            event.IsKill,
		Weapon:            event.Weapon,
		Hitgroup:          event.Hitgroup,
		Damage:            event.Damage,
		VictimHealthAfter: event.VictimHealthAfter,
		VictimArmorAfter:  event.VictimArmorAfter,
	}
}

func CombatHitgroup(hitgroup events.HitGroup) string {
	switch hitgroup {
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

func reactionMetricsFromRecentShot(
	ctx *models.DemoContext,
	attackerID uint64,
	victimID uint64,
	currentTick int,
) CombatReactionMetrics {
	if ctx.MatchData == nil {
		return CombatReactionMetrics{}
	}
	playerData, exists := ctx.MatchData.Players[attackerID]
	if !exists || playerData == nil {
		return CombatReactionMetrics{}
	}

	tickRate := combatReactionTickRate(ctx)
	for index := len(playerData.ReactionTimes) - 1; index >= 0; index-- {
		reaction := playerData.ReactionTimes[index]
		deltaTicks := currentTick - reaction.FirstShotTick
		if reaction.EnemyID != victimID || deltaTicks < 0 || float64(deltaTicks) >= 2.5*tickRate {
			continue
		}
		return CombatReactionMetrics{
			CrosshairError: reaction.CrosshairPlacementError,
			PitchError:     reaction.PitchError,
			YawError:       reaction.YawError,
			TimeToReaction: float64(reaction.ReactionTimeMs),
			TimeToDamage:   reaction.TimeToDamage,
			FirstSeenTick:  reaction.FirstSeenTick,
		}
	}
	return CombatReactionMetrics{}
}

func enrichReactionMetricsFromVisibility(
	ctx *models.DemoContext,
	metrics CombatReactionMetrics,
	attackerID uint64,
	victimID uint64,
	currentTick int,
) CombatReactionMetrics {
	firstSeenMap, exists := ctx.EnemyFirstSeenTick[attackerID]
	if !exists {
		return metrics
	}
	firstSeen, exists := firstSeenMap[victimID]
	if !exists {
		return metrics
	}

	tickRate := combatReactionTickRate(ctx)
	deltaTicks := currentTick - firstSeen.Tick
	if deltaTicks < 0 || float64(deltaTicks) > 2.5*tickRate {
		return metrics
	}
	if metrics.FirstSeenTick == 0 {
		metrics.FirstSeenTick = firstSeen.Tick
	}
	if metrics.TimeToDamage == 0 {
		metrics.TimeToDamage = float64(deltaTicks) * (1000.0 / tickRate)
	}
	if metrics.TimeToReaction == 0 && firstSeen.FirstShotTick >= firstSeen.Tick {
		metrics.TimeToReaction = float64(firstSeen.FirstShotTick-firstSeen.Tick) * (1000.0 / tickRate)
	}
	if metrics.CrosshairError == 0 {
		metrics.CrosshairError = firstSeen.CrosshairPlacementError
		metrics.PitchError = firstSeen.PitchError
		metrics.YawError = firstSeen.YawError
	}
	return metrics
}

func hasPriorVisibilityWindowDamage(
	ctx *models.DemoContext,
	attackerID uint64,
	victimID uint64,
	firstSeenTick int,
	priorEventLimit int,
) bool {
	if firstSeenTick == 0 {
		return false
	}
	if priorEventLimit > len(ctx.RawCombatEvents) {
		priorEventLimit = len(ctx.RawCombatEvents)
	}
	for index := priorEventLimit - 1; index >= 0; index-- {
		event := ctx.RawCombatEvents[index]
		if event.Round != ctx.ActualRoundNumber {
			break
		}
		if event.AttackerSteamID == attackerID &&
			event.VictimSteamID == victimID &&
			event.FirstSeenTick == firstSeenTick {
			return true
		}
	}
	return false
}

func findLatestCombatEvent(events []models.RawCombatEvent, key CombatReactionEventKey) int {
	for index := len(events) - 1; index >= 0; index-- {
		if combatEventMatchesKey(events[index], key) {
			return index
		}
	}
	return -1
}

func combatEventMatchesKey(event models.RawCombatEvent, key CombatReactionEventKey) bool {
	if event.Round != key.Round ||
		event.Tick != key.Tick ||
		event.AttackerSteamID != key.AttackerSteamID ||
		event.VictimSteamID != key.VictimSteamID ||
		event.IsKill != key.IsKill ||
		event.Weapon != key.Weapon ||
		event.Hitgroup != key.Hitgroup {
		return false
	}
	if key.IsKill {
		return true
	}
	return event.Damage == key.Damage &&
		event.VictimHealthAfter == key.VictimHealthAfter &&
		event.VictimArmorAfter == key.VictimArmorAfter
}

func applyCombatReactionMetrics(event *models.RawCombatEvent, metrics CombatReactionMetrics) {
	event.CrosshairError = metrics.CrosshairError
	event.PitchError = metrics.PitchError
	event.YawError = metrics.YawError
	event.TimeToReaction = metrics.TimeToReaction
	event.TimeToDamage = metrics.TimeToDamage
	event.FirstSeenTick = metrics.FirstSeenTick
}

func combatReactionTickRate(ctx *models.DemoContext) float64 {
	if ctx.Parser != nil && ctx.Parser.TickRate() > 0 {
		return ctx.Parser.TickRate()
	}
	return 64
}
