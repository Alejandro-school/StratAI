package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
	"math"
	"slices"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func registerUtilityThrowHandlers(ctx *models.DemoContext, handler *UtilityHandler) {
	ctx.Parser.RegisterEventHandler(func(event events.GrenadeProjectileThrow) {
		if !handler.acceptsCallback(utility.CallbackGroupThrows) {
			return
		}
		if event.Projectile == nil {
			handler.invalidCallback(utility.CallbackGroupThrows)
			return
		}
		ctx.Utilities.RecordThrow(buildUtilityThrowInput(ctx, handler.state.round, event.Projectile))
	})
	ctx.Parser.RegisterEventHandler(func(event events.GrenadeProjectileBounce) {
		if !handler.acceptsCallback(utility.CallbackGroupBounces) {
			return
		}
		if event.Projectile == nil {
			handler.invalidCallback(utility.CallbackGroupBounces)
			return
		}
		position, positionStatus := observedProjectilePosition(event.Projectile)
		ctx.Utilities.RecordBounce(
			handler.state.round,
			projectileEntityID(event.Projectile),
			ctx.Parser.GameState().IngameTick(),
			position,
			positionStatus,
			event.BounceNr,
		)
	})
	ctx.Parser.RegisterEventHandler(func(events.FrameDone) {
		warmup, available := handler.warmupStatus()
		if !available || !handler.state.records(warmup) {
			return
		}
		for _, projectile := range ctx.Parser.GameState().GrenadeProjectiles() {
			position, observed := projectilePosition(projectile)
			if !observed {
				continue
			}
			ctx.Utilities.RecordTrajectoryPosition(
				handler.state.round,
				projectileEntityID(projectile),
				ctx.Parser.GameState().IngameTick(),
				position,
			)
		}
	})
	ctx.Parser.RegisterEventHandler(func(event events.GrenadeProjectileDestroy) {
		if !handler.acceptsCallback(utility.CallbackGroupLifecycle) {
			return
		}
		if event.Projectile == nil {
			handler.invalidCallback(utility.CallbackGroupLifecycle)
			return
		}
		recordFinalTrajectory(ctx, handler.state.round, event.Projectile)
		position, positionStatus := observedProjectilePosition(event.Projectile)
		ctx.Utilities.RecordDestroy(utility.CallbackHint{
			Round:           handler.state.round,
			RuntimeEntityID: projectileEntityID(event.Projectile),
			EntitySource:    utility.SourceProjectileEntity,
			Type:            utilityType(event.Projectile.WeaponInstance),
			ActorID:         utilityActorID(event.Projectile),
			Tick:            ctx.Parser.GameState().IngameTick(),
			TickRate:        getTickRate(ctx),
			Position:        position,
			PositionStatus:  positionStatus,
		})
	})
}

func buildUtilityThrowInput(
	ctx *models.DemoContext,
	round int,
	projectile *common.GrenadeProjectile,
) utility.ThrowInput {
	tick := ctx.Parser.GameState().IngameTick()
	actor, actorSource := utilityActor(projectile)
	position, positionStatus := observedProjectilePosition(projectile)
	return utility.ThrowInput{
		Round:           round,
		RuntimeEntityID: projectileEntityID(projectile),
		EntitySource:    utility.SourceProjectileEntity,
		Type:            utilityType(projectile.WeaponInstance),
		TypeSource:      utilityTypeSource(projectile.WeaponInstance),
		Actor:           utilityPlayer(actor, actorSource),
		Launch: utility.ThrowSnapshot{
			Tick: utility.TickObservation{
				Tick: tick, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
			},
			Position:                  observedVector(position, positionStatus, utility.SourceProjectilePosition),
			View:                      observeThrowView(actor),
			ThrowerVelocity:           observeThrowerVelocity(ctx, actor, round, tick),
			ProjectileInitialVelocity: observeProjectileVelocity(projectile, tick),
			Stance:                    observeThrowStance(actor),
			Area:                      observePlayerArea(ctx, actor),
		},
	}
}

