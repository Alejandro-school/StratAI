"""Validate the versioned round facts before exposing them to interpretation."""
from copy import deepcopy
import math
from ai_coach.support.anchors import require

def checked_round_context(view, rate, *, strict=False):
    r = view['frame'].get('round_context')
    require(isinstance(r, dict) and r.get('schema_id') == ('stratai.replay_round_context@2' if strict else 'stratai.replay_round_context@1'), 'round source absent')
    observed, cut = r.get('observation_tick'), view['tick']
    require(type(observed) is int and observed == view['availability_tick']
            and observed == r.get('availability_tick') and observed <= cut, 'future round context')
    require(r.get('round') == view.get('round'), 'round identity differs')
    require(bool(r.get('source', {}).get('reference')), 'round source absent')
    phase = r.get('phase')
    require(phase in ('unknown','freezetime','preplant','planted','ended'), 'invalid round phase')
    end, winner = r.get('end_tick'), r.get('winner')
    require(end is None or (phase == 'ended' and type(end) is int and 0 < end <= observed), 'future round outcome')
    require(winner in ('','T','CT') and (not winner or phase == 'ended' and end is not None), 'winner before round end')
    require(r.get('decision_status') == ('round_closed' if phase == 'ended' else 'active_round'
            if phase in ('preplant','planted') else 'not_established'), 'round decision state differs')
    elapsed, origin = r.get('elapsed_seconds'), r.get('freeze_end_tick')
    require(elapsed is None or (type(origin) is int and 0 < origin <= observed
            and elapsed == (observed-origin)/rate), 'elapsed origin differs')
    require(r.get('pause_state') in ('unknown','paused','running'), 'invalid pause status')
    require(r.get('clock_status') in ('observed','unavailable','not_applicable'), 'invalid clock status')
    for key, required_phase in [('round_remaining_seconds','preplant'),('bomb_remaining_seconds','planted')]:
        value = r.get(key)
        require(value is None or (type(value) in (float,int) and math.isfinite(value) and value >= 0
                and phase == required_phase and r['clock_status'] == 'observed'
                and r['pause_state'] == 'running' and observed == cut), 'unaccredited countdown')
    gate = r.get('eligibility', {})
    require(gate.get('kind') in ('bomb_objective','unknown'), 'unsupported round classification')
    if gate['kind'] == 'bomb_objective':
        require(type(gate.get('basis_tick')) is int and gate['basis_tick'] <= cut, 'future eligibility')
        require(view['frame'].get('bomb') or any(e['type'].startswith('bomb_') for e in view.get('events',[]))
                or any(p.get('has_c4') for p in view['frame']['players'] if p['team'] == 'T'), 'objective eligibility without evidence')
    if strict:
        from ai_coach.interpretation.replay_native_clock import checked_strict_round
        checked_strict_round(view, r)
    return deepcopy(r)
