import { describe, expect, it } from 'vitest';
import { projectReplayRoundContext } from './replayRoundContext';
import { createTeamReplayProjector } from './teamReplayProjection';

const RATE = 64;
const START = 100;
const ROUND_SCHEMA = 'stratai.replay_round_context@2';
const FACTUAL_PROJECTION = 'stratai.team_replay_projection@8';
const source = { map_name: 'de_mirage', tick_rate: RATE };

function scalar(value, declared) {
  return { status: 'observed', value, declared_type: declared,
    native_type: declared === 'GameTime_t' ? 'float32' : typeof value === 'boolean' ? 'bool' : 'int32' };
}

function clockEvidence(tick, phase = 'preplant') {
  const rules = Object.fromEntries([
    'm_bFreezePeriod', 'm_bWarmupPeriod', 'm_bMatchWaitingForResume',
    'm_bTerroristTimeOutActive', 'm_bCTTimeOutActive', 'm_bGameRestart',
    'm_bTechnicalTimeOut', 'm_bGamePaused',
  ].map(name => [name, scalar(false, 'bool')]));
  Object.assign(rules, {
    m_bBombPlanted: scalar(phase === 'planted', 'bool'),
    m_iRoundTime: scalar(115, 'int32'),
    m_fRoundStartTime: scalar(100, 'GameTime_t'),
    m_flRestartRoundTime: scalar(0, 'GameTime_t'),
  });
  const server = 6400 + tick - START;
  const message = { order: 0, frame: tick, tick: 0,
    value: { status: 'observed', value: server, native_type: 'uint32', declared_type: null } };
  return {
    source: 'demoinfocs_native_rules_and_current_network_messages', rules,
    server_tick: server, interval_seconds: 1 / RATE,
    current_frame: tick, freeze_frame: START,
    tick_message: structuredClone(message), packet_message: structuredClone(message),
    origin_witness: { frame: START, tick: START, freeze_end_tick: START,
      server_tick: 6400, packet_tick: 6400, interval_seconds: 1 / RATE,
      round_start_seconds: 100, pause_state: 'running', phase: 'preplant' },
    native_now_seconds: server / RATE, origin_accredited: true, engine_paused: null,
    bombs: phase === 'planted' ? [{ m_bBombTicking: scalar(true, 'bool'),
      m_bBombDefused: scalar(false, 'bool'), m_flC4Blow: scalar(140, 'GameTime_t') }] : null,
  };
}

function sampled(tick = 104, phase = 'preplant') {
  const evidence = clockEvidence(tick, phase);
  const carries = phase === 'preplant';
  const weapons = carries ? ['Glock-18', 'C4', 'Knife'] : ['Glock-18', 'Knife'];
  return {
    tick, time_remaining: 777,
    players: [{ steam_id: '1', team: 'T', alive: true, x: 0, y: 0, z: 0,
      health: 100, armor: 0, money: 800,
      weapon: 'Glock-18', weapons, has_c4: carries, has_helmet: false,
      inventory_context: { schema_id: 'stratai.replay_inventory@1', status: 'observed',
        observation_tick: tick, availability_tick: tick,
        source: { kind: 'demo_inventory_snapshot', reference: 'frame:' + tick + ':inventory' },
        weapon: 'Glock-18', weapons: structuredClone(weapons), has_helmet: false } }],
    bomb: { state: phase === 'ended' ? 'resolved' : carries ? 'carried' : 'planted',
      objective_phase: phase === 'ended' ? 'resolved' : phase,
      is_planted_now: phase === 'planted', was_planted_this_round: phase === 'planted',
      position_status: 'unavailable', observation_tick: tick, availability_tick: tick,
      ...(carries ? { carrier_id: '1' } : {}),
      source: { kind: 'demo_objective_state', reference: 'round:1:frame:' + tick + ':bomb' } },
    round_context: { schema_id: ROUND_SCHEMA, round: 1, tick, availability_tick: tick,
      phase, freeze_end_tick: START, elapsed_seconds: (tick - START) / RATE,
      end_tick: phase === 'ended' ? tick : null, winner: phase === 'ended' ? 'T' : '',
      configured_duration_seconds: 115, pause_state: 'running',
      clock_status: phase === 'ended' ? 'not_applicable' : 'observed',
      clock_reason: phase === 'ended' ? 'round_ended' : '',
      round_remaining_seconds: phase === 'preplant' ? 215 - evidence.native_now_seconds : null,
      bomb_remaining_seconds: phase === 'planted' ? 140 - evidence.native_now_seconds : null,
      clock_evidence: evidence },
  };
}

