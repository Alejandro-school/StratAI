"""Recompute supplied native clock evidence; no defaults or fitted time origin."""
import math

from ai_coach.support.anchors import require

PAUSE_FIELDS = ("m_bFreezePeriod", "m_bWarmupPeriod", "m_bMatchWaitingForResume",
                "m_bTerroristTimeOutActive", "m_bCTTimeOutActive", "m_bGameRestart", "m_bTechnicalTimeOut")


def checked_evidence_fields(evidence):
    def closed(value, keys):
        require(isinstance(value, dict) and set(value) <= set(keys), 'unsupported clock evidence field')
    def field(value):
        closed(value, ('status','native_type','declared_type','value'))
        require(value.get('status') in ('observed','absent','nonfinite','unsupported_type'), 'invalid scalar status')
        require(all(value.get(key) is None or isinstance(value[key], str) and len(value[key]) <= 80
                    for key in ('native_type','declared_type')), 'invalid scalar type metadata')
        require(value.get('value') is None or type(value['value']) in (bool, int, float), 'invalid clock scalar payload')
    closed(evidence, ('source','rules','server_tick','interval_seconds','current_frame','freeze_frame',
                     'tick_message','packet_message','origin_witness','native_now_seconds','origin_accredited','engine_paused','bombs'))
    for key in ('server_tick','interval_seconds','native_now_seconds'):
        require(evidence.get(key) is None or type(evidence[key]) in (int,float) and math.isfinite(evidence[key]), 'invalid clock metadata number')
    for key in ('current_frame','freeze_frame'):
        require(key not in evidence or type(evidence[key]) is int, 'invalid clock metadata frame')
    for key in ('origin_accredited','engine_paused'):
        require(evidence.get(key) is None or type(evidence[key]) is bool, 'invalid clock metadata flag')
    require('source' not in evidence or evidence['source'] == 'demoinfocs_native_rules_and_current_network_messages', 'invalid clock metadata source')
    rules = evidence.get('rules', {})
    closed(rules, (*PAUSE_FIELDS,'m_bGamePaused','m_bBombPlanted','m_iRoundTime','m_fRoundStartTime','m_flRestartRoundTime'))
    for value in rules.values():
        field(value)
    for key in ('tick_message','packet_message'):
        message = evidence.get(key)
        if message is not None:
            closed(message, ('order','frame','tick','value'))
            require(all(key not in message or type(message[key]) is int for key in ('order','frame','tick')), 'invalid network message metadata')
            field(message.get('value'))
    witness = evidence.get('origin_witness')
    if witness is not None:
        closed(witness, ('frame','tick','freeze_end_tick','server_tick','packet_tick','interval_seconds','round_start_seconds','pause_state','phase'))
        require(all(key not in witness or type(witness[key]) is int for key in ('frame','tick','freeze_end_tick')), 'invalid witness date')
        require(all(key not in witness or type(witness[key]) in (int,float) and math.isfinite(witness[key])
                    for key in ('server_tick','packet_tick','interval_seconds','round_start_seconds')), 'invalid witness number')
        require(witness.get('pause_state') in ('running','paused','unknown') and witness.get('phase') == 'preplant', 'invalid witness phase')
    require(evidence.get('bombs') is None or isinstance(evidence['bombs'], list), 'invalid native bomb list')
    for bomb in evidence.get('bombs') or []:
        closed(bomb, ('m_bBombTicking','m_bBombDefused','m_flC4Blow'))
        for value in bomb.values():
            field(value)


def scalar(row, game_time=False):
    require(isinstance(row, dict) and row.get("status") == "observed", "native scalar unavailable")
    value = row.get("value")
    require(type(value) in (int, float) and math.isfinite(value) and 0 <= value <= 1e8, "invalid native number")
    require(not game_time or row.get("declared_type") == "GameTime_t", "native time type unaccredited")
    return value


def boolean(row):
    if not isinstance(row, dict) or row.get("status") != "observed":
        return None
    value = row.get("value")
    if type(value) is bool:
        return value
    if row.get("declared_type") == "bool" and type(value) in (int, float) and value in (0, 1):
        return bool(value)
    return None


