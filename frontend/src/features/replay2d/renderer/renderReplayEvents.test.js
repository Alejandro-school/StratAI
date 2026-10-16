import { describe, expect, it, vi } from "vitest";
import { drawShots } from "./renderReplayCombat";
import { drawEffects, isReplayEffectVisible } from "./renderReplayUtility";
import { createViewport } from "./replayViewport";

const config = { pos_x: 0, pos_y: 0, scale: 1 };
const viewport = createViewport(1024, 1024);
const canvas = () => Object.fromEntries([
  "beginPath", "arc", "stroke", "fill", "moveTo", "lineTo", "closePath", "fillText", "createRadialGradient",
].map((method) => [method, vi.fn()]));

describe("native causal event rendering", () => {
  it("draws an atomic shot at its allowed origin without inventing an impact ray", () => {
    const context = canvas();
    drawShots(context, { shots: [{ type: "weapon_fire", actor_team: "CT", weapon: "USP-S", x: 20, y: -30 }] }, config, viewport, 104, 64);
    expect(context.arc).toHaveBeenCalledWith(20, 30, 5, 0, Math.PI * 2);
    expect(context.stroke).toHaveBeenCalledOnce();
    expect(context.moveTo).not.toHaveBeenCalled();
    expect(context.lineTo).not.toHaveBeenCalled();
  });

  it("does not draw a native shot when its allowed position is missing", () => {
    const context = canvas();
    drawShots(context, { shots: [{ type: "weapon_fire", weapon: "USP-S" }] }, config, viewport, 104, 64);
    expect(context.arc).not.toHaveBeenCalled();
    expect(context.stroke).not.toHaveBeenCalled();
  });

  it.each(["smoke", "inferno"])("renders an observed %s beyond the legacy lifetime without a countdown", (type) => {
    const context = canvas();
    const effect = { type, knowledge: "simulated_team_utility_tracking", start_tick: 100,
      observation_tick: 2100, availability_tick: 2100, x: 20, y: -30,
      ...(type === "smoke" ? { radius: 144 } : { hull: [10, -20, 30, -20, 30, -40] }) };
    expect(isReplayEffectVisible(effect, 2099, 64)).toBe(false);
    expect(isReplayEffectVisible(effect, 2100, 64)).toBe(true);
    drawEffects(context, { active_effects: [effect] }, config, viewport, 2100, 64, true);
    expect(context.fill).toHaveBeenCalledOnce();
    expect(context.fillText).not.toHaveBeenCalled();
    expect(context.createRadialGradient).not.toHaveBeenCalled();
    expect(context.stroke).not.toHaveBeenCalled();
    if (type === "smoke") expect(context.arc).toHaveBeenCalledWith(20, 30, 144, 0, Math.PI * 2);
    else expect(context.lineTo).toHaveBeenCalledTimes(2);
  });
});
