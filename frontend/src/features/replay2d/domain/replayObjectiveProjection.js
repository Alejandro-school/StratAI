const PUBLIC_EVENTS = new Set(["bomb_plant", "bomb_defuse", "bomb_explode"]);
const OWN_EVENTS = new Set(["bomb_carrier_snapshot", "bomb_drop", "bomb_pickup", "bomb_plant_start",
  "bomb_plant_abort", "bomb_defuse_start", "bomb_defuse_abort"]);
const finitePosition = (value) => Number.isFinite(value?.x) && Number.isFinite(value?.y);

// Objective knowledge is projected separately from geometry. Event coordinates
// may belong to the actor; only a current native bomb sample supplies location.
export function projectReplayObjective(round, sampled, causalEvents, ownPlayers, side, tick) {
  const ownIds = new Set(ownPlayers.map((p) => String(p.steam_id)));
  const availableEvents = causalEvents.filter((e) => Number.isFinite(e.availability_tick ?? e.tick)
    && (e.availability_tick ?? e.tick) >= e.tick && (e.availability_tick ?? e.tick) <= tick);
  const publicEvents = availableEvents.filter((e) => PUBLIC_EVENTS.has(e.type));
  const events = availableEvents.filter((e) => PUBLIC_EVENTS.has(e.type)
    || (OWN_EVENTS.has(e.type) && (e.player_team === side || ownIds.has(String(e.player_id))))).map((e) => {
    const own = e.player_team === side || ownIds.has(String(e.player_id));
    const result = { id: e.id, type: e.type, tick: e.tick, availability_tick: e.availability_tick ?? e.tick,
      lane: "objective", label: e.label,
      ...(e.objective_sequence ? { objective_sequence: e.objective_sequence } : {}),
      knowledge: own ? "own_team_objective_event" : "public_objective_event",
      source: { reference: e.sourceId || e.id, kind: e.source || "demo_objective_event" } };
    if (own) {
      result.player_id = String(e.player_id);
      if (e.state_after) result.state_after = e.state_after;
      if (e.objective_phase) result.objective_phase = e.objective_phase;
    }
    return result;
  });
  const latestPublic = publicEvents.at(-1);
  const plant = publicEvents.find((e) => e.type === "bomb_plant");
  let bomb = latestPublic ? {
    state: { bomb_plant: "planted", bomb_defuse: "defused", bomb_explode: "exploded" }[latestPublic.type],
    objective_phase: latestPublic.type === "bomb_plant" ? "planted" : "resolved",
    is_planted_now: latestPublic.type === "bomb_plant", was_planted_this_round: true,
    position_status: "unavailable", position_reason: "perspective_source_unavailable",
    observation_tick: latestPublic.tick, availability_tick: latestPublic.availability_tick ?? latestPublic.tick,
    source: { reference: latestPublic.sourceId || latestPublic.id, kind: "public_objective_event" },
    ...(plant ? { plant_tick: plant.tick } : {}),
  } : null;
  const candidate = sampled?.bomb;
  const native = candidate?.observation_tick != null && candidate.observation_tick !== sampled.tick ? null : candidate;
  const carrier = ownPlayers.find((p) => p.alive && String(p.steam_id) === String(native?.carrier_id));
  const defuserOwn = native?.defuser_id != null && ownIds.has(String(native.defuser_id));
  const ownPlant = plant && ownIds.has(String(plant.player_id));
  // T's ground-C4 knowledge was explicitly framed for this replay instrument.
  // Hidden enemy defuse attempts remain planted from T's public perspective.
  const stateAllowed = native && (side === "T" || defuserOwn || native.objective_phase === "resolved");
  if (stateAllowed && (!latestPublic || sampled.tick >= latestPublic.tick)) {
    const hiddenDefuse = side === "T" && native.state === "defusing" && !defuserOwn;
    bomb = {
      state: hiddenDefuse ? "planted" : native.state,
      ...(native.objective_phase ? { objective_phase: hiddenDefuse ? "planted" : native.objective_phase } : {}),
      ...(typeof native.is_planted_now === "boolean" ? { is_planted_now: native.is_planted_now } : {}),
      ...(typeof native.was_planted_this_round === "boolean" ? { was_planted_this_round: native.was_planted_this_round } : {}),
      ...(native.plant_tick > 0 && native.plant_tick <= tick ? { plant_tick: native.plant_tick } : {}),
      position_status: native.position_status === "unknown" ? "unknown" : "unavailable", position_reason: "native_position_unavailable",
      observation_tick: sampled.tick, availability_tick: sampled.tick,
      source: { reference: `round:${round.round}:frame:${sampled.tick}:bomb`, kind: "demo_objective_state" },
    };
    if (carrier && ["carried", "planting"].includes(native.state)) bomb.carrier_id = String(carrier.steam_id);
    if (defuserOwn) bomb.defuser_id = String(native.defuser_id);
    // A resolved round is public; its hidden preplant history/location is not.
    if (side === "CT" && !defuserOwn) {
      delete bomb.was_planted_this_round;
      delete bomb.plant_tick;
      if (plant) { bomb.was_planted_this_round = true; bomb.plant_tick = plant.tick; }
    }
  }
  const newerEvent = events.filter((e) => e.tick > (sampled?.tick ?? -Infinity)).at(-1);
  if (newerEvent) {
    const state = newerEvent.state_after || {
      bomb_carrier_snapshot: "carried", bomb_drop: "dropped", bomb_pickup: "carried",
      bomb_plant_start: "planting", bomb_plant_abort: "carried", bomb_plant: "planted",
      bomb_defuse_start: "defusing", bomb_defuse_abort: "planted", bomb_defuse: "defused", bomb_explode: "exploded",
    }[newerEvent.type];
    bomb = { ...bomb, state, objective_phase: newerEvent.objective_phase ||
      ({ carried: "preplant", dropped: "preplant", defused: "resolved", exploded: "resolved" }[state] || state),
      is_planted_now: ["planted", "defusing"].includes(state),
      position_status: "unavailable", position_reason: "event_without_current_native_position",
      observation_tick: newerEvent.tick, availability_tick: newerEvent.availability_tick, source: newerEvent.source };
    delete bomb.carrier_id; delete bomb.defuser_id;
    if (["carried", "planting"].includes(state) && newerEvent.player_id) bomb.carrier_id = newerEvent.player_id;
    if (state === "defusing" && newerEvent.player_id) bomb.defuser_id = newerEvent.player_id;
  }
  if (bomb && native) {
    const locationAllowed = carrier || defuserOwn || (side === "T" && (native.state === "dropped" || ownPlant));
    if (sampled.tick === tick && native.position_status === "observed" && finitePosition(native) && locationAllowed
      && (!latestPublic || sampled.tick >= latestPublic.tick)) {
      bomb.x = native.x; bomb.y = native.y;
      if (Number.isFinite(native.z)) bomb.z = native.z;
      bomb.position_status = "observed"; delete bomb.position_reason;
      bomb.position_source = native.position_source;
      bomb.position_tick = sampled.tick;
      if (native.site && ownPlant) bomb.site = native.site;
    } else if (sampled.tick !== tick) {
      bomb.position_reason = "between_samples";
      // Explicitly historical, never a current coordinate. The renderer labels
      // its age; a newer transition or missing sample immediately removes it.
      if (!newerEvent && native.position_status === "observed" && finitePosition(native) && locationAllowed) {
        bomb.last_observed_position = { x: native.x, y: native.y, observation_tick: sampled.tick,
          source: { reference: `round:${round.round}:frame:${sampled.tick}:bomb`, kind: native.position_source } };
      }
    }
  }
  const snapshotCarrierDied = !newerEvent && ["carried", "planting"].includes(bomb?.state)
    && ownPlayers.some((p) => !p.alive && String(p.steam_id) === String(native?.carrier_id));
  if (snapshotCarrierDied || (bomb?.carrier_id && !ownPlayers.some((p) => p.alive && String(p.steam_id) === bomb.carrier_id))) {
    delete bomb.carrier_id;
    bomb.state = "unknown"; bomb.objective_phase = "unknown";
    bomb.state_reason = "carrier_dead_without_objective_transition";
    delete bomb.x; delete bomb.y; delete bomb.z; delete bomb.last_observed_position;
    bomb.position_status = "unavailable";
  }
  return { bomb, events };
}
