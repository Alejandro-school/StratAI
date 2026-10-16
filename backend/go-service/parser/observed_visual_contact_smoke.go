package parser

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/visualcontact"

	"github.com/golang/geo/r3"
)

const (
	ObservedVisualContactSmokeSchemaID          = "stratai.observed_visual_contact_single_demo_smoke@1"
	ObservedVisualContactSmokeSelectionContract = "causal_round_prefix_with_smoke@1"

	ObservedVisualContactSmokePhysicsGLTFRole   = "physics_gltf"
	ObservedVisualContactSmokePhysicsBufferRole = "physics_buffer"

	minimumObservedVisualContactSmokeFrames = 16
	maximumObservedVisualContactSmokeFrames = 256
)

// ObservedVisualContactSmokeAsset records the exact geometry inputs validated
// by the isolated caller. References must be safe relative paths; absolute host
// paths are deliberately excluded from this non-canonical report.
type ObservedVisualContactSmokeAsset struct {
	Reference string `json:"reference"`
	SHA256    string `json:"sha256"`
	Role      string `json:"role"`
}

// ObservedVisualContactSmokeSelection describes the real, contiguous causal
// prefix passed to the candidate producer. It always begins at the source
// round boundary and is bounded so the smoke never materializes a whole match.
type ObservedVisualContactSmokeSelection struct {
	Contract         string `json:"contract"`
	RoundNumber      int    `json:"round_number"`
	StartTick        int    `json:"start_tick"`
	EndTick          int    `json:"end_tick"`
	T0Tick           int    `json:"t0_tick"`
	InputRoundFrames int    `json:"input_round_frames"`
	SelectedFrames   int    `json:"selected_frames"`
	FrameLimit       int    `json:"frame_limit"`
}

type ObservedVisualContactSmokeSummary struct {
	Evaluations              int            `json:"evaluations"`
	ObservedEvaluations      int            `json:"observed_evaluations"`
	UnavailableEvaluations   int            `json:"unavailable_evaluations"`
	LineOfSightClear         int            `json:"line_of_sight_clear"`
	LineOfSightBlocked       int            `json:"line_of_sight_blocked"`
	LineOfSightUnavailable   int            `json:"line_of_sight_unavailable"`
	FOVInside                int            `json:"fov_inside"`
	FOVOutside               int            `json:"fov_outside"`
	FOVUnavailable           int            `json:"fov_unavailable"`
	SmokeOccluded            int            `json:"smoke_occluded"`
	SmokeClear               int            `json:"smoke_clear"`
	SmokeUnavailable         int            `json:"smoke_unavailable"`
	VisualContactTrue        int            `json:"visual_contact_true"`
	VisualContactFalse       int            `json:"visual_contact_false"`
	VisualContactUnavailable int            `json:"visual_contact_unavailable"`
	ActiveSmokeEvaluations   int            `json:"active_smoke_evaluations"`
	Boundaries               map[string]int `json:"boundaries"`
	Episodes                 int            `json:"episodes"`
	ClosedEpisodes           int            `json:"closed_episodes"`
	OpenEpisodes             int            `json:"open_episodes"`
	ParticipantAliases       int            `json:"participant_aliases"`
	AbstentionReasons        map[string]int `json:"abstention_reasons"`
	RawSteamIDViolations     int            `json:"raw_steam_id_violations"`
}

// ObservedVisualContactSmokeReport is a bounded, non-canonical validation
// artifact. It is written beside, never inside, a canonical 3.8 bundle.
type ObservedVisualContactSmokeReport struct {
	SchemaID                 string                              `json:"schema_id"`
	Status                   string                              `json:"status"`
	SourceAlias              string                              `json:"source_alias"`
	SourceSHA256             string                              `json:"source_sha256"`
	MapName                  string                              `json:"map_name"`
	CandidateSchemaID        string                              `json:"candidate_schema_id"`
	CandidateContractVersion string                              `json:"candidate_contract_version"`
	BuildIdentifier          string                              `json:"build_identifier"`
	ProcessingDurationMS     int64                               `json:"processing_duration_ms"`
	Selection                ObservedVisualContactSmokeSelection `json:"selection"`
	GeometryAssets           []ObservedVisualContactSmokeAsset   `json:"geometry_assets"`
	Summary                  ObservedVisualContactSmokeSummary   `json:"summary"`
	Ledger                   models.ObservedVisualContactLedger  `json:"ledger"`
}

