package parser

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/navigation"

	"github.com/golang/geo/r3"
)

const ObservedNavigationSmokeSchemaID = "stratai.observed_navigation_single_demo_smoke@1"

type ObservedNavigationSmokeSummary struct {
	Rows                          int            `json:"rows"`
	ReachableRows                 int            `json:"reachable_rows"`
	UnavailableRows               int            `json:"unavailable_rows"`
	IdealBaselineRows             int            `json:"ideal_baseline_rows"`
	ActiveWeaponBaselineRows      int            `json:"active_weapon_baseline_rows"`
	ObservedPaceProjectionRows    int            `json:"observed_pace_projection_rows"`
	WalkingContexts               int            `json:"walking_contexts"`
	NonWalkingMovingContexts      int            `json:"non_walking_moving_contexts"`
	DuckingContexts               int            `json:"ducking_contexts"`
	ScopedContexts                int            `json:"scoped_contexts"`
	KnifeContexts                 int            `json:"knife_contexts"`
	ActiveWeapons                 []string       `json:"active_weapons"`
	WeaponBaselineSpeeds          []float64      `json:"weapon_baseline_speeds_world_units_per_second"`
	ExactUserFacingTimeViolations int            `json:"exact_user_facing_time_violations"`
	AbstentionReasons             map[string]int `json:"abstention_reasons"`
	TimingUnavailabilityReasons   map[string]int `json:"timing_unavailability_reasons"`
}

// ObservedNavigationSmokeReport is a non-canonical validation artifact. It is
// deliberately bounded and contains real candidate rows so the versioned JSON
// schema and timing semantics can be checked after one isolated demo parse.
type ObservedNavigationSmokeReport struct {
	SchemaID                 string                          `json:"schema_id"`
	Status                   string                          `json:"status"`
	SourceAlias              string                          `json:"source_alias"`
	SourceSHA256             string                          `json:"source_sha256"`
	MapName                  string                          `json:"map_name"`
	CandidateSchemaID        string                          `json:"candidate_schema_id"`
	CandidateContractVersion string                          `json:"candidate_contract_version"`
	GraphContractVersion     string                          `json:"graph_contract_version"`
	GraphAssetReference      string                          `json:"graph_asset_reference"`
	GraphAssetSHA256         string                          `json:"graph_asset_sha256"`
	GraphNodeCount           int                             `json:"graph_node_count"`
	GraphEdgeCount           int                             `json:"graph_edge_count"`
	QueryLimit               int                             `json:"query_limit"`
	BuildIdentifier          string                          `json:"build_identifier"`
	ProcessingDurationMS     int64                           `json:"processing_duration_ms"`
	Summary                  ObservedNavigationSmokeSummary  `json:"summary"`
	Rows                     []models.ObservedNavigationPath `json:"rows"`
}

type navigationSmokeFrame struct {
	roundNumber int
	tick        int
	players     []models.ReplayPlayerState
}

type navigationSmokeCandidate struct {
	frameIndex  int
	originIndex int
}

