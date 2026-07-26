import { clamp } from "../domain/replayModel";

export function createViewport(width, height, zoom = 1, pan = { x: 0, y: 0 }) {
  const baseScale = Math.min(width / 1024, height / 1024);
  const mapSize = 1024 * baseScale * zoom;
  const originX = (width - mapSize) / 2 + pan.x;
  const originY = (height - mapSize) / 2 + pan.y;
  return { width, height, zoom, baseScale, mapSize, originX, originY, pan };
}

export function gameToMap(point, config) {
  return {
    x: (point.x - config.pos_x) / config.scale,
    y: (config.pos_y - point.y) / config.scale,
  };
}

export function mapToGame(point, config) {
  return {
    x: config.pos_x + point.x * config.scale,
    y: config.pos_y - point.y * config.scale,
  };
}

export function worldToScreen(point, config, viewport) {
  const map = gameToMap(point, config);
  return {
    x: viewport.originX + map.x * viewport.baseScale * viewport.zoom,
    y: viewport.originY + map.y * viewport.baseScale * viewport.zoom,
  };
}

export function screenToWorld(point, config, viewport) {
  const scale = viewport.baseScale * viewport.zoom;
  return mapToGame({
    x: (point.x - viewport.originX) / scale,
    y: (point.y - viewport.originY) / scale,
  }, config);
}

export function zoomAtPoint(viewport, point, nextZoom) {
  const zoom = clamp(nextZoom, 0.65, 5);
  const oldScale = viewport.baseScale * viewport.zoom;
  const mapX = (point.x - viewport.originX) / oldScale;
  const mapY = (point.y - viewport.originY) / oldScale;
  const nextMapSize = 1024 * viewport.baseScale * zoom;
  const centeredX = (viewport.width - nextMapSize) / 2;
  const centeredY = (viewport.height - nextMapSize) / 2;
  return {
    zoom,
    pan: {
      x: point.x - mapX * viewport.baseScale * zoom - centeredX,
      y: point.y - mapY * viewport.baseScale * zoom - centeredY,
    },
  };
}
