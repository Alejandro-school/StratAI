package parser

import (
	"bytes"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/cover"
)

func coverTestRegistryRoot() string {
	return filepath.Join("..", "..", "..", "ai_coach", "spatial", "cover")
}

func coverTestMapsRoot() string {
	return filepath.Join("..", "..", "data", "maps")
}

func coverTestResolution(t *testing.T) cover.PackageResolution {
	t.Helper()
	resolution := cover.ResolvePackage(coverTestRegistryRoot(), coverTestMapsRoot(), "de_mirage", cover.ContractVersion, "")
	if resolution.Status != cover.GeometryAvailable || resolution.Package == nil {
		t.Fatalf("cover package unavailable: %+v", resolution)
	}
	return resolution
}

func TestObservedCoverRecordUsesCurrentPostureAndAbstainsTactically(t *testing.T) {
	resolution := coverTestResolution(t)
	sample := resolution.Package.GeometrySamples[0]
	player := models.ReplayPlayerState{
		SteamID:             7,
		X:                   int(sample.Origin.X),
		Y:                   int(sample.Origin.Y),
		Z:                   sample.Origin.Z,
		Yaw:                 15,
		Alive:               true,
		IsDucking:           true,
		ViewAnglesAvailable: true,
	}
	record, err := BuildObservedCoverRecord(
		"de_mirage",
		"demo-aaaaaaaaaaaaaaaaaaaa",
		1,
		64,
		"participant-001",
		player,
		resolution,
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.Geometry.Status != "observed" || record.PostureID != "crouching" || len(record.Geometry.Directions) != 24 {
		t.Fatalf("objective geometry missing: %+v", record.Geometry)
	}
	if record.Geometry.Directions[0].ViewRelativeAzimuthDegrees != 345 {
		t.Fatalf("relative direction did not wrap: %+v", record.Geometry.Directions[0])
	}
	for _, direction := range record.Geometry.Directions {
		if direction.ProbeCount != 9 || direction.CompleteVolumeOcclusion != (direction.OccludedProbeCount == 9) {
			t.Fatalf("multi-probe aggregate is invalid: %+v", direction)
		}
	}
	if record.Geometry.FieldID != "FLD-GEOMETRIC-OCCLUSION" {
		t.Fatalf("wrong objective field id: %+v", record.Geometry)
	}
	if record.Tactical.Status != "unavailable" || !reflect.DeepEqual(record.Tactical.ReasonCodes, []string{SpatialTacticalJudgmentUnavailableReason}) {
		t.Fatalf("tactical judgment was fabricated: %+v", record.Tactical)
	}
}

func TestObservedCoverRecordFailsClosedForUnknownMapAndUnavailableView(t *testing.T) {
	unknown := cover.PackageResolution{Status: cover.GeometryUnavailable, ReasonCodes: []string{cover.ReasonMapNotRegistered}}
	record, err := BuildObservedCoverRecord(
		"de_dust2", "demo-aaaaaaaaaaaaaaaaaaaa", 1, 1, "participant-001",
		models.ReplayPlayerState{Alive: true}, unknown,
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.Geometry.Status != "unavailable" || !reflect.DeepEqual(record.Geometry.ReasonCodes, []string{cover.ReasonMapNotRegistered}) {
		t.Fatalf("unknown map did not fail closed: %+v", record)
	}

	resolution := coverTestResolution(t)
	sample := resolution.Package.GeometrySamples[0]
	record, err = BuildObservedCoverRecord(
		"de_mirage", "demo-aaaaaaaaaaaaaaaaaaaa", 1, 1, "participant-001",
		models.ReplayPlayerState{X: int(sample.Origin.X), Y: int(sample.Origin.Y), Z: sample.Origin.Z, Alive: true}, resolution,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(record.Geometry.ReasonCodes, []string{cover.ReasonViewAngles}) {
		t.Fatalf("missing view did not abstain: %+v", record.Geometry)
	}
	record, err = BuildObservedCoverRecord(
		"de_mirage", "demo-aaaaaaaaaaaaaaaaaaaa", 1, 2, "participant-001",
		models.ReplayPlayerState{X: int(sample.Origin.X), Y: int(sample.Origin.Y), Z: math.NaN(), Alive: true, ViewAnglesAvailable: true}, resolution,
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.Position != nil || !reflect.DeepEqual(record.Geometry.ReasonCodes, []string{cover.ReasonPosition}) {
		t.Fatalf("non-finite position did not become explicit unavailable: %+v", record)
	}
	if _, err := json.Marshal(record); err != nil {
		t.Fatalf("unavailable non-finite input did not remain JSON-safe: %v", err)
	}
}

func TestObservedCoverFullDemoIsDeterministicAndFutureIndependent(t *testing.T) {
	resolution := coverTestResolution(t)
	sample := resolution.Package.GeometrySamples[0]
	player := models.ReplayPlayerState{SteamID: 7, X: int(sample.Origin.X), Y: int(sample.Origin.Y), Z: sample.Origin.Z, Yaw: 30, Alive: true, ViewAnglesAvailable: true}
	replay := &models.ReplayData{Metadata: models.ReplayMetadata{MapName: "de_mirage", TickRate: 64}, Rounds: []models.ReplayRound{{Round: 1, StartTick: 0, EndTick: 128, Frames: []models.ReplayFrame{{Tick: 0, Players: []models.ReplayPlayerState{player}}, {Tick: 64, Players: []models.ReplayPlayerState{player}}}}}}
	checksum := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var first, second bytes.Buffer
	report1, err := BuildObservedCoverFullDemoReport(replay, resolution, "demo-aaaaaaaaaaaaaaaaaaaa", checksum, &first)
	if err != nil {
		t.Fatal(err)
	}
	replay.Rounds[0].Frames[1].Players[0].Health = 99
	replay.Rounds[0].Frames[1].Players[0].Weapon = "oracle-like-future-mutation"
	report2, err := BuildObservedCoverFullDemoReport(replay, resolution, "demo-aaaaaaaaaaaaaaaaaaaa", checksum, &second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) || report1.RowsArtifact.LogicalSHA256 != report2.RowsArtifact.LogicalSHA256 {
		t.Fatal("unrelated future/outcome mutation changed causal cover geometry")
	}
	if report1.Coverage.Records != 2 || report1.Coverage.GeometryObserved != 2 || report1.Coverage.TacticalUnavailable != 2 || report1.Coverage.TacticalReasonCounts[SpatialTacticalJudgmentUnavailableReason] != 2 {
		t.Fatalf("unexpected report counts: %+v", report1.Coverage)
	}
	for _, line := range bytes.Split(bytes.TrimSpace(first.Bytes()), []byte{'\n'}) {
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatal(err)
		}
		if _, found := row["steam_id"]; found {
			t.Fatal("raw identity leaked into sidecar")
		}
	}
}
