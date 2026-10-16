package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

func TestObservedLastKnownPositionFullDemoStreamsEveryRoundAndRecomputesSummary(t *testing.T) {
	replay := lastKnownFullDemoReplay()
	var rows bytes.Buffer
	report, err := BuildObservedLastKnownPositionFullDemoReport(
		&replay,
		lastKnownGeometry(),
		"demo-0123456789abcdefabcd",
		strings.Repeat("a", 64),
		&rows,
	)
	if err != nil {
		t.Fatalf("BuildObservedLastKnownPositionFullDemoReport() error = %v", err)
	}
	if report.Coverage.RoundNumbers[0] != 1 || report.Coverage.RoundNumbers[1] != 2 ||
		report.Coverage.RoundCount != 2 || report.Coverage.RowCount != 8 ||
		report.Coverage.ContactUpdates != 4 || report.Coverage.Carries != 4 ||
		report.Coverage.Unavailable != 0 || report.Coverage.RoundResets != 2 {
		t.Fatalf("unexpected coverage: %+v", report.Coverage)
	}
	if report.Summary.AgeDistribution.ZeroCount != 4 ||
		report.Summary.AgeDistribution.PositiveCount != 4 ||
		report.Summary.ContactObservations["true"] != 4 ||
		report.Summary.ContactObservations["false"] != 4 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.RowsArtifact.LogicalBytes != int64(rows.Len()) ||
		report.RowsArtifact.LogicalSHA256 != observedLastKnownSHA256(rows.Bytes()) {
		t.Fatalf("logical artifact mismatch: %+v", report.RowsArtifact)
	}
	lines := bytes.Split(bytes.TrimSpace(rows.Bytes()), []byte("\n"))
	if len(lines) != report.Coverage.RowCount {
		t.Fatalf("row lines = %d, want %d", len(lines), report.Coverage.RowCount)
	}
	for index, line := range lines {
		var row models.ObservedLastKnownPositionRow
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatalf("row %d is invalid JSON: %v", index, err)
		}
		if row.RowID != fmt.Sprintf("last-known-%09d", index+1) {
			t.Fatalf("row %d has non-global ID %q", index, row.RowID)
		}
	}
	encoded := rows.String()
	for _, forbidden := range []string{"secret-player", "steam_id", "observer_id", "subject_id", "entity_id", "serial"} {
		if strings.Contains(strings.ToLower(encoded), forbidden) {
			t.Fatalf("full-demo rows leak %q", forbidden)
		}
	}
}

func TestObservedLastKnownPositionFullDemoIsDeterministicAndReportFailsClosed(t *testing.T) {
	replay := lastKnownFullDemoReplay()
	var firstRows, secondRows bytes.Buffer
	first, err := BuildObservedLastKnownPositionFullDemoReport(
		&replay, lastKnownGeometry(), "demo-0123456789abcdefabcd", strings.Repeat("b", 64), &firstRows,
	)
	if err != nil {
		t.Fatal(err)
	}
	permuted := cloneLastKnownReplay(t, replay)
	permuted.Rounds[0], permuted.Rounds[1] = permuted.Rounds[1], permuted.Rounds[0]
	second, err := BuildObservedLastKnownPositionFullDemoReport(
		&permuted, lastKnownGeometry(), "demo-0123456789abcdefabcd", strings.Repeat("b", 64), &secondRows,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstRows.Bytes(), secondRows.Bytes()) || first.RowsArtifact.LogicalSHA256 != second.RowsArtifact.LogicalSHA256 {
		t.Fatal("full-demo candidate is not deterministic")
	}

	first.BuildIdentifier = "test-build"
	first.RowsArtifact.Bytes = int64(firstRows.Len())
	first.RowsArtifact.SHA256 = observedLastKnownSHA256(firstRows.Bytes())
	if err := ValidateObservedLastKnownPositionFullDemoReport(first); err != nil {
		t.Fatalf("valid physical report rejected: %v", err)
	}
	first.Coverage.Carries++
	if err := ValidateObservedLastKnownPositionFullDemoReport(first); err == nil {
		t.Fatal("tampered coverage was accepted")
	}

	if _, err := BuildObservedLastKnownPositionFullDemoReport(
		&replay, lastKnownGeometry(), "demo-0123456789abcdefabcd", strings.Repeat("b", 64), lastKnownFailWriter{},
	); err == nil {
		t.Fatal("writer failure was accepted")
	}
}

func lastKnownFullDemoReplay() models.ReplayData {
	first := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(104, lastKnownPlayer(1, "CT", r3.Vector{X: 1000}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 2000}, r3.Vector{X: -100, Z: 64}, 180, true), true),
	})
	second := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(200, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 20}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(204, lastKnownPlayer(1, "CT", r3.Vector{X: 3000}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 4000}, r3.Vector{X: -100, Z: 64}, 180, true), true),
	})
	second.Rounds[0].Round = 2
	first.Rounds = append(first.Rounds, second.Rounds[0])
	return first
}

type lastKnownFailWriter struct{}

func (lastKnownFailWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func observedLastKnownSHA256(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
