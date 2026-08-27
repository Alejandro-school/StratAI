package handlers

import (
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestCombatHitgroupUsesStableSemanticLabels(t *testing.T) {
	tests := []struct {
		hitgroup events.HitGroup
		expected string
	}{
		{events.HitGroupGeneric, "generic"},
		{events.HitGroupHead, "head"},
		{events.HitGroupChest, "chest"},
		{events.HitGroupStomach, "stomach"},
		{events.HitGroupLeftArm, "left_arm"},
		{events.HitGroupRightArm, "right_arm"},
		{events.HitGroupLeftLeg, "left_leg"},
		{events.HitGroupRightLeg, "right_leg"},
		{events.HitGroupNeck, "neck"},
		{events.HitGroupGear, "gear"},
	}
	for _, test := range tests {
		if actual := combatHitgroup(test.hitgroup); actual != test.expected {
			t.Fatalf("hitgroup %d = %q, want %q", test.hitgroup, actual, test.expected)
		}
	}
	if actual := combatHitgroup(events.HitGroup(9)); actual != "unknown_9" {
		t.Fatalf("unknown hitgroup lost factual code: %q", actual)
	}
}

func TestOfficialRoundClosureClearsLocalScopeBeforeOtherHandlers(t *testing.T) {
	activeRound := 6
	if closed := closeAtomicCombatRound(&activeRound); closed != 6 || activeRound != 0 {
		t.Fatalf("official closure did not atomically clear scope: closed=%d active=%d", closed, activeRound)
	}
	if closed := closeAtomicCombatRound(&activeRound); closed != 0 {
		t.Fatalf("duplicate official closure reopened round: %d", closed)
	}
}
