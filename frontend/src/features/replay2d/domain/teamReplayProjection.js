import { normalizeReplayEvent } from "./replayModel";
import { playerIdentity } from "./weaponPresentation";
import { projectReplayObjective } from "./replayObjectiveProjection";
import { projectReplayPlaces, REPLAY_PLACE_REFERENCE } from "./replayPlaces";
import { projectReplayRoundContext, ROUND_TEAM_REPLAY_SCHEMA } from "./replayRoundContext";
import { createReplayEventsProjector } from "./replayEvents";
import { createReplayInformationProjector, INFORMATION_TEAM_REPLAY_SCHEMA } from "./replayInformation";
import { createReplayUtilityInformationProjector, UTILITY_TEAM_REPLAY_SCHEMA } from './replayUtilityInformation';
import { createReplaySoundInformationProjector, SOUND_TEAM_REPLAY_SCHEMA } from './replaySoundInformation';

export const TEAM_REPLAY_SCHEMA = "stratai.team_replay_projection@4";
export const ZONES_TEAM_REPLAY_SCHEMA = "stratai.team_replay_projection@5";
const positionKnown = (point) => Number.isFinite(point?.x) && Number.isFinite(point?.y);
const sourceKnown = (source) => typeof source?.reference === "string" && source.reference.length > 0;

