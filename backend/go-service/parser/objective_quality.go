package parser

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type objectiveQualityStats struct {
	trackerAvailable             bool
	events                       int
	carrierSnapshots             int
	bombDrops                    int
	bombPickups                  int
	plantStarts                  int
	plantAborts                  int
	plants                       int
	defuseStarts                 int
	defuseAborts                 int
	defuses                      int
	bombExplosions               int
	missingActors                int
	missingPositions             int
	attempts                     int
	attemptsMissingStart         int
	attemptsUnclosed             int
	contractViolations           int
	roundMismatches              int
	terminalMismatches           int
	lifecycleViolations          int
	trackingCarrierSamples       int
	replayCarrierSamples         int
	replayBombStateMissing       int
	trackingCarrierMismatches    int
	replayCarrierMismatches      int
	replayEventMismatches        int
	replayProjectionExpected     int
	replayProjectionActual       int
	replayProjectionAvailable    bool
	expectsObjectiveObservations bool
	failureDetails               []string
}

const maxObjectiveFailureDetails = 8

func (stats *objectiveQualityStats) addFailureDetail(format string, arguments ...interface{}) {
	if len(stats.failureDetails) >= maxObjectiveFailureDetails {
		return
	}
	stats.failureDetails = append(stats.failureDetails, fmt.Sprintf(format, arguments...))
}

func (stats *objectiveQualityStats) addContractViolation(format string, arguments ...interface{}) {
	stats.contractViolations++
	stats.addFailureDetail(format, arguments...)
}

type objectiveRoundFacts struct {
	events       int
	attempts     int
	plants       int
	plantTick    int
	plantSite    string
	terminals    int
	terminalType objective.EventType
	terminalTick int
	terminalSite string
	resolution   objective.EventType
	lastState    objective.State
}

type objectiveCarrierProjection struct {
	known   bool
	state   objective.State
	phase   objective.Phase
	carrier uint64
}

type objectiveSampleKey struct {
	round int
	tick  int
}

type objectiveHolderSample struct {
	holders       map[uint64]struct{}
	hasDeadHolder bool
	phase         objective.Phase
	phaseConflict bool
}

type objectiveReplayProjection struct {
	round             int
	tick              int
	eventType         string
	actorID           uint64
	site              string
	x                 float64
	y                 float64
	positionAvailable bool
}

func assessObjectiveQuality(ctx *models.DemoContext) objectiveQualityStats {
	stats := objectiveQualityStats{
		trackerAvailable:             ctx.Objectives != nil,
		expectsObjectiveObservations: ctx.MatchData != nil && strings.HasPrefix(strings.ToLower(ctx.MatchData.MapName), "de_"),
	}
	if ctx.Objectives == nil {
		stats.addContractViolation("objective tracker is unavailable")
		return stats
	}

	events := ctx.Objectives.Events()
	attempts := ctx.Objectives.Attempts()
	stats.events = len(events)
	stats.attempts = len(attempts)
	assessObjectiveEventContract(ctx, events, &stats)
	stats.lifecycleViolations = assessObjectiveLifecycle(events, attempts, &stats)
	facts := buildObjectiveRoundFacts(events, attempts)
	stats.roundMismatches = assessObjectiveRoundReconciliation(ctx, facts)
	stats.terminalMismatches = assessObjectiveTerminalReconciliation(ctx, facts)
	assessObjectiveReplayProjection(ctx.ReplayData, events, &stats)
	assessTrackingCarrierConsistency(ctx.AI_TrackingEventsWithRound, events, &stats)
	assessReplayCarrierConsistency(ctx.ReplayData, &stats)
	return stats
}

func (stats objectiveQualityStats) hasHardFailure() bool {
	return !stats.trackerAvailable || stats.contractViolations > 0 ||
		stats.roundMismatches > 0 || stats.terminalMismatches > 0 ||
		stats.lifecycleViolations > 0 || stats.trackingCarrierMismatches > 0 ||
		stats.replayCarrierMismatches > 0 || stats.replayEventMismatches > 0
}

func (stats objectiveQualityStats) checks() []qualityCheck {
	return []qualityCheck{
		stats.eventContractCheck(),
		stats.roundReconciliationCheck(),
		stats.terminalReconciliationCheck(),
		stats.lifecycleCheck(),
		stats.replayProjectionCheck(),
		stats.carrierConsistencyCheck(),
	}
}

