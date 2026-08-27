package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
	"fmt"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func registerAtomicCombatHandlers(ctx *models.DemoContext) {
	activeRound := 0
	ctx.Parser.RegisterEventHandler(func(event events.RoundStart) {
		gameState := ctx.Parser.GameState()
		if gameState == nil || gameState.IsWarmupPeriod() {
			activeRound = 0
			return
		}
		activeRound = replayRoundNumber(gameState.TotalRoundsPlayed())
	})

	ctx.Parser.RegisterEventHandler(func(event events.WeaponFire) {
		if !acceptAtomicCombatCallback(ctx, activeRound, combat.EventWeaponFire) {
			return
		}
		tick := combatTick(ctx)
		actor := combatPlayerRef(event.Shooter)
		weapon := combatWeaponRef(event.Weapon, combat.SourceCallbackWeapon)
		var yaw, pitch *float64
		if event.Shooter != nil {
			yawValue := float64(event.Shooter.ViewDirectionX())
			pitchValue := float64(event.Shooter.ViewDirectionY())
			if finite(yawValue) && finite(pitchValue) {
				yaw, pitch = &yawValue, &pitchValue
			}
		}
		ctx.Combat.RecordWeaponFire(combat.FireInput{
			Round: activeRound, Tick: tick, Actor: actor, Weapon: weapon,
			ViewYaw: yaw, ViewPitch: pitch,
			Ammo: combatAmmo(event.Shooter, event.Weapon),
		})
	})

	ctx.Parser.RegisterEventHandler(func(event events.BulletDamage) {
		if !acceptAtomicCombatCallback(ctx, activeRound, combat.EventBulletDamage) {
			return
		}
		ctx.Combat.RecordBulletDamage(combat.BulletDamageInput{
			Round: activeRound, Tick: combatTick(ctx),
			Actor: combatPlayerRef(event.Attacker), Target: combatPlayerRef(event.Victim),
			Distance:          float64(event.Distance),
			Direction:         combat.Vector{X: float64(event.DamageDirX), Y: float64(event.DamageDirY), Z: float64(event.DamageDirZ)},
			PenetratedObjects: event.NumPenetrations, NoScope: event.IsNoScope,
			AttackerInAir: event.IsAttackerInAir,
		})
	})

	ctx.Parser.RegisterEventHandler(func(event events.PlayerHurt) {
		if !acceptAtomicCombatCallback(ctx, activeRound, combat.EventPlayerHurt) {
			return
		}
		ctx.Combat.RecordPlayerHurt(combat.HurtInput{
			Round: activeRound, Tick: combatTick(ctx),
			Actor: combatPlayerRef(event.Attacker), Target: combatPlayerRef(event.Player),
			Weapon:       combatWeaponRef(event.Weapon, combat.SourceCallbackWeapon),
			HealthDamage: event.HealthDamage, HealthDamageTaken: event.HealthDamageTaken,
			ArmorDamage: event.ArmorDamage, ArmorDamageTaken: event.ArmorDamageTaken,
			HealthAfter: event.Health, ArmorAfter: event.Armor,
			Hitgroup: combatHitgroup(event.HitGroup),
		})
	})

	ctx.Parser.RegisterEventHandler(func(event events.Kill) {
		if !acceptAtomicCombatCallback(ctx, activeRound, combat.EventKill) {
			return
		}
		ctx.Combat.RecordKill(combat.KillInput{
			Round: activeRound, Tick: combatTick(ctx),
			Actor: combatPlayerRef(event.Killer), Target: combatPlayerRef(event.Victim),
			Assister:   combatPlayerRef(event.Assister),
			Weapon:     combatWeaponRef(event.Weapon, combat.SourceCallbackWeapon),
			IsHeadshot: event.IsHeadshot, AssistedFlash: event.AssistedFlash,
			PenetratedObjects: event.PenetratedObjects, NoScope: event.NoScope,
			ThroughSmoke: event.ThroughSmoke, AttackerBlind: event.AttackerBlind,
			Distance: float64(event.Distance),
		})
	})

	ctx.Parser.RegisterEventHandler(func(event events.WeaponReload) {
		if !acceptAtomicCombatCallback(ctx, activeRound, combat.EventWeaponReload) {
			return
		}
		weapon := activeCombatWeapon(event.Player)
		ctx.Combat.RecordWeaponReload(combat.ReloadInput{
			Round: activeRound, Tick: combatTick(ctx), Actor: combatPlayerRef(event.Player),
			Weapon: weapon, Ammo: combatAmmo(event.Player, activeEquipment(event.Player)),
		})
	})

	ctx.Parser.RegisterEventHandler(func(event events.ItemEquip) {
		if !acceptAtomicCombatCallback(ctx, activeRound, combat.EventWeaponEquip) {
			return
		}
		ctx.Combat.RecordWeaponEquip(combat.EquipInput{
			Round: activeRound, Tick: combatTick(ctx), Actor: combatPlayerRef(event.Player),
			Weapon: combatWeaponRef(event.Weapon, combat.SourceCallbackWeapon),
			Ammo:   combatAmmo(event.Player, event.Weapon),
		})
	})

	ctx.Parser.RegisterEventHandler(func(event events.RoundEndOfficial) {
		round := closeAtomicCombatRound(&activeRound)
		if round <= 0 || ctx == nil || ctx.Combat == nil {
			return
		}
		ctx.Combat.EndRound(round, combatTick(ctx))
	})
}

