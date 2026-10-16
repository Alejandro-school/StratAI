package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

type observedVisualContactFullDemoGeometry struct{}

func (observedVisualContactFullDemoGeometry) IsLoaded() bool { return true }
func (observedVisualContactFullDemoGeometry) IsVisible(r3.Vector, r3.Vector) bool {
	return true
}

func TestBuildObservedVisualContactFullDemoReportAggregatesEveryRound(t *testing.T) {
	replay := observedVisualContactFullDemoFixture()
	var audit bytes.Buffer
	report, err := BuildObservedVisualContactFullDemoReport(
		&replay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		&audit,
	)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactFullDemoReport() error = %v", err)
	}
	report = finalizedObservedVisualContactFullDemoReportFixture(report)
	if err := ValidateObservedVisualContactFullDemoReport(report); err != nil {
		t.Fatalf("ValidateObservedVisualContactFullDemoReport() error = %v", err)
	}

	if report.Selection.RoundCount != 2 || report.Selection.FirstRound != 1 || report.Selection.LastRound != 2 ||
		report.Selection.InputSourceFrames != 9 || report.Selection.CanonicalSourceFrames != 9 {
		t.Fatalf("unexpected complete-demo selection: %#v", report.Selection)
	}
	if len(report.Rounds) != 2 || report.Rounds[0].RoundNumber != 1 || report.Rounds[1].RoundNumber != 2 {
		t.Fatalf("round summaries are incomplete or unordered: %#v", report.Rounds)
	}

	first := report.Rounds[0]
	if first.Metrics.Evaluations != 24 || first.Metrics.SmokePairSampleIntersections != 8 ||
		first.Metrics.OtherwiseVisiblePairSamplesBlockedBySmoke != 6 ||
		first.Metrics.ActiveSmokePairSampleEvaluations != 8 {
		t.Fatalf("round-one smoke pair-sample semantics are wrong: %#v", first.Metrics)
	}
	if first.Metrics.UniqueValidActiveSmokeEffects != 1 ||
		first.Metrics.SourceFramesWithValidActiveSmoke != 2 ||
		first.Metrics.EvaluatedSourceFramesWithActiveSmoke != 2 ||
		first.Metrics.EvaluatedSourceFramesWithSmokeIntersection != 2 {
		t.Fatalf("one repeated smoke was not separated from sampled intersections: %#v", first.Metrics)
	}
	if first.Metrics.DirectionalEpisodes != 6 || first.Metrics.ClosedDirectionalEpisodes != 3 ||
		first.Metrics.RightCensoredDirectionalEpisodes != 3 ||
		first.UniqueDirectedPairsWithContact != 3 || first.UniqueUnorderedPairsWithContact != 2 ||
		first.ReacquisitionEpisodesWithinRound != 3 {
		t.Fatalf("episodes were confused with unique enemy pairs: %#v", first)
	}

	summary := report.Summary
	if summary.Metrics.Evaluations != 36 || summary.Metrics.DirectionalEpisodes != 9 ||
		summary.Metrics.ClosedDirectionalEpisodes != 3 || summary.Metrics.RightCensoredDirectionalEpisodes != 6 {
		t.Fatalf("global evaluation/episode counts are wrong: %#v", summary.Metrics)
	}
	if summary.ParticipantAliases != 3 || summary.ParticipantsWithContact != 3 ||
		summary.UniqueMatchDirectedPairsWithContact != 3 || summary.UniqueMatchUnorderedPairsWithContact != 2 ||
		summary.UniqueRoundDirectedPairsWithContact != 6 || summary.ReacquisitionEpisodesWithinRound != 3 {
		t.Fatalf("global participant/pair counts are wrong: %#v", summary)
	}
	if summary.RoundsWithEvaluations != 2 || summary.RoundsWithVisualContact != 2 ||
		summary.RoundsWithValidActiveSmoke != 1 || summary.RoundsWithSmokeIntersection != 1 ||
		summary.RoundsWithOtherwiseVisibleSmokeBlock != 1 {
		t.Fatalf("round coverage counts are wrong: %#v", summary)
	}

	if report.Audit.EvaluationRecords != 36 || report.Audit.EpisodeRecords != 9 ||
		report.Audit.LogicalRecords != 45 || report.Audit.LogicalBytes != int64(audit.Len()) {
		t.Fatalf("audit record counts are wrong: %#v", report.Audit)
	}
	digest := sha256.Sum256(audit.Bytes())
	if report.Audit.LogicalSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("logical audit digest = %s, want %x", report.Audit.LogicalSHA256, digest)
	}
	assertObservedVisualContactFullDemoAudit(t, report, audit.Bytes())
}

