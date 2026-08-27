package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
)

type combatCallbackQualityDiagnostics struct {
	Observed  int `json:"observed"`
	Recorded  int `json:"recorded"`
	Discarded int `json:"discarded"`
}

type combatQualityAssessment struct {
	events                       int
	contractViolations           int
	callbackAccountingViolations int
	playerStatsMismatches        int
	replayProjectionMismatches   int
	nativeDeltaMismatches        int
	determinismViolations        int
	missingImpactPositions       int
	missingReloadEnds            int
	unavailableShotResults       int
	discardedCallbacks           int
	diagnostics                  combat.Diagnostics
	failureDetails               []string
}

func assessCombatQuality(ctx *models.DemoContext) combatQualityAssessment {
	assessment := combatQualityAssessment{}
	if ctx == nil || ctx.Combat == nil {
		assessment.contractViolations = 1
		assessment.failureDetails = append(assessment.failureDetails, "combat tracker is unavailable")
		return assessment
	}
	events := ctx.Combat.Snapshot()
	assessment.events = len(events)
	assessment.diagnostics = ctx.Combat.Diagnostics()
	assessment.discardedCallbacks = combatDiscardedCallbackCount(assessment.diagnostics)
	assessment.callbackAccountingViolations = combatCallbackAccountingViolations(assessment.diagnostics)
	assessment.contractViolations = combatContractViolations(ctx, events, assessment.diagnostics)
	assessment.playerStatsMismatches, assessment.nativeDeltaMismatches = combatStatsMismatches(ctx, events)
	assessment.replayProjectionMismatches = combatReplayMismatches(ctx.ReplayData, events)
	assessment.determinismViolations = combatDeterminismViolations(ctx.Combat)
	for _, event := range events {
		if event.Type == combat.EventBulletDamage && event.ImpactPositionStatus != combat.AvailabilityObserved {
			assessment.missingImpactPositions++
		}
		if event.Type == combat.EventWeaponReload && event.ReloadEndStatus != combat.AvailabilityObserved {
			assessment.missingReloadEnds++
		}
		if event.Type == combat.EventWeaponFire && event.ShotResultStatus == combat.AvailabilityUnavailable {
			assessment.unavailableShotResults++
		}
	}
	assessment.failureDetails = combatFailureDetails(assessment)
	if assessment.contractViolations > 0 {
		assessment.failureDetails = append(assessment.failureDetails, combatContractViolationExamples(ctx, events, assessment.diagnostics)...)
	}
	if assessment.replayProjectionMismatches > 0 {
		assessment.failureDetails = append(assessment.failureDetails, combatReplayMismatchExamples(ctx.ReplayData, events)...)
	}
	return assessment
}

func combatContractViolationExamples(ctx *models.DemoContext, events []combat.Event, diagnostics combat.Diagnostics) []string {
	details := make([]string, 0, 8)
	add := func(event combat.Event, reason string) {
		if len(details) < 8 {
			details = append(details, fmt.Sprintf("combat_event=%s type=%s round=%d tick=%d: %s", event.LocalID, event.Type, event.Round, event.Tick, reason))
		}
	}
	participants := make(map[uint64]struct{})
	if ctx != nil && ctx.MatchData != nil {
		for playerID := range ctx.MatchData.Players {
			participants[playerID] = struct{}{}
		}
	}
	byID := make(map[string]combat.Event, len(events))
	for _, event := range events {
		byID[event.LocalID] = event
	}
	for _, event := range events {
		if event.LocalID == "" || event.Round <= 0 || event.Tick < 0 || !knownCombatEventType(event.Type) || event.Source != expectedCombatEventSource(event.Type) {
			add(event, "invalid identity, time, type, or callback source")
		}
		for _, role := range []struct {
			name   string
			player combat.PlayerRef
		}{{"actor", event.Actor}, {"target", event.Target}, {"assister", event.Assister}} {
			if combatPlayerContractViolations(role.player, participants) > 0 {
				add(event, fmt.Sprintf("invalid %s reference/status/provenance id=%d status=%s source=%s", role.name, role.player.ID, role.player.Status, role.player.Source))
			}
		}
		if !combatWeaponContractValid(event.Weapon) {
			add(event, fmt.Sprintf("invalid weapon availability/provenance status=%s source=%s name=%q", event.Weapon.Status, event.Weapon.Source, event.Weapon.Name))
		}
		if !combatAmmoContractValid(event.Ammo) {
			add(event, fmt.Sprintf("invalid ammo availability/provenance status=%s source=%s", event.Ammo.Status, event.Ammo.Source))
		}
		if !combatCorrelationValid(event) {
			add(event, fmt.Sprintf("invalid shot correlation status=%s source=%s shot=%q", event.CorrelationStatus, event.CorrelationSource, event.ShotID))
		}
		for _, sourceID := range event.SourceEventIDs {
			source, exists := byID[sourceID]
			if !exists || sourceID == event.LocalID || !validCombatSourceLink(source.Type, event.Type) || source.Round > event.Round || source.Round == event.Round && source.Tick > event.Tick {
				add(event, fmt.Sprintf("invalid causal source link %q", sourceID))
			}
		}
	}
	if diagnostics.InvalidLinks+diagnostics.FutureLinks+diagnostics.DuplicateLocalIDs+diagnostics.DuplicateShotIDs > 0 && len(details) < 8 {
		details = append(details, fmt.Sprintf("combat tracker diagnostics: invalid_links=%d future_links=%d duplicate_event_ids=%d duplicate_shot_ids=%d",
			diagnostics.InvalidLinks, diagnostics.FutureLinks, diagnostics.DuplicateLocalIDs, diagnostics.DuplicateShotIDs))
	}
	return details
}