func (stats objectiveQualityStats) replayProjectionCheck() qualityCheck {
	status := "pass"
	message := "Replay bomb markers exactly project accepted plant, defuse and explosion events from the objective ledger."
	switch {
	case !stats.trackerAvailable:
		status = "not_available"
	case stats.replayEventMismatches > 0:
		status = "fail"
	case !stats.replayProjectionAvailable:
		status = "not_available"
		message = "Replay data was unavailable and the objective ledger contained no projected bomb markers."
	}
	return qualityCheck{
		Name: "objective_replay_projection", Status: status, Expected: "0",
		Actual: fmt.Sprintf(
			"mismatches=%d,expected=%d,actual=%d",
			stats.replayEventMismatches,
			stats.replayProjectionExpected,
			stats.replayProjectionActual,
		),
		Message: message,
	}
}

func (stats objectiveQualityStats) eventContractCheck() qualityCheck {
	check := qualityCheck{
		Name: "objective_event_contract", Expected: "0 contradictions",
		Actual:  fmt.Sprintf("violations=%d,missing_actors=%d,missing_positions=%d", stats.contractViolations, stats.missingActors, stats.missingPositions),
		Message: "Objective events must preserve their causal state, phase, ordering and observation provenance.",
	}
	switch {
	case !stats.trackerAvailable || stats.contractViolations > 0:
		check.Status = "fail"
	case stats.events == 0 && stats.expectsObjectiveObservations:
		check.Status = "warning"
		check.Message = "No objective observations were captured on a bomb-defusal map."
	case stats.events == 0:
		check.Status = "not_available"
		check.Message = "No objective observations were available for this match."
	case stats.missingActors > 0 || stats.missingPositions > 0:
		check.Status = "warning"
		check.Message = "Objective lifecycle is coherent, but some callback observations lack an actor or position."
	default:
		check.Status = "pass"
	}
	return check
}

func (stats objectiveQualityStats) roundReconciliationCheck() qualityCheck {
	status := "pass"
	if !stats.trackerAvailable {
		status = "not_available"
	} else if stats.roundMismatches > 0 {
		status = "fail"
	}
	return qualityCheck{
		Name: "objective_round_reconciliation", Status: status, Expected: "0", Actual: strconv.Itoa(stats.roundMismatches),
		Message: "Per-round bomb facts must match the objective ledger, tracker summary and match result.",
	}
}

func (stats objectiveQualityStats) terminalReconciliationCheck() qualityCheck {
	status := "pass"
	if !stats.trackerAvailable {
		status = "not_available"
	} else if stats.terminalMismatches > 0 {
		status = "fail"
	}
	return qualityCheck{
		Name: "objective_terminal_reconciliation", Status: status, Expected: "0", Actual: strconv.Itoa(stats.terminalMismatches),
		Message: "Bomb defuse and explosion terminals must agree with plant order, site and the native round-end reason.",
	}
}

func (stats objectiveQualityStats) lifecycleCheck() qualityCheck {
	status := "pass"
	switch {
	case !stats.trackerAvailable:
		status = "not_available"
	case stats.lifecycleViolations > 0:
		status = "fail"
	case stats.attemptsMissingStart > 0 || stats.attemptsUnclosed > 0:
		status = "warning"
	}
	return qualityCheck{
		Name: "objective_lifecycle", Status: status, Expected: "0 contradictions",
		Actual:  fmt.Sprintf("violations=%d,missing_start=%d,unclosed=%d", stats.lifecycleViolations, stats.attemptsMissingStart, stats.attemptsUnclosed),
		Message: "Plant and defuse attempts must link observed callbacks without inventing missing start or end times.",
	}
}

func (stats objectiveQualityStats) carrierConsistencyCheck() qualityCheck {
	status := "pass"
	message := "Tracking and replay snapshots preserve a single living C4 carrier and clear it after plant or resolution."
	switch {
	case !stats.trackerAvailable:
		status = "not_available"
	case stats.trackingCarrierMismatches+stats.replayCarrierMismatches > 0:
		status = "fail"
	case stats.replayBombStateMissing > 0:
		status = "warning"
		message = "Carrier snapshots are coherent where present, but some replay frames lack bomb state."
	case stats.trackingCarrierSamples+stats.replayCarrierSamples == 0:
		status = "not_available"
		message = "No tracking or replay carrier snapshots were available."
	}
	return qualityCheck{
		Name: "objective_carrier_consistency", Status: status, Expected: "0",
		Actual: strconv.Itoa(stats.trackingCarrierMismatches + stats.replayCarrierMismatches), Message: message,
	}
}

const replayObjectivePositionTolerance = 1.0
const objectiveExplosionPositionTolerance = 16.0

