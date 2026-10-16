package parser

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
)

func TestObservedWeaponStatePublishesPositiveZeroAndReloadBooleans(t *testing.T) {
	tests := []struct {
		name    string
		mag     int
		reserve int
		reload  bool
	}{
		{name: "positive ammo and reload true", mag: 17, reserve: 61, reload: true},
		{name: "observed empty magazine and reload false", mag: 0, reserve: 45, reload: false},
		{name: "observed empty reserve", mag: 9, reserve: 0, reload: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", test.mag, test.reserve, test.reload))
			ledger, err := BuildObservedWeaponStateLedger(replay, nil, 100)
			if err != nil {
				t.Fatal(err)
			}
			row := observedWeaponOnlyRow(t, ledger)
			if row.ActiveWeapon.Value == nil || *row.ActiveWeapon.Value != "AK-47" ||
				row.AmmoInMagazine.Value == nil || *row.AmmoInMagazine.Value != test.mag ||
				row.AmmoReserve.Value == nil || *row.AmmoReserve.Value != test.reserve ||
				row.IsReloading.Value == nil || *row.IsReloading.Value != test.reload {
				t.Fatalf("unexpected observed facts: %+v", row)
			}
			for _, tick := range []*int{
				row.ActiveWeapon.AvailabilityTick,
				row.AmmoInMagazine.AvailabilityTick,
				row.AmmoReserve.AvailabilityTick,
				row.IsReloading.AvailabilityTick,
			} {
				if tick == nil || *tick != 100 {
					t.Fatalf("availability tick = %v, want 100", tick)
				}
			}
		})
	}
}