function round(frames = [sampled(100), sampled(104)]) {
  return { round: 1, start_tick: START, end_tick: 300, winner: 'CT', frames, events: [],
    event_context: { schema_id: 'stratai.replay_events@1', events: [] } };
}

function project(frame = sampled(), tick = frame.tick) {
  return projectReplayRoundContext(round([frame]), frame, [], frame.players, frame.bomb, tick);
}

function expectValidControl() {
  const value = project();
  expect(value.schema_id).toBe(ROUND_SCHEMA);
  expect(value.phase).toBe('preplant');
  expect(value.round_remaining_seconds).toBe(114.9375);
}

describe('strict factual round context @2', () => {
  it('retains causal clock evidence and selects projection @8 without changing legacy opt-in', () => {
    const frame = sampled();
    const value = project(frame);
    expect(value.schema_id).toBe(ROUND_SCHEMA);
    expect(value.round_remaining_seconds).toBe(114.9375);
    expect(value.clock_evidence).toEqual(frame.round_context.clock_evidence);
    expect(createTeamReplayProjector(round(), 'T', null, source)(104).schema_id).toBe(FACTUAL_PROJECTION);
  });

  it('keeps round and bomb countdowns distinct', () => {
    expectValidControl();
    const value = project(sampled(104, 'planted'));
    expect(value.phase).toBe('planted');
    expect(value.round_remaining_seconds).toBeNull();
    expect(value.bomb_remaining_seconds).toBe(39.9375);
  });

  it.each([999, 114])('does not expose claimed countdown %s when native evidence disagrees', remaining => {
    expectValidControl();
    const frame = sampled();
    frame.round_context.round_remaining_seconds = remaining;
    const value = project(frame);
    // Exact recomputation or abstention is safe; the false claim must not survive.
    expect([null, 114.9375]).toContain(value.round_remaining_seconds);
  });

  it('does not expose a negative configured duration or active countdown', () => {
    expectValidControl();
    const frame = sampled();
    frame.round_context.configured_duration_seconds = -10;
    const value = project(frame);
    expect(value.configured_duration_seconds === null || value.configured_duration_seconds >= 0).toBe(true);
    expect(value.round_remaining_seconds).toBeNull();
  });

  it('withholds a winner when its end tick is still in the future', () => {
    expectValidControl();
    const frame = sampled(104, 'ended');
    frame.round_context.end_tick = 108;
    const value = project(frame);
    expect(value.end_tick).toBeNull();
    expect(value.winner).toBe('');
    expect(value.decision_status).not.toBe('round_closed');
  });

  it('requires a dated end before representing the round as closed', () => {
    expectValidControl();
    const frame = sampled(104, 'ended');
    frame.round_context.end_tick = null;
    frame.round_context.winner = '';
    const value = project(frame);
    expect(value.phase).not.toBe('ended');
    expect(value.decision_status).not.toBe('round_closed');
  });

  it('accepts a properly dated terminal outcome without an active clock', () => {
    expectValidControl();
    const value = project(sampled(104, 'ended'));
    expect(value).toMatchObject({ phase: 'ended', winner: 'T', end_tick: 104,
      decision_status: 'round_closed', round_remaining_seconds: null, bomb_remaining_seconds: null });
  });

  it('does not declare a planted phase against a same-tick carried objective', () => {
    expectValidControl();
    const frame = sampled(104, 'planted');
    frame.bomb = sampled(104).bomb;
    frame.players = sampled(104).players;
    const value = project(frame);
    expect(value.phase).not.toBe('planted');
    expect(value.bomb_remaining_seconds).toBeNull();
  });

  it.each([
    ['packet tick mismatch', evidence => { evidence.packet_message.value.value = 6405; }],
    ['historical packet', evidence => { evidence.packet_message.frame = 103; }],
    ['unaligned clock', evidence => { evidence.origin_accredited = false; }],
    ['wrong native time', evidence => { evidence.native_now_seconds = 100; }],
    ['wrong origin witness', evidence => { evidence.origin_witness.packet_tick = 6401; }],
    ['unaccredited native type', evidence => { evidence.rules.m_fRoundStartTime.declared_type = 'float32'; }],
    ['native pause', evidence => { evidence.rules.m_bGamePaused.value = true; }],
  ])('withholds countdown for %s', (_name, change) => {
    expectValidControl();
    const frame = sampled();
    change(frame.round_context.clock_evidence);
    const value = project(frame);
    expect(value.round_remaining_seconds).toBeNull();
    expect(value.bomb_remaining_seconds).toBeNull();
  });

  it('does not interpolate a clock or promote later winner between source samples', () => {
    expectValidControl();
    const context = project(sampled(104), 105);
    expect(context.observation_tick).toBe(104);
    expect(context.clock_status).toBe('unavailable');
    expect(context.round_remaining_seconds).toBeNull();
    expect(context.winner).toBe('');
  });

  it('keeps native @1 on its historic context and projection contracts', () => {
    const data = round();
    for (const frame of data.frames) {
      frame.round_context.schema_id = 'stratai.replay_round_context@1';
      delete frame.round_context.clock_evidence;
    }
    const frame = data.frames.at(-1);
    const context = projectReplayRoundContext(data, frame, [], frame.players, frame.bomb, frame.tick);
    expect(context.schema_id).toBe('stratai.replay_round_context@1');
    expect(context.round_remaining_seconds).toBe(114.9375);
    expect(context.clock_evidence).toBeUndefined();
    expect(createTeamReplayProjector(data, 'T', null, source)(104).schema_id).toBe('stratai.team_replay_projection@7');
  });

  it('keeps the public bomb timer while redacting private defuse activity', () => {
    const frame = sampled(104, 'planted');
    const bomb = frame.round_context.clock_evidence.bombs[0];
    const publicFields = structuredClone(bomb);
    Object.assign(bomb, { m_hBombDefuser: scalar(9001, 'int32'),
      m_bBeingDefused: scalar(true, 'bool'), m_flDefuseCountDown: scalar(105, 'GameTime_t'),
      m_flDefuseLength: scalar(5, 'float32') });
    const value = project(frame);
    expect(value.bomb_remaining_seconds).toBe(39.9375);
    expect(value.clock_evidence.bombs).toEqual([publicFields]);
    expect(frame.round_context.clock_evidence.bombs[0].m_hBombDefuser.value).toBe(9001);
  });

  it.each([
    ['top level', e => { e.hidden_enemy = { steam_id: '9001', x: 3141 }; }],
    ['rules', e => { e.rules.m_hBombDefuser = scalar(9001, 'int32'); }],
    ['origin witness', e => { e.origin_witness.hidden_enemy = { steam_id: '9001' }; }],
    ['network message', e => { e.packet_message.hidden_enemy = { steam_id: '9001' }; }],
    ['native scalar', e => { e.rules.m_iRoundTime.hidden_enemy = { steam_id: '9001' }; }],
    ['native type payload', e => { e.rules.m_iRoundTime.native_type = { hidden_enemy: '9001' }; }],
    ['message order payload', e => { e.packet_message.order = { hidden_enemy: '9001' }; }],
    ['message tick payload', e => { e.packet_message.tick = { hidden_enemy: '9001' }; }],
  ])('does not forward unapproved %s clock fields into the team projection', (_name, change) => {
    expectValidControl();
    const frame = sampled();
    change(frame.round_context.clock_evidence);
    const value = project(frame);
    expect(value.round_remaining_seconds).toBe(114.9375);
    expect(JSON.stringify(value.clock_evidence)).not.toContain('9001');
  });

  it('keeps a terminal prefix stable and rejects reopening only when that cut arrives', () => {
    const initial = sampled(100);
    const ended = sampled(104, 'ended');
    const futureReopened = sampled(108);
    const prefix = createTeamReplayProjector(round([initial, ended]), 'T', null, source);
    const full = createTeamReplayProjector(round([initial, ended, futureReopened]), 'T', null, source);
    const before = full(104);
    expect(before).toEqual(prefix(104));
    expect(before.frame.round_context).toMatchObject({ phase: 'ended', winner: 'T', end_tick: 104 });
    expect(() => full(108)).toThrow(/Terminal round reopened/);
    expect(full(104)).toEqual(before);
  });

  it.each(['end_tick', 'winner'])('rejects a changed terminal %s without poisoning the earlier prefix', field => {
    const initial = sampled(100);
    const ended = sampled(104, 'ended');
    const later = sampled(108, 'ended');
    later.round_context.end_tick = 104;
    later.round_context[field] = field === 'end_tick' ? 108 : 'CT';
    const full = createTeamReplayProjector(round([initial, ended, later]), 'T', null, source);
    const before = full(104);
    expect(() => full(108)).toThrow(/outcome changed/);
    expect(full(104)).toEqual(before);
  });
});
