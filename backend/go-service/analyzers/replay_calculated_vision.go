package analyzers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// Publish this calculation's results, never firstSeen/refined history or stale
// LastVisibleEnemies. Positions are captured at this same available tick.
func recordCalculatedVision(ctx *models.DemoContext, tick int, results []visibilityResult, players map[uint64]visibilityPlayerState, available bool) {
	if !ctx.CaptureCalculatedVision {
		return
	}
	if ctx.CalculatedVisionSamples == nil {
		ctx.CalculatedVisionSamples = make(map[int][]models.ReplayVisionSample)
	}
	status := "calculated"
	if !available {
		status = "unavailable"
	}
	sample := models.ReplayVisionSample{Tick: tick, AvailabilityTick: tick, Status: status, Contacts: []models.ReplayCalculatedContact{}}
	side := func(team common.Team) string {
		if team == common.TeamTerrorists {
			return "T"
		}
		if team == common.TeamCounterTerrorists {
			return "CT"
		}
		return ""
	}
	for _, result := range results {
		observer, observerOK := players[result.shooterID]
		subject, subjectOK := players[result.enemyID]
		if !result.isVisible || !observerOK || !subjectOK || !observer.alive || !subject.alive || observer.team == subject.team || side(observer.team) == "" || side(subject.team) == "" {
			continue
		}
		sample.Contacts = append(sample.Contacts, models.ReplayCalculatedContact{
			ObserverID: result.shooterID, SubjectID: result.enemyID,
			ObserverSide: side(observer.team), SubjectSide: side(subject.team),
			X: subject.chest.X, Y: subject.chest.Y, Z: subject.chest.Z - 40,
		})
	}
	ctx.CalculatedVisionSamples[ctx.ActualRoundNumber] = append(ctx.CalculatedVisionSamples[ctx.ActualRoundNumber], sample)
}
