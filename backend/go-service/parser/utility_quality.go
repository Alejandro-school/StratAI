package parser

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
	"fmt"
	"math"
	"sort"
	"strconv"
)

type utilityQualityStats struct {
	throws                            int
	canonicalEvents                   int
	flashEffects                      int
	damageEffects                     int
	contractViolations                int
	throwReconciliationMismatches     int
	playerStatsMismatches             int
	replayProjectionMismatches        int
	callbackAccountingViolations      int
	unmatchedCallbacks                int
	orphanCallbacks                   int
	inferredCallbacks                 int
	deduplicatedCallbacks             int
	lifecycleViolations               int
	flashAttributionMismatches        int
	damageReconciliationMismatches    int
	temporalSpatialViolations         int
	determinismViolations             int
	observationWarnings               int
	missingTypeObservations           int
	missingActorObservations          int
	missingAffectedPlayerObservations int
	missingFlashDurationObservations  int
	missingLaunchTicks                int
	missingLaunchPositions            int
	missingLaunchViews                int
	missingThrowerVelocities          int
	missingProjectileVelocities       int
	missingTrajectoryObservations     int
	missingLifecycleObservations      int
	inferredCorrelations              int
	observedEffectCorrelations        int
	inferredEffectCorrelations        int
	unavailableEffectCorrelations     int
	diagnostics                       utility.Diagnostics
	failureDetails                    []string
}

func assessUtilityQuality(ctx *models.DemoContext) utilityQualityStats {
	stats := utilityQualityStats{failureDetails: make([]string, 0)}
	if ctx.Utilities == nil {
		stats.contractViolations = 1
		stats.addFailure("utility tracker is unavailable")
		return stats
	}
	throws := ctx.Utilities.Snapshot()
	stats.throws = len(throws)
	stats.canonicalEvents = len(projectCanonicalUtilityThrows("quality", throws))
	stats.assessDiagnostics(ctx.Utilities.Diagnostics())
	stats.assessDiagnosticThrowReconciliation(throws)
	stats.assessDeterminism(throws)
	stats.assessThrowReconciliation(throws, ctx.AI_GrenadeEvents)
	stats.assessPlayerStats(throws, ctx.AI_PlayersSummary)
	stats.assessReplayProjection(throws, ctx.ReplayData)
	bounds := utilityReplayBounds(ctx.ReplayData)
	throwByID := make(map[string]utility.Throw, len(throws))
	for _, throw := range throws {
		throwByID[throw.ID] = throw
		stats.assessThrow(throw, bounds[throw.Round])
	}
	stats.assessExtinguishLinks(throws, throwByID)
	return stats
}

type expectedUtilityReplayMarker struct {
	round int
	event models.ReplayEvent
}

func (stats *utilityQualityStats) assessReplayProjection(throws []utility.Throw, replay *models.ReplayData) {
	expected := make(map[string]expectedUtilityReplayMarker)
	for _, throw := range throws {
		moment, ok := utilityReplayEffectMoment(throw.Lifecycle)
		if !ok {
			continue
		}
		expected[throw.ID] = expectedUtilityReplayMarker{round: throw.Round, event: expectedReplayUtilityMarker(throw, moment)}
	}
	seen := make(map[string]struct{}, len(expected))
	if replay != nil {
		for _, round := range replay.Rounds {
			for _, marker := range round.Events {
				if marker.Type != "utility_detonate" {
					continue
				}
				expectedMarker, exists := expected[marker.SourceThrowID]
				if !exists {
					stats.replayProjectionMismatches++
					stats.addFailure(fmt.Sprintf("unexpected replay utility marker %q", marker.SourceThrowID))
					continue
				}
				if _, duplicate := seen[marker.SourceThrowID]; duplicate {
					stats.replayProjectionMismatches++
					stats.addFailure(fmt.Sprintf("duplicate replay utility marker %q", marker.SourceThrowID))
					continue
				}
				seen[marker.SourceThrowID] = struct{}{}
				if round.Round != expectedMarker.round || !utilityReplayMarkersEqual(marker, expectedMarker.event) {
					stats.replayProjectionMismatches++
					stats.addFailure(fmt.Sprintf("replay utility marker mismatch for %q", marker.SourceThrowID))
				}
			}
		}
	}
	for throwID := range expected {
		if _, exists := seen[throwID]; !exists {
			stats.replayProjectionMismatches++
			stats.addFailure(fmt.Sprintf("replay utility marker missing for %q", throwID))
		}
	}
}

func utilityReplayEffectMoment(lifecycle utility.Lifecycle) (utility.TickPositionObservation, bool) {
	if lifecycle.EffectStart.Status == utility.AvailabilityObserved {
		return lifecycle.EffectStart, true
	}
	if lifecycle.Detonation.Status == utility.AvailabilityObserved {
		return lifecycle.Detonation, true
	}
	return utility.TickPositionObservation{}, false
}

func expectedReplayUtilityMarker(throw utility.Throw, moment utility.TickPositionObservation) models.ReplayEvent {
	marker := models.ReplayEvent{
		ID: "utility:" + throw.ID, Tick: moment.Tick, Type: "utility_detonate",
		GrenadeType: string(throw.Type), UtilityType: string(throw.Type),
		PlayerID: throw.Actor.ID, ActorID: throw.Actor.ID,
		PositionStatus: string(moment.PositionStatus), PositionSource: utility.SourceUnavailable,
		CorrelationStatus: string(throw.Lifecycle.Correlation.Status),
		CorrelationSource: throw.Lifecycle.Correlation.Source,
		SourceThrowID:     throw.ID,
	}
	if moment.PositionStatus == utility.AvailabilityObserved {
		marker.X, marker.Y, marker.Z = moment.Position.X, moment.Position.Y, moment.Position.Z
		marker.PositionSource = moment.Source
	}
	marker.AffectedPlayerIDs = expectedReplayAffectedPlayerIDs(throw.Flashes)
	for _, effect := range throw.Damage {
		marker.Damage += max(0, effect.HealthDamage)
	}
	marker.DurationMS, marker.DurationStatus, marker.DurationSource = expectedReplayUtilityDuration(throw)
	return marker
}

func expectedReplayAffectedPlayerIDs(effects []utility.FlashEffect) []string {
	seen := make(map[uint64]struct{}, len(effects))
	for _, effect := range effects {
		if effect.Victim.Status == utility.AvailabilityObserved && effect.Victim.ID != 0 {
			seen[effect.Victim.ID] = struct{}{}
		}
	}
	ids := make([]uint64, 0, len(seen))
	for playerID := range seen {
		ids = append(ids, playerID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]string, 0, len(ids))
	for _, playerID := range ids {
		result = append(result, strconv.FormatUint(playerID, 10))
	}
	return result
}

func expectedReplayUtilityDuration(throw utility.Throw) (int, string, string) {
	if throw.Lifecycle.Duration.Status == utility.AvailabilityObserved {
		return int(math.Round(throw.Lifecycle.Duration.Value * 1000)),
			string(utility.AvailabilityObserved), throw.Lifecycle.Duration.Source
	}
	maximum := 0.0
	source := utility.SourceUnavailable
	for _, effect := range throw.Flashes {
		if effect.Duration.Status == utility.AvailabilityObserved && effect.Duration.Value >= maximum {
			maximum, source = effect.Duration.Value, effect.Duration.Source
		}
	}
	if source != utility.SourceUnavailable {
		return int(math.Round(maximum * 1000)), string(utility.AvailabilityObserved), source
	}
	return 0, string(utility.AvailabilityUnavailable), utility.SourceUnavailable
}

func utilityReplayMarkersEqual(actual, expected models.ReplayEvent) bool {
	return actual.ID == expected.ID && actual.Tick == expected.Tick && actual.Type == expected.Type &&
		actual.GrenadeType == expected.GrenadeType && actual.UtilityType == expected.UtilityType &&
		actual.PlayerID == expected.PlayerID && actual.ActorID == expected.ActorID &&
		actual.X == expected.X && actual.Y == expected.Y && actual.Z == expected.Z &&
		actual.PositionStatus == expected.PositionStatus && actual.PositionSource == expected.PositionSource &&
		actual.CorrelationStatus == expected.CorrelationStatus && actual.CorrelationSource == expected.CorrelationSource &&
		actual.SourceThrowID == expected.SourceThrowID && actual.Damage == expected.Damage &&
		actual.DurationMS == expected.DurationMS && actual.DurationStatus == expected.DurationStatus &&
		actual.DurationSource == expected.DurationSource && utilityStringSlicesEqual(actual.AffectedPlayerIDs, expected.AffectedPlayerIDs)
}

func utilityStringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

type utilityPlayerStatsProjection struct {
	grenadesThrown          int
	flashesThrown           int
	heThrown                int
	smokesThrown            int
	molotovsThrown          int
	molotovGrenadesThrown   int
	incendiariesThrown      int
	decoysThrown            int
	enemiesFlashed          int
	teammatesFlashed        int
	selfFlashed             int
	enemyFlashDurationMS    float64
	teammateFlashDurationMS float64
	selfFlashDurationMS     float64
	utilityDamageObserved   int
	heDamageObserved        int
	molotovDamageObserved   int
	unknownDamageObserved   int
}

