// Native countdown proof shared by the round projection and its factual input.
// Missing evidence causes abstention; no interpolation or guessed clock origin.
const finite = n => typeof n === 'number' && Number.isFinite(n);
const pauseFields = ['m_bFreezePeriod', 'm_bWarmupPeriod', 'm_bMatchWaitingForResume',
  'm_bTerroristTimeOutActive', 'm_bCTTimeOutActive', 'm_bGameRestart', 'm_bTechnicalTimeOut'];
const boolean = row => row?.status !== 'observed' ? null : typeof row.value === 'boolean' ? row.value
  : row.declared_type === 'bool' && [0, 1].includes(row.value) ? Boolean(row.value) : null;
function scalar(row, gameTime = false) {
  if (row?.status !== 'observed' || !finite(row.value) || row.value < 0 || row.value > 1e8
    || (gameTime && row.declared_type !== 'GameTime_t')) throw new Error('invalid native clock scalar');
  return row.value;
}
export function clockPause(evidence) {
  const rules = evidence?.rules || {}, fields = pauseFields.map(name => boolean(rules[name]));
  const general = boolean(rules.m_bGamePaused), engine = evidence?.engine_paused;
  if (engine != null && typeof engine !== 'boolean') return 'unknown';
  if ([...fields, general, engine].includes(true)) return 'paused';
  if (fields.some(value => value === null) || (general === null && typeof engine !== 'boolean')) return 'unknown';
  return 'running';
}
export function nativeCountdown(context) {
  try {
    const e = context.clock_evidence, rules = e.rules, w = e.origin_witness;
    if (e.source !== 'demoinfocs_native_rules_and_current_network_messages' || clockPause(e) !== 'running') return null;
    const t = e.tick_message, p = e.packet_message, frame = e.current_frame;
    if (!Number.isInteger(frame) || frame < 0 || !Number.isInteger(t.frame) || !Number.isInteger(p.frame) || t.frame !== frame || p.frame !== frame) return null;
    const server = scalar(t.value), step = e.interval_seconds;
    if (server !== scalar(p.value) || server !== e.server_tick || !finite(step) || step < 1/256 || step > 1/16) return null;
    const now = server * step, origin = scalar(rules.m_fRoundStartTime, true);
    if (!Number.isInteger(context.freeze_end_tick) || context.freeze_end_tick <= 0 || context.freeze_end_tick > context.tick
      || !Number.isInteger(e.freeze_frame) || e.freeze_frame < 0 || e.freeze_frame > frame || !Number.isInteger(w?.frame)) return null;
    if (e.native_now_seconds !== now || e.origin_accredited !== true || !w
      || !finite(w.server_tick) || w.server_tick !== w.packet_tick || w.server_tick > server || w.interval_seconds !== step
      || w.server_tick * step !== w.round_start_seconds || w.round_start_seconds !== origin
      || w.tick !== w.freeze_end_tick || w.tick !== context.freeze_end_tick
      || w.frame !== e.freeze_frame || w.frame > frame || w.pause_state !== 'running' || w.phase !== 'preplant') return null;
    const duration = scalar(rules.m_iRoundTime);
    if (duration !== context.configured_duration_seconds || boolean(rules.m_bBombPlanted) !== (context.phase === 'planted')) return null;
    let deadline;
    if (context.phase === 'preplant') {
      if (origin > now) return null;
      deadline = origin + duration;
    } else if (context.phase === 'planted') {
      const active = (e.bombs || []).filter(b => boolean(b.m_bBombTicking) === true && boolean(b.m_bBombDefused) === false);
      if (active.length !== 1) return null;
      deadline = scalar(active[0].m_flC4Blow, true);
    } else return null;
    return Math.max(0, deadline - now);
  } catch { return null; }
}
export function publicClockEvidence(raw) {
  if (!raw || typeof raw !== 'object') return {};
  // Defuse progress/duration is hidden opponent activity, not a public clock.
  const pick = (value, keys) => value && typeof value === 'object' ? Object.fromEntries(
    keys.filter(key => key in value).map(key => [key, value[key]])) : value;
  const leaves = value => value && typeof value === 'object' ? Object.fromEntries(Object.entries(value)
    .filter(([, item]) => item === null || ['string', 'boolean'].includes(typeof item) || finite(item))) : value;
  const field = value => leaves(pick(value, ['status', 'native_type', 'declared_type', 'value']));
  const result = pick(raw, ['source', 'rules', 'server_tick', 'interval_seconds', 'current_frame', 'freeze_frame',
    'tick_message', 'packet_message', 'origin_witness', 'native_now_seconds', 'origin_accredited', 'engine_paused', 'bombs']);
  for (const key of ['source', 'server_tick', 'interval_seconds', 'current_frame', 'freeze_frame',
    'native_now_seconds', 'origin_accredited', 'engine_paused']) {
    if (result[key] !== null && typeof result[key] === 'object') delete result[key];
  }
  if (result.rules) result.rules = Object.fromEntries(Object.entries(pick(result.rules,
    [...pauseFields, 'm_bGamePaused', 'm_bBombPlanted', 'm_iRoundTime', 'm_fRoundStartTime', 'm_flRestartRoundTime']))
    .map(([key, value]) => [key, field(value)]));
  for (const key of ['tick_message', 'packet_message']) if (result[key]) {
    const value = field(result[key].value);
    result[key] = { ...leaves(pick(result[key], ['order', 'frame', 'tick'])), value };
  }
  if (result.origin_witness) result.origin_witness = leaves(pick(result.origin_witness,
    ['frame', 'tick', 'freeze_end_tick', 'server_tick', 'packet_tick', 'interval_seconds', 'round_start_seconds', 'pause_state', 'phase']));
  if (Array.isArray(result.bombs)) result.bombs = result.bombs.map(b => Object.fromEntries(
    Object.entries(pick(b, ['m_bBombTicking', 'm_bBombDefused', 'm_flC4Blow'])).map(([key, value]) => [key, field(value)])));
  return result;
}

