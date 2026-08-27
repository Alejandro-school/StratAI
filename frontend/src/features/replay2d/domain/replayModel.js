export const clamp = (value, min, max) => Math.min(max, Math.max(min, value));

export const lerp = (a, b, amount) => a + (b - a) * amount;

export function lerpAngle(a = 0, b = 0, amount) {
  let delta = ((b - a + 540) % 360) - 180;
  return a + delta * amount;
}

export function closestFrameIndex(frames, targetTick) {
  if (!frames?.length || !Number.isFinite(Number(targetTick))) return -1;
  let low = 0;
  let high = frames.length - 1;
  while (low <= high) {
    const middle = (low + high) >> 1;
    const tick = Number(frames[middle]?.tick || 0);
    if (tick === targetTick) return middle;
    if (tick < targetTick) low = middle + 1;
    else high = middle - 1;
  }
  if (low >= frames.length) return frames.length - 1;
  if (high < 0) return 0;
  return Math.abs(frames[low].tick - targetTick) < Math.abs(frames[high].tick - targetTick)
    ? low
    : high;
}

export function activeCombatShotsAtTick(shots, targetTick, visibilityTicks = 32) {
  const tick = Number(targetTick);
  if (!Array.isArray(shots) || !Number.isFinite(tick)) return [];
  const lowerTick = tick - visibilityTicks;
  let low = 0;
  let high = shots.length;
  while (low < high) {
    const middle = (low + high) >> 1;
    if (Number(shots[middle]?.tick) < lowerTick) low = middle + 1;
    else high = middle;
  }
  const first = low;
  high = shots.length;
  while (low < high) {
    const middle = (low + high) >> 1;
    if (Number(shots[middle]?.tick) <= tick) low = middle + 1;
    else high = middle;
  }
  return shots.slice(first, low);
}

export function interpolateFrame(frames, targetTick) {
  if (!frames?.length) return null;
  const index = closestFrameIndex(frames, targetTick);
  if (index < 0) return null;
  const nearest = frames[index];
  const before = nearest.tick > targetTick ? frames[Math.max(0, index - 1)] : nearest;
  const after = nearest.tick < targetTick ? frames[Math.min(frames.length - 1, index + 1)] : nearest;
  if (before === after || after.tick === before.tick) return { ...nearest, interpolatedTick: nearest.tick };
  const amount = clamp((targetTick - before.tick) / (after.tick - before.tick), 0, 1);
  const nextById = new Map((after.players || []).map((player) => [String(player.steam_id), player]));
  const players = (before.players || []).map((player) => {
    const next = nextById.get(String(player.steam_id));
    if (!next) return player;
    return {
      ...player,
      x: lerp(player.x, next.x, amount),
      y: lerp(player.y, next.y, amount),
      z: lerp(player.z || 0, next.z || 0, amount),
      yaw: lerpAngle(player.yaw, next.yaw, amount),
      flash_duration: lerp(player.flash_duration || 0, next.flash_duration || 0, amount),
    };
  });
  return {
    ...before,
    players,
    time_remaining: lerp(before.time_remaining || 0, after.time_remaining || 0, amount),
    interpolatedTick: targetTick,
  };
}

const LEGACY_UTILITY = new Map([
  ["flash_explode", "flashbang"],
  ["he_explode", "he"],
  ["smoke_start", "smoke"],
  ["inferno_start", "inferno"],
  ["molotov_start", "inferno"],
]);

function eventLabel(type, subtype) {
  if (type === "kill") return "Eliminación";
  if (type === "player_hurt") return "Impacto recibido";
  if (type.startsWith("bomb_")) return {
    bomb_plant: "Bomba plantada",
    bomb_defuse: "Bomba desactivada",
    bomb_explode: "Bomba detonada",
  }[type] || "Objetivo";
  return {
    flashbang: "Flash",
    he: "HE",
    smoke: "Humo",
    inferno: "Fuego",
    molotov: "Molotov",
  }[subtype] || "Utilidad";
}

