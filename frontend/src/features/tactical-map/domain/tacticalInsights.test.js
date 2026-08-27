import { describe, expect, it } from "vitest";
import { buildTacticalInsights } from "./tacticalInsights";

const callout = (name, kills, deaths, context = {}, flashDeathPct = 0) => ({
  name,
  kills,
  deaths,
  sample_size: kills + deaths,
  win_rate: kills + deaths ? kills / (kills + deaths) * 100 : 50,
  context_stats: context,
  flash_death_pct: flashDeathPct,
});

describe("buildTacticalInsights", () => {
  it("smooths raw win rates towards the weighted map baseline", () => {
    const input = [
      callout("One duel", 1, 0),
      callout("Stable", 9, 9),
    ];
    const snapshot = structuredClone(input);
    const result = buildTacticalInsights({ callouts: input, matchesAnalyzed: 18 });

    expect(result.baselineWinRate).toBe(52.6);
    expect(result.callouts[0].win_rate).toBe(100);
    expect(result.callouts[0].adjustedWinRate).toBe(57.9);
    expect(result.callouts[1].adjustedWinRate).toBe(50.8);
    expect(input).toEqual(snapshot);
  });

  it("uses both demos and duel volume to express confidence", () => {
    const data = [callout("A", 12, 8)];
    const oneDemo = buildTacticalInsights({ callouts: data, matchesAnalyzed: 1 });
    const eighteenDemos = buildTacticalInsights({ callouts: data, matchesAnalyzed: 18 });

    expect(oneDemo.callouts[0].confidenceLabel).toBe("low");
    expect(eighteenDemos.callouts[0].confidenceLabel).toBe("high");
    expect(eighteenDemos.callouts[0].confidenceScore)
      .toBeGreaterThan(oneDemo.callouts[0].confidenceScore);
  });

  it("keeps the real duel volume visible as a percentage of the map", () => {
    const result = buildTacticalInsights({
      callouts: [callout("A", 6, 4), callout("B", 2, 3), callout("C", 1, 4)],
      matchesAnalyzed: 18,
    });

    expect(result.totalDuels).toBe(20);
    expect(result.callouts.map((item) => item.volumeShare)).toEqual([50, 25, 25]);
    expect(result.callouts.reduce((sum, item) => sum + item.volumeShare, 0)).toBe(100);
  });

  it("uses opening, trade and flash context to separate impact from risk", () => {
    const result = buildTacticalInsights({
      callouts: [
        callout("Strong context", 5, 5, {
          opening_kills: 4,
          opening_attempts: 4,
          trade_kills: 4,
          trade_deaths: 0,
        }),
        callout("Leaky context", 5, 5, {
          opening_kills: 0,
          opening_attempts: 4,
          trade_kills: 0,
          trade_deaths: 4,
        }, 40),
      ],
      matchesAnalyzed: 18,
    });
    const [strong, leaky] = result.callouts;

    expect(strong.impactScore).toBeGreaterThan(leaky.impactScore);
    expect(leaky.riskScore).toBeGreaterThan(strong.riskScore);
    expect(leaky.tacticalEvidence).toMatchObject({
      openingAttempts: 4,
      tradeDeaths: 4,
      flashDeathPct: 40,
    });
  });

  it("returns the three stable briefing slots with real evidence", () => {
    const result = buildTacticalInsights({
      callouts: [
        callout("Impact", 12, 4, { opening_kills: 3, opening_attempts: 4 }),
        callout("Risk", 3, 12, { opening_kills: 0, opening_attempts: 4 }, 35),
      ],
      matchesAnalyzed: 18,
      movementMetrics: {
        top_positions: [
          { area: "Mid", time_percent: 30, sample_count: 900 },
          { area: "B", time_percent: 45, sample_count: 800 },
        ],
      },
    });

    expect(result.insights.map((insight) => insight.type))
      .toEqual(["territory", "impact", "risk"]);
    expect(result.insights[0]).toMatchObject({
      status: "ready",
      calloutName: "B",
      score: 45,
      evidence: { timePercent: 45, sampleCount: 800, matchesAnalyzed: 18 },
    });
    expect(result.insights[1].calloutName).toBe("Impact");
    expect(result.insights[2].calloutName).toBe("Risk");
    expect(result.insights[1].evidence.sampleSize).toBe(16);
  });

  it("marks missing briefing evidence instead of inventing conclusions", () => {
    const result = buildTacticalInsights();

    expect(result).toMatchObject({
      baselineWinRate: 50,
      totalDuels: 0,
      callouts: [],
    });
    expect(result.insights).toHaveLength(3);
    expect(result.insights.every((insight) =>
      insight.status === "insufficient-data" && insight.score === null)).toBe(true);
  });

  it("handles incomplete numeric payloads without producing NaN", () => {
    const result = buildTacticalInsights({
      callouts: [{
        name: "Unknown",
        sample_size: "5",
        win_rate: "60",
        context_stats: { opening_attempts: "bad", flash_deaths: 2 },
      }],
      matchesAnalyzed: "3",
    });
    const enriched = result.callouts[0];

    expect(enriched.adjustedWinRate).toBe(60);
    expect(enriched.tacticalEvidence.flashDeathPct).toBe(0);
    expect([
      enriched.confidenceScore,
      enriched.volumeShare,
      enriched.impactScore,
      enriched.riskScore,
    ].every(Number.isFinite)).toBe(true);
  });
});
