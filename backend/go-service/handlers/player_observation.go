package handlers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type activeWeaponState struct {
	CurrentWeapon   *string
	Status          string
	LastObservation *models.ActiveWeaponObservation
}

func RegisterPlayerObservationHandler(ctx *models.DemoContext) {
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		gameState := ctx.Parser.GameState()
		tick := gameState.IngameTick()
		for _, player := range gameState.Participants().Playing() {
			if player == nil || player.SteamID64 == 0 {
				continue
			}
			ctx.PlayerMotion.ObservePlayer(
				player,
				ctx.ActualRoundNumber,
				tick,
				ctx.Parser.TickRate(),
			)
			cacheActiveWeapon(ctx, player, tick)
		}
	})
}

func observeActiveWeapon(
	ctx *models.DemoContext,
	player *common.Player,
	tick int,
) activeWeaponState {
	if player == nil || player.SteamID64 == 0 {
		return activeWeaponState{Status: models.ActiveWeaponStatusUnavailable}
	}

	currentWeapon := ""
	if player.IsAlive() {
		if activeWeapon := player.ActiveWeapon(); activeWeapon != nil {
			currentWeapon = activeWeapon.String()
		}
	}
	if currentWeapon != "" {
		cacheActiveWeaponName(ctx, player.SteamID64, currentWeapon, tick)
	}

	lastObservation, hasLastObservation := ctx.LastActiveWeapon[player.SteamID64]
	var last *models.ActiveWeaponObservation
	if hasLastObservation &&
		lastObservation.RoundNumber == ctx.ActualRoundNumber &&
		lastObservation.Tick <= tick {
		copy := lastObservation
		last = &copy
	}
	return resolveActiveWeaponState(player.IsAlive(), currentWeapon, last)
}

func resolveActiveWeaponState(
	isAlive bool,
	currentWeapon string,
	lastObservation *models.ActiveWeaponObservation,
) activeWeaponState {
	state := activeWeaponState{LastObservation: lastObservation}
	if !isAlive {
		state.Status = models.ActiveWeaponStatusNotApplicable
		return state
	}
	if currentWeapon == "" {
		state.Status = models.ActiveWeaponStatusUnavailable
		return state
	}
	weapon := currentWeapon
	state.CurrentWeapon = &weapon
	state.Status = models.ActiveWeaponStatusObserved
	return state
}

func cacheActiveWeapon(ctx *models.DemoContext, player *common.Player, tick int) {
	if player == nil || !player.IsAlive() || player.SteamID64 == 0 {
		return
	}
	activeWeapon := player.ActiveWeapon()
	if activeWeapon == nil || activeWeapon.String() == "" {
		return
	}
	cacheActiveWeaponName(ctx, player.SteamID64, activeWeapon.String(), tick)
}

func cacheActiveWeaponName(
	ctx *models.DemoContext,
	playerID uint64,
	weapon string,
	tick int,
) {
	if playerID == 0 || weapon == "" || ctx.ActualRoundNumber <= 0 {
		return
	}
	ctx.LastActiveWeapon[playerID] = models.ActiveWeaponObservation{
		Weapon:      weapon,
		Tick:        tick,
		RoundNumber: ctx.ActualRoundNumber,
	}
}