export function normalizeReplayEvent(event, index = 0) {
  const originalType = String(event?.type || "unknown").toLowerCase();
  let type = originalType;
  let subtype = String(event?.utility_type || event?.grenade_type || "").toLowerCase();
  if (LEGACY_UTILITY.has(originalType)) {
    type = "utility_detonate";
    subtype = LEGACY_UTILITY.get(originalType);
  } else if (originalType === "grenade_explode") {
    type = "utility_detonate";
    subtype ||= "he";
  }
  if (subtype === "flash") subtype = "flashbang";
  if (subtype === "incendiary") subtype = "inferno";
  const lane = type === "kill" || type === "player_hurt" ? "combat" : type.startsWith("bomb_") ? "objective" : type === "utility_detonate" ? "utility" : "other";
  const actorId = event.actor_id || event.player_id || event.killer_id || null;
  const sourceId = event.id || `${type}-${event.tick || 0}-${actorId || "system"}`;
  return {
    ...event,
    // Some parser versions reuse event ids within the same round. Keeping the
    // source id is useful for diagnostics, while the rendered id must be unique.
    sourceId,
    id: `${sourceId}-${index}`,
    type,
    originalType,
    subtype,
    actorId,
    lane,
    label: eventLabel(type, subtype),
    tick: Number(event.tick || 0),
    x: Number(event.x ?? event.victim_x ?? 0),
    y: Number(event.y ?? event.victim_y ?? 0),
    z: Number(event.z || 0),
    durationMs: Number(event.duration_ms || (
      type === "player_hurt" ? 320 : subtype === "smoke" ? 18000 : subtype === "inferno" ? 7000 : subtype === "flashbang" ? 450 : 600
    )),
  };
}

function derivePlayerHurtEvents(frames) {
  const events = [];
  for (let frameIndex = 1; frameIndex < frames.length; frameIndex += 1) {
    const previousPlayers = new Map((frames[frameIndex - 1].players || []).map((player) => [String(player.steam_id), player]));
    for (const player of frames[frameIndex].players || []) {
      const previous = previousPlayers.get(String(player.steam_id));
      const damage = Number(previous?.health || 0) - Number(player.health || 0);
      if (!previous || damage <= 0) continue;
      events.push({
        id: `derived-hurt-${frames[frameIndex].tick}-${player.steam_id}`,
        tick: frames[frameIndex].tick,
        type: "player_hurt",
        victim_id: player.steam_id,
        victim_name: player.name,
        victim_team: player.team,
        victim_x: player.x,
        victim_y: player.y,
        damage,
        duration_ms: 320,
        derived: true,
      });
    }
  }
  return events;
}

function bombAtTick(source, events, tick) {
  const plant = events.find((event) => event.type === "bomb_plant");
  if (!plant) return source?.state === "carried" ? null : source;
  if (tick < plant.tick) {
    return source?.state === "dropped" ? { ...source, site: "", plant_tick: 0 } : null;
  }

  const defuse = events.find((event) => event.type === "bomb_defuse" && event.tick <= tick);
  const explosion = events.find((event) => event.type === "bomb_explode" && event.tick <= tick);
  const state = explosion ? "exploded" : defuse ? "defused" : source?.state === "defusing" ? "defusing" : "planted";
  return {
    ...source,
    state,
    x: plant.x,
    y: plant.y,
    site: plant.site || source?.site || "",
    plant_tick: plant.tick,
    defuser_id: defuse?.player_id || source?.defuser_id || 0,
  };
}

