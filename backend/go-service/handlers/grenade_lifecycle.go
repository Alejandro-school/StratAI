package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

const infernoExtinguishRadius = 180.0

func registerUtilityLifecycleHandlers(ctx *models.DemoContext, handler *UtilityHandler) {
	ctx.Parser.RegisterEventHandler(func(event events.FlashExplode) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		ctx.Utilities.RecordDetonation(
			grenadeCallbackHint(ctx, handler.state.round, utility.TypeFlashbang, event.GrenadeEvent),
			utility.SourceFlashExplode,
		)
	})
	ctx.Parser.RegisterEventHandler(func(event events.HeExplode) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		ctx.Utilities.RecordDetonation(
			grenadeCallbackHint(ctx, handler.state.round, utility.TypeHE, event.GrenadeEvent),
			utility.SourceHEExplode,
		)
	})
	ctx.Parser.RegisterEventHandler(func(event events.SmokeStart) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		hint := grenadeCallbackHint(ctx, handler.state.round, utility.TypeSmoke, event.GrenadeEvent)
		throwID, _ := ctx.Utilities.StartEffect(utility.EffectInput{
			Hint: hint, RuntimeEffectEntityID: grenadeEntityID(event.GrenadeEntityID, event.Grenade),
			Source: utility.SourceSmokeStart,
		})
		key := handler.state.smokeEffectKey(event.GrenadeEntityID, event.Grenade, true)
		if key != 0 {
			ctx.ActiveSmokes[key] = event.Position
		}
		if entry, exists := ctx.Utilities.Lookup(throwID); exists {
			ctx.Utilities.MarkExtinguishCandidates(
				throwID,
				handler.state.round,
				hint.Tick,
				hint.Position,
				infernoExtinguishRadius,
				entry.Lifecycle.Correlation,
			)
		}
	})
	ctx.Parser.RegisterEventHandler(func(event events.SmokeExpired) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		ctx.Utilities.ExpireEffect(utility.EffectInput{
			Hint:                  grenadeCallbackHint(ctx, handler.state.round, utility.TypeSmoke, event.GrenadeEvent),
			RuntimeEffectEntityID: grenadeEntityID(event.GrenadeEntityID, event.Grenade),
			Source:                utility.SourceSmokeExpired,
		}, getTickRate(ctx))
		key := handler.state.smokeEffectKey(event.GrenadeEntityID, event.Grenade, false)
		if key != 0 {
			delete(ctx.ActiveSmokes, key)
			delete(handler.state.equipmentEffects, event.Grenade)
		}
	})
	ctx.Parser.RegisterEventHandler(func(event events.InfernoStart) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		if event.Inferno == nil {
			handler.invalidCallback(utility.CallbackGroupLifecycle)
			return
		}
		position, status := deterministicInfernoCenter(event.Inferno)
		hint := effectCallbackHint(
			ctx, handler.state.round, utility.TypeUnknown, event.Inferno.Thrower(), position, status,
		)
		ctx.Utilities.StartEffect(utility.EffectInput{
			Hint: hint, RuntimeEffectEntityID: infernoEntityID(event.Inferno),
			Source: utility.SourceInfernoStart,
		})
	})
	ctx.Parser.RegisterEventHandler(func(event events.InfernoExpired) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		if event.Inferno == nil {
			handler.invalidCallback(utility.CallbackGroupLifecycle)
			return
		}
		position, status := deterministicInfernoCenter(event.Inferno)
		ctx.Utilities.ExpireEffect(utility.EffectInput{
			Hint: effectCallbackHint(
				ctx, handler.state.round, utility.TypeUnknown, event.Inferno.Thrower(), position, status,
			),
			RuntimeEffectEntityID: infernoEntityID(event.Inferno),
			Source:                utility.SourceInfernoExpired,
		}, getTickRate(ctx))
	})
	ctx.Parser.RegisterEventHandler(func(event events.DecoyStart) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		ctx.Utilities.StartEffect(utility.EffectInput{
			Hint:                  grenadeCallbackHint(ctx, handler.state.round, utility.TypeDecoy, event.GrenadeEvent),
			RuntimeEffectEntityID: grenadeEntityID(event.GrenadeEntityID, event.Grenade),
			Source:                utility.SourceDecoyStart,
		})
	})
	ctx.Parser.RegisterEventHandler(func(event events.DecoyExpired) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		ctx.Utilities.ExpireEffect(utility.EffectInput{
			Hint:                  grenadeCallbackHint(ctx, handler.state.round, utility.TypeDecoy, event.GrenadeEvent),
			RuntimeEffectEntityID: grenadeEntityID(event.GrenadeEntityID, event.Grenade),
			Source:                utility.SourceDecoyExpired,
		}, getTickRate(ctx))
	})
}

func grenadeCallbackHint(
	ctx *models.DemoContext,
	round int,
	typeName utility.Type,
	event events.GrenadeEvent,
) utility.CallbackHint {
	position := toUtilityVector(event.Position)
	status := utility.AvailabilityUnavailable
	if finiteUtilityVector(position) {
		status = utility.AvailabilityObserved
	}
	actorID := uint64(0)
	actor := utility.PlayerRef{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
	if event.Thrower != nil {
		actorID = event.Thrower.SteamID64
		actor = utilityPlayer(event.Thrower, utility.SourceCallbackActor)
	}
	entityID := grenadeEntityID(event.GrenadeEntityID, event.Grenade)
	entitySource := utility.SourceUnavailable
	if entityID != 0 {
		entitySource = utility.SourceGrenadeEntityID
	}
	return utility.CallbackHint{
		Round: round, RuntimeEntityID: entityID, EntitySource: entitySource,
		Type: typeName, ActorID: actorID, Actor: actor,
		Tick: ctx.Parser.GameState().IngameTick(), TickRate: getTickRate(ctx),
		Position: position, PositionStatus: status,
		Area: observePositionArea(ctx, position),
	}
}

func effectCallbackHint(
	ctx *models.DemoContext,
	round int,
	typeName utility.Type,
	actor *common.Player,
	position utility.Vector,
	status utility.Availability,
) utility.CallbackHint {
	area := unavailableArea()
	if status == utility.AvailabilityObserved {
		area = observePositionArea(ctx, position)
	}
	actorID := uint64(0)
	actorObservation := utility.PlayerRef{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
	if actor != nil {
		actorID = actor.SteamID64
		actorObservation = utilityPlayer(actor, utility.SourceCallbackActor)
	}
	return utility.CallbackHint{
		Round: round, Type: typeName, TypeFamily: utility.TypeFamilyFire,
		ActorID: actorID, Actor: actorObservation,
		Tick: ctx.Parser.GameState().IngameTick(), TickRate: getTickRate(ctx),
		Position: position, PositionStatus: status, Area: area,
	}
}
