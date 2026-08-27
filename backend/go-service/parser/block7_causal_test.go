package parser

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"cs2-demo-service/models"
	engagementpkg "cs2-demo-service/pkg/engagement"
)

func exportBlock7DeterminismFixture(t *testing.T, matchDir string, gomaxprocs int) map[string]string {
	t.Helper()
	previous := runtime.GOMAXPROCS(gomaxprocs)
	defer runtime.GOMAXPROCS(previous)
	ctx := canonicalTestContext()
	matchID := "block7-determinism"
	if err := ExportCanonicalBundle(
		ctx, matchID, matchDir, "2026-08-21T00:00:00Z", 42.5, 64,
		map[string]interface{}{"status": "pass", "usable_for_training": true},
	); err != nil {
		t.Fatal(err)
	}
	eventIDs, shotIDs := canonicalCombatReferenceMaps(ctx.Combat.Snapshot(), matchID)
	if err := exportCanonicalReplayPresentation(matchDir, matchID, ctx.ReplayData, eventIDs, shotIDs); err != nil {
		t.Fatal(err)
	}
	if err := finalizeCanonicalBlock7Quality(matchDir); err != nil {
		t.Fatal(err)
	}
	canonicalDir := filepath.Join(matchDir, "canonical")
	manifest := readCanonicalJSON[models.CanonicalManifest](t, filepath.Join(canonicalDir, "manifest.json"))
	replayCompressionFound := false
	for _, artifact := range manifest.Artifacts {
		if artifact.ArtifactType == "replay_round" {
			replayCompressionFound = true
			if artifact.Compression != "gzip" {
				t.Fatalf("replay compression lost during quality finalization: %+v", artifact)
			}
		}
	}
	if !replayCompressionFound {
		t.Fatal("determinism fixture did not export replay rounds")
	}
	hashes := map[string]string{}
	if err := filepath.Walk(canonicalDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(canonicalDir, path)
		if err != nil {
			return err
		}
		hashes[filepath.ToSlash(relative)] = fmt.Sprintf("%x", sha256.Sum256(data))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return hashes
}

func readBlock7JSONLines[T any](t *testing.T, path string) []T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]T, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var row T
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	return rows
}

func TestCanonicalCausalPartitionsAreAlignedAndLeakageFree(t *testing.T) {
	matchID := "block7-causal"
	matchDir := t.TempDir()
	if err := ExportCanonicalBundle(
		canonicalTestContext(), matchID, matchDir, "2026-08-21T00:00:00Z", 42.5, 64,
		map[string]interface{}{"status": "pass", "usable_for_training": true},
	); err != nil {
		t.Fatal(err)
	}
	causalDir := filepath.Join(matchDir, "canonical", "causal")
	decisions := readBlock7JSONLines[models.CanonicalDecision](t, filepath.Join(causalDir, "decisions.jsonl"))
	features := readBlock7JSONLines[models.CanonicalDecisionFeatures](t, filepath.Join(causalDir, "decision_features.jsonl"))
	oracles := readBlock7JSONLines[models.CanonicalOracleContext](t, filepath.Join(causalDir, "oracle_context.jsonl"))
	outcomes := readBlock7JSONLines[models.CanonicalDecisionOutcome](t, filepath.Join(causalDir, "outcomes.jsonl"))
	masks := readBlock7JSONLines[models.CanonicalQualityMask](t, filepath.Join(causalDir, "quality_masks.jsonl"))
	if len(features) == 0 || len(features) != len(decisions) || len(features) != len(oracles) || len(features) != len(outcomes) || len(features) != len(masks) {
		t.Fatalf("causal partition counts differ: %d/%d/%d/%d/%d", len(decisions), len(features), len(oracles), len(outcomes), len(masks))
	}
	decisionTypes := make(map[string]struct{})
	for index := range features {
		feature := features[index]
		decision := decisions[index]
		if feature.AvailabilityTickMax > feature.T0Tick {
			t.Fatalf("feature %s is available after t0", feature.DecisionID)
		}
		if feature.DecisionID != decision.DecisionID || feature.DecisionID != oracles[index].DecisionID || feature.DecisionID != outcomes[index].DecisionID || feature.DecisionID != masks[index].DecisionID {
			t.Fatalf("partitions are not aligned at row %d", index)
		}
		if decision.ActorPlayerID == "" || decision.ActorIDUsage != "join_only" || decision.ActionTaken == "" ||
			(decision.DecisionType != block7EngagementDecision && decision.DecisionType != block7TradeDecision) ||
			feature.DecisionType != decision.DecisionType {
			t.Fatalf("decision contract is incomplete: %+v", decision)
		}
		decisionTypes[decision.DecisionType] = struct{}{}
		if decision.AvailabilityTick > decision.T0Tick {
			t.Fatalf("decision %s action is available after t0", decision.DecisionID)
		}
		payload, err := json.Marshal(feature)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]interface{}
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatal(err)
		}
		for _, prohibited := range []string{"player_id", "winner", "outcome", "score", "rating", "name"} {
			if _, exists := fields[prohibited]; exists {
				t.Fatalf("decision feature exposes prohibited field %q", prohibited)
			}
		}
		if len(outcomes[index].Horizons) != 3 || !reflect.DeepEqual([]int{
			outcomes[index].Horizons[0].HorizonSeconds,
			outcomes[index].Horizons[1].HorizonSeconds,
			outcomes[index].Horizons[2].HorizonSeconds,
		}, []int{2, 5, 10}) {
			t.Fatalf("decision %s lacks 2/5/10s outcomes: %+v", decision.DecisionID, outcomes[index].Horizons)
		}
	}
	if _, exists := decisionTypes[block7EngagementDecision]; !exists {
		t.Fatal("peek/hold/reposition decision family was not produced")
	}
	if _, exists := decisionTypes[block7TradeDecision]; !exists {
		t.Fatal("spacing/trade-connection decision family was not produced")
	}
}