export function normalizeRound(round) {
  if (!round) return null;
  const frames = [...(round.frames || [])].sort((a, b) => a.tick - b.tick);
  const hasAtomicCombatShots = Array.isArray(round.combat_shots);
  const combatShots = [...(round.combat_shots || [])].sort((a, b) => a.tick - b.tick);
  const suppliedEvents = round.events || [];
  const rawEvents = suppliedEvents.some((event) => String(event.type).toLowerCase() === "player_hurt")
    ? suppliedEvents
    : [...suppliedEvents, ...derivePlayerHurtEvents(frames)];
  const events = rawEvents.map(normalizeReplayEvent).sort((a, b) => a.tick - b.tick);
  const effectStarts = new Map();
  const normalizedFrames = frames.map((frame) => {
    const activeKeys = new Set();
    const activeEffects = (frame.active_effects || []).map((effect) => {
      const key = `${effect.type}:${Math.round(effect.x / 24)}:${Math.round(effect.y / 24)}`;
      activeKeys.add(key);
      if (!effectStarts.has(key)) effectStarts.set(key, Number(effect.start_tick || frame.tick));
      return { ...effect, start_tick: Number(effect.start_tick || effectStarts.get(key)) };
    });
    for (const key of effectStarts.keys()) {
      if (!activeKeys.has(key)) effectStarts.delete(key);
    }
    return { ...frame, active_effects: activeEffects, bomb: bombAtTick(frame.bomb, events, frame.tick) };
  });
  const firstFrameTick = Number(normalizedFrames[0]?.tick);
  const lastFrameTick = Number(normalizedFrames.at(-1)?.tick);
  const lastCombatShotTick = Number(combatShots.at(-1)?.tick);
  const declaredEndTick = Number(round.end_tick);
  const atomicEndTick = Math.max(
    Number.isFinite(lastFrameTick) ? lastFrameTick : 0,
    Number.isFinite(lastCombatShotTick) ? lastCombatShotTick : 0,
    Number.isFinite(declaredEndTick) ? declaredEndTick : 0,
  );
  return {
    ...round,
    // Older exports used RoundStart/RoundEnd as their boundaries even though
    // frames are sampled from live play through RoundEndOfficial. Frame bounds
    // keep those replays useful without forcing a demo re-processing pass.
    start_tick: Number.isFinite(firstFrameTick) ? firstFrameTick : Number(round.start_tick || 0),
    end_tick: hasAtomicCombatShots
      ? atomicEndTick
      : (Number.isFinite(lastFrameTick) ? lastFrameTick : Number(round.end_tick || round.start_tick || 0)),
    frames: normalizedFrames,
    events,
    combat_shots: combatShots,
  };
}

export function effectSecondsRemaining(effect, currentTick, tickRate = 64) {
  const duration = effect.type === "smoke" ? 18 : effect.type === "inferno" ? 7 : 0;
  if (!duration) return Math.max(0, Number(effect.time_remaining || 0));
  if (!Number.isFinite(effect.start_tick)) return Math.max(0, Number(effect.time_remaining ?? duration));
  return Math.max(0, duration - (currentTick - effect.start_tick) / Math.max(1, tickRate));
}

export function effectProgress(currentTick, event, tickRate = 64) {
  const elapsedMs = ((currentTick - event.tick) / Math.max(tickRate, 1)) * 1000;
  return clamp(elapsedMs / Math.max(event.durationMs, 1), 0, 1);
}

export function activeEventsAtTick(events, tick, tickRate = 64) {
  return (events || []).filter((event) => {
    if (event.tick > tick) return false;
    const duration = event.type === "kill" ? 1100 : event.durationMs;
    return tick - event.tick <= (duration / 1000) * tickRate;
  });
}

export function directorRate(events, tick, tickRate, enabled, selectedRate = 1) {
  if (!enabled) return selectedRate;
  const closest = (events || []).reduce((distance, event) => Math.min(distance, Math.abs(event.tick - tick)), Infinity);
  if (closest <= tickRate * 4) return selectedRate;
  return Math.max(selectedRate, 2);
}

export function deriveZThreshold(frames, fallback) {
  const values = [];
  for (const frame of frames || []) {
    for (const player of frame.players || []) {
      if (player.alive && Number.isFinite(player.z)) values.push(player.z);
    }
  }
  if (values.length < 2) return 0;
  values.sort((a, b) => a - b);
  let largestGap = 0;
  let threshold = values[Math.floor(values.length / 2)];
  for (let index = 1; index < values.length; index += 1) {
    const gap = values[index] - values[index - 1];
    if (gap > largestGap) {
      largestGap = gap;
      threshold = (values[index] + values[index - 1]) / 2;
    }
  }
  if (largestGap >= 128) return threshold;
  return Number.isFinite(fallback) ? fallback : values[Math.floor(values.length / 2)];
}

export function resolveLevel(players, threshold, focusPlayerId) {
  const alive = (players || []).filter((player) => player.alive);
  const focused = alive.find((player) => String(player.steam_id) === String(focusPlayerId));
  if (focused) return focused.z >= threshold ? "upper" : "lower";
  const upper = alive.filter((player) => player.z >= threshold).length;
  return upper >= alive.length / 2 ? "upper" : "lower";
}

export function formatClock(seconds) {
  const safe = Math.max(0, Number(seconds) || 0);
  return `${Math.floor(safe / 60)}:${Math.floor(safe % 60).toString().padStart(2, "0")}`;
}
