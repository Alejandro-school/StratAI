package handlers

import (
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
)

func TestLegacyUtilityProjectionIsIdempotentAndLedgerDerived(t *testing.T) {
	tracker := utility.NewTracker()
	actor := utility.PlayerRef{
		ID: 1, Name: "thrower", Side: "T",
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrower,
	}
	flashID, _ := tracker.RecordThrow(projectionThrowInput(1, 10, utility.TypeFlashbang, actor, 100))
	tracker.RecordDetonation(utility.CallbackHint{
		Round: 1, RuntimeEntityID: 10, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, ActorID: 1, Tick: 110,
		Position: utility.Vector{}, PositionStatus: utility.AvailabilityObserved,
	}, utility.SourceFlashExplode)
	tracker.RecordFlash(utility.FlashInput{
		Round: 1, RuntimeEntityID: 10, Actor: actor, Tick: 110,
		Victim: actor, Relation: utility.RelationSelf,
		Duration: utility.ScalarObservation{
			Value: 0.5, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed,
		},
	})
	heID, _ := tracker.RecordThrow(projectionThrowInput(1, 11, utility.TypeHE, actor, 120))
	tracker.RecordDetonation(utility.CallbackHint{
		Round: 1, RuntimeEntityID: 11, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeHE, ActorID: 1, Tick: 130,
		Position: utility.Vector{X: 20}, PositionStatus: utility.AvailabilityObserved,
	}, utility.SourceHEExplode)
	tracker.RecordDamage(utility.DamageInput{
		Round: 1, Type: utility.TypeHE, ActorID: 1, Actor: actor, Tick: 130,
		Victim: utility.PlayerRef{
			ID: 2, Name: "enemy", Side: "CT", Status: utility.AvailabilityObserved,
			Source: utility.SourcePlayerHurt,
		},
		Relation: utility.RelationEnemy, ArmorDamage: 5,
	})
	tracker.StartEffect(utility.EffectInput{
		Hint: utility.CallbackHint{
			Round: 1, Type: utility.TypeUnknown, TypeFamily: utility.TypeFamilyFire, Tick: 140,
			PositionStatus: utility.AvailabilityUnavailable,
		},
		Source: utility.SourceInfernoStart,
	})

	ctx := &models.DemoContext{
		Utilities: tracker, MatchData: &models.MatchData{}, CurrentRound: 1,
		AI_GrenadeEvents: []models.AI_GrenadeEvent{{Type: "stale"}},
	}
	refreshLegacyUtilityProjection(ctx)
	refreshLegacyUtilityProjection(ctx)

	if len(ctx.AI_GrenadeEvents) != 3 || len(ctx.MatchData.Flashes) != 1 ||
		len(ctx.MatchData.HEGrenades) != 1 || len(ctx.Timeline) != 2 {
		t.Fatalf("projection was incomplete or duplicated: events=%+v match=%+v timeline=%+v",
			ctx.AI_GrenadeEvents, ctx.MatchData, ctx.Timeline)
	}
	flash := legacyGrenadeByTick(t, ctx.AI_GrenadeEvents, 100)
	if flashID == "" || len(flash.BlindedPlayers) != 1 || !flash.BlindedPlayers[0].IsSelf ||
		flash.AlliesBlinded != 1 {
		t.Fatalf("self flash relation was lost: %+v", flash)
	}
	he := legacyGrenadeByTick(t, ctx.AI_GrenadeEvents, 120)
	if heID == "" || he.ArmorDamageDealt != 5 || he.EnemyArmorDamage != 5 ||
		he.EnemiesDamaged != 1 || len(he.DamagedPlayers) != 1 || he.DamagedPlayers[0].ArmorDamage != 5 {
		t.Fatalf("armor-only damage was not projected: %+v", he)
	}
	if ctx.AI_GrenadeEvents[2].Type != "Unknown" || ctx.AI_GrenadeEvents[2].TickThrow != 0 {
		t.Fatalf("sparse inferno fabricated launch/type facts: %+v", ctx.AI_GrenadeEvents[2])
	}
}

func projectionThrowInput(
	round int,
	entityID int,
	typeName utility.Type,
	actor utility.PlayerRef,
	tick int,
) utility.ThrowInput {
	return utility.ThrowInput{
		Round: round, RuntimeEntityID: entityID, EntitySource: utility.SourceProjectileEntity,
		Type: typeName, TypeSource: utility.SourceWeaponInstance, Actor: actor,
		Launch: utility.ThrowSnapshot{
			Tick: utility.TickObservation{
				Tick: tick, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
			},
		},
	}
}

func legacyGrenadeByTick(
	t *testing.T,
	events []models.AI_GrenadeEvent,
	tick int,
) models.AI_GrenadeEvent {
	t.Helper()
	for _, event := range events {
		if event.TickThrow == tick {
			return event
		}
	}
	t.Fatalf("legacy grenade at tick %d not found", tick)
	return models.AI_GrenadeEvent{}
}
