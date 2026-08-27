import { describe, expect, it } from "vitest";
import { fitIconDimensions, isReplayEffectVisible, projectileHasDetonated } from "./renderReplayUtility";

describe("fitIconDimensions", () => {
  it("fits official icons inside a constant 20px box without distortion", () => {
    expect(fitIconDimensions(80, 20)).toEqual({ width: 20, height: 5 });
    expect(fitIconDimensions(20, 40)).toEqual({ width: 10, height: 20 });
    expect(fitIconDimensions(0, 0)).toEqual({ width: 20, height: 20 });
  });
});

describe("projectileHasDetonated", () => {
  const he = { type: "he", x: 100, y: 200 };
  const explosion = { type: "utility_detonate", subtype: "he", tick: 640, x: 105, y: 198 };

  it("removes a grenade icon as soon as its nearby detonation is emitted", () => {
    expect(projectileHasDetonated(he, [explosion], 640, 64)).toBe(true);
    expect(projectileHasDetonated(he, [explosion], 960, 64)).toBe(true);
  });

  it("does not remove a different or still-flying grenade", () => {
    expect(projectileHasDetonated(he, [explosion], 639, 64)).toBe(false);
    expect(projectileHasDetonated({ ...he, x: 900 }, [explosion], 640, 64)).toBe(false);
    expect(projectileHasDetonated(he, [{ ...explosion, subtype: "flashbang" }], 640, 64)).toBe(false);
  });
});

describe("isReplayEffectVisible", () => {
  it("removes smoke as soon as its timer reaches zero", () => {
    const smoke = { type: "smoke", start_tick: 100 };
    expect(isReplayEffectVisible(smoke, 100 + 18 * 64 - 1, 64)).toBe(true);
    expect(isReplayEffectVisible(smoke, 100 + 18 * 64, 64)).toBe(false);
  });
});