func (stats *utilityQualityStats) assessPlayerStats(throws []utility.Throw, players []models.AI_PlayerStats) {
	expected := make(map[uint64]*utilityPlayerStatsProjection)
	for _, throw := range throws {
		if throw.Actor.Status != utility.AvailabilityObserved || throw.Actor.ID == 0 {
			continue
		}
		projection := expected[throw.Actor.ID]
		if projection == nil {
			projection = &utilityPlayerStatsProjection{}
			expected[throw.Actor.ID] = projection
		}
		if throw.Launch.Tick.Status == utility.AvailabilityObserved {
			projection.grenadesThrown++
			switch throw.Type {
			case utility.TypeFlashbang:
				projection.flashesThrown++
			case utility.TypeHE:
				projection.heThrown++
			case utility.TypeSmoke:
				projection.smokesThrown++
			case utility.TypeMolotov:
				projection.molotovsThrown++
				projection.molotovGrenadesThrown++
			case utility.TypeIncendiary:
				projection.molotovsThrown++
				projection.incendiariesThrown++
			case utility.TypeDecoy:
				projection.decoysThrown++
			}
		}
		for _, effect := range throw.Flashes {
			durationMS := 0.0
			if effect.Duration.Status == utility.AvailabilityObserved {
				durationMS = effect.Duration.Value * 1000
			}
			switch effect.Relation {
			case utility.RelationEnemy:
				projection.enemiesFlashed++
				projection.enemyFlashDurationMS += durationMS
			case utility.RelationTeammate:
				projection.teammatesFlashed++
				projection.teammateFlashDurationMS += durationMS
			case utility.RelationSelf:
				projection.selfFlashed++
				projection.selfFlashDurationMS += durationMS
			}
		}
		for _, effect := range throw.Damage {
			if effect.Relation != utility.RelationEnemy {
				continue
			}
			damage := max(0, effect.HealthDamage)
			switch throw.Type {
			case utility.TypeHE:
				projection.heDamageObserved += damage
			case utility.TypeMolotov, utility.TypeIncendiary:
				projection.molotovDamageObserved += damage
			case utility.TypeUnknown:
				projection.unknownDamageObserved += damage
			default:
				continue
			}
			projection.utilityDamageObserved += damage
		}
	}

	seen := make(map[uint64]struct{}, len(players))
	for _, player := range players {
		playerID, err := strconv.ParseUint(player.SteamID, 10, 64)
		if err != nil || playerID == 0 {
			if utilityPlayerStatsContainData(player) {
				stats.playerStatsMismatches++
				stats.addFailure(fmt.Sprintf("utility player stats have invalid steam_id %q", player.SteamID))
			}
			continue
		}
		if _, duplicate := seen[playerID]; duplicate {
			stats.playerStatsMismatches++
			stats.addFailure(fmt.Sprintf("duplicate utility player stats for %d", playerID))
			continue
		}
		seen[playerID] = struct{}{}
		projection := expected[playerID]
		if projection == nil {
			projection = &utilityPlayerStatsProjection{}
		}
		if !utilityPlayerStatsEqual(player, *projection) {
			stats.playerStatsMismatches++
			stats.addFailure(fmt.Sprintf("utility player stats mismatch for %d", playerID))
		}
		delete(expected, playerID)
	}
	for playerID := range expected {
		stats.playerStatsMismatches++
		stats.addFailure(fmt.Sprintf("utility player stats missing for %d", playerID))
	}
}

func utilityPlayerStatsContainData(player models.AI_PlayerStats) bool {
	return player.GrenadesThrownTotal != 0 || player.FlashesThrown != 0 || player.HEThrown != 0 ||
		player.SmokesThrown != 0 || player.MolotovsThrown != 0 || player.MolotovGrenadesThrown != 0 ||
		player.IncendiariesThrown != 0 || player.DecoysThrown != 0 || player.EnemiesFlashedTotal != 0 ||
		player.TeammatesFlashedTotal != 0 || player.SelfFlashesTotal != 0 || player.EnemyFlashDurationTotalMS != 0 ||
		player.TeammateFlashDurationTotalMS != 0 || player.SelfFlashDurationTotalMS != 0 || player.FlashDurationTotal != 0 ||
		player.UtilityDamageObserved != 0 || player.HEDamagePerNade != 0 || player.MolotovDamagePerNade != 0 ||
		len(player.GrenadeDamage) != 0
}

func utilityPlayerStatsEqual(player models.AI_PlayerStats, expected utilityPlayerStatsProjection) bool {
	expectedEnemiesPerFlash, expectedBlindTimePerFlash := 0.0, 0.0
	if expected.flashesThrown > 0 {
		expectedEnemiesPerFlash = float64(expected.enemiesFlashed) / float64(expected.flashesThrown)
		expectedBlindTimePerFlash = expected.enemyFlashDurationMS / 1000 / float64(expected.flashesThrown)
	}
	expectedHEDamagePerNade, expectedMolotovDamagePerNade := 0.0, 0.0
	if expected.heThrown > 0 {
		expectedHEDamagePerNade = float64(expected.heDamageObserved) / float64(expected.heThrown)
	}
	if expected.molotovsThrown > 0 {
		expectedMolotovDamagePerNade = float64(expected.molotovDamageObserved) / float64(expected.molotovsThrown)
	}
	return player.GrenadesThrownTotal == expected.grenadesThrown && player.FlashesThrown == expected.flashesThrown &&
		player.HEThrown == expected.heThrown && player.SmokesThrown == expected.smokesThrown &&
		player.MolotovsThrown == expected.molotovsThrown && player.MolotovGrenadesThrown == expected.molotovGrenadesThrown &&
		player.IncendiariesThrown == expected.incendiariesThrown && player.DecoysThrown == expected.decoysThrown &&
		player.EnemiesFlashedTotal == expected.enemiesFlashed && player.TeammatesFlashedTotal == expected.teammatesFlashed &&
		player.SelfFlashesTotal == expected.selfFlashed &&
		math.Abs(player.EnemyFlashDurationTotalMS-expected.enemyFlashDurationMS) <= 1e-6 &&
		math.Abs(player.TeammateFlashDurationTotalMS-expected.teammateFlashDurationMS) <= 1e-6 &&
		math.Abs(player.SelfFlashDurationTotalMS-expected.selfFlashDurationMS) <= 1e-6 &&
		math.Abs(player.FlashDurationTotal-expected.enemyFlashDurationMS/1000) <= 1e-6 &&
		math.Abs(player.EnemiesFlashedPerFlash-expectedEnemiesPerFlash) <= 1e-6 &&
		math.Abs(player.BlindTimePerFlash-expectedBlindTimePerFlash) <= 1e-6 &&
		player.UtilityDamageObserved == expected.utilityDamageObserved &&
		utilityGrenadeDamageEqual(player.GrenadeDamage, expected) &&
		math.Abs(player.HEDamagePerNade-expectedHEDamagePerNade) <= 1e-6 &&
		math.Abs(player.MolotovDamagePerNade-expectedMolotovDamagePerNade) <= 1e-6
}

func utilityGrenadeDamageEqual(actual map[string]int, expected utilityPlayerStatsProjection) bool {
	if actual["he"] != expected.heDamageObserved || actual["molotov"] != expected.molotovDamageObserved ||
		actual["unknown"] != expected.unknownDamageObserved {
		return false
	}
	for grenadeType := range actual {
		if grenadeType != "he" && grenadeType != "molotov" && grenadeType != "unknown" {
			return false
		}
	}
	return true
}

func (stats *utilityQualityStats) assessDiagnostics(diagnostics utility.Diagnostics) {
	stats.diagnostics = diagnostics
	groups := []struct {
		name  string
		value utility.CallbackDiagnostics
	}{
		{name: "throws", value: diagnostics.Throws},
		{name: "bounces", value: diagnostics.Bounces},
		{name: "lifecycle", value: diagnostics.Lifecycle},
		{name: "flashes", value: diagnostics.Flashes},
		{name: "damage", value: diagnostics.Damage},
	}
	for _, group := range groups {
		accounted := group.value.ExactCorrelated + group.value.InferredCorrelated + group.value.Orphaned +
			group.value.Deduplicated + group.value.Unmatched
		if group.value.Observed < 0 || accounted != group.value.Observed {
			stats.callbackAccountingViolations++
			stats.contractViolations++
			stats.addFailure(fmt.Sprintf("utility callback accounting mismatch for %s: observed=%d accounted=%d", group.name, group.value.Observed, accounted))
		}
		stats.unmatchedCallbacks += group.value.Unmatched
		stats.orphanCallbacks += group.value.Orphaned
		stats.inferredCallbacks += group.value.InferredCorrelated
		stats.deduplicatedCallbacks += group.value.Deduplicated
	}
	if stats.unmatchedCallbacks > 0 {
		stats.throwReconciliationMismatches += stats.unmatchedCallbacks
		stats.addFailure(fmt.Sprintf("%d utility callbacks were not represented", stats.unmatchedCallbacks))
	}
}

