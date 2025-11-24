package demo

import (
	"crypto/sha1"
	"cs2-demo-service/models"
	"encoding/hex"
	"io"
	"math"
	"os"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
)

// vectorDistance calcula la distancia euclídea entre dos vectores
func vectorDistance(v1, v2 r3.Vector) float64 {
	return math.Sqrt(math.Pow(v1.X-v2.X, 2) +
		math.Pow(v1.Y-v2.Y, 2) +
		math.Pow(v1.Z-v2.Z, 2))
}

// getAlivePlayersCount calcula jugadores vivos por equipo - FUNCIÓN OPTIMIZADA
func getAlivePlayersCount(ctx *DemoContext) (tAlive, ctAlive int) {
	for _, p := range ctx.parser.GameState().Participants().Playing() {
		if p.Team == common.TeamTerrorists && p.IsAlive() {
			tAlive++
		} else if p.Team == common.TeamCounterTerrorists && p.IsAlive() {
			ctAlive++
		}
	}
	return tAlive, ctAlive
}

// ComputeFileHash calcula el hash SHA1
func ComputeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	return "match_" + hash[:10], nil
}

// nameOrNil devuelve el nombre del *common.Player o <nil> si es nulo.
func nameOrNil(p *common.Player) string {
	if p == nil {
		return "<nil>"
	}
	return p.Name
}

// newEventLog con timestamp del tick actual - OPTIMIZADO PARA IA
func newEventLog(ctx *DemoContext, eventType, details, team string) models.EventLog {
	tick := 0
	if ctx != nil && ctx.parser != nil && ctx.parser.GameState() != nil {
		tick = ctx.parser.GameState().IngameTick()
	}
	return models.EventLog{
		Round:     ctx.RoundNumber,
		Timestamp: tick,
		EventType: eventType,
		Details:   details,
		Team:      team,
	}
}

// newEventLogWithTimestamp - DEPRECADO: usar newEventLog directamente
func newEventLogWithTimestamp(ctx *DemoContext, eventType, details, team string) models.EventLog {
	return newEventLog(ctx, eventType, details, team)
}

// teamString devuelve "T" o "CT" o "" según el equipo
func teamString(p *common.Player) string {
	if p == nil {
		return ""
	}
	switch p.Team {
	case common.TeamTerrorists:
		return "T"
	case common.TeamCounterTerrorists:
		return "CT"
	default:
		return ""
	}
}

// GetRoundContext calcula el contexto de la ronda actual (halftime, OT, etc.)
func GetRoundContext(roundNumber int) models.RoundContext {
	context := models.RoundContext{
		RoundNumber: roundNumber,
	}

	// Determinar fase del juego (MR12 format: 24 rondas regulares)
	if roundNumber <= 12 {
		context.IsFirstHalf = true
		context.RoundsUntilHalftime = 12 - roundNumber
		context.IsLastBeforeSwitch = (roundNumber == 12)
	} else if roundNumber <= 24 {
		context.IsSecondHalf = true
		context.RoundsUntilHalftime = 24 - roundNumber
		context.IsLastBeforeSwitch = (roundNumber == 24)

		// Reset de economía en ronda 13 (cambio de lado)
		if roundNumber == 13 {
			context.WasEconomyReset = true
		}
	} else {
		// Overtime (BO6, 4 para ganar)
		context.IsOvertime = true

		// Calcular qué overtime es
		overtimeRound := roundNumber - 24
		context.OvertimeNumber = ((overtimeRound - 1) / 6) + 1

		// Posición dentro del OT actual
		posInOT := ((overtimeRound - 1) % 6) + 1

		// Cambio de lado en ronda 3 de cada OT (25+3=28, 31+3=34, etc.)
		if posInOT == 4 {
			context.WasEconomyReset = true
		}

		context.RoundsUntilHalftime = 6 - posInOT
		context.IsLastBeforeSwitch = (posInOT == 3 || posInOT == 6)
	}

	return context
}