func assessObjectiveReplayProjection(replay *models.ReplayData, events []objective.Event, stats *objectiveQualityStats) {
	expected, resolutionTicks := expectedObjectiveReplayProjection(events)
	stats.replayProjectionExpected = len(expected)
	stats.replayProjectionAvailable = replay != nil
	if replay == nil {
		for _, marker := range expected {
			stats.addReplayProjectionMismatch("missing replay marker because replay data is unavailable: %s", describeObjectiveReplayMarker(marker))
		}
		return
	}

	actual := replayObjectiveMarkers(replay)
	stats.replayProjectionActual = len(actual)
	used := make([]bool, len(actual))
	for index, marker := range actual {
		resolutionTick, resolved := resolutionTicks[marker.round]
		if resolved && marker.tick > resolutionTick {
			used[index] = true
			stats.addReplayProjectionMismatch(
				"unexpected replay marker after objective resolution at tick %d: %s",
				resolutionTick,
				describeObjectiveReplayMarker(marker),
			)
		}
	}

	for _, marker := range expected {
		index := findObjectiveReplayMarker(marker, actual, used)
		if index < 0 {
			stats.addReplayProjectionMismatch("missing replay marker: %s", describeObjectiveReplayMarker(marker))
			continue
		}
		used[index] = true
		if reason := objectiveReplayMarkerDifference(marker, actual[index]); reason != "" {
			stats.addReplayProjectionMismatch(
				"replay marker differs from objective ledger (%s): expected %s, actual %s",
				reason,
				describeObjectiveReplayMarker(marker),
				describeObjectiveReplayMarker(actual[index]),
			)
		}
	}
	for index, marker := range actual {
		if !used[index] {
			stats.addReplayProjectionMismatch("unexpected replay marker: %s", describeObjectiveReplayMarker(marker))
		}
	}
}

func (stats *objectiveQualityStats) addReplayProjectionMismatch(format string, arguments ...interface{}) {
	stats.replayEventMismatches++
	stats.addFailureDetail("objective replay projection: "+format, arguments...)
}

func expectedObjectiveReplayProjection(events []objective.Event) ([]objectiveReplayProjection, map[int]int) {
	markers := make([]objectiveReplayProjection, 0)
	resolutionTicks := make(map[int]int)
	for _, event := range events {
		eventType, accepted := objectiveReplayEventType(event.Type)
		if !accepted {
			continue
		}
		marker := objectiveReplayProjection{
			round: event.Round, tick: event.Tick, eventType: eventType,
			actorID: event.Actor.SteamID, site: event.Site,
		}
		if event.Position.Available() {
			marker.x = event.Position.X
			marker.y = event.Position.Y
			marker.positionAvailable = true
		}
		markers = append(markers, marker)
		if event.Type == objective.EventDefuse || event.Type == objective.EventExplode {
			if previous, exists := resolutionTicks[event.Round]; !exists || event.Tick < previous {
				resolutionTicks[event.Round] = event.Tick
			}
		}
	}
	return markers, resolutionTicks
}

func objectiveReplayEventType(eventType objective.EventType) (string, bool) {
	switch eventType {
	case objective.EventPlant:
		return "bomb_plant", true
	case objective.EventDefuse:
		return "bomb_defuse", true
	case objective.EventExplode:
		return "bomb_explode", true
	default:
		return "", false
	}
}

func replayObjectiveMarkers(replay *models.ReplayData) []objectiveReplayProjection {
	markers := make([]objectiveReplayProjection, 0)
	for _, round := range replay.Rounds {
		for _, event := range round.Events {
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(event.Type)), "bomb_") {
				continue
			}
			markers = append(markers, objectiveReplayProjection{
				round: round.Round, tick: event.Tick, eventType: event.Type,
				actorID: event.PlayerID, site: event.Site, x: event.X, y: event.Y,
				positionAvailable: event.X != 0 || event.Y != 0 || event.Z != 0,
			})
		}
	}
	return markers
}

func findObjectiveReplayMarker(expected objectiveReplayProjection, actual []objectiveReplayProjection, used []bool) int {
	fallback := -1
	for index, marker := range actual {
		if used[index] || marker.round != expected.round || marker.tick != expected.tick || marker.eventType != expected.eventType {
			continue
		}
		if marker.actorID == expected.actorID && marker.site == expected.site {
			return index
		}
		if fallback < 0 {
			fallback = index
		}
	}
	return fallback
}

func objectiveReplayMarkerDifference(expected, actual objectiveReplayProjection) string {
	differences := make([]string, 0, 3)
	if expected.actorID != actual.actorID {
		differences = append(differences, "actor")
	}
	if expected.site != actual.site {
		differences = append(differences, "site")
	}
	if expected.positionAvailable {
		distance := math.Hypot(expected.x-actual.x, expected.y-actual.y)
		if math.IsNaN(actual.x) || math.IsInf(actual.x, 0) || math.IsNaN(actual.y) || math.IsInf(actual.y, 0) ||
			math.IsNaN(distance) || math.IsInf(distance, 0) || distance > replayObjectivePositionTolerance {
			differences = append(differences, "position")
		}
	}
	return strings.Join(differences, ",")
}