func (stats *utilityQualityStats) assessDiagnosticThrowReconciliation(throws []utility.Throw) {
	launchObserved := 0
	for _, throw := range throws {
		if throw.Launch.Tick.Status == utility.AvailabilityObserved {
			launchObserved++
		}
	}
	expected := stats.diagnostics.Throws.Observed - stats.diagnostics.Throws.Deduplicated - stats.diagnostics.Throws.Unmatched
	if expected < 0 || launchObserved != expected {
		stats.throwReconciliationMismatches += absoluteDifference(launchObserved, expected)
		if expected < 0 {
			stats.throwReconciliationMismatches++
		}
		stats.addFailure(fmt.Sprintf("utility throw callbacks=%d launch_observed=%d deduplicated=%d unmatched=%d",
			stats.diagnostics.Throws.Observed, launchObserved, stats.diagnostics.Throws.Deduplicated, stats.diagnostics.Throws.Unmatched))
	}
}

type utilityCallbackQualityDiagnostics struct {
	Observed           int `json:"observed"`
	ExactCorrelated    int `json:"exact_correlated"`
	InferredCorrelated int `json:"inferred_correlated"`
	Orphaned           int `json:"orphaned"`
	Deduplicated       int `json:"deduplicated"`
	Unmatched          int `json:"unmatched"`
}

func (stats utilityQualityStats) callbackDiagnosticsReport() map[string]utilityCallbackQualityDiagnostics {
	return map[string]utilityCallbackQualityDiagnostics{
		"throws":         utilityCallbackQuality(stats.diagnostics.Throws),
		"bounces":        utilityCallbackQuality(stats.diagnostics.Bounces),
		"lifecycle":      utilityCallbackQuality(stats.diagnostics.Lifecycle),
		"player_flashed": utilityCallbackQuality(stats.diagnostics.Flashes),
		"damage":         utilityCallbackQuality(stats.diagnostics.Damage),
	}
}

func utilityCallbackQuality(source utility.CallbackDiagnostics) utilityCallbackQualityDiagnostics {
	return utilityCallbackQualityDiagnostics{
		Observed: source.Observed, ExactCorrelated: source.ExactCorrelated,
		InferredCorrelated: source.InferredCorrelated, Orphaned: source.Orphaned,
		Deduplicated: source.Deduplicated, Unmatched: source.Unmatched,
	}
}

func (stats *utilityQualityStats) assessThrow(throw utility.Throw, bounds *models.ReplayRound) {
	stats.flashEffects += len(throw.Flashes)
	stats.damageEffects += len(throw.Damage)
	stats.contractViolations += utilityThrowContractViolations(throw)
	stats.lifecycleViolations += utilityLifecycleViolations(throw)
	stats.flashAttributionMismatches += utilityFlashAttributionMismatches(throw)
	stats.damageReconciliationMismatches += utilityDamageMismatches(throw)
	stats.temporalSpatialViolations += utilityTemporalSpatialViolations(throw, bounds)
	stats.countObservationWarnings(throw)
}

func (stats *utilityQualityStats) assessDeterminism(throws []utility.Throw) {
	seenIDs := make(map[string]struct{}, len(throws))
	lastRound, lastSequence := 0, 0
	for index, throw := range throws {
		expectedID := fmt.Sprintf("r%d-u%04d", throw.Round, throw.Sequence)
		if throw.ID == "" || throw.ID != expectedID {
			stats.determinismViolations++
			stats.addFailure(fmt.Sprintf("utility throw %d has unstable id %q (expected %q)", index, throw.ID, expectedID))
		}
		if _, duplicate := seenIDs[throw.ID]; duplicate {
			stats.determinismViolations++
			stats.addFailure(fmt.Sprintf("duplicate utility throw id %q", throw.ID))
		}
		seenIDs[throw.ID] = struct{}{}
		if index > 0 && (throw.Round < lastRound || throw.Round == lastRound && throw.Sequence <= lastSequence) {
			stats.determinismViolations++
			stats.addFailure(fmt.Sprintf("utility throws are not ordered at %q", throw.ID))
		}
		if throw.Round != lastRound {
			if throw.Sequence != 1 {
				stats.determinismViolations++
			}
			lastSequence = 0
		} else if throw.Sequence != lastSequence+1 {
			stats.determinismViolations++
		}
		if !utilityTrajectorySorted(throw.Trajectory, throw.Lifecycle) || !utilityFlashEffectsSorted(throw.Flashes) || !utilityDamageEffectsSorted(throw.Damage) {
			stats.determinismViolations++
			stats.addFailure(fmt.Sprintf("utility effects are not deterministically ordered for %q", throw.ID))
		}
		lastRound, lastSequence = throw.Round, throw.Sequence
	}
}

func (stats *utilityQualityStats) assessThrowReconciliation(throws []utility.Throw, legacy []models.AI_GrenadeEvent) {
	if len(throws) != len(legacy) {
		stats.throwReconciliationMismatches += absoluteDifference(len(throws), len(legacy))
		stats.addFailure(fmt.Sprintf("utility throws=%d legacy_events=%d", len(throws), len(legacy)))
	}
	limit := len(throws)
	if len(legacy) < limit {
		limit = len(legacy)
	}
	for index := 0; index < limit; index++ {
		expected := projectLegacyUtilityDetails(throws[index])
		if !legacyUtilityCoreEqual(expected, legacy[index]) {
			stats.throwReconciliationMismatches++
			stats.addFailure(fmt.Sprintf("legacy utility projection mismatch at index %d", index))
		}
	}
}

func (stats *utilityQualityStats) assessExtinguishLinks(throws []utility.Throw, byID map[string]utility.Throw) {
	for _, throw := range throws {
		lifecycle := throw.Lifecycle
		if lifecycle.EndReason != utility.EndReasonSmokeExtinguished {
			continue
		}
		smokeID := lifecycle.ExtinguishedByThrowID.Value
		smoke, exists := byID[smokeID]
		if !exists || smoke.Type != utility.TypeSmoke || smoke.ID == throw.ID {
			stats.lifecycleViolations++
			stats.addFailure(fmt.Sprintf("utility %q has invalid extinguishing smoke %q", throw.ID, smokeID))
			continue
		}
		if lifecycle.Extinguish.Status == utility.AvailabilityObserved && smoke.Launch.Tick.Status == utility.AvailabilityObserved &&
			smoke.Launch.Tick.Tick > lifecycle.Extinguish.Tick {
			stats.lifecycleViolations++
			stats.addFailure(fmt.Sprintf("utility %q is extinguished before smoke %q was thrown", throw.ID, smoke.ID))
		}
	}
}

func (stats *utilityQualityStats) countObservationWarnings(throw utility.Throw) {
	if throw.Type == utility.TypeUnknown || throw.TypeSource == utility.SourceUnavailable {
		stats.missingTypeObservations++
	}
	if throw.Actor.Status != utility.AvailabilityObserved || normalizeSide(throw.Actor.Side) == "unknown" {
		stats.missingActorObservations++
	}
	for _, effect := range throw.Flashes {
		if effect.Victim.Status != utility.AvailabilityObserved {
			stats.missingAffectedPlayerObservations++
		}
		stats.countEffectCorrelation(effect.Correlation)
		if effect.Duration.Status != utility.AvailabilityObserved {
			stats.missingFlashDurationObservations++
		}
	}
	for _, effect := range throw.Damage {
		if effect.Victim.Status != utility.AvailabilityObserved {
			stats.missingAffectedPlayerObservations++
		}
		stats.countEffectCorrelation(effect.Correlation)
	}
	if throw.Launch.Tick.Status != utility.AvailabilityObserved {
		stats.missingLaunchTicks++
	}
	if throw.Launch.Position.Status != utility.AvailabilityObserved {
		stats.missingLaunchPositions++
	}
	if throw.Launch.View.Status != utility.AvailabilityObserved {
		stats.missingLaunchViews++
	}
	if throw.Launch.ThrowerVelocity.Status != utility.AvailabilityObserved {
		stats.missingThrowerVelocities++
	}
	if throw.Launch.ProjectileInitialVelocity.Status != utility.AvailabilityObserved {
		stats.missingProjectileVelocities++
	}
	if string(throw.Trajectory.Status) != string(utility.AvailabilityObserved) {
		stats.missingTrajectoryObservations++
	}
	if utilityExpectedLifecycleObservationMissing(throw) || utilityLifecyclePositionCoverageMissing(throw.Lifecycle) ||
		throw.Lifecycle.Area.Status != utility.AvailabilityObserved ||
		throw.Lifecycle.Correlation.Status == utility.CorrelationUnavailable {
		stats.missingLifecycleObservations++
	}
	if throw.Lifecycle.Correlation.Status == utility.CorrelationInferred {
		stats.inferredCorrelations++
	}
	if throw.Lifecycle.ExtinguishAttribution.Status == utility.CorrelationInferred {
		stats.inferredCorrelations++
	}
	stats.observationWarnings = stats.missingTypeObservations + stats.missingActorObservations + stats.missingAffectedPlayerObservations +
		stats.missingFlashDurationObservations +
		stats.missingLaunchTicks + stats.missingLaunchPositions + stats.missingLaunchViews + stats.missingThrowerVelocities +
		stats.missingProjectileVelocities +
		stats.missingTrajectoryObservations + stats.missingLifecycleObservations + stats.inferredCorrelations +
		stats.inferredEffectCorrelations + stats.unavailableEffectCorrelations
	stats.observationWarnings += stats.orphanCallbacks + stats.inferredCallbacks
}

