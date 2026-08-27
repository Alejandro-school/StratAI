package parser

import (
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

func TestTacticalSelfScopePreservesObservedEmptyGrenadeInventory(t *testing.T) {
	state := models.TacticalPhysicalState{
		Grenades:          []string{},
		FieldAvailability: tacticalUnavailableAvailability(),
	}
	state.FieldAvailability["grenades"] = "observed"
	projected := tacticalPhysicalStateForScope(state, models.TacticalVisibilitySelf)
	if projected.Grenades == nil || len(projected.Grenades) != 0 {
		t.Fatalf("empty observed grenade inventory must encode as []: %+v", projected.Grenades)
	}
}

type tacticalGeometryStub struct {
	loaded  bool
	calls   int
	visible func(start, end r3.Vector) bool
}

func (stub *tacticalGeometryStub) IsLoaded() bool {
	return stub.loaded
}

func (stub *tacticalGeometryStub) IsVisible(start, end r3.Vector) bool {
	stub.calls++
	if stub.visible == nil {
		return false
	}
	return stub.visible(start, end)
}

func TestBuildTacticalExportDoesNotLeakHiddenEnemies(t *testing.T) {
	replay := tacticalTestReplay(64, models.ReplayRound{
		Round:     1,
		StartTick: 100,
		EndTick:   100,
		Frames: []models.ReplayFrame{{
			Tick: 100,
			Players: []models.ReplayPlayerState{
				tacticalTestPlayer(1, "observer-secret", "CT", 100, 0),
				tacticalTestPlayer(2, "teammate-secret", "CT", 200, 0),
				tacticalTestPlayer(3, "visible-enemy-secret", "T", 300, 0),
				tacticalTestPlayer(4, "hidden-enemy-secret", "T", 400, 0),
			},
		}},
	})
	geometry := &tacticalGeometryStub{
		loaded: true,
		visible: func(start, end r3.Vector) bool {
			return start.X == 100 && end.X == 300
		},
	}

	export, err := BuildTacticalExport(&replay, geometry)
	if err != nil {
		t.Fatalf("BuildTacticalExport() error = %v", err)
	}

	observerRows := tacticalPhysicalRowsForObserver(export.PhysicalRows, "steam:1")
	if got, want := tacticalSubjectIDs(observerRows), []string{"steam:1", "steam:2", "steam:3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("observer subjects = %v, want %v", got, want)
	}
	if row := tacticalPhysicalRow(observerRows, "steam:3"); row == nil {
		t.Fatal("visible enemy row is missing")
	} else {
		if row.VisibilityScope != models.TacticalVisibilityEnemyLOS {
			t.Fatalf("visible enemy scope = %q, want %q", row.VisibilityScope, models.TacticalVisibilityEnemyLOS)
		}
		if row.Provenance.LineOfSight == nil || !*row.Provenance.LineOfSight {
			t.Fatalf("visible enemy LOS provenance = %v, want true", row.Provenance.LineOfSight)
		}
		if row.State.Health != nil || row.State.Money != nil || row.State.HasC4 != nil ||
			row.State.FieldAvailability["health"] != "unavailable" {
			t.Fatalf("visible enemy received hidden HUD/economy state: %+v", row.State)
		}
	}
	if teammate := tacticalPhysicalRow(observerRows, "steam:2"); teammate == nil {
		t.Fatal("teammate row is missing")
	} else if teammate.State.ActiveWeapon != nil || teammate.State.Money != nil ||
		teammate.State.FieldAvailability["active_weapon"] != "unavailable" {
		t.Fatalf("teammate row leaked omniscient equipment/economy: %+v", teammate.State)
	}
	if tacticalPhysicalRow(observerRows, "steam:4") != nil {
		t.Fatal("hidden enemy leaked into observer physical rows")
	}

	if got, want := tacticalOracleSubjectIDs(export.OracleRows), []string{"steam:1", "steam:2", "steam:3", "steam:4"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("oracle subjects = %v, want %v", got, want)
	}

	encoded, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("json.Marshal(export) error = %v", err)
	}
	for _, forbidden := range []string{
		"observer-secret",
		"teammate-secret",
		"visible-enemy-secret",
		"hidden-enemy-secret",
		`"name"`,
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("export contains forbidden identity/name data %q: %s", forbidden, encoded)
		}
	}

	unloaded := &tacticalGeometryStub{
		loaded: false,
		visible: func(start, end r3.Vector) bool {
			t.Fatalf("IsVisible called with unloaded geometry: %v -> %v", start, end)
			return true
		},
	}
	unloadedExport, err := BuildTacticalExport(&replay, unloaded)
	if err != nil {
		t.Fatalf("BuildTacticalExport(unloaded geometry) error = %v", err)
	}
	unloadedRows := tacticalPhysicalRowsForObserver(unloadedExport.PhysicalRows, "steam:1")
	if got, want := tacticalSubjectIDs(unloadedRows), []string{"steam:1", "steam:2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unloaded geometry subjects = %v, want self/team only %v", got, want)
	}
	if unloaded.calls != 0 {
		t.Fatalf("unloaded geometry IsVisible calls = %d, want 0", unloaded.calls)
	}
}