func describeObjectiveReplayMarker(marker objectiveReplayProjection) string {
	return fmt.Sprintf(
		"round=%d tick=%d type=%s actor=%d site=%q position=(%.2f,%.2f)",
		marker.round,
		marker.tick,
		marker.eventType,
		marker.actorID,
		marker.site,
		marker.x,
		marker.y,
	)
}

func assessObjectiveEventContract(ctx *models.DemoContext, events []objective.Event, stats *objectiveQualityStats) {
	seenIDs := make(map[string]struct{}, len(events))
	seenSequences := make(map[uint64]struct{}, len(events))
	lastRound, lastTick, lastSequence := 0, -1, uint64(0)
	for _, event := range events {
		countObjectiveEvent(event, stats)
		if objectiveEventRequiresActor(event.Type) && event.Actor.SteamID == 0 {
			stats.missingActors++
		}
		if !event.Position.Available() {
			stats.missingPositions++
		}
		if event.ID == "" || event.Sequence == 0 || event.Round < 1 || event.Round > ctx.CurrentRound || event.Tick < 0 {
			stats.addContractViolation("invalid event identity/bounds: id=%q sequence=%d round=%d tick=%d", event.ID, event.Sequence, event.Round, event.Tick)
		}
		if _, exists := seenIDs[event.ID]; exists && event.ID != "" {
			stats.addContractViolation("duplicate objective event id %q", event.ID)
		}
		if _, exists := seenSequences[event.Sequence]; exists && event.Sequence != 0 {
			stats.addContractViolation("duplicate objective event sequence %d", event.Sequence)
		}
		seenIDs[event.ID] = struct{}{}
		seenSequences[event.Sequence] = struct{}{}
		if event.Sequence <= lastSequence || event.Round < lastRound || event.Round == lastRound && event.Tick < lastTick {
			stats.addContractViolation("objective ledger order regressed at %s (round=%d tick=%d sequence=%d)", event.ID, event.Round, event.Tick, event.Sequence)
		}
		lastRound, lastTick, lastSequence = event.Round, event.Tick, event.Sequence
		if !validObjectiveSource(event.Source) || !validObjectivePosition(event.Position) || !validObjectiveActor(event.Actor) || !validObjectiveSite(event.Site) {
			stats.addContractViolation("invalid observation metadata on %s (source=%q position=%+v actor=%+v site=%q)", event.ID, event.Source, event.Position, event.Actor, event.Site)
		}
	}
	assessObjectiveExplosionPosition(events, stats)
	validateCanonicalObjectiveQuality(ctx, stats)
}

func assessObjectiveExplosionPosition(events []objective.Event, stats *objectiveQualityStats) {
	plantedPositions := make(map[int]objective.Position)
	for _, event := range events {
		switch event.Type {
		case objective.EventPlant:
			plantedPositions[event.Round] = event.Position
		case objective.EventExplode:
			plantedPosition, exists := plantedPositions[event.Round]
			if !exists || !plantedPosition.Available() || !event.Position.Available() {
				continue
			}
			distance := math.Hypot(event.Position.X-plantedPosition.X, event.Position.Y-plantedPosition.Y)
			if distance > objectiveExplosionPositionTolerance {
				stats.addContractViolation(
					"bomb explosion position drifted %.2f units from planted C4 in round %d (plant=(%.2f,%.2f), explode=(%.2f,%.2f))",
					distance,
					event.Round,
					plantedPosition.X,
					plantedPosition.Y,
					event.Position.X,
					event.Position.Y,
				)
			}
		}
	}
}

func countObjectiveEvent(event objective.Event, stats *objectiveQualityStats) {
	switch event.Type {
	case objective.EventCarrierSnapshot:
		stats.carrierSnapshots++
	case objective.EventDrop:
		stats.bombDrops++
	case objective.EventPickup:
		stats.bombPickups++
	case objective.EventPlantStart:
		stats.plantStarts++
	case objective.EventPlantAbort:
		stats.plantAborts++
	case objective.EventPlant:
		stats.plants++
	case objective.EventDefuseStart:
		stats.defuseStarts++
	case objective.EventDefuseAbort:
		stats.defuseAborts++
	case objective.EventDefuse:
		stats.defuses++
	case objective.EventExplode:
		stats.bombExplosions++
	default:
		stats.addContractViolation("unknown objective event type %q on %s", event.Type, event.ID)
	}
}

