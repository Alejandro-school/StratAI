import { TEAM_COLORS } from "../domain/replayConfig";
import { playerIdentity, stableRosterPlayers } from "../domain/weaponPresentation";
import { worldToScreen } from "./replayViewport";
import { TAU } from "./canvasPrimitives";

const isOnActiveLevel = (player, activeLevel, threshold) => (
  activeLevel === "upper" ? player.z >= threshold : player.z < threshold
);

const drawNameplate = (context, player, position, selected) => {
  const team = TEAM_COLORS[player.team] || TEAM_COLORS.T;
  const name = String(player.name || "Jugador").slice(0, selected ? 18 : 12);
  context.font = `${selected ? 750 : 650} ${selected ? 11 : 10}px system-ui`;
  const width = context.measureText(name).width + 12;
  const above = player.team === "CT";
  const top = above ? position.y - 33 : position.y + 18;
  context.fillStyle = "rgba(3,8,14,.9)";
  context.fillRect(position.x - width / 2, top, width, 16);
  context.fillStyle = team.solid;
  context.fillRect(position.x - width / 2, top, 2, 16);
  context.fillStyle = "#f7fbff";
  context.textAlign = "center";
  context.fillText(name, position.x, top + 11.5);
};

const drawDeadPlayers = (context, players, config, viewport, showDeaths) => {
  if (!showDeaths) return;
  for (const player of players.filter((item) => !item.alive)) {
    const position = worldToScreen(player, config, viewport);
    context.strokeStyle = "rgba(154,164,180,.5)";
    context.lineWidth = 2;
    context.beginPath();
    context.moveTo(position.x - 5, position.y - 5);
    context.lineTo(position.x + 5, position.y + 5);
    context.moveTo(position.x + 5, position.y - 5);
    context.lineTo(position.x - 5, position.y + 5);
    context.stroke();
  }
};

const drawDirectionalMarker = (context, position, yaw, team, number, selected) => {
  const isCounterTerrorist = team.solid === TEAM_COLORS.CT.solid;
  const playerFill = isCounterTerrorist ? "#d9f1ff" : "#ffedbd";

  // Direction is a small pointer attached to the player disc, not the marker's
  // main silhouette.
  context.save();
  context.translate(position.x, position.y);
  context.rotate(yaw);
  context.beginPath();
  context.moveTo(14.5, 0);
  context.lineTo(7, -3.5);
  context.lineTo(7, 3.5);
  context.closePath();
  context.fillStyle = playerFill;
  context.shadowColor = "rgba(0,0,0,.78)";
  context.shadowBlur = selected ? 6 : 3;
  context.shadowOffsetY = 1;
  context.fill();
  context.shadowColor = "transparent";
  context.strokeStyle = selected ? "#ffffff" : "rgba(225,235,242,.92)";
  context.lineWidth = selected ? 1.6 : 1.1;
  context.lineJoin = "round";
  context.stroke();
  context.restore();

  context.beginPath();
  context.arc(position.x, position.y, 9.25, 0, TAU);
  context.fillStyle = playerFill;
  context.fill();
  context.strokeStyle = selected ? "#ffffff" : team.solid;
  context.lineWidth = selected ? 2.2 : 1.5;
  context.stroke();
  context.fillStyle = "#08121c";
  context.font = "850 9px system-ui";
  context.textAlign = "center";
  context.textBaseline = "middle";
  context.fillText(String(number || ""), position.x, position.y + 0.25);
  context.textBaseline = "alphabetic";
};

export const drawPlayers = (context, frame, config, viewport, options) => {
  const hitTargets = [];
  const players = frame?.players || [];
  const ordered = [
    ...stableRosterPlayers(players, "CT"),
    ...stableRosterPlayers(players, "T"),
  ];
  const teamNumber = new Map([
    ...stableRosterPlayers(players, "CT").map((player, index) => [playerIdentity(player), index + 1]),
    ...stableRosterPlayers(players, "T").map((player, index) => [playerIdentity(player), index + 1]),
  ]);
  drawDeadPlayers(context, ordered, config, viewport, options.layers.deaths);
  for (const player of ordered.filter((item) => item.alive)) {
    const position = worldToScreen(player, config, viewport);
    const playerId = playerIdentity(player);
    const selected = playerId === String(options.focusPlayerId);
    const onLevel = !options.hasLevels || isOnActiveLevel(player, options.activeLevel, options.zThreshold);
    const team = TEAM_COLORS[player.team] || TEAM_COLORS.T;
    context.globalAlpha = onLevel ? 1 : 0.25;
    if (options.layers.fov && (selected || viewport.zoom >= 1.15)) {
      const yaw = (-player.yaw * Math.PI) / 180;
      context.beginPath();
      context.moveTo(position.x, position.y);
      context.arc(position.x, position.y, selected ? 58 : 36, yaw - 0.38, yaw + 0.38);
      context.closePath();
      context.fillStyle = team.soft;
      context.fill();
    }
    const yaw = (-player.yaw * Math.PI) / 180;
    drawDirectionalMarker(context, position, yaw, team, teamNumber.get(playerId), selected);
    if (player.flash_duration > 0) {
      context.beginPath();
      context.arc(position.x, position.y, 17, 0, TAU);
      context.strokeStyle = `rgba(255,255,218,${Math.min(1, player.flash_duration / 3)})`;
      context.stroke();
    }
    if (options.layers.names) drawNameplate(context, player, position, selected);
    context.globalAlpha = 1;
    hitTargets.push({ id: playerId, x: position.x, y: position.y, radius: 17 });
  }
  return hitTargets;
};