func (stats *utilityQualityStats) countEffectCorrelation(correlation utility.Correlation) {
	switch correlation.Status {
	case utility.CorrelationObserved:
		stats.observedEffectCorrelations++
	case utility.CorrelationInferred:
		stats.inferredEffectCorrelations++
	default:
		stats.unavailableEffectCorrelations++
	}
}

func (stats *utilityQualityStats) addFailure(detail string) {
	const maxFailureDetails = 64
	if len(stats.failureDetails) < maxFailureDetails {
		stats.failureDetails = append(stats.failureDetails, detail)
	}
}

func (stats utilityQualityStats) hasHardFailure() bool {
	return stats.contractViolations > 0 || stats.throwReconciliationMismatches > 0 ||
		stats.playerStatsMismatches > 0 || stats.replayProjectionMismatches > 0 ||
		stats.lifecycleViolations > 0 || stats.flashAttributionMismatches > 0 ||
		stats.damageReconciliationMismatches > 0 || stats.temporalSpatialViolations > 0 ||
		stats.determinismViolations > 0
}

func (stats utilityQualityStats) checks() []qualityCheck {
	checks := []qualityCheck{
		utilityCountQualityCheck("utility_event_contract", stats.contractViolations, "Utility events must satisfy the closed causal contract."),
		utilityCountQualityCheck("utility_throw_reconciliation", stats.throwReconciliationMismatches+stats.playerStatsMismatches+stats.replayProjectionMismatches, "Every tracked throw must have exactly one compatible legacy/canonical/replay projection and reconcile with derived player statistics."),
		utilityCountQualityCheck("utility_lifecycle", stats.lifecycleViolations, "Utility lifecycle states and causal links must be internally consistent."),
		utilityCountQualityCheck("utility_flash_attribution", stats.flashAttributionMismatches, "Flash effects must come from exact PlayerFlashed callbacks with correct relations."),
		utilityCountQualityCheck("utility_damage_reconciliation", stats.damageReconciliationMismatches, "Utility damage events and their aggregates must reconcile exactly."),
		utilityCountQualityCheck("utility_temporal_spatial_consistency", stats.temporalSpatialViolations, "Utility ticks and positions must be finite, causal and inside their rounds."),
		utilityCountQualityCheck("utility_determinism", stats.determinismViolations, "Utility identities and collections must have a stable total order."),
	}
	coverage := qualityCheck{
		Name: "utility_observation_coverage", Status: "pass", Expected: "0",
		Actual:  fmt.Sprint(stats.observationWarnings),
		Message: "Unavailable, partial or inferred utility observations remain explicit and usable with caution.",
	}
	if stats.observationWarnings > 0 {
		coverage.Status = "warning"
	}
	return append(checks, coverage)
}

func utilityCountQualityCheck(name string, actual int, message string) qualityCheck {
	status := "pass"
	if actual > 0 {
		status = "fail"
	}
	return qualityCheck{Name: name, Status: status, Expected: "0", Actual: fmt.Sprint(actual), Message: message}
}

func utilityThrowContractViolations(throw utility.Throw) int {
	violations := 0
	if throw.Round < 1 || throw.Sequence < 1 || !validUtilityType(throw.Type) || !validUtilityTypeSource(throw.Type, throw.TypeSource) {
		violations++
	}
	if !validUtilityEntityObservation(throw) || !validUtilityThrowerRef(throw.Actor) ||
		!validUtilityTickObservation(throw.Launch.Tick) || !validUtilityVectorObservation(throw.Launch.Position) ||
		!validUtilityViewObservation(throw.Launch.View) || !validUtilityVelocityObservation(throw.Launch.ThrowerVelocity, false) ||
		!validUtilityVelocityObservation(throw.Launch.ProjectileInitialVelocity, true) ||
		!validUtilityStanceObservation(throw.Launch.Stance) || !validUtilityStringObservation(throw.Launch.Area) {
		violations++
	}
	if !validUtilityTrajectory(throw.Trajectory, throw.Lifecycle) || !validUtilityLifecycleObservations(throw.Lifecycle) {
		violations++
	}
	return violations
}

func utilityLifecycleViolations(throw utility.Throw) int {
	lifecycle := throw.Lifecycle
	violations := 0
	isInstant := throw.Type == utility.TypeFlashbang || throw.Type == utility.TypeHE
	isPersistent := throw.Type == utility.TypeSmoke || throw.Type == utility.TypeDecoy ||
		throw.Type == utility.TypeMolotov || throw.Type == utility.TypeIncendiary
	isUnavailableType := throw.Type == utility.TypeUnknown && throw.TypeSource == utility.SourceUnavailable
	switch lifecycle.Status {
	case utility.LifecycleThrown:
		if lifecycle.Detonation.Status == utility.AvailabilityObserved || lifecycle.EffectStart.Status == utility.AvailabilityObserved {
			violations++
		}
	case utility.LifecycleDetonated:
		if !isInstant || lifecycle.Detonation.Status != utility.AvailabilityObserved {
			violations++
		}
	case utility.LifecycleEffectActive:
		if (!isPersistent && !isUnavailableType) || lifecycle.EffectStart.Status != utility.AvailabilityObserved {
			violations++
		}
	case utility.LifecycleEffectExpired:
		if (!isPersistent && !isUnavailableType) || lifecycle.Expiration.Status != utility.AvailabilityObserved {
			violations++
		}
	case utility.LifecycleDestroyedWithoutDetonation:
		if lifecycle.Destroy.Status != utility.AvailabilityObserved || lifecycle.Detonation.Status == utility.AvailabilityObserved {
			violations++
		}
	case utility.LifecycleRoundEndedUnresolved:
		if lifecycle.EndReason != utility.EndReasonRoundEnd {
			violations++
		}
	default:
		violations++
	}
	if isInstant && (lifecycle.EffectStart.Status == utility.AvailabilityObserved ||
		lifecycle.Expiration.Status == utility.AvailabilityObserved || lifecycle.Duration.Status == utility.AvailabilityObserved) {
		violations++
	}
	if lifecycle.Duration.Status == utility.AvailabilityObserved &&
		(lifecycle.EffectStart.Status != utility.AvailabilityObserved || lifecycle.Expiration.Status != utility.AvailabilityObserved) {
		violations++
	}
	switch lifecycle.EndReason {
	case utility.EndReasonExpired:
		if (!isPersistent && !isUnavailableType) || lifecycle.Expiration.Status != utility.AvailabilityObserved || utilityHasExtinguishAttribution(lifecycle) {
			violations++
		}
	case utility.EndReasonSmokeExtinguished:
		if throw.Type != utility.TypeMolotov && throw.Type != utility.TypeIncendiary && !isUnavailableType {
			violations++
		}
		if lifecycle.Expiration.Status != utility.AvailabilityObserved || lifecycle.Extinguish.Status != utility.AvailabilityObserved ||
			lifecycle.ExtinguishedByThrowID.Status != utility.AvailabilityObserved || lifecycle.ExtinguishedByThrowID.Value == "" ||
			lifecycle.ExtinguishAttribution.Status == utility.CorrelationUnavailable {
			violations++
		}
	case utility.EndReasonDestroyed:
		if lifecycle.Destroy.Status != utility.AvailabilityObserved || utilityHasExtinguishAttribution(lifecycle) {
			violations++
		}
	case utility.EndReasonRoundEnd:
		if utilityHasExtinguishAttribution(lifecycle) {
			violations++
		}
	case utility.EndReasonUnavailable:
		if utilityHasExtinguishAttribution(lifecycle) {
			violations++
		}
	default:
		violations++
	}
	return violations
}

func utilityHasExtinguishAttribution(lifecycle utility.Lifecycle) bool {
	return lifecycle.Extinguish.Status == utility.AvailabilityObserved ||
		lifecycle.ExtinguishedByThrowID.Status == utility.AvailabilityObserved ||
		lifecycle.ExtinguishAttribution.Status != utility.CorrelationUnavailable
}

func utilityFlashAttributionMismatches(throw utility.Throw) int {
	if len(throw.Flashes) > 0 && throw.Type != utility.TypeFlashbang {
		return len(throw.Flashes)
	}
	violations := 0
	seenVictims := make(map[uint64]struct{}, len(throw.Flashes))
	for _, effect := range throw.Flashes {
		if effect.Source != utility.SourcePlayerFlashed || !validUtilityFlashDuration(effect.Duration) ||
			!validUtilityEffectVictim(effect.Victim, utility.SourcePlayerFlashed) ||
			!validUtilityFlashEffectCorrelation(effect.Correlation) ||
			!validUtilityRelation(throw.Actor, effect.Victim, effect.Relation) {
			violations++
		}
		if throw.Lifecycle.Detonation.Status == utility.AvailabilityObserved && effect.Tick != throw.Lifecycle.Detonation.Tick {
			violations++
		}
		if effect.Victim.ID != 0 {
			if _, duplicate := seenVictims[effect.Victim.ID]; duplicate {
				violations++
			}
			seenVictims[effect.Victim.ID] = struct{}{}
		}
	}
	affected, _ := projectCanonicalUtilityEffects(throw)
	if !canonicalUtilityFlashSummariesEqual(canonicalUtilityFlashSummary(affected), expectedCanonicalUtilityFlashSummary(throw.Flashes)) {
		violations++
	}
	return violations
}