// BuildObservedNavigationSmokeReport exercises the real candidate producer on
// deterministic, same-team, alive position pairs from one parsed demo. It
// fails closed unless the sample proves graph loading, routing, both contextual
// timing scenarios, and the no-exact-user-time policy.
func BuildObservedNavigationSmokeReport(
	replay *models.ReplayData,
	graph *navigation.Graph,
	sourceAlias string,
	sourceSHA256 string,
	queryLimit int,
) (ObservedNavigationSmokeReport, error) {
	if replay == nil {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: replay data is nil")
	}
	if strings.TrimSpace(sourceAlias) == "" {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: source alias is empty")
	}
	if !validLowerSHA256(sourceSHA256) {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: source sha256 is invalid")
	}
	if strings.TrimSpace(replay.Metadata.MapName) == "" {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: map name is empty")
	}
	if queryLimit < 1 || queryLimit > 256 {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: query limit must be between 1 and 256")
	}
	if graph == nil || len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: loaded NAV graph is unavailable or empty")
	}
	if graph.Version != navigation.GraphContractVersion {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: unexpected graph contract %q", graph.Version)
	}
	if strings.TrimSpace(graph.AssetReference) == "" || !validLowerSHA256(graph.AssetSHA256) {
		return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: NAV provenance is incomplete")
	}

	frames := eligibleNavigationSmokeFrames(replay)
	candidates := selectNavigationSmokeCandidates(frames, queryLimit)
	participantAliases := navigationSmokeParticipantAliases(frames)
	rows := make([]models.ObservedNavigationPath, 0, len(candidates))
	seenQueries := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		frame := frames[candidate.frameIndex]
		origin, target, ok := navigationSmokePairForOrigin(frame.players, candidate.originIndex)
		if !ok {
			continue
		}
		originAlias, originKnown := participantAliases[origin.SteamID]
		targetAlias, targetKnown := participantAliases[target.SteamID]
		if !originKnown || !targetKnown {
			return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: participant alias is unavailable")
		}
		queryID := fmt.Sprintf(
			"%s:round:%03d:path:%s:%s:%d",
			sourceAlias,
			frame.roundNumber,
			originAlias,
			targetAlias,
			frame.tick,
		)
		if _, duplicate := seenQueries[queryID]; duplicate {
			continue
		}
		seenQueries[queryID] = struct{}{}

		request := models.ObservedNavigationPathRequest{
			JoinKeys: models.ObservedNavigationPathJoinKeys{
				MatchID:        sourceAlias,
				RoundID:        fmt.Sprintf("%s:round:%03d", sourceAlias, frame.roundNumber),
				QueryID:        queryID,
				OriginPlayerID: originAlias,
				TargetPlayerID: targetAlias,
			},
			RoundNumber:            frame.roundNumber,
			T0Tick:                 frame.tick,
			OriginPosition:         replayPlayerNavigationPosition(origin),
			TargetPosition:         replayPlayerNavigationPosition(target),
			OriginAvailabilityTick: frame.tick,
			TargetAvailabilityTick: frame.tick,
			OriginMotion:           replayNavigationMotionInput(origin, frame.tick),
		}
		row, err := BuildObservedNavigationPath(request, graph)
		if err != nil {
			return ObservedNavigationSmokeReport{}, fmt.Errorf("build navigation smoke: query %s: %w", queryID, err)
		}
		rows = append(rows, row)
	}

	report := ObservedNavigationSmokeReport{
		SchemaID:                 ObservedNavigationSmokeSchemaID,
		Status:                   "failed",
		SourceAlias:              sourceAlias,
		SourceSHA256:             sourceSHA256,
		MapName:                  replay.Metadata.MapName,
		CandidateSchemaID:        models.ObservedNavigationPathSchemaID,
		CandidateContractVersion: models.ObservedNavigationPathContractVersion,
		GraphContractVersion:     graph.Version,
		GraphAssetReference:      graph.AssetReference,
		GraphAssetSHA256:         graph.AssetSHA256,
		GraphNodeCount:           len(graph.Nodes),
		GraphEdgeCount:           len(graph.Edges),
		QueryLimit:               queryLimit,
		Rows:                     rows,
	}
	report.Summary = summarizeObservedNavigationSmoke(rows)
	if err := validateObservedNavigationSmokeReport(report); err != nil {
		return ObservedNavigationSmokeReport{}, err
	}
	report.Status = "passed"
	return report, nil
}

func eligibleNavigationSmokeFrames(replay *models.ReplayData) []navigationSmokeFrame {
	rounds := append([]models.ReplayRound(nil), replay.Rounds...)
	sort.Slice(rounds, func(i, j int) bool {
		if rounds[i].Round != rounds[j].Round {
			return rounds[i].Round < rounds[j].Round
		}
		return rounds[i].StartTick < rounds[j].StartTick
	})
	result := make([]navigationSmokeFrame, 0)
	for _, round := range rounds {
		if round.Round < 1 {
			continue
		}
		frames := append([]models.ReplayFrame(nil), round.Frames...)
		sort.SliceStable(frames, func(i, j int) bool { return frames[i].Tick < frames[j].Tick })
		for _, frame := range frames {
			players := navigationSmokePlayers(frame.Players)
			if _, _, ok := navigationSmokePair(players); !ok {
				continue
			}
			result = append(result, navigationSmokeFrame{
				roundNumber: round.Round,
				tick:        frame.Tick,
				players:     players,
			})
		}
	}
	return result
}

