package handlers

import (
	"testing"

	"cs2-demo-service/models"
)

func TestApplyBulletDamageToPersistentEventKeepsCanonicalCopyCorrelated(t *testing.T) {
	ctx := &models.DemoContext{AI_CombatEvents: []models.RawCombatEvent{{
		Tick: 42, AttackerSteamID: 11, VictimSteamID: 22,
	}}}
	snapshot := models.BulletDamageSnapshot{
		Tick: 42, AttackerSteamID: 11, VictimSteamID: 22,
		Distance: 512, NumPenetrations: 1, IsNoScope: true,
	}

	applyBulletDamageToPersistentEvent(ctx, snapshot)

	event := ctx.AI_CombatEvents[0]
	if !event.HasBulletDamage || event.BulletDistance != 512 || event.PenetratedObjects != 1 || !event.NoScope {
		t.Fatalf("persistent event was not correlated: %+v", event)
	}
}
