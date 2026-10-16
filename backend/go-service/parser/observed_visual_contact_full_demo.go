package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/visualcontact"
)

const (
	ObservedVisualContactFullDemoSchemaID            = "stratai.observed_visual_contact_full_demo_validation@1"
	ObservedVisualContactFullDemoSelectionContract   = "all_valid_replay_rounds_per_round_t0@1"
	ObservedVisualContactFullDemoAuditRecordSchemaID = "stratai.observed_visual_contact_full_demo_audit_record@1"
	ObservedVisualContactFullDemoReportFilename      = "observed_visual_contact_full_demo.json"
	ObservedVisualContactFullDemoAuditFilename       = "observed_visual_contact_full_demo_audit.jsonl.gz"

	ObservedVisualContactFullDemoAuditEncoding    = "jsonl"
	ObservedVisualContactFullDemoAuditCompression = "gzip"

	ObservedVisualContactFullDemoEvaluationRecord = "evaluation"
	ObservedVisualContactFullDemoEpisodeRecord    = "episode"
)

var fullDemoBoundaryNames = []string{
	models.VisualContactBoundaryStarted,
	models.VisualContactBoundaryContinued,
	models.VisualContactBoundaryEnded,
	models.VisualContactBoundaryInactive,
	models.VisualContactBoundaryUnavailable,
}

// ObservedVisualContactFullDemoSelection makes the diagnostic scope explicit.
// T0 is deliberately not global: every round is evaluated with its own real
// EndTick, which bounds memory and preserves the candidate producer semantics.
type ObservedVisualContactFullDemoSelection struct {
	Contract              string `json:"contract"`
	RoundCount            int    `json:"round_count"`
	FirstRound            int    `json:"first_round"`
	LastRound             int    `json:"last_round"`
	MinimumStartTick      int    `json:"minimum_start_tick"`
	MaximumEndTick        int    `json:"maximum_end_tick"`
	InputSourceFrames     int    `json:"input_source_frames"`
	CanonicalSourceFrames int    `json:"canonical_source_frames"`
	PerRoundT0            string `json:"per_round_t0"`
}

// ObservedVisualContactFullDemoMetrics contains only additive counts. The
// names distinguish pair-sample smoke intersections from cases where smoke is
// the final gate preventing an otherwise geometrically visible contact.
type ObservedVisualContactFullDemoMetrics struct {
	Evaluations                                int            `json:"evaluations"`
	ObservedEvaluations                        int            `json:"observed_evaluations"`
	UnavailableEvaluations                     int            `json:"unavailable_evaluations"`
	LineOfSightClearPairSamples                int            `json:"line_of_sight_clear_pair_samples"`
	LineOfSightBlockedPairSamples              int            `json:"line_of_sight_blocked_pair_samples"`
	LineOfSightUnavailablePairSamples          int            `json:"line_of_sight_unavailable_pair_samples"`
	FOVInsidePairSamples                       int            `json:"fov_inside_pair_samples"`
	FOVOutsidePairSamples                      int            `json:"fov_outside_pair_samples"`
	FOVUnavailablePairSamples                  int            `json:"fov_unavailable_pair_samples"`
	SmokePairSampleIntersections               int            `json:"smoke_pair_sample_intersections"`
	SmokeClearPairSamples                      int            `json:"smoke_clear_pair_samples"`
	SmokeUnavailablePairSamples                int            `json:"smoke_unavailable_pair_samples"`
	ActiveSmokePairSampleEvaluations           int            `json:"active_smoke_pair_sample_evaluations"`
	OtherwiseVisiblePairSamplesBlockedBySmoke  int            `json:"otherwise_visible_pair_samples_blocked_by_smoke"`
	VisualContactPairSamples                   int            `json:"visual_contact_pair_samples"`
	NoVisualContactPairSamples                 int            `json:"no_visual_contact_pair_samples"`
	VisualContactUnavailablePairSamples        int            `json:"visual_contact_unavailable_pair_samples"`
	InputSourceFrames                          int            `json:"input_source_frames"`
	CanonicalSourceFrames                      int            `json:"canonical_source_frames"`
	SourceFramesWithValidActiveSmoke           int            `json:"source_frames_with_valid_active_smoke"`
	EvaluatedSourceFrames                      int            `json:"evaluated_source_frames"`
	EvaluatedSourceFramesWithActiveSmoke       int            `json:"evaluated_source_frames_with_active_smoke"`
	EvaluatedSourceFramesWithSmokeIntersection int            `json:"evaluated_source_frames_with_smoke_intersection"`
	UniqueValidActiveSmokeEffects              int            `json:"unique_valid_active_smoke_effects"`
	DirectionalEpisodes                        int            `json:"directional_episodes"`
	ClosedDirectionalEpisodes                  int            `json:"closed_directional_episodes"`
	RightCensoredDirectionalEpisodes           int            `json:"right_censored_directional_episodes"`
	Boundaries                                 map[string]int `json:"boundaries"`
	AbstentionReasons                          map[string]int `json:"abstention_reasons"`
	RawSteamIDViolations                       int            `json:"raw_steam_id_violations"`
}

type ObservedVisualContactFullDemoRoundSummary struct {
	RoundNumber                          int                                  `json:"round_number"`
	StartTick                            int                                  `json:"start_tick"`
	EndTick                              int                                  `json:"end_tick"`
	Metrics                              ObservedVisualContactFullDemoMetrics `json:"metrics"`
	ParticipantsEvaluated                int                                  `json:"participants_evaluated"`
	ParticipantsWithContact              int                                  `json:"participants_with_contact"`
	UniqueDirectedPairsWithContact       int                                  `json:"unique_directed_pairs_with_contact"`
	UniqueUnorderedPairsWithContact      int                                  `json:"unique_unordered_pairs_with_contact"`
	UniqueDirectedPairsSmokeIntersected  int                                  `json:"unique_directed_pairs_smoke_intersected"`
	UniqueUnorderedPairsSmokeIntersected int                                  `json:"unique_unordered_pairs_smoke_intersected"`
	ReacquisitionEpisodesWithinRound     int                                  `json:"reacquisition_episodes_within_round"`
	AuditLogicalRecords                  int                                  `json:"audit_logical_records"`
	AuditLogicalSHA256                   string                               `json:"audit_logical_sha256"`
}

type ObservedVisualContactFullDemoSummary struct {
	Metrics                                   ObservedVisualContactFullDemoMetrics `json:"metrics"`
	ParticipantAliases                        int                                  `json:"participant_aliases"`
	ParticipantsEvaluated                     int                                  `json:"participants_evaluated"`
	ParticipantsWithContact                   int                                  `json:"participants_with_contact"`
	UniqueMatchDirectedPairsWithContact       int                                  `json:"unique_match_directed_pairs_with_contact"`
	UniqueMatchUnorderedPairsWithContact      int                                  `json:"unique_match_unordered_pairs_with_contact"`
	UniqueRoundDirectedPairsWithContact       int                                  `json:"unique_round_directed_pairs_with_contact"`
	UniqueMatchDirectedPairsSmokeIntersected  int                                  `json:"unique_match_directed_pairs_smoke_intersected"`
	UniqueMatchUnorderedPairsSmokeIntersected int                                  `json:"unique_match_unordered_pairs_smoke_intersected"`
	ReacquisitionEpisodesWithinRound          int                                  `json:"reacquisition_episodes_within_round"`
	RoundsWithEvaluations                     int                                  `json:"rounds_with_evaluations"`
	RoundsWithVisualContact                   int                                  `json:"rounds_with_visual_contact"`
	RoundsWithValidActiveSmoke                int                                  `json:"rounds_with_valid_active_smoke"`
	RoundsWithSmokeIntersection               int                                  `json:"rounds_with_smoke_intersection"`
	RoundsWithOtherwiseVisibleSmokeBlock      int                                  `json:"rounds_with_otherwise_visible_smoke_block"`
}

