package parser

import (
	"fmt"
	"strings"
	"testing"

	"cs2-demo-service/models"
)

func economyV2Fixture() (*models.DemoContext, []models.CanonicalRound, []EconomyCanonicalActor) {
	ctx := economyCandidateFixture()
	ctx.EconomyObservations.ActorAliases = map[uint64]string{}
	probe := models.EconomySnapshot{Phase: "round_start_probe", Tick: 100, CallbackSequence: 1, Actors: []models.EconomyActorSnapshot{}}
	actors := []EconomyCanonicalActor{}
	id := "demo-" + strings.Repeat("a", 20) + ":round:001"
	for i := 1; i <= 10; i++ {
		alias, side, canonicalSide := fmt.Sprintf("p%04d", i), "CT", "ct"
		if i > 5 {
			side, canonicalSide = "T", "t"
		}
		ctx.EconomyObservations.ActorAliases[uint64(i)] = alias
		probe.Actors = append(probe.Actors, models.EconomyActorSnapshot{ActorAlias: alias, Side: side})
		actors = append(actors, EconomyCanonicalActor{RoundID: id, RoundNumber: 1, PlayerID: fmt.Sprintf("steam:%d", i), Side: canonicalSide})
	}
	start, end := 100, 350
	rules := map[string]models.EconomyFact{}
	for key, value := range map[string]bool{"match_started": true, "warmup": false, "freeze": true} {
		rules[key] = models.EconomyFact{Status: "observed", Value: value}
	}
	ctx.EconomyObservations.InitialFrontiers = []models.EconomyInitialFrontier{{Instance: 1, RoundStartProbe: &probe, OfficialEndTick: &end, NativeRules: rules, PreProbeCallbacks: []models.EconomyEarlyCallback{}}}
	return ctx, []models.CanonicalRound{{RoundID: id, RoundNumber: 1, StartTick: &start, EndTick: &end}}, actors
}

func TestEconomyRoundLinkRequiresBoundariesNativeStateAndIdentity(t *testing.T) {
	for _, name := range []string{"valid", "same number other start", "official end missing", "other official end", "duplicate canonical", "wrong identity", "side change", "missing participant", "native false", "native missing"} {
		t.Run(name, func(t *testing.T) {
			ctx, rounds, actors := economyV2Fixture()
			frontier := &ctx.EconomyObservations.InitialFrontiers[0]
			changed := 101
			switch name {
			case "same number other start":
				rounds[0].StartTick = &changed
			case "official end missing":
				frontier.OfficialEndTick = nil
			case "other official end":
				changed = 351
				rounds[0].EndTick = &changed
			case "duplicate canonical":
				rounds = append(rounds, rounds[0])
			case "wrong identity":
				rounds[0].RoundID = "wrong"
			case "side change":
				actors[0].Side = "t"
			case "missing participant":
				actors = actors[1:]
			case "native false":
				frontier.NativeRules["match_started"] = models.EconomyFact{Status: "observed", Value: false}
			case "native missing":
				delete(frontier.NativeRules, "freeze")
			}
			candidate, err := BuildEconomyCandidateV2(ctx, "demo-"+strings.Repeat("a", 20), strings.Repeat("a", 64), "fixture", rounds, actors)
			if err != nil {
				t.Fatal(err)
			}
			if (candidate.InitialFrontiers[0].RoundLink.Status == "verified") != (name == "valid") {
				t.Fatalf("invalid link: %+v", candidate.InitialFrontiers[0].RoundLink)
			}
		})
	}
}

func TestEconomyRepeatedDisplayNumberKeepsOnlyUniqueTemporalInstance(t *testing.T) {
	ctx, canonical, actors := economyV2Fixture()
	second := ctx.EconomyObservations.Rounds[0]
	second.Instance = 2
	ctx.EconomyObservations.Rounds = append(ctx.EconomyObservations.Rounds, second)
	frontier := ctx.EconomyObservations.InitialFrontiers[0]
	frontier.Instance = 2
	ctx.EconomyObservations.InitialFrontiers = append(ctx.EconomyObservations.InitialFrontiers, frontier)
	build := func() EconomyCandidateV2 {
		value, err := BuildEconomyCandidateV2(ctx, "demo-"+strings.Repeat("a", 20), strings.Repeat("a", 64), "fixture", canonical, actors)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	for _, item := range build().InitialFrontiers {
		if item.RoundLink.Reason != "ambiguous_instance" {
			t.Fatal("two instances claimed one canonical round")
		}
	}
	ctx.EconomyObservations.Rounds[0].StartTick = 90
	result := build()
	if result.InitialFrontiers[0].RoundLink.Status != "excluded" || result.InitialFrontiers[1].RoundLink.Status != "verified" {
		t.Fatal("restart was joined by number")
	}
}