func selectNavigationSmokeCandidates(
	frames []navigationSmokeFrame,
	limit int,
) []navigationSmokeCandidate {
	all := make([]navigationSmokeCandidate, 0)
	for frameIndex, frame := range frames {
		for originIndex := range frame.players {
			if _, _, ok := navigationSmokePairForOrigin(frame.players, originIndex); ok {
				all = append(all, navigationSmokeCandidate{
					frameIndex: frameIndex, originIndex: originIndex,
				})
			}
		}
	}
	if len(all) == 0 || limit <= 0 {
		return []navigationSmokeCandidate{}
	}

	selected := make([]navigationSmokeCandidate, 0, minNavigationSmoke(limit, len(all)))
	seen := make(map[navigationSmokeCandidate]struct{}, cap(selected))
	add := func(candidate navigationSmokeCandidate) bool {
		if len(selected) >= limit {
			return false
		}
		if _, exists := seen[candidate]; exists {
			return true
		}
		seen[candidate] = struct{}{}
		selected = append(selected, candidate)
		return len(selected) < limit
	}
	playerFor := func(candidate navigationSmokeCandidate) models.ReplayPlayerState {
		return frames[candidate.frameIndex].players[candidate.originIndex]
	}

	// Exercise each known current weapon found in the real demo before filling
	// temporal coverage. This deliberately includes Knife when it is present and
	// avoids a smoke that proves only one rifle-speed lookup.
	seenWeapons := make(map[string]struct{})
	for _, candidate := range all {
		player := playerFor(candidate)
		if player.ActiveWeaponObservationStatus != models.ActiveWeaponStatusObserved {
			continue
		}
		weapon := strings.TrimSpace(player.ObservedActiveWeapon)
		if _, known := models.LookupWeaponMaxSpeed(weapon); !known {
			continue
		}
		key := strings.ToLower(weapon)
		if _, exists := seenWeapons[key]; exists {
			continue
		}
		seenWeapons[key] = struct{}{}
		if !add(candidate) {
			return selected
		}
	}

	// Ensure the sample covers the directly observed movement modes whenever
	// the selected real demo contains them.
	predicates := []func(models.ReplayPlayerState) bool{
		func(player models.ReplayPlayerState) bool { return player.IsWalking },
		func(player models.ReplayPlayerState) bool {
			return !player.IsWalking && player.MotionAvailable && player.MotionHorizontalSpeed > 0
		},
		func(player models.ReplayPlayerState) bool { return player.IsDucking },
		func(player models.ReplayPlayerState) bool { return player.IsScoped },
		func(player models.ReplayPlayerState) bool {
			return player.MotionAvailable && player.MotionHorizontalSpeed == 0
		},
	}
	for _, predicate := range predicates {
		for _, candidate := range all {
			if predicate(playerFor(candidate)) {
				if !add(candidate) {
					return selected
				}
				break
			}
		}
	}

	for _, index := range evenNavigationSmokeIndexes(len(all), limit) {
		if !add(all[index]) {
			return selected
		}
	}
	for _, candidate := range all {
		if !add(candidate) {
			break
		}
	}
	return selected
}

func navigationSmokeParticipantAliases(frames []navigationSmokeFrame) map[uint64]string {
	identities := make(map[uint64]struct{})
	for _, frame := range frames {
		for _, player := range frame.players {
			identities[player.SteamID] = struct{}{}
		}
	}
	ordered := make([]uint64, 0, len(identities))
	for identity := range identities {
		ordered = append(ordered, identity)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	aliases := make(map[uint64]string, len(ordered))
	for index, identity := range ordered {
		aliases[identity] = fmt.Sprintf("participant-%03d", index+1)
	}
	return aliases
}

func navigationSmokePlayers(players []models.ReplayPlayerState) []models.ReplayPlayerState {
	result := make([]models.ReplayPlayerState, 0, len(players))
	for _, player := range players {
		if player.SteamID == 0 || !player.Alive || (player.Team != "CT" && player.Team != "T") {
			continue
		}
		position := replayPlayerNavigationPosition(player)
		if !finiteNavigation(position.X) || !finiteNavigation(position.Y) || !finiteNavigation(position.Z) {
			continue
		}
		result = append(result, player)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SteamID < result[j].SteamID })
	return result
}

func navigationSmokePair(
	players []models.ReplayPlayerState,
) (models.ReplayPlayerState, models.ReplayPlayerState, bool) {
	if len(players) < 2 {
		return models.ReplayPlayerState{}, models.ReplayPlayerState{}, false
	}
	originIndex := -1
	for index, candidate := range players {
		hasTeammate := false
		for _, other := range players {
			if other.SteamID != candidate.SteamID && other.Team == candidate.Team {
				hasTeammate = true
				break
			}
		}
		if !hasTeammate {
			continue
		}
		if originIndex < 0 || betterNavigationSmokeOrigin(candidate, players[originIndex]) {
			originIndex = index
		}
	}
	if originIndex < 0 {
		return models.ReplayPlayerState{}, models.ReplayPlayerState{}, false
	}
	return navigationSmokePairForOrigin(players, originIndex)
}