// ObservedVisualContactFullDemoAudit describes the logical JSONL emitted by
// this package. The caller owns compression and fills the gzip fields only
// after it has closed the gzip writer and hashed the physical artifact.
type ObservedVisualContactFullDemoAudit struct {
	RecordSchemaID    string `json:"record_schema_id"`
	Encoding          string `json:"encoding"`
	Compression       string `json:"compression"`
	EvaluationRecords int    `json:"evaluation_records"`
	EpisodeRecords    int    `json:"episode_records"`
	LogicalRecords    int    `json:"logical_records"`
	LogicalBytes      int64  `json:"logical_bytes"`
	LogicalSHA256     string `json:"logical_sha256"`
	GzipRelativePath  string `json:"gzip_relative_path"`
	GzipBytes         int64  `json:"gzip_bytes"`
	GzipSHA256        string `json:"gzip_sha256"`
}

// ObservedVisualContactFullDemoReport is compact: every detailed candidate
// row is written to the audit stream and no full ledger is retained here.
type ObservedVisualContactFullDemoReport struct {
	SchemaID                 string                                      `json:"schema_id"`
	Status                   string                                      `json:"status"`
	SourceAlias              string                                      `json:"source_alias"`
	SourceSHA256             string                                      `json:"source_sha256"`
	MapName                  string                                      `json:"map_name"`
	CandidateSchemaID        string                                      `json:"candidate_schema_id"`
	CandidateContractVersion string                                      `json:"candidate_contract_version"`
	TrainingAllowed          bool                                        `json:"training_allowed"`
	BuildIdentifier          string                                      `json:"build_identifier"`
	ProcessingDurationMS     int64                                       `json:"processing_duration_ms"`
	Selection                ObservedVisualContactFullDemoSelection      `json:"selection"`
	Sampling                 models.TacticalSampling                     `json:"sampling"`
	GeometryAssets           []ObservedVisualContactSmokeAsset           `json:"geometry_assets"`
	Summary                  ObservedVisualContactFullDemoSummary        `json:"summary"`
	Rounds                   []ObservedVisualContactFullDemoRoundSummary `json:"rounds"`
	Audit                    ObservedVisualContactFullDemoAudit          `json:"audit"`
}

type observedVisualContactFullDemoAuditEnvelope struct {
	RecordType string `json:"record_type"`
	Round      int    `json:"round"`
	Value      any    `json:"value"`
}

type observedVisualContactFullDemoSets struct {
	participantsEvaluated       map[string]struct{}
	participantsWithContact     map[string]struct{}
	directedContactPairs        map[string]struct{}
	unorderedContactPairs       map[string]struct{}
	directedSmokePairs          map[string]struct{}
	unorderedSmokePairs         map[string]struct{}
	evaluatedFrames             map[int]struct{}
	framesWithActiveSmoke       map[int]struct{}
	framesWithSmokeIntersection map[int]struct{}
}

func newObservedVisualContactFullDemoSets() observedVisualContactFullDemoSets {
	return observedVisualContactFullDemoSets{
		participantsEvaluated:       make(map[string]struct{}),
		participantsWithContact:     make(map[string]struct{}),
		directedContactPairs:        make(map[string]struct{}),
		unorderedContactPairs:       make(map[string]struct{}),
		directedSmokePairs:          make(map[string]struct{}),
		unorderedSmokePairs:         make(map[string]struct{}),
		evaluatedFrames:             make(map[int]struct{}),
		framesWithActiveSmoke:       make(map[int]struct{}),
		framesWithSmokeIntersection: make(map[int]struct{}),
	}
}

type observedVisualContactFullDemoAuditEmitter struct {
	writer            io.Writer
	globalHash        hash.Hash
	globalRecords     int
	globalBytes       int64
	evaluationRecords int
	episodeRecords    int
}

func newObservedVisualContactFullDemoAuditEmitter(writer io.Writer) (*observedVisualContactFullDemoAuditEmitter, error) {
	if writer == nil {
		return nil, fmt.Errorf("build full-demo visual contact: audit writer is required")
	}
	return &observedVisualContactFullDemoAuditEmitter{writer: writer, globalHash: sha256.New()}, nil
}

func (emitter *observedVisualContactFullDemoAuditEmitter) emit(
	recordType string,
	round int,
	value any,
	roundHash hash.Hash,
) error {
	payload, err := json.Marshal(observedVisualContactFullDemoAuditEnvelope{
		RecordType: recordType,
		Round:      round,
		Value:      value,
	})
	if err != nil {
		return fmt.Errorf("encode %s audit record: %w", recordType, err)
	}
	payload = append(payload, '\n')
	written, err := emitter.writer.Write(payload)
	if err != nil {
		return fmt.Errorf("write %s audit record: %w", recordType, err)
	}
	if written != len(payload) {
		return fmt.Errorf("write %s audit record: %w", recordType, io.ErrShortWrite)
	}
	_, _ = emitter.globalHash.Write(payload)
	_, _ = roundHash.Write(payload)
	emitter.globalRecords++
	emitter.globalBytes += int64(written)
	if recordType == ObservedVisualContactFullDemoEvaluationRecord {
		emitter.evaluationRecords++
	} else if recordType == ObservedVisualContactFullDemoEpisodeRecord {
		emitter.episodeRecords++
	}
	return nil
}