func expectedCanonicalUtilityFlashSummary(effects []utility.FlashEffect) models.CanonicalUtilityFlashSummary {
	total := canonicalUtilityDurationAggregate{complete: true}
	buckets := map[utility.Relation]*canonicalUtilityDurationAggregate{
		utility.RelationEnemy:    {complete: true},
		utility.RelationTeammate: {complete: true},
		utility.RelationSelf:     {complete: true},
		utility.RelationUnknown:  {complete: true},
	}
	summary := models.CanonicalUtilityFlashSummary{PlayersTotal: len(effects)}
	for _, effect := range effects {
		relation := effect.Relation
		bucket := buckets[relation]
		if bucket == nil {
			relation = utility.RelationUnknown
			bucket = buckets[relation]
		}
		bucket.count++
		total.count++
		if effect.Duration.Status != utility.AvailabilityObserved {
			bucket.complete = false
			total.complete = false
		} else {
			milliseconds := effect.Duration.Value * 1000
			bucket.totalMS += milliseconds
			total.totalMS += milliseconds
		}
		switch relation {
		case utility.RelationEnemy:
			summary.EnemiesFlashed++
		case utility.RelationTeammate:
			summary.TeammatesFlashed++
		case utility.RelationSelf:
			summary.SelfFlashed++
		default:
			summary.UnknownFlashed++
		}
	}
	summary.TotalDurationMS = canonicalUtilityAggregateDuration(total)
	summary.EnemyDurationMS = canonicalUtilityAggregateDuration(*buckets[utility.RelationEnemy])
	summary.TeammateDurationMS = canonicalUtilityAggregateDuration(*buckets[utility.RelationTeammate])
	summary.SelfDurationMS = canonicalUtilityAggregateDuration(*buckets[utility.RelationSelf])
	summary.UnknownDurationMS = canonicalUtilityAggregateDuration(*buckets[utility.RelationUnknown])
	return summary
}

func canonicalUtilityFlashSummariesEqual(left, right models.CanonicalUtilityFlashSummary) bool {
	return left.PlayersTotal == right.PlayersTotal && left.EnemiesFlashed == right.EnemiesFlashed &&
		left.TeammatesFlashed == right.TeammatesFlashed && left.SelfFlashed == right.SelfFlashed &&
		left.UnknownFlashed == right.UnknownFlashed && utilityFloatPointersEqual(left.TotalDurationMS, right.TotalDurationMS) &&
		utilityFloatPointersEqual(left.EnemyDurationMS, right.EnemyDurationMS) &&
		utilityFloatPointersEqual(left.TeammateDurationMS, right.TeammateDurationMS) &&
		utilityFloatPointersEqual(left.SelfDurationMS, right.SelfDurationMS) &&
		utilityFloatPointersEqual(left.UnknownDurationMS, right.UnknownDurationMS)
}

func utilityFloatPointersEqual(left, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return math.Abs(*left-*right) <= 1e-9
}

func validUtilityFlashDuration(duration utility.ScalarObservation) bool {
	if duration.Status == utility.AvailabilityObserved {
		return duration.Source == utility.SourcePlayerFlashed && finiteNonNegative(duration.Value)
	}
	return duration.Status == utility.AvailabilityUnavailable && duration.Source == utility.SourceUnavailable && duration.Value == 0
}

func utilityDamageMismatches(throw utility.Throw) int {
	violations := 0
	if len(throw.Damage) > 0 && throw.Type != utility.TypeHE && throw.Type != utility.TypeMolotov &&
		throw.Type != utility.TypeIncendiary && throw.Type != utility.TypeUnknown {
		violations += len(throw.Damage)
	}
	seen := make(map[string]struct{}, len(throw.Damage))
	for index, effect := range throw.Damage {
		if effect.Source != utility.SourcePlayerHurt || effect.HealthDamage < 0 || effect.ArmorDamage < 0 ||
			!validUtilityEffectVictim(effect.Victim, utility.SourcePlayerHurt) ||
			!validUtilityDamageEffectCorrelation(effect.Correlation) ||
			!validUtilityRelation(throw.Actor, effect.Victim, effect.Relation) {
			violations++
		}
		victimKey := fmt.Sprint(effect.Victim.ID)
		if effect.Victim.ID == 0 {
			victimKey = fmt.Sprintf("unavailable:%d", index)
		}
		key := fmt.Sprintf("%s:%d:%d:%d:%t", victimKey, effect.Tick, effect.HealthDamage, effect.ArmorDamage, effect.Kill)
		if _, duplicate := seen[key]; duplicate {
			violations++
		}
		seen[key] = struct{}{}
	}
	_, actualSummary := projectCanonicalUtilityEffects(throw)
	if actualSummary != expectedCanonicalUtilityDamageSummary(throw.Damage) {
		violations++
	}
	return violations
}

type utilityDamageAggregate struct {
	relation utility.Relation
	health   int
	armor    int
	kill     bool
}

func expectedCanonicalUtilityDamageSummary(effects []utility.DamageEffect) models.CanonicalUtilityDamageSummary {
	players := make(map[string]*utilityDamageAggregate, len(effects))
	for index, effect := range effects {
		key := canonicalUtilityVictimKey(effect.Victim, "damage", index)
		player := players[key]
		if player == nil {
			player = &utilityDamageAggregate{relation: effect.Relation}
			players[key] = player
		} else if player.relation == utility.RelationUnknown && effect.Relation != utility.RelationUnknown {
			player.relation = effect.Relation
		}
		player.health += effect.HealthDamage
		player.armor += effect.ArmorDamage
		player.kill = player.kill || effect.Kill
	}
	summary := models.CanonicalUtilityDamageSummary{}
	for _, player := range players {
		summary.TotalDamage += player.health
		summary.TotalArmorDamage += player.armor
		hasDamage := player.health > 0 || player.armor > 0
		switch player.relation {
		case utility.RelationEnemy:
			summary.EnemyDamage += player.health
			summary.EnemyArmorDamage += player.armor
			if hasDamage {
				summary.EnemiesDamaged++
			}
			if player.kill {
				summary.EnemyKills++
			}
		case utility.RelationTeammate:
			summary.TeammateDamage += player.health
			summary.TeammateArmorDamage += player.armor
			if hasDamage {
				summary.TeammatesDamaged++
			}
			if player.kill {
				summary.TeammateKills++
			}
		case utility.RelationSelf:
			summary.SelfDamage += player.health
			summary.SelfArmorDamage += player.armor
			summary.SelfDamaged = summary.SelfDamaged || hasDamage
			if player.kill {
				summary.SelfKills++
			}
		default:
			summary.UnknownDamage += player.health
			summary.UnknownArmorDamage += player.armor
			if hasDamage {
				summary.UnknownPlayersDamaged++
			}
			if player.kill {
				summary.UnknownKills++
			}
		}
	}
	return summary
}

func utilityTemporalSpatialViolations(throw utility.Throw, bounds *models.ReplayRound) int {
	violations := 0
	launchTick := throw.Launch.Tick.Tick
	hasLaunchTick := throw.Launch.Tick.Status == utility.AvailabilityObserved
	if hasLaunchTick && launchTick < 0 {
		violations++
	}
	if throw.Launch.Position.Status == utility.AvailabilityObserved && !finiteUtilityVector(throw.Launch.Position.Value) {
		violations++
	}
	if throw.Launch.View.Status == utility.AvailabilityObserved && !finiteUtilityVector(throw.Launch.View.Vector) {
		violations++
	}
	for _, velocity := range []utility.VelocityObservation{throw.Launch.ThrowerVelocity, throw.Launch.ProjectileInitialVelocity} {
		if velocity.Status != utility.AvailabilityObserved {
			continue
		}
		horizontal := math.Hypot(velocity.Vector.X, velocity.Vector.Y)
		if !finiteUtilityVector(velocity.Vector) || !finiteNonNegative(velocity.HorizontalSpeed) ||
			math.Abs(horizontal-velocity.HorizontalSpeed) > 1e-6 || !hasLaunchTick || velocity.ObservedTick != launchTick ||
			velocity.MeasurementWindowTicks < 0 {
			violations++
		}
	}
	lifecycleOrderValid := utilityLifecycleOrderValid(throw.Lifecycle)
	for _, moment := range []struct {
		observation utility.TickPositionObservation
		terminal    bool
	}{
		{observation: throw.Lifecycle.Detonation},
		{observation: throw.Lifecycle.EffectStart},
		{observation: throw.Lifecycle.Expiration, terminal: true},
		{observation: throw.Lifecycle.Destroy, terminal: true},
		{observation: throw.Lifecycle.Extinguish, terminal: true},
	} {
		observation := moment.observation
		if hasLaunchTick && observation.Status == utility.AvailabilityObserved && observation.Tick < launchTick {
			violations++
		}
		if observation.Status == utility.AvailabilityObserved &&
			!utilityLifecycleMomentInsideBounds(observation.Tick, bounds, moment.terminal, lifecycleOrderValid) {
			violations++
		}
		if observation.PositionStatus == utility.AvailabilityObserved && !finiteUtilityVector(observation.Position) {
			violations++
		}
	}
	for _, sample := range throw.Trajectory.Samples {
		if !finiteUtilityVector(sample.Position) || !utilityTickInsideBounds(sample.Tick, bounds) ||
			(hasLaunchTick && sample.Tick < launchTick) || !utilityTrajectorySampleWithinTerminal(sample, throw.Lifecycle) {
			violations++
		}
	}
	for _, bounce := range throw.Trajectory.Bounces {
		if !utilityTickInsideBounds(bounce.Tick, bounds) || (hasLaunchTick && bounce.Tick < launchTick) ||
			!utilityTrajectoryBounceWithinTerminal(bounce, throw.Lifecycle) {
			violations++
		}
	}
	for _, effect := range throw.Flashes {
		if hasLaunchTick && effect.Tick < launchTick || !utilityTickInsideBounds(effect.Tick, bounds) {
			violations++
		}
	}
	for _, effect := range throw.Damage {
		if hasLaunchTick && effect.Tick < launchTick || !utilityTickInsideBounds(effect.Tick, bounds) {
			violations++
		}
	}
	if hasLaunchTick && !utilityTickInsideBounds(launchTick, bounds) {
		violations++
	}
	if !lifecycleOrderValid {
		violations++
	}
	return violations
}

