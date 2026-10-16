package handlers

import (
	"cs2-demo-service/models"
	"fmt"
)

func observeReplayInventory(previous *models.ReplayInventoryContext, alive bool, tick int, weapons []string, activeWeapon string, hasHelmet bool) *models.ReplayInventoryContext {
	if alive && len(weapons) > 0 {
		result := &models.ReplayInventoryContext{
			SchemaID: "stratai.replay_inventory@1", Status: "observed",
			ObservationTick: &tick, AvailabilityTick: &tick,
			Source:  map[string]string{"kind": "demo_inventory_snapshot", "reference": fmt.Sprintf("frame:%d:inventory", tick)},
			Weapons: append([]string{}, weapons...), HasHelmet: &hasHelmet,
		}
		if activeWeapon != "" {
			result.Weapon = &activeWeapon
		}
		return result
	}
	if previous != nil && previous.ObservationTick != nil {
		result := *previous
		result.Status = "last_observed"
		result.Weapons = append([]string{}, previous.Weapons...)
		result.Source = map[string]string{}
		for key, value := range previous.Source {
			result.Source[key] = value
		}
		return &result
	}
	return &models.ReplayInventoryContext{SchemaID: "stratai.replay_inventory@1", Status: "unavailable", Weapons: []string{}}
}