func validateCanonicalObjectiveQuality(ctx *models.DemoContext, stats *objectiveQualityStats) {
	events, err := buildCanonicalObjectiveEvents(ctx, "quality", 1)
	if err != nil {
		stats.addContractViolation("canonical objective projection failed: %v", err)
		return
	}
	stateByRound := make(map[int]string, ctx.CurrentRound)
	for _, event := range events {
		if _, known := canonicalObjectiveEventTypes[event.EventType]; !known {
			stats.addContractViolation("canonical event %s has unknown type %q", event.EventID, event.EventType)
			continue
		}
		if err := validateCanonicalObjectiveObservation(event); err != nil {
			stats.addContractViolation("canonical event %s: %v", event.EventID, err)
			continue
		}
		shapePrevious := objectiveShapePreviousState(event.EventType)
		if event.EventType == "bomb_plant_abort" && event.StateAfter == "dropped" {
			shapePrevious = "dropped"
		}
		if err := validateCanonicalObjectiveTransition(shapePrevious, event); err != nil {
			stats.addContractViolation("canonical event %s shape: %v", event.EventID, err)
			continue
		}
		previous := stateByRound[event.RoundNumber]
		if err := validateCanonicalObjectiveTransition(previous, event); err != nil && !objectiveTransitionIsCallbackGap(previous, event) {
			stats.addContractViolation("canonical event %s sequence: %v", event.EventID, err)
		}
		stateByRound[event.RoundNumber] = event.StateAfter
	}
}

func objectiveShapePreviousState(eventType string) string {
	switch eventType {
	case "bomb_plant_abort", "bomb_plant":
		return "planting"
	case "bomb_defuse_start", "bomb_explode":
		return "planted"
	case "bomb_defuse_abort", "bomb_defuse":
		return "defusing"
	default:
		return ""
	}
}

func objectiveTransitionIsCallbackGap(previous string, event models.CanonicalObjectiveEvent) bool {
	if event.AttemptStartObserved == nil || *event.AttemptStartObserved {
		return false
	}
	switch event.EventType {
	case "bomb_plant", "bomb_plant_abort":
		return previous == "" || previous == "carried" || previous == "dropped"
	case "bomb_defuse", "bomb_defuse_abort":
		return previous == "planted"
	default:
		return false
	}
}

func validObjectiveSource(source string) bool {
	return source == objective.SourceDemoinfocsEvent || source == objective.SourceDemoinfocsNativeSnapshot
}

func validObjectivePosition(position objective.Position) bool {
	if !validObjectiveSource(position.Source) {
		return false
	}
	if position.Status == objective.PositionUnavailable {
		return position.X == 0 && position.Y == 0 && position.Z == 0
	}
	return position.Status == objective.PositionObserved && finiteObjectivePosition(position)
}

func validObjectiveActor(actor objective.Actor) bool {
	return actor.Side == "" || actor.Side == "T" || actor.Side == "CT"
}

func validObjectiveSite(site string) bool {
	return site == "" || site == "A" || site == "B"
}

func objectiveEventRequiresActor(eventType objective.EventType) bool {
	return eventType != objective.EventExplode
}

func buildObjectiveRoundFacts(events []objective.Event, attempts []objective.Attempt) map[int]objectiveRoundFacts {
	facts := make(map[int]objectiveRoundFacts)
	for _, event := range events {
		round := facts[event.Round]
		round.events++
		round.lastState = event.StateAfter
		switch event.Type {
		case objective.EventPlant:
			round.plants++
			round.plantTick = event.Tick
			round.plantSite = event.Site
		case objective.EventDefuse, objective.EventExplode:
			round.terminals++
			round.terminalType = event.Type
			round.terminalTick = event.Tick
			round.terminalSite = event.Site
			round.resolution = event.Type
		}
		facts[event.Round] = round
	}
	for _, attempt := range attempts {
		round := facts[attempt.Round]
		round.attempts++
		facts[attempt.Round] = round
	}
	return facts
}

func assessObjectiveRoundReconciliation(ctx *models.DemoContext, facts map[int]objectiveRoundFacts) int {
	if ctx.MatchData == nil {
		return max(1, ctx.CurrentRound)
	}
	results, mismatches := objectiveResultsByRound(ctx.MatchData.Rounds, ctx.CurrentRound)
	for roundNumber := 1; roundNumber <= ctx.CurrentRound; roundNumber++ {
		factsForRound := facts[roundNumber]
		summary, hasSummary := ctx.Objectives.RoundSummary(roundNumber)
		result, hasResult := results[roundNumber]
		if !hasSummary || !hasResult || objectiveSummaryDiffers(summary, factsForRound, ctx.ParseCompleted) || objectiveResultDiffers(result, factsForRound) {
			mismatches++
		}
	}
	return mismatches
}

func objectiveResultsByRound(rounds []models.RoundData, expectedRounds int) (map[int]models.RoundData, int) {
	results := make(map[int]models.RoundData, len(rounds))
	mismatches := 0
	for _, round := range rounds {
		if round.Round < 1 || round.Round > expectedRounds {
			mismatches++
			continue
		}
		if _, duplicate := results[round.Round]; duplicate {
			mismatches++
			continue
		}
		results[round.Round] = round
	}
	return results, mismatches
}

