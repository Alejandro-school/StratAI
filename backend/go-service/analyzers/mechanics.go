package analyzers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

const (
	counterStrafeLookbackTicks = 5
	velocityHistoryCapacity    = 16
)

type horizontalVelocitySample struct {
	round     int
	tick      int
	speed     float64
	available bool
}

type horizontalVelocityHistory map[uint64][]horizontalVelocitySample

// RegisterMechanicsAnalyzer registra el analizador de mecánicas avanzadas.
func RegisterMechanicsAnalyzer(ctx *models.DemoContext) {
	velocityHistory := make(horizontalVelocityHistory)

	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		if ctx.ActualRoundNumber <= 0 {
			return
		}
		currentTick := ctx.Parser.GameState().IngameTick()
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			if player == nil || player.SteamID64 == 0 || !player.IsAlive() {
				continue
			}
			velocityHistory.record(player.SteamID64, observeHorizontalVelocity(ctx, player, currentTick))
		}
	})

	ctx.Parser.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil || e.Weapon == nil {
			return
		}
		if !isCounterStrafeRifle(e.Weapon.Type) || e.Shooter.IsDucking() || ctx.ActualRoundNumber <= 0 {
			return
		}

		shooterID := e.Shooter.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()
		current := observeHorizontalVelocity(ctx, e.Shooter, currentTick)
		velocityHistory.record(shooterID, current)
		if !current.available {
			return
		}

		previous, available := velocityHistory.at(
			shooterID,
			ctx.ActualRoundNumber,
			currentTick-counterStrafeLookbackTicks,
		)
		if !available {
			return
		}

		weaponName := e.Weapon.String()
		accuracyThreshold := models.GetAccuracyThreshold(weaponName)
		if previous <= accuracyThreshold {
			return
		}

		rating := calculateCounterStrafeRating(
			current.speed,
			accuracyThreshold,
			models.GetWeaponMaxSpeed(weaponName),
		)
		recordMechanicStat(ctx, shooterID, "counter_strafe", rating)
		ctx.LastShotMechanics[shooterID] = &models.ShotMechanics{
			CounterStrafeRating: rating,
			Tick:                currentTick,
		}
	})
}

func observeHorizontalVelocity(ctx *models.DemoContext, player *common.Player, tick int) horizontalVelocitySample {
	estimate := ctx.PlayerMotion.ObservePlayer(
		player,
		ctx.ActualRoundNumber,
		tick,
		ctx.Parser.TickRate(),
	)
	return horizontalVelocitySample{
		round:     ctx.ActualRoundNumber,
		tick:      tick,
		speed:     estimate.HorizontalSpeed(),
		available: estimate.Available,
	}
}

func (history horizontalVelocityHistory) record(playerID uint64, sample horizontalVelocitySample) {
	samples := history[playerID]
	if len(samples) > 0 && samples[len(samples)-1].tick == sample.tick {
		samples[len(samples)-1] = sample
		history[playerID] = samples
		return
	}
	samples = append(samples, sample)
	if len(samples) > velocityHistoryCapacity {
		samples = samples[len(samples)-velocityHistoryCapacity:]
	}
	history[playerID] = samples
}

func (history horizontalVelocityHistory) at(playerID uint64, round, tick int) (float64, bool) {
	samples := history[playerID]
	for index := len(samples) - 1; index >= 0; index-- {
		sample := samples[index]
		if sample.tick < tick {
			return 0, false
		}
		if sample.tick == tick {
			return sample.speed, sample.round == round && sample.available
		}
	}
	return 0, false
}

func isCounterStrafeRifle(weaponType common.EquipmentType) bool {
	return weaponType == common.EqAK47 || weaponType == common.EqM4A4 ||
		weaponType == common.EqM4A1 || weaponType == common.EqGalil ||
		weaponType == common.EqFamas || weaponType == common.EqSG556 || weaponType == common.EqAUG
}

func calculateCounterStrafeRating(currentSpeed, accuracyThreshold, weaponMaxSpeed float64) float64 {
	if currentSpeed <= accuracyThreshold {
		return 100
	}
	if currentSpeed >= weaponMaxSpeed || weaponMaxSpeed <= accuracyThreshold {
		return 0
	}
	return 100 * (1 - (currentSpeed-accuracyThreshold)/(weaponMaxSpeed-accuracyThreshold))
}

// Helper para registrar stats
func recordMechanicStat(ctx *models.DemoContext, steamID uint64, statType string, value float64) {
	// Asegurar que existe el jugador en MatchData
	if ctx.MatchData.Players[steamID] == nil {
		return // O inicializarlo
	}

	player := ctx.MatchData.Players[steamID]
	if player.Mechanics == nil {
		player.Mechanics = &models.MechanicsStats{
			CounterStrafeValues: make([]float64, 0, 100), // Pre-allocate for efficiency
		}
	}

	// Accumulate values for MEDIAN calculation at end of match
	switch statType {
	case "counter_strafe":
		player.Mechanics.CounterStrafeValues = append(player.Mechanics.CounterStrafeValues, value)
	}
}