func TestBuildTacticalExportKeepsPhysicalAndOracleStateSeparate(t *testing.T) {
	player := tacticalTestPlayer(7, "private-name", "CT", 0, 0)
	player.Health = 73
	player.Armor = 42
	player.Money = 1600
	player.HasC4 = true
	player.Weapons = []string{"weapon_ak47", "weapon_knife"}
	player.Weapon = ""
	player.Yaw = float32(math.Inf(1))
	player.FlashDuration = math.NaN()
	replay := tacticalTestReplay(64, models.ReplayRound{
		Round:     1,
		StartTick: 20,
		EndTick:   20,
		Frames: []models.ReplayFrame{{
			Tick:    20,
			Players: []models.ReplayPlayerState{player},
		}},
	})

	export, err := BuildTacticalExport(&replay, nil)
	if err != nil {
		t.Fatalf("BuildTacticalExport() error = %v", err)
	}
	if len(export.PhysicalRows) != 1 || len(export.OracleRows) != 1 {
		t.Fatalf("row counts physical/oracle = %d/%d, want 1/1", len(export.PhysicalRows), len(export.OracleRows))
	}

	physicalJSON, err := json.Marshal(export.PhysicalRows[0].State)
	if err != nil {
		t.Fatalf("json.Marshal(physical state) error = %v", err)
	}
	for _, forbiddenField := range []string{
		`"weapons"`,
		`"has_helmet"`,
	} {
		if strings.Contains(string(physicalJSON), forbiddenField) {
			t.Fatalf("physical state contains oracle-only field %s: %s", forbiddenField, physicalJSON)
		}
	}
	if !strings.Contains(string(physicalJSON), `"active_weapon":null`) {
		t.Fatalf("missing active weapon was not encoded as null: %s", physicalJSON)
	}
	if !strings.Contains(string(physicalJSON), `"yaw":null`) {
		t.Fatalf("non-finite yaw was not encoded as null: %s", physicalJSON)
	}
	physical := export.PhysicalRows[0].State
	if physical.Health == nil || *physical.Health != 73 || physical.Armor == nil || *physical.Armor != 42 ||
		physical.Money == nil || *physical.Money != 1600 {
		t.Fatalf("self-visible health/armor/economy missing: %+v", physical)
	}
	if physical.IsReloading != nil || physical.AmmoInMagazine != nil || physical.AmmoReserve != nil ||
		physical.FieldAvailability["is_reloading"] != "unavailable" ||
		physical.FieldAvailability["ammo_in_magazine"] != "unavailable" {
		t.Fatalf("unproduced reload/ammo facts were invented: %+v", physical)
	}

	oracle := export.OracleRows[0]
	if oracle.CausalRole != models.TacticalCausalRoleOracle {
		t.Fatalf("oracle causal role = %q, want %q", oracle.CausalRole, models.TacticalCausalRoleOracle)
	}
	if oracle.State.Health == nil || *oracle.State.Health != 73 {
		t.Fatalf("oracle health = %v, want 73", oracle.State.Health)
	}
	if oracle.State.Armor == nil || *oracle.State.Armor != 42 {
		t.Fatalf("oracle armor = %v, want 42", oracle.State.Armor)
	}
	if oracle.State.Money == nil || *oracle.State.Money != 1600 {
		t.Fatalf("oracle money = %v, want 1600", oracle.State.Money)
	}
	if oracle.State.HasC4 == nil || !*oracle.State.HasC4 {
		t.Fatalf("oracle has_c4 = %v, want true", oracle.State.HasC4)
	}
	if oracle.State.FlashDurationSeconds != nil {
		t.Fatalf("non-finite flash duration = %v, want nil", oracle.State.FlashDurationSeconds)
	}
	if got, want := oracle.State.Weapons, []string{"weapon_ak47", "weapon_knife"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("oracle weapons = %v, want %v", got, want)
	}

	if export.PhysicalRows[0].IdentitySemantics != models.TacticalIdentityJoinOnly ||
		export.OracleRows[0].IdentitySemantics != models.TacticalIdentityJoinOnly {
		t.Fatal("row identity semantics must be join_only")
	}
	if export.PhysicalRows[0].JoinKeys.ObserverID == nil || export.PhysicalRows[0].JoinKeys.SubjectID == nil {
		t.Fatal("physical observer/subject IDs must be isolated in join_keys")
	}
}

