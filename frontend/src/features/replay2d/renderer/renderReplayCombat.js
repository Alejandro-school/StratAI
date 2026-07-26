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
  context.fillText(bomb.site ? `C4 · ${bomb.site}` : "C4", position.x, position.y - 13);
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
