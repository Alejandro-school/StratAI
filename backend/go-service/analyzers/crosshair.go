package analyzers

import (
	"cs2-demo-service/models"
)

// RegisterCrosshairAnalyzer registra el analizador de crosshair placement
// NOTA: El tracking básico ya se hace en handlers/player.go
// Este analyzer podría agregar métricas avanzadas en el futuro

func RegisterCrosshairAnalyzer(ctx *models.DemoContext) {
	// El análisis básico ya se hace en handlers/player.go:
	// - TimeAtHeadLevel (pitch -15 a 15)
	// - TimeAtBodyLevel (pitch -30 a -15)
	// - TimeAtGroundLevel (pitch < -30)

	// Aquí podríamos agregar análisis más avanzado:
	// - Pre-aim en ángulos comunes
	// - Crosshair tracking cuando hay enemigo visible
	// - Snap shots vs tracking
	// - Correlación entre crosshair placement y kills

	// Por ahora, el analyzer básico es suficiente
	// TODO: Implementar análisis avanzado si se necesita
}

// calculateCrosshairScore calcula un score de 0-100 basado en estadísticas
func calculateCrosshairScore(stats *models.CrosshairStats) float64 {
	total := stats.TimeAtHeadLevel + stats.TimeAtBodyLevel + stats.TimeAtGroundLevel
	if total == 0 {
		return 0
	}

	// Pesos: Head=100%, Body=50%, Ground=0%
	headPercent := float64(stats.TimeAtHeadLevel) / float64(total)
	bodyPercent := float64(stats.TimeAtBodyLevel) / float64(total)

	score := (headPercent * 100) + (bodyPercent * 50)

	if score > 100 {
		score = 100
	}

	return score
}