func TestObservedWeaponStateSerializesEmptyAbstentionsAsArray(t *testing.T) {
	ledger := observedWeaponBuild(
		t,
		observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 17, 61, false)),
		nil,
		100,
	)
	row := observedWeaponOnlyRow(t, ledger)
	payload, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"abstention_reasons":[]`) {
		t.Fatalf("empty abstentions must serialize as an array: %s", payload)
	}
}

func TestObservedWeaponStateAbstainsWithoutActiveWeaponOrSnapshot(t *testing.T) {
	tests := []struct {
		name        string
		observation models.ReplayWeaponStateObservation
	}{
		{name: "snapshot absent"},
		{name: "active weapon absent", observation: models.ReplayWeaponStateObservation{
			ActiveWeaponStatus: models.ObservedWeaponFactUnavailable,
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := observedWeaponOnlyRow(t, observedWeaponBuild(t, observedWeaponReplay(test.observation), nil, 100))
			assertObservedWeaponFactsUnavailable(t, row, true)
		})
	}
}

func TestObservedWeaponStateNeverTurnsKnifeC4OrUtilityIntoZeroAmmo(t *testing.T) {
	for _, weapon := range []string{"Knife", "Karambit", "Shadow Daggers", "C4", "Smoke Grenade"} {
		t.Run(weapon, func(t *testing.T) {
			observation := observedWeaponTestObservation(100, weapon, 0, 0, false)
			row := observedWeaponOnlyRow(t, observedWeaponBuild(t, observedWeaponReplay(observation), nil, 100))
			if row.ActiveWeapon.Value == nil || *row.ActiveWeapon.Value != weapon {
				t.Fatalf("active weapon missing: %+v", row.ActiveWeapon)
			}
			if row.AmmoInMagazine.Status != models.ObservedWeaponFactUnavailable ||
				row.AmmoReserve.Status != models.ObservedWeaponFactUnavailable ||
				row.IsReloading.Status != models.ObservedWeaponFactUnavailable ||
				row.AmmoInMagazine.Value != nil || row.AmmoReserve.Value != nil || row.IsReloading.Value != nil {
				t.Fatalf("non-ammunition weapon fabricated facts: %+v", row)
			}
			if row.AmmoInMagazine.Provenance.Reconciliation != models.ObservedWeaponReconciliationNotApplicable {
				t.Fatalf("non-ammunition provenance = %+v", row.AmmoInMagazine.Provenance)
			}
		})
	}
}

func TestObservedWeaponStateRejectsNegativeAndNonFiniteSentinels(t *testing.T) {
	observation := observedWeaponTestObservation(100, "AK-47", -1, -7, false)
	row := observedWeaponOnlyRow(t, observedWeaponBuild(t, observedWeaponReplay(observation), nil, 100))
	if row.AmmoInMagazine.Status != models.ObservedWeaponFactUnavailable || row.AmmoInMagazine.Value != nil ||
		row.AmmoReserve.Status != models.ObservedWeaponFactUnavailable || row.AmmoReserve.Value != nil {
		t.Fatalf("negative sentinel leaked: %+v", row)
	}

	replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 30, 90, false))
	replay.Metadata.TickRate = math.NaN()
	if _, err := BuildObservedWeaponStateLedger(replay, nil, 100); err == nil {
		t.Fatal("non-finite tick rate was accepted")
	}
}

func TestObservedWeaponStateRejectsAvailabilityAfterT0(t *testing.T) {
	observation := observedWeaponTestObservation(101, "AK-47", 30, 90, false)
	row := observedWeaponOnlyRow(t, observedWeaponBuild(t, observedWeaponReplay(observation), nil, 100))
	assertObservedWeaponFactsUnavailable(t, row, true)
	if !observedWeaponContainsReason(row.AbstentionReasons, weaponAbstentionAvailability) {
		t.Fatalf("future availability abstention missing: %v", row.AbstentionReasons)
	}
}

func TestObservedWeaponStateIsInvariantToFutureAndOracleMutation(t *testing.T) {
	replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 29, 88, false))
	replay.Rounds[0].EndTick = 104
	baseline := observedWeaponBuild(t, replay, nil, 100)

	// A future player with a lower raw ID must not renumber the aliases at T0.
	replay.Rounds[0].Frames = append(replay.Rounds[0].Frames, models.ReplayFrame{
		Tick: 104,
		Players: []models.ReplayPlayerState{{
			SteamID: 1, Name: "future-name", Health: 1,
			ObservedWeaponState: observedWeaponTestObservation(104, "M4A4", 12, 40, true),
		}},
	})
	afterFuture := observedWeaponBuild(t, replay, nil, 100)
	if !reflect.DeepEqual(baseline, afterFuture) {
		t.Fatalf("future mutation changed T0 ledger:\nbefore=%+v\nafter=%+v", baseline, afterFuture)
	}
	fullFuture := observedWeaponBuild(t, replay, nil, 104)
	if len(fullFuture.Rows) < 1 || !reflect.DeepEqual(baseline.Rows, fullFuture.Rows[:1]) {
		t.Fatalf("future participant renumbered prior aliases:\nbefore=%+v\nafter=%+v", baseline.Rows, fullFuture.Rows)
	}

	// Presentation/oracle/legacy fields are not inputs to the candidate.
	replay.Rounds[0].Frames[0].Players[0].Name = "oracle-mutated"
	replay.Rounds[0].Frames[0].Players[0].Health = 1
	replay.Rounds[0].Frames[0].Players[0].Armor = 99
	replay.Rounds[0].Frames[0].Players[0].Weapon = "Knife"
	replay.Rounds[0].Frames[0].Players[0].IsReloading = true
	afterOracle := observedWeaponBuild(t, replay, nil, 100)
	if !reflect.DeepEqual(baseline, afterOracle) {
		t.Fatalf("oracle/legacy mutation changed candidate ledger:\nbefore=%+v\nafter=%+v", baseline, afterOracle)
	}
}

func TestObservedWeaponStateOutputOwnsValuesAndKeepsOpaqueJoinAliases(t *testing.T) {
	observation := observedWeaponTestObservation(100, "AK-47", 20, 60, false)
	replay := observedWeaponReplay(observation)
	replay.Rounds[0].Frames[0].Players[0].SteamID = 76561198012345678
	replay.Rounds[0].Frames[0].Players[0].Name = "sensitive-name"
	ledger := observedWeaponBuild(t, replay, nil, 100)

	*replay.Rounds[0].Frames[0].Players[0].ObservedWeaponState.AmmoInMagazine = 1
	row := observedWeaponOnlyRow(t, ledger)
	if row.AmmoInMagazine.Value == nil || *row.AmmoInMagazine.Value != 20 {
		t.Fatalf("output aliases mutable input pointer: %+v", row.AmmoInMagazine)
	}
	if row.JoinKeys.PlayerID != "participant-001" || !strings.Contains(row.JoinKeys.ObservationID, "participant-001") {
		t.Fatalf("opaque aliases missing: %+v", row.JoinKeys)
	}
	payload, err := json.Marshal(ledger)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	if strings.Contains(text, "76561198012345678") || strings.Contains(text, "sensitive-name") {
		t.Fatalf("raw identity leaked outside local aliases: %s", text)
	}
}

func TestObservedWeaponStateIsDeterministicAcrossFrameAndEventOrder(t *testing.T) {
	first := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 19, 50, false))
	secondPlayer := models.ReplayPlayerState{
		SteamID:             7,
		Name:                "second",
		ObservedWeaponState: observedWeaponTestObservation(100, "M4A4", 11, 30, false),
	}
	first.Rounds[0].Frames[0].Players = append(first.Rounds[0].Frames[0].Players, secondPlayer)
	first.Rounds[0].EndTick = 104
	first.Rounds[0].Frames = append(first.Rounds[0].Frames, models.ReplayFrame{
		Tick: 104,
		Players: []models.ReplayPlayerState{
			{SteamID: 42, Name: "first", ObservedWeaponState: observedWeaponTestObservation(104, "AK-47", 18, 49, false)},
			{SteamID: 7, Name: "second", ObservedWeaponState: observedWeaponTestObservation(104, "M4A4", 10, 29, false)},
		},
	})
	second := observedWeaponCloneReplay(t, first)
	second.Rounds[0].Frames[0], second.Rounds[0].Frames[1] = second.Rounds[0].Frames[1], second.Rounds[0].Frames[0]
	for frameIndex := range second.Rounds[0].Frames {
		second.Rounds[0].Frames[frameIndex].Players[0], second.Rounds[0].Frames[frameIndex].Players[1] =
			second.Rounds[0].Frames[frameIndex].Players[1], second.Rounds[0].Frames[frameIndex].Players[0]
	}

	events := []combat.Event{
		observedWeaponCombatEvent(combat.EventWeaponFire, 42, "AK-47", 100, 19, 50),
		observedWeaponCombatEvent(combat.EventWeaponFire, 7, "M4A4", 100, 11, 30),
		observedWeaponCombatEvent(combat.EventWeaponFire, 42, "AK-47", 104, 18, 49),
		observedWeaponCombatEvent(combat.EventWeaponFire, 7, "M4A4", 104, 10, 29),
	}
	reversedEvents := []combat.Event{events[3], events[2], events[1], events[0]}
	left := observedWeaponBuild(t, first, events, 104)
	right := observedWeaponBuild(t, second, reversedEvents, 104)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("equivalent order changed output:\nleft=%+v\nright=%+v", left, right)
	}
}

func TestObservedWeaponStateReconcilesOnlyMatchingCausalEvents(t *testing.T) {
	t.Run("weapon fire confirms matching snapshot", func(t *testing.T) {
		replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 18, 54, false))
		event := observedWeaponCombatEvent(combat.EventWeaponFire, 42, "AK-47", 100, 18, 54)
		row := observedWeaponOnlyRow(t, observedWeaponBuild(t, replay, []combat.Event{event}, 100))
		if row.ActiveWeapon.Provenance.Reconciliation != models.ObservedWeaponReconciliationFire ||
			row.AmmoInMagazine.Provenance.Reconciliation != models.ObservedWeaponReconciliationFire ||
			row.AmmoReserve.Provenance.Reconciliation != models.ObservedWeaponReconciliationFire ||
			row.IsReloading.Provenance.Reconciliation != models.ObservedWeaponReconciliationNotApplicable {
			t.Fatalf("fire reconciliation incorrect: %+v", row)
		}
	})

	t.Run("reload confirms true snapshot", func(t *testing.T) {
		replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 8, 70, true))
		event := observedWeaponCombatEvent(combat.EventWeaponReload, 42, "AK-47", 100, 8, 70)
		row := observedWeaponOnlyRow(t, observedWeaponBuild(t, replay, []combat.Event{event}, 100))
		if row.IsReloading.Value == nil || !*row.IsReloading.Value ||
			row.IsReloading.Provenance.Reconciliation != models.ObservedWeaponReconciliationReload ||
			row.AmmoInMagazine.Provenance.Reconciliation != models.ObservedWeaponReconciliationReload {
			t.Fatalf("reload reconciliation incorrect: %+v", row)
		}
	})

	t.Run("wrong actor weapon or tick cannot confirm", func(t *testing.T) {
		replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 18, 54, false))
		events := []combat.Event{
			observedWeaponCombatEvent(combat.EventWeaponFire, 99, "AK-47", 100, 18, 54),
			observedWeaponCombatEvent(combat.EventWeaponFire, 42, "M4A4", 100, 18, 54),
			observedWeaponCombatEvent(combat.EventWeaponFire, 42, "AK-47", 99, 18, 54),
		}
		row := observedWeaponOnlyRow(t, observedWeaponBuild(t, replay, events, 100))
		if row.ActiveWeapon.Provenance.Reconciliation != models.ObservedWeaponReconciliationNotApplicable ||
			row.AmmoInMagazine.Provenance.Reconciliation != models.ObservedWeaponReconciliationNotApplicable {
			t.Fatalf("non-matching event altered facts: %+v", row)
		}
	})

	t.Run("conflicting fire and reload abstain", func(t *testing.T) {
		replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 8, 70, true))
		events := []combat.Event{
			observedWeaponCombatEvent(combat.EventWeaponFire, 42, "AK-47", 100, 8, 70),
			observedWeaponCombatEvent(combat.EventWeaponReload, 42, "AK-47", 100, 8, 70),
		}
		row := observedWeaponOnlyRow(t, observedWeaponBuild(t, replay, events, 100))
		if row.AmmoInMagazine.Status != models.ObservedWeaponFactUnavailable ||
			row.AmmoReserve.Status != models.ObservedWeaponFactUnavailable ||
			row.IsReloading.Status != models.ObservedWeaponFactUnavailable ||
			!observedWeaponContainsReason(row.AbstentionReasons, weaponAbstentionEventAmbiguity) {
			t.Fatalf("ambiguous events did not abstain: %+v", row)
		}
	})
}

func TestObservedWeaponStateDoesNotInferFromWeaponNameOrNominalCapacity(t *testing.T) {
	tick := 100
	weapon := "AK-47"
	observation := models.ReplayWeaponStateObservation{
		ActiveWeapon: &weapon, ActiveWeaponStatus: models.ObservedWeaponFactObserved,
		ActiveWeaponAvailabilityTick: &tick,
	}
	row := observedWeaponOnlyRow(t, observedWeaponBuild(t, observedWeaponReplay(observation), nil, 100))
	if row.ActiveWeapon.Status != models.ObservedWeaponFactObserved ||
		row.AmmoInMagazine.Status != models.ObservedWeaponFactUnavailable ||
		row.AmmoReserve.Status != models.ObservedWeaponFactUnavailable ||
		row.IsReloading.Status != models.ObservedWeaponFactUnavailable {
		t.Fatalf("weapon name or nominal capacity fabricated facts: %+v", row)
	}
}

func TestObservedWeaponStateDoesNotRewriteCanonicalCombatAmmo(t *testing.T) {
	replay := observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 0, 30, false))
	event := observedWeaponCombatEvent(combat.EventWeaponFire, 42, "AK-47", 100, -1, 0)
	original := event
	original.Ammo.InMagazine = observedWeaponTestInt(*event.Ammo.InMagazine)
	original.Ammo.Reserve = observedWeaponTestInt(*event.Ammo.Reserve)
	_ = observedWeaponBuild(t, replay, []combat.Event{event}, 100)
	if !reflect.DeepEqual(event, original) {
		t.Fatalf("candidate mutated canonical combat event:\nbefore=%+v\nafter=%+v", original, event)
	}
}

func TestObservedWeaponStateCandidateDoesNotReinterpretTacticalOrReplay38(t *testing.T) {
	observation := observedWeaponTestObservation(100, "AK-47", 13, 39, true)
	replay := observedWeaponReplay(observation)
	player := &replay.Rounds[0].Frames[0].Players[0]
	player.Team = "CT"
	player.Alive = true
	player.ObservedActiveWeapon = "AK-47"
	player.ActiveWeaponObservationStatus = models.ActiveWeaponStatusObserved
	player.ActiveWeaponAvailabilityTick = 100

	export, err := BuildTacticalExport(replay, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(export.PhysicalRows) == 0 {
		t.Fatal("expected existing tactical self row")
	}
	state := export.PhysicalRows[0].State
	if state.AmmoInMagazine != nil || state.AmmoReserve != nil || state.IsReloading != nil ||
		state.FieldAvailability["ammo_in_magazine"] != models.TacticalStatusUnavailable ||
		state.FieldAvailability["ammo_reserve"] != models.TacticalStatusUnavailable ||
		state.FieldAvailability["is_reloading"] != models.TacticalStatusUnavailable {
		t.Fatalf("candidate facts leaked into tactical 3.8.0: %+v", state)
	}

	payload, err := json.Marshal(replay)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, internal := range []string{"ObservedWeaponState", "ammo_in_magazine", "ammo_reserve"} {
		if strings.Contains(text, internal) {
			t.Fatalf("in-memory candidate field %q leaked into replay JSON: %s", internal, text)
		}
	}
}

func TestValidateObservedWeaponStateLedgerRejectsTampering(t *testing.T) {
	ledger := observedWeaponBuild(t, observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 30, 90, false)), nil, 100)
	ledger.Rows[0].AmmoInMagazine.Value = nil
	if err := ValidateObservedWeaponStateLedger(ledger); err == nil {
		t.Fatal("observed-null tampering was accepted")
	}

	ledger = observedWeaponBuild(t, observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 30, 90, false)), nil, 100)
	ledger.Rows[0].JoinKeys.PlayerID = "steam:76561198000000000"
	if err := ValidateObservedWeaponStateLedger(ledger); err == nil {
		t.Fatal("raw identity alias was accepted")
	}

	ledger = observedWeaponBuild(t, observedWeaponReplay(observedWeaponTestObservation(100, "AK-47", 30, 90, false)), nil, 100)
	negativeTick := -1
	ledger.Rows[0].AmmoReserve.AvailabilityTick = &negativeTick
	ledger.Rows[0].AmmoReserve.Provenance.SourceTick = &negativeTick
	if err := ValidateObservedWeaponStateLedger(ledger); err == nil {
		t.Fatal("negative availability/source tick was accepted")
	}
}

func observedWeaponReplay(observation models.ReplayWeaponStateObservation) *models.ReplayData {
	return &models.ReplayData{
		Metadata: models.ReplayMetadata{SchemaVersion: 5, MatchID: "match-opaque", TickRate: 64},
		Rounds: []models.ReplayRound{{
			Round: 1, StartTick: 100, EndTick: 100,
			Frames: []models.ReplayFrame{{
				Tick: 100,
				Players: []models.ReplayPlayerState{{
					SteamID: 42, Name: "private-name", Health: 100, Armor: 50,
					ObservedWeaponState: observation,
				}},
			}},
		}},
	}
}

func observedWeaponTestObservation(
	tick int,
	weapon string,
	magazine, reserve int,
	reloading bool,
) models.ReplayWeaponStateObservation {
	return models.ReplayWeaponStateObservation{
		ActiveWeapon: &weapon, ActiveWeaponStatus: models.ObservedWeaponFactObserved,
		ActiveWeaponAvailabilityTick: observedWeaponTestInt(tick),
		AmmoInMagazine:               &magazine, AmmoInMagazineStatus: models.ObservedWeaponFactObserved,
		AmmoInMagazineAvailabilityTick: observedWeaponTestInt(tick),
		AmmoReserve:                    &reserve, AmmoReserveStatus: models.ObservedWeaponFactObserved,
		AmmoReserveAvailabilityTick: observedWeaponTestInt(tick),
		IsReloading:                 &reloading, IsReloadingStatus: models.ObservedWeaponFactObserved,
		IsReloadingAvailabilityTick: observedWeaponTestInt(tick),
	}
}

func observedWeaponCombatEvent(
	eventType combat.EventType,
	actor uint64,
	weapon string,
	tick, magazine, reserve int,
) combat.Event {
	return combat.Event{
		Round: 1, Tick: tick, Type: eventType,
		Actor:  combat.PlayerRef{ID: actor, Status: combat.AvailabilityObserved},
		Weapon: combat.WeaponRef{Name: weapon, Status: combat.AvailabilityObserved},
		Ammo: combat.AmmoObservation{
			InMagazine: observedWeaponTestInt(magazine),
			Reserve:    observedWeaponTestInt(reserve),
			Status:     combat.AvailabilityObserved,
		},
	}
}

func observedWeaponBuild(
	t *testing.T,
	replay *models.ReplayData,
	events []combat.Event,
	cutoff int,
) models.ObservedWeaponStateLedger {
	t.Helper()
	ledger, err := BuildObservedWeaponStateLedger(replay, events, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func observedWeaponOnlyRow(t *testing.T, ledger models.ObservedWeaponStateLedger) models.ObservedWeaponStateRow {
	t.Helper()
	if len(ledger.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(ledger.Rows))
	}
	return ledger.Rows[0]
}

func assertObservedWeaponFactsUnavailable(t *testing.T, row models.ObservedWeaponStateRow, includeWeapon bool) {
	t.Helper()
	if includeWeapon && (row.ActiveWeapon.Status != models.ObservedWeaponFactUnavailable || row.ActiveWeapon.Value != nil) {
		t.Fatalf("active weapon is not unavailable: %+v", row.ActiveWeapon)
	}
	if row.AmmoInMagazine.Status != models.ObservedWeaponFactUnavailable || row.AmmoInMagazine.Value != nil ||
		row.AmmoReserve.Status != models.ObservedWeaponFactUnavailable || row.AmmoReserve.Value != nil ||
		row.IsReloading.Status != models.ObservedWeaponFactUnavailable || row.IsReloading.Value != nil {
		t.Fatalf("dependent facts are not unavailable: %+v", row)
	}
}

func observedWeaponContainsReason(reasons []string, prefix string) bool {
	for _, reason := range reasons {
		if strings.HasPrefix(reason, prefix) {
			return true
		}
	}
	return false
}

func observedWeaponTestInt(value int) *int {
	copy := value
	return &copy
}

func observedWeaponCloneReplay(t *testing.T, replay *models.ReplayData) *models.ReplayData {
	t.Helper()
	clone := *replay
	clone.Rounds = append([]models.ReplayRound(nil), replay.Rounds...)
	for roundIndex := range clone.Rounds {
		clone.Rounds[roundIndex].Frames = append([]models.ReplayFrame(nil), replay.Rounds[roundIndex].Frames...)
		for frameIndex := range clone.Rounds[roundIndex].Frames {
			clone.Rounds[roundIndex].Frames[frameIndex].Players = append(
				[]models.ReplayPlayerState(nil),
				replay.Rounds[roundIndex].Frames[frameIndex].Players...,
			)
		}
	}
	return &clone
}