func TestBuildTacticalExportDerives16HzAndEmitsSamplingGaps(t *testing.T) {
	replay := tacticalTestReplay(64, models.ReplayRound{
		Round:     1,
		StartTick: 100,
		EndTick:   112,
		Frames: []models.ReplayFrame{
			{Tick: 112, Players: []models.ReplayPlayerState{tacticalTestPlayer(1, "p1", "CT", 112, 0)}},
			{Tick: 100, Players: []models.ReplayPlayerState{tacticalTestPlayer(1, "p1", "CT", 100, 0)}},
			{Tick: 107, Players: []models.ReplayPlayerState{tacticalTestPlayer(1, "p1", "CT", 107, 0)}},
		},
	})

	export, err := BuildTacticalExport(&replay, nil)
	if err != nil {
		t.Fatalf("BuildTacticalExport() error = %v", err)
	}
	if export.Sampling.TargetHz != 16 {
		t.Fatalf("target Hz = %d, want 16", export.Sampling.TargetHz)
	}
	if export.Sampling.PeriodTicks != 4 {
		t.Fatalf("period ticks = %v, want 4", export.Sampling.PeriodTicks)
	}
	if export.Sampling.Strategy != models.TacticalSamplingStrategy {
		t.Fatalf("sampling strategy = %q, want %q", export.Sampling.Strategy, models.TacticalSamplingStrategy)
	}

	if got, want := tacticalRowTicks(export.PhysicalRows), []int{100, 108, 112}; !reflect.DeepEqual(got, want) {
		t.Fatalf("physical target ticks = %v, want %v", got, want)
	}
	if got, want := tacticalAvailabilityTicks(export.PhysicalRows), []int{100, 107, 112}; !reflect.DeepEqual(got, want) {
		t.Fatalf("physical availability ticks = %v, want %v", got, want)
	}
	for _, row := range export.PhysicalRows {
		if row.AvailabilityTick == nil || *row.AvailabilityTick > row.Tick {
			t.Fatalf("physical availability_tick = %v exceeds/does not exist at tick %d", row.AvailabilityTick, row.Tick)
		}
		if row.Status != models.TacticalStatusObserved {
			t.Fatalf("physical status = %q, want observed", row.Status)
		}
	}
	velocityRow := tacticalPhysicalRowAtTick(export.PhysicalRows, "steam:1", 108)
	if velocityRow == nil || velocityRow.State.HorizontalVelocity == nil ||
		math.Abs(*velocityRow.State.HorizontalVelocity-64) > 1e-9 ||
		velocityRow.State.FieldAvailability["horizontal_velocity_world_units_per_second"] != "derived" {
		t.Fatalf("derived 16 Hz velocity missing at tick 108: %+v", velocityRow)
	}
	for _, row := range export.OracleRows {
		if row.AvailabilityTick == nil || *row.AvailabilityTick > row.Tick {
			t.Fatalf("oracle availability_tick = %v exceeds/does not exist at tick %d", row.AvailabilityTick, row.Tick)
		}
	}

	if len(export.Gaps) != 1 {
		t.Fatalf("gaps = %d, want 1: %#v", len(export.Gaps), export.Gaps)
	}
	gap := export.Gaps[0]
	if gap.Tick != 104 || gap.Reason != tacticalGapMissingFrame {
		t.Fatalf("gap = tick %d reason %q, want tick 104 reason %q", gap.Tick, gap.Reason, tacticalGapMissingFrame)
	}
	if gap.AvailabilityTick != nil || gap.Status != models.TacticalStatusUnavailable {
		t.Fatalf("gap availability/status = %v/%q, want nil/unavailable", gap.AvailabilityTick, gap.Status)
	}
	if gap.CausalRole != models.TacticalCausalRoleGap || gap.VisibilityScope != models.TacticalVisibilityGap {
		t.Fatalf("gap causal metadata = %q/%q", gap.CausalRole, gap.VisibilityScope)
	}
	if gap.Provenance.SourceFrameTick != nil {
		t.Fatalf("missing-frame provenance tick = %v, want nil", gap.Provenance.SourceFrameTick)
	}
}