func TestObservedVisualContactFullDemoIsDeterministicAndPrivate(t *testing.T) {
	leftReplay := observedVisualContactFullDemoFixture()
	rightReplay := observedVisualContactFullDemoFixture()
	reverseObservedVisualContactFullDemoFixture(&rightReplay)
	var leftAudit, rightAudit bytes.Buffer
	left, err := BuildObservedVisualContactFullDemoReport(
		&leftReplay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		&leftAudit,
	)
	if err != nil {
		t.Fatalf("left BuildObservedVisualContactFullDemoReport() error = %v", err)
	}
	assets := observedVisualContactSmokeAssets()
	assets[0], assets[1] = assets[1], assets[0]
	right, err := BuildObservedVisualContactFullDemoReport(
		&rightReplay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		assets,
		&rightAudit,
	)
	if err != nil {
		t.Fatalf("right BuildObservedVisualContactFullDemoReport() error = %v", err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("full-demo reports differ after source reordering\nleft=%#v\nright=%#v", left, right)
	}
	if !bytes.Equal(leftAudit.Bytes(), rightAudit.Bytes()) {
		t.Fatal("logical JSONL audit differs after source reordering")
	}

	reportJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatal(err)
	}
	combined := string(reportJSON) + leftAudit.String()
	for _, forbidden := range []string{
		"76561198000000101",
		"76561198000000102",
		"76561198000000103",
		"Raw Player",
		"steam:",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("full-demo evidence leaked %q", forbidden)
		}
	}
	if strings.Contains(string(reportJSON), `"ledger"`) || strings.Contains(string(reportJSON), `"join_keys"`) {
		t.Fatalf("compact report retained a detailed ledger: %s", reportJSON)
	}
}

func TestObservedVisualContactFullDemoReportDoesNotRetainDetailedRows(t *testing.T) {
	ticks := make([]int, 512)
	for index := range ticks {
		ticks[index] = 100 + index*4
	}
	replay := observedVisualContactFullDemoFixture()
	replay.Rounds = []models.ReplayRound{
		observedVisualContactFullDemoRound(1, 100, ticks, false),
	}
	report, err := BuildObservedVisualContactFullDemoReport(
		&replay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		io.Discard,
	)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactFullDemoReport() error = %v", err)
	}
	if report.Audit.EvaluationRecords != 512*4 {
		t.Fatalf("evaluation records = %d, want %d", report.Audit.EvaluationRecords, 512*4)
	}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > 20_000 {
		t.Fatalf("compact report grew to %d bytes for %d detailed rows", len(payload), report.Audit.LogicalRecords)
	}
	for _, forbidden := range []string{`"ledger"`, `"join_keys"`, `"contacts"`} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("compact report retained detailed field %s", forbidden)
		}
	}
}

func TestValidateObservedVisualContactFullDemoReportRejectsTampering(t *testing.T) {
	replay := observedVisualContactFullDemoFixture()
	report, err := BuildObservedVisualContactFullDemoReport(
		&replay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	report = finalizedObservedVisualContactFullDemoReportFixture(report)

	tampered := report
	tampered.Summary.Metrics.Evaluations++
	if err := ValidateObservedVisualContactFullDemoReport(tampered); err == nil ||
		!strings.Contains(err.Error(), "partitions") {
		t.Fatalf("tampered global count was accepted: %v", err)
	}

	tampered = report
	tampered.Audit.GzipRelativePath = "audit.jsonl.gz"
	tampered.Audit.GzipBytes = 0
	tampered.Audit.GzipSHA256 = ""
	if err := ValidateObservedVisualContactFullDemoReport(tampered); err == nil ||
		!strings.Contains(err.Error(), "wholly pending or complete") {
		t.Fatalf("partial gzip metadata was accepted: %v", err)
	}

	tampered = report
	tampered.Audit.GzipRelativePath = "../audit.jsonl.gz"
	tampered.Audit.GzipBytes = 10
	tampered.Audit.GzipSHA256 = strings.Repeat("d", 64)
	if err := ValidateObservedVisualContactFullDemoReport(tampered); err == nil ||
		!strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unsafe gzip path was accepted: %v", err)
	}

	complete := report
	complete.Audit.GzipRelativePath = ObservedVisualContactFullDemoAuditFilename
	complete.Audit.GzipBytes = 123
	complete.Audit.GzipSHA256 = strings.Repeat("d", 64)
	if err := ValidateObservedVisualContactFullDemoReport(complete); err != nil {
		t.Fatalf("complete safe gzip metadata was rejected: %v", err)
	}
}

