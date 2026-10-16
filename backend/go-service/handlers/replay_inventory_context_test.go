package handlers

import (
	"encoding/json"
	"reflect"
	"testing"
)

func inventoryWire(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestReplayInventoryContextRecordsFreshNativeLoadoutWithoutAliasing(t *testing.T) {
	weapons := []string{"AK-47", "Knife"}
	observation := observeReplayInventory(nil, true, 100, weapons, "AK-47", true)
	weapons[0] = "mutated input"
	actual := inventoryWire(t, observation)
	if actual["schema_id"] != "stratai.replay_inventory@1" || actual["status"] != "observed" ||
		actual["observation_tick"] != float64(100) || actual["availability_tick"] != float64(100) {
		t.Fatalf("fresh inventory lacks causal provenance: %+v", actual)
	}
	if actual["weapon"] != "AK-47" || actual["has_helmet"] != true ||
		!reflect.DeepEqual(actual["weapons"], []any{"AK-47", "Knife"}) {
		t.Fatalf("native inventory was lost or aliased: %+v", actual)
	}
	source, ok := actual["source"].(map[string]any)
	if !ok || source["kind"] != "demo_inventory_snapshot" || source["reference"] != "frame:100:inventory" {
		t.Fatalf("fresh inventory source differs: %+v", actual["source"])
	}
}

func TestReplayInventoryContextDeathPreservesTheOriginalObservationDate(t *testing.T) {
	previous := observeReplayInventory(nil, true, 100, []string{"Desert Eagle", "Knife"}, "Desert Eagle", true)
	before := inventoryWire(t, previous)
	dead := observeReplayInventory(previous, false, 300, []string{"Knife"}, "Knife", false)
	actual := inventoryWire(t, dead)
	if actual["status"] != "last_observed" || actual["observation_tick"] != float64(100) || actual["availability_tick"] != float64(100) ||
		actual["weapon"] != "Desert Eagle" || !reflect.DeepEqual(actual["weapons"], []any{"Desert Eagle", "Knife"}) {
		t.Fatalf("death falsely refreshed or replaced the last live inventory: %+v", actual)
	}
	if !reflect.DeepEqual(actual["source"], before["source"]) || !reflect.DeepEqual(inventoryWire(t, previous), before) {
		t.Fatal("historical inventory rewrote its original snapshot or source")
	}
}

func TestReplayInventoryContextLiveMissingObservationNeverRefreshesCache(t *testing.T) {
	previous := observeReplayInventory(nil, true, 100, []string{"AWP", "Knife"}, "AWP", true)
	actual := inventoryWire(t, observeReplayInventory(previous, true, 104, nil, "Knife", false))
	if actual["status"] != "last_observed" || actual["observation_tick"] != float64(100) || actual["availability_tick"] != float64(100) ||
		actual["weapon"] != "AWP" || actual["has_helmet"] != true {
		t.Fatalf("missing live inventory manufactured a fresh sample: %+v", actual)
	}
}

func TestReplayInventoryContextNoPriorLiveObservationRemainsUnavailable(t *testing.T) {
	for _, alive := range []bool{false, true} {
		var weapons []string
		if !alive {
			weapons = []string{"Knife"} // A corpse fallback is not a live observation.
		}
		actual := inventoryWire(t, observeReplayInventory(nil, alive, 100, weapons, "Knife", true))
		if actual["status"] != "unavailable" || actual["observation_tick"] != nil || actual["availability_tick"] != nil ||
			actual["weapon"] != nil || actual["has_helmet"] != nil || !reflect.DeepEqual(actual["weapons"], []any{}) {
			t.Fatalf("invented equipment without a live inventory (alive=%v): %+v", alive, actual)
		}
	}
}

func TestReplayInventoryContextNewLoadoutDoesNotBorrowTheCachedActiveWeapon(t *testing.T) {
	previous := observeReplayInventory(nil, true, 100, []string{"AK-47", "Knife"}, "AK-47", true)
	actual := inventoryWire(t, observeReplayInventory(previous, true, 104, []string{"Knife"}, "", false))
	if actual["status"] != "observed" || actual["weapon"] != nil || actual["observation_tick"] != float64(104) ||
		actual["has_helmet"] != false || !reflect.DeepEqual(actual["weapons"], []any{"Knife"}) {
		t.Fatalf("new inventory reused an unobserved cached active weapon: %+v", actual)
	}
}

func TestReplayInventoryContextObservedTickZeroIsNotAbsent(t *testing.T) {
	actual := inventoryWire(t, observeReplayInventory(nil, true, 0, []string{"Knife"}, "Knife", false))
	if actual["observation_tick"] != float64(0) || actual["availability_tick"] != float64(0) || actual["status"] != "observed" {
		t.Fatalf("observed zero tick became unavailable: %+v", actual)
	}
}
