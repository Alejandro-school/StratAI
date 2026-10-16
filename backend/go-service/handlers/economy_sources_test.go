package handlers

import (
	"reflect"
	"testing"

	"cs2-demo-service/models"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type sourceTestEntity struct {
	st.Entity
	values map[string]any
}
type sourceTestProperty struct {
	st.Property
	value any
}

func (p sourceTestProperty) Value() st.PropertyValue { return st.PropertyValue{Any: p.value} }
func (e sourceTestEntity) Property(name string) st.Property {
	v, exists := e.values[name]
	if !exists {
		return nil
	}
	return sourceTestProperty{value: v}
}

func TestNativePurchaseArrayDoesNotTurnMissingIntoEmpty(t *testing.T) {
	for _, tc := range []struct {
		name           string
		values         map[string]any
		status, reason string
		length         *int
	}{
		{"missing", map[string]any{}, "unavailable", "native_array_unavailable", nil},
		{"nil", map[string]any{purchasesRoundSource: nil}, "unavailable", "native_array_unavailable", nil},
		{"empty", map[string]any{purchasesRoundSource: []any{}}, "observed", "native_array_and_members_observed", intPointer(0)},
		{"wrong type", map[string]any{purchasesRoundSource: 0}, "unavailable", "native_array_shape_unsupported", nil},
		{"oversized", map[string]any{purchasesRoundSource: make([]any, 257)}, "unavailable", "native_array_shape_unsupported", nil},
		{"member missing", map[string]any{purchasesRoundSource: []any{nil}}, "partial", "native_member_unavailable", intPointer(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := nativeSourceArray(sourceTestEntity{values: tc.values}, purchasesRoundSource, purchaseMembers, 100)
			if got.Status != tc.status || got.Reason != tc.reason || !reflect.DeepEqual(got.Length, tc.length) {
				t.Fatalf("unexpected native array: %+v", got)
			}
		})
	}
}

func TestNativePurchaseMembersPreserveZeroFalseAndRejectInvalidTypes(t *testing.T) {
	values := map[string]any{sellbackSource: []any{nil}}
	for _, name := range sellbackMembers {
		values[sellbackSource+".0000."+name] = uint64(0)
	}
	values[sellbackSource+".0000.m_bPrevHelmet"] = false
	values[sellbackSource+".0000.m_nPrevArmor"] = int32(-1)
	values[sellbackSource+".0000.m_hItem"] = uint64(4294967295)
	entity := sourceTestEntity{values: values}
	got := nativeSourceArray(entity, sellbackSource, sellbackMembers, 100)
	if got.Entries[0]["m_nPrevArmor"].Value != int64(-1) {
		t.Fatal("native previous-armor sentinel was lost", got)
	}
	if got.Status != "observed" || got.Entries[0]["m_bPrevHelmet"].Value != false || got.Entries[0]["m_nCost"].Value != int64(0) {
		t.Fatal(got)
	}
	values[sellbackSource+".0000.m_nCost"] = int64(-1)
	changed := nativeSourceArray(entity, sellbackSource, sellbackMembers, 101)
	if changed.Status != "partial" || got.Entries[0]["m_nCost"].Value != int64(0) || *got.Entries[0]["m_nCost"].AvailabilityTick != 100 {
		t.Fatal("snapshot changed or invalid value accepted")
	}
	for _, v := range []any{true, float64(3), "800", uint64(1 << 63)} {
		values[sellbackSource+".0000.m_nCost"] = v
		if nativeSourceArray(entity, sellbackSource, sellbackMembers, 102).Status != "partial" {
			t.Fatalf("invalid cost accepted: %v", v)
		}
	}
}

func TestNativeBuildPresenceIsNotDefaultVersion(t *testing.T) {
	zero, valid := int32(0), int32(14000)
	for _, v := range []*int32{nil, &zero} {
		got := sourceVersionFact(v, "CDemoFileHeader.build_num", 0)
		if got.Status != "unavailable" || got.Value != nil || got.AvailabilityTick != nil {
			t.Fatal(got)
		}
	}
	got := sourceVersionFact(&valid, "CDemoFileHeader.build_num", 0)
	valid = 15000
	if got.Value != int32(14000) || *got.AvailabilityTick != 0 {
		t.Fatal("version observation was mutated")
	}
}

func TestNativeChangeDetectionIgnoresOnlyTimestamps(t *testing.T) {
	one := models.EconomySourceSnapshot{Tick: 100, ActorAlias: "p0001", Side: "CT", Connected: true, Money: observedEconomyFact(800, "money", 100)}
	two := one
	two.Tick = 101
	two.Money = observedEconomyFact(800, "money", 101)
	if nativeSourceFingerprint(one) != nativeSourceFingerprint(two) {
		t.Fatal("time alone emitted a changed snapshot")
	}
	two.Money = observedEconomyFact(0, "money", 101)
	if nativeSourceFingerprint(one) == nativeSourceFingerprint(two) {
		t.Fatal("zero change lost")
	}
	two = one
	two.Connected = false
	if nativeSourceFingerprint(one) == nativeSourceFingerprint(two) {
		t.Fatal("disconnect lost")
	}
	two = one
	two.Side = "T"
	if nativeSourceFingerprint(one) == nativeSourceFingerprint(two) {
		t.Fatal("side change lost")
	}
}

func TestNativeDisconnectedActorDoesNotReuseResidualEntityValues(t *testing.T) {
	entity := sourceTestEntity{values: map[string]any{sellbackSource: []any{}, purchasesRoundSource: []any{}, purchasesMatchSource: []any{}}}
	row := nativeSourceActor(1, 100, "p0001", "CT", false, entity, entity)
	if row.Connected || row.Money.Value != nil || row.SpentNative.Value != nil || row.Sellback.Status != "unavailable" || row.PurchasesRound.Status != "unavailable" {
		t.Fatal("disconnected properties were reused", row)
	}
}

func TestNativeSourceCannotAttachNextFreezeToPreviousRound(t *testing.T) {
	round := models.EconomyObservedRound{StartTick: 100, FreezeEndTick: intPointer(200)}
	for _, tc := range []struct {
		tick                     int
		warmup, freeze, expected bool
	}{
		{99, false, true, false}, {100, false, true, true}, {199, false, true, true},
		{200, false, false, true}, {201, false, true, false}, {500, false, true, false}, {100, true, true, false},
	} {
		if nativeSourceTickAllowed(round, tc.tick, tc.warmup, tc.freeze) != tc.expected {
			t.Fatalf("bad native window: %+v", tc)
		}
	}
}
