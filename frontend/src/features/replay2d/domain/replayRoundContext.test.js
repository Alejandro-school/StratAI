import { describe, expect, it } from 'vitest';
import { createTeamReplayProjector } from './teamReplayProjection';

const source = { map_name: 'de_mirage', tick_rate: 64 };
const own = { steam_id: '1', team: 'T', alive: true, x: 0, y: 0, z: 0, weapon: 'knife' };
const native = (tick, phase = 'preplant') => ({ schema_id: 'stratai.replay_round_context@1', round: 1,
  tick, availability_tick: tick, phase, freeze_end_tick: 100, elapsed_seconds: (tick - 100) / 64,
  end_tick: phase === 'ended' ? 104 : null, winner: phase === 'ended' ? 'T' : '',
  configured_duration_seconds: 115, pause_state: 'unknown', clock_status: 'unavailable',
  clock_reason: 'native_pause_controls_incomplete', round_remaining_seconds: null, bomb_remaining_seconds: null });
const data = () => ({ round: 1, start_tick: 100, end_tick: 180, winner: 'T', events: [],
  frames: [{ tick: 100, time_remaining: 115, players: [own], round_context: native(100) },
    { tick: 108, time_remaining: 0, players: [own], round_context: native(108, 'ended') }] });

describe('round context in causal team view', () => {
  it('uses the current phase and does not reveal the eventual winner', () => {
    const project = createTeamReplayProjector(data(), 'T', null, source);
    const early = project(100);
    expect(early.schema_id).toBe('stratai.team_replay_projection@6');
    expect(early.frame.round_context).toMatchObject({ phase: 'preplant', winner: '', end_tick: null });
    expect(project(108).frame.round_context).toMatchObject({ phase: 'ended', winner: 'T', end_tick: 104 });
    expect(early.frame.time_remaining).toBeNull();
  });
  it('does not infer knife eligibility from the active weapon or a future C4', () => {
    const round = data();
    round.events.push({ id: 'c4', type: 'bomb_pickup', tick: 108, player_id: '1', player_team: 'T' });
    expect(createTeamReplayProjector(round, 'T', null, source)(100).frame.round_context.eligibility.kind).toBe('unknown');
  });
  it('rejects wrong-round and future source cuts instead of trusting the legacy timer', () => {
    for (const change of [{ round: 2 }, { availability_tick: 101 }, { tick: 99 }]) {
      const round = data(); Object.assign(round.frames[0].round_context, change);
      const view = createTeamReplayProjector(round, 'T', null, source)(100);
      expect(view.frame.round_context.phase).toBe('unknown');
      expect(view.frame.time_remaining).toBeNull();
    }
  });
  it('does not extrapolate a sampled countdown across missing frames', () => {
    const round = data(); Object.assign(round.frames[0].round_context, {
      pause_state: 'running', clock_status: 'observed', clock_reason: '', round_remaining_seconds: 115 });
    const view = createTeamReplayProjector(round, 'T', null, source)(105);
    expect(view.frame.round_context.round_remaining_seconds).toBeNull();
    expect(view.frame.round_context.clock_status).toBe('unavailable');
  });
});
