import { activeEventsAtTick, effectProgress, effectSecondsRemaining } from "../domain/replayModel";
import { UTILITY_COLORS } from "../domain/replayConfig";
import { worldToScreen } from "./replayViewport";
import { pathWorld, TAU } from "./canvasPrimitives";

export const fitIconDimensions = (width, height, maxSize = 20) => {
  if (!width || !height) return { width: maxSize, height: maxSize };
  const scale = Math.min(maxSize / width, maxSize / height);
  return { width: width * scale, height: height * scale };
};

const drawGrenadeIcon = (context, projectile, position, icon) => {
  context.save();
  context.translate(position.x, position.y);
  context.shadowColor = UTILITY_COLORS[projectile.type] || "#fff";
  context.shadowBlur = 6;
  if (icon?.complete && icon.naturalWidth > 0) {
    const size = fitIconDimensions(icon.naturalWidth, icon.naturalHeight);
    context.drawImage(icon, -size.width / 2, -size.height / 2, size.width, size.height);
  } else {
    context.beginPath();
    context.arc(0, 0, 4, 0, TAU);
    context.fillStyle = UTILITY_COLORS[projectile.type] || "#fff";
    context.fill();
    context.strokeStyle = "#fff";
    context.lineWidth = 1;
    context.stroke();
  }
  context.restore();
};

export const drawProjectiles = (context, frame, config, viewport, projectileIcons = {}) => {
  for (const projectile of frame?.projectiles || []) {
    const trajectory = projectile.trajectory || [];
    const points = [];
    for (let index = 0; index < trajectory.length; index += 2) {
      points.push({ x: trajectory[index], y: trajectory[index + 1] });
    }
    if (points.length > 1) {
      context.beginPath();
      pathWorld(context, points, config, viewport);
      context.strokeStyle = UTILITY_COLORS[projectile.type] || "#fff";
      context.globalAlpha = 0.72;
      context.lineWidth = 1.35;
      context.setLineDash([6, 5]);
      context.stroke();
      context.setLineDash([]);
      context.globalAlpha = 1;
    }
    const position = worldToScreen(projectile, config, viewport);
    drawGrenadeIcon(context, projectile, position, projectileIcons[projectile.type]);
  }
};

const drawSmoke = (context, effect, config, viewport, tick, tickRate, reducedMotion) => {
  const position = worldToScreen(effect, config, viewport);
  const radius = Math.max(18, ((effect.radius || 144) / config.scale) * viewport.baseScale * viewport.zoom);
  const remaining = effectSecondsRemaining(effect, tick, tickRate);
  const life = Math.max(0, Math.min(1, remaining / 18));
  const deployed = Math.min(1, (18 - remaining) / 0.7);
  const breath = reducedMotion ? 1 : 0.98 + Math.sin(life * 18) * 0.02;
  const gradient = context.createRadialGradient(position.x, position.y, radius * 0.1, position.x, position.y, radius * breath);
  gradient.addColorStop(0, `rgba(179,188,202,${0.64 * deployed})`);
  gradient.addColorStop(0.7, `rgba(106,116,131,${0.52 * deployed})`);
  gradient.addColorStop(1, "rgba(66,75,90,0)");
  context.beginPath();
  context.arc(position.x, position.y, radius * breath, 0, TAU);
  context.fillStyle = gradient;
  context.fill();
  context.beginPath();
  context.arc(position.x, position.y, radius + 3, -Math.PI / 2, -Math.PI / 2 + TAU * life);
  context.strokeStyle = "rgba(225,235,248,.82)";
  context.lineWidth = 2;
  context.stroke();
  context.fillStyle = "#e6eef8";
  context.font = "650 10px system-ui";
  context.textAlign = "center";
  context.fillText(`${Math.ceil(remaining)}s`, position.x, position.y + 3);
};

