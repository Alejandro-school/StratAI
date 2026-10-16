import { describe, expect, it } from 'vitest';
import { createTeamReplayProjector } from './teamReplayProjection';

const source = { map_name: 'de_mirage', tick_rate: 64 };

function fixture() {
  const inventory = {
    schema_id: 'stratai.replay_inventory@1', status: 'observed',
    observation_tick: 104, availability_tick: 104,
    source: { kind: 'demo_inventory_snapshot', reference: 'frame:104:inventory' },
    weapon: 'Glock-18', weapons: ['Glock-18', 'Knife'], has_helmet: true,
  };
  const snapshot = tick => ({
    tick, time_remaining: null,
    players: [{ steam_id: '1', team: 'T', alive: tick < 106, health: tick < 106 ? 100 : 0,
      x: -1500, y: 750, z: -40, armor: 100, money: 800, has_c4: false,
      weapon: 'Glock-18', weapons: ['Glock-18', 'Knife'], has_helmet: true,
      inventory_context: { ...structuredClone(inventory), status: tick < 106 ? 'observed' : 'last_observed' } }],
    bomb: { state: 'resolved', objective_phase: 'resolved', position_status: 'unavailable' },
    round_context: {
      schema_id: 'stratai.replay_round_context@2', round: 1, tick, availability_tick: tick,
      phase: 'ended', end_tick: 104, winner: 'T', freeze_end_tick: 100,
      elapsed_seconds: (tick - 100) / 64, configured_duration_seconds: 115,
      pause_state: 'unknown', clock_status: 'not_applicable', clock_reason: 'round_ended',
      round_remaining_seconds: null, bomb_remaining_seconds: null, clock_evidence: {},
    },
  });
  return {
    round: 1, start_tick: 100, end_tick: 120, frames: [snapshot(104), snapshot(108)],
    events: [{ id: 'kill-ally', type: 'kill', tick: 106, victim_id: '1', victim_team: 'T',
      killer_id: '2', killer_team: 'CT', weapon: 'AK-47' }],
    event_context: { schema_id: 'stratai.replay_events@1', events: [] },
  };
}

describe('factual life event between replay samples', () => {
  it('dates death at the callback while preserving the previous inventory and position dates', () => {
    const round = fixture();
    const project = createTeamReplayProjector(round, 'T', null, source);
    const before = project(105).frame.players[0];
    expect(before.alive).toBe(true);
    expect(before.health).toBe(100);
    expect(before.life_context).toBeUndefined();
    expect(before.inventory_context.status).toBe('observed');

    const view = project(106), player = view.frame.players[0];
    const kill = view.events.find(event => event.type === 'kill');
    expect(view.schema_id).toBe('stratai.team_replay_projection@8');
    expect(player.alive).toBe(false);
    expect(player.health).toBe(0);
    expect(player.observation_tick).toBe(104);
    expect(player.availability_tick).toBe(104);
    expect(player.source).toEqual({ kind: 'demo_replay_frame', reference: 'round:1:frame:104:player:1' });
    expect(player.life_context).toEqual({
      schema_id: 'stratai.replay_life_context@1', status: 'observed', alive: false, health: 0,
      observation_tick: 106, availability_tick: 106,
      source: { kind: 'demo_kill_event', reference: kill.id },
    });
    expect(player.inventory_context).toEqual({ ...round.frames[0].players[0].inventory_context, status: 'last_observed' });
    expect(player.inventory_context.source.reference).toBe('frame:104:inventory');
  });

  it('does not mutate the raw snapshot or invent a death on reverse seek', () => {
    const round = fixture(), original = structuredClone(round);
    const project = createTeamReplayProjector(round, 'T', null, source);
    const after = project(107);
    expect(round).toEqual(original);
    const earlier = project(105).frame.players[0];
    expect(earlier.alive).toBe(true);
    expect(earlier.health).toBe(100);
    expect(earlier.life_context).toBeUndefined();
    expect(earlier.inventory_context.status).toBe('observed');
    expect(after.frame.players[0].inventory_context.status).toBe('last_observed');
    expect(project(107)).toEqual(after);
  });

  it('uses a later native dead snapshot without falsely dating its old inventory again', () => {
    const player = createTeamReplayProjector(fixture(), 'T', null, source)(108).frame.players[0];
    expect(player.alive).toBe(false);
    expect(player.health).toBe(0);
    expect(player.observation_tick).toBe(108);
    expect(player.life_context).toBeUndefined();
    expect(player.inventory_context.observation_tick).toBe(104);
    expect(player.inventory_context.availability_tick).toBe(104);
    expect(player.inventory_context.status).toBe('last_observed');
  });

  it('does not use future or previous round deaths to refresh factual life', () => {
    const round = fixture();
    round.events.push({ id: 'old-instance', type: 'kill', tick: 99, victim_id: '1', victim_team: 'T' });
    const player = createTeamReplayProjector(round, 'T', null, source)(105).frame.players[0];
    expect(player.alive).toBe(true);
    expect(player.health).toBe(100);
    expect(player.life_context).toBeUndefined();
    expect(player.inventory_context.status).toBe('observed');
  });
});