func closeAtomicCombatRound(activeRound *int) int {
	if activeRound == nil {
		return 0
	}
	round := *activeRound
	*activeRound = 0
	return round
}

func acceptAtomicCombatCallback(ctx *models.DemoContext, activeRound int, eventType combat.EventType) bool {
	if ctx == nil || ctx.Combat == nil || ctx.Parser == nil {
		return false
	}
	gameState := ctx.Parser.GameState()
	if gameState == nil {
		ctx.Combat.RecordDiscardedCallback(eventType, combat.DiscardInvalidObservation)
		return false
	}
	if gameState.IsWarmupPeriod() {
		ctx.Combat.RecordDiscardedCallback(eventType, combat.DiscardWarmup)
		return false
	}
	if activeRound <= 0 {
		ctx.Combat.RecordDiscardedCallback(eventType, combat.DiscardOutsideRound)
		return false
	}
	return true
}

func combatHitgroup(hitgroup events.HitGroup) string {
	switch hitgroup {
	case events.HitGroupGeneric:
		return "generic"
	case events.HitGroupHead:
		return "head"
	case events.HitGroupChest:
		return "chest"
	case events.HitGroupStomach:
		return "stomach"
	case events.HitGroupLeftArm:
		return "left_arm"
	case events.HitGroupRightArm:
		return "right_arm"
	case events.HitGroupLeftLeg:
		return "left_leg"
	case events.HitGroupRightLeg:
		return "right_leg"
	case events.HitGroupNeck:
		return "neck"
	case events.HitGroupGear:
		return "gear"
	default:
		return fmt.Sprintf("unknown_%d", hitgroup)
	}
}

func combatTick(ctx *models.DemoContext) int {
	if ctx == nil || ctx.Parser == nil || ctx.Parser.GameState() == nil {
		return -1
	}
	return ctx.Parser.GameState().IngameTick()
}

func combatPlayerRef(player *common.Player) combat.PlayerRef {
	if player == nil {
		return combat.PlayerRef{
			Status: combat.AvailabilityUnavailable, Source: combat.SourceUnavailable,
			PositionStatus: combat.AvailabilityUnavailable, PositionSource: combat.SourceUnavailable,
		}
	}
	position := player.Position()
	ref := combat.PlayerRef{
		ID: player.SteamID64, Name: player.Name, Side: getPlayerTeamString(player),
		Status: combat.AvailabilityObserved, Source: combat.SourceCallbackPlayer,
		PositionStatus: combat.AvailabilityUnavailable, PositionSource: combat.SourceUnavailable,
	}
	if ref.ID == 0 {
		ref.Status = combat.AvailabilityUnavailable
	}
	if finite(position.X) && finite(position.Y) && finite(position.Z) {
		ref.Position = &combat.Vector{X: position.X, Y: position.Y, Z: position.Z}
		ref.PositionStatus = combat.AvailabilityObserved
		ref.PositionSource = combat.SourceCallbackPosition
	}
	return ref
}

func combatWeaponRef(weapon *common.Equipment, source string) combat.WeaponRef {
	if weapon == nil || weapon.Type == common.EqUnknown {
		return combat.WeaponRef{Status: combat.AvailabilityUnavailable, Source: source}
	}
	isUtility := weapon.Class() == common.EqClassGrenade
	return combat.WeaponRef{
		Name: weapon.String(), Status: combat.AvailabilityObserved, Source: source, IsUtility: &isUtility,
	}
}

func activeCombatWeapon(player *common.Player) combat.WeaponRef {
	return combatWeaponRef(activeEquipment(player), combat.SourceActiveWeaponAmmo)
}

func activeEquipment(player *common.Player) *common.Equipment {
	if player == nil {
		return nil
	}
	return player.ActiveWeapon()
}

func combatAmmo(player *common.Player, factualWeapon *common.Equipment) combat.AmmoObservation {
	active := activeEquipment(player)
	if active == nil || factualWeapon == nil || active.Type != factualWeapon.Type {
		return combat.AmmoObservation{Status: combat.AvailabilityUnavailable, Source: combat.SourceUnavailable}
	}
	inMagazine, reserve := active.AmmoInMagazine(), active.AmmoReserve()
	return combat.AmmoObservation{
		InMagazine: &inMagazine, Reserve: &reserve,
		Status: combat.AvailabilityObserved, Source: combat.SourceActiveWeaponAmmo,
	}
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