func objectiveSummaryDiffers(summary objective.RoundSummary, facts objectiveRoundFacts, parseCompleted bool) bool {
	wasPlanted := facts.plants > 0
	if summary.WasPlanted != wasPlanted || summary.EventCount != facts.events || summary.AttemptCount != facts.attempts {
		return true
	}
	if wasPlanted && (summary.Site != facts.plantSite || summary.PlantTick != facts.plantTick) {
		return true
	}
	if summary.Resolution != facts.resolution || summary.ResolutionTick != facts.terminalTick {
		return true
	}
	if facts.events > 0 && summary.FinalState != facts.lastState {
		return true
	}
	return parseCompleted && summary.ResolvedAtRoundEnd != (facts.resolution == "")
}

func objectiveResultDiffers(result models.RoundData, facts objectiveRoundFacts) bool {
	wasPlanted := facts.plants > 0
	if result.BombPlanted != wasPlanted {
		return true
	}
	if !wasPlanted {
		return result.BombSite != "" || result.BombTick != 0
	}
	return strings.ToUpper(result.BombSite) != facts.plantSite || result.BombTick != facts.plantTick
}

func assessObjectiveTerminalReconciliation(ctx *models.DemoContext, facts map[int]objectiveRoundFacts) int {
	if ctx.MatchData == nil {
		return 0
	}
	results, _ := objectiveResultsByRound(ctx.MatchData.Rounds, ctx.CurrentRound)
	mismatches := 0
	for roundNumber := 1; roundNumber <= ctx.CurrentRound; roundNumber++ {
		result, exists := results[roundNumber]
		if !exists {
			continue
		}
		if objectiveTerminalDiffers(facts[roundNumber], result.Reason) {
			mismatches++
		}
	}
	return mismatches
}

func objectiveTerminalDiffers(facts objectiveRoundFacts, rawReason string) bool {
	if facts.plants > 1 || facts.terminals > 1 {
		return true
	}
	if facts.terminals == 1 {
		if facts.plants != 1 || facts.terminalTick < facts.plantTick || facts.terminalSite == "" || facts.terminalSite != facts.plantSite {
			return true
		}
	}
	reason, err := strconv.Atoi(strings.TrimSpace(rawReason))
	if err != nil {
		return facts.plants > 0 || facts.terminals > 0
	}
	switch reason {
	case 1:
		return facts.terminals != 1 || facts.terminalType != objective.EventExplode
	case 7:
		return facts.terminals != 1 || facts.terminalType != objective.EventDefuse
	case 8:
		return facts.plants == 1 && (facts.terminals != 1 || facts.terminalType != objective.EventDefuse)
	case 12:
		return facts.plants != 0 || facts.terminals != 0
	default:
		return facts.terminals != 0
	}
}

func assessObjectiveLifecycle(events []objective.Event, attempts []objective.Attempt, stats *objectiveQualityStats) int {
	eventsByID := make(map[string]objective.Event, len(events))
	attemptsByID := make(map[string]objective.Attempt, len(attempts))
	violations := 0
	for _, event := range events {
		eventsByID[event.ID] = event
	}
	for _, attempt := range attempts {
		if !attempt.StartObserved {
			stats.attemptsMissingStart++
		}
		if attempt.Outcome == objective.AttemptInProgress || attempt.EndTick == nil {
			stats.attemptsUnclosed++
		}
		if _, duplicate := attemptsByID[attempt.ID]; duplicate || !validObjectiveAttemptShape(attempt, eventsByID) {
			violations++
		}
		attemptsByID[attempt.ID] = attempt
	}
	for _, event := range events {
		kind, outcome, isAttemptEvent := objectiveAttemptEventContract(event.Type)
		if !isAttemptEvent {
			if event.AttemptID != "" {
				violations++
			}
			continue
		}
		attempt, exists := attemptsByID[event.AttemptID]
		if !exists || attempt.Kind != kind || event.AttemptOutcome != outcome || event.AttemptStartObserved != attempt.StartObserved {
			violations++
			continue
		}
		if !sameOptionalBool(event.HasKit, attempt.HasKit) || event.Round != attempt.Round {
			violations++
		}
	}
	return violations
}

