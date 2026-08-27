package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func registerUtilityImpactHandlers(ctx *models.DemoContext, handler *UtilityHandler) {
	ctx.Parser.RegisterEventHandler(func(event events.PlayerFlashed) {
		if !handler.acceptsCallback(utility.CallbackGroupPlayerFlashed) {
			return
		}
		actor := event.Attacker
		if actor == nil && event.Projectile != nil {
			actor, _ = utilityActor(event.Projectile)
		}
		actorID := uint64(0)
		if actor != nil {
			actorID = actor.SteamID64
		}
		ctx.Utilities.RecordFlash(utility.FlashInput{
			Round: handler.state.round, RuntimeEntityID: projectileEntityID(event.Projectile),
			ActorID: actorID, Tick: ctx.Parser.GameState().IngameTick(), TickRate: getTickRate(ctx),
			Actor:    utilityPlayer(actor, utility.SourcePlayerFlashed),
			Victim:   utilityPlayer(event.Player, utility.SourcePlayerFlashed),
			Relation: utilityRelation(actor, event.Player),
			Duration: observedFlashDuration(event),
		})
	})
	ctx.Parser.RegisterEventHandler(func(event events.PlayerHurt) {
		if event.Weapon == nil {
			return
		}
		typeName := utilityType(event.Weapon)
		if typeName != utility.TypeHE && typeName != utility.TypeMolotov &&
			typeName != utility.TypeIncendiary {
			return
		}
		if !handler.acceptsCallback(utility.CallbackGroupDamage) {
			return
		}
		actorID := uint64(0)
		if event.Attacker != nil {
			actorID = event.Attacker.SteamID64
		}
		victimPosition, victimPositionStatus := observedPlayerPosition(event.Player)
		ctx.Utilities.RecordDamage(utility.DamageInput{
			Round: handler.state.round, Type: typeName, ActorID: actorID,
			Actor: utilityPlayer(event.Attacker, utility.SourcePlayerHurt),
			Tick:  ctx.Parser.GameState().IngameTick(), TickRate: getTickRate(ctx),
			Victim:         utilityPlayer(event.Player, utility.SourcePlayerHurt),
			VictimPosition: victimPosition, VictimPositionStatus: victimPositionStatus,
			Relation:     utilityRelation(event.Attacker, event.Player),
			HealthDamage: max(0, event.HealthDamageTaken),
			ArmorDamage:  max(0, event.ArmorDamageTaken),
			Kill:         event.Player != nil && event.Health == 0,
		})
	})
}

func observedFlashDuration(event events.PlayerFlashed) (observation utility.ScalarObservation) {
	observation = utility.ScalarObservation{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
	}
	if event.Player == nil {
		return observation
	}
	defer func() {
		if recover() != nil {
			observation = utility.ScalarObservation{
				Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
			}
		}
	}()
	duration := event.FlashDuration().Seconds()
	if math.IsNaN(duration) || math.IsInf(duration, 0) || duration < 0 {
		return observation
	}
	return utility.ScalarObservation{
		Value: duration, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed,
	}
}

func observedPlayerPosition(player *common.Player) (utility.Vector, utility.Availability) {
	if player == nil || player.Entity == nil {
		return utility.Vector{}, utility.AvailabilityUnavailable
	}
	position := toUtilityVector(player.Position())
	if !finiteUtilityVector(position) {
		return utility.Vector{}, utility.AvailabilityUnavailable
	}
	return position, utility.AvailabilityObserved
}