func combatReplayMismatchExamples(replay *models.ReplayData, events []combat.Event) []string {
	wantMarkers := make(map[string]combat.Event)
	wantShots := make(map[string]combat.Event)
	for _, event := range events {
		switch event.Type {
		case combat.EventPlayerHurt, combat.EventKill:
			wantMarkers[event.LocalID] = event
		case combat.EventWeaponFire:
			wantShots[event.LocalID] = event
		}
	}
	markerCounts := make(map[string]int)
	shotCounts := make(map[string]int)
	if replay != nil {
		for _, round := range replay.Rounds {
			for _, marker := range round.Events {
				if len(marker.SourceEventIDs) == 1 {
					markerCounts[marker.SourceEventIDs[0]]++
				}
			}
			for _, shot := range round.CombatShots {
				shotCounts[shot.SourceEventID]++
			}
		}
	}
	details := make([]string, 0, 8)
	for _, event := range events {
		if len(details) >= 8 {
			break
		}
		if _, wanted := wantMarkers[event.LocalID]; wanted && markerCounts[event.LocalID] != 1 {
			details = append(details, fmt.Sprintf("replay marker source=%s type=%s round=%d tick=%d count=%d", event.LocalID, event.Type, event.Round, event.Tick, markerCounts[event.LocalID]))
		}
		if _, wanted := wantShots[event.LocalID]; wanted && shotCounts[event.LocalID] != 1 {
			details = append(details, fmt.Sprintf("replay combat_shot source=%s round=%d tick=%d count=%d", event.LocalID, event.Round, event.Tick, shotCounts[event.LocalID]))
		}
	}
	return details
}

func combatCallbackAccountingViolations(diagnostics combat.Diagnostics) int {
	violations := 0
	discardedByType := 0
	for _, eventType := range combatEventTypes() {
		observed := diagnostics.ObservedByType[eventType]
		recorded := diagnostics.RecordedByType[eventType]
		discarded := diagnostics.DiscardedByType[eventType]
		discardedByType += discarded
		if observed != recorded+discarded {
			violations++
		}
	}
	discardedByReason := 0
	for reason, count := range diagnostics.DiscardedByReason {
		discardedByReason += count
		if count < 0 || !knownCombatDiscardReason(reason) {
			violations++
		}
	}
	if discardedByType != discardedByReason {
		violations++
	}
	return violations
}

func combatDiscardedCallbackCount(diagnostics combat.Diagnostics) int {
	total := 0
	for _, count := range diagnostics.DiscardedByType {
		total += count
	}
	return total
}

func knownCombatDiscardReason(reason string) bool {
	switch reason {
	case combat.DiscardWarmup, combat.DiscardOutsideRound, combat.DiscardInvalidObservation:
		return true
	default:
		return false
	}
}