// Build once per round; every output uses a causal prefix. Raw frames, enemy
// coordinates and legacy LOS/firstSeen flags never form the emitted context.
export function createTeamReplayProjector(round, side, confirmedVision = null, sourceIdentity = null, contactPolicy = "calculated", options = {}) {
  if (!round || !["T", "CT"].includes(side)) return null;
  const withPlaces = sourceIdentity?.map_name === "de_mirage";
  const withRound = withPlaces && (round.frames || []).some(frame => ['stratai.replay_round_context@1', 'stratai.replay_round_context@2'].includes(frame.round_context?.schema_id));
  const withEvents = withRound && round.event_context?.schema_id === 'stratai.replay_events@1';
  const withFacts = withEvents && (round.frames || []).some(frame => frame.round_context?.schema_id === 'stratai.replay_round_context@2');
  const projectEvents = withEvents ? createReplayEventsProjector(round, side, contactPolicy, withFacts) : null;
  const frames = [...(round.frames || [])].sort((a, b) => a.tick - b.tick);
  const roundStart = Number.isFinite(round.start_tick) ? round.start_tick : frames[0]?.tick;
  const ownFrames = frames.map((frame) => ({
    tick: frame.tick,
    time_remaining: frame.time_remaining,
    players: (frame.players || []).filter((player) => player.team === side).map((player) => {
      const copy = { ...player };
      if (withPlaces) delete copy.place;
      return copy;
    }),
    projectiles: [], active_effects: [], shots: [], bomb: null,
  }));
  // A reset can reuse the numeric round identifier. Events from an earlier
  // instance must not turn freshly spawned players into corpses in this round.
  const events = (round.events || []).map(normalizeReplayEvent)
    .filter((event) => event.tick >= roundStart).sort((a, b) => a.tick - b.tick);
  const calculatedSource = contactPolicy === "calculated" && round.calculated_vision?.schema_id === "stratai.calculated_vision@2"
    && round.calculated_vision.producer === "go.reaction.world_occlusion_v2"
    && round.calculated_vision.sample_stride_ticks === 4 ? round.calculated_vision : null;
  const samples = [...(calculatedSource?.samples || [])].filter((sample) => Number.isFinite(sample.tick)
    && sample.tick >= roundStart && sample.availability_tick === sample.tick).sort((a, b) => a.tick - b.tick);
  // Only an independently validated, round-bound source may supply confirmed
  // observations. The existing Go proxies are intentionally not adapted here.
  const contacts = confirmedVision?.schema_id === "stratai.confirmed_vision_source@1"
    && confirmedVision.round === round.round && sourceKnown(confirmedVision.source)
    && typeof sourceIdentity?.match_id === "string" && sourceIdentity.match_id.length > 0
    && /^[a-f0-9]{64}$/.test(sourceIdentity?.demo_sha256 || "")
    && confirmedVision.match_id === sourceIdentity.match_id
    && confirmedVision.demo_sha256 === sourceIdentity.demo_sha256
    ? confirmedVision.observations || [] : [];
  const projectInformation = options.information === true && withFacts
    ? createReplayInformationProjector(round, side, contacts, contactPolicy) : null;
  const projectUtilityInformation = projectInformation && options.utilityInformation === true
    ? createReplayUtilityInformationProjector(round, side, sourceIdentity?.tick_rate) : null;
  const projectSoundInformation = projectUtilityInformation && options.soundInformation === true
    ? createReplaySoundInformationProjector(round, side, sourceIdentity?.map_name) : null;

  return (tick) => {
    let low = 0;
    let high = frames.length;
    while (low < high) {
      const middle = (low + high) >> 1;
      if (frames[middle].tick <= tick) low = middle + 1;
      else high = middle;
    }
    const index = low - 1;
    const sampled = frames[index];
    if (withFacts) {
      const prefix = frames.slice(0, index + 1);
      const ended = prefix.find(f => f.round_context?.phase === 'ended' && Number.isInteger(f.round_context.end_tick)
        && f.round_context.end_tick <= f.tick && f.round_context.end_tick > 0);
      if (ended && prefix.some(f => f.tick > ended.tick && (f.round_context.phase !== 'ended'
        || f.round_context.end_tick !== ended.round_context.end_tick || f.round_context.winner !== ended.round_context.winner))) {
        throw new Error('Terminal round reopened or outcome changed');
      }
    }
    const ownFrame = ownFrames[index];
    const ownIds = new Set((ownFrame?.players || []).map(playerIdentity));
    const observers = new Set((ownFrame?.players || []).filter((p) => p.alive).map(playerIdentity));
    const causalEvents = events.filter((event) => event.tick <= tick);
    const deadIds = new Set(causalEvents.filter((event) => event.type === "kill").map((event) => String(event.victim_id)));
    const visible = new Map();
    let sampleLow = 0, sampleHigh = samples.length;
    while (sampleLow < sampleHigh) {
      const middle = (sampleLow + sampleHigh) >> 1;
      if (samples[middle].tick <= tick) sampleLow = middle + 1;
      else sampleHigh = middle;
    }
    const contactSample = samples[sampleLow - 1];
    const calculationAvailable = contactSample?.status === "calculated" && tick < contactSample.tick + 4;
    if (calculationAvailable) for (const contact of contactSample.contacts || []) {
      const id = String(contact.subject_id);
      if (contact.observer_side !== side || !["T", "CT"].includes(contact.subject_side) || contact.subject_side === side
        || !observers.has(String(contact.observer_id)) || deadIds.has(String(contact.observer_id))
        || !/^\d+$/.test(id) || ownIds.has(id) || deadIds.has(id) || !positionKnown(contact)) continue;
      visible.set(id, {
        steam_id: id, name: id.length <= 3 ? `P${id.padStart(2, "0")}` : "Rival visible", team: contact.subject_side, alive: true,
        x: contact.x, y: contact.y, ...(Number.isFinite(contact.z) ? { z: contact.z } : withPlaces ? {} : { z: 0 }),
        knowledge: "calculated_visual_contact", observation_tick: contactSample.tick,
        availability_tick: contactSample.availability_tick,
        source: { kind: "go_calculated_contact", reference: `round:${round.round}:contact:${contactSample.tick}:${contact.observer_id}:${id}` },
      });
    }
    for (const contact of contacts) {
      if (contact.tick !== sampled?.tick || contact.availability_tick > tick
        || !Number.isFinite(contact.availability_tick) || contact.availability_tick < contact.tick
        || contact.status !== "observed" || contact.visual_contact !== true
        || contact.observer_side !== side || contact.subject_side === side
        || !["T", "CT"].includes(contact.subject_side)
        || !observers.has(String(contact.observer_id)) || ownIds.has(String(contact.subject_id))
        || !sourceKnown(contact.source) || !positionKnown(contact.position)) continue;
      visible.set(String(contact.subject_id), {
        steam_id: String(contact.subject_id), name: contact.subject_name || "Rival visible",
        team: contact.subject_side, alive: true,
        x: contact.position.x, y: contact.position.y,
        ...(Number.isFinite(contact.position.z) ? { z: contact.position.z } : withPlaces ? {} : { z: 0 }),
        knowledge: "confirmed_visual_contact", source: contact.source,
        observation_tick: contact.tick, availability_tick: contact.availability_tick,
      });
    }
    const deaths = causalEvents.filter((event) => event.type === "kill" && ["T", "CT"].includes(event.victim_team)).map((event) => ({
      id: event.id, tick: event.tick, availability_tick: event.tick,
      victim_id: String(event.victim_id), name: event.victim_name || "Jugador",
      team: event.victim_team,
      position: event.victim_position_available && Number.isFinite(event.victim_x) && Number.isFinite(event.victim_y)
        ? { x: event.victim_x, y: event.victim_y } : null,
      knowledge: "simulated_team_communication",
      source: { reference: event.sourceId || event.id, kind: "demo_kill_event" },
    }));
    const players = (ownFrame?.players || []).map((player) => ({
      ...player, alive: deadIds.has(playerIdentity(player)) ? false : player.alive,
      knowledge: "own_team_snapshot", observation_tick: sampled.tick,
      availability_tick: sampled.tick,
      source: { reference: `round:${round.round}:frame:${sampled.tick}:player:${playerIdentity(player)}`, kind: "demo_replay_frame" },
    }));
    if (withFacts) for (const player of players) {
      const death = causalEvents.find(event => event.type === 'kill' && event.tick > sampled.tick
        && String(event.victim_id) === playerIdentity(player) && event.victim_team === side);
      if (!death) continue;
      // The callback updates life, not the observation time of position/equipment.
      player.alive = false;
      player.health = 0;
      player.life_context = { schema_id: 'stratai.replay_life_context@1', status: 'observed',
        alive: false, health: 0, observation_tick: death.tick, availability_tick: death.tick,
        source: { kind: 'demo_kill_event', reference: death.id } };
      if (player.inventory_context?.status === 'observed') {
        player.inventory_context = { ...player.inventory_context, status: 'last_observed' };
      }
    }
    for (const [id, player] of visible) if (!deadIds.has(id)) players.push(player);
    const safeEvents = deaths.map((death) => ({
      id: death.id, type: "kill", tick: death.tick, lane: "combat", label: "Muerte comunicada",
      victim_id: death.victim_id, victim_name: death.name, victim_team: death.team,
      // No killer/assister position, weapon, line, or hidden resource survives.
      knowledge: death.knowledge, source: death.source,
    }));
    const objective = projectReplayObjective(round, sampled, causalEvents, players.filter((p) => p.team === side), side, tick);
    const bomb = objective.bomb;
    safeEvents.push(...objective.events);
    const actionContext = projectEvents?.(tick);
    if (actionContext) {
      // Kill feed identities are public event facts. They never authorize the
      // killer's position, health, remaining equipment or visual observation.
      for (const event of safeEvents.filter(e => e.type === 'kill')) {
        const raw = causalEvents.find(e => e.id === event.id);
        event.availability_tick = event.tick;
        event.knowledge = 'public_killfeed';
        if (raw?.killer_id) { event.killer_id = String(raw.killer_id); event.killer_team = raw.killer_team; }
        if (raw?.weapon) event.weapon = raw.weapon;
      }
      safeEvents.push(...actionContext.events);
    }
    safeEvents.sort((a, b) => a.tick - b.tick);
    // The tracker identifies one live carrier. Cached inventories can retain C4
    // on earlier owners and are never used as possession or tactical intent.
    for (const player of players) if (player.team === side && bomb) {
      player.has_c4 = player.alive && String(player.steam_id) === bomb.carrier_id;
    }
    const frame = ownFrame ? { ...ownFrame, players: withPlaces ? projectReplayPlaces(players, sourceIdentity.map_name) : players,
      death_markers: deaths.filter((death) => death.position), bomb,
      perspective: side } : null;
    if (withRound && frame) {
      frame.round_context = projectReplayRoundContext(round, sampled, safeEvents, players.filter(p => p.team === side), bomb, tick);
      frame.time_remaining = frame.round_context.round_remaining_seconds ?? frame.round_context.bomb_remaining_seconds;
    }
    if (actionContext && frame) Object.assign(frame, {shots:actionContext.shots,
      projectiles:actionContext.projectiles, active_effects:actionContext.active_effects});
    const information = projectInformation?.(tick);
    if (information && frame) {
      frame.last_seen_markers = information.contacts.filter(c => c.state === 'last_observed')
        .map(c => ({ ...c, name: c.subject_id.length <= 3 ? `P${c.subject_id.padStart(2, '0')}` : 'Rival observado',
          team: c.subject_side, knowledge: 'historical_contact', current_position_status: 'unknown' }));
    }
    const utilityInformation = projectUtilityInformation?.(tick);
    if (utilityInformation && frame) {
      frame.utility_markers = utilityInformation.objects;
      frame.utility_segments = utilityInformation.segments;
    }
    const soundInformation = projectSoundInformation?.(tick);
    return {
      schema_id: soundInformation ? SOUND_TEAM_REPLAY_SCHEMA : utilityInformation ? UTILITY_TEAM_REPLAY_SCHEMA : information ? INFORMATION_TEAM_REPLAY_SCHEMA : withFacts ? 'stratai.team_replay_projection@8' : withEvents ? 'stratai.team_replay_projection@7' : withRound ? ROUND_TEAM_REPLAY_SCHEMA : withPlaces ? ZONES_TEAM_REPLAY_SCHEMA : TEAM_REPLAY_SCHEMA, round: round.round, side, tick,
      ...(information ? { information } : {}),
      ...(utilityInformation ? { utility_information: utilityInformation } : {}),
      ...(soundInformation ? { sound_information: soundInformation } : {}),
      ...(withPlaces ? { place_reference_id: REPLAY_PLACE_REFERENCE } : {}),
      availability_tick: sampled?.tick ?? null,
      contact_policy: contactPolicy,
      vision_status: [...visible.values()].some((p) => p.knowledge === "confirmed_visual_contact") ? "confirmed_contact_observed"
        : visible.size ? "calculated_contact_visible" : calculationAvailable ? "calculated_no_contact" : "unavailable",
      vision_reason: visible.size || calculationAvailable ? null : "visual_source_unavailable",
      frame, frames: ownFrames.slice(0, index + 1), events: safeEvents,
      deaths, unknown_death_positions: deaths.filter((death) => !death.position).length,
      uncertainty: { hidden_enemy_positions: "unknown", enemy_resources: "unknown",
        visual_contact_between_samples: "unknown", calculated_vision_accuracy: "estimated_not_confirmed",
        unprojected_utility_and_shots: actionContext ? "partial_causal_events_supplied" : "unknown", ...(actionContext ? {events:actionContext.limitations} : {}) },
    };
  };
}
