package handlers

import (
	"testing"

	"cs2-demo-service/models"
)

func TestRoundRecordsUseNativeRoundIdentity(t *testing.T) {
	match := &models.MatchData{Rounds: []models.RoundData{{Round: 1}}}
	ensureMatchRound(match, 2)
	ensureMatchRound(match, 2)

	if len(match.Rounds) != 2 || match.Rounds[1].Round != 2 {
		t.Fatalf("round identity must be native and deduplicated: %+v", match.Rounds)
	}

	applyMatchRoundResult(match, 1, "CT", "elimination", 1, 0, true, "A", 123)
	if match.Rounds[0].Winner != "CT" || match.Rounds[0].CTScore != 1 || !match.Rounds[0].BombPlanted {
		t.Fatalf("round 1 was not updated by identity: %+v", match.Rounds[0])
	}
	if match.Rounds[1].Winner != "" {
		t.Fatalf("updating round 1 must not mutate the last round: %+v", match.Rounds[1])
	}
}