const drawInferno = (context, effect, config, viewport, tick, tickRate, reducedMotion) => {
  const hull = [];
  for (let index = 0; index < (effect.hull || []).length; index += 2) {
    hull.push({ x: effect.hull[index], y: effect.hull[index + 1] });
  }
  if (hull.length < 3) {
    const center = worldToScreen(effect, config, viewport);
    context.beginPath();
    context.arc(center.x, center.y, 30, 0, TAU);
  } else {
    context.beginPath();
    pathWorld(context, hull, config, viewport);
    context.closePath();
  }
  const remaining = effectSecondsRemaining(effect, tick, tickRate);
  const flicker = reducedMotion ? 0.72 : 0.66 + Math.sin(remaining * 11) * 0.08;
  context.fillStyle = `rgba(255,94,22,${flicker})`;
  context.fill();
  context.strokeStyle = "rgba(255,205,87,.92)";
  context.lineWidth = 2;
  context.stroke();
};

export const drawEffects = (context, frame, config, viewport, tick, tickRate, reducedMotion) => {
  for (const effect of frame?.active_effects || []) {
    if (effect.type === "smoke") drawSmoke(context, effect, config, viewport, tick, tickRate, reducedMotion);
    if (effect.type === "inferno") drawInferno(context, effect, config, viewport, tick, tickRate, reducedMotion);
  }
};

export const drawEventEffects = (context, events, tick, tickRate, config, viewport, reducedMotion) => {
  for (const event of activeEventsAtTick(events, tick, tickRate)) {
    if (event.type !== "utility_detonate") continue;
    const position = worldToScreen(event, config, viewport);
    const progress = effectProgress(tick, event, tickRate);
    if (event.subtype === "he") {
      const radius = 10 + progress * 56;
      const gradient = context.createRadialGradient(position.x, position.y, 0, position.x, position.y, radius);
      gradient.addColorStop(0, `rgba(255,242,200,${0.9 * (1 - progress)})`);
      gradient.addColorStop(0.24, `rgba(255,126,52,${0.72 * (1 - progress)})`);
      gradient.addColorStop(1, "rgba(255,65,36,0)");
      context.beginPath();
      context.arc(position.x, position.y, radius, 0, TAU);
      context.fillStyle = gradient;
      context.fill();
      context.beginPath();
      context.arc(position.x, position.y, 13 + progress * 50, 0, TAU);
      context.strokeStyle = `rgba(255,184,104,${1 - progress})`;
      context.lineWidth = 3 - progress * 2;
      context.stroke();
      if (event.damage > 0) {
        context.fillStyle = `rgba(255,255,255,${1 - progress})`;
        context.font = "750 11px system-ui";
        context.textAlign = "center";
        context.fillText(`-${event.damage}`, position.x, position.y - 13);
      }
    }
    if (event.subtype === "flashbang") {
      const radius = reducedMotion ? 20 : 16 + progress * 34;
      const gradient = context.createRadialGradient(position.x, position.y, 0, position.x, position.y, radius);
      gradient.addColorStop(0, `rgba(255,255,238,${0.9 * (1 - progress)})`);
      gradient.addColorStop(1, "rgba(255,246,180,0)");
      context.beginPath();
      context.arc(position.x, position.y, radius, 0, TAU);
      context.fillStyle = gradient;
      context.fill();
    }
  }
};

export const getHeShake = (events, tick, tickRate, reducedMotion) => {
  if (reducedMotion) return { x: 0, y: 0 };
  const explosion = [...(events || [])].reverse().find((event) => (
    event.type === "utility_detonate"
    && event.subtype === "he"
    && event.tick <= tick
    && tick - event.tick <= tickRate * 0.28
  ));
  if (!explosion) return { x: 0, y: 0 };
  const progress = (tick - explosion.tick) / (tickRate * 0.28);
  const amplitude = Math.pow(1 - progress, 2) * 5;
  const phase = progress * Math.PI * 9;
  return { x: Math.sin(phase) * amplitude, y: Math.cos(phase * 1.17) * amplitude * 0.68 };
};