func cloneStringIntMap(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func combatContractViolations(ctx *models.DemoContext, events []combat.Event, diagnostics combat.Diagnostics) int {
	violations := diagnostics.InvalidLinks + diagnostics.FutureLinks + diagnostics.DuplicateLocalIDs + diagnostics.DuplicateShotIDs
	participants := make(map[uint64]struct{})
	if ctx.MatchData != nil {
		for playerID := range ctx.MatchData.Players {
			participants[playerID] = struct{}{}
		}
	}
	lastRound, lastTick, lastTickSequence, lastRoundSequence := 0, -1, 0, 0
	byID := make(map[string]combat.Event, len(events))
	shotFire := make(map[string]combat.Event)
	for _, event := range events {
		byID[event.LocalID] = event
		if event.Type == combat.EventWeaponFire {
			shotFire[event.ShotID] = event
		}
	}
	for _, event := range events {
		if event.LocalID == "" || event.Round <= 0 || event.Tick < 0 || !knownCombatEventType(event.Type) || event.Source != expectedCombatEventSource(event.Type) {
			violations++
		}
		if event.Round < lastRound || event.Round == lastRound && event.Tick < lastTick {
			violations++
		}
		if event.Round != lastRound {
			lastRoundSequence = 0
		}
		if event.Round != lastRound || event.Tick != lastTick {
			lastTickSequence = 0
		}
		if event.SequenceInTick != lastTickSequence+1 || event.SequenceInRound != lastRoundSequence+1 {
			violations++
		}
		lastRound, lastTick = event.Round, event.Tick
		lastTickSequence, lastRoundSequence = event.SequenceInTick, event.SequenceInRound
		violations += combatPlayerContractViolations(event.Actor, participants)
		violations += combatPlayerContractViolations(event.Target, participants)
		violations += combatPlayerContractViolations(event.Assister, participants)
		if !combatWeaponContractValid(event.Weapon) || !combatAmmoContractValid(event.Ammo) ||
			!combatAvailabilitySourceValid(event.ImpactPositionStatus, event.ImpactPositionSource, true) ||
			!combatAvailabilitySourceValid(event.DamageStatus, event.DamageSource, false) ||
			!combatAvailabilitySourceValid(event.HitgroupStatus, event.HitgroupSource, false) ||
			!combatAvailabilitySourceValid(event.ShotResultStatus, event.ShotResultSource, true) ||
			!combatCorrelationValid(event) {
			violations++
		}
		if expectedCombatRelation(event.Actor, event.Target) != event.Relation {
			violations++
		}
		if !combatEventFinite(event) {
			violations++
		}
		if event.Type == combat.EventPlayerHurt {
			if !validCombatDamage(event) || event.IsKill {
				violations++
			}
		} else if event.HealthDamage != nil || event.HealthDamageTaken != nil || event.ArmorDamage != nil || event.ArmorDamageTaken != nil {
			violations++
		}
		if event.Type == combat.EventKill && (!event.IsKill || event.DamageStatus != combat.AvailabilityUnavailable) {
			violations++
		}
		if event.Type == combat.EventWeaponFire {
			if event.ShotID == "" {
				violations++
			}
			if event.ShotResult == combat.ShotMiss && (event.ShotResultSource != combat.SourceRoundClosure ||
				event.ShotResultAvailabilityTick == nil || *event.ShotResultAvailabilityTick < event.Tick) {
				violations++
			}
		} else if event.ShotID != "" {
			fire, exists := shotFire[event.ShotID]
			if !exists || fire.Round > event.Round || fire.Round == event.Round && fire.Tick > event.Tick {
				violations++
			}
		}
		for _, sourceID := range event.SourceEventIDs {
			source, exists := byID[sourceID]
			if !exists || sourceID == event.LocalID || source.Round > event.Round || source.Round == event.Round && source.Tick > event.Tick ||
				!validCombatSourceLink(source.Type, event.Type) {
				violations++
			}
		}
	}
	return violations
}

func combatPlayerContractViolations(player combat.PlayerRef, participants map[uint64]struct{}) int {
	if player.Status == combat.AvailabilityObserved {
		if player.ID == 0 || player.Source == "" || player.Source == combat.SourceUnavailable {
			return 1
		}
		if _, exists := participants[player.ID]; !exists {
			return 1
		}
		if player.PositionStatus == combat.AvailabilityObserved {
			if player.Position == nil || player.PositionSource == "" || player.PositionSource == combat.SourceUnavailable {
				return 1
			}
		} else if player.PositionStatus != combat.AvailabilityUnavailable || player.Position != nil || player.PositionSource != combat.SourceUnavailable {
			return 1
		}
		return 0
	}
	if player.Status != combat.AvailabilityUnavailable || player.Source != combat.SourceUnavailable || player.ID != 0 || player.Position != nil ||
		player.PositionStatus != combat.AvailabilityUnavailable || player.PositionSource != combat.SourceUnavailable {
		return 1
	}
	return 0
}

func combatAvailabilitySourceValid(status combat.Availability, source string, allowDerived bool) bool {
	switch status {
	case combat.AvailabilityObserved:
		return source != "" && source != combat.SourceUnavailable
	case combat.AvailabilityDerived:
		return allowDerived && source != "" && source != combat.SourceUnavailable
	case combat.AvailabilityUnavailable:
		return source == combat.SourceUnavailable
	default:
		return false
	}
}

func combatWeaponContractValid(weapon combat.WeaponRef) bool {
	if weapon.Status == combat.AvailabilityObserved {
		return weapon.Name != "" && weapon.Source != "" && weapon.Source != combat.SourceUnavailable && weapon.IsUtility != nil
	}
	return weapon.Status == combat.AvailabilityUnavailable && weapon.Name == "" && weapon.Source == combat.SourceUnavailable && weapon.IsUtility == nil
}

func combatAmmoContractValid(ammo combat.AmmoObservation) bool {
	if ammo.Status == combat.AvailabilityObserved {
		return ammo.Source != "" && ammo.Source != combat.SourceUnavailable && (ammo.InMagazine != nil || ammo.Reserve != nil)
	}
	return ammo.Status == combat.AvailabilityUnavailable && ammo.Source == combat.SourceUnavailable && ammo.InMagazine == nil && ammo.Reserve == nil
}

func combatCorrelationValid(event combat.Event) bool {
	if event.CorrelationStatus == combat.CorrelationUnavailable {
		if event.CorrelationSource != combat.SourceUnavailable {
			return false
		}
		if event.Type == combat.EventWeaponFire {
			return event.ShotID != ""
		}
		return event.ShotID == ""
	}
	if (event.CorrelationStatus != combat.CorrelationExact && event.CorrelationStatus != combat.CorrelationInferred) ||
		event.CorrelationSource == "" || event.CorrelationSource == combat.SourceUnavailable {
		return false
	}
	if event.ShotID != "" {
		return true
	}
	return event.CorrelationStatus == combat.CorrelationExact && len(event.SourceEventIDs) > 0 &&
		(event.CorrelationSource == combat.SourceBulletCorrelation || event.CorrelationSource == combat.SourceFatalHurt)
}

func validCombatSourceLink(sourceType, targetType combat.EventType) bool {
	switch targetType {
	case combat.EventBulletDamage:
		return sourceType == combat.EventWeaponFire
	case combat.EventPlayerHurt:
		return sourceType == combat.EventWeaponFire || sourceType == combat.EventBulletDamage
	case combat.EventKill:
		return sourceType == combat.EventWeaponFire || sourceType == combat.EventPlayerHurt
	default:
		return false
	}
}

func expectedCombatEventSource(eventType combat.EventType) string {
	switch eventType {
	case combat.EventWeaponEquip:
		return combat.SourceItemEquip
	case combat.EventWeaponReload:
		return combat.SourceWeaponReload
	case combat.EventWeaponFire:
		return combat.SourceWeaponFire
	case combat.EventBulletDamage:
		return combat.SourceBulletDamage
	case combat.EventPlayerHurt:
		return combat.SourcePlayerHurt
	case combat.EventKill:
		return combat.SourceKill
	default:
		return ""
	}
}

func expectedCombatRelation(actor, target combat.PlayerRef) combat.Relation {
	if target.Status != combat.AvailabilityObserved {
		return combat.RelationUnknown
	}
	if actor.Status != combat.AvailabilityObserved {
		return combat.RelationWorld
	}
	if actor.ID == target.ID {
		return combat.RelationSelf
	}
	if actor.Side == "" || target.Side == "" {
		return combat.RelationUnknown
	}
	if actor.Side == target.Side {
		return combat.RelationFriendly
	}
	return combat.RelationEnemy
}

func validCombatDamage(event combat.Event) bool {
	values := []*int{event.HealthDamage, event.HealthDamageTaken, event.ArmorDamage, event.ArmorDamageTaken,
		event.HealthBefore, event.HealthAfter, event.ArmorBefore, event.ArmorAfter}
	for _, value := range values {
		if value == nil || *value < 0 {
			return false
		}
	}
	return event.DamageStatus == combat.AvailabilityObserved &&
		*event.HealthBefore >= *event.HealthAfter && *event.ArmorBefore >= *event.ArmorAfter &&
		*event.HealthDamageTaken == *event.HealthBefore-*event.HealthAfter &&
		*event.ArmorDamageTaken == *event.ArmorBefore-*event.ArmorAfter
}

func combatEventFinite(event combat.Event) bool {
	for _, value := range []*float64{event.ViewYaw, event.ViewPitch, event.BulletDistance, event.KillDistance} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return false
		}
	}
	for _, vector := range []*combat.Vector{event.Actor.Position, event.Target.Position, event.Assister.Position, event.ImpactPosition, event.DamageDirection} {
		if vector != nil && (!finiteCombatValue(vector.X) || !finiteCombatValue(vector.Y) || !finiteCombatValue(vector.Z)) {
			return false
		}
	}
	return true
}

