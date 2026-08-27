import { activeEventsAtTick, clamp, effectProgress, effectSecondsRemaining } from "../domain/replayModel";
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

const utilityType = (value) => {
  const type = String(value || "").toLowerCase();
  if (type === "flash") return "flashbang";
  if (type === "molotov" || type === "incendiary") return "inferno";
  return type;
};

export const projectileHasDetonated = (projectile, events, tick, _tickRate = 64) => {
  const type = utilityType(projectile?.type);
  return (events || []).some((event) => {
    if (event.type !== "utility_detonate" || utilityType(event.subtype) !== type) return false;
    if (event.tick > tick) return false;
    return Math.hypot(Number(event.x) - Number(projectile.x), Number(event.y) - Number(projectile.y)) <= 96;
  });
};

export const drawProjectiles = (
  context, frame, config, viewport, projectileIcons = {}, events = [], tick = 0, tickRate = 64,
) => {
  for (const projectile of frame?.projectiles || []) {
    // The sampled frame immediately after a detonation can still contain the
    // projectile. The event position lets us hide only the grenade that has
    // actually exploded instead of flashing a stale HE icon over the blast.
    if (projectileHasDetonated(projectile, events, tick, tickRate)) continue;
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

export const isReplayEffectVisible = (effect, tick, tickRate = 64) => (
  effectSecondsRemaining(effect, tick, tickRate) > 0
);

const drawSmoke = (context, effect, config, viewport, tick, tickRate, reducedMotion) => {
  const position = worldToScreen(effect, config, viewport);
  const radius = Math.max(18, ((effect.radius || 144) / config.scale) * viewport.baseScale * viewport.zoom);
  const remaining = effectSecondsRemaining(effect, tick, tickRate);
  if (remaining <= 0) return;
  const life = Math.max(0, Math.min(1, remaining / 18));
  const deployed = Math.min(1, (18 - remaining) / 0.7);
  const fadeOut = Math.min(1, remaining / 0.85);
  const opacity = deployed * fadeOut;
  const breath = reducedMotion ? 1 : 0.98 + Math.sin(life * 18) * 0.02;
  const gradient = context.createRadialGradient(position.x, position.y, radius * 0.1, position.x, position.y, radius * breath);
  gradient.addColorStop(0, `rgba(179,188,202,${0.64 * opacity})`);
  gradient.addColorStop(0.7, `rgba(106,116,131,${0.52 * opacity})`);
  gradient.addColorStop(1, "rgba(66,75,90,0)");
  context.beginPath();
  context.arc(position.x, position.y, radius * breath, 0, TAU);
  context.fillStyle = gradient;
  context.fill();
  context.beginPath();
  context.arc(position.x, position.y, radius + 3, -Math.PI / 2, -Math.PI / 2 + TAU * life);
  context.strokeStyle = `rgba(225,235,248,${0.82 * fadeOut})`;
  context.lineWidth = 2;
  context.stroke();
  context.fillStyle = `rgba(230,238,248,${fadeOut})`;
  context.font = "650 10px system-ui";
  context.textAlign = "center";
  context.fillText(`${Math.ceil(remaining)}s`, position.x, position.y + 3);
};

const drawFlameCell = (context, point, index, elapsed, visibility, reducedMotion, scale) => {
  const seed = Math.sin((index + 1) * 91.73 + point.x * 0.017 + point.y * 0.013) * 43758.5453;
  const random = seed - Math.floor(seed);
  const phase = reducedMotion ? random * TAU : elapsed * (8 + random * 4) + random * TAU;
  const pulse = reducedMotion ? 1 : 0.88 + Math.sin(phase) * 0.12;
  const lean = reducedMotion ? 0 : Math.sin(phase * 0.73) * 1.8 * scale;
  const radius = (4.5 + random * 2.5) * scale * pulse;

  context.save();
  context.translate(point.x + lean, point.y);
  context.rotate((random - 0.5) * 0.55);
  const glow = context.createRadialGradient(0, 0, 0, 0, 0, radius * 2.1);
  glow.addColorStop(0, `rgba(255,190,38,${0.58 * visibility})`);
  glow.addColorStop(0.45, `rgba(255,70,8,${0.35 * visibility})`);
  glow.addColorStop(1, "rgba(115,16,2,0)");
  context.beginPath();
  context.arc(0, 0, radius * 2.1, 0, TAU);
  context.fillStyle = glow;
  context.fill();

  context.beginPath();
  context.moveTo(lean * 0.25, -radius * 1.75);
  context.bezierCurveTo(radius * 0.95, -radius * 0.65, radius * 0.88, radius * 0.72, 0, radius);
  context.bezierCurveTo(-radius * 0.88, radius * 0.72, -radius * 0.9, -radius * 0.48, lean * 0.25, -radius * 1.75);
  const outer = context.createLinearGradient(0, -radius * 1.75, 0, radius);
  outer.addColorStop(0, `rgba(255,232,105,${0.9 * visibility})`);
  outer.addColorStop(0.48, `rgba(255,112,14,${0.96 * visibility})`);
  outer.addColorStop(1, `rgba(164,25,3,${0.82 * visibility})`);
  context.fillStyle = outer;
  context.fill();

  context.beginPath();
  context.moveTo(0, -radius * 0.72);
  context.bezierCurveTo(radius * 0.42, -radius * 0.12, radius * 0.35, radius * 0.55, 0, radius * 0.58);
  context.bezierCurveTo(-radius * 0.35, radius * 0.55, -radius * 0.38, -radius * 0.05, 0, -radius * 0.72);
  context.fillStyle = `rgba(255,249,184,${0.92 * visibility})`;
  context.fill();
  context.restore();
};

const drawInferno = (context, effect, config, viewport, tick, tickRate, reducedMotion) => {
  const hull = [];
  for (let index = 0; index < (effect.hull || []).length; index += 2) {
    hull.push(worldToScreen({ x: effect.hull[index], y: effect.hull[index + 1] }, config, viewport));
  }
  const center = worldToScreen(effect, config, viewport);
  const fallbackRadius = Math.max(24, (120 / config.scale) * viewport.baseScale * viewport.zoom);
  const remaining = effectSecondsRemaining(effect, tick, tickRate);
  if (remaining <= 0) return;
  const elapsed = Math.max(0, 7 - remaining);
  const visibility = Math.min(1, elapsed / 0.18, remaining / 0.55);

  const traceArea = () => {
    context.beginPath();
    if (hull.length < 3) {
      context.arc(center.x, center.y, fallbackRadius, 0, TAU);
      return;
    }
    context.moveTo(hull[0].x, hull[0].y);
    for (let index = 1; index < hull.length; index += 1) context.lineTo(hull[index].x, hull[index].y);
    context.closePath();
  };

  const firePoints = [];
  for (let index = 0; index < (effect.points || []).length; index += 2) {
    firePoints.push(worldToScreen({ x: effect.points[index], y: effect.points[index + 1] }, config, viewport));
  }
  if (!firePoints.length) {
    firePoints.push(center);
    for (let index = 0; index < hull.length && firePoints.length < 14; index += 1) {
      const current = hull[index];
      const next = hull[(index + 1) % hull.length];
      firePoints.push({ x: (current.x + center.x) / 2, y: (current.y + center.y) / 2 });
      if (next) firePoints.push({ x: (current.x + next.x + center.x) / 3, y: (current.y + next.y + center.y) / 3 });
    }
  }

  context.save();
  if (hull.length >= 3) {
    traceArea();
    context.clip();
  }
  context.globalCompositeOperation = "lighter";
  const scale = clamp(viewport.zoom, 0.82, 1.35);
  firePoints.forEach((point, index) => drawFlameCell(context, point, index, elapsed, visibility, reducedMotion, scale));
  context.restore();
};

export const drawEffects = (context, frame, config, viewport, tick, tickRate, reducedMotion) => {
  for (const effect of frame?.active_effects || []) {
    if (!isReplayEffectVisible(effect, tick, tickRate)) continue;
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
