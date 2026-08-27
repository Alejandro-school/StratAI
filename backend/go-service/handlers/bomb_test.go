package handlers

import (
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestBombSiteUsesOnlyKnownNativeSites(t *testing.T) {
	if bombSite(events.BombsiteA) != "A" || bombSite(events.BombsiteB) != "B" {
		t.Fatal("native bombsite was not normalized")
	}
	if bombSite(events.BomsiteUnknown) != "" {
		t.Fatal("unknown native bombsite was fabricated")
	}
}

func TestLegacyBombProjectionKeepsFrontendMarker(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:       6,
		MatchData:          &models.MatchData{BombEvents: []models.BombEvent{}},
		Timeline:           []models.TimelineEvent{},
		CurrentRoundEvents: []models.TimelineEvent{},
	}
	event := objective.Event{
		Type:     objective.EventPlant,
		Round:    6,
		Tick:     900,
		Actor:    objective.Actor{Name: "planter"},
		Site:     "A",
		Position: objective.ObservedPosition(10, 20, 30, objective.SourceDemoinfocsEvent),
	}

	appendLegacyBombMarker(ctx, "plant", event)

	if len(ctx.MatchData.BombEvents) != 1 || len(ctx.Timeline) != 1 {
		t.Fatalf("frontend bomb marker was not projected: match=%+v timeline=%+v", ctx.MatchData.BombEvents, ctx.Timeline)
	}
	marker := ctx.MatchData.BombEvents[0]
	if marker.EventType != "plant" || marker.Round != 6 || marker.Site != "A" || marker.X != 10 {
		t.Fatalf("frontend bomb marker changed contract: %+v", marker)
	}
	if ctx.Timeline[0].Bomb == nil || ctx.Timeline[0].Bomb.EventType != "plant" {
		t.Fatalf("timeline marker changed contract: %+v", ctx.Timeline[0])
	}
}
