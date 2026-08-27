package handlers

import (
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestUtilityStatsAreDerivedFromLedger(t *testing.T) {
	tracker := utility.NewTracker()
	actor := utility.PlayerRef{
		ID: 7, Name: "thrower", Side: "T",
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrower,
	}
	flashID, _ := tracker.RecordThrow(utility.ThrowInput{
		Round: 1, RuntimeEntityID: 11, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, TypeSource: utility.SourceWeaponInstance, Actor: actor,
		Launch: utility.ThrowSnapshot{Tick: utility.TickObservation{
			Tick: 100, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
		}},
	})
	_ = flashID
	for _, effect := range []utility.FlashInput{
		{Round: 1, RuntimeEntityID: 11, Actor: actor, Tick: 110,
			Victim:   utility.PlayerRef{ID: 8, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed},
			Relation: utility.RelationEnemy,
			Duration: utility.ScalarObservation{Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed}},
		{Round: 1, RuntimeEntityID: 11, Actor: actor, Tick: 111,
			Victim:   utility.PlayerRef{ID: 9, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed},
			Relation: utility.RelationTeammate,
			Duration: utility.ScalarObservation{Value: 1.25, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed}},
		{Round: 1, RuntimeEntityID: 11, Actor: actor, Tick: 112,
			Victim: actor, Relation: utility.RelationSelf,
			Duration: utility.ScalarObservation{Value: 0.5, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed}},
	} {
		tracker.RecordFlash(effect)
	}
	tracker.RecordThrow(utility.ThrowInput{
		Round: 1, RuntimeEntityID: 12, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeDecoy, TypeSource: utility.SourceWeaponInstance, Actor: actor,
		Launch: utility.ThrowSnapshot{Tick: utility.TickObservation{
			Tick: 120, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
		}},
	})
	tracker.RecordThrow(utility.ThrowInput{
		Round: 1, RuntimeEntityID: 13, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeIncendiary, TypeSource: utility.SourceWeaponInstance, Actor: actor,
		Launch: utility.ThrowSnapshot{Tick: utility.TickObservation{
			Tick: 130, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
		}},
	})
	tracker.RecordThrow(utility.ThrowInput{
		Round: 1, RuntimeEntityID: 14, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeHE, TypeSource: utility.SourceWeaponInstance, Actor: actor,
		Launch: utility.ThrowSnapshot{Tick: utility.TickObservation{
			Tick: 140, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
		}},
	})
	tracker.RecordDetonation(utility.CallbackHint{
		Round: 1, RuntimeEntityID: 14, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeHE, ActorID: actor.ID, Tick: 145, TickRate: 64,
	}, utility.SourceHEExplode)
	tracker.RecordDamage(utility.DamageInput{
		Round: 1, Type: utility.TypeHE, ActorID: actor.ID, Actor: actor, Tick: 146, TickRate: 64,
		Victim: utility.PlayerRef{
			ID: 10, Side: "CT", Status: utility.AvailabilityObserved, Source: utility.SourcePlayerHurt,
		},
		Relation: utility.RelationEnemy, HealthDamage: 24,
	})

	handler := NewPlayerStatsHandler()
	handler.utilities = tracker
	handler.stats[actor.ID] = &models.AI_PlayerStats{
		SteamID: "7", UtilityDamage: 73,
		NativeScoreboard: models.AI_NativePlayerStats{UtilityDamage: 73},
	}
	handler.applyUtilityStats()
	stats := handler.stats[actor.ID]
	if stats == nil || stats.GrenadesThrownTotal != 4 || stats.FlashesThrown != 1 || stats.HEThrown != 1 ||
		stats.DecoysThrown != 1 || stats.IncendiariesThrown != 1 || stats.MolotovsThrown != 1 {
		t.Fatalf("throw counts do not reconcile with ledger: %+v", stats)
	}
	if stats.EnemiesFlashedTotal != 1 || stats.TeammatesFlashedTotal != 1 || stats.SelfFlashesTotal != 1 ||
		stats.EnemyFlashDurationTotalMS != 0 || stats.TeammateFlashDurationTotalMS != 1250 ||
		stats.SelfFlashDurationTotalMS != 500 || stats.FlashDurationTotal != 0 {
		t.Fatalf("flash relations or durations do not reconcile: %+v", stats)
	}
	if stats.UtilityDamage != 73 || stats.NativeScoreboard.UtilityDamage != 73 ||
		stats.UtilityDamageObserved != 24 || stats.GrenadeDamage["he"] != 24 {
		t.Fatalf("native and ledger-derived utility damage were conflated: %+v", stats)
	}
	handler.applyUtilityStats()
	if stats.UtilityDamage != 73 || stats.UtilityDamageObserved != 24 || stats.GrenadeDamage["he"] != 24 {
		t.Fatalf("ledger-derived utility damage is not idempotent: %+v", stats)
	}
}

func TestUtilityProjectionPrecedesNativeScoreboardForLedgerOnlyActor(t *testing.T) {
	tracker := utility.NewTracker()
	actor := utility.PlayerRef{
		ID: 17, Name: "ledger-only", Side: "CT",
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrower,
	}
	tracker.RecordThrow(utility.ThrowInput{
		Round: 1, RuntimeEntityID: 171, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, TypeSource: utility.SourceWeaponInstance, Actor: actor,
		Launch: utility.ThrowSnapshot{Tick: utility.TickObservation{
			Tick: 100, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
		}},
	})

	handler := NewPlayerStatsHandler()
	handler.utilities = tracker
	native := models.AI_NativePlayerStats{
		Kills: 9, Deaths: 4, Assists: 3, TotalDamage: 912, UtilityDamage: 73,
	}
	for iteration := 0; iteration < 2; iteration++ {
		handler.applyUtilityStats()
		if handler.stats[actor.ID] == nil {
			t.Fatal("utility projection did not create the ledger-only actor")
		}
		handler.applyNativePlayerStats(actor.ID, native)
		result := handler.finalizeStats()
		if len(result) != 1 || result[0].GrenadesThrownTotal != 1 || result[0].FlashesThrown != 1 ||
			result[0].Kills != native.Kills || result[0].Deaths != native.Deaths ||
			result[0].TotalDamage != native.TotalDamage || result[0].UtilityDamage != native.UtilityDamage ||
			result[0].NativeScoreboard != native {
			t.Fatalf("utility/native projection order drifted on iteration %d: %+v", iteration, result)
		}
	}
}

func TestCalculateFinalStatsUsesPlayerRounds(t *testing.T) {
	h := NewPlayerStatsHandler()
	player := &models.AI_PlayerStats{SteamID: "7", TotalDamage: 100, WeaponStats: map[string]models.AI_WeaponStat{}}
	h.currentRound = 23
	h.ctRoundsPlayed[7] = 1

	h.calculateFinalStats(player)

	if player.RoundsPlayed != 1 || player.ADR != 100 {
		t.Fatalf("expected one player-round and ADR 100, got rounds=%d ADR=%v", player.RoundsPlayed, player.ADR)
	}
}

func TestFinalizeStatsExcludesAnonymousIdentity(t *testing.T) {
	h := NewPlayerStatsHandler()
	h.stats[0] = &models.AI_PlayerStats{SteamID: "0", Name: "anonymous"}
	h.stats[7] = &models.AI_PlayerStats{SteamID: "7", Name: "observed"}

	result := h.finalizeStats()
	if len(result) != 1 || result[0].SteamID != "7" {
		t.Fatalf("anonymous identity leaked into final stats: %+v", result)
	}
}

func TestRoundEndDoesNotCreditAbsentPlayers(t *testing.T) {
	h := NewPlayerStatsHandler()
	h.stats[1] = &models.AI_PlayerStats{}
	h.stats[2] = &models.AI_PlayerStats{}
	h.roundPlayers[1] = common.TeamCounterTerrorists

	h.HandleRoundEnd(events.RoundEnd{}, nil)

	if h.stats[1].RoundsSurvived != 1 {
		t.Fatal("participating survivor was not credited")
	}
	if h.stats[2].RoundsSurvived != 0 || h.stats[2].KAST != 0 {
		t.Fatalf("absent player received round credit: %+v", h.stats[2])
	}
}

func TestFriendlyDamageIsSeparatedFromPerformanceDamage(t *testing.T) {
	h := NewPlayerStatsHandler()
	attacker := &common.Player{SteamID64: 1, Name: "attacker", Team: common.TeamTerrorists}
	victim := &common.Player{SteamID64: 2, Name: "victim", Team: common.TeamTerrorists}

	h.HandleDamage(events.PlayerHurt{Attacker: attacker, Player: victim, HealthDamageTaken: 35})

	stats := h.stats[1]
	if stats.FriendlyDamage != 35 || stats.TotalDamage != 0 || stats.UtilityDamage != 0 {
		t.Fatalf("friendly damage contaminated performance totals: %+v", stats)
	}
}

func TestShotgunDamageAndAccuracyUseShotLevelSemantics(t *testing.T) {
	tests := []struct {
		name       string
		weaponType common.EquipmentType
	}{
		{name: "MAG-7", weaponType: common.EqMag7},
		{name: "XM1014", weaponType: common.EqXM1014},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := NewPlayerStatsHandler()
			h.eventTick = func() int { return 100 }
			h.ctRoundsPlayed[1] = 1
			attacker := &common.Player{SteamID64: 1, Name: "attacker", Team: common.TeamCounterTerrorists}
			victim := &common.Player{SteamID64: 2, Name: "victim", Team: common.TeamTerrorists}
			weapon := &common.Equipment{Type: test.weaponType}

			h.HandleWeaponFire(events.WeaponFire{Shooter: attacker, Weapon: weapon})
			for _, hit := range []struct {
				health int
				damage int
			}{
				{health: 82, damage: 18},
				{health: 64, damage: 18},
				{health: 47, damage: 17},
				{health: 30, damage: 17},
				{health: 13, damage: 17},
				{health: 0, damage: 100},
			} {
				h.HandleDamage(events.PlayerHurt{
					Attacker: attacker, Player: victim, Weapon: weapon,
					Health: hit.health, HealthDamageTaken: hit.damage,
				})
			}

			stats := h.stats[attacker.SteamID64]
			h.calculateFinalStats(stats)
			weaponStats := stats.WeaponStats[test.name]
			if stats.TotalDamage != 100 || weaponStats.Damage != 100 {
				t.Fatalf("shotgun damage was duplicated: total=%d weapon=%d", stats.TotalDamage, weaponStats.Damage)
			}
			if stats.ShotsFired != 1 || stats.ShotsHit != 1 || stats.AccuracyOverall != 100 {
				t.Fatalf("expected one damaging shot, got fired=%d hit=%d accuracy=%f", stats.ShotsFired, stats.ShotsHit, stats.AccuracyOverall)
			}
			if weaponStats.ShotsFired != 1 || weaponStats.ShotsHit != 1 || weaponStats.Accuracy != 100 {
				t.Fatalf("unexpected weapon accuracy: %+v", weaponStats)
			}
		})
	}
}

func TestCombatMetricsDoNotClassifyUnknownVelocityAsHold(t *testing.T) {
	h := NewPlayerStatsHandler()
	h.stats[1] = &models.AI_PlayerStats{}
	ctx := &models.DemoContext{MatchData: &models.MatchData{Players: map[uint64]*models.PlayerData{
		1: {
			ReactionTimes: []models.ReactionTimeEvent{
				{CrosshairPlacementError: 10, ShooterVelocity: 0, ShooterVelocityAvailable: false},
				{CrosshairPlacementError: 30, ShooterVelocity: 0, ShooterVelocityAvailable: true},
			},
		},
	}}}

	h.calculateCombatMetrics(ctx)

	if h.stats[1].CrosshairPlacementAvgError != 20 {
		t.Fatalf("unexpected overall crosshair average: %v", h.stats[1].CrosshairPlacementAvgError)
	}
	if h.stats[1].CrosshairPlacementHold != 30 {
		t.Fatalf("unknown velocity contaminated hold metric: %v", h.stats[1].CrosshairPlacementHold)
	}
}
