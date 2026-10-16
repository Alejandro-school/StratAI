import { describe, expect, it } from 'vitest';
import { createReplayInformationProjector } from './replayInformation';
import { createTeamReplayProjector } from './teamReplayProjection';

const player = (id, team = 'T', alive = true) => ({ steam_id: id, team, alive });
const contact = (observer = '1', x = 100) => ({ observer_id: observer, observer_side: 'T',
  subject_id: '6', subject_side: 'CT', x, y: 20, z: 0, health: 88, weapon: 'AWP' });
const sample = (tick, contacts = []) => ({ tick, availability_tick: tick, status: 'calculated', contacts });
function fixture() {
  return { round: 1, start_tick: 100, frames: [100, 104, 108, 112].map(tick => ({ tick,
    players: [player('1'), player('2'), player('3', 'T', false), player('6', 'CT')] })),
    events: [], calculated_vision: { schema_id: 'stratai.calculated_vision@2',
      producer: 'go.reaction.world_occlusion_v2', sample_stride_ticks: 4,
      samples: [sample(104, [contact()]), sample(108), sample(112, [contact('2', 200)])] } };
}
const project = round => createReplayInformationProjector(round, 'T');

describe('available information with possible immediate team communication', () => {
  it('records the living observer and present possible recipients, including dead teammates, without private resources', () => {
    const info = project(fixture())(104);
    const observation = info.observations[0];
    expect(observation.observer_id).toBe('1');
    expect(observation.recipients).toEqual([
      { player_id: '1', kind: 'individual_observation', reception: 'origin', availability_tick: 104 },
      { player_id: '2', kind: 'simulated_team_communication', reception: 'possible', availability_tick: 104 },
      { player_id: '3', kind: 'simulated_team_communication', reception: 'possible', availability_tick: 104 },
    ]);
    expect(observation.status).toBe('estimated');
    expect(observation.health).toBeUndefined(); expect(observation.weapon).toBeUndefined();
    expect(info.contacts[0].state).toBe('current_contact');
  });
  it('retains a dated last position, updates only from a new observation, and reconstructs reverse seek', () => {
    const get = project(fixture());
    expect(get(103).contacts).toEqual([]);
    expect(get(108).contacts[0]).toMatchObject({ state: 'last_observed', observation_tick: 104,
      age_ticks: 4, position: { x: 100, y: 20, z: 0 } });
    expect(get(110).contacts[0].position.x).toBe(100);
    expect(get(112).contacts[0].position.x).toBe(200);
    expect(get(108).contacts[0].position.x).toBe(100);
  });
  it('preserves two observers of one rival and distinguishes direct from possible shared reception', () => {
    const round = fixture(); round.calculated_vision.samples[0].contacts.push(contact('2'));
    const info = project(round)(104);
    expect(info.contacts).toHaveLength(1); expect(info.contacts[0].observation_ids).toHaveLength(2);
    expect(info.observations.map(o => o.observer_id)).toEqual(['1', '2']);
    expect(info.by_player.find(p => p.player_id === '1').observation_ids).toHaveLength(2);
  });
  it('keeps a dead teammate as a possible shared recipient and retains their earlier direct observation', () => {
    const round = fixture();
    round.events.push({ type: 'kill', id: 'death-1', tick: 106, victim_id: '1', victim_team: 'T' });
    const info = project(round)(112);
    const ownRefs = info.by_player.find(p => p.player_id === '1').observation_ids;
    expect(ownRefs).toEqual(['round:1:start:100:contact:104:1:6', 'round:1:start:100:contact:112:2:6']);
    expect(info.observations.find(o => o.observation_tick === 112).recipients.map(r => r.player_id)).toEqual(['1', '2', '3']);
    expect(info.contacts[0].position.x).toBe(200);
  });
  it('retains information from a teammate who leaves the snapshots, without delivering new contacts', () => {
    const round = fixture();
    round.frames.filter(f => f.tick >= 108).forEach(f => { f.players = f.players.filter(p => p.steam_id !== '1'); });
    const info = project(round)(112);
    expect(info.by_player.find(p => p.player_id === '1').observation_ids).toEqual(['round:1:start:100:contact:104:1:6']);
    expect(info.observations.find(o => o.observation_tick === 112).recipients.map(r => r.player_id)).toEqual(['2', '3']);
  });
  it('removes the live/memory marker after death while keeping the observation as history', () => {
    const round = fixture(); round.events.push({ type: 'kill', id: 'death-6', tick: 106, victim_id: '6', victim_team: 'CT' });
    const info = project(round)(110);
    expect(info.contacts).toEqual([]); expect(info.observations).toHaveLength(1);
    expect(info.observations[0].observation_tick).toBe(104);
  });
  it('is invariant to hidden/future positions, resources and future roster changes', () => {
    const round = fixture(), changed = structuredClone(round);
    changed.frames[0].players[3].x = 999999;
    changed.frames[3].players.push(player('5'));
    changed.calculated_vision.samples[2].contacts[0].x = 999;
    expect(project(changed)(110)).toEqual(project(round)(110));
  });
  it('does not refresh memory through missing vision, expired samples or a dead observer', () => {
    const round = fixture(); round.calculated_vision.samples[1].status = 'unavailable';
    expect(project(round)(108).contacts[0].state).toBe('last_observed');
    expect(project(round)(107.99).contacts[0].state).toBe('current_contact');
    round.events.push({ type: 'kill', tick: 105, victim_id: '1', victim_team: 'T' });
    expect(project(round)(105).contacts[0].state).toBe('last_observed');
  });
  it('rejects a new observation from a dead observer even if the last snapshot still says alive', () => {
    const round = fixture();
    round.events.push({ type: 'kill', tick: 106, victim_id: '1', victim_team: 'T' });
    round.calculated_vision.samples[2].contacts = [contact('1', 200)];
    const info = project(round)(112);
    expect(info.contacts[0].observation_tick).toBe(104);
    expect(info.contacts[0].position.x).toBe(100);
  });
  it('rejects previous-round contacts, future availability and the old wall-blind producer', () => {
    const round = fixture(); round.start_tick = 108;
    expect(project(round)(110).contacts).toEqual([]);
    round.calculated_vision.samples[2].availability_tick = 116;
    expect(project(round)(112).contacts).toEqual([]);
    round.calculated_vision.producer = 'go.reaction.same_endpoint_v1';
    expect(project(round)(120).contacts).toEqual([]);
  });
  it('applies the same recipient rule for CT', () => {
    const round = fixture(); round.frames.forEach(f => f.players.push(player('7', 'CT')));
    round.calculated_vision.samples[0].contacts = [{ ...contact('6'), observer_side: 'CT', subject_id: '1', subject_side: 'T' }];
    const info = createReplayInformationProjector(round, 'CT')(104);
    expect(info.observations[0].recipients.map(r => r.player_id)).toEqual(['6', '7']);
  });
  it('does not replace a newer observation with older evidence that becomes available later', () => {
    const round = fixture();
    const confirmed = [104, 108].map(tick => ({ ...contact(), tick,
      availability_tick: tick === 104 ? 112 : 108, status: 'observed', visual_contact: true,
      source: { kind: 'independent_visual_source', reference: `confirmed:${tick}` }, position: { x: tick, y: 20, z: 0 } }));
    const info = createReplayInformationProjector(round, 'T', confirmed, 'confirmed')(112);
    expect(info.contacts[0].observation_tick).toBe(108);
    expect(info.contacts[0].position.x).toBe(108);
  });
  it('adds history separately from live players through an opt-in version, retaining the old projection', () => {
    const round = fixture();
    round.frames.forEach(f => {
      f.players.forEach(p => Object.assign(p, { x: 0, y: 0, z: 0, health: p.alive ? 100 : 0 }));
      f.round_context = { schema_id: 'stratai.replay_round_context@2', phase: 'unknown',
        round: 1, tick: f.tick, availability_tick: f.tick, freeze_end_tick: 100,
        end_tick: null, clock_status: 'unavailable', pause_state: 'unknown', clock_evidence: {} };
    });
    round.event_context = { schema_id: 'stratai.replay_events@1', events: [] };
    const identity = { map_name: 'de_mirage', tick_rate: 64 };
    const old = createTeamReplayProjector(round, 'T', null, identity)(108);
    const view = createTeamReplayProjector(round, 'T', null, identity, 'calculated', { information: true })(108);
    expect(old.schema_id).toBe('stratai.team_replay_projection@8'); expect(old.information).toBeUndefined();
    expect(view.schema_id).toBe('stratai.team_replay_projection@9');
    expect(view.frame.players.some(p => p.team === 'CT')).toBe(false);
    expect(view.frame.last_seen_markers[0]).toMatchObject({ subject_id: '6', observation_tick: 104, age_ticks: 4 });
    expect(view.frame.last_seen_markers[0].alive).toBeUndefined();
    expect(view.information.contacts[0].position).toEqual(view.frame.last_seen_markers[0].position);
  });
});
