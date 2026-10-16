package handlers

import (
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func TestReplayReloadCallbackAbsentStaysUnavailable(t *testing.T) {
	tracker := newReplayReloadTracker()
	observation := tracker.refreshSnapshot(7, true, playerstate.EquipmentEntityID(70, 1), false, 100)
	result := unavailableReplayWeaponState()
	applyReplayReloadObservation(&result, observation, 100)

	if observation.available || result.IsReloading != nil ||
		result.IsReloadingStatus != models.ObservedWeaponFactUnavailable {
		t.Fatalf("callback-free zero value became observed: observation=%+v result=%+v", observation, result)
	}
	if len(result.AbstentionReasons) != 1 || result.AbstentionReasons[0] != replayReloadAbstentionUnavailable {
		t.Fatalf("missing callback abstention: %v", result.AbstentionReasons)
	}
}

func TestReplayReloadStartCallbackAccreditsTrue(t *testing.T) {
	tracker := newReplayReloadTracker()
	weaponID := playerstate.EquipmentEntityID(71, 1)
	tracker.record(7, weaponID, true, 101, replayReloadSignalStart)

	observation := tracker.refreshSnapshot(7, true, weaponID, true, 101)
	result := unavailableReplayWeaponState()
	applyReplayReloadObservation(&result, observation, 101)
	if !observation.available || observation.signal != replayReloadSignalStart ||
		result.IsReloading == nil || !*result.IsReloading ||
		result.IsReloadingAvailabilityTick == nil || *result.IsReloadingAvailabilityTick != 101 ||
		result.IsReloadingStatus != models.ObservedWeaponFactObserved {
		t.Fatalf("reload start was not retained as observed true: observation=%+v result=%+v", observation, result)
	}
}

func TestReplayReloadFalseRequiresAccreditedSignal(t *testing.T) {
	for _, test := range []struct {
		name   string
		signal replayReloadSignal
	}{
		{name: "item equip", signal: replayReloadSignalItemEquip},
		{name: "weapon fire", signal: replayReloadSignalWeaponFire},
		{name: "clip update", signal: replayReloadSignalClipUpdate},
	} {
		t.Run(test.name, func(t *testing.T) {
			tracker := newReplayReloadTracker()
			weaponID := playerstate.EquipmentEntityID(90, 1)
			tracker.record(9, weaponID, false, 55, test.signal)
			observation := tracker.refreshSnapshot(9, true, weaponID, false, 56)
			result := unavailableReplayWeaponState()
			applyReplayReloadObservation(&result, observation, 56)

			if !observation.available || observation.signal != test.signal ||
				result.IsReloading == nil || *result.IsReloading ||
				result.IsReloadingAvailabilityTick == nil || *result.IsReloadingAvailabilityTick != 56 ||
				result.IsReloadingStatus != models.ObservedWeaponFactObserved {
				t.Fatalf("accredited false was not retained: observation=%+v result=%+v", observation, result)
			}
		})
	}

	tracker := newReplayReloadTracker()
	weaponID := playerstate.EquipmentEntityID(91, 1)
	tracker.record(9, weaponID, false, 55, replayReloadSignalStart)
	if observation := tracker.refreshSnapshot(9, true, weaponID, false, 55); observation.available {
		t.Fatalf("reload-start signal incorrectly accredited false: %+v", observation)
	}
}

func TestReplayReloadTrackerResetsAndScopesStateToWeapon(t *testing.T) {
	tracker := newReplayReloadTracker()
	firstWeaponID := playerstate.EquipmentEntityID(110, 1)
	secondWeaponID := playerstate.EquipmentEntityID(111, 1)
	tracker.record(11, firstWeaponID, true, 20, replayReloadSignalStart)

	if observation := tracker.refreshSnapshot(11, true, secondWeaponID, false, 20); observation.available {
		t.Fatalf("reload state crossed active weapons: %+v", observation)
	}
	if observation := tracker.refreshSnapshot(11, true, firstWeaponID, true, 20); observation.available {
		t.Fatalf("stale state revived after an unaccredited weapon switch: %+v", observation)
	}
	tracker.record(11, secondWeaponID, false, 21, replayReloadSignalItemEquip)
	if observation := tracker.refreshSnapshot(11, true, secondWeaponID, false, 21); !observation.available || observation.value {
		t.Fatalf("weapon callback did not accredit the replacement weapon: %+v", observation)
	}

	tracker.resetRound()
	if observation := tracker.refreshSnapshot(11, true, secondWeaponID, false, 22); observation.available {
		t.Fatalf("reload state carried across round reset: %+v", observation)
	}
}

func TestReplayReloadTrackerRejectsFutureAndOutOfOrderSignals(t *testing.T) {
	tracker := newReplayReloadTracker()
	weaponID := playerstate.EquipmentEntityID(130, 1)
	tracker.record(13, weaponID, true, 40, replayReloadSignalStart)
	if observation := tracker.refreshSnapshot(13, true, weaponID, true, 39); observation.available {
		t.Fatalf("future callback was exposed at T0: %+v", observation)
	}
	result := unavailableReplayWeaponState()
	applyReplayReloadObservation(&result, replayReloadObservation{
		value: true, available: true, availabilityTick: 40,
		weaponID: weaponID, signal: replayReloadSignalStart,
	}, 39)
	if result.IsReloading != nil || result.IsReloadingStatus != models.ObservedWeaponFactUnavailable {
		t.Fatalf("future observation bypassed the final causal guard: %+v", result)
	}

	tracker.record(13, weaponID, false, 38, replayReloadSignalClipUpdate)
	observation := tracker.refreshSnapshot(13, true, weaponID, true, 40)
	if !observation.available || !observation.value || observation.availabilityTick != 40 ||
		observation.signal != replayReloadSignalStart {
		t.Fatalf("out-of-order callback replaced newer truth: %+v", observation)
	}
}

func TestReplayReloadTrackerDistinguishesSameNamedWeaponEntities(t *testing.T) {
	tracker := newReplayReloadTracker()
	firstAK := playerstate.EquipmentEntityID(200, 1)
	secondAK := playerstate.EquipmentEntityID(200, 2)
	tracker.record(21, firstAK, true, 10, replayReloadSignalStart)

	if observation := tracker.refreshSnapshot(21, true, secondAK, false, 11); observation.available {
		t.Fatalf("same weapon name with a different entity reused state: %+v", observation)
	}
	if observation := tracker.refreshSnapshot(21, true, firstAK, true, 11); observation.available {
		t.Fatalf("entity mismatch did not invalidate the original state: %+v", observation)
	}
}

func TestReplayReloadTrackerRefreshesAtSampleTick(t *testing.T) {
	tracker := newReplayReloadTracker()
	weaponID := playerstate.EquipmentEntityID(220, 3)
	tracker.record(22, weaponID, true, 10, replayReloadSignalStart)

	stale := tracker.observations[22]
	staleResult := unavailableReplayWeaponState()
	applyReplayReloadObservation(&staleResult, stale, 12)
	if staleResult.IsReloading != nil {
		t.Fatalf("callback tick was published as a later frame snapshot: %+v", staleResult)
	}

	observation := tracker.refreshSnapshot(22, true, weaponID, false, 12)
	result := unavailableReplayWeaponState()
	applyReplayReloadObservation(&result, observation, 12)
	if !observation.available || observation.value || observation.availabilityTick != 12 ||
		result.IsReloading == nil || *result.IsReloading ||
		result.IsReloadingAvailabilityTick == nil || *result.IsReloadingAvailabilityTick != 12 {
		t.Fatalf("valid native transition was not refreshed at the sample tick: observation=%+v result=%+v", observation, result)
	}
}

func TestReplayReloadTrackerInvalidatesDeathMissingEntityAndRespawn(t *testing.T) {
	weaponID := playerstate.EquipmentEntityID(230, 1)
	for _, test := range []struct {
		name           string
		alive          bool
		activeWeaponID int64
	}{
		{name: "dead player", alive: false, activeWeaponID: weaponID},
		{name: "nil weapon entity", alive: true, activeWeaponID: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			tracker := newReplayReloadTracker()
			tracker.record(23, weaponID, true, 10, replayReloadSignalStart)
			if observation := tracker.refreshSnapshot(23, test.alive, test.activeWeaponID, true, 11); observation.available {
				t.Fatalf("invalid snapshot retained reload state: %+v", observation)
			}
			if observation := tracker.refreshSnapshot(23, true, weaponID, true, 12); observation.available {
				t.Fatalf("invalidated reload state revived: %+v", observation)
			}
		})
	}
}