func validUtilityEntityObservation(throw utility.Throw) bool {
	if !validUtilitySource(throw.EntitySource) {
		return false
	}
	if throw.EntityStatus == utility.AvailabilityObserved {
		return throw.SourceEntityID > 0 && throw.SourceEntityGeneration > 0 &&
			(throw.EntitySource == utility.SourceProjectileEntity || throw.EntitySource == utility.SourceGrenadeEntityID)
	}
	return (throw.EntityStatus == utility.AvailabilityUnavailable || throw.EntityStatus == utility.AvailabilityNotApplicable) &&
		throw.SourceEntityID == 0 && throw.SourceEntityGeneration == 0 && throw.EntitySource == utility.SourceUnavailable
}

func validUtilityPlayerRef(player utility.PlayerRef) bool {
	if !validUtilitySource(player.Source) {
		return false
	}
	if player.Status == utility.AvailabilityObserved {
		return player.ID != 0 && player.Source != utility.SourceUnavailable
	}
	return (player.Status == utility.AvailabilityUnavailable || player.Status == utility.AvailabilityNotApplicable) &&
		player.ID == 0 && (player.Source == utility.SourceUnavailable || player.Source == utility.SourceCallbackActor)
}

func validUtilityThrowerRef(player utility.PlayerRef) bool {
	if !validUtilityPlayerRef(player) {
		return false
	}
	if player.Status != utility.AvailabilityObserved {
		return player.Source == utility.SourceUnavailable
	}
	switch player.Source {
	case utility.SourceProjectileThrower, utility.SourceProjectileOwner, utility.SourceCallbackActor:
		return true
	default:
		return false
	}
}

func validUtilityEffectVictim(player utility.PlayerRef, expectedSource string) bool {
	if player.Source != expectedSource {
		return false
	}
	if player.Status == utility.AvailabilityObserved {
		return player.ID != 0
	}
	return (player.Status == utility.AvailabilityUnavailable || player.Status == utility.AvailabilityNotApplicable) && player.ID == 0
}

func validUtilityTickObservation(observation utility.TickObservation) bool {
	if observation.Status == utility.AvailabilityObserved {
		return observation.Tick >= 0 && observation.Source == utility.SourceProjectileThrow
	}
	return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
		observation.Tick == 0 && observation.Source == utility.SourceUnavailable
}

func validUtilityVectorObservation(observation utility.VectorObservation) bool {
	if observation.Status == utility.AvailabilityObserved {
		return finiteUtilityVector(observation.Value) && observation.Source != utility.SourceUnavailable && validUtilitySource(observation.Source)
	}
	return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
		utilityVectorIsZero(observation.Value) && observation.Source == utility.SourceUnavailable
}

func validUtilityViewObservation(observation utility.ViewObservation) bool {
	if observation.Status == utility.AvailabilityObserved {
		magnitude := math.Sqrt(observation.Vector.X*observation.Vector.X + observation.Vector.Y*observation.Vector.Y + observation.Vector.Z*observation.Vector.Z)
		return finiteUtilityVector(observation.Vector) && finite(observation.Yaw) && observation.Yaw >= -180 && observation.Yaw <= 180 &&
			finite(observation.Pitch) && observation.Pitch >= -90 && observation.Pitch <= 90 &&
			math.Abs(magnitude-1) <= 0.02 && observation.Source == utility.SourcePlayerView
	}
	return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
		utilityVectorIsZero(observation.Vector) && observation.Yaw == 0 && observation.Pitch == 0 &&
		observation.Source == utility.SourceUnavailable
}

func validUtilityVelocityObservation(observation utility.VelocityObservation, projectile bool) bool {
	if observation.Status == utility.AvailabilityObserved {
		if !finiteUtilityVector(observation.Vector) || !finiteNonNegative(observation.HorizontalSpeed) ||
			observation.ObservedTick < 0 || observation.MeasurementWindowTicks < 0 {
			return false
		}
		if projectile {
			return observation.Source == utility.SourceProjectileVelocity && observation.MeasurementWindowTicks == 0
		}
		if observation.Source == utility.SourceVelocityPositionDelta {
			return observation.MeasurementWindowTicks > 0
		}
		return observation.Source == utility.SourceVelocityNative && observation.MeasurementWindowTicks == 0
	}
	if observation.Status == utility.AvailabilityNotApplicable {
		return utilityVelocityIsZero(observation) && observation.Source == utility.SourceVelocityNotApplicable
	}
	return observation.Status == utility.AvailabilityUnavailable && utilityVelocityIsZero(observation) &&
		validUnavailableUtilityVelocitySource(observation.Source)
}

func validUtilityStanceObservation(observation utility.StanceObservation) bool {
	if observation.Status == utility.AvailabilityObserved {
		return validUtilityStance(observation.Value) && observation.Source == utility.SourcePlayerState
	}
	return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
		(observation.Value == "" || observation.Value == utility.StanceUnknown) && observation.Source == utility.SourceUnavailable
}

func validUtilityStance(value utility.Stance) bool {
	switch value {
	case utility.StanceStanding, utility.StanceWalking, utility.StanceCrouching,
		utility.StanceCrouchWalking, utility.StanceAirborne, utility.StanceUnknown:
		return true
	default:
		return false
	}
}

func utilityVelocityIsZero(observation utility.VelocityObservation) bool {
	return utilityVectorIsZero(observation.Vector) && observation.HorizontalSpeed == 0 &&
		observation.ObservedTick == 0 && observation.MeasurementWindowTicks == 0
}

func validUnavailableUtilityVelocitySource(source string) bool {
	switch source {
	case utility.SourceUnavailable, utility.SourceVelocityNoHistory, utility.SourceVelocityRejected,
		utility.SourceVelocityStaleGap, utility.SourceVelocityEntityChanged, utility.SourceVelocityNonMonotonicTick:
		return true
	default:
		return false
	}
}

func validUtilityStringObservation(observation utility.StringObservation) bool {
	if observation.Status == utility.AvailabilityObserved {
		return observation.Value != "" && observation.Source != utility.SourceUnavailable && validUtilitySource(observation.Source)
	}
	return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
		observation.Value == "" && observation.Source == utility.SourceUnavailable
}

func validUtilityMomentObservation(observation utility.TickPositionObservation) bool {
	if observation.Status != utility.AvailabilityObserved {
		return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
			observation.Tick == 0 && observation.PositionStatus != utility.AvailabilityObserved &&
			utilityVectorIsZero(observation.Position) && observation.Source == utility.SourceUnavailable
	}
	if observation.Tick < 0 || observation.Source == utility.SourceUnavailable || !validUtilitySource(observation.Source) {
		return false
	}
	if observation.PositionStatus == utility.AvailabilityObserved {
		return finiteUtilityVector(observation.Position)
	}
	return (observation.PositionStatus == utility.AvailabilityUnavailable ||
		observation.PositionStatus == utility.AvailabilityNotApplicable) && utilityVectorIsZero(observation.Position)
}

func validUtilityScalarObservation(observation utility.ScalarObservation) bool {
	if observation.Status == utility.AvailabilityObserved {
		return finiteNonNegative(observation.Value) && observation.Source != utility.SourceUnavailable && validUtilitySource(observation.Source)
	}
	return (observation.Status == utility.AvailabilityUnavailable || observation.Status == utility.AvailabilityNotApplicable) &&
		observation.Value == 0 && observation.Source == utility.SourceUnavailable
}