func TestCanonicalCausalProjectionRejectsPostT0Observation(t *testing.T) {
	matchDir := t.TempDir()
	if err := ExportCanonicalBundle(
		canonicalTestContext(), "block7-post-t0", matchDir, "", 42.5, 64,
		map[string]interface{}{"status": "pass", "usable_for_training": true},
	); err != nil {
		t.Fatal(err)
	}
	engagements := readCanonicalJSON[models.CanonicalEngagementsExport](t, filepath.Join(matchDir, "canonical", "derived", "engagements.json"))
	if len(engagements.Engagements) == 0 || len(engagements.Engagements[0].CausalContext.ParticipantStates) == 0 {
		t.Fatal("fixture does not contain a causal state")
	}
	engagement := engagements.Engagements[0]
	postT0 := engagement.CausalContext.T0Tick + 1
	engagement.CausalContext.ParticipantStates[0].AvailabilityTick = &postT0
	if _, err := buildCanonicalCausalPartitions("block7-post-t0", []models.CanonicalEngagement{engagement}); err == nil {
		t.Fatal("post-t0 observation must fail the causal gate")
	}
}

func TestCanonicalCausalPartitionsAbstainWithoutT0Actor(t *testing.T) {
	partitions, err := buildCanonicalCausalPartitions("block7-no-actor", []models.CanonicalEngagement{{
		EngagementID: "engagement-no-actor",
		RoundNumber:  1,
		CausalContext: models.CanonicalEngagementCausalContext{
			T0Tick: 100,
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(partitions.Decisions) != 0 || len(partitions.DecisionFeatures) != 0 ||
		len(partitions.OracleContext) != 0 || len(partitions.Outcomes) != 0 || len(partitions.QualityMasks) != 0 {
		t.Fatalf("actorless engagement produced a causal decision: %+v", partitions)
	}
}

func TestCanonicalDecisionFeaturesIgnoreFutureEngagementParticipants(t *testing.T) {
	matchDir := t.TempDir()
	if err := ExportCanonicalBundle(
		canonicalTestContext(), "block7-future-participant", matchDir, "", 42.5, 64,
		map[string]interface{}{"status": "pass", "usable_for_training": true},
	); err != nil {
		t.Fatal(err)
	}
	engagements := readCanonicalJSON[models.CanonicalEngagementsExport](t, filepath.Join(matchDir, "canonical", "derived", "engagements.json"))
	if len(engagements.Engagements) == 0 {
		t.Fatal("fixture does not contain an engagement")
	}
	baseline := engagements.Engagements[0]
	mutated := baseline
	mutated.Participants = append(append([]models.CanonicalEngagementParticipant(nil), baseline.Participants...), models.CanonicalEngagementParticipant{
		PlayerID: "future-only-player", Roles: []string{},
	})
	want, _, err := causalDecisionProjection("block7-future-participant", baseline)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := causalDecisionProjection("block7-future-participant", mutated)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("future participant changed t0 features\nwant=%+v\ngot=%+v", want, got)
	}
}

func TestCanonicalDecisionFeaturesDoNotExposeOracleDistance(t *testing.T) {
	distance := 321.5
	engagement := models.CanonicalEngagement{
		EngagementID: "engagement-distance",
		RoundNumber:  1,
		CausalContext: models.CanonicalEngagementCausalContext{
			T0Tick:                    100,
			InitialDistanceWorldUnits: &distance,
			InitialDistanceStatus:     "derived",
		},
	}

	features, mask, err := causalDecisionProjection("block7-distance", engagement)
	if err != nil {
		t.Fatal(err)
	}
	if features.InitialDistanceWorldUnits != nil || features.InitialDistanceStatus != "unavailable" {
		t.Fatalf("oracle distance leaked into observed features: %+v", features)
	}
	distanceUnavailable := false
	for _, field := range mask.UnavailableFields {
		if field == "initial_distance_world_units" {
			distanceUnavailable = true
			break
		}
	}
	if !distanceUnavailable {
		t.Fatalf("distance absence was not explicit in the quality mask: %+v", mask)
	}
}

func TestTradeDecisionFeaturesKeepExactActorTargetDistanceUnavailable(t *testing.T) {
	distance := 512.0
	connectionTime := 2048.0
	eligible := true
	alive := true
	result := &canonicalCausalPartitions{}
	appendTradeCausalPartitions("trade-distance", models.CanonicalTradesExport{
		Candidates: []models.CanonicalTradeCandidate{{
			TradeCandidateID: "candidate-1", RoundNumber: 1, DeathTick: 100,
			OriginalKillEventID: "kill-1", Evaluation: "completed",
			WindowMS: 5000, WindowEndTick: 420,
			Connections: []models.CanonicalTradeConnection{{
				TeammatePlayerID: "teammate", StateStatus: "observed", Alive: &alive,
				DistanceWorldUnits: &distance, DistanceStatus: "derived",
				ConnectionTimeMS: &connectionTime, ConnectionTimeStatus: "derived",
				Eligible: &eligible, EligibilityStatus: "derived",
			}},
		}},
	}, result)
	if len(result.DecisionFeatures) != 1 {
		t.Fatalf("decision feature count = %d, want 1", len(result.DecisionFeatures))
	}
	features := result.DecisionFeatures[0]
	if features.InitialDistanceWorldUnits != nil || features.InitialDistanceStatus != "unavailable" {
		t.Fatalf("exact actor-target distance leaked: %+v", features)
	}
	if features.NearestTeammateDistance == nil || *features.NearestTeammateDistance != distance {
		t.Fatalf("physical teammate distance was lost: %+v", features)
	}
}

func TestCanonicalDecisionBytesAreInvariantToRawPostT0Mutation(t *testing.T) {
	matchID := "block7-raw-future"
	start, end := 0, 500
	rounds := []models.CanonicalRound{{
		RoundID: matchID + ":round:001", RoundNumber: 1, StartTick: &start, EndTick: &end,
	}}
	participants := []models.CanonicalParticipant{
		{PlayerID: "steam:1", TeamID: "ct-team"},
		{PlayerID: "steam:2", TeamID: "t-team"},
	}
	velocity := 0.0
	states := map[int][]models.CanonicalPlayerState{1: {
		{SchemaID: "stratai.player_state@3", MatchID: matchID, StateID: matchID + ":state:001:000000090:steam:1", RoundID: matchID + ":round:001", RoundNumber: 1, Tick: 90, PlayerID: "steam:1", Side: "ct", IsAlive: true, HorizontalVelocityWorldUPS: &velocity, VelocitySource: "position_delta"},
		{SchemaID: "stratai.player_state@3", MatchID: matchID, StateID: matchID + ":state:001:000000090:steam:2", RoundID: matchID + ":round:001", RoundNumber: 1, Tick: 90, PlayerID: "steam:2", Side: "t", IsAlive: true, HorizontalVelocityWorldUPS: &velocity, VelocitySource: "position_delta"},
	}}
	baselineEvents := []models.CanonicalCombatEvent{
		block7HurtEvent(matchID, "hurt-12", 100, 1, "steam:1", "ct", "steam:2", "t"),
		block7HurtEvent(matchID, "hurt-21", 110, 1, "steam:2", "t", "steam:1", "ct"),
	}
	mutatedEvents := append([]models.CanonicalCombatEvent(nil), baselineEvents...)
	mutatedEvents[1].Tick = 120

	derive := func(events []models.CanonicalCombatEvent) canonicalCausalPartitions {
		engagements, _, err := engagementpkg.Derive(matchID, 64, rounds, participants, events, states)
		if err != nil {
			t.Fatal(err)
		}
		partitions, err := buildCanonicalCausalPartitions(matchID, engagements.Engagements)
		if err != nil {
			t.Fatal(err)
		}
		return partitions
	}
	want := derive(baselineEvents)
	got := derive(mutatedEvents)
	if len(want.Decisions) != 1 || len(got.Decisions) != 1 {
		t.Fatalf("expected one causal decision, got %d/%d", len(want.Decisions), len(got.Decisions))
	}
	wantDecision, _ := json.Marshal(want.Decisions)
	gotDecision, _ := json.Marshal(got.Decisions)
	wantFeatures, _ := json.Marshal(want.DecisionFeatures)
	gotFeatures, _ := json.Marshal(got.DecisionFeatures)
	if !reflect.DeepEqual(wantDecision, gotDecision) || !reflect.DeepEqual(wantFeatures, gotFeatures) {
		t.Fatalf("post-t0 raw mutation changed decision/features\nwant=%s\ngot=%s", wantFeatures, gotFeatures)
	}
	if reflect.DeepEqual(want.Outcomes, got.Outcomes) {
		t.Fatal("post-t0 raw mutation did not remain isolated in the outcome partition")
	}
}

func block7HurtEvent(
	matchID, eventID string,
	tick, sequence int,
	actor, actorSide, target, targetSide string,
) models.CanonicalCombatEvent {
	damage := 20
	return models.CanonicalCombatEvent{
		SchemaID: "stratai.combat_event@2", MatchID: matchID, EventID: eventID,
		RoundID: matchID + ":round:001", RoundNumber: 1, Tick: tick,
		SequenceInTick: sequence, SequenceInRound: sequence, EventType: "player_hurt",
		Relation: "enemy", ActorPlayerID: &actor, ActorSide: &actorSide,
		TargetPlayerID: &target, TargetSide: &targetSide, HealthDamage: &damage,
	}
}

func TestExportMatchBundleFinalizesTwentyQualityDomains(t *testing.T) {
	matchID := "block7-quality"
	matchDir := t.TempDir()
	if err := ExportCanonicalBundle(
		canonicalTestContext(), matchID, matchDir, "2026-08-21T00:00:00Z", 42.5, 64,
		map[string]interface{}{"status": "pass", "usable_for_training": true},
	); err != nil {
		t.Fatal(err)
	}
	if err := finalizeCanonicalBlock7Quality(matchDir); err != nil {
		t.Fatal(err)
	}
	quality := readCanonicalJSON[models.CanonicalQualityReportExport](t, filepath.Join(matchDir, "canonical", "diagnostics", "quality_report.json"))
	report, err := canonicalQualityReport(quality.Report)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(report.Domains))
	for _, domain := range report.Domains {
		got = append(got, domain.Name)
		if domain.Status == "fail" {
			t.Fatalf("quality domain %s failed: %v", domain.Name, domain.HardFailureDetails)
		}
	}
	if !reflect.DeepEqual(got, block7RequiredDomains) {
		t.Fatalf("quality domains = %v, want %v", got, block7RequiredDomains)
	}
	manifest := readCanonicalJSON[models.CanonicalManifest](t, filepath.Join(matchDir, "canonical", "manifest.json"))
	if len(manifest.Lineage.SchemaVersions) == 0 || manifest.Lineage.ValidatorVersion != block7ValidatorVersion {
		t.Fatalf("lineage was not finalized: %+v", manifest.Lineage)
	}
	if manifest.Lineage.GoldenCorpusVersion != "stratai.golden_demo_corpus@2" ||
		manifest.Lineage.GoldenCorpusManifestID != "golden-demos-v2" ||
		manifest.Lineage.AlgorithmVersions["trade"] != "trade_response@2" {
		t.Fatalf("release lineage is stale: %+v", manifest.Lineage)
	}
}

func TestBlock7CanonicalTreeIsDeterministicAcrossGOMAXPROCS(t *testing.T) {
	first := exportBlock7DeterminismFixture(t, t.TempDir(), 1)
	second := exportBlock7DeterminismFixture(t, t.TempDir(), 4)
	if !reflect.DeepEqual(first, second) {
		paths := make([]string, 0, len(first)+len(second))
		for path := range first {
			paths = append(paths, path)
		}
		for path := range second {
			if _, exists := first[path]; !exists {
				paths = append(paths, path)
			}
		}
		sort.Strings(paths)
		for _, path := range paths {
			if first[path] != second[path] {
				t.Errorf("determinism mismatch %s: %s != %s", path, first[path], second[path])
			}
		}
	}
}

func TestRealDemoCanonicalTreeIsDeterministicAcrossGOMAXPROCS(t *testing.T) {
	demoPath := strings.TrimSpace(os.Getenv("STRATAI_REAL_DEMO_PATH"))
	if demoPath == "" {
		t.Skip("set STRATAI_REAL_DEMO_PATH to run the real-demo determinism gate")
	}
	demoChecksum, err := checksumFile(demoPath)
	if err != nil {
		t.Fatal(err)
	}
	if expected := strings.TrimSpace(os.Getenv("STRATAI_REAL_DEMO_SHA256")); expected != "" && expected != demoChecksum {
		t.Fatalf("real-demo checksum = %s, want %s", demoChecksum, expected)
	}

	export := func(gomaxprocs int) map[string]string {
		t.Helper()
		previous := runtime.GOMAXPROCS(gomaxprocs)
		defer runtime.GOMAXPROCS(previous)
		result, err := ParseDemoWithReplay(demoPath)
		if err != nil {
			t.Fatalf("parse real demo with GOMAXPROCS=%d: %v", gomaxprocs, err)
		}
		objectiveStats := assessObjectiveQuality(result.Context)
		if objectiveStats.roundMismatches > 0 {
			facts := buildObjectiveRoundFacts(result.Context.Objectives.Events(), result.Context.Objectives.Attempts())
			results, resultMismatches := objectiveResultsByRound(result.Context.MatchData.Rounds, result.Context.CurrentRound)
			t.Logf("objective reconciliation: round_mismatches=%d result_index_mismatches=%d current_round=%d", objectiveStats.roundMismatches, resultMismatches, result.Context.CurrentRound)
			for roundNumber := 1; roundNumber <= result.Context.CurrentRound; roundNumber++ {
				summary, hasSummary := result.Context.Objectives.RoundSummary(roundNumber)
				resultRound, hasResult := results[roundNumber]
				t.Logf("objective round %d: facts=%+v summary=%+v has_summary=%t result=%+v has_result=%t", roundNumber, facts[roundNumber], summary, hasSummary, resultRound, hasResult)
			}
		}
		outputRoot := t.TempDir()
		const matchID = "golden-real-determinism"
		if err := ExportMatchBundleWithProvenance(
			result.Context,
			matchID,
			outputRoot,
			"2026-08-24T00:00:00Z",
			models.CanonicalExportProvenance{
				Source:          "demo",
				DemoChecksum:    demoChecksum,
				BuildIdentifier: "gate1-real-demo-determinism",
			},
		); err != nil {
			t.Fatalf("export real demo with GOMAXPROCS=%d: %v", gomaxprocs, err)
		}
		return hashCanonicalDirectory(t, filepath.Join(outputRoot, "match_"+matchID, "canonical"))
	}

	first := export(1)
	second := export(4)
	if !reflect.DeepEqual(first, second) {
		for path, firstHash := range first {
			if second[path] != firstHash {
				t.Errorf("real-demo determinism mismatch %s: %s != %s", path, firstHash, second[path])
			}
		}
		for path, secondHash := range second {
			if _, exists := first[path]; !exists {
				t.Errorf("real-demo determinism produced extra artifact %s=%s", path, secondHash)
			}
		}
	}
}

func hashCanonicalDirectory(t *testing.T, canonicalDir string) map[string]string {
	t.Helper()
	hashes := make(map[string]string)
	if err := filepath.Walk(canonicalDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(canonicalDir, path)
		if err != nil {
			return err
		}
		hashes[filepath.ToSlash(relative)] = fmt.Sprintf("%x", sha256.Sum256(data))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return hashes
}