type observedVisualContactSmokeWindow struct {
	round            models.ReplayRound
	frames           []models.ReplayFrame
	inputRoundFrames int
}

// BuildObservedVisualContactSmokeReport invokes the real @1 producer on
// deterministic causal round prefixes, in round order, until one exercises the
// complete smoke-gate coverage. Starting every candidate at the real round
// boundary preserves contact warm-up and prevents artificial started events at
// a centered-window cut. The caller supplies already-hashed geometry assets
// because this package neither owns nor re-opens the map loader's files.
func BuildObservedVisualContactSmokeReport(
	replay *models.ReplayData,
	geometry TacticalGeometry,
	sourceAlias, sourceSHA256 string,
	frameLimit int,
	geometryAssets []ObservedVisualContactSmokeAsset,
) (ObservedVisualContactSmokeReport, error) {
	if replay == nil {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("build visual contact smoke: replay data is nil")
	}
	if !validObservedVisualContactSmokeIdentity(sourceAlias, sourceSHA256) {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("build visual contact smoke: source alias or sha256 is invalid")
	}
	if strings.TrimSpace(replay.Metadata.MapName) == "" {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("build visual contact smoke: map name is empty")
	}
	if frameLimit < minimumObservedVisualContactSmokeFrames || frameLimit > maximumObservedVisualContactSmokeFrames {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf(
			"build visual contact smoke: frame limit must be between %d and %d",
			minimumObservedVisualContactSmokeFrames,
			maximumObservedVisualContactSmokeFrames,
		)
	}
	if geometry == nil || !geometry.IsLoaded() {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("build visual contact smoke: raycast geometry is unavailable")
	}
	assets, err := canonicalObservedVisualContactSmokeAssets(geometryAssets)
	if err != nil {
		return ObservedVisualContactSmokeReport{}, err
	}

	windows, err := observedVisualContactSmokeCandidateWindows(replay, frameLimit)
	if err != nil {
		return ObservedVisualContactSmokeReport{}, err
	}
	rejections := make([]string, 0, len(windows))
	for _, window := range windows {
		report, candidateErr := buildObservedVisualContactSmokeCandidate(
			replay,
			geometry,
			sourceAlias,
			sourceSHA256,
			frameLimit,
			assets,
			window,
		)
		if candidateErr == nil {
			return report, nil
		}
		rejections = append(rejections, fmt.Sprintf("round %d: %v", window.round.Round, candidateErr))
	}
	return ObservedVisualContactSmokeReport{}, fmt.Errorf(
		"build visual contact smoke: no causal round prefix passed required coverage: %s",
		strings.Join(rejections, "; "),
	)
}

func buildObservedVisualContactSmokeCandidate(
	replay *models.ReplayData,
	geometry TacticalGeometry,
	sourceAlias, sourceSHA256 string,
	frameLimit int,
	assets []ObservedVisualContactSmokeAsset,
	window observedVisualContactSmokeWindow,
) (ObservedVisualContactSmokeReport, error) {
	windowReplay, aliases := observedVisualContactSmokeReplay(replay, window, sourceAlias)
	ledger, err := BuildObservedVisualContactLedger(windowReplay, geometry, window.round.EndTick)
	if err != nil {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("candidate producer: %w", err)
	}
	if err := pseudonymizeObservedVisualContactSmokeLedger(&ledger, aliases); err != nil {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("pseudonymize ledger: %w", err)
	}
	if err := ValidateObservedVisualContactLedger(ledger); err != nil {
		return ObservedVisualContactSmokeReport{}, fmt.Errorf("pseudonymized ledger: %w", err)
	}

	report := ObservedVisualContactSmokeReport{
		SchemaID:                 ObservedVisualContactSmokeSchemaID,
		Status:                   "passed",
		SourceAlias:              sourceAlias,
		SourceSHA256:             sourceSHA256,
		MapName:                  strings.TrimSpace(replay.Metadata.MapName),
		CandidateSchemaID:        models.ObservedVisualContactSchemaID,
		CandidateContractVersion: models.ObservedVisualContactContractVersion,
		Selection: ObservedVisualContactSmokeSelection{
			Contract:         ObservedVisualContactSmokeSelectionContract,
			RoundNumber:      window.round.Round,
			StartTick:        window.round.StartTick,
			EndTick:          window.round.EndTick,
			T0Tick:           window.round.EndTick,
			InputRoundFrames: window.inputRoundFrames,
			SelectedFrames:   len(window.frames),
			FrameLimit:       frameLimit,
		},
		GeometryAssets: assets,
		Ledger:         ledger,
	}
	report.Summary = summarizeObservedVisualContactSmoke(ledger)
	if err := ValidateObservedVisualContactSmokeReport(report); err != nil {
		return ObservedVisualContactSmokeReport{}, err
	}
	return report, nil
}