func validObjectiveAttemptShape(attempt objective.Attempt, eventsByID map[string]objective.Event) bool {
	if attempt.ID == "" || attempt.Round < 1 || attempt.StartTick < 0 ||
		(attempt.Kind != objective.AttemptPlant && attempt.Kind != objective.AttemptDefuse) {
		return false
	}
	if attempt.StartObserved {
		start, exists := eventsByID[attempt.StartEventID]
		if !exists || start.Tick != attempt.StartTick || start.AttemptID != attempt.ID || start.Type != objectiveAttemptStartType(attempt.Kind) {
			return false
		}
	} else if attempt.StartEventID != "" || attempt.DurationTicks != nil {
		return false
	}
	if attempt.Outcome == objective.AttemptInProgress {
		return attempt.EndTick == nil && attempt.EndEventID == "" && attempt.DurationTicks == nil
	}
	if attempt.Outcome != objective.AttemptCompleted && attempt.Outcome != objective.AttemptAborted || attempt.EndTick == nil || attempt.EndEventID == "" {
		return false
	}
	end, exists := eventsByID[attempt.EndEventID]
	if !exists || end.Tick != *attempt.EndTick || end.AttemptID != attempt.ID || *attempt.EndTick < attempt.StartTick {
		return false
	}
	if end.Type != objectiveAttemptEndType(attempt.Kind, attempt.Outcome) {
		return false
	}
	return !attempt.StartObserved || attempt.DurationTicks != nil && *attempt.DurationTicks == *attempt.EndTick-attempt.StartTick
}

func objectiveAttemptEventContract(eventType objective.EventType) (objective.AttemptKind, objective.AttemptOutcome, bool) {
	switch eventType {
	case objective.EventPlantStart:
		return objective.AttemptPlant, objective.AttemptInProgress, true
	case objective.EventPlantAbort:
		return objective.AttemptPlant, objective.AttemptAborted, true
	case objective.EventPlant:
		return objective.AttemptPlant, objective.AttemptCompleted, true
	case objective.EventDefuseStart:
		return objective.AttemptDefuse, objective.AttemptInProgress, true
	case objective.EventDefuseAbort:
		return objective.AttemptDefuse, objective.AttemptAborted, true
	case objective.EventDefuse:
		return objective.AttemptDefuse, objective.AttemptCompleted, true
	default:
		return "", "", false
	}
}

func objectiveAttemptStartType(kind objective.AttemptKind) objective.EventType {
	if kind == objective.AttemptPlant {
		return objective.EventPlantStart
	}
	return objective.EventDefuseStart
}

func objectiveAttemptEndType(kind objective.AttemptKind, outcome objective.AttemptOutcome) objective.EventType {
	if kind == objective.AttemptPlant {
		if outcome == objective.AttemptAborted {
			return objective.EventPlantAbort
		}
		return objective.EventPlant
	}
	if outcome == objective.AttemptAborted {
		return objective.EventDefuseAbort
	}
	return objective.EventDefuse
}

func sameOptionalBool(left, right *bool) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func assessTrackingCarrierConsistency(tracking []models.AI_TrackingEventWithRound, events []objective.Event, stats *objectiveQualityStats) {
	samples := make(map[objectiveSampleKey]*objectiveHolderSample)
	for _, wrapped := range tracking {
		key := objectiveSampleKey{round: wrapped.Round, tick: wrapped.Event.Tick}
		sample := samples[key]
		if sample == nil {
			sample = &objectiveHolderSample{holders: make(map[uint64]struct{})}
			samples[key] = sample
		}
		if wrapped.Event.HasC4 {
			sample.holders[wrapped.Event.PlayerSteamID] = struct{}{}
			sample.hasDeadHolder = sample.hasDeadHolder || !wrapped.Event.IsAlive
		}
		phase := objective.Phase(wrapped.Event.ObjectivePhase)
		if sample.phase == "" {
			sample.phase = phase
		} else if sample.phase != phase {
			sample.phaseConflict = true
		}
	}
	projections := objectiveCarrierProjections(events, samples)
	for key, sample := range samples {
		stats.trackingCarrierSamples++
		projection := projections[key]
		if objectiveHolderSampleInvalid(*sample, projection) {
			stats.trackingCarrierMismatches++
			stats.addFailureDetail(
				"tracking carrier mismatch r%d/t%d: holders=%v dead=%t projected_state=%s projected_carrier=%d known=%t",
				key.round, key.tick, sortedObjectiveHolderIDs(sample.holders), sample.hasDeadHolder,
				projection.state, projection.carrier, projection.known,
			)
		}
	}
}

func objectiveCarrierProjections(events []objective.Event, samples map[objectiveSampleKey]*objectiveHolderSample) map[objectiveSampleKey]objectiveCarrierProjection {
	keys := make([]objectiveSampleKey, 0, len(samples))
	for key := range samples {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].round < keys[j].round || keys[i].round == keys[j].round && keys[i].tick < keys[j].tick
	})
	orderedEvents := append([]objective.Event(nil), events...)
	sort.SliceStable(orderedEvents, func(i, j int) bool {
		return orderedEvents[i].Round < orderedEvents[j].Round ||
			orderedEvents[i].Round == orderedEvents[j].Round && (orderedEvents[i].Tick < orderedEvents[j].Tick ||
				orderedEvents[i].Tick == orderedEvents[j].Tick && orderedEvents[i].Sequence < orderedEvents[j].Sequence)
	})
	projections := make(map[objectiveSampleKey]objectiveCarrierProjection, len(keys))
	index := 0
	currentRound := 0
	projection := objectiveCarrierProjection{}
	for _, key := range keys {
		if key.round != currentRound {
			currentRound = key.round
			projection = objectiveCarrierProjection{}
		}
		for index < len(orderedEvents) && (orderedEvents[index].Round < key.round || orderedEvents[index].Round == key.round && orderedEvents[index].Tick <= key.tick) {
			if orderedEvents[index].Round == key.round {
				projection = applyObjectiveCarrierEvent(projection, orderedEvents[index])
			}
			index++
		}
		projections[key] = projection
	}
	return projections
}

