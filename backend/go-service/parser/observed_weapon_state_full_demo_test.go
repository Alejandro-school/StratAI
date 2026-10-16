package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
)

func TestBuildObservedWeaponStateFullDemoReportStreamsEverySample(t *testing.T) {
	replay, events := observedWeaponFullDemoFixture()
	var rows bytes.Buffer
	report, err := BuildObservedWeaponStateFullDemoReport(
		replay,
		events,
		"demo-opaque",
		strings.Repeat("a", 64),
		&rows,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.RowsArtifact.RecordCount != 4 || report.RowsArtifact.LogicalBytes != int64(rows.Len()) {
		t.Fatalf("rows artifact = %+v, bytes=%d", report.RowsArtifact, rows.Len())
	}
	digest := sha256.Sum256(rows.Bytes())
	if report.RowsArtifact.LogicalSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("logical SHA = %s, want %x", report.RowsArtifact.LogicalSHA256, digest)
	}
	if !reflect.DeepEqual(report.Coverage.RoundsEligible, []int{1}) ||
		!reflect.DeepEqual(report.Coverage.RoundNumbers, []int{1}) ||
		report.Coverage.TargetSampleCount != 2 || report.Coverage.SampleCount != 2 ||
		report.Coverage.PlayerCount != 2 {
		t.Fatalf("coverage = %+v", report.Coverage)
	}
	if got := report.Summary.FactStatus["ammo_in_magazine"]; got.Observed != 2 || got.Unavailable != 2 {
		t.Fatalf("magazine status = %+v", got)
	}
	if report.Summary.AmmoInMagazineValues.Zero != 1 || report.Summary.AmmoInMagazineValues.Positive != 1 ||
		report.Summary.AmmoReserveValues.Zero != 0 || report.Summary.AmmoReserveValues.Positive != 2 ||
		report.Summary.ReloadValues.True != 1 || report.Summary.ReloadValues.False != 1 ||
		report.Summary.NonAmmunitionRows != 2 || report.Summary.AmbiguousRows != 0 {
		t.Fatalf("summary diversity = %+v", report.Summary)
	}
	if report.Summary.Reconciliations["ammo_in_magazine"][models.ObservedWeaponReconciliationFire] != 1 ||
		report.Summary.Reconciliations["ammo_in_magazine"][models.ObservedWeaponReconciliationReload] != 1 {
		t.Fatalf("reconciliation summary = %+v", report.Summary.Reconciliations)
	}

	lines := bytes.Split(bytes.TrimSpace(rows.Bytes()), []byte("\n"))
	if len(lines) != 4 {
		t.Fatalf("logical row lines = %d, want 4", len(lines))
	}
	for index, line := range lines {
		var row models.ObservedWeaponStateRow
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatalf("decode row %d: %v", index, err)
		}
		if !observedWeaponParticipantAliasPattern.MatchString(row.JoinKeys.PlayerID) {
			t.Fatalf("row %d lacks opaque alias: %+v", index, row.JoinKeys)
		}
	}
	text := rows.String()
	for _, secret := range []string{"76561198000000042", "76561198000000007", "private-one", "private-two"} {
		if strings.Contains(text, secret) {
			t.Fatalf("full-demo rows leaked %q: %s", secret, text)
		}
	}
}

func TestObservedWeaponStateFullDemoCoverageRejectsOmittedEligibleSample(t *testing.T) {
	replay, events := observedWeaponFullDemoFixture()
	replay.Metadata.MatchID = "demo-opaque"
	ledger, err := BuildObservedWeaponStateLedger(replay, events, 104)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := expectedObservedWeaponFullDemoCoverage(replay, 104)
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Rows) != 4 {
		t.Fatalf("fixture rows = %d, want 4", len(ledger.Rows))
	}
	if _, err := validateObservedWeaponFullDemoCoverage(expected, ledger.Rows[:2]); err == nil {
		t.Fatal("omitted eligible target sample was accepted")
	}
}

func TestObservedWeaponStateFullDemoIsPrivateAndDeterministic(t *testing.T) {
	replay, events := observedWeaponFullDemoFixture()
	second := observedWeaponCloneReplay(t, replay)
	for frameIndex := range second.Rounds[0].Frames {
		players := second.Rounds[0].Frames[frameIndex].Players
		players[0], players[1] = players[1], players[0]
	}
	reversed := []combat.Event{events[1], events[0]}

	var leftRows, rightRows bytes.Buffer
	left, err := BuildObservedWeaponStateFullDemoReport(replay, events, "demo-opaque", strings.Repeat("b", 64), &leftRows)
	if err != nil {
		t.Fatal(err)
	}
	right, err := BuildObservedWeaponStateFullDemoReport(second, reversed, "demo-opaque", strings.Repeat("b", 64), &rightRows)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftRows.Bytes(), rightRows.Bytes()) || !reflect.DeepEqual(left, right) {
		t.Fatalf("input order changed full-demo result:\nleft=%+v\nright=%+v", left, right)
	}
}

func TestValidateObservedWeaponStateFullDemoReportRejectsTampering(t *testing.T) {
	replay, events := observedWeaponFullDemoFixture()
	var rows bytes.Buffer
	report, err := BuildObservedWeaponStateFullDemoReport(replay, events, "demo-opaque", strings.Repeat("c", 64), &rows)
	if err != nil {
		t.Fatal(err)
	}
	report.BuildIdentifier = "test-build"
	report.RowsArtifact.Bytes = 42
	report.RowsArtifact.SHA256 = strings.Repeat("d", 64)
	if err := ValidateObservedWeaponStateFullDemoReport(report); err != nil {
		t.Fatal(err)
	}

	report.Summary.FactStatus["ammo_reserve"] = ObservedWeaponStateFactStatusCounts{}
	if err := ValidateObservedWeaponStateFullDemoReport(report); err == nil {
		t.Fatal("tampered summary was accepted")
	}
}

func observedWeaponFullDemoFixture() (*models.ReplayData, []combat.Event) {
	firstAK := observedWeaponTestObservation(100, "AK-47", 0, 30, false)
	secondAK := observedWeaponTestObservation(104, "AK-47", 5, 25, true)
	firstKnife := observedWeaponTestObservation(100, "Knife", 0, 0, false)
	secondC4 := observedWeaponTestObservation(104, "C4", 0, 0, false)
	replay := &models.ReplayData{
		Metadata: models.ReplayMetadata{SchemaVersion: 5, MatchID: "raw-match", TickRate: 64},
		Rounds: []models.ReplayRound{{
			Round: 1, StartTick: 100, EndTick: 104,
			Frames: []models.ReplayFrame{
				{Tick: 100, Players: []models.ReplayPlayerState{
					{SteamID: 76561198000000042, Name: "private-one", ObservedWeaponState: firstAK},
					{SteamID: 76561198000000007, Name: "private-two", ObservedWeaponState: firstKnife},
				}},
				{Tick: 104, Players: []models.ReplayPlayerState{
					{SteamID: 76561198000000042, Name: "private-one", ObservedWeaponState: secondAK},
					{SteamID: 76561198000000007, Name: "private-two", ObservedWeaponState: secondC4},
				}},
			},
		}},
	}
	events := []combat.Event{
		observedWeaponCombatEvent(combat.EventWeaponFire, 76561198000000042, "AK-47", 100, 0, 30),
		observedWeaponCombatEvent(combat.EventWeaponReload, 76561198000000042, "AK-47", 104, 5, 25),
	}
	return replay, events
}
