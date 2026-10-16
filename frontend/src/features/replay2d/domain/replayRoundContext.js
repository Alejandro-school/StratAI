export const ROUND_TEAM_REPLAY_SCHEMA = 'stratai.team_replay_projection@6';
const nativeSchema = 'stratai.replay_round_context@1';
const finite = value => typeof value === 'number' && Number.isFinite(value);

// Round identity and outcome come from the dated frame, never the final round
// summary. Legacy time_remaining mixes defaults and clock kinds, so it cannot
// repair a missing native observation.
export function projectReplayRoundContext(round, sampled, events, ownPlayers, bomb, tick) {
  const raw = sampled?.round_context;
  const strict = raw?.schema_id === 'stratai.replay_round_context@2';
  const valid = (raw?.schema_id === nativeSchema || strict) && raw.round === round.round
    && raw.tick === sampled.tick && raw.availability_tick === raw.tick && raw.tick <= tick;
  const current = valid && raw.tick === tick;
  const allowedPhases = ['freezetime', 'preplant', 'planted', 'ended'];
  let phase = valid && allowedPhases.includes(raw.phase) ? raw.phase : 'unknown';
  if (strict && phase === 'ended' && (!Number.isInteger(raw.end_tick) || raw.end_tick <= 0 || raw.end_tick > raw.tick)) phase = 'unknown';
  if (strict && bomb?.observation_tick === raw.tick && (
    (phase === 'planted' && (bomb.objective_phase !== 'planted' || !['planted','defusing'].includes(bomb.state) || bomb.is_planted_now !== true))
    || (['preplant','freezetime'].includes(phase) && ['planted','resolved'].includes(bomb.objective_phase))
    || (phase === 'ended' && bomb.objective_phase !== 'resolved'))) phase = 'unknown';
  const result = {
    schema_id: strict ? 'stratai.replay_round_context@2' : nativeSchema, round: round.round, observation_tick: sampled?.tick ?? null,
    availability_tick: sampled?.tick ?? null, phase,
    source: { kind: 'demo_round_context', reference: `round:${round.round}:frame:${sampled?.tick}:round_context` },
    freeze_end_tick: valid ? raw.freeze_end_tick : null,
    elapsed_seconds: valid && finite(raw.elapsed_seconds) ? raw.elapsed_seconds : null,
    elapsed_kind: 'demo_ticks_since_freeze_end_not_active_game_clock',
    end_tick: phase === 'ended' && finite(raw.end_tick) && raw.end_tick <= raw.tick ? raw.end_tick : null,
    winner: phase === 'ended' && ['T','CT'].includes(raw.winner) ? raw.winner : '',
    configured_duration_seconds: valid && finite(raw.configured_duration_seconds) ? raw.configured_duration_seconds : null,
    pause_state: current && ['running','paused'].includes(raw.pause_state) ? raw.pause_state : 'unknown',
    clock_status: current && ['observed','not_applicable'].includes(raw.clock_status) ? raw.clock_status : 'unavailable',
    clock_reason: current ? raw.clock_reason : valid ? 'sample_not_current' : 'round_source_unavailable',
    round_remaining_seconds: current && raw.clock_status === 'observed' && phase === 'preplant'
      && finite(raw.round_remaining_seconds) ? raw.round_remaining_seconds : null,
    bomb_remaining_seconds: current && raw.clock_status === 'observed' && phase === 'planted'
      && finite(raw.bomb_remaining_seconds) ? raw.bomb_remaining_seconds : null,
  };
  if (strict) {
    result.clock_evidence = publicClockEvidence(raw.clock_evidence);
    if (!finite(result.configured_duration_seconds) || result.configured_duration_seconds < 0) result.configured_duration_seconds = null;
    const expected = current && phase === raw.phase ? nativeCountdown(raw) : null;
    const claimed = phase === 'preplant' ? raw.round_remaining_seconds : raw.bomb_remaining_seconds;
    const verified = expected !== null && claimed === expected && raw.pause_state === 'running' && raw.clock_status === 'observed';
    result.pause_state = current ? clockPause(raw.clock_evidence) : 'unknown';
    result.round_remaining_seconds = verified && phase === 'preplant' ? expected : null;
    result.bomb_remaining_seconds = verified && phase === 'planted' ? expected : null;
    result.clock_status = phase === 'ended' && current ? 'not_applicable' : verified ? 'observed' : 'unavailable';
    result.clock_reason = verified ? '' : phase === 'ended' && current ? 'round_ended' : current ? 'native_clock_not_verified' : 'sample_not_current';
  }
  const objectiveEvent = events.find(e => e.type.startsWith('bomb_') && e.tick <= tick);
  const ownC4 = ownPlayers.some(p => p.alive && p.has_c4);
  result.eligibility = bomb || objectiveEvent || ownC4
    ? { kind: 'bomb_objective', status: 'observed', reason: 'causal_objective_evidence',
      basis_tick: objectiveEvent?.tick ?? sampled.tick }
    : { kind: 'unknown', status: 'unknown', reason: 'round_type_not_accredited' };
  result.decision_status = phase === 'ended' ? 'round_closed' : ['preplant','planted'].includes(phase)
    ? 'active_round' : 'not_established';
  return result;
}
import { nativeCountdown, clockPause, publicClockEvidence } from './replayNativeClock';
