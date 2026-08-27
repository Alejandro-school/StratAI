package handlers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func captureBulletDamageEvent(ctx *models.DemoContext, event *events.BulletDamage) {
	if event == nil || event.Attacker == nil || event.Victim == nil {
		return
	}
	if event.Attacker.SteamID64 == event.Victim.SteamID64 || event.Attacker.Team == event.Victim.Team {
		return
	}

	snapshot := models.BulletDamageSnapshot{
		Tick:            ctx.Parser.GameState().IngameTick(),
		AttackerSteamID: event.Attacker.SteamID64,
		VictimSteamID:   event.Victim.SteamID64,
		Distance:        float64(event.Distance),
		DamageDirection: models.AI_Vector{
			X: float64(event.DamageDirX),
			Y: float64(event.DamageDirY),
			Z: float64(event.DamageDirZ),
		},
		NumPenetrations: event.NumPenetrations,
		IsNoScope:       event.IsNoScope,
		IsAttackerInAir: event.IsAttackerInAir,
	}
	ctx.BulletDamageEvents++

	for i := len(ctx.RawCombatEvents) - 1; i >= 0; i-- {
		raw := &ctx.RawCombatEvents[i]
		if raw.Tick < snapshot.Tick {
			break
		}
		if raw.Tick == snapshot.Tick && raw.AttackerSteamID == snapshot.AttackerSteamID && raw.VictimSteamID == snapshot.VictimSteamID && !raw.HasBulletDamage {
			applyBulletDamage(raw, snapshot)
			applyBulletDamageToPersistentEvent(ctx, snapshot)
			ctx.BulletDamageCorrelated++
			return
		}
	}

	ctx.PendingBulletDamage = append(ctx.PendingBulletDamage, snapshot)
}

func applyBulletDamageToPersistentEvent(ctx *models.DemoContext, snapshot models.BulletDamageSnapshot) {
	for i := len(ctx.AI_CombatEvents) - 1; i >= 0; i-- {
		event := &ctx.AI_CombatEvents[i]
		if event.Tick < snapshot.Tick {
			return
		}
		if event.Tick == snapshot.Tick && event.AttackerSteamID == snapshot.AttackerSteamID && event.VictimSteamID == snapshot.VictimSteamID && !event.HasBulletDamage {
			applyBulletDamage(event, snapshot)
			return
		}
	}
}

func correlatePendingBulletDamage(ctx *models.DemoContext, raw *models.RawCombatEvent) {
	for i, snapshot := range ctx.PendingBulletDamage {
		if snapshot.Tick != raw.Tick || snapshot.AttackerSteamID != raw.AttackerSteamID || snapshot.VictimSteamID != raw.VictimSteamID {
			continue
		}
		applyBulletDamage(raw, snapshot)
		ctx.PendingBulletDamage = append(ctx.PendingBulletDamage[:i], ctx.PendingBulletDamage[i+1:]...)
		ctx.BulletDamageCorrelated++
		return
	}
}

func discardStaleBulletDamage(ctx *models.DemoContext, currentTick int) {
	kept := ctx.PendingBulletDamage[:0]
	for _, snapshot := range ctx.PendingBulletDamage {
		if snapshot.Tick >= currentTick {
			kept = append(kept, snapshot)
		}
	}
	ctx.PendingBulletDamage = kept
}

func applyBulletDamage(raw *models.RawCombatEvent, snapshot models.BulletDamageSnapshot) {
	raw.HasBulletDamage = true
	raw.BulletDistance = snapshot.Distance
	raw.DamageDirection = snapshot.DamageDirection
	if snapshot.NumPenetrations > raw.PenetratedObjects {
		raw.PenetratedObjects = snapshot.NumPenetrations
	}
	raw.IsWallbang = raw.IsWallbang || snapshot.NumPenetrations > 0
	raw.NoScope = raw.NoScope || snapshot.IsNoScope
	raw.AttackerInAir = snapshot.IsAttackerInAir
}