func applyObjectiveCarrierEvent(projection objectiveCarrierProjection, event objective.Event) objectiveCarrierProjection {
	projection.known = true
	projection.state = event.StateAfter
	projection.phase = event.PhaseAfter
	switch event.StateAfter {
	case objective.StateCarried, objective.StatePlanting:
		projection.carrier = event.Actor.SteamID
	default:
		projection.carrier = 0
	}
	return projection
}

func objectiveHolderSampleInvalid(sample objectiveHolderSample, projection objectiveCarrierProjection) bool {
	if sample.phaseConflict {
		return true
	}
	if sample.phase == objective.PhaseResolved {
		return sample.hasDeadHolder || len(sample.holders) != 0
	}
	if sample.hasDeadHolder || len(sample.holders) > 1 || !projection.known {
		return sample.hasDeadHolder || len(sample.holders) > 1
	}
	switch projection.state {
	case objective.StateCarried:
		_, matchesCarrier := sample.holders[projection.carrier]
		return projection.carrier == 0 || len(sample.holders) != 1 || !matchesCarrier
	case objective.StateDropped, objective.StatePlanted, objective.StateDefusing, objective.StateDefused, objective.StateExploded, objective.StateResolved:
		return projection.carrier != 0 || len(sample.holders) != 0
	default:
		return false
	}
}

func assessReplayCarrierConsistency(replay *models.ReplayData, stats *objectiveQualityStats) {
	if replay == nil {
		return
	}
	for _, round := range replay.Rounds {
		for _, frame := range round.Frames {
			if frame.Bomb == nil {
				stats.replayBombStateMissing++
				continue
			}
			stats.replayCarrierSamples++
			if replayCarrierFrameInvalid(frame) {
				stats.replayCarrierMismatches++
				holders, hasDeadHolder := replayObjectiveHolders(frame)
				stats.addFailureDetail(
					"replay carrier mismatch r%d/t%d: holders=%v dead=%t state=%s carrier=%d",
					round.Round, frame.Tick, sortedObjectiveHolderIDs(holders), hasDeadHolder,
					frame.Bomb.State, optionalReplayPlayerID(frame.Bomb.CarrierID),
				)
			}
		}
	}
}

func replayCarrierFrameInvalid(frame models.ReplayFrame) bool {
	holders, hasDeadHolder := replayObjectiveHolders(frame)
	if hasDeadHolder || len(holders) > 1 || frame.Bomb == nil {
		return true
	}
	carrierID := optionalReplayPlayerID(frame.Bomb.CarrierID)
	if frame.Bomb.ObjectivePhase == objective.PhaseResolved {
		validState := frame.Bomb.State == objective.StateResolved || frame.Bomb.State == objective.StateDefused || frame.Bomb.State == objective.StateExploded
		return !validState || carrierID != 0 || len(holders) != 0
	}
	if frame.Bomb.State == objective.StateResolved {
		return true
	}
	switch frame.Bomb.State {
	case objective.StateCarried:
		_, matchesCarrier := holders[carrierID]
		return carrierID == 0 || len(holders) != 1 || !matchesCarrier
	case objective.StateDropped, objective.StatePlanted, objective.StateDefusing, objective.StateDefused, objective.StateExploded, objective.StateResolved:
		return carrierID != 0 || len(holders) != 0
	default:
		return false
	}
}

func replayObjectiveHolders(frame models.ReplayFrame) (map[uint64]struct{}, bool) {
	holders := make(map[uint64]struct{})
	hasDeadHolder := false
	for _, player := range frame.Players {
		if !player.HasC4 {
			continue
		}
		holders[player.SteamID] = struct{}{}
		hasDeadHolder = hasDeadHolder || !player.Alive
	}
	return holders, hasDeadHolder
}

func sortedObjectiveHolderIDs(holders map[uint64]struct{}) []uint64 {
	ids := make([]uint64, 0, len(holders))
	for id := range holders {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func optionalReplayPlayerID(playerID *uint64) uint64 {
	if playerID == nil {
		return 0
	}
	return *playerID
}

func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