// BuildObservedVisualContactFullDemoReport executes the real @1 candidate on
// every source round independently. It retains one round ledger at a time and
// streams all pseudonymous detailed rows to auditWriter in deterministic order.
func BuildObservedVisualContactFullDemoReport(
	replay *models.ReplayData,
	geometry TacticalGeometry,
	sourceAlias, sourceSHA256 string,
	geometryAssets []ObservedVisualContactSmokeAsset,
	auditWriter io.Writer,
) (ObservedVisualContactFullDemoReport, error) {
	if replay == nil {
		return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: replay data is nil")
	}
	if !validObservedVisualContactSmokeIdentity(sourceAlias, sourceSHA256) {
		return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: source alias or sha256 is invalid")
	}
	mapName := strings.TrimSpace(replay.Metadata.MapName)
	if mapName == "" {
		return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: map name is empty")
	}
	if geometry == nil || !geometry.IsLoaded() {
		return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: loaded raycast geometry is required")
	}
	assets, err := canonicalObservedVisualContactSmokeAssets(geometryAssets)
	if err != nil {
		return ObservedVisualContactFullDemoReport{}, err
	}
	rounds, err := canonicalObservedVisualContactFullDemoRounds(replay.Rounds)
	if err != nil {
		return ObservedVisualContactFullDemoReport{}, err
	}
	syntheticByRaw, participantAliases := observedVisualContactFullDemoAliases(rounds)
	emitter, err := newObservedVisualContactFullDemoAuditEmitter(auditWriter)
	if err != nil {
		return ObservedVisualContactFullDemoReport{}, err
	}

	report := ObservedVisualContactFullDemoReport{
		SchemaID:                 ObservedVisualContactFullDemoSchemaID,
		Status:                   "passed",
		SourceAlias:              sourceAlias,
		SourceSHA256:             sourceSHA256,
		MapName:                  mapName,
		CandidateSchemaID:        models.ObservedVisualContactSchemaID,
		CandidateContractVersion: models.ObservedVisualContactContractVersion,
		TrainingAllowed:          false,
		Selection: ObservedVisualContactFullDemoSelection{
			Contract:         ObservedVisualContactFullDemoSelectionContract,
			RoundCount:       len(rounds),
			FirstRound:       rounds[0].Round,
			LastRound:        rounds[len(rounds)-1].Round,
			MinimumStartTick: rounds[0].StartTick,
			MaximumEndTick:   rounds[0].EndTick,
			PerRoundT0:       "round_end_tick",
		},
		Sampling: models.TacticalSampling{
			TargetHz:    models.TacticalTargetHz,
			TickRate:    replay.Metadata.TickRate,
			PeriodTicks: replay.Metadata.TickRate / float64(models.TacticalTargetHz),
			Strategy:    models.TacticalSamplingStrategy,
		},
		GeometryAssets: assets,
		Rounds:         make([]ObservedVisualContactFullDemoRoundSummary, 0, len(rounds)),
	}
	report.Summary.Metrics = newObservedVisualContactFullDemoMetrics()
	report.Summary.ParticipantAliases = len(participantAliases)

	globalSets := newObservedVisualContactFullDemoSets()
	globalRoundDirectedPairs := make(map[string]struct{})
	globalSmokeEffects := make(map[string]struct{})
	contactSequence := 0
	for _, round := range rounds {
		roundReplay, err := observedVisualContactFullDemoRoundReplay(
			replay,
			round,
			sourceAlias,
			syntheticByRaw,
		)
		if err != nil {
			return ObservedVisualContactFullDemoReport{}, err
		}
		ledger, err := BuildObservedVisualContactLedger(roundReplay, geometry, round.EndTick)
		if err != nil {
			return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d producer: %w", round.Round, err)
		}
		if err := ValidateObservedVisualContactLedger(ledger); err != nil {
			return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d producer validation: %w", round.Round, err)
		}
		if err := pseudonymizeObservedVisualContactFullDemoLedger(
			&ledger,
			participantAliases,
			&contactSequence,
		); err != nil {
			return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d privacy: %w", round.Round, err)
		}
		if err := validateObservedVisualContactFullDemoLedgerAliases(ledger, sourceAlias); err != nil {
			return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d privacy: %w", round.Round, err)
		}
		if err := ValidateObservedVisualContactLedger(ledger); err != nil {
			return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d pseudonymous validation: %w", round.Round, err)
		}

		roundSummary, roundSets, roundSmokeEffects, err := summarizeObservedVisualContactFullDemoRound(round, ledger)
		if err != nil {
			return ObservedVisualContactFullDemoReport{}, err
		}
		roundHash := sha256.New()
		for index := range ledger.Evaluations {
			if err := emitter.emit(
				ObservedVisualContactFullDemoEvaluationRecord,
				round.Round,
				ledger.Evaluations[index],
				roundHash,
			); err != nil {
				return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d: %w", round.Round, err)
			}
		}
		for index := range ledger.Contacts {
			if err := emitter.emit(
				ObservedVisualContactFullDemoEpisodeRecord,
				round.Round,
				ledger.Contacts[index],
				roundHash,
			); err != nil {
				return ObservedVisualContactFullDemoReport{}, fmt.Errorf("build full-demo visual contact: round %d: %w", round.Round, err)
			}
		}
		roundSummary.AuditLogicalRecords = len(ledger.Evaluations) + len(ledger.Contacts)
		roundSummary.AuditLogicalSHA256 = hex.EncodeToString(roundHash.Sum(nil))
		report.Rounds = append(report.Rounds, roundSummary)

		addObservedVisualContactFullDemoMetrics(&report.Summary.Metrics, roundSummary.Metrics)
		mergeObservedVisualContactFullDemoSets(&globalSets, roundSets)
		for pair := range roundSets.directedContactPairs {
			globalRoundDirectedPairs[fmt.Sprintf("%d:%s", round.Round, pair)] = struct{}{}
		}
		for effect := range roundSmokeEffects {
			globalSmokeEffects[fmt.Sprintf("%d:%s", round.Round, effect)] = struct{}{}
		}
		if roundSummary.Metrics.Evaluations > 0 {
			report.Summary.RoundsWithEvaluations++
		}
		if roundSummary.Metrics.VisualContactPairSamples > 0 {
			report.Summary.RoundsWithVisualContact++
		}
		if roundSummary.Metrics.SourceFramesWithValidActiveSmoke > 0 {
			report.Summary.RoundsWithValidActiveSmoke++
		}
		if roundSummary.Metrics.SmokePairSampleIntersections > 0 {
			report.Summary.RoundsWithSmokeIntersection++
		}
		if roundSummary.Metrics.OtherwiseVisiblePairSamplesBlockedBySmoke > 0 {
			report.Summary.RoundsWithOtherwiseVisibleSmokeBlock++
		}
		if round.StartTick < report.Selection.MinimumStartTick {
			report.Selection.MinimumStartTick = round.StartTick
		}
		if round.EndTick > report.Selection.MaximumEndTick {
			report.Selection.MaximumEndTick = round.EndTick
		}
	}

	report.Selection.InputSourceFrames = report.Summary.Metrics.InputSourceFrames
	report.Selection.CanonicalSourceFrames = report.Summary.Metrics.CanonicalSourceFrames
	report.Summary.Metrics.UniqueValidActiveSmokeEffects = len(globalSmokeEffects)
	report.Summary.ParticipantsEvaluated = len(globalSets.participantsEvaluated)
	report.Summary.ParticipantsWithContact = len(globalSets.participantsWithContact)
	report.Summary.UniqueMatchDirectedPairsWithContact = len(globalSets.directedContactPairs)
	report.Summary.UniqueMatchUnorderedPairsWithContact = len(globalSets.unorderedContactPairs)
	report.Summary.UniqueRoundDirectedPairsWithContact = len(globalRoundDirectedPairs)
	report.Summary.UniqueMatchDirectedPairsSmokeIntersected = len(globalSets.directedSmokePairs)
	report.Summary.UniqueMatchUnorderedPairsSmokeIntersected = len(globalSets.unorderedSmokePairs)
	report.Summary.ReacquisitionEpisodesWithinRound =
		report.Summary.Metrics.DirectionalEpisodes - report.Summary.UniqueRoundDirectedPairsWithContact
	report.Audit = ObservedVisualContactFullDemoAudit{
		RecordSchemaID:    ObservedVisualContactFullDemoAuditRecordSchemaID,
		Encoding:          ObservedVisualContactFullDemoAuditEncoding,
		Compression:       ObservedVisualContactFullDemoAuditCompression,
		EvaluationRecords: emitter.evaluationRecords,
		EpisodeRecords:    emitter.episodeRecords,
		LogicalRecords:    emitter.globalRecords,
		LogicalBytes:      emitter.globalBytes,
		LogicalSHA256:     hex.EncodeToString(emitter.globalHash.Sum(nil)),
	}
	if err := validateObservedVisualContactFullDemoReport(report, true); err != nil {
		return ObservedVisualContactFullDemoReport{}, err
	}
	return report, nil
}

