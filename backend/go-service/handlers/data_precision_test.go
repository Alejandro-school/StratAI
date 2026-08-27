package handlers

import (
	"math"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestAnglesToVectorUsesYawThenPitch(t *testing.T) {
	vector := anglesToVector(90, 0)
	if math.Abs(vector.X) > 1e-9 || math.Abs(vector.Y-1) > 1e-9 || math.Abs(vector.Z) > 1e-9 {
		t.Fatalf("unexpected vector for yaw=90 pitch=0: %+v", vector)
	}
}

func TestGrenadeExportRejectsWarmupAndRoundZero(t *testing.T) {
	if shouldRecordGrenadeRound(0, false) {
		t.Fatal("round zero must not be exported")
	}
	if shouldRecordGrenadeRound(1, true) {
		t.Fatal("warmup grenades must not be exported")
	}
	if !shouldRecordGrenadeRound(1, false) {
		t.Fatal("live round grenade was rejected")
	}
}

func TestPendingBulletDamageEnrichesRawCombatEvent(t *testing.T) {
	ctx := &models.DemoContext{
		PendingBulletDamage: []models.BulletDamageSnapshot{{
			Tick: 100, AttackerSteamID: 1, VictimSteamID: 2,
			Distance: 512, DamageDirection: models.AI_Vector{X: 1, Y: 2, Z: 3},
			NumPenetrations: 2, IsNoScope: true, IsAttackerInAir: true,
		}},
	}
	raw := models.RawCombatEvent{Tick: 100, AttackerSteamID: 1, VictimSteamID: 2}

	correlatePendingBulletDamage(ctx, &raw)

	if !raw.HasBulletDamage || !raw.IsWallbang || raw.PenetratedObjects != 2 || raw.BulletDistance != 512 || !raw.NoScope || !raw.AttackerInAir {
		t.Fatalf("raw event was not enriched: %+v", raw)
	}
	if raw.DamageDirection.Z != 3 || len(ctx.PendingBulletDamage) != 0 || ctx.BulletDamageCorrelated != 1 {
		t.Fatalf("unexpected correlation state: raw=%+v ctx=%+v", raw, ctx)
	}
}

func TestRemainingKillDamageExcludesSameTickShotgunPellets(t *testing.T) {
	ctx := &models.DemoContext{RawCombatEvents: []models.RawCombatEvent{
		{Tick: 99, AttackerSteamID: 1, VictimSteamID: 2, Damage: 30},
		{Tick: 100, AttackerSteamID: 1, VictimSteamID: 2, Damage: 18},
		{Tick: 100, AttackerSteamID: 1, VictimSteamID: 2, Damage: 18},
		{Tick: 100, AttackerSteamID: 1, VictimSteamID: 2, Damage: 17},
		{Tick: 100, AttackerSteamID: 1, VictimSteamID: 2, Damage: 17},
		{Tick: 100, AttackerSteamID: 1, VictimSteamID: 2, Damage: 17},
	}}

	if got := remainingKillDamage(ctx, 1, 2, 100, 100); got != 13 {
		t.Fatalf("kill damage = %d; expected remaining 13 HP", got)
	}
}

func TestKillDistanceIsConvertedFromMetersToSourceUnits(t *testing.T) {
	got := killDistanceSourceUnits(10)
	if math.Abs(got-393.7007874015748) > 1e-9 {
		t.Fatalf("10 meters = %f Source units", got)
	}
}

func TestDistancePointToSegmentHandlesZeroLengthSegment(t *testing.T) {
	distance := distancePointToSegment(
		r3.Vector{X: 3, Y: 4},
		r3.Vector{},
		r3.Vector{},
	)
	if distance != 5 {
		t.Fatalf("expected 5, got %f", distance)
	}
}

func TestCombatMetricsExcludeFlashedAndSmokeSamples(t *testing.T) {
	const steamID = uint64(7)
	handler := &PlayerStatsHandler{stats: map[uint64]*models.AI_PlayerStats{
		steamID: {},
	}}
	ctx := &models.DemoContext{MatchData: &models.MatchData{Players: map[uint64]*models.PlayerData{
		steamID: {
			ReactionTimes: []models.ReactionTimeEvent{
				{ReactionTimeMs: 100, TimeToDamage: 120, CrosshairPlacementError: 1, WasFlashed: true},
				{ReactionTimeMs: 200, TimeToDamage: 220, CrosshairPlacementError: 2, SmokeInPath: true},
				{ReactionTimeMs: 300, TimeToDamage: 320, CrosshairPlacementError: 3},
			},
		},
	}}}

	handler.calculateCombatMetrics(ctx)
	stats := handler.stats[steamID]
	if stats.AvgTimeToReaction != 300 || stats.TimeToDamageAvgMS != 320 || stats.CrosshairPlacementAvgError != 3 {
		t.Fatalf("invalid visual samples affected combat metrics: %+v", stats)
	}
}

func TestCombatMetricAvgFieldsUseArithmeticMean(t *testing.T) {
	const steamID = uint64(8)
	handler := &PlayerStatsHandler{stats: map[uint64]*models.AI_PlayerStats{steamID: {}}}
	ctx := &models.DemoContext{MatchData: &models.MatchData{Players: map[uint64]*models.PlayerData{
		steamID: {
			ReactionTimes: []models.ReactionTimeEvent{
				{ReactionTimeMs: 100, TimeToDamage: 120, CrosshairPlacementError: 1},
				{ReactionTimeMs: 200, TimeToDamage: 320, CrosshairPlacementError: 3},
				{ReactionTimeMs: 900, TimeToDamage: 760, CrosshairPlacementError: 8},
				{ReactionTimeMs: 2400, TimeToDamage: 2400, CrosshairPlacementError: 80, SmokeInPath: true},
			},
			Mechanics: &models.MechanicsStats{CounterStrafeValues: []float64{10, 20, 90}},
		},
	}}}

	handler.calculateCombatMetrics(ctx)
	stats := handler.stats[steamID]
	if stats.AvgTimeToReaction != 400 || stats.TimeToDamageAvgMS != 400 || stats.CrosshairPlacementAvgError != 4 {
		t.Fatalf("avg fields do not contain arithmetic means: %+v", stats)
	}
	if stats.AvgCounterStrafeRating != 40 {
		t.Fatalf("counter-strafe avg = %f; expected 40", stats.AvgCounterStrafeRating)
	}
}

func TestLossBonusUsesOfficialLevels(t *testing.T) {
	tests := map[int]int{-1: 1400, 0: 1400, 1: 1900, 2: 2400, 3: 2900, 4: 3400, 5: 3400}
	for level, expected := range tests {
		if actual := calculateLossBonus(level); actual != expected {
			t.Errorf("level %d bonus = %d; expected %d", level, actual, expected)
		}
	}
}

func TestLossBonusResetsAtMR12AndOvertimeHalfStarts(t *testing.T) {
	for _, round := range []int{1, 13, 25, 28, 31} {
		if !isLossBonusResetRound(round) {
			t.Errorf("round %d must reset loss bonus", round)
		}
	}
	for _, round := range []int{2, 12, 14, 24, 26, 27, 29} {
		if isLossBonusResetRound(round) {
			t.Errorf("round %d must not reset loss bonus", round)
		}
	}
}

func TestFirstRoundLoserReceives1900ThenAdvancesTier(t *testing.T) {
	ctx := models.NewDemoContext(nil)
	if ctx.CTConsecutiveLosses != initialLossBonusLevel || ctx.TConsecutiveLosses != initialLossBonusLevel {
		t.Fatalf("new match levels = CT:%d T:%d; expected level 1", ctx.CTConsecutiveLosses, ctx.TConsecutiveLosses)
	}
	if bonus := calculateLossBonus(initialLossBonusLevel); bonus != 1900 {
		t.Fatalf("first-round loss bonus = %d; expected 1900", bonus)
	}
	ctLevel, tLevel := advanceLossBonusLevels(1, 1, common.TeamCounterTerrorists)
	if ctLevel != 0 || tLevel != 2 {
		t.Fatalf("unexpected levels after CT win: CT=%d T=%d", ctLevel, tLevel)
	}
}

func TestWeaponPricesUseCurrentCS2Values(t *testing.T) {
	tests := map[string]int{
		"MP7":       1400,
		"MP5-SD":    1400,
		"PP-Bizon":  1300,
		"MAG-7":     1300,
		"CZ75 Auto": 500,
	}
	for weapon, expected := range tests {
		if actual := getWeaponPrice(weapon); actual != expected {
			t.Errorf("%s price = %d; expected %d", weapon, actual, expected)
		}
	}
}

func TestEconomyMoneySeparatesCalculatedPostBuyFromUnknownNextRoundRewards(t *testing.T) {
	postBuy := calculateMoneyAfterBuy(5000, 4300, 300)
	if postBuy != 1000 {
		t.Fatalf("post-buy money = %d; expected 1000", postBuy)
	}
	if postBuy == postBuy+1900 {
		t.Fatal("next-round money must not be inferred from post-buy money and loss bonus alone")
	}
}

func TestAppendMissingFlashbangsRepresentsStack(t *testing.T) {
	items := []models.AI_WeaponItem{{Weapon: "Flashbang", Price: 200, OriginalOwnerID: 765, OriginalOwnerStatus: "observed"}}
	items = appendMissingFlashbangs(items, 2, 0, "not_observed")
	if len(items) != 2 || equipmentItemsValue(items) != 400 {
		t.Fatalf("flashbang stack was not represented: %+v", items)
	}
	if items[1].OriginalOwnerID != 765 || items[1].OriginalOwnerStatus != "observed" {
		t.Fatalf("stacked flash lost ownership: %+v", items)
	}
}

func TestWeaponPriceDistinguishesUnknownAndKnownZero(t *testing.T) {
	if price, status := getWeaponPriceQuote("C4"); price != 0 || status != "known_zero" {
		t.Fatalf("real zero price was not explicit: price=%d status=%s", price, status)
	}
	if price, status := getWeaponPriceQuote("Unpriced Future Item"); price != 0 || status != "unknown" {
		t.Fatalf("unknown price was silently converted: price=%d status=%s", price, status)
	}
}

func TestEconomyDropWindowUsesDemoTickRate(t *testing.T) {
	if ticks := economyDropMatchWindowTicks(128); ticks != 1280 {
		t.Fatalf("128-tick window = %d; expected 1280", ticks)
	}
	if ticks := economyDropMatchWindowTicks(0); ticks != 640 {
		t.Fatalf("fallback window = %d; expected 640", ticks)
	}
}

func TestEconomyDropMatchingPrefersEntityIdentity(t *testing.T) {
	drop := models.AI_EconomyDrop{Tick: 100, Weapon: "AWP", EntityID: 42}
	if !economyDropMatches(drop, 200, 42, "AWP", 640) {
		t.Fatal("same entity was not matched")
	}
	if economyDropMatches(drop, 200, 99, "AWP", 640) {
		t.Fatal("same-name weapon with a different entity was matched")
	}
	if economyDropMatches(drop, 800, 42, "AWP", 640) {
		t.Fatal("pickup outside the time window was matched")
	}
}