func navigationSmokePairForOrigin(
	players []models.ReplayPlayerState,
	originIndex int,
) (models.ReplayPlayerState, models.ReplayPlayerState, bool) {
	if originIndex < 0 || originIndex >= len(players) {
		return models.ReplayPlayerState{}, models.ReplayPlayerState{}, false
	}
	origin := players[originIndex]
	targetIndex := -1
	bestDistance := -1.0
	for index, candidate := range players {
		if candidate.SteamID == origin.SteamID || candidate.Team != origin.Team {
			continue
		}
		distance := navigationSmokeDistance(origin, candidate)
		if targetIndex < 0 || distance > bestDistance ||
			(distance == bestDistance && candidate.SteamID < players[targetIndex].SteamID) {
			targetIndex = index
			bestDistance = distance
		}
	}
	if targetIndex < 0 {
		return models.ReplayPlayerState{}, models.ReplayPlayerState{}, false
	}
	return origin, players[targetIndex], true
}

func minNavigationSmoke(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func betterNavigationSmokeOrigin(left, right models.ReplayPlayerState) bool {
	leftMoving := left.MotionAvailable && left.MotionHorizontalSpeed > 0
	rightMoving := right.MotionAvailable && right.MotionHorizontalSpeed > 0
	if leftMoving != rightMoving {
		return leftMoving
	}
	if leftMoving && left.MotionHorizontalSpeed != right.MotionHorizontalSpeed {
		return left.MotionHorizontalSpeed > right.MotionHorizontalSpeed
	}
	leftWeapon := left.ActiveWeaponObservationStatus == models.ActiveWeaponStatusObserved &&
		strings.TrimSpace(left.ObservedActiveWeapon) != ""
	rightWeapon := right.ActiveWeaponObservationStatus == models.ActiveWeaponStatusObserved &&
		strings.TrimSpace(right.ObservedActiveWeapon) != ""
	if leftWeapon != rightWeapon {
		return leftWeapon
	}
	return left.SteamID < right.SteamID
}

func evenNavigationSmokeIndexes(length, limit int) []int {
	if length == 0 || limit <= 0 {
		return []int{}
	}
	count := limit
	if length < count {
		count = length
	}
	if count == 1 {
		return []int{0}
	}
	indexes := make([]int, 0, count)
	for index := 0; index < count; index++ {
		indexes = append(indexes, index*(length-1)/(count-1))
	}
	return indexes
}

func replayNavigationMotionInput(
	player models.ReplayPlayerState,
	t0Tick int,
) *models.ObservedNavigationMotionInput {
	walking, ducking, scoped := player.IsWalking, player.IsDucking, player.IsScoped
	input := &models.ObservedNavigationMotionInput{
		AvailabilityTick: t0Tick,
		IsWalking:        &walking,
		IsDucking:        &ducking,
		IsScoped:         &scoped,
	}
	if player.MotionAvailable && player.MotionAvailabilityTick <= t0Tick &&
		finiteNavigation(player.MotionHorizontalSpeed) && player.MotionHorizontalSpeed >= 0 &&
		strings.TrimSpace(player.MotionSource) != "" {
		speed := player.MotionHorizontalSpeed
		input.HorizontalSpeedWorldUnitsPerSecond = &speed
		input.HorizontalSpeedSource = player.MotionSource
	}
	if player.ActiveWeaponObservationStatus == models.ActiveWeaponStatusObserved &&
		player.ActiveWeaponAvailabilityTick <= t0Tick && strings.TrimSpace(player.ObservedActiveWeapon) != "" {
		weapon := strings.TrimSpace(player.ObservedActiveWeapon)
		input.ActiveWeapon = &weapon
	}
	return input
}

func replayPlayerNavigationPosition(player models.ReplayPlayerState) r3.Vector {
	return r3.Vector{X: float64(player.X), Y: float64(player.Y), Z: player.Z}
}

func navigationSmokeDistance(left, right models.ReplayPlayerState) float64 {
	delta := replayPlayerNavigationPosition(left).Sub(replayPlayerNavigationPosition(right))
	return math.Sqrt(delta.Dot(delta))
}

func summarizeObservedNavigationSmoke(
	rows []models.ObservedNavigationPath,
) ObservedNavigationSmokeSummary {
	summary := ObservedNavigationSmokeSummary{
		AbstentionReasons:           make(map[string]int),
		TimingUnavailabilityReasons: make(map[string]int),
	}
	weapons := make(map[string]struct{})
	weaponSpeeds := make(map[float64]struct{})
	for _, row := range rows {
		summary.Rows++
		if row.Reachability.Value != nil && *row.Reachability.Value {
			summary.ReachableRows++
		} else {
			summary.UnavailableRows++
		}
		if row.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline.Status == models.NavigationFactEstimated {
			summary.IdealBaselineRows++
		}
		if row.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline.Status == models.NavigationFactEstimated {
			summary.ActiveWeaponBaselineRows++
			if row.OriginMotion.ActiveWeapon != nil {
				weapon := strings.TrimSpace(*row.OriginMotion.ActiveWeapon)
				weapons[weapon] = struct{}{}
				if strings.EqualFold(weapon, "Knife") {
					summary.KnifeContexts++
				}
			}
			if speed := row.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline.SpeedWorldUnitsPerSecond; speed != nil {
				weaponSpeeds[*speed] = struct{}{}
			}
		}
		if row.InternalTiming.Scenarios.ObservedPaceProjection.Status == models.NavigationFactEstimated {
			summary.ObservedPaceProjectionRows++
		}
		if row.OriginMotion.IsWalking != nil && *row.OriginMotion.IsWalking {
			summary.WalkingContexts++
		}
		if row.OriginMotion.IsWalking != nil && !*row.OriginMotion.IsWalking &&
			row.OriginMotion.HorizontalSpeedStatus == models.NavigationFactDerived &&
			row.OriginMotion.HorizontalSpeedWorldUnitsPerSecond != nil &&
			*row.OriginMotion.HorizontalSpeedWorldUnitsPerSecond > 0 {
			summary.NonWalkingMovingContexts++
		}
		if row.OriginMotion.IsDucking != nil && *row.OriginMotion.IsDucking {
			summary.DuckingContexts++
		}
		if row.OriginMotion.IsScoped != nil && *row.OriginMotion.IsScoped {
			summary.ScopedContexts++
		}
		if row.InternalTiming.ExactUserFacingTimeAllowed || row.InternalTiming.ArrivalPrediction {
			summary.ExactUserFacingTimeViolations++
		}
		for _, reason := range row.AbstentionReasons {
			summary.AbstentionReasons[reason]++
		}
		for _, scenario := range []models.ObservedNavigationTimingScenario{
			row.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline,
			row.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
			row.InternalTiming.Scenarios.ObservedPaceProjection,
		} {
			if scenario.Status == models.NavigationFactUnavailable && scenario.UnavailabilityReason != nil {
				summary.TimingUnavailabilityReasons[*scenario.UnavailabilityReason]++
			}
		}
	}
	for weapon := range weapons {
		summary.ActiveWeapons = append(summary.ActiveWeapons, weapon)
	}
	sort.Strings(summary.ActiveWeapons)
	for speed := range weaponSpeeds {
		summary.WeaponBaselineSpeeds = append(summary.WeaponBaselineSpeeds, speed)
	}
	sort.Float64s(summary.WeaponBaselineSpeeds)
	return summary
}

func validateObservedNavigationSmokeReport(report ObservedNavigationSmokeReport) error {
	minimumRows := minNavigationSmoke(report.QueryLimit, 8)
	if len(report.Rows) < minimumRows {
		return fmt.Errorf(
			"build navigation smoke: produced %d causal same-team queries, require at least %d",
			len(report.Rows),
			minimumRows,
		)
	}
	if report.Summary.ReachableRows == 0 {
		return fmt.Errorf("build navigation smoke: no real query produced a reachable NAV path")
	}
	if report.Summary.IdealBaselineRows != report.Summary.ReachableRows {
		return fmt.Errorf("build navigation smoke: optimistic baseline coverage differs from reachable paths")
	}
	if report.Summary.ActiveWeaponBaselineRows == 0 {
		return fmt.Errorf("build navigation smoke: no known current weapon produced an internal baseline")
	}
	if report.Summary.ObservedPaceProjectionRows == 0 {
		return fmt.Errorf("build navigation smoke: no causal observed speed produced a pace projection")
	}
	if report.Summary.WalkingContexts == 0 || report.Summary.NonWalkingMovingContexts == 0 {
		return fmt.Errorf("build navigation smoke: real walking and non-walking movement contexts are both required")
	}
	if report.Summary.KnifeContexts == 0 {
		return fmt.Errorf("build navigation smoke: no real Knife context exercised the weapon-speed lookup")
	}
	if len(report.Summary.ActiveWeapons) < 2 || len(report.Summary.WeaponBaselineSpeeds) < 2 {
		return fmt.Errorf("build navigation smoke: at least two real weapons with distinct baseline speeds are required")
	}
	if report.Summary.ExactUserFacingTimeViolations != 0 {
		return fmt.Errorf("build navigation smoke: exact user-facing timing policy was violated")
	}
	for _, row := range report.Rows {
		if err := validateObservedNavigationPath(row); err != nil {
			return fmt.Errorf("build navigation smoke: invalid candidate row %s: %w", row.JoinKeys.QueryID, err)
		}
	}
	return nil
}

func validLowerSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