def countdown(context):
    e = context.get("clock_evidence", {})
    require(e.get("source") == "demoinfocs_native_rules_and_current_network_messages", "clock evidence absent")
    rules, witness = e.get("rules", {}), e.get("origin_witness", {})
    pauses = [boolean(rules.get(key)) for key in PAUSE_FIELDS]
    general, engine = boolean(rules.get("m_bGamePaused")), e.get("engine_paused")
    require(engine is None or type(engine) is bool, 'invalid engine pause control')
    require(all(v is False for v in pauses) and general is not True and engine is not True
            and (general is False or engine is False), "native clock paused or unknown")
    tick, packet = e.get("tick_message", {}), e.get("packet_message", {})
    frame = e.get("current_frame")
    require(type(frame) is int and frame >= 0 and type(tick.get('frame')) is int and type(packet.get('frame')) is int
            and tick.get("frame") == packet.get("frame") == frame, "stale network clock")
    server = scalar(tick.get("value"))
    require(server == scalar(packet.get("value")) == e.get("server_tick"), "network clock disagreement")
    step = e.get("interval_seconds")
    require(type(step) in (int, float) and 1/256 <= step <= 1/16, "invalid native tick interval")
    now, origin = server * step, scalar(rules.get("m_fRoundStartTime"), True)
    require(e.get("native_now_seconds") == now and e.get("origin_accredited") is True, "native now unaccredited")
    require(type(context.get('freeze_end_tick')) is int and 0 < context['freeze_end_tick'] <= context['observation_tick'],
            'future freeze origin')
    require(type(e.get('freeze_frame')) is int and 0 <= e['freeze_frame'] <= frame
            and type(witness.get('frame')) is int, 'invalid freeze frame')
    require(isinstance(witness, dict) and witness.get("server_tick") == witness.get("packet_tick")
            and type(witness.get("server_tick")) in (int, float)
            and witness["server_tick"] <= server and witness.get("interval_seconds") == step
            and witness["server_tick"] * step == witness.get("round_start_seconds") == origin
            and witness.get("tick") == witness.get("freeze_end_tick") == context.get("freeze_end_tick")
            and witness.get("frame") == e.get("freeze_frame") and witness["frame"] <= frame
            and witness.get("pause_state") == "running" and witness.get("phase") == "preplant", "native origin witness differs")
    duration = scalar(rules.get("m_iRoundTime"))
    require(duration == context.get("configured_duration_seconds"), "configured duration differs")
    planted = boolean(rules.get("m_bBombPlanted"))
    require(planted is (context["phase"] == "planted"), "native planted phase differs")
    if context["phase"] == "preplant":
        require(origin <= now, "round origin after native time")
        deadline = origin + duration
    else:
        active = [b for b in e.get("bombs") or [] if boolean(b.get("m_bBombTicking")) is True
                  and boolean(b.get("m_bBombDefused")) is False]
        require(len(active) == 1, "ambiguous native bomb clock")
        deadline = scalar(active[0].get("m_flC4Blow"), True)
    return max(0, deadline - now)


def checked_strict_round(view, context):
    checked_evidence_fields(context.get('clock_evidence', {}))
    observed = context["observation_tick"]
    require(type(context.get("round")) is int and context["round"] > 0, "invalid round identity")
    require(context.get("source") == {"kind": "demo_round_context",
            "reference": f"round:{view['round']}:frame:{observed}:round_context"}, "round source differs")
    duration = context.get("configured_duration_seconds")
    require(duration is None or type(duration) in (int, float) and math.isfinite(duration) and duration >= 0,
            "invalid configured duration")
    phase = context["phase"]
    for bomb_clock in context.get('clock_evidence', {}).get('bombs') or []:
        require(set(bomb_clock) <= {'m_bBombTicking','m_bBombDefused','m_flC4Blow'}, 'hidden bomb clock field')
    require(phase != "ended" or type(context.get("end_tick")) is int and 0 < context["end_tick"] <= observed,
            "round end lacks causal date")
    bomb = view["frame"].get("bomb")
    if bomb and bomb.get("observation_tick") == observed and phase != "unknown":
        objective = bomb.get("objective_phase")
        require(phase != "planted" or objective == "planted" and bomb.get("state") in ("planted", "defusing")
                and bomb.get("is_planted_now") is True, "planted phase conflicts with objective")
        require(phase not in ("preplant", "freezetime") or objective not in ("planted", "resolved"),
                "active phase conflicts with objective")
        require(phase != "ended" or objective == "resolved", "ended phase conflicts with objective")
    clock = context.get("round_remaining_seconds") if phase == "preplant" else context.get("bomb_remaining_seconds")
    if context["clock_status"] == "observed":
        require(phase in ("preplant", "planted") and clock is not None, "observed clock absent")
        require(clock == countdown(context), "countdown differs from native evidence")