func validObservedVisualContactSmokeIdentity(sourceAlias, sourceSHA256 string) bool {
	if len(sourceSHA256) != 64 || sourceSHA256 != strings.ToLower(sourceSHA256) {
		return false
	}
	for _, character := range sourceSHA256 {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return sourceAlias == "demo-"+sourceSHA256[:20]
}

func canonicalObservedVisualContactSmokeAssets(
	values []ObservedVisualContactSmokeAsset,
) ([]ObservedVisualContactSmokeAsset, error) {
	result := append([]ObservedVisualContactSmokeAsset(nil), values...)
	for index := range result {
		result[index].Reference = strings.TrimSpace(result[index].Reference)
		result[index].Role = strings.TrimSpace(result[index].Role)
		if result[index].Reference == "" || strings.HasPrefix(result[index].Reference, "/") ||
			strings.Contains(result[index].Reference, "\\") || strings.Contains(result[index].Reference, ":") ||
			result[index].Reference == ".." || strings.HasPrefix(result[index].Reference, "../") ||
			strings.Contains(result[index].Reference, "/../") {
			return nil, fmt.Errorf("build visual contact smoke: geometry asset %d has an unsafe reference", index)
		}
		if !validObservedVisualContactSmokeSHA256(result[index].SHA256) {
			return nil, fmt.Errorf("build visual contact smoke: geometry asset %d has an invalid sha256", index)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Role != result[j].Role {
			return result[i].Role < result[j].Role
		}
		return result[i].Reference < result[j].Reference
	})
	seen := make(map[string]struct{}, len(result))
	for _, asset := range result {
		if asset.Role != ObservedVisualContactSmokePhysicsGLTFRole &&
			asset.Role != ObservedVisualContactSmokePhysicsBufferRole {
			return nil, fmt.Errorf("build visual contact smoke: unsupported geometry asset role %q", asset.Role)
		}
		if _, exists := seen[asset.Role]; exists {
			return nil, fmt.Errorf("build visual contact smoke: duplicate geometry asset role %q", asset.Role)
		}
		seen[asset.Role] = struct{}{}
	}
	for _, required := range []string{
		ObservedVisualContactSmokePhysicsGLTFRole,
		ObservedVisualContactSmokePhysicsBufferRole,
	} {
		if _, exists := seen[required]; !exists {
			return nil, fmt.Errorf("build visual contact smoke: missing geometry asset role %q", required)
		}
	}
	return result, nil
}

func validObservedVisualContactSmokeSHA256(value string) bool {
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

func observedVisualContactSmokeCandidateWindows(
	replay *models.ReplayData,
	frameLimit int,
) ([]observedVisualContactSmokeWindow, error) {
	rounds := append([]models.ReplayRound(nil), replay.Rounds...)
	sort.SliceStable(rounds, func(i, j int) bool {
		if rounds[i].Round != rounds[j].Round {
			return rounds[i].Round < rounds[j].Round
		}
		if rounds[i].StartTick != rounds[j].StartTick {
			return rounds[i].StartTick < rounds[j].StartTick
		}
		return rounds[i].EndTick < rounds[j].EndTick
	})

	windows := make([]observedVisualContactSmokeWindow, 0, len(rounds))
	for _, sourceRound := range rounds {
		if sourceRound.Round <= 0 || sourceRound.EndTick < sourceRound.StartTick {
			continue
		}
		frames := append([]models.ReplayFrame(nil), sourceRound.Frames...)
		sort.SliceStable(frames, func(i, j int) bool { return frames[i].Tick < frames[j].Tick })
		bounded := frames[:0]
		for _, frame := range frames {
			if frame.Tick >= sourceRound.StartTick && frame.Tick <= sourceRound.EndTick {
				bounded = append(bounded, frame)
			}
		}
		if len(bounded) == 0 {
			continue
		}
		selectedCount := frameLimit
		if len(bounded) < selectedCount {
			selectedCount = len(bounded)
		}
		selected := append([]models.ReplayFrame(nil), bounded[:selectedCount]...)
		hasEligibleSmoke := false
		for _, frame := range selected {
			if observedVisualContactSmokeFrameEligible(frame) {
				hasEligibleSmoke = true
				break
			}
		}
		if !hasEligibleSmoke {
			continue
		}
		windowRound := models.ReplayRound{
			Round:     sourceRound.Round,
			StartTick: sourceRound.StartTick,
			EndTick:   selected[len(selected)-1].Tick,
			Frames:    selected,
		}
		windows = append(windows, observedVisualContactSmokeWindow{
			round:            windowRound,
			frames:           selected,
			inputRoundFrames: len(sourceRound.Frames),
		})
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf(
			"build visual contact smoke: no causal round prefix contains an eligible active-smoke frame",
		)
	}
	return windows, nil
}

func observedVisualContactSmokeFrameEligible(frame models.ReplayFrame) bool {
	if frame.ActiveEffects == nil || !observedVisualContactSmokeHasValidSmoke(frame) {
		return false
	}
	for _, observer := range frame.Players {
		if !observedVisualContactSmokeObserverEligible(observer) {
			continue
		}
		observerTeam := canonicalTacticalTeam(observer.Team)
		for _, subject := range frame.Players {
			if observer.SteamID == subject.SteamID || subject.SteamID == 0 || !subject.Alive ||
				canonicalTacticalTeam(subject.Team) == "" || canonicalTacticalTeam(subject.Team) == observerTeam ||
				!observedVisualContactSmokeEyesAvailable(subject) {
				continue
			}
			return true
		}
	}
	return false
}

func observedVisualContactSmokeHasValidSmoke(frame models.ReplayFrame) bool {
	for _, effect := range frame.ActiveEffects {
		if strings.EqualFold(strings.TrimSpace(effect.Type), "smoke") &&
			effect.StartTick <= frame.Tick && visualcontact.IsFinite(effect.X) &&
			visualcontact.IsFinite(effect.Y) && visualcontact.IsFinite(effect.Radius) && effect.Radius > 0 {
			return true
		}
	}
	return false
}

func observedVisualContactSmokeObserverEligible(player models.ReplayPlayerState) bool {
	if player.SteamID == 0 || !player.Alive || canonicalTacticalTeam(player.Team) == "" ||
		!observedVisualContactSmokeEyesAvailable(player) || !player.ViewAnglesAvailable ||
		!visualcontact.IsFinite(float64(player.Yaw)) || !visualcontact.IsFinite(float64(player.Pitch)) {
		return false
	}
	return !player.IsScoped || player.ZoomLevelAvailable
}

func observedVisualContactSmokeEyesAvailable(player models.ReplayPlayerState) bool {
	return player.EyePositionAvailable && visualcontact.IsFiniteVector(r3.Vector{
		X: player.EyeX,
		Y: player.EyeY,
		Z: player.EyeZ,
	})
}

func observedVisualContactSmokeReplay(
	replay *models.ReplayData,
	window observedVisualContactSmokeWindow,
	sourceAlias string,
) (*models.ReplayData, map[string]string) {
	identities := make(map[uint64]struct{})
	for _, frame := range window.frames {
		for _, player := range frame.Players {
			if player.SteamID != 0 {
				identities[player.SteamID] = struct{}{}
			}
		}
	}
	ordered := make([]uint64, 0, len(identities))
	for identity := range identities {
		ordered = append(ordered, identity)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	syntheticByRaw := make(map[uint64]uint64, len(ordered))
	aliases := make(map[string]string, len(ordered))
	for index, identity := range ordered {
		synthetic := uint64(index + 1)
		syntheticByRaw[identity] = synthetic
		aliases[tacticalPlayerID(synthetic)] = fmt.Sprintf("participant-%03d", index+1)
	}

	frames := make([]models.ReplayFrame, len(window.frames))
	for frameIndex, frame := range window.frames {
		players := make([]models.ReplayPlayerState, len(frame.Players))
		for playerIndex, player := range frame.Players {
			player.Name = ""
			if player.SteamID != 0 {
				player.SteamID = syntheticByRaw[player.SteamID]
			}
			players[playerIndex] = player
		}
		var activeEffects []models.ReplayActiveEffect
		if frame.ActiveEffects != nil {
			activeEffects = make([]models.ReplayActiveEffect, len(frame.ActiveEffects))
			copy(activeEffects, frame.ActiveEffects)
		}
		frames[frameIndex] = models.ReplayFrame{
			Tick:          frame.Tick,
			Players:       players,
			ActiveEffects: activeEffects,
		}
	}

	metadata := replay.Metadata
	metadata.MatchID = sourceAlias
	return &models.ReplayData{
		Metadata: metadata,
		Rounds: []models.ReplayRound{{
			Round:     window.round.Round,
			StartTick: window.round.StartTick,
			EndTick:   window.round.EndTick,
			Frames:    frames,
		}},
	}, aliases
}

func pseudonymizeObservedVisualContactSmokeLedger(
	ledger *models.ObservedVisualContactLedger,
	aliases map[string]string,
) error {
	if ledger == nil {
		return fmt.Errorf("ledger is nil")
	}
	contactAliases := make(map[string]string, len(ledger.Contacts))
	for index, episode := range ledger.Contacts {
		if episode.JoinKeys.ContactID == nil || strings.TrimSpace(*episode.JoinKeys.ContactID) == "" {
			return fmt.Errorf("contact episode %d has no contact id", index)
		}
		contactAliases[*episode.JoinKeys.ContactID] = fmt.Sprintf("contact-%03d", index+1)
	}

	applyJoinKeys := func(keys *models.ObservedVisualContactJoinKeys) error {
		observer, observerKnown := aliases[keys.ObserverID]
		subject, subjectKnown := aliases[keys.SubjectID]
		if !observerKnown || !subjectKnown {
			return fmt.Errorf("participant alias is unavailable")
		}
		keys.ObserverID = observer
		keys.SubjectID = subject
		if keys.ContactID != nil {
			contact, exists := contactAliases[*keys.ContactID]
			if !exists {
				return fmt.Errorf("contact alias is unavailable")
			}
			keys.ContactID = copyVisualContactString(&contact)
		}
		return nil
	}
	for index := range ledger.Evaluations {
		if err := applyJoinKeys(&ledger.Evaluations[index].JoinKeys); err != nil {
			return fmt.Errorf("evaluation %d: %w", index, err)
		}
	}
	for index := range ledger.Contacts {
		if err := applyJoinKeys(&ledger.Contacts[index].JoinKeys); err != nil {
			return fmt.Errorf("episode %d: %w", index, err)
		}
	}
	return nil
}

func summarizeObservedVisualContactSmoke(
	ledger models.ObservedVisualContactLedger,
) ObservedVisualContactSmokeSummary {
	summary := ObservedVisualContactSmokeSummary{
		Boundaries:        make(map[string]int),
		AbstentionReasons: make(map[string]int),
	}
	participants := make(map[string]struct{})
	for _, evaluation := range ledger.Evaluations {
		summary.Evaluations++
		if evaluation.Status == models.TacticalStatusObserved {
			summary.ObservedEvaluations++
		} else {
			summary.UnavailableEvaluations++
		}
		countObservedVisualContactSmokeFact(
			evaluation.LineOfSight.Value,
			&summary.LineOfSightClear,
			&summary.LineOfSightBlocked,
			&summary.LineOfSightUnavailable,
		)
		countObservedVisualContactSmokeFact(
			evaluation.FieldOfView.Inside,
			&summary.FOVInside,
			&summary.FOVOutside,
			&summary.FOVUnavailable,
		)
		countObservedVisualContactSmokeFact(
			evaluation.SmokeOcclusion.Occluded,
			&summary.SmokeOccluded,
			&summary.SmokeClear,
			&summary.SmokeUnavailable,
		)
		countObservedVisualContactSmokeFact(
			evaluation.VisualContact.Value,
			&summary.VisualContactTrue,
			&summary.VisualContactFalse,
			&summary.VisualContactUnavailable,
		)
		if evaluation.SmokeOcclusion.ActiveSmokeCount != nil &&
			*evaluation.SmokeOcclusion.ActiveSmokeCount > 0 {
			summary.ActiveSmokeEvaluations++
		}
		summary.Boundaries[evaluation.Boundary]++
		for _, reason := range evaluation.AbstentionReasons {
			summary.AbstentionReasons[reason]++
		}
		participants[evaluation.JoinKeys.ObserverID] = struct{}{}
		participants[evaluation.JoinKeys.SubjectID] = struct{}{}
		summary.RawSteamIDViolations += observedVisualContactSmokeRawIDViolations(evaluation.JoinKeys)
	}
	for _, episode := range ledger.Contacts {
		summary.Episodes++
		if episode.Status == models.VisualContactEpisodeClosed {
			summary.ClosedEpisodes++
		} else if episode.Status == models.VisualContactEpisodeOpen {
			summary.OpenEpisodes++
		}
		participants[episode.JoinKeys.ObserverID] = struct{}{}
		participants[episode.JoinKeys.SubjectID] = struct{}{}
		summary.RawSteamIDViolations += observedVisualContactSmokeRawIDViolations(episode.JoinKeys)
	}
	summary.ParticipantAliases = len(participants)
	return summary
}

func countObservedVisualContactSmokeFact(value *bool, positive, negative, unavailable *int) {
	if value == nil {
		(*unavailable)++
		return
	}
	if *value {
		(*positive)++
	} else {
		(*negative)++
	}
}

func observedVisualContactSmokeRawIDViolations(keys models.ObservedVisualContactJoinKeys) int {
	violations := 0
	for _, value := range []string{keys.ObserverID, keys.SubjectID} {
		if strings.Contains(strings.ToLower(value), "steam:") {
			violations++
		}
	}
	if keys.ContactID != nil && strings.Contains(strings.ToLower(*keys.ContactID), "steam:") {
		violations++
	}
	return violations
}

// ValidateObservedVisualContactSmokeReport recomputes the report counts and
// enforces that a real smoke exercised every positive/negative branch needed
// to trust this candidate before continuing to another phase.
func ValidateObservedVisualContactSmokeReport(report ObservedVisualContactSmokeReport) error {
	if report.SchemaID != ObservedVisualContactSmokeSchemaID || report.Status != "passed" {
		return fmt.Errorf("build visual contact smoke: unsupported report schema or status")
	}
	if !validObservedVisualContactSmokeIdentity(report.SourceAlias, report.SourceSHA256) ||
		strings.TrimSpace(report.MapName) == "" {
		return fmt.Errorf("build visual contact smoke: invalid source identity or map")
	}
	if report.CandidateSchemaID != models.ObservedVisualContactSchemaID ||
		report.CandidateContractVersion != models.ObservedVisualContactContractVersion {
		return fmt.Errorf("build visual contact smoke: candidate schema or version mismatch")
	}
	if report.ProcessingDurationMS < 0 ||
		(report.BuildIdentifier != "" && strings.TrimSpace(report.BuildIdentifier) == "") {
		return fmt.Errorf("build visual contact smoke: invalid processing metadata")
	}
	selection := report.Selection
	if selection.Contract != ObservedVisualContactSmokeSelectionContract || selection.RoundNumber <= 0 ||
		selection.StartTick < 0 || selection.EndTick < selection.StartTick ||
		selection.T0Tick != selection.EndTick || selection.SelectedFrames < 3 ||
		selection.SelectedFrames > selection.FrameLimit ||
		selection.InputRoundFrames < selection.SelectedFrames ||
		selection.FrameLimit < minimumObservedVisualContactSmokeFrames ||
		selection.FrameLimit > maximumObservedVisualContactSmokeFrames {
		return fmt.Errorf("build visual contact smoke: invalid causal round-prefix selection")
	}
	assets, err := canonicalObservedVisualContactSmokeAssets(report.GeometryAssets)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(assets, report.GeometryAssets) {
		return fmt.Errorf("build visual contact smoke: geometry assets are not canonical")
	}
	if report.Ledger.JoinKeys.MatchID != report.SourceAlias ||
		report.Ledger.T0Tick != selection.T0Tick {
		return fmt.Errorf("build visual contact smoke: ledger identity or T0 differs from selection")
	}
	if err := ValidateObservedVisualContactLedger(report.Ledger); err != nil {
		return fmt.Errorf("build visual contact smoke: invalid candidate ledger: %w", err)
	}
	if err := validateObservedVisualContactSmokeAliases(report.Ledger); err != nil {
		return err
	}
	for index, evaluation := range report.Ledger.Evaluations {
		if evaluation.RoundNumber != selection.RoundNumber || evaluation.Tick < selection.StartTick ||
			evaluation.Tick > selection.EndTick || evaluation.T0Tick != selection.T0Tick ||
			evaluation.Provenance.RaycastMethod != visualcontact.RaycastMethodVersion ||
			evaluation.Provenance.FOVMethod != visualcontact.FOVMethodVersion ||
			evaluation.Provenance.SmokeMethod != visualcontact.SmokeMethodVersion ||
			!reflect.DeepEqual(evaluation.Provenance.InputFields, visualContactInputFields) {
			return fmt.Errorf("build visual contact smoke: evaluation %d has invalid prefix or provenance", index)
		}
	}
	recomputed := summarizeObservedVisualContactSmoke(report.Ledger)
	if !reflect.DeepEqual(recomputed, report.Summary) {
		return fmt.Errorf("build visual contact smoke: summary does not match candidate ledger")
	}
	if report.Summary.Evaluations == 0 || report.Summary.ObservedEvaluations == 0 {
		return fmt.Errorf("build visual contact smoke: no observed real evaluations")
	}
	coverage := []struct {
		name  string
		value int
	}{
		{"line_of_sight_clear", report.Summary.LineOfSightClear},
		{"line_of_sight_blocked", report.Summary.LineOfSightBlocked},
		{"fov_inside", report.Summary.FOVInside},
		{"fov_outside", report.Summary.FOVOutside},
		{"smoke_occluded", report.Summary.SmokeOccluded},
		{"smoke_clear", report.Summary.SmokeClear},
		{"visual_contact_true", report.Summary.VisualContactTrue},
		{"visual_contact_false", report.Summary.VisualContactFalse},
		{"active_smoke_evaluations", report.Summary.ActiveSmokeEvaluations},
		{"episodes", report.Summary.Episodes},
		{"boundary_started", report.Summary.Boundaries[models.VisualContactBoundaryStarted]},
		{"boundary_ended", report.Summary.Boundaries[models.VisualContactBoundaryEnded]},
		{"boundary_inactive", report.Summary.Boundaries[models.VisualContactBoundaryInactive]},
	}
	for _, requirement := range coverage {
		if requirement.value == 0 {
			return fmt.Errorf("build visual contact smoke: required coverage %s is zero", requirement.name)
		}
	}
	if report.Summary.ParticipantAliases < 2 || report.Summary.RawSteamIDViolations != 0 {
		return fmt.Errorf("build visual contact smoke: participant privacy coverage failed")
	}
	return nil
}

func validateObservedVisualContactSmokeAliases(ledger models.ObservedVisualContactLedger) error {
	contacts := make(map[string]struct{}, len(ledger.Contacts))
	for index, episode := range ledger.Contacts {
		if !validObservedVisualContactSmokeParticipantAlias(episode.JoinKeys.ObserverID) ||
			!validObservedVisualContactSmokeParticipantAlias(episode.JoinKeys.SubjectID) ||
			episode.JoinKeys.ContactID == nil || !validObservedVisualContactSmokeContactAlias(*episode.JoinKeys.ContactID) {
			return fmt.Errorf("build visual contact smoke: episode %d contains a non-pseudonymous join key", index)
		}
		contacts[*episode.JoinKeys.ContactID] = struct{}{}
	}
	for index, evaluation := range ledger.Evaluations {
		if !validObservedVisualContactSmokeParticipantAlias(evaluation.JoinKeys.ObserverID) ||
			!validObservedVisualContactSmokeParticipantAlias(evaluation.JoinKeys.SubjectID) {
			return fmt.Errorf("build visual contact smoke: evaluation %d contains a non-pseudonymous participant", index)
		}
		if evaluation.JoinKeys.ContactID != nil {
			if !validObservedVisualContactSmokeContactAlias(*evaluation.JoinKeys.ContactID) {
				return fmt.Errorf("build visual contact smoke: evaluation %d contains a non-pseudonymous contact", index)
			}
			if _, exists := contacts[*evaluation.JoinKeys.ContactID]; !exists {
				return fmt.Errorf("build visual contact smoke: evaluation %d references an unknown contact", index)
			}
		}
	}
	return nil
}

func validObservedVisualContactSmokeParticipantAlias(value string) bool {
	return validObservedVisualContactSmokeNumberedAlias(value, "participant-")
}

func validObservedVisualContactSmokeContactAlias(value string) bool {
	return validObservedVisualContactSmokeNumberedAlias(value, "contact-")
}

func validObservedVisualContactSmokeNumberedAlias(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	digits := strings.TrimPrefix(value, prefix)
	if len(digits) < 3 {
		return false
	}
	parsed, err := strconv.Atoi(digits)
	return err == nil && parsed > 0 && fmt.Sprintf("%03d", parsed) == digits
}
