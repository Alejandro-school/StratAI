package handlers

import (
	"cs2-demo-service/models"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"testing"
)

type identityFixtureEntity struct {
	st.Entity
	index, serial int
}

func (e *identityFixtureEntity) ID() int        { return e.index }
func (e *identityFixtureEntity) SerialNum() int { return e.serial }

func TestNativeIdentityRequiresFullSerialAndLiveEntity(t *testing.T) {
	e := &identityFixtureEntity{index: 5, serial: 20000}
	tracker := &economyIdentityTracker{live: map[int]*identityEntity{5: {alias: "native-entity-000001", kind: "equipment", entity: e, alive: true}}}
	for _, tc := range []struct {
		value        any
		kind, status string
	}{
		{nil, "equipment", "absent"}, {true, "equipment", "unsupported_type"}, {"5", "equipment", "unsupported_type"},
		{uint64(4294967295), "equipment", "invalid_sentinel"}, {uint64(16383), "equipment", "invalid_sentinel"},
		{uint64(5), "equipment", "serial_mismatch"}, {uint64(6), "equipment", "entity_unavailable"},
		{uint64(20000<<14 | 5), "pawn", "kind_mismatch"}, {uint64(20000<<14 | 5), "equipment", "resolved"},
	} {
		r := tracker.reference(tc.value, tc.kind)
		if r.Status != tc.status {
			t.Fatalf("%v: %+v", tc.value, r)
		}
		if (r.TargetAlias != nil) != (tc.status == "resolved") {
			t.Fatal("invented target")
		}
	}
	// Reusing the index must not preserve the old handle binding, even when the
	// former serial would match after truncation to the SDK's legacy 10 bits.
	e.serial += 1024
	if tracker.reference(uint64(20000<<14|5), "equipment").Status != "serial_mismatch" {
		t.Fatal("serial truncated")
	}
	tracker.live[5].alive = false
	if tracker.reference(uint64(e.serial<<14|5), "equipment").Status != "entity_unavailable" {
		t.Fatal("destroyed entity joined")
	}
}

func TestNativeClockEpochsDoNotFitAnOffset(t *testing.T) {
	ptr := func(v uint32) *uint32 { return &v }
	previous := &models.EconomyIdentityClock{NetworkTick: ptr(100), DemoTick: 10, Epoch: 3}
	for _, tc := range []struct {
		previous    *models.EconomyIdentityClock
		tick        *uint32
		demo, epoch int
		reason      string
	}{
		{nil, ptr(0), 0, 1, "first_sample"}, {previous, ptr(100), 10, 3, "continuous"},
		{previous, ptr(99), 11, 4, "network_tick_regression"}, {previous, ptr(101), 9, 4, "demo_tick_regression"},
		{previous, nil, 11, 4, "missing_tick_boundary"}, {&models.EconomyIdentityClock{Epoch: 4}, ptr(102), 12, 5, "missing_tick_boundary"},
	} {
		epoch, reason := identityEpoch(tc.previous, tc.tick, tc.demo)
		if epoch != tc.epoch || reason != tc.reason {
			t.Fatal(epoch, reason)
		}
	}
}