func canonicalObservedVisualContactFullDemoRounds(values []models.ReplayRound) ([]models.ReplayRound, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("build full-demo visual contact: no replay rounds")
	}
	rounds := append([]models.ReplayRound(nil), values...)
	sort.SliceStable(rounds, func(i, j int) bool {
		if rounds[i].Round != rounds[j].Round {
			return rounds[i].Round < rounds[j].Round
		}
		if rounds[i].StartTick != rounds[j].StartTick {
			return rounds[i].StartTick < rounds[j].StartTick
		}
		return rounds[i].EndTick < rounds[j].EndTick
	})
	seen := make(map[int]struct{}, len(rounds))
	for index, round := range rounds {
		if round.Round <= 0 || round.StartTick < 0 || round.EndTick < round.StartTick {
			return nil, fmt.Errorf("build full-demo visual contact: invalid replay round at index %d", index)
		}
		if _, exists := seen[round.Round]; exists {
			return nil, fmt.Errorf("build full-demo visual contact: duplicate round number %d", round.Round)
		}
		seen[round.Round] = struct{}{}
	}
	return rounds, nil
}

func observedVisualContactFullDemoAliases(
	rounds []models.ReplayRound,
) (map[uint64]uint64, map[string]string) {
	rawSet := make(map[uint64]struct{})
	for _, round := range rounds {
		for _, frame := range round.Frames {
			for _, player := range frame.Players {
				if player.SteamID != 0 {
					rawSet[player.SteamID] = struct{}{}
				}
			}
		}
	}
	rawIDs := make([]uint64, 0, len(rawSet))
	for rawID := range rawSet {
		rawIDs = append(rawIDs, rawID)
	}
	sort.Slice(rawIDs, func(i, j int) bool { return rawIDs[i] < rawIDs[j] })
	syntheticByRaw := make(map[uint64]uint64, len(rawIDs))
	aliases := make(map[string]string, len(rawIDs))
	for index, rawID := range rawIDs {
		synthetic := uint64(index + 1)
		syntheticByRaw[rawID] = synthetic
		aliases[tacticalPlayerID(synthetic)] = fmt.Sprintf("participant-%03d", index+1)
	}
	return syntheticByRaw, aliases
}

func observedVisualContactFullDemoRoundReplay(
	replay *models.ReplayData,
	round models.ReplayRound,
	sourceAlias string,
	syntheticByRaw map[uint64]uint64,
) (*models.ReplayData, error) {
	frames := make([]models.ReplayFrame, len(round.Frames))
	for frameIndex, frame := range round.Frames {
		players := make([]models.ReplayPlayerState, len(frame.Players))
		for playerIndex, player := range frame.Players {
			player.Name = ""
			if player.SteamID != 0 {
				synthetic, exists := syntheticByRaw[player.SteamID]
				if !exists {
					return nil, fmt.Errorf("round %d frame %d: participant alias is unavailable", round.Round, frame.Tick)
				}
				player.SteamID = synthetic
			}
			players[playerIndex] = player
		}
		var effects []models.ReplayActiveEffect
		if frame.ActiveEffects != nil {
			effects = make([]models.ReplayActiveEffect, len(frame.ActiveEffects))
			copy(effects, frame.ActiveEffects)
		}
		frames[frameIndex] = models.ReplayFrame{
			Tick:          frame.Tick,
			Players:       players,
			ActiveEffects: effects,
		}
	}
	metadata := replay.Metadata
	metadata.MatchID = sourceAlias
	return &models.ReplayData{
		Metadata: metadata,
		Rounds: []models.ReplayRound{{
			Round:     round.Round,
			StartTick: round.StartTick,
			EndTick:   round.EndTick,
			Frames:    frames,
		}},
	}, nil
}

