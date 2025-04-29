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

// newEventLog con campo "team"
func newEventLog(round int, eventType, details, team string) models.EventLog {
	return models.EventLog{
		Round:     round,
		Timestamp: 0, // Podrías usar parser.GameState().IngameTick() si lo deseas
		EventType: eventType,
		Details:   details,
		Team:      team,
	}
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

// LogTrajectory guarda la trayectoria de la granada en ctx.ProjectileTrajectories
func LogTrajectory(ctx *DemoContext, gp *common.GrenadeProjectile) {
	if gp == nil || gp.Entity == nil {
		return
	}
	entityID := gp.Entity.ID()

	for _, entry := range gp.Trajectory2 {
		ctx.ProjectileTrajectories[entityID] = append(
			ctx.ProjectileTrajectories[entityID],
			models.ProjectileTrajectoryEntry{
				FrameID: entry.FrameID,
				X:       entry.Position.X,
				Y:       entry.Position.Y,
				Z:       entry.Position.Z,
				TimeMS:  entry.Time.Milliseconds(),
			},
		)

	}

	// Evitamos loguear nuevamente en la próxima iteración
	gp.Trajectory2 = nil
}
