import { worldToScreen } from "./replayViewport";

export const TAU = Math.PI * 2;

export const pathWorld = (context, points, config, viewport) => {
  points.forEach((point, index) => {
    const screen = worldToScreen(point, config, viewport);
    if (index === 0) context.moveTo(screen.x, screen.y);
    else context.lineTo(screen.x, screen.y);
  });
};