func pseudonymizeObservedVisualContactFullDemoLedger(
	ledger *models.ObservedVisualContactLedger,
	participantAliases map[string]string,
	contactSequence *int,
) error {
	if ledger == nil || contactSequence == nil {
		return fmt.Errorf("ledger or contact sequence is nil")
	}
	contactAliases := make(map[string]string, len(ledger.Contacts))
	for index := range ledger.Contacts {
		contactID := ledger.Contacts[index].JoinKeys.ContactID
		if contactID == nil || strings.TrimSpace(*contactID) == "" {
			return fmt.Errorf("episode %d has no contact id", index)
		}
		(*contactSequence)++
		contactAliases[*contactID] = fmt.Sprintf("contact-%06d", *contactSequence)
	}
	apply := func(keys *models.ObservedVisualContactJoinKeys) error {
		observer, observerExists := participantAliases[keys.ObserverID]
		subject, subjectExists := participantAliases[keys.SubjectID]
		if !observerExists || !subjectExists {
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
		if err := apply(&ledger.Evaluations[index].JoinKeys); err != nil {
			return fmt.Errorf("evaluation %d: %w", index, err)
		}
	}
	for index := range ledger.Contacts {
		if err := apply(&ledger.Contacts[index].JoinKeys); err != nil {
			return fmt.Errorf("episode %d: %w", index, err)
		}
	}
	return nil
}

func validateObservedVisualContactFullDemoLedgerAliases(
	ledger models.ObservedVisualContactLedger,
	sourceAlias string,
) error {
	if ledger.JoinKeys.MatchID != sourceAlias {
		return fmt.Errorf("ledger match identity differs from the opaque source alias")
	}
	validateKeys := func(keys models.ObservedVisualContactJoinKeys) error {
		if keys.MatchID != sourceAlias ||
			!validObservedVisualContactFullDemoAlias(keys.ObserverID, "participant-", 3) ||
			!validObservedVisualContactFullDemoAlias(keys.SubjectID, "participant-", 3) {
			return fmt.Errorf("participant join key is not a stable opaque alias")
		}
		if keys.ContactID != nil &&
			!validObservedVisualContactFullDemoAlias(*keys.ContactID, "contact-", 6) {
			return fmt.Errorf("contact join key is not a stable opaque alias")
		}
		return nil
	}
	for index, evaluation := range ledger.Evaluations {
		if err := validateKeys(evaluation.JoinKeys); err != nil {
			return fmt.Errorf("evaluation %d: %w", index, err)
		}
	}
	for index, episode := range ledger.Contacts {
		if episode.JoinKeys.ContactID == nil {
			return fmt.Errorf("episode %d: contact alias is missing", index)
		}
		if err := validateKeys(episode.JoinKeys); err != nil {
			return fmt.Errorf("episode %d: %w", index, err)
		}
	}
	return nil
}

func newObservedVisualContactFullDemoMetrics() ObservedVisualContactFullDemoMetrics {
	boundaries := make(map[string]int, len(fullDemoBoundaryNames))
	for _, boundary := range fullDemoBoundaryNames {
		boundaries[boundary] = 0
	}
	return ObservedVisualContactFullDemoMetrics{
		Boundaries:        boundaries,
		AbstentionReasons: make(map[string]int),
	}
}

func summarizeObservedVisualContactFullDemoRound(
	round models.ReplayRound,
	ledger models.ObservedVisualContactLedger,
) (
	ObservedVisualContactFullDemoRoundSummary,
	observedVisualContactFullDemoSets,
	map[string]struct{},
	error,
) {
	metrics := newObservedVisualContactFullDemoMetrics()
	metrics.InputSourceFrames = len(round.Frames)
	canonicalFrames, err := canonicalVisualContactFrames(round, round.EndTick)
	if err != nil {
		return ObservedVisualContactFullDemoRoundSummary{}, observedVisualContactFullDemoSets{}, nil,
			fmt.Errorf("build full-demo visual contact: round %d source frames: %w", round.Round, err)
	}
	metrics.CanonicalSourceFrames = len(canonicalFrames)
	uniqueSmokes := make(map[string]struct{})
	for _, frame := range canonicalFrames {
		hasValidSmoke := false
		for _, smoke := range frame.activeEffects {
			if smoke.startTick <= frame.tick && visualcontact.IsFinite(smoke.x) &&
				visualcontact.IsFinite(smoke.y) && visualcontact.IsFinite(smoke.radius) && smoke.radius > 0 {
				hasValidSmoke = true
				uniqueSmokes[smoke.key] = struct{}{}
			}
		}
		if hasValidSmoke {
			metrics.SourceFramesWithValidActiveSmoke++
		}
	}
	metrics.UniqueValidActiveSmokeEffects = len(uniqueSmokes)

	sets := newObservedVisualContactFullDemoSets()
	for _, evaluation := range ledger.Evaluations {
		metrics.Evaluations++
		if evaluation.Status == models.TacticalStatusObserved {
			metrics.ObservedEvaluations++
		} else {
			metrics.UnavailableEvaluations++
		}
		countObservedVisualContactFullDemoFact(
			evaluation.LineOfSight.Value,
			&metrics.LineOfSightClearPairSamples,
			&metrics.LineOfSightBlockedPairSamples,
			&metrics.LineOfSightUnavailablePairSamples,
		)
		countObservedVisualContactFullDemoFact(
			evaluation.FieldOfView.Inside,
			&metrics.FOVInsidePairSamples,
			&metrics.FOVOutsidePairSamples,
			&metrics.FOVUnavailablePairSamples,
		)
		countObservedVisualContactFullDemoFact(
			evaluation.SmokeOcclusion.Occluded,
			&metrics.SmokePairSampleIntersections,
			&metrics.SmokeClearPairSamples,
			&metrics.SmokeUnavailablePairSamples,
		)
		countObservedVisualContactFullDemoFact(
			evaluation.VisualContact.Value,
			&metrics.VisualContactPairSamples,
			&metrics.NoVisualContactPairSamples,
			&metrics.VisualContactUnavailablePairSamples,
		)
		metrics.Boundaries[evaluation.Boundary]++
		for _, reason := range evaluation.AbstentionReasons {
			metrics.AbstentionReasons[reason]++
		}
		sets.participantsEvaluated[evaluation.JoinKeys.ObserverID] = struct{}{}
		sets.participantsEvaluated[evaluation.JoinKeys.SubjectID] = struct{}{}
		sets.evaluatedFrames[evaluation.Provenance.SourceFrameTick] = struct{}{}
		directed := observedVisualContactFullDemoDirectedPair(evaluation.JoinKeys.ObserverID, evaluation.JoinKeys.SubjectID)
		unordered := observedVisualContactFullDemoUnorderedPair(evaluation.JoinKeys.ObserverID, evaluation.JoinKeys.SubjectID)
		if evaluation.SmokeOcclusion.ActiveSmokeCount != nil && *evaluation.SmokeOcclusion.ActiveSmokeCount > 0 {
			metrics.ActiveSmokePairSampleEvaluations++
			sets.framesWithActiveSmoke[evaluation.Provenance.SourceFrameTick] = struct{}{}
		}
		if evaluation.SmokeOcclusion.Occluded != nil && *evaluation.SmokeOcclusion.Occluded {
			sets.framesWithSmokeIntersection[evaluation.Provenance.SourceFrameTick] = struct{}{}
			sets.directedSmokePairs[directed] = struct{}{}
			sets.unorderedSmokePairs[unordered] = struct{}{}
			if evaluation.LineOfSight.Value != nil && *evaluation.LineOfSight.Value &&
				evaluation.FieldOfView.Inside != nil && *evaluation.FieldOfView.Inside {
				metrics.OtherwiseVisiblePairSamplesBlockedBySmoke++
			}
		}
		if evaluation.VisualContact.Value != nil && *evaluation.VisualContact.Value {
			sets.participantsWithContact[evaluation.JoinKeys.ObserverID] = struct{}{}
			sets.participantsWithContact[evaluation.JoinKeys.SubjectID] = struct{}{}
			sets.directedContactPairs[directed] = struct{}{}
			sets.unorderedContactPairs[unordered] = struct{}{}
		}
		metrics.RawSteamIDViolations += observedVisualContactSmokeRawIDViolations(evaluation.JoinKeys)
	}
	metrics.EvaluatedSourceFrames = len(sets.evaluatedFrames)
	metrics.EvaluatedSourceFramesWithActiveSmoke = len(sets.framesWithActiveSmoke)
	metrics.EvaluatedSourceFramesWithSmokeIntersection = len(sets.framesWithSmokeIntersection)
	for _, episode := range ledger.Contacts {
		metrics.DirectionalEpisodes++
		if episode.Status == models.VisualContactEpisodeClosed {
			metrics.ClosedDirectionalEpisodes++
		} else if episode.Status == models.VisualContactEpisodeOpen {
			metrics.RightCensoredDirectionalEpisodes++
		}
		metrics.RawSteamIDViolations += observedVisualContactSmokeRawIDViolations(episode.JoinKeys)
	}
	summary := ObservedVisualContactFullDemoRoundSummary{
		RoundNumber:                          round.Round,
		StartTick:                            round.StartTick,
		EndTick:                              round.EndTick,
		Metrics:                              metrics,
		ParticipantsEvaluated:                len(sets.participantsEvaluated),
		ParticipantsWithContact:              len(sets.participantsWithContact),
		UniqueDirectedPairsWithContact:       len(sets.directedContactPairs),
		UniqueUnorderedPairsWithContact:      len(sets.unorderedContactPairs),
		UniqueDirectedPairsSmokeIntersected:  len(sets.directedSmokePairs),
		UniqueUnorderedPairsSmokeIntersected: len(sets.unorderedSmokePairs),
		ReacquisitionEpisodesWithinRound:     metrics.DirectionalEpisodes - len(sets.directedContactPairs),
	}
	if err := validateObservedVisualContactFullDemoRoundSummary(summary); err != nil {
		return ObservedVisualContactFullDemoRoundSummary{}, observedVisualContactFullDemoSets{}, nil, err
	}
	return summary, sets, uniqueSmokes, nil
}

func countObservedVisualContactFullDemoFact(value *bool, positive, negative, unavailable *int) {
	if value == nil {
		(*unavailable)++
	} else if *value {
		(*positive)++
	} else {
		(*negative)++
	}
}

func observedVisualContactFullDemoDirectedPair(observer, subject string) string {
	return observer + "\x00" + subject
}

func observedVisualContactFullDemoUnorderedPair(left, right string) string {
	if right < left {
		left, right = right, left
	}
	return left + "\x00" + right
}

func mergeObservedVisualContactFullDemoSets(target *observedVisualContactFullDemoSets, source observedVisualContactFullDemoSets) {
	merge := func(targetMap, sourceMap map[string]struct{}) {
		for value := range sourceMap {
			targetMap[value] = struct{}{}
		}
	}
	merge(target.participantsEvaluated, source.participantsEvaluated)
	merge(target.participantsWithContact, source.participantsWithContact)
	merge(target.directedContactPairs, source.directedContactPairs)
	merge(target.unorderedContactPairs, source.unorderedContactPairs)
	merge(target.directedSmokePairs, source.directedSmokePairs)
	merge(target.unorderedSmokePairs, source.unorderedSmokePairs)
}

func addObservedVisualContactFullDemoMetrics(target *ObservedVisualContactFullDemoMetrics, value ObservedVisualContactFullDemoMetrics) {
	target.Evaluations += value.Evaluations
	target.ObservedEvaluations += value.ObservedEvaluations
	target.UnavailableEvaluations += value.UnavailableEvaluations
	target.LineOfSightClearPairSamples += value.LineOfSightClearPairSamples
	target.LineOfSightBlockedPairSamples += value.LineOfSightBlockedPairSamples
	target.LineOfSightUnavailablePairSamples += value.LineOfSightUnavailablePairSamples
	target.FOVInsidePairSamples += value.FOVInsidePairSamples
	target.FOVOutsidePairSamples += value.FOVOutsidePairSamples
	target.FOVUnavailablePairSamples += value.FOVUnavailablePairSamples
	target.SmokePairSampleIntersections += value.SmokePairSampleIntersections
	target.SmokeClearPairSamples += value.SmokeClearPairSamples
	target.SmokeUnavailablePairSamples += value.SmokeUnavailablePairSamples
	target.ActiveSmokePairSampleEvaluations += value.ActiveSmokePairSampleEvaluations
	target.OtherwiseVisiblePairSamplesBlockedBySmoke += value.OtherwiseVisiblePairSamplesBlockedBySmoke
	target.VisualContactPairSamples += value.VisualContactPairSamples
	target.NoVisualContactPairSamples += value.NoVisualContactPairSamples
	target.VisualContactUnavailablePairSamples += value.VisualContactUnavailablePairSamples
	target.InputSourceFrames += value.InputSourceFrames
	target.CanonicalSourceFrames += value.CanonicalSourceFrames
	target.SourceFramesWithValidActiveSmoke += value.SourceFramesWithValidActiveSmoke
	target.EvaluatedSourceFrames += value.EvaluatedSourceFrames
	target.EvaluatedSourceFramesWithActiveSmoke += value.EvaluatedSourceFramesWithActiveSmoke
	target.EvaluatedSourceFramesWithSmokeIntersection += value.EvaluatedSourceFramesWithSmokeIntersection
	target.UniqueValidActiveSmokeEffects += value.UniqueValidActiveSmokeEffects
	target.DirectionalEpisodes += value.DirectionalEpisodes
	target.ClosedDirectionalEpisodes += value.ClosedDirectionalEpisodes
	target.RightCensoredDirectionalEpisodes += value.RightCensoredDirectionalEpisodes
	target.RawSteamIDViolations += value.RawSteamIDViolations
	for key, count := range value.Boundaries {
		target.Boundaries[key] += count
	}
	for key, count := range value.AbstentionReasons {
		target.AbstentionReasons[key] += count
	}
}

func validateObservedVisualContactFullDemoMetrics(metrics ObservedVisualContactFullDemoMetrics, location string) error {
	metricValue := reflect.ValueOf(metrics)
	metricType := metricValue.Type()
	for index := 0; index < metricValue.NumField(); index++ {
		if metricValue.Field(index).Kind() == reflect.Int && metricValue.Field(index).Int() < 0 {
			return fmt.Errorf("%s: metric %s is negative", location, metricType.Field(index).Name)
		}
	}
	if metrics.Evaluations != metrics.ObservedEvaluations+metrics.UnavailableEvaluations ||
		metrics.Evaluations != metrics.LineOfSightClearPairSamples+metrics.LineOfSightBlockedPairSamples+metrics.LineOfSightUnavailablePairSamples ||
		metrics.Evaluations != metrics.FOVInsidePairSamples+metrics.FOVOutsidePairSamples+metrics.FOVUnavailablePairSamples ||
		metrics.Evaluations != metrics.SmokePairSampleIntersections+metrics.SmokeClearPairSamples+metrics.SmokeUnavailablePairSamples ||
		metrics.Evaluations != metrics.VisualContactPairSamples+metrics.NoVisualContactPairSamples+metrics.VisualContactUnavailablePairSamples {
		return fmt.Errorf("%s: evaluation partitions do not reconcile", location)
	}
	if metrics.ObservedEvaluations != metrics.VisualContactPairSamples+metrics.NoVisualContactPairSamples ||
		metrics.UnavailableEvaluations != metrics.VisualContactUnavailablePairSamples {
		return fmt.Errorf("%s: observed/unavailable contact counts do not reconcile", location)
	}
	if len(metrics.Boundaries) != len(fullDemoBoundaryNames) {
		return fmt.Errorf("%s: boundary catalog is incomplete", location)
	}
	for _, boundary := range fullDemoBoundaryNames {
		count, exists := metrics.Boundaries[boundary]
		if !exists || count < 0 {
			return fmt.Errorf("%s: boundary %s is negative", location, boundary)
		}
	}
	for boundary := range metrics.Boundaries {
		if !containsObservedVisualContactFullDemoBoundary(boundary) {
			return fmt.Errorf("%s: unknown boundary %s", location, boundary)
		}
	}
	if metrics.VisualContactPairSamples != metrics.Boundaries[models.VisualContactBoundaryStarted]+metrics.Boundaries[models.VisualContactBoundaryContinued] ||
		metrics.NoVisualContactPairSamples != metrics.Boundaries[models.VisualContactBoundaryInactive]+metrics.Boundaries[models.VisualContactBoundaryEnded] ||
		metrics.VisualContactUnavailablePairSamples != metrics.Boundaries[models.VisualContactBoundaryUnavailable] {
		return fmt.Errorf("%s: contact values and boundaries do not reconcile", location)
	}
	if metrics.DirectionalEpisodes != metrics.Boundaries[models.VisualContactBoundaryStarted] ||
		metrics.ClosedDirectionalEpisodes != metrics.Boundaries[models.VisualContactBoundaryEnded] ||
		metrics.DirectionalEpisodes != metrics.ClosedDirectionalEpisodes+metrics.RightCensoredDirectionalEpisodes {
		return fmt.Errorf("%s: directional episodes do not reconcile", location)
	}
	if metrics.SmokePairSampleIntersections > metrics.ActiveSmokePairSampleEvaluations ||
		metrics.OtherwiseVisiblePairSamplesBlockedBySmoke > metrics.SmokePairSampleIntersections ||
		metrics.OtherwiseVisiblePairSamplesBlockedBySmoke > metrics.NoVisualContactPairSamples ||
		metrics.EvaluatedSourceFrames > metrics.CanonicalSourceFrames ||
		metrics.EvaluatedSourceFramesWithActiveSmoke > metrics.EvaluatedSourceFrames ||
		metrics.EvaluatedSourceFramesWithSmokeIntersection > metrics.EvaluatedSourceFramesWithActiveSmoke ||
		metrics.EvaluatedSourceFramesWithActiveSmoke > metrics.SourceFramesWithValidActiveSmoke ||
		metrics.SourceFramesWithValidActiveSmoke > metrics.CanonicalSourceFrames ||
		metrics.CanonicalSourceFrames > metrics.InputSourceFrames {
		return fmt.Errorf("%s: smoke/frame counts violate their denominators", location)
	}
	if metrics.SmokePairSampleIntersections > 0 && metrics.UniqueValidActiveSmokeEffects == 0 {
		return fmt.Errorf("%s: smoke intersections require a valid active smoke effect", location)
	}
	if metrics.RawSteamIDViolations != 0 {
		return fmt.Errorf("%s: raw Steam identifiers were retained", location)
	}
	for reason, count := range metrics.AbstentionReasons {
		if strings.TrimSpace(reason) == "" || count <= 0 {
			return fmt.Errorf("%s: invalid abstention count", location)
		}
	}
	return nil
}

func validateObservedVisualContactFullDemoRoundSummary(summary ObservedVisualContactFullDemoRoundSummary) error {
	location := fmt.Sprintf("full-demo visual contact round %d", summary.RoundNumber)
	if summary.RoundNumber <= 0 || summary.StartTick < 0 || summary.EndTick < summary.StartTick {
		return fmt.Errorf("%s: invalid round identity or bounds", location)
	}
	if err := validateObservedVisualContactFullDemoMetrics(summary.Metrics, location+".metrics"); err != nil {
		return err
	}
	if summary.ParticipantsEvaluated < 0 || summary.ParticipantsWithContact < 0 ||
		summary.UniqueDirectedPairsWithContact < 0 || summary.UniqueUnorderedPairsWithContact < 0 ||
		summary.UniqueDirectedPairsSmokeIntersected < 0 || summary.UniqueUnorderedPairsSmokeIntersected < 0 ||
		summary.AuditLogicalRecords < 0 ||
		summary.ParticipantsWithContact > summary.ParticipantsEvaluated ||
		summary.UniqueUnorderedPairsWithContact > summary.UniqueDirectedPairsWithContact ||
		summary.UniqueDirectedPairsWithContact > 2*summary.UniqueUnorderedPairsWithContact ||
		summary.UniqueUnorderedPairsSmokeIntersected > summary.UniqueDirectedPairsSmokeIntersected ||
		summary.UniqueDirectedPairsSmokeIntersected > 2*summary.UniqueUnorderedPairsSmokeIntersected ||
		summary.UniqueDirectedPairsWithContact > summary.Metrics.VisualContactPairSamples ||
		summary.UniqueDirectedPairsSmokeIntersected > summary.Metrics.SmokePairSampleIntersections ||
		summary.ReacquisitionEpisodesWithinRound != summary.Metrics.DirectionalEpisodes-summary.UniqueDirectedPairsWithContact ||
		summary.ReacquisitionEpisodesWithinRound < 0 {
		return fmt.Errorf("%s: participant, pair, or reacquisition counts are inconsistent", location)
	}
	return nil
}

func containsObservedVisualContactFullDemoBoundary(candidate string) bool {
	for _, boundary := range fullDemoBoundaryNames {
		if candidate == boundary {
			return true
		}
	}
	return false
}

// ValidateObservedVisualContactFullDemoReport recomputes every compact
// cross-field invariant. The detailed JSONL is validated separately by the
// caller after its gzip artifact has been finalized.
func ValidateObservedVisualContactFullDemoReport(report ObservedVisualContactFullDemoReport) error {
	return validateObservedVisualContactFullDemoReport(report, false)
}

func validateObservedVisualContactFullDemoReport(
	report ObservedVisualContactFullDemoReport,
	allowPendingPhysicalAudit bool,
) error {
	if report.SchemaID != ObservedVisualContactFullDemoSchemaID || report.Status != "passed" {
		return fmt.Errorf("full-demo visual contact: unsupported report schema or status")
	}
	if !validObservedVisualContactSmokeIdentity(report.SourceAlias, report.SourceSHA256) ||
		strings.TrimSpace(report.MapName) == "" || report.TrainingAllowed {
		return fmt.Errorf("full-demo visual contact: invalid source identity, map, or training policy")
	}
	if report.CandidateSchemaID != models.ObservedVisualContactSchemaID ||
		report.CandidateContractVersion != models.ObservedVisualContactContractVersion {
		return fmt.Errorf("full-demo visual contact: candidate schema or version mismatch")
	}
	if report.ProcessingDurationMS < 0 ||
		(!allowPendingPhysicalAudit && strings.TrimSpace(report.BuildIdentifier) == "") ||
		(allowPendingPhysicalAudit && report.BuildIdentifier != "") {
		return fmt.Errorf("full-demo visual contact: invalid processing metadata")
	}
	if report.Sampling.TargetHz != models.TacticalTargetHz ||
		report.Sampling.TickRate < models.TacticalTargetHz ||
		math.IsNaN(report.Sampling.TickRate) || math.IsInf(report.Sampling.TickRate, 0) ||
		report.Sampling.PeriodTicks != report.Sampling.TickRate/float64(models.TacticalTargetHz) ||
		report.Sampling.Strategy != models.TacticalSamplingStrategy {
		return fmt.Errorf("full-demo visual contact: invalid sampling contract")
	}
	assets, err := canonicalObservedVisualContactSmokeAssets(report.GeometryAssets)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(assets, report.GeometryAssets) {
		return fmt.Errorf("full-demo visual contact: geometry assets are not canonical")
	}
	selection := report.Selection
	if selection.Contract != ObservedVisualContactFullDemoSelectionContract ||
		selection.RoundCount <= 0 || selection.RoundCount != len(report.Rounds) ||
		selection.PerRoundT0 != "round_end_tick" || selection.FirstRound <= 0 ||
		selection.LastRound < selection.FirstRound || selection.MinimumStartTick < 0 ||
		selection.MaximumEndTick < selection.MinimumStartTick {
		return fmt.Errorf("full-demo visual contact: invalid complete-demo selection")
	}
	if err := validateObservedVisualContactFullDemoMetrics(report.Summary.Metrics, "full-demo visual contact.summary.metrics"); err != nil {
		return err
	}
	if report.Summary.Metrics.Evaluations == 0 || report.Summary.Metrics.ObservedEvaluations == 0 {
		return fmt.Errorf("full-demo visual contact: no observed real evaluations")
	}

	recomputedMetrics := newObservedVisualContactFullDemoMetrics()
	recomputedRoundDirectedPairs := 0
	recomputedRoundsWithEvaluations := 0
	recomputedRoundsWithContact := 0
	recomputedRoundsWithSmoke := 0
	recomputedRoundsWithIntersection := 0
	recomputedRoundsWithBlocking := 0
	previousRound := 0
	previousEndTick := -1
	minimumStartTick := report.Rounds[0].StartTick
	maximumEndTick := report.Rounds[0].EndTick
	inputFrames := 0
	canonicalFrames := 0
	maximumParticipantsEvaluated := 0
	maximumParticipantsWithContact := 0
	maximumDirectedPairsWithContact := 0
	maximumUnorderedPairsWithContact := 0
	maximumDirectedPairsSmokeIntersected := 0
	maximumUnorderedPairsSmokeIntersected := 0
	for index, round := range report.Rounds {
		if err := validateObservedVisualContactFullDemoRoundSummary(round); err != nil {
			return err
		}
		if round.RoundNumber <= previousRound {
			return fmt.Errorf("full-demo visual contact: rounds are not strictly ordered")
		}
		if index > 0 && round.StartTick < previousEndTick {
			return fmt.Errorf("full-demo visual contact: round tick ranges overlap or regress")
		}
		previousRound = round.RoundNumber
		previousEndTick = round.EndTick
		if index == 0 && round.RoundNumber != selection.FirstRound {
			return fmt.Errorf("full-demo visual contact: first round differs from selection")
		}
		if index == len(report.Rounds)-1 && round.RoundNumber != selection.LastRound {
			return fmt.Errorf("full-demo visual contact: last round differs from selection")
		}
		if !validObservedVisualContactSmokeSHA256(round.AuditLogicalSHA256) ||
			round.AuditLogicalRecords != round.Metrics.Evaluations+round.Metrics.DirectionalEpisodes {
			return fmt.Errorf("full-demo visual contact: round %d audit metadata is invalid", round.RoundNumber)
		}
		addObservedVisualContactFullDemoMetrics(&recomputedMetrics, round.Metrics)
		recomputedRoundDirectedPairs += round.UniqueDirectedPairsWithContact
		inputFrames += round.Metrics.InputSourceFrames
		canonicalFrames += round.Metrics.CanonicalSourceFrames
		maximumParticipantsEvaluated = max(maximumParticipantsEvaluated, round.ParticipantsEvaluated)
		maximumParticipantsWithContact = max(maximumParticipantsWithContact, round.ParticipantsWithContact)
		maximumDirectedPairsWithContact = max(maximumDirectedPairsWithContact, round.UniqueDirectedPairsWithContact)
		maximumUnorderedPairsWithContact = max(maximumUnorderedPairsWithContact, round.UniqueUnorderedPairsWithContact)
		maximumDirectedPairsSmokeIntersected = max(maximumDirectedPairsSmokeIntersected, round.UniqueDirectedPairsSmokeIntersected)
		maximumUnorderedPairsSmokeIntersected = max(maximumUnorderedPairsSmokeIntersected, round.UniqueUnorderedPairsSmokeIntersected)
		if round.Metrics.Evaluations > 0 {
			recomputedRoundsWithEvaluations++
		}
		if round.Metrics.VisualContactPairSamples > 0 {
			recomputedRoundsWithContact++
		}
		if round.Metrics.SourceFramesWithValidActiveSmoke > 0 {
			recomputedRoundsWithSmoke++
		}
		if round.Metrics.SmokePairSampleIntersections > 0 {
			recomputedRoundsWithIntersection++
		}
		if round.Metrics.OtherwiseVisiblePairSamplesBlockedBySmoke > 0 {
			recomputedRoundsWithBlocking++
		}
		if round.StartTick < minimumStartTick {
			minimumStartTick = round.StartTick
		}
		if round.EndTick > maximumEndTick {
			maximumEndTick = round.EndTick
		}
	}
	// Unique smoke effects are scoped by round, so their global value is the
	// additive per-round total. All other additive metrics must match exactly.
	if !reflect.DeepEqual(recomputedMetrics, report.Summary.Metrics) {
		return fmt.Errorf("full-demo visual contact: global metrics differ from per-round totals")
	}
	if selection.MinimumStartTick != minimumStartTick || selection.MaximumEndTick != maximumEndTick ||
		selection.InputSourceFrames != inputFrames || selection.CanonicalSourceFrames != canonicalFrames ||
		report.Summary.UniqueRoundDirectedPairsWithContact != recomputedRoundDirectedPairs ||
		report.Summary.ReacquisitionEpisodesWithinRound != report.Summary.Metrics.DirectionalEpisodes-recomputedRoundDirectedPairs ||
		report.Summary.RoundsWithEvaluations != recomputedRoundsWithEvaluations ||
		report.Summary.RoundsWithVisualContact != recomputedRoundsWithContact ||
		report.Summary.RoundsWithValidActiveSmoke != recomputedRoundsWithSmoke ||
		report.Summary.RoundsWithSmokeIntersection != recomputedRoundsWithIntersection ||
		report.Summary.RoundsWithOtherwiseVisibleSmokeBlock != recomputedRoundsWithBlocking {
		return fmt.Errorf("full-demo visual contact: selection or global round counts do not reconcile")
	}
	if report.Summary.ParticipantAliases < 0 || report.Summary.ParticipantsEvaluated < maximumParticipantsEvaluated ||
		report.Summary.ParticipantsWithContact < maximumParticipantsWithContact ||
		report.Summary.UniqueMatchDirectedPairsWithContact < maximumDirectedPairsWithContact ||
		report.Summary.UniqueMatchUnorderedPairsWithContact < maximumUnorderedPairsWithContact ||
		report.Summary.UniqueMatchDirectedPairsSmokeIntersected < maximumDirectedPairsSmokeIntersected ||
		report.Summary.UniqueMatchUnorderedPairsSmokeIntersected < maximumUnorderedPairsSmokeIntersected ||
		report.Summary.ParticipantsWithContact > report.Summary.ParticipantsEvaluated ||
		report.Summary.ParticipantsEvaluated > report.Summary.ParticipantAliases ||
		report.Summary.UniqueMatchUnorderedPairsWithContact > report.Summary.UniqueMatchDirectedPairsWithContact ||
		report.Summary.UniqueMatchDirectedPairsWithContact > 2*report.Summary.UniqueMatchUnorderedPairsWithContact ||
		report.Summary.UniqueMatchDirectedPairsWithContact > report.Summary.UniqueRoundDirectedPairsWithContact ||
		report.Summary.UniqueMatchUnorderedPairsSmokeIntersected > report.Summary.UniqueMatchDirectedPairsSmokeIntersected ||
		report.Summary.UniqueMatchDirectedPairsSmokeIntersected > 2*report.Summary.UniqueMatchUnorderedPairsSmokeIntersected ||
		report.Summary.UniqueMatchDirectedPairsSmokeIntersected > report.Summary.Metrics.SmokePairSampleIntersections {
		return fmt.Errorf("full-demo visual contact: global participant or pair counts are inconsistent")
	}
	audit := report.Audit
	if audit.RecordSchemaID != ObservedVisualContactFullDemoAuditRecordSchemaID ||
		audit.Encoding != ObservedVisualContactFullDemoAuditEncoding ||
		audit.Compression != ObservedVisualContactFullDemoAuditCompression ||
		audit.EvaluationRecords != report.Summary.Metrics.Evaluations ||
		audit.EpisodeRecords != report.Summary.Metrics.DirectionalEpisodes ||
		audit.LogicalRecords != audit.EvaluationRecords+audit.EpisodeRecords ||
		audit.LogicalBytes < 1 ||
		!validObservedVisualContactSmokeSHA256(audit.LogicalSHA256) {
		return fmt.Errorf("full-demo visual contact: logical audit metadata is invalid")
	}
	physicalPending := audit.GzipRelativePath == "" && audit.GzipBytes == 0 && audit.GzipSHA256 == ""
	physicalComplete := audit.GzipRelativePath != "" && audit.GzipBytes > 0 && validObservedVisualContactSmokeSHA256(audit.GzipSHA256)
	if (!physicalPending && !physicalComplete) ||
		(allowPendingPhysicalAudit && !physicalPending) ||
		(!allowPendingPhysicalAudit && !physicalComplete) {
		return fmt.Errorf("full-demo visual contact: gzip audit metadata must be wholly pending or complete")
	}
	if physicalComplete && (!validObservedVisualContactFullDemoRelativePath(audit.GzipRelativePath) ||
		audit.GzipRelativePath != ObservedVisualContactFullDemoAuditFilename) {
		return fmt.Errorf("full-demo visual contact: gzip audit path is unsafe")
	}
	return nil
}

func validObservedVisualContactFullDemoRelativePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, ":") || strings.HasPrefix(value, "/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validObservedVisualContactFullDemoAlias(value, prefix string, width int) bool {
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	digits := strings.TrimPrefix(value, prefix)
	if len(digits) < width {
		return false
	}
	parsed, err := strconv.Atoi(digits)
	if err != nil || parsed <= 0 {
		return false
	}
	return fmt.Sprintf("%0*d", width, parsed) == digits
}