func recordFinalTrajectory(
	ctx *models.DemoContext,
	round int,
	projectile *common.GrenadeProjectile,
) {
	trajectory := slices.Clone(projectile.Trajectory)
	slices.SortFunc(trajectory, func(left, right common.TrajectoryEntry) int {
		if left.Tick != right.Tick {
			return left.Tick - right.Tick
		}
		if left.FrameID != right.FrameID {
			return left.FrameID - right.FrameID
		}
		return compareUtilityVector(toUtilityVector(left.Position), toUtilityVector(right.Position))
	})
	for _, sample := range trajectory {
		position := toUtilityVector(sample.Position)
		if !finiteUtilityVector(position) {
			continue
		}
		ctx.Utilities.RecordTrajectoryPosition(
			round,
			projectileEntityID(projectile),
			sample.Tick,
			position,
		)
	}
}

func projectileEntityID(projectile *common.GrenadeProjectile) int {
	if projectile == nil || projectile.Entity == nil {
		return 0
	}
	return projectile.Entity.ID()
}

func utilityActor(projectile *common.GrenadeProjectile) (*common.Player, string) {
	if projectile == nil {
		return nil, utility.SourceUnavailable
	}
	if projectile.Thrower != nil {
		return projectile.Thrower, utility.SourceProjectileThrower
	}
	if projectile.Owner != nil {
		return projectile.Owner, utility.SourceProjectileOwner
	}
	return nil, utility.SourceUnavailable
}

func utilityActorID(projectile *common.GrenadeProjectile) uint64 {
	actor, _ := utilityActor(projectile)
	if actor == nil {
		return 0
	}
	return actor.SteamID64
}

func utilityType(equipment *common.Equipment) utility.Type {
	if equipment == nil {
		return utility.TypeUnknown
	}
	switch equipment.Type {
	case common.EqFlash:
		return utility.TypeFlashbang
	case common.EqSmoke:
		return utility.TypeSmoke
	case common.EqHE:
		return utility.TypeHE
	case common.EqMolotov:
		return utility.TypeMolotov
	case common.EqIncendiary:
		return utility.TypeIncendiary
	case common.EqDecoy:
		return utility.TypeDecoy
	default:
		return utility.TypeUnknown
	}
}

func utilityTypeSource(equipment *common.Equipment) string {
	if equipment == nil {
		return utility.SourceUnavailable
	}
	return utility.SourceWeaponInstance
}

func projectilePosition(projectile *common.GrenadeProjectile) (utility.Vector, bool) {
	if projectile == nil || projectile.Entity == nil {
		return utility.Vector{}, false
	}
	position := toUtilityVector(projectile.Position())
	return position, finiteUtilityVector(position)
}

func observedProjectilePosition(projectile *common.GrenadeProjectile) (utility.Vector, utility.Availability) {
	position, observed := projectilePosition(projectile)
	if !observed {
		return utility.Vector{}, utility.AvailabilityUnavailable
	}
	return position, utility.AvailabilityObserved
}

func observeProjectileVelocity(
	projectile *common.GrenadeProjectile,
	tick int,
) (observation utility.VelocityObservation) {
	observation.Status = utility.AvailabilityUnavailable
	observation.Source = utility.SourceUnavailable
	if projectile == nil || projectile.Entity == nil {
		return observation
	}
	defer func() {
		if recover() != nil {
			observation = utility.VelocityObservation{
				Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
			}
		}
	}()
	vector := toUtilityVector(projectile.Velocity())
	if !finiteUtilityVector(vector) {
		return observation
	}
	return utility.VelocityObservation{
		Vector: vector, HorizontalSpeed: math.Hypot(vector.X, vector.Y),
		ObservedTick: tick, MeasurementWindowTicks: 0,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	}
}

func compareUtilityVector(left, right utility.Vector) int {
	for _, pair := range [][2]float64{{left.X, right.X}, {left.Y, right.Y}, {left.Z, right.Z}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}
