package analyzers

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"cs2-demo-service/models"
)

func TestCombatReactionEnrichmentIsIndependentOfHandlerOrder(t *testing.T) {
	tests := []struct {
		name     string
		isKill   bool
		damage   int
		health   int
		armor    int
		hitgroup string
	}{
		{name: "damage", damage: 34, health: 66, armor: 72, hitgroup: "chest"},
		{name: "kill", isKill: true, damage: 66, health: 0, armor: 0, hitgroup: "head"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseEvent := models.RawCombatEvent{
				Round:             1,
				Tick:              100,
				AttackerSteamID:   11,
				VictimSteamID:     22,
				IsKill:            test.isKill,
				Weapon:            "AK-47",
				Hitgroup:          test.hitgroup,
				Damage:            test.damage,
				VictimHealthAfter: test.health,
				VictimArmorAfter:  test.armor,
			}

			reactionFirst := newCombatReactionTestContext()
			finalizeCombatReactionTestState(reactionFirst)
			eventAfterReaction := baseEvent
			applyCombatReactionMetrics(
				&eventAfterReaction,
				ResolveCombatReactionMetrics(reactionFirst, 11, 22, 100, len(reactionFirst.RawCombatEvents)),
			)
			reactionFirst.RawCombatEvents = append(reactionFirst.RawCombatEvents, eventAfterReaction)
			reactionFirst.AI_CombatEvents = append(reactionFirst.AI_CombatEvents, eventAfterReaction)

			captureFirst := newCombatReactionTestContext()
			captureFirst.RawCombatEvents = append(captureFirst.RawCombatEvents, baseEvent)
			captureFirst.AI_CombatEvents = append(captureFirst.AI_CombatEvents, baseEvent)
			finalizeCombatReactionTestState(captureFirst)
			EnrichCapturedCombatReaction(captureFirst, CombatReactionKeyFromRaw(baseEvent))

			if !reflect.DeepEqual(reactionFirst.RawCombatEvents, captureFirst.RawCombatEvents) {
				t.Fatalf("raw events differ by callback order:\nreaction-first: %#v\ncapture-first: %#v", reactionFirst.RawCombatEvents, captureFirst.RawCombatEvents)
			}
			if !reflect.DeepEqual(reactionFirst.AI_CombatEvents, captureFirst.AI_CombatEvents) {
				t.Fatalf("atomic events differ by callback order:\nreaction-first: %#v\ncapture-first: %#v", reactionFirst.AI_CombatEvents, captureFirst.AI_CombatEvents)
			}

			reactionFirstJSON, err := json.Marshal(reactionFirst.AI_CombatEvents)
			if err != nil {
				t.Fatal(err)
			}
			captureFirstJSON, err := json.Marshal(captureFirst.AI_CombatEvents)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(reactionFirstJSON, captureFirstJSON) {
				t.Fatalf("serialized atomic events differ by callback order:\n%s\n%s", reactionFirstJSON, captureFirstJSON)
			}
			if got := captureFirst.AI_CombatEvents[0].TimeToReaction; got != 453 {
				t.Fatalf("reaction time = %v, want 453", got)
			}
			if got := captureFirst.AI_CombatEvents[0].TimeToDamage; got != 453.125 {
				t.Fatalf("time to damage = %v, want 453.125", got)
			}
		})
	}
}

func TestCombatReactionEnrichmentSelectsExactSameTickEvent(t *testing.T) {
	ctx := newCombatReactionTestContext()
	first := models.RawCombatEvent{
		Round:             1,
		Tick:              100,
		AttackerSteamID:   11,
		VictimSteamID:     22,
		Weapon:            "AK-47",
		Hitgroup:          "left_arm",
		Damage:            10,
		VictimHealthAfter: 90,
		VictimArmorAfter:  95,
		CrosshairError:    99,
		PitchError:        98,
		YawError:          97,
		TimeToReaction:    96,
		TimeToDamage:      95,
		FirstSeenTick:     71,
	}
	second := models.RawCombatEvent{
		Round:             1,
		Tick:              100,
		AttackerSteamID:   11,
		VictimSteamID:     22,
		Weapon:            "AK-47",
		Hitgroup:          "chest",
		Damage:            34,
		VictimHealthAfter: 56,
		VictimArmorAfter:  72,
	}
	ctx.RawCombatEvents = []models.RawCombatEvent{first, second}
	ctx.AI_CombatEvents = []models.RawCombatEvent{first, second}
	finalizeCombatReactionTestState(ctx)

	EnrichCapturedCombatReaction(ctx, CombatReactionKeyFromRaw(second))

	if !reflect.DeepEqual(ctx.RawCombatEvents[0], first) || !reflect.DeepEqual(ctx.AI_CombatEvents[0], first) {
		t.Fatal("enrichment modified the other combat fact sharing tick and player IDs")
	}
	if !reflect.DeepEqual(ctx.RawCombatEvents[1], ctx.AI_CombatEvents[1]) {
		t.Fatal("raw and atomic copies of the selected fact diverged")
	}
	if got := ctx.RawCombatEvents[1].CrosshairError; got != 4.5 {
		t.Fatalf("selected crosshair error = %v, want 4.5", got)
	}
	if got := ctx.RawCombatEvents[1].FirstSeenTick; got != 71 {
		t.Fatalf("selected first seen tick = %d, want 71", got)
	}
	if ctx.RawCombatEvents[1].TimeToReaction != 0 || ctx.RawCombatEvents[1].TimeToDamage != 0 {
		t.Fatal("second fact in the same visibility window duplicated first-contact timings")
	}
}

func newCombatReactionTestContext() *models.DemoContext {
	ctx := models.NewDemoContext(nil)
	ctx.ActualRoundNumber = 1
	ctx.CurrentRound = 1
	ctx.MatchData.Players[11] = &models.PlayerData{SteamID: 11}
	return ctx
}

func finalizeCombatReactionTestState(ctx *models.DemoContext) {
	ctx.EnemyFirstSeenTick[11] = map[uint64]models.FirstSeenData{
		22: {
			Tick:                    71,
			LastSeenTick:            100,
			FirstShotTick:           100,
			FirstDamageTick:         100,
			CrosshairPlacementError: 4.5,
			PitchError:              1.25,
			YawError:                4.32,
		},
	}
	ctx.MatchData.Players[11].ReactionTimes = []models.ReactionTimeEvent{
		{
			Round:                   1,
			EnemyID:                 22,
			FirstSeenTick:           71,
			FirstShotTick:           100,
			ReactionTimeMs:          453,
			CrosshairPlacementError: 4.5,
			PitchError:              1.25,
			YawError:                4.32,
			TimeToDamage:            453.125,
		},
	}
}
