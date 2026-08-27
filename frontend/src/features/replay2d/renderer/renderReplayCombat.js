import { clamp } from "../domain/replayModel";
import { TEAM_COLORS } from "../domain/replayConfig";
import { worldToScreen } from "./replayViewport";
import { TAU } from "./canvasPrimitives";

const excludedWeapons = /grenade|flash|smoke|molotov|incendiary|decoy|knife|c4/i;

export const drawShots = (context, frame, config, viewport, tick, tickRate) => {
  for (const shot of frame?.shots || []) {
    if (excludedWeapons.test(shot.weapon || "")) continue;
    const from = worldToScreen({ x: shot.from_x, y: shot.from_y }, config, viewport);
    const target = worldToScreen({ x: shot.to_x, y: shot.to_y }, config, viewport);
    const shooter = frame.players?.find((player) => String(player.steam_id) === String(shot.shooter_id));
    const color = TEAM_COLORS[shooter?.team || "T"].solid;
    const deltaX = target.x - from.x;
    const deltaY = target.y - from.y;
    const age = shot.tick ? clamp((tick - shot.tick) / Math.max(1, tickRate * 0.5), 0, 1) : 0;
    context.beginPath();
    context.moveTo(from.x, from.y);
    context.lineTo(from.x + deltaX * 5, from.y + deltaY * 5);
    context.strokeStyle = color;
    context.globalAlpha = 1 - age * 0.72;
    context.lineWidth = shot.hit ? 2 : 1.5;
    context.stroke();
    context.globalAlpha = 1;
  }
};

export const drawBomb = (context, bomb, config, viewport, tick, tickRate) => {
  if (!bomb || !Number.isFinite(bomb.x) || !Number.isFinite(bomb.y)) return;
  if (bomb.state === "carried" || bomb.state === "resolved" || (bomb.plant_tick && tick < bomb.plant_tick)) return;
  const position = worldToScreen(bomb, config, viewport);
  const color = bomb.state === "defused" ? "#69c7ff" : bomb.state === "exploded" ? "#ff665b" : "#ffbd54";
  context.beginPath();
  context.arc(position.x, position.y, bomb.state === "exploded" ? 30 : 9, 0, TAU);
  context.fillStyle = `${color}55`;
  context.fill();
  context.strokeStyle = color;
  context.lineWidth = 2;
  context.stroke();
  context.fillStyle = "#fff";
  context.font = "700 10px system-ui";
  context.textAlign = "center";
  if ((bomb.state === "planted" || bomb.state === "defusing") && bomb.plant_tick) {
    const remaining = Math.max(0, 40 - (tick - bomb.plant_tick) / tickRate);
    context.fillStyle = remaining <= 10 ? "#ff7772" : "#fff";
    context.fillText(`${remaining.toFixed(1)}s`, position.x, position.y + 23);
  }
  if (bomb.state === "defusing") {
    context.beginPath();
    context.arc(position.x, position.y, 14, -Math.PI / 2, Math.PI);
    context.strokeStyle = "#69c7ff";
    context.stroke();
  }
};

export const drawKillEffects = (context, events, tick, tickRate, config, viewport) => {
  for (const event of events || []) {
    const elapsed = (tick - event.tick) / tickRate;
    if (event.type !== "kill" || elapsed < 0 || elapsed > 1.1 || (!event.killer_x && !event.victim_x)) continue;
    const from = worldToScreen({ x: event.killer_x, y: event.killer_y }, config, viewport);
    const to = worldToScreen({ x: event.victim_x, y: event.victim_y }, config, viewport);
    const progress = elapsed / 1.1;
    context.beginPath();
    context.moveTo(from.x, from.y);
    context.lineTo(to.x, to.y);
    context.strokeStyle = `rgba(255,91,91,${1 - progress})`;
    context.lineWidth = 3;
    context.stroke();
    context.beginPath();
    context.arc(to.x, to.y, 8 + progress * 14, 0, TAU);
    context.stroke();
  }
};

export const drawPlayerHitEffects = (context, events, frame, tick, tickRate, config, viewport, reducedMotion) => {
  for (const event of events || []) {
    if (event.type !== "player_hurt" || event.tick > tick) continue;
    const duration = Math.max(0.16, Number(event.durationMs || 320) / 1000);
    const elapsed = (tick - event.tick) / Math.max(1, tickRate);
    if (elapsed > duration) continue;
    const victim = (frame?.players || []).find((player) => String(player.steam_id) === String(event.victim_id));
    const worldPosition = victim || { x: event.victim_x ?? event.x, y: event.victim_y ?? event.y };
    if (!Number.isFinite(Number(worldPosition?.x)) || !Number.isFinite(Number(worldPosition?.y))) continue;
    const position = worldToScreen(worldPosition, config, viewport);
    const progress = clamp(elapsed / duration, 0, 1);
    const blink = reducedMotion ? 1 : 0.45 + Math.max(0, Math.sin(progress * Math.PI * 5)) * 0.55;
    const alpha = (1 - progress) * blink;

    context.beginPath();
    context.arc(position.x, position.y, 10.5 + progress * 5, 0, TAU);
    context.fillStyle = `rgba(255,38,48,${0.58 * alpha})`;
    context.fill();
    context.strokeStyle = `rgba(255,119,119,${0.95 * alpha})`;
    context.lineWidth = 2 - progress * 0.8;
    context.stroke();
  }
};
