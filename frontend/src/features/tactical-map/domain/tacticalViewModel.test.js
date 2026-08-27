import { describe, expect, it } from "vitest";
import {
  buildTacticalPresentation,
  getMovementMetricsForSide,
  getMovementSideDistribution,
} from "./tacticalViewModel";

const readyInsight = (type, calloutName, score, confidenceScore = 70) => ({
  type,
  calloutName,
  score,
  confidenceScore,
  confidenceLabel: "medium",
  status: "ready",
});

describe("getMovementMetricsForSide", () => {
  it.each([
    {
      side: "ct",
      expectedSamples: [80, 10],
      expectedPercentages: [80 / 90 * 100, 10 / 90 * 100],
    },
    {
      side: "t",
      expectedSamples: [40, 20],
      expectedPercentages: [40 / 60 * 100, 20 / 60 * 100],
    },
  ])("recalculates $side presence while preserving side weights", ({
    side,
    expectedSamples,
    expectedPercentages,
  }) => {
    const result = getMovementMetricsForSide({
      top_positions: [
        { area: "A", sample_count: 100, ct_percent: 80, time_percent: 50 },
        { area: "B", sample_count: 50, ct_percent: 20, time_percent: 50 },
      ],
    }, side);

    expect(result.top_positions.map((position) => position.sample_count))
      .toEqual(expectedSamples);
    expect(result.top_positions.map((position) => position.time_percent))
      .toEqual(expectedPercentages);
    expect(result.top_positions.reduce(
      (total, position) => total + position.time_percent,
      0,
    )).toBeCloseTo(100);
  });

  it("does not mutate movement metrics or their positions", () => {
    const metrics = {
      sample_count: 150,
      top_positions: [
        { area: "A", sample_count: 100, ct_percent: 80, time_percent: 66.7 },
        { area: "B", sample_count: 50, ct_percent: 20, time_percent: 33.3 },
      ],
    };
    const snapshot = structuredClone(metrics);
    const result = getMovementMetricsForSide(metrics, "ct");

    expect(metrics).toEqual(snapshot);
    expect(result).not.toBe(metrics);
    expect(result.top_positions).not.toBe(metrics.top_positions);
    expect(result.top_positions[0]).not.toBe(metrics.top_positions[0]);
  });

  it("calculates the CT/T distribution from weighted samples", () => {
    expect(getMovementSideDistribution({
      top_positions: [
        { sample_count: 100, ct_percent: 80 },
        { sample_count: 50, ct_percent: 20 },
      ],
    })).toEqual({ ct: 60, t: 40 });
  });
});

describe("buildTacticalPresentation", () => {
  it("adapts evidence into strength, habit and risk in briefing order", () => {
    const result = buildTacticalPresentation({
      tacticalInsights: {
        insights: [
          readyInsight("territory", "Mid", 34.6),
          readyInsight("impact", "A Site", 81.4),
          readyInsight("risk", "B Site", 63.7),
        ],
      },
    });

    expect(result.insights).toHaveLength(3);
    expect(result.insights.map((insight) => ({
      id: insight.id,
      type: insight.type,
      zone: insight.zone,
      value: insight.value,
    }))).toEqual([
      {
        id: "strength:A Site",
        type: "strength",
        zone: "A Site",
        value: "81/100 de impacto",
      },
      {
        id: "habit:Mid",
        type: "habit",
        zone: "Mid",
        value: "35% del tiempo",
      },
      {
        id: "risk:B Site",
        type: "risk",
        zone: "B Site",
        value: "64/100 de riesgo",
      },
    ]);
    expect(result.insights.every((insight) => insight.recommendation)).toBe(true);
    expect(result.insights.every((insight) => 'evidence' in insight)).toBe(true);
  });

  it("omits map signals that do not have finite coordinates", () => {
    const result = buildTacticalPresentation({
      tacticalInsights: {
        callouts: [
          { name: "A Site", position: { x: 0.25, y: 0.75 } },
          { name: "B Site" },
        ],
        insights: [
          readyInsight("impact", "A Site", 80),
          readyInsight("risk", "B Site", 70),
          readyInsight("territory", "Mid", 40),
        ],
      },
      movementMetrics: {
        top_positions: [
          { area: "Mid", position: { x: Number.NaN, y: 0.5 } },
        ],
      },
    });

    expect(result.signals).toHaveLength(1);
    expect(result.signals[0]).toMatchObject({
      id: "strength:A Site",
      name: "A Site",
      position: { x: 0.25, y: 0.75 },
      x: 0.25,
      y: 0.75,
    });
  });

  it("averages confidence from ready insights and ignores pending evidence", () => {
    const result = buildTacticalPresentation({
      tacticalInsights: {
        insights: [
          readyInsight("territory", "Mid", 40, 80),
          readyInsight("impact", "A Site", 70, 55),
          {
            type: "risk",
            status: "insufficient-data",
            calloutName: null,
            score: null,
            confidenceScore: 100,
            confidenceLabel: "high",
          },
        ],
      },
    });
    const empty = buildTacticalPresentation({
      tacticalInsights: { insights: [] },
    });

    expect(result.globalConfidence).toBe(68);
    expect(empty.globalConfidence).toBe(0);
  });
});