func TestBuildObservedVisualContactFullDemoReportFailsClosed(t *testing.T) {
	replay := observedVisualContactFullDemoFixture()
	_, err := BuildObservedVisualContactFullDemoReport(
		&replay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "audit writer is required") {
		t.Fatalf("nil audit writer error = %v", err)
	}

	duplicate := observedVisualContactFullDemoFixture()
	duplicate.Rounds[1].Round = duplicate.Rounds[0].Round
	_, err = BuildObservedVisualContactFullDemoReport(
		&duplicate,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		io.Discard,
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate round") {
		t.Fatalf("duplicate round error = %v", err)
	}

	_, err = BuildObservedVisualContactFullDemoReport(
		&replay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		observedVisualContactFullDemoFailWriter{},
	)
	if err == nil || !strings.Contains(err.Error(), "write evaluation audit record") {
		t.Fatalf("audit writer failure = %v", err)
	}

	_, err = BuildObservedVisualContactFullDemoReport(
		&replay,
		observedVisualContactFullDemoGeometry{},
		observedVisualContactFullDemoAlias(),
		observedVisualContactFullDemoSHA256(),
		observedVisualContactSmokeAssets(),
		observedVisualContactFullDemoShortWriter{},
	)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short audit write error = %v, want io.ErrShortWrite", err)
	}
}

type observedVisualContactFullDemoFailWriter struct{}

func (observedVisualContactFullDemoFailWriter) Write([]byte) (int, error) {
	return 0, errors.New("fixture writer failed")
}

type observedVisualContactFullDemoShortWriter struct{}

func (observedVisualContactFullDemoShortWriter) Write(payload []byte) (int, error) {
	return len(payload) - 1, nil
}

func finalizedObservedVisualContactFullDemoReportFixture(
	report ObservedVisualContactFullDemoReport,
) ObservedVisualContactFullDemoReport {
	report.BuildIdentifier = "full-demo-test-build"
	report.Audit.GzipRelativePath = ObservedVisualContactFullDemoAuditFilename
	report.Audit.GzipBytes = 123
	report.Audit.GzipSHA256 = strings.Repeat("d", 64)
	return report
}

func assertObservedVisualContactFullDemoAudit(
	t *testing.T,
	report ObservedVisualContactFullDemoReport,
	payload []byte,
) {
	t.Helper()
	lines := bytes.Split(bytes.TrimSuffix(payload, []byte{'\n'}), []byte{'\n'})
	if len(lines) != report.Audit.LogicalRecords {
		t.Fatalf("JSONL lines = %d, want %d", len(lines), report.Audit.LogicalRecords)
	}
	roundHashes := make(map[int][]byte)
	previousRound := 0
	episodesStarted := make(map[int]bool)
	for index, line := range lines {
		var envelope struct {
			RecordType string          `json:"record_type"`
			Round      int             `json:"round"`
			Value      json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			t.Fatalf("audit line %d: %v", index, err)
		}
		if envelope.Round < previousRound {
			t.Fatalf("audit round order regressed at line %d", index)
		}
		previousRound = envelope.Round
		if envelope.RecordType == ObservedVisualContactFullDemoEpisodeRecord {
			episodesStarted[envelope.Round] = true
		} else if envelope.RecordType != ObservedVisualContactFullDemoEvaluationRecord {
			t.Fatalf("audit line %d has unknown record type %q", index, envelope.RecordType)
		} else if episodesStarted[envelope.Round] {
			t.Fatalf("round %d emitted an evaluation after its episodes", envelope.Round)
		}
		withNewline := append(append([]byte(nil), line...), '\n')
		roundHashes[envelope.Round] = append(roundHashes[envelope.Round], withNewline...)
		if bytes.Contains(envelope.Value, []byte("steam:")) || bytes.Contains(envelope.Value, []byte("Raw Player")) {
			t.Fatalf("audit line %d retained raw identity", index)
		}
	}
	for _, round := range report.Rounds {
		digest := sha256.Sum256(roundHashes[round.RoundNumber])
		if round.AuditLogicalSHA256 != hex.EncodeToString(digest[:]) {
			t.Fatalf("round %d logical SHA mismatch", round.RoundNumber)
		}
	}
}

