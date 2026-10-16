package handlers

import (
	"cs2-demo-service/models"
	"fmt"
	"testing"
)

func entryTrackerFixture() (*economyIdentityTracker, *ownershipGameState, *ownershipEntity, *ownershipEntity) {
	t, gs, pawn, weapon := ownershipTrackerFixture()
	t.ctx.EconomyEntryObservations = &models.EconomyEntryObservations{}
	pawn.values[sellbackSource] = []any{nil}
	for k, v := range map[string]any{"m_hItem": uint64(20<<14 | 5), "m_unDefIdx": uint32(7), "m_nCost": int32(0), "m_nPrevArmor": int32(-1), "m_bPrevHelmet": uint32(0)} {
		pawn.values[sellbackSource+".0000."+k] = v
	}
	weapon.values["m_iItemDefinitionIndex"] = uint64(7)
	return t, gs, pawn, weapon
}

func TestEntryCutsAreFreshAndShareExactOwnershipFrame(t *testing.T) {
	tracker, gs, pawn, _ := entryTrackerFixture()
	tracker.captureOwnership()
	tracker.frame++ // A distinct command can have the SAME demo tick.
	tracker.captureOwnership()
	ledger := tracker.ctx.EconomyEntryObservations
	if len(ledger.Cuts) != 2 || len(ledger.States) != 1 {
		t.Fatal("fresh identical cut lost")
	}
	pawn.values[sellbackSource+".0000.m_nCost"] = int32(300)
	tracker.frame++
	tracker.captureOwnership()
	if len(ledger.States) != 2 || ledger.States[1].FirstCut != 3 || ledger.States[1].OwnershipState != 1 {
		t.Fatal("same-tick entry change lost its common cut")
	}
	first := ledger.States[0].Pawns[0].Entries[0]
	if first.Fields["m_nCost"].Value != int32(0) || first.Fields["m_nPrevArmor"].Value != int32(-1) || first.Fields["m_bPrevHelmet"].Value != uint32(0) {
		t.Fatal("raw metadata normalized or backfilled")
	}
	if first.Item.Status != "resolved" || first.EntityDefinition.Value != uint64(7) {
		t.Fatal("full native reference missing")
	}
	gs.tick = 10
	tracker.captureOwnership()
	if len(ledger.Cuts) != 3 {
		t.Fatal("entry escaped ownership strict window")
	}
}

func TestEntryListsPreserveDuplicatesAbsenceAndSpecialValues(t *testing.T) {
	tracker, _, pawn, _ := entryTrackerFixture()
	pawn.values[sellbackSource] = []any{nil, nil}
	for _, k := range sellbackMembers {
		pawn.values[fmt.Sprintf("%s.0001.%s", sellbackSource, k)] = pawn.values[sellbackSource+".0000."+k]
	}
	tracker.captureOwnership()
	p := tracker.ctx.EconomyEntryObservations.States[0].Pawns[0]
	if len(p.Entries) != 2 || p.Entries[1].Index != 1 {
		t.Fatal("duplicate ordinal lost")
	}
	delete(pawn.values, sellbackSource+".0000.m_nCost")
	pawn.values[sellbackSource+".0000.m_nPrevArmor"] = uint32(4294967295)
	pawn.values[sellbackSource+".0000.m_hItem"] = uint64(16777215)
	p = tracker.entryPawn(tracker.live[1])
	if p.Status != "partial" || p.Entries[0].Fields["m_nCost"].Status != "absent" || p.Entries[0].Fields["m_nPrevArmor"].Value != uint32(4294967295) || p.Entries[0].Item.Status != "invalid_sentinel" {
		t.Fatal("special metadata or missing cost changed")
	}
	for _, raw := range []any{nil, "unsupported", make([]any, 257), []any{}} {
		pawn.values[sellbackSource] = raw
		p = tracker.entryPawn(tracker.live[1])
		if raw == nil && (p.Status != "absent" || p.Length != nil) {
			t.Fatal("absence became empty")
		}
		if a, ok := raw.([]any); ok && len(a) == 0 && (p.Status != "observed" || *p.Length != 0) {
			t.Fatal("observed empty list lost")
		}
	}
}

func TestEntryReferenceRechecksSeriesAndLateActorAtEveryCut(t *testing.T) {
	tracker, _, pawn, weapon := entryTrackerFixture()
	weapon.serial = 20000
	pawn.values[sellbackSource+".0000.m_hItem"] = uint64(20000<<14 | 5)
	tracker.captureOwnership()
	tracker.ctx.EconomyObservations.ActorAliases[7] = "p0001"
	tracker.frame++
	weapon.serial = 20001
	tracker.captureOwnership()
	states := tracker.ctx.EconomyEntryObservations.States
	if states[0].Pawns[0].Entries[0].Item.Status != "resolved" || states[1].Pawns[0].Entries[0].Item.Status != "serial_mismatch" {
		t.Fatal("full serial reused")
	}
	own := tracker.ctx.EconomyOwnershipObservations.States
	if own[0].Pawns[0].ActorAlias != nil || own[1].Pawns[0].ActorAlias == nil {
		t.Fatal("actor backfilled")
	}
	pawn.values[sellbackSource+".0000.m_hItem"] = uint64(0)
	p := tracker.entryPawn(tracker.live[1])
	if p.Entries[0].Item.Handle == nil || *p.Entries[0].Item.Handle != 0 {
		t.Fatal("zero handle lost presence")
	}
}
