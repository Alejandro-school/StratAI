export const REPLAY_INFORMATION_SCHEMA = 'stratai.replay_information@1';
export const INFORMATION_TEAM_REPLAY_SCHEMA = 'stratai.team_replay_projection@9';
const id = value => String(value ?? '');
const finite = Number.isFinite;
const latest = (rows, tick) => {
  let low = 0, high = rows.length;
  while (low < high) { const mid = (low + high) >> 1; if (rows[mid].tick <= tick) low = mid + 1; else high = mid; }
  return rows[low - 1];
};
const order = (a, b) => a.localeCompare(b, 'en', { numeric: true });

// Recipients are eligible under the agreed immediate communication simulation.
// This never proves hearing, attention, memory, or actual voice communication.
export function createReplayInformationProjector(round, side, confirmedContacts = [], contactPolicy = 'calculated') {
  const start = round.start_tick, frames = [...(round.frames || [])].filter(f => f.tick >= start).sort((a, b) => a.tick - b.tick);
  const deaths = (round.events || []).filter(e => e.type === 'kill' && e.tick >= start);
  const alive = (who, tick) => !deaths.some(e => id(e.victim_id) === who && e.tick <= tick);
  const ownAt = tick => (latest(frames, tick)?.players || []).filter(p => p.team === side);
  const vision = round.calculated_vision;
  const samples = contactPolicy === 'calculated' && vision?.schema_id === 'stratai.calculated_vision@2'
    && vision.producer === 'go.reaction.world_occlusion_v2' && vision.sample_stride_ticks === 4
    ? [...(vision.samples || [])].filter(s => finite(s.tick) && s.tick >= start && s.availability_tick === s.tick).sort((a, b) => a.tick - b.tick) : [];
  const observations = [];
  function add(contact, tick, available, position, status, source) {
    const observer = id(contact.observer_id), subject = id(contact.subject_id);
    if (tick < start || available < tick || !finite(available) || contact.observer_side !== side
      || !['T', 'CT'].includes(contact.subject_side) || contact.subject_side === side
      || !/^\d+$/.test(subject) || !finite(position?.x) || !finite(position?.y)
      || !alive(subject, tick) || !alive(observer, tick)) return;
    const own = ownAt(tick);
    if (!own.some(p => id(p.steam_id) === observer && p.alive)
      || own.some(p => id(p.steam_id) === subject)) return;
    const basis = latest(frames, available);
    const recipients = ownAt(available)
      .map(p => id(p.steam_id)).sort(order).map(player_id => ({ player_id,
        kind: player_id === observer ? 'individual_observation' : 'simulated_team_communication',
        reception: player_id === observer ? 'origin' : 'possible', availability_tick: available }));
    // Death prevents new visual observations, not possible team communication.
    if (!recipients.some(r => r.player_id === observer)) return;
    observations.push({ id: `round:${round.round}:start:${start}:contact:${tick}:${observer}:${subject}`,
      observer_id: observer, subject_id: subject, subject_side: contact.subject_side,
      position: { x: position.x, y: position.y, ...(finite(position.z) ? { z: position.z } : {}) },
      observation_tick: tick, availability_tick: available, status, source: { ...source },
      recipient_basis: { frame_tick: basis.tick,
        source: { kind: 'demo_replay_frame', reference: `round:${round.round}:frame:${basis.tick}:recipients:${side}` },
        present_player_ids: recipients.map(r => r.player_id) }, recipients });
  }
  for (const s of samples) if (s.status === 'calculated') for (const c of s.contacts || []) {
    add(c, s.tick, s.availability_tick, c, 'estimated', { kind: 'go_calculated_contact',
      reference: `round:${round.round}:contact:${s.tick}:${c.observer_id}:${c.subject_id}` });
  }
  for (const c of confirmedContacts) if (c.status === 'observed' && c.visual_contact === true && c.source?.reference) {
    add(c, c.tick, c.availability_tick, c.position, 'observed', c.source);
  }
  observations.sort((a, b) => a.availability_tick - b.availability_tick || a.observation_tick - b.observation_tick || order(a.id, b.id));

  return tick => {
    const prefix = observations.filter(o => o.availability_tick <= tick);
    const sample = latest(samples, tick), currentFrame = latest(frames, tick);
    const current = o => alive(o.subject_id, tick) && alive(o.observer_id, tick)
      && ownAt(tick).some(p => id(p.steam_id) === o.observer_id && p.alive)
      && (o.status === 'estimated' ? sample?.status === 'calculated' && sample.tick === o.observation_tick && tick < sample.tick + 4
        : o.observation_tick === currentFrame?.tick);
    const group = rows => {
      const groups = new Map();
      for (const o of rows) {
        const previous = groups.get(o.subject_id) || [];
        if (!previous.length || o.observation_tick > previous[0].observation_tick) groups.set(o.subject_id, [o]);
        else if (o.observation_tick === previous[0].observation_tick) previous.push(o);
      }
      return [...groups.values()].sort((a, b) => order(a[0].subject_id, b[0].subject_id));
    };
    const recentObserver = new Map();
    for (const o of prefix) {
      const key = `${o.observer_id}:${o.subject_id}`, previous = recentObserver.get(key);
      if (!previous || o.observation_tick >= previous.observation_tick) recentObserver.set(key, o);
    }
    const team = group([...recentObserver.values()]).filter(rows => alive(rows[0].subject_id, tick));
    const contacts = team.map(rows => {
      const first = rows[0];
      if (rows.some(o => JSON.stringify(o.position) !== JSON.stringify(first.position))) throw new Error('Conflicting simultaneous contact positions');
      return { subject_id: first.subject_id, subject_side: first.subject_side,
        state: rows.some(current) ? 'current_contact' : 'last_observed', position: { ...first.position },
        observation_tick: first.observation_tick, availability_tick: Math.max(...rows.map(o => o.availability_tick)),
        age_ticks: tick - first.observation_tick, observation_ids: rows.map(o => o.id).sort(order) };
    });
    const known = new Map();
    for (const f of frames) {
      if (f.tick > tick) break;
      for (const p of f.players.filter(p => p.team === side)) if (!known.has(id(p.steam_id))) {
        const player_id = id(p.steam_id);
        known.set(player_id, { player_id, team: side, observation_tick: f.tick,
          source: { kind: 'demo_replay_frame', reference: `round:${round.round}:frame:${f.tick}:player:${player_id}` } });
      }
    }
    const playerIds = [...known.keys()].sort(order);
    const by_player = playerIds.map(player_id => {
      const received = prefix.filter(o => o.recipients.some(r => r.player_id === player_id));
      // A newer possible message must not erase a previous direct observation:
      // actual receipt is unconfirmed, so keep the last report of each channel.
      const direct = group(received.filter(o => o.observer_id === player_id)).flat();
      const shared = group(received.filter(o => o.observer_id !== player_id)).flat();
      return { player_id, observation_ids: [...direct, ...shared].map(o => o.id).sort(order) };
    });
    // Keep all observers' last reports, including historical reports from dead
    // players, and the last report each possible recipient could have received.
    const kept = new Set([...recentObserver.values()].map(o => o.id));
    by_player.forEach(p => p.observation_ids.forEach(ref => kept.add(ref)));
    return { schema_id: REPLAY_INFORMATION_SCHEMA, side, round_start_tick: start,
      communication_model: 'immediate_simulated_possible_reception',
      reception_status: 'possible_not_confirmed',
      own_team_state_scope: 'collective_demo_overview_not_individual_knowledge',
      enemy_identity_scope: 'opaque_demo_tracking_not_recognized_person',
      known_team_players: playerIds.map(who => known.get(who)),
      observations: prefix.filter(o => kept.has(o.id)), contacts, by_player };
  };
}