func TestReplayReloadClipRejectsNilAndSentinelValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		value st.PropertyValue
	}{
		{name: "nil", value: st.PropertyValue{}},
		{name: "negative sentinel", value: st.PropertyValue{Any: int32(-1)}},
		{name: "unexpected type", value: st.PropertyValue{Any: uint32(7)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			tracker := newReplayReloadTracker()
			weaponID := playerstate.EquipmentEntityID(235, 1)
			tracker.record(235, weaponID, true, 10, replayReloadSignalStart)
			if _, available := replayReloadClipValue(test.value); available {
				t.Fatalf("invalid clip was accepted: %#v", test.value.Any)
			}
			tracker.invalidateWeapon(235, weaponID)
			if observation := tracker.refreshSnapshot(235, true, weaponID, true, 11); observation.available {
				t.Fatalf("invalid clip preserved entity state: %+v", observation)
			}
		})
	}

	if value, available := replayReloadClipValue(st.PropertyValue{Any: int32(0)}); !available || value != 0 {
		t.Fatalf("observed empty clip was rejected: value=%d available=%v", value, available)
	}
}

func TestReplayReloadTrackerRejectsMismatchedActiveEventEntity(t *testing.T) {
	tracker := newReplayReloadTracker()
	activeWeaponID := playerstate.EquipmentEntityID(240, 1)
	eventWeaponID := playerstate.EquipmentEntityID(241, 1)
	tracker.record(24, activeWeaponID, true, 14, replayReloadSignalStart)
	tracker.recordActiveSignal(
		24, true, activeWeaponID, eventWeaponID, false, 15, replayReloadSignalWeaponFire,
	)
	if observation := tracker.refreshSnapshot(24, true, activeWeaponID, false, 15); observation.available {
		t.Fatalf("mismatched event weapon accredited reload false: %+v", observation)
	}
}