func validUtilityTrajectory(trajectory utility.Trajectory, lifecycle utility.Lifecycle) bool {
	samplesValid := true
	frameSamples, destroySamples := 0, 0
	var destroySample utility.TrajectorySample
	for _, sample := range trajectory.Samples {
		samplesValid = samplesValid && sample.Tick >= 0 && finiteUtilityVector(sample.Position) &&
			(sample.Source == utility.SourceProjectileFrames || sample.Source == utility.SourceProjectileDestroy) &&
			utilityTrajectorySampleWithinTerminal(sample, lifecycle)
		if sample.Source == utility.SourceProjectileFrames {
			frameSamples++
		} else if sample.Source == utility.SourceProjectileDestroy {
			destroySamples++
			destroySample = sample
		}
	}
	flightTerminalObserved := utilityTrajectoryFlightTerminalTick(lifecycle) >= 0
	switch trajectory.Status {
	case utility.TrajectoryObserved:
		if !samplesValid || frameSamples == 0 || trajectory.Source != utility.SourceProjectileFrames {
			return false
		}
		if flightTerminalObserved {
			if destroySamples != 0 {
				return false
			}
		} else if destroySamples != 1 || !utilityDestroySampleMatchesLifecycle(destroySample, lifecycle.Destroy) {
			return false
		}
	case utility.TrajectoryPartial:
		if len(trajectory.Samples) == 0 || !samplesValid ||
			!validPartialUtilityTrajectorySource(trajectory, frameSamples, destroySamples, destroySample, lifecycle, flightTerminalObserved) {
			return false
		}
	case utility.TrajectoryUnavailable:
		if len(trajectory.Samples) != 0 || trajectory.Source != utility.SourceUnavailable ||
			(!flightTerminalObserved && utilityDestroyPositionObserved(lifecycle.Destroy)) {
			return false
		}
	default:
		return false
	}
	lastBounceNumber := 0
	for _, bounce := range trajectory.Bounces {
		if bounce.Tick < 0 || bounce.Number <= lastBounceNumber || bounce.Source != utility.SourceProjectileBounce ||
			!utilityTrajectoryBounceWithinTerminal(bounce, lifecycle) {
			return false
		}
		lastBounceNumber = bounce.Number
		if bounce.PositionStatus == utility.AvailabilityObserved {
			if !finiteUtilityVector(bounce.Position) {
				return false
			}
		} else if (bounce.PositionStatus != utility.AvailabilityUnavailable && bounce.PositionStatus != utility.AvailabilityNotApplicable) ||
			!utilityVectorIsZero(bounce.Position) {
			return false
		}
	}
	if trajectory.BounceStatus == utility.AvailabilityObserved {
		return len(trajectory.Bounces) > 0 && trajectory.BounceCount == len(trajectory.Bounces) &&
			trajectory.BounceSource == utility.SourceProjectileBounce
	}
	return (trajectory.BounceStatus == utility.AvailabilityUnavailable || trajectory.BounceStatus == utility.AvailabilityNotApplicable) &&
		trajectory.BounceCount == 0 && len(trajectory.Bounces) == 0 && trajectory.BounceSource == utility.SourceUnavailable
}

func validPartialUtilityTrajectorySource(
	trajectory utility.Trajectory,
	frameSamples, destroySamples int,
	destroySample utility.TrajectorySample,
	lifecycle utility.Lifecycle,
	flightTerminalObserved bool,
) bool {
	if frameSamples > 0 {
		if flightTerminalObserved {
			return false
		}
		return destroySamples == 0 && trajectory.Source == utility.SourceProjectileFrames &&
			(lifecycle.Destroy.Status != utility.AvailabilityObserved || lifecycle.Destroy.PositionStatus != utility.AvailabilityObserved)
	}
	return !flightTerminalObserved && destroySamples == 1 && trajectory.Source == utility.SourceProjectileDestroy &&
		utilityDestroySampleMatchesLifecycle(destroySample, lifecycle.Destroy)
}

func utilityDestroySampleMatchesLifecycle(sample utility.TrajectorySample, destroy utility.TickPositionObservation) bool {
	return utilityDestroyPositionObserved(destroy) &&
		sample.Tick == destroy.Tick && sample.Position == destroy.Position
}

func utilityDestroyPositionObserved(destroy utility.TickPositionObservation) bool {
	return destroy.Status == utility.AvailabilityObserved && destroy.PositionStatus == utility.AvailabilityObserved
}

func validUtilityLifecycleObservations(lifecycle utility.Lifecycle) bool {
	for _, observation := range []utility.TickPositionObservation{
		lifecycle.Detonation, lifecycle.EffectStart, lifecycle.Expiration, lifecycle.Destroy, lifecycle.Extinguish,
	} {
		if !validUtilityMomentObservation(observation) {
			return false
		}
	}
	if !validUtilityScalarObservation(lifecycle.Duration) || !validUtilityStringObservation(lifecycle.Area) ||
		!validUtilityStringObservation(lifecycle.ExtinguishedByThrowID) || !validUtilityCorrelation(lifecycle.Correlation) ||
		!validUtilityCorrelation(lifecycle.ExtinguishAttribution) {
		return false
	}
	return validUtilityEndReason(lifecycle.EndReason, lifecycle.EndReasonSource)
}

func validUtilityCorrelation(correlation utility.Correlation) bool {
	switch correlation.Status {
	case utility.CorrelationObserved:
		return correlation.Source == utility.SourceProjectileEntity || correlation.Source == utility.SourceGrenadeEntityID ||
			correlation.Source == utility.SourceEffectEntityID
	case utility.CorrelationInferred:
		return correlation.Source == utility.SourceThrowerTypePositionTick || correlation.Source == utility.SourceTypePositionTick ||
			correlation.Source == utility.SourceSpatialSmokeOverlap
	case utility.CorrelationUnavailable:
		return correlation.Source == utility.SourceUnavailable
	default:
		return false
	}
}

func validUtilityFlashEffectCorrelation(correlation utility.Correlation) bool {
	switch correlation.Status {
	case utility.CorrelationObserved:
		return correlation.Source == utility.SourceProjectileEntity
	case utility.CorrelationInferred:
		return correlation.Source == utility.SourceThrowerTypePositionTick || correlation.Source == utility.SourceTypePositionTick
	case utility.CorrelationUnavailable:
		return correlation.Source == utility.SourceUnavailable
	default:
		return false
	}
}

func validUtilityDamageEffectCorrelation(correlation utility.Correlation) bool {
	return correlation.Status == utility.CorrelationInferred && correlation.Source == utility.SourceThrowerTypePositionTick ||
		correlation.Status == utility.CorrelationUnavailable && correlation.Source == utility.SourceUnavailable
}

func validUtilityEndReason(reason utility.EndReason, source string) bool {
	switch reason {
	case utility.EndReasonExpired:
		return source == utility.SourceExpirationCallback
	case utility.EndReasonSmokeExtinguished:
		return source == utility.SourceSpatialSmokeOverlap
	case utility.EndReasonDestroyed:
		return source == utility.SourceProjectileDestroy
	case utility.EndReasonRoundEnd:
		return source == utility.SourceRoundBoundary
	case utility.EndReasonUnavailable:
		return source == utility.SourceUnavailable
	default:
		return false
	}
}

func validUtilityType(value utility.Type) bool {
	switch value {
	case utility.TypeFlashbang, utility.TypeSmoke, utility.TypeHE, utility.TypeMolotov,
		utility.TypeIncendiary, utility.TypeDecoy, utility.TypeUnknown:
		return true
	default:
		return false
	}
}

func validUtilityTypeSource(value utility.Type, source string) bool {
	if value == utility.TypeUnknown {
		return source == utility.SourceUnavailable
	}
	return source == utility.SourceWeaponInstance || source == utility.SourceCallbackType
}

func validUtilitySource(source string) bool {
	_, valid := utilitySources[source]
	return valid
}

var utilitySources = map[string]struct{}{
	utility.SourceUnavailable: {}, utility.SourceWeaponInstance: {}, utility.SourceCallbackType: {},
	utility.SourceProjectileEntity: {}, utility.SourceGrenadeEntityID: {}, utility.SourceEffectEntityID: {}, utility.SourceProjectileThrow: {},
	utility.SourceProjectileThrower: {}, utility.SourceProjectileOwner: {}, utility.SourceProjectilePosition: {},
	utility.SourceProjectileVelocity: {}, utility.SourcePlayerView: {}, utility.SourcePlayerState: {},
	utility.SourceVelocityNative: {}, utility.SourceVelocityPositionDelta: {}, utility.SourceVelocityNoHistory: {},
	utility.SourceVelocityRejected: {}, utility.SourceVelocityStaleGap: {}, utility.SourceVelocityEntityChanged: {},
	utility.SourceVelocityNonMonotonicTick: {}, utility.SourceVelocityNotApplicable: {}, utility.SourceMapCallout: {},
	utility.SourcePlayerLastPlace: {}, utility.SourceProjectileFrames: {}, utility.SourceProjectileBounce: {},
	utility.SourceProjectileDestroy: {}, utility.SourceFlashExplode: {}, utility.SourceHEExplode: {},
	utility.SourceSmokeStart: {}, utility.SourceSmokeExpired: {}, utility.SourceInfernoStart: {},
	utility.SourceInfernoExpired: {}, utility.SourceDecoyStart: {}, utility.SourceDecoyExpired: {},
	utility.SourcePlayerFlashed: {}, utility.SourcePlayerHurt: {}, utility.SourceCallbackActor: {}, utility.SourceCallbackTicks: {},
	utility.SourceExpirationCallback: {}, utility.SourceSpatialSmokeOverlap: {}, utility.SourceRoundBoundary: {},
	utility.SourceThrowerTypePositionTick: {}, utility.SourceTypePositionTick: {},
}