func TestBuildTacticalExportIsDeterministicAcrossInputOrderAndNames(t *testing.T) {
	p1 := tacticalTestPlayer(10, "first-name", "CT", 10, 20)
	p1.Weapons = []string{"weapon_knife", "weapon_m4a1", "weapon_knife"}
	p2 := tacticalTestPlayer(2, "second-name", "T", 30, 40)
	p2.Weapons = []string{"weapon_glock", "weapon_knife"}

	roundOne := models.ReplayRound{
		Round:     1,
		StartTick: 100,
		EndTick:   104,
		Frames: []models.ReplayFrame{
			{Tick: 104, Players: []models.ReplayPlayerState{p1, p2}},
			{Tick: 100, Players: []models.ReplayPlayerState{p2, p1}},
		},
	}
	roundTwo := models.ReplayRound{
		Round:     2,
		StartTick: 200,
		EndTick:   200,
		Frames: []models.ReplayFrame{{
			Tick:    200,
			Players: []models.ReplayPlayerState{p2, p1},
		}},
	}

	left := tacticalTestReplay(64, roundTwo, roundOne)
	p1.Name = "renamed-player-one"
	p1.Weapons = []string{"weapon_m4a1", "weapon_knife"}
	p2.Name = "renamed-player-two"
	p2.Weapons = []string{"weapon_knife", "weapon_glock", "weapon_glock"}
	rightRoundOne := models.ReplayRound{
		Round:     1,
		StartTick: 100,
		EndTick:   104,
		Frames: []models.ReplayFrame{
			{Tick: 100, Players: []models.ReplayPlayerState{p1, p2}},
			{Tick: 104, Players: []models.ReplayPlayerState{p2, p1}},
		},
	}
	rightRoundTwo := models.ReplayRound{
		Round:     2,
		StartTick: 200,
		EndTick:   200,
		Frames: []models.ReplayFrame{{
			Tick:    200,
			Players: []models.ReplayPlayerState{p1, p2},
		}},
	}
	right := tacticalTestReplay(64, rightRoundOne, rightRoundTwo)

	leftGeometry := &tacticalGeometryStub{loaded: true, visible: func(start, end r3.Vector) bool { return true }}
	rightGeometry := &tacticalGeometryStub{loaded: true, visible: func(start, end r3.Vector) bool { return true }}
	leftExport, err := BuildTacticalExport(&left, leftGeometry)
	if err != nil {
		t.Fatalf("BuildTacticalExport(left) error = %v", err)
	}
	rightExport, err := BuildTacticalExport(&right, rightGeometry)
	if err != nil {
		t.Fatalf("BuildTacticalExport(right) error = %v", err)
	}

	leftJSON, err := json.Marshal(leftExport)
	if err != nil {
		t.Fatalf("json.Marshal(left) error = %v", err)
	}
	rightJSON, err := json.Marshal(rightExport)
	if err != nil {
		t.Fatalf("json.Marshal(right) error = %v", err)
	}
	if string(leftJSON) != string(rightJSON) {
		t.Fatalf("exports differ across input order/name changes\nleft:  %s\nright: %s", leftJSON, rightJSON)
	}
}

