import { worldToScreen } from "./replayViewport";
import { pathWorld, TAU } from "./canvasPrimitives";
import { drawBomb, drawKillEffects, drawShots } from "./renderReplayCombat";
import { drawPlayers } from "./renderReplayPlayers";
import { drawEffects, drawEventEffects, drawProjectiles, getHeShake } from "./renderReplayUtility";

const drawMap = (context, viewport, images, activeLevel) => {
  const destination = [viewport.originX, viewport.originY, viewport.mapSize, viewport.mapSize];
  const inactive = activeLevel === "upper" ? images.lower : images.upper;
  if (inactive?.complete) {
    context.globalAlpha = 0.13;
    context.filter = "saturate(.2) brightness(.55)";
    context.drawImage(inactive, ...destination);
  }
  const active = images[activeLevel] || images.upper;
  if (active?.complete) {
    context.globalAlpha = 0.94;
    context.filter = "saturate(.72) contrast(1.08) brightness(.76)";
    context.drawImage(active, ...destination);
  }
  context.filter = "none";
  context.globalAlpha = 1;
  context.strokeStyle = "rgba(255,255,255,.12)";
  context.strokeRect(...destination);
};

const annotationPoints = (annotation) => {
  if (Array.isArray(annotation.points)) return annotation.points;
  if (Number.isFinite(annotation.x) && Number.isFinite(annotation.y)) return [{ x: annotation.x, y: annotation.y }];
  return [];
};

const drawAnnotations = (context, annotations, config, viewport) => {
  for (const annotation of annotations || []) {
    const points = annotationPoints(annotation);
    if (!points.length) continue;
    const color = annotation.color || "#63d7ff";
    if (annotation.type === "circle" || annotation.type === "DANGER_ZONE") {
      const center = worldToScreen(points[0], config, viewport);
      const edge = points[1] ? worldToScreen(points[1], config, viewport) : null;
      const radius = edge ? Math.hypot(edge.x - center.x, edge.y - center.y) : ((annotation.radius || 120) / config.scale) * viewport.baseScale * viewport.zoom;
      context.beginPath();
      context.arc(center.x, center.y, radius, 0, TAU);
      context.fillStyle = `${color}2b`;
      context.fill();
      context.strokeStyle = color;
      context.lineWidth = 2;
      context.stroke();
    } else if (annotation.type === "note") {
      const position = worldToScreen(points[0], config, viewport);
      context.fillStyle = "rgba(5,10,18,.9)";
      context.strokeStyle = color;
      context.fillRect(position.x - 5, position.y - 18, 118, 24);
      context.strokeRect(position.x - 5, position.y - 18, 118, 24);
      context.fillStyle = "#fff";
      context.font = "600 11px system-ui";
      context.textAlign = "left";
      context.fillText((annotation.text || "Nota táctica").slice(0, 18), position.x + 2, position.y - 2);
    } else {
      context.beginPath();
      pathWorld(context, points, config, viewport);
      context.strokeStyle = color;
      context.lineWidth = 3;
      context.lineCap = "round";
      context.lineJoin = "round";
      context.stroke();
      if (annotation.type === "arrow" && points.length > 1) {
        const from = worldToScreen(points.at(-2), config, viewport);
        const to = worldToScreen(points.at(-1), config, viewport);
        const angle = Math.atan2(to.y - from.y, to.x - from.x);
        context.beginPath();
        context.moveTo(to.x, to.y);
        context.lineTo(to.x - Math.cos(angle - 0.55) * 12, to.y - Math.sin(angle - 0.55) * 12);
        context.lineTo(to.x - Math.cos(angle + 0.55) * 12, to.y - Math.sin(angle + 0.55) * 12);
        context.closePath();
        context.fillStyle = color;
        context.fill();
      }
    }
  }
};

const drawTrail = (context, trail, config, viewport) => {
  if (!trail?.length) return;
  context.beginPath();
  pathWorld(context, trail, config, viewport);
  context.strokeStyle = "rgba(255,255,255,.46)";
  context.lineWidth = 2;
  context.setLineDash([2, 5]);
  context.stroke();
  context.setLineDash([]);
};

export function renderReplayScene({
  canvas, size, viewport, images, projectileIcons, frame, events, annotations, draft, trail, config,
  tick, tickRate, layers, activeLevel, hasLevels, zThreshold, focusPlayerId, reducedMotion,
}) {
  const context = canvas.getContext("2d");
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(size.width * dpr);
  canvas.height = Math.round(size.height * dpr);
  context.setTransform(dpr, 0, 0, dpr, 0, 0);
  context.clearRect(0, 0, size.width, size.height);
  context.fillStyle = "#070b11";
  context.fillRect(0, 0, size.width, size.height);
  const shake = getHeShake(events, tick, tickRate, reducedMotion);
  const sceneViewport = { ...viewport, originX: viewport.originX + shake.x, originY: viewport.originY + shake.y };
  drawMap(context, sceneViewport, images, activeLevel);
  if (layers.trajectories) drawProjectiles(context, frame, config, sceneViewport, projectileIcons);
  if (layers.utility) {
    drawEffects(context, frame, config, sceneViewport, tick, tickRate, reducedMotion);
    drawEventEffects(context, events, tick, tickRate, config, sceneViewport, reducedMotion);
  }
  if (layers.shots) drawShots(context, frame, config, sceneViewport, tick, tickRate);
  drawKillEffects(context, events, tick, tickRate, config, sceneViewport);
  drawBomb(context, frame?.bomb, config, sceneViewport, tick, tickRate);
  if (layers.annotations) drawAnnotations(context, [...(annotations || []), ...(draft ? [draft] : [])], config, sceneViewport);
  drawTrail(context, trail, config, sceneViewport);
  return drawPlayers(context, frame, config, sceneViewport, {
    layers, activeLevel, hasLevels, zThreshold, focusPlayerId,
  });
}
