package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"fmt"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"math"
)

func entryScalar(entity st.Entity, path string) models.EconomyEntryScalar {
	r := models.EconomyEntryScalar{Status: "absent"}
	if entity == nil || entity.Property(path) == nil || entity.Property(path).Value().Any == nil {
		return r
	}
	v := entity.Property(path).Value().Any
	typ := fmt.Sprintf("%T", v)
	r.NativeType, r.Status = &typ, "unsupported_type"
	valid := false
	switch n := v.(type) {
	case bool:
		valid = true
	case int:
		valid = int64(n) >= math.MinInt32 && int64(n) <= math.MaxUint32
	case int32:
		valid = true
	case uint32:
		valid = true
	case int64:
		valid = n >= math.MinInt32 && n <= math.MaxUint32
	case uint64:
		valid = n <= math.MaxUint32
	}
	if valid {
		r.Status, r.Value = "observed", v
	}
	return r
}

func (t *economyIdentityTracker) entryPawn(e *identityEntity) models.EconomyEntryPawn {
	p := models.EconomyEntryPawn{EntityAlias: e.alias, Status: "absent", Entries: []models.EconomyEntry{}}
	prop := e.entity.Property(sellbackSource)
	if prop == nil || prop.Value().Any == nil {
		return p
	}
	p.Status = "unsupported_type"
	array, ok := prop.Value().Any.([]any)
	if !ok {
		return p
	}
	n := len(array)
	p.Length, p.Status = &n, "outside_probe"
	if n > 256 {
		return p
	}
	p.Status = "observed"
	for index := range array {
		entry := models.EconomyEntry{Index: index, Fields: map[string]models.EconomyEntryScalar{}, EntityDefinition: models.EconomyEntryScalar{Status: "absent"}}
		for _, member := range sellbackMembers {
			field := entryScalar(e.entity, fmt.Sprintf("%s.%04d.%s", sellbackSource, index, member))
			entry.Fields[member] = field
			if field.Status != "observed" {
				p.Status = "partial"
			}
		}
		handle := entry.Fields["m_hItem"]
		entry.Item = t.reference(handle.Value, "equipment")
		if handle.Status == "unsupported_type" {
			entry.Item.Status = "unsupported_type"
		}
		if entry.Item.Status == "resolved" {
			equipment := t.live[int(*entry.Item.Handle&16383)]
			entry.EntityDefinition = entryScalar(equipment.entity, "m_iItemDefinitionIndex")
		}
		p.Entries = append(p.Entries, entry)
	}
	return p
}

// Synchronous fresh read inside ownership FrameDone; no parser advance.
func (t *economyIdentityTracker) captureEntries(entities []*identityEntity, cut models.EconomyOwnershipCut) {
	ledger := t.ctx.EconomyEntryObservations
	if ledger == nil {
		return
	}
	state := models.EconomyEntryState{OwnershipState: cut.StateSequence, Pawns: []models.EconomyEntryPawn{}}
	for _, e := range entities {
		if e.kind == "pawn" {
			state.Pawns = append(state.Pawns, t.entryPawn(e))
		}
	}
	fingerprint, _ := json.Marshal(state)
	if len(ledger.States) == 0 || string(fingerprint) != t.entryFingerprint {
		state.Sequence, state.FirstCut = len(ledger.States)+1, cut.Sequence
		ledger.States = append(ledger.States, state)
		t.entryFingerprint = string(fingerprint)
	}
	ledger.Cuts = append(ledger.Cuts, models.EconomyEntryCut{OwnershipCut: cut.Sequence, StateSequence: len(ledger.States)})
	t.captureCashflow(entities, cut, len(ledger.States))
}

func RegisterEconomyEntryHandler(ctx *models.DemoContext) {
	ctx.EconomyEntryObservations = &models.EconomyEntryObservations{Cuts: []models.EconomyEntryCut{}, States: []models.EconomyEntryState{}}
	registerEconomyNativeIdentity(ctx, true)
}