func tacticalTestReplay(tickRate float64, rounds ...models.ReplayRound) models.ReplayData {
	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			SchemaVersion: 1,
			MatchID:       "match-tactical-test",
			MapName:       "de_test",
			TickRate:      tickRate,
		},
		Rounds: rounds,
	}
}

func tacticalTestPlayer(steamID uint64, name, team string, x, y int) models.ReplayPlayerState {
	return models.ReplayPlayerState{
		SteamID: steamID,
		Name:    name,
		Team:    team,
		X:       x,
		Y:       y,
		Z:       64,
		Yaw:     90,
		Pitch:   5,
		Health:  100,
		Armor:   50,
		Alive:   true,
		Weapon:  "weapon_knife",
		Money:   800,
	}
}

func tacticalPhysicalRowsForObserver(rows []models.TacticalPhysicalRow, observerID string) []models.TacticalPhysicalRow {
	result := make([]models.TacticalPhysicalRow, 0)
	for _, row := range rows {
		if row.JoinKeys.ObserverID != nil && *row.JoinKeys.ObserverID == observerID {
			result = append(result, row)
		}
	}
	return result
}

func tacticalSubjectIDs(rows []models.TacticalPhysicalRow) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.JoinKeys.SubjectID != nil {
			result = append(result, *row.JoinKeys.SubjectID)
		}
	}
	sort.Strings(result)
	return result
}

func tacticalOracleSubjectIDs(rows []models.TacticalOracleRow) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.JoinKeys.SubjectID != nil {
			result = append(result, *row.JoinKeys.SubjectID)
		}
	}
	sort.Strings(result)
	return result
}

func tacticalPhysicalRow(rows []models.TacticalPhysicalRow, subjectID string) *models.TacticalPhysicalRow {
	for index := range rows {
		if rows[index].JoinKeys.SubjectID != nil && *rows[index].JoinKeys.SubjectID == subjectID {
			return &rows[index]
		}
	}
	return nil
}

func tacticalPhysicalRowAtTick(rows []models.TacticalPhysicalRow, subjectID string, tick int) *models.TacticalPhysicalRow {
	for index := range rows {
		if rows[index].Tick == tick && rows[index].JoinKeys.SubjectID != nil && *rows[index].JoinKeys.SubjectID == subjectID {
			return &rows[index]
		}
	}
	return nil
}

func tacticalRowTicks(rows []models.TacticalPhysicalRow) []int {
	result := make([]int, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.Tick)
	}
	return result
}

func tacticalAvailabilityTicks(rows []models.TacticalPhysicalRow) []int {
	result := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.AvailabilityTick != nil {
			result = append(result, *row.AvailabilityTick)
		}
	}
	return result
}