func observedVisualContactFullDemoFixture() models.ReplayData {
	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			SchemaVersion: 5,
			MapName:       "de_full_demo_fixture",
			TickRate:      64,
			SampleRate:    62,
		},
		Rounds: []models.ReplayRound{
			observedVisualContactFullDemoRound(1, 100, []int{100, 104, 108, 112, 116, 120}, true),
			observedVisualContactFullDemoRound(2, 200, []int{200, 204, 208}, false),
		},
	}
}

func observedVisualContactFullDemoRound(
	roundNumber int,
	startTick int,
	ticks []int,
	withSmoke bool,
) models.ReplayRound {
	frames := make([]models.ReplayFrame, 0, len(ticks))
	for _, tick := range ticks {
		effects := []models.ReplayActiveEffect{}
		if withSmoke && (tick == startTick+8 || tick == startTick+12) {
			effects = []models.ReplayActiveEffect{{
				Type: "smoke", X: 50, Y: 50, Radius: 80, StartTick: startTick + 8,
			}}
		}
		frames = append(frames, models.ReplayFrame{
			Tick: tick,
			Players: []models.ReplayPlayerState{
				observedVisualContactFullDemoPlayer(76561198000000101, "CT", 0, 0, 0),
				observedVisualContactFullDemoPlayer(76561198000000102, "T", 100, 0, 180),
				observedVisualContactFullDemoPlayer(76561198000000103, "T", 100, 200, -116.565),
			},
			ActiveEffects: effects,
		})
	}
	return models.ReplayRound{
		Round: roundNumber, StartTick: startTick, EndTick: ticks[len(ticks)-1], Frames: frames,
	}
}

func observedVisualContactFullDemoPlayer(
	steamID uint64,
	team string,
	x, y float64,
	yaw float32,
) models.ReplayPlayerState {
	return models.ReplayPlayerState{
		SteamID:              steamID,
		Name:                 "Raw Player " + team,
		Team:                 team,
		Alive:                true,
		Weapon:               "AK-47",
		Yaw:                  yaw,
		Pitch:                0,
		EyeX:                 x,
		EyeY:                 y,
		EyeZ:                 64,
		EyePositionAvailable: true,
		EyePositionSource:    "native_eye_position",
		ViewAnglesAvailable:  true,
		ZoomLevelAvailable:   true,
	}
}

func observedVisualContactFullDemoAlias() string {
	return "demo-" + observedVisualContactFullDemoSHA256()[:20]
}

func observedVisualContactFullDemoSHA256() string {
	return strings.Repeat("e", 64)
}

func reverseObservedVisualContactFullDemoFixture(replay *models.ReplayData) {
	for left, right := 0, len(replay.Rounds)-1; left < right; left, right = left+1, right-1 {
		replay.Rounds[left], replay.Rounds[right] = replay.Rounds[right], replay.Rounds[left]
	}
	for roundIndex := range replay.Rounds {
		frames := replay.Rounds[roundIndex].Frames
		for left, right := 0, len(frames)-1; left < right; left, right = left+1, right-1 {
			frames[left], frames[right] = frames[right], frames[left]
		}
		for frameIndex := range frames {
			players := frames[frameIndex].Players
			for left, right := 0, len(players)-1; left < right; left, right = left+1, right-1 {
				players[left], players[right] = players[right], players[left]
			}
			effects := frames[frameIndex].ActiveEffects
			for left, right := 0, len(effects)-1; left < right; left, right = left+1, right-1 {
				effects[left], effects[right] = effects[right], effects[left]
			}
		}
	}
}