func validUtilityRelation(actor, victim utility.PlayerRef, relation utility.Relation) bool {
	if relation != utility.RelationSelf && relation != utility.RelationTeammate && relation != utility.RelationEnemy && relation != utility.RelationUnknown {
		return false
	}
	if actor.Status != utility.AvailabilityObserved || victim.Status != utility.AvailabilityObserved {
		return relation == utility.RelationUnknown
	}
	if actor.ID == victim.ID {
		return relation == utility.RelationSelf
	}
	actorSide, victimSide := normalizeSide(actor.Side), normalizeSide(victim.Side)
	if actorSide == "unknown" || victimSide == "unknown" {
		return relation == utility.RelationUnknown
	}
	if actorSide == victimSide {
		return relation == utility.RelationTeammate
	}
	return relation == utility.RelationEnemy
}

func utilityExpectedLifecycleObservationMissing(throw utility.Throw) bool {
	switch throw.Type {
	case utility.TypeFlashbang, utility.TypeHE:
		return throw.Lifecycle.Detonation.Status != utility.AvailabilityObserved
	case utility.TypeSmoke, utility.TypeMolotov, utility.TypeIncendiary, utility.TypeDecoy:
		return throw.Lifecycle.EffectStart.Status != utility.AvailabilityObserved ||
			throw.Lifecycle.Expiration.Status != utility.AvailabilityObserved
	default:
		return true
	}
}

func utilityLifecyclePositionCoverageMissing(lifecycle utility.Lifecycle) bool {
	for _, observation := range []utility.TickPositionObservation{
		lifecycle.Detonation, lifecycle.EffectStart, lifecycle.Expiration, lifecycle.Destroy, lifecycle.Extinguish,
	} {
		if observation.Status == utility.AvailabilityObserved && observation.PositionStatus != utility.AvailabilityObserved {
			return true
		}
	}
	return false
}

func utilityLifecycleOrderValid(lifecycle utility.Lifecycle) bool {
	start := observedUtilityTick(lifecycle.EffectStart)
	if start < 0 {
		start = observedUtilityTick(lifecycle.Detonation)
	}
	for _, terminal := range []utility.TickPositionObservation{lifecycle.Expiration, lifecycle.Destroy, lifecycle.Extinguish} {
		if tick := observedUtilityTick(terminal); tick >= 0 && start >= 0 && tick < start {
			return false
		}
	}
	return true
}

func observedUtilityTick(observation utility.TickPositionObservation) int {
	if observation.Status != utility.AvailabilityObserved {
		return -1
	}
	return observation.Tick
}

func utilityTrajectoryFlightTerminalTick(lifecycle utility.Lifecycle) int {
	terminal := -1
	for _, observation := range []utility.TickPositionObservation{
		lifecycle.Detonation, lifecycle.EffectStart,
	} {
		if tick := observedUtilityTick(observation); tick >= 0 && (terminal < 0 || tick < terminal) {
			terminal = tick
		}
	}
	return terminal
}

func utilityTrajectoryTerminalTick(lifecycle utility.Lifecycle) int {
	if terminal := utilityTrajectoryFlightTerminalTick(lifecycle); terminal >= 0 {
		return terminal
	}
	return observedUtilityTick(lifecycle.Destroy)
}

func utilityTrajectorySampleWithinTerminal(sample utility.TrajectorySample, lifecycle utility.Lifecycle) bool {
	if terminal := utilityTrajectoryFlightTerminalTick(lifecycle); terminal >= 0 {
		return sample.Tick < terminal
	}
	destroyTick := observedUtilityTick(lifecycle.Destroy)
	if destroyTick < 0 {
		return true
	}
	if sample.Source == utility.SourceProjectileDestroy {
		return sample.Tick == destroyTick
	}
	return sample.Tick < destroyTick
}

func utilityTrajectoryBounceWithinTerminal(bounce utility.BounceObservation, lifecycle utility.Lifecycle) bool {
	terminal := utilityTrajectoryTerminalTick(lifecycle)
	return terminal < 0 || bounce.Tick < terminal
}

func utilityReplayBounds(replay *models.ReplayData) map[int]*models.ReplayRound {
	bounds := make(map[int]*models.ReplayRound)
	if replay == nil {
		return bounds
	}
	for index := range replay.Rounds {
		round := &replay.Rounds[index]
		bounds[round.Round] = round
	}
	return bounds
}

func utilityTickInsideBounds(tick int, bounds *models.ReplayRound) bool {
	return bounds == nil || tick >= bounds.StartTick && tick <= bounds.EndTick
}

func utilityLifecycleMomentInsideBounds(tick int, bounds *models.ReplayRound, terminal, lifecycleOrderValid bool) bool {
	if utilityTickInsideBounds(tick, bounds) {
		return true
	}
	return terminal && lifecycleOrderValid && bounds != nil && tick > bounds.EndTick
}

func utilityFlashEffectsSorted(effects []utility.FlashEffect) bool {
	return sort.SliceIsSorted(effects, func(i, j int) bool {
		if effects[i].Tick != effects[j].Tick {
			return effects[i].Tick < effects[j].Tick
		}
		return effects[i].Victim.ID < effects[j].Victim.ID
	})
}

func utilityTrajectorySorted(trajectory utility.Trajectory, lifecycle utility.Lifecycle) bool {
	for _, sample := range trajectory.Samples {
		if !utilityTrajectorySampleWithinTerminal(sample, lifecycle) {
			return false
		}
	}
	for _, bounce := range trajectory.Bounces {
		if !utilityTrajectoryBounceWithinTerminal(bounce, lifecycle) {
			return false
		}
	}
	if !sort.SliceIsSorted(trajectory.Samples, func(i, j int) bool {
		left, right := trajectory.Samples[i], trajectory.Samples[j]
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if compared := compareUtilityVectors(left.Position, right.Position); compared != 0 {
			return compared < 0
		}
		return left.Source < right.Source
	}) {
		return false
	}
	for index := 1; index < len(trajectory.Samples); index++ {
		if trajectory.Samples[index] == trajectory.Samples[index-1] {
			return false
		}
	}
	if !sort.SliceIsSorted(trajectory.Bounces, func(i, j int) bool {
		left, right := trajectory.Bounces[i], trajectory.Bounces[j]
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if left.Number != right.Number {
			return left.Number < right.Number
		}
		if left.PositionStatus != right.PositionStatus {
			return left.PositionStatus < right.PositionStatus
		}
		if compared := compareUtilityVectors(left.Position, right.Position); compared != 0 {
			return compared < 0
		}
		return left.Source < right.Source
	}) {
		return false
	}
	for index := 1; index < len(trajectory.Bounces); index++ {
		if trajectory.Bounces[index] == trajectory.Bounces[index-1] ||
			trajectory.Bounces[index].Number <= trajectory.Bounces[index-1].Number {
			return false
		}
	}
	return true
}

func utilityDamageEffectsSorted(effects []utility.DamageEffect) bool {
	return sort.SliceIsSorted(effects, func(i, j int) bool {
		if effects[i].Tick != effects[j].Tick {
			return effects[i].Tick < effects[j].Tick
		}
		if effects[i].Victim.ID != effects[j].Victim.ID {
			return effects[i].Victim.ID < effects[j].Victim.ID
		}
		if effects[i].HealthDamage != effects[j].HealthDamage {
			return effects[i].HealthDamage < effects[j].HealthDamage
		}
		if effects[i].ArmorDamage != effects[j].ArmorDamage {
			return effects[i].ArmorDamage < effects[j].ArmorDamage
		}
		if effects[i].Kill != effects[j].Kill {
			return !effects[i].Kill
		}
		if effects[i].Relation != effects[j].Relation {
			return effects[i].Relation < effects[j].Relation
		}
		return effects[i].Source < effects[j].Source
	})
}

func compareUtilityVectors(left, right utility.Vector) int {
	for _, pair := range [][2]float64{{left.X, right.X}, {left.Y, right.Y}, {left.Z, right.Z}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}

func legacyUtilityCoreEqual(expected, actual models.AI_GrenadeEvent) bool {
	return expected.Round == actual.Round && expected.Type == actual.Type &&
		expected.ThrowerSteamID == actual.ThrowerSteamID && expected.TickThrow == actual.TickThrow &&
		expected.TickExplode == actual.TickExplode && expected.EnemiesBlinded == actual.EnemiesBlinded &&
		expected.AlliesBlinded == actual.AlliesBlinded && expected.DamageDealt == actual.DamageDealt &&
		expected.ArmorDamageDealt == actual.ArmorDamageDealt &&
		expected.EnemyDamage == actual.EnemyDamage && expected.FriendlyDamage == actual.FriendlyDamage &&
		expected.EnemyArmorDamage == actual.EnemyArmorDamage &&
		expected.FriendlyArmorDamage == actual.FriendlyArmorDamage &&
		expected.SelfDamage == actual.SelfDamage && expected.SelfArmorDamage == actual.SelfArmorDamage &&
		expected.EnemiesDamaged == actual.EnemiesDamaged &&
		expected.AlliesDamaged == actual.AlliesDamaged && expected.SelfDamaged == actual.SelfDamaged &&
		expected.Kills == actual.Kills && expected.Extinguished == actual.Extinguished &&
		math.Abs(expected.Duration-actual.Duration) <= 1e-6
}

func finiteUtilityVector(vector utility.Vector) bool {
	return finite(vector.X) && finite(vector.Y) && finite(vector.Z)
}

func utilityVectorIsZero(vector utility.Vector) bool {
	return vector.X == 0 && vector.Y == 0 && vector.Z == 0
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func absoluteDifference(left, right int) int {
	if left > right {
		return left - right
	}
	return right - left
}