func finiteCombatValue(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func combatStatsMismatches(ctx *models.DemoContext, events []combat.Event) (statsMismatches, nativeDeltaMismatches int) {
	expectedByPlayer := combat.Summaries(events)
	seen := make(map[uint64]struct{}, len(ctx.AI_PlayersSummary))
	for _, player := range ctx.AI_PlayersSummary {
		playerID, err := strconv.ParseUint(player.SteamID, 10, 64)
		if err != nil || playerID == 0 {
			if combatPlayerStatsContainData(player) {
				statsMismatches++
			}
			continue
		}
		if _, duplicate := seen[playerID]; duplicate {
			statsMismatches++
			continue
		}
		seen[playerID] = struct{}{}
		expected := expectedByPlayer[playerID]
		if player.KillsObserved != expected.Kills || player.DeathsObserved != expected.Deaths ||
			player.AssistsObserved != expected.Assists || player.FlashAssists != expected.FlashAssists ||
			player.Headshots != expected.Headshots || player.CombatDamageObserved != expected.EnemyDamage ||
			player.FriendlyDamage != expected.FriendlyDamage || player.SelfDamage != expected.SelfDamage ||
			player.ShotsFired != expected.ShotsFired || player.ShotsHit != expected.ShotsHit || player.ShotsMissed != expected.ShotsMissed ||
			!combatBodyPartHitsEqual(player.BodyPartHits, expected.BodyPartHits) || !combatWeaponStatsEqual(player.WeaponStats, expected.WeaponStats) {
			statsMismatches++
		}
		if player.KillsNativeMinusObserved != player.NativeScoreboard.Kills-expected.Kills ||
			player.DeathsNativeMinusObserved != player.NativeScoreboard.Deaths-expected.Deaths ||
			player.AssistsNativeMinusObserved != player.NativeScoreboard.Assists-expected.Assists ||
			player.CombatDamageUnattributedDelta != player.NativeScoreboard.TotalDamage-expected.EnemyDamage {
			nativeDeltaMismatches++
		}
	}
	for playerID := range expectedByPlayer {
		if _, exists := seen[playerID]; !exists {
			statsMismatches++
		}
	}
	return statsMismatches, nativeDeltaMismatches
}

func combatPlayerStatsContainData(player models.AI_PlayerStats) bool {
	return player.KillsObserved != 0 || player.DeathsObserved != 0 || player.AssistsObserved != 0 ||
		player.FlashAssists != 0 || player.Headshots != 0 || player.CombatDamageObserved != 0 ||
		player.FriendlyDamage != 0 || player.SelfDamage != 0 || player.ShotsFired != 0 ||
		player.ShotsHit != 0 || player.ShotsMissed != 0 || len(player.BodyPartHits) != 0 ||
		len(player.WeaponStats) != 0 || player.KillsNativeMinusObserved != 0 ||
		player.DeathsNativeMinusObserved != 0 || player.AssistsNativeMinusObserved != 0 ||
		player.CombatDamageUnattributedDelta != 0 || player.NativeScoreboard.Kills != 0 ||
		player.NativeScoreboard.Deaths != 0 || player.NativeScoreboard.Assists != 0 ||
		player.NativeScoreboard.TotalDamage != 0
}

func combatBodyPartHitsEqual(actual, expected map[string]int) bool {
	if len(actual) != len(expected) {
		return false
	}
	for hitgroup, want := range expected {
		if actual[hitgroup] != want {
			return false
		}
	}
	return true
}

func combatWeaponStatsEqual(actual map[string]models.AI_WeaponStat, expected map[string]combat.WeaponSummary) bool {
	if len(actual) != len(expected) {
		return false
	}
	for weapon, want := range expected {
		got, exists := actual[weapon]
		if !exists || got.Kills != want.Kills || got.Headshots != want.Headshots || got.Damage != want.Damage ||
			got.ShotsFired != want.ShotsFired || got.ShotsHit != want.ShotsHit || got.ShotsMissed != want.ShotsMissed {
			return false
		}
	}
	return true
}

func combatReplayMismatches(replay *models.ReplayData, events []combat.Event) int {
	wantMarkers := make(map[string]combat.Event)
	wantShots := make(map[string]combat.Event)
	for _, event := range events {
		switch event.Type {
		case combat.EventPlayerHurt, combat.EventKill:
			wantMarkers[event.LocalID] = event
		case combat.EventWeaponFire:
			wantShots[event.LocalID] = event
		}
	}
	if replay == nil {
		return len(wantMarkers) + len(wantShots)
	}
	markerCounts := make(map[string]int)
	shotCounts := make(map[string]int)
	mismatches := 0
	for _, round := range replay.Rounds {
		for _, event := range round.Events {
			if event.Type != "player_hurt" && event.Type != "kill" {
				continue
			}
			if len(event.SourceEventIDs) != 1 {
				mismatches++
				continue
			}
			sourceID := event.SourceEventIDs[0]
			source, exists := wantMarkers[sourceID]
			if !exists || source.Round != round.Round || string(source.Type) != event.Type {
				mismatches++
				continue
			}
			if !replayCombatMarkerMatches(event, source) {
				mismatches++
			}
			markerCounts[sourceID]++
		}
		for _, shot := range round.CombatShots {
			source, exists := wantShots[shot.SourceEventID]
			if !exists || source.Round != round.Round || !replayShotMatches(shot, source) {
				mismatches++
				continue
			}
			shotCounts[shot.SourceEventID]++
		}
		for _, frame := range round.Frames {
			for _, shot := range frame.Shots {
				source, exists := wantShots[shot.SourceEventID]
				if !exists || source.Round != round.Round || frame.Tick < shot.Tick || !replayShotMatches(shot, source) {
					mismatches++
				}
			}
		}
	}
	for sourceID := range wantMarkers {
		if markerCounts[sourceID] != 1 {
			mismatches++
		}
	}
	for sourceID := range wantShots {
		if shotCounts[sourceID] != 1 {
			mismatches++
		}
	}
	return mismatches
}

func replayShotMatches(shot models.ReplayShot, source combat.Event) bool {
	expectedWeapon := ""
	if source.Weapon.Status == combat.AvailabilityObserved {
		expectedWeapon = source.Weapon.Name
	}
	return source.ShotID == shot.ShotID && shot.Tick == source.Tick && shot.Weapon == expectedWeapon &&
		shot.ShooterID == source.Actor.ID && shot.Result == string(source.ShotResult) &&
		shot.ResultStatus == string(source.ShotResultStatus) &&
		shot.Hit == (source.ShotResult == combat.ShotHit) &&
		shot.PositionStatus == string(source.Actor.PositionStatus) &&
		shot.PositionSource == source.Actor.PositionSource
}

func replayCombatMarkerMatches(marker models.ReplayEvent, source combat.Event) bool {
	if marker.KillerID != source.Actor.ID || marker.VictimID != source.Target.ID || marker.AssisterID != source.Assister.ID ||
		marker.KillerName != source.Actor.Name || marker.VictimName != source.Target.Name || marker.AssisterName != source.Assister.Name ||
		marker.KillerTeam != source.Actor.Side || marker.VictimTeam != source.Target.Side || marker.AssisterTeam != source.Assister.Side {
		return false
	}
	expectedWeapon := ""
	if source.Weapon.Status == combat.AvailabilityObserved {
		expectedWeapon = source.Weapon.Name
	}
	if marker.Weapon != expectedWeapon || marker.Headshot != (source.IsHeadshot != nil && *source.IsHeadshot) {
		return false
	}
	if source.Type == combat.EventPlayerHurt {
		return source.HealthDamageTaken != nil && marker.Damage == *source.HealthDamageTaken
	}
	return marker.Wallbang == (source.PenetratedObjects != nil && *source.PenetratedObjects > 0) &&
		marker.NoScope == (source.NoScope != nil && *source.NoScope)
}

func combatDeterminismViolations(tracker *combat.Tracker) int {
	first, firstErr := json.Marshal(tracker.Snapshot())
	second, secondErr := json.Marshal(tracker.Snapshot())
	if firstErr != nil || secondErr != nil || string(first) != string(second) {
		return 1
	}
	return 0
}

func (assessment combatQualityAssessment) checks() []qualityCheck {
	checks := []qualityCheck{
		combatQualityCheck("combat_contract", assessment.contractViolations, "Atomic combat rows and causal references must satisfy contract v2."),
		combatQualityCheck("combat_callback_accounting", assessment.callbackAccountingViolations, "Every observed combat callback must be recorded exactly once without silent discard."),
		combatQualityCheck("combat_player_stats_projection", assessment.playerStatsMismatches, "Ledger-derived player and weapon statistics must reconcile exactly."),
		combatQualityCheck("combat_replay_projection", assessment.replayProjectionMismatches, "Replay hurt, kill and shot projections must reference the combat ledger exactly."),
		combatQualityCheck("combat_native_deltas", assessment.nativeDeltaMismatches, "Native-minus-observed deltas must reconcile without fabricated events."),
		combatQualityCheck("combat_determinism", assessment.determinismViolations, "Repeated combat snapshots must serialize identically."),
	}
	warningCount := assessment.missingImpactPositions + assessment.missingReloadEnds + assessment.unavailableShotResults + assessment.discardedCallbacks
	status := "pass"
	if warningCount > 0 {
		status = "warning"
	}
	checks = append(checks, qualityCheck{
		Name: "combat_observation_coverage", Status: status, Expected: "explicit availability",
		Actual:  fmt.Sprintf("impact_positions=%d,reload_ends=%d,shot_results=%d,discarded_callbacks=%d", assessment.missingImpactPositions, assessment.missingReloadEnds, assessment.unavailableShotResults, assessment.discardedCallbacks),
		Message: "Unavailable observations and callbacks outside the official round boundary are retained explicitly and never inferred as facts.",
	})
	return checks
}

func combatQualityCheck(name string, violations int, message string) qualityCheck {
	status := "pass"
	if violations > 0 {
		status = "fail"
	}
	return qualityCheck{Name: name, Status: status, Expected: "0", Actual: strconv.Itoa(violations), Message: message}
}

func (assessment combatQualityAssessment) callbackDiagnosticsReport() map[string]combatCallbackQualityDiagnostics {
	report := make(map[string]combatCallbackQualityDiagnostics)
	for _, eventType := range combatEventTypes() {
		report[string(eventType)] = combatCallbackQualityDiagnostics{
			Observed:  assessment.diagnostics.ObservedByType[eventType],
			Recorded:  assessment.diagnostics.RecordedByType[eventType],
			Discarded: assessment.diagnostics.DiscardedByType[eventType],
		}
	}
	return report
}

func combatFailureDetails(assessment combatQualityAssessment) []string {
	details := make([]string, 0, 6)
	values := []struct {
		name  string
		count int
	}{
		{"contract", assessment.contractViolations},
		{"callback_accounting", assessment.callbackAccountingViolations},
		{"player_stats", assessment.playerStatsMismatches},
		{"replay_projection", assessment.replayProjectionMismatches},
		{"native_delta", assessment.nativeDeltaMismatches},
		{"determinism", assessment.determinismViolations},
	}
	for _, value := range values {
		if value.count > 0 {
			details = append(details, fmt.Sprintf("%s=%d", value.name, value.count))
		}
	}
	return details
}

func combatEventTypes() []combat.EventType {
	result := []combat.EventType{
		combat.EventWeaponEquip, combat.EventWeaponReload, combat.EventWeaponFire,
		combat.EventBulletDamage, combat.EventPlayerHurt, combat.EventKill,
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

func knownCombatEventType(eventType combat.EventType) bool {
	for _, known := range combatEventTypes() {
		if eventType == known {
			return true
		}
	}
	return false
}
