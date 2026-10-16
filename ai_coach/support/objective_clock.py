"""Causal native game clocks with explicit domain, pause and legal-view gates."""

import argparse
import copy
import json
import math
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import c4
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import physical as p

DATA = "support_objective_clock_observations.json"
REPORT = "support_objective_clock_context.json"
RULES = (
    "m_fRoundStartTime",
    "m_iRoundTime",
    "m_flRestartRoundTime",
    "m_bFreezePeriod",
    "m_bWarmupPeriod",
    "m_bMatchWaitingForResume",
    "m_bTerroristTimeOutActive",
    "m_bCTTimeOutActive",
    "m_bTechnicalTimeOut",
    "m_bGamePaused",
    "m_bGameRestart",
)
PAWN = ("m_flSimulationTime", "m_nTickBase")
BOMB = (
    "m_flSimulationTime",
    "m_flC4Blow",
    "m_flTimerLength",
    "m_flDefuseCountDown",
    "m_flDefuseLength",
    "m_bBombTicking",
    "m_bBombDefused",
)
ITEM = ("m_flSimulationTime", "m_bStartedArming", "m_fArmedTime", "m_flArmingLength")
PLAYER = (
    "pause_state",
    "clock_state",
    "round_remaining_seconds",
    "round_clock_state",
    "bomb_remaining_seconds",
    "bomb_clock_state",
    "attempt_state",
    "attempt_remaining_seconds",
    "attempt_progress",
    "defuse_deadline_margin_seconds",
)
FIELDS = tuple(role + "." + key for role in ("self", "ally") for key in PLAYER)
LIMITS = [
    *[x for x in c4.LIMITS if x != "countdown_and_pause_clock_not_accredited"],
    "countdowns_require_current_native_pause_and_clock_alignment",
    "native_clocks_do_not_establish_visible_hud_rounding_or_wall_time",
    "scheduled_attempt_margin_does_not_predict_completion_or_tactical_viability",
]
INTEGERS = {"int", "int32", "int64", "uint32", "uint64"}


def validate_scalar(s):
    status, value, typ = s["status"], s["value"], s["native_type"]
    a.require(
        (status == "observed") == (value is not None), "clock scalar presence mismatch"
    )
    a.require(
        (status == "absent") == (typ is None), "clock scalar type presence mismatch"
    )
    if status == "observed":
        good = (
            (typ == "bool" and type(value) is bool)
            or (typ in INTEGERS and type(value) is int)
            or (typ in ("float32", "float64") and type(value) in (int, float))
        )
        a.require(
            good and (type(value) is bool or math.isfinite(value)),
            "clock scalar decoded type mismatch",
        )
    if status == "nonfinite":
        a.require(typ in ("float32", "float64"), "invalid nonfinite clock type")


def validate_source(data, support, objective, ownership):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            data[key] == support[key] == objective[key] == ownership[key],
            "clock source identity mismatch",
        )
    frames = support["observations"]["frames"]
    objectives = {f["source_sequence"]: f for f in objective["observations"]["frames"]}
    c4cuts = {f["source_sequence"]: f for f in ownership["observations"]["cuts"]}
    states = ownership["observations"]["states"]
    previous, identities, messages = 0, {}, {}
    previous_order = 0
    for row in data["observations"]["frames"]:
        seq = row["source_sequence"]
        a.require(previous < seq <= len(frames), "clock source sequence mismatch")
        previous = seq
        base, objective_cut, c4cut = (
            frames[seq - 1],
            objectives.get(seq),
            c4cuts.get(seq),
        )
        a.require(
            all(row[k] == base[k] for k in ("frame", "tick", "instance", "epoch")),
            "clock exact cut mismatch",
        )
        a.require(
            objective_cut is not None
            and row["objective_order"] == objective_cut["order"],
            "clock objective order mismatch",
        )
        a.require(
            c4cut is not None and row["c4_state_sequence"] == c4cut["state_sequence"],
            "clock ownership frontier mismatch",
        )
        a.require(row["order"] > previous_order, "clock cut order mismatch")
        previous_order = row["order"]
        for kind, message in row["network"].items():
            if message is None:
                continue
            validate_scalar(message["value"])
            a.require(
                message["order"] < row["order"] and message["frame"] <= row["frame"],
                "future clock message",
            )
            a.require(
                message["frame"] != row["frame"] or message["tick"] == row["tick"],
                "clock delivery tick mismatch",
            )
            identity = kind, message["order"]
            a.require(
                messages.setdefault(identity, message) == message,
                "clock message mutated",
            )
        actors = a.actor_index(base)
        aliases = [x["actor_alias"] for x in row["players"]]
        a.require(
            len(aliases) == len(set(aliases)) and set(aliases) == set(actors),
            "clock actor coverage mismatch",
        )
        for player in row["players"]:
            base_player = actors[player["actor_alias"]]
            a.require(
                (player["pawn_life"] is None) == (player["pawn_handle"] is None),
                "clock pawn presence mismatch",
            )
            if player["pawn_life"] is not None:
                a.require(
                    a.usable(base_player)
                    and all(
                        player[k] == base_player[k]
                        for k in ("pawn_life", "pawn_handle")
                    ),
                    "clock pawn identity mismatch",
                )
            for scalar in player["fields"].values():
                validate_scalar(scalar)
                a.require(
                    player["pawn_life"] is not None or scalar["status"] == "absent",
                    "clock fields on unavailable pawn",
                )
        entities = (
            row["bombs"] + row["items"] + ([row["rules"]] if row["rules"] else [])
        )
        a.require(
            len({e["entity_life"] for e in entities}) == len(entities),
            "duplicate clock entity life",
        )
        a.require(
            len({e["handle"] for e in entities}) == len(entities),
            "duplicate clock live handle",
        )
        c4entities = {
            e["entity_life"]: e for e in states[c4cut["state_sequence"] - 1]["entities"]
        }
        for e in entities:
            identity = e["class_name"], e["handle"]
            a.require(e["handle"] & 16383 != 16383, "clock sentinel entity")
            a.require(
                identities.setdefault(e["entity_life"], identity) == identity,
                "clock entity identity changed",
            )
            for scalar in e["fields"].values():
                validate_scalar(scalar)
            if e["class_name"] == "CC4":
                other = c4entities.get(e["entity_life"])
                a.require(
                    other is not None
                    and (other["class_name"], other["handle"]) == identity,
                    "clock item ownership identity mismatch",
                )
            if e["defuser"] is not None:
                c4.validate_reference(e["defuser"])


def native(s, kind="number", game_time=False):
    if s["status"] != "observed":
        return o.fact(reason="native_clock_" + s["status"])
    value, typ = s["value"], s["native_type"]
    valid = (
        typ == "bool"
        if kind == "bool"
        else typ in INTEGERS
        if kind == "integer"
        else typ in ("float32", "float64")
    )
    if not valid:
        return o.fact(reason="native_clock_type_mismatch")
    if kind != "bool" and not 0 <= value <= 1e8:
        return o.fact(reason="native_clock_outside_range")
    if game_time and s["declared_type"] != "GameTime_t":
        return o.fact(reason="native_game_time_domain_unaccredited")
    return o.fact(value)


def pause_state(row):
    rules = row["rules"]
    if rules is None:
        return o.fact(reason="native_rules_life_unavailable")
    fields = rules["fields"]
    required = (
        "m_bFreezePeriod",
        "m_bWarmupPeriod",
        "m_bMatchWaitingForResume",
        "m_bTerroristTimeOutActive",
        "m_bCTTimeOutActive",
        "m_bGameRestart",
        "m_bTechnicalTimeOut",
    )
    flags = [native(fields[k], "bool") for k in required]
    optional = [
        native(fields[k], "bool") for k in ("m_bGamePaused", "m_bTechnicalTimeOut")
    ]
    message = row["network"]["paused"]
    engine = (
        native(message["value"], "bool")
        if message
        else o.fact(reason="general_pause_state_unavailable")
    )
    if any(f["value"] is True for f in [*flags, *optional, engine]):
        return o.fact("paused")
    if any(f["status"] != "observed" for f in flags):
        return o.fact(reason="native_pause_controls_incomplete")
    if engine["value"] is not False and optional[0]["value"] is not False:
        return o.fact(reason="general_pause_state_unavailable")
    return o.fact("running")


def alignment_prefixes(support, objective, data):
    """Bind GameTime_t to server time at a positively observed round origin.

    Never fit an offset from outcomes or use a later origin for earlier cuts.
    Both native server ticks must agree in the exact public freeze-end cut.
    A restart, changed origin, interval/session, life or rewind drops the proof.
    """
    public = o.public_prefixes(objective)
    bases = {r["sequence"]: r for r in support["observations"]["frames"]}
    objectives = {r["source_sequence"]: r for r in objective["observations"]["frames"]}
    result, witnesses = {}, []
    previous_key = previous_tick = witness = None
    for row in data["observations"]["frames"]:
        seq = row["source_sequence"]
        rules, network = row["rules"], row["network"]
        interval = network["interval"]
        start = (
            native(rules["fields"]["m_fRoundStartTime"], game_time=True)
            if rules
            else o.fact(reason="native_rules_life_unavailable")
        )
        step = (
            native(interval["value"])
            if interval
            else o.fact(reason="native_tick_interval_absent")
        )
        ticks = [network[k] for k in ("tick", "packet_tick")]
        valid = (
            start["status"] == step["status"] == "observed"
            and 1 / 256 <= step["value"] <= 1 / 16
        )
        valid = valid and all(
            m is not None
            and m["frame"] == row["frame"]
            and native(m["value"], "integer")["status"] == "observed"
            for m in ticks
        )
        valid = valid and ticks[0]["value"]["value"] == ticks[1]["value"]["value"]
        if not valid:
            previous_key = previous_tick = witness = None
            result[seq] = None
            continue
        tick = ticks[0]["value"]["value"]
        key = (
            row["instance"],
            row["epoch"],
            rules["entity_life"],
            interval["order"],
            step["value"],
            start["value"],
        )
        if key != previous_key or (previous_tick is not None and tick < previous_tick):
            witness = None
        state = public.get(seq, {})
        freeze = state.get("freeze")
        public_plant = o.native(objectives[seq]["rules"]["m_bBombPlanted"])
        if (
            witness is None
            and freeze is not None
            and freeze["frame"] == row["frame"]
            and freeze["tick"] == row["tick"]
            and state.get("phase") == "preplant"
            and public_plant["value"] is False
            and bases[seq]["phase"] == "playing"
            and pause_state(row)["value"] == "running"
            and start["value"] == tick * step["value"]
        ):
            witness = {
                "source_sequence": seq,
                "frame": row["frame"],
                "tick": row["tick"],
                "instance": row["instance"],
                "epoch": row["epoch"],
                "rules_life": rules["entity_life"],
                "interval_order": interval["order"],
                "server_tick": tick,
                "native_interval_seconds": step["value"],
                "native_round_start_seconds": start["value"],
                "public_freeze_event_sequence": freeze["sequence"],
            }
            witnesses.append(witness)
        result[seq] = witness
        previous_key, previous_tick = key, tick
    return result, witnesses


def aligned_now(row, player, witness=None):
    if player is None or player["pawn_life"] is None:
        return o.fact(reason="actor_clock_context_unavailable")
    interval = row["network"]["interval"]
    step = (
        native(interval["value"])
        if interval
        else o.fact(reason="native_tick_interval_absent")
    )
    if witness is not None:
        ticks = [row["network"][k] for k in ("tick", "packet_tick")]
        if (
            all(
                m is not None
                and m["frame"] == row["frame"]
                and native(m["value"], "integer")["status"] == "observed"
                for m in ticks
            )
            and ticks[0]["value"]["value"] == ticks[1]["value"]["value"]
            and step["value"] == witness["native_interval_seconds"]
        ):
            return o.fact(ticks[0]["value"]["value"] * step["value"])
    if step["status"] != "observed" or step["value"] != 1 / 64:
        # The pinned SDK simulationTimeDecoder uses 1/64, not TickRate().
        return o.fact(reason="sdk_simtime_interval_unaccredited")
    clocks = []
    for kind in ("tick", "packet_tick"):
        message = row["network"][kind]
        if message is not None and message["frame"] == row["frame"]:
            value = native(message["value"], "integer")
            if value["status"] == "observed":
                clocks.append(value["value"])
    if not clocks:
        return o.fact(reason="current_native_server_tick_absent")
    if len(set(clocks)) != 1:
        return o.fact(reason="native_server_ticks_disagree")
    simulation = native(player["fields"]["m_flSimulationTime"])
    now = clocks[0] * step["value"]
    if simulation["status"] != "observed" or simulation["value"] != now:
        return o.fact(reason="current_actor_simulation_alignment_unaccredited")
    return o.fact(now)


def remaining(deadline, now):
    if deadline["status"] != "observed":
        return deadline, o.fact(reason=deadline["reason"])
    if now["status"] != "observed":
        return now, o.fact(reason=now["reason"])
    signed = deadline["value"] - now["value"]
    return o.fact(max(0, signed)), o.fact("expired" if signed <= 0 else "active")


def no_attempt(reason="no_current_own_attempt", state="none"):
    return {
        "attempt_state": o.fact(state),
        **{
            k: o.fact(reason=reason, status="not_applicable")
            for k in (
                "attempt_remaining_seconds",
                "attempt_progress",
                "defuse_deadline_margin_seconds",
            )
        },
    }


def attempt(player, objective_player, owned, row, phase, now, pause):
    def unavailable(reason):
        return {
            k: o.fact(reason=reason)
            for k in (
                "attempt_state",
                "attempt_remaining_seconds",
                "attempt_progress",
                "defuse_deadline_margin_seconds",
            )
        }

    if player is None or player["pawn_life"] is None or objective_player is None:
        return unavailable("actor_attempt_context_unavailable")
    defusing = o.native(objective_player["fields"]["m_bIsDefusing"])
    if phase == "planted":
        if defusing["value"] is False:
            if any(
                b["defuser"]["status"] == "resolved"
                and b["defuser"]["target_life"] == player["pawn_life"]
                and b["defuser"]["handle"] == player["pawn_handle"]
                for b in row["bombs"]
            ):
                return unavailable("native_attempt_flag_reference_conflict")
            return no_attempt()
        if defusing["value"] is not True:
            return unavailable("native_attempt_flag_unavailable")
        # Select only the current own/ally defuser. Hidden enemies cannot alter
        # this result by adding an unrelated attempt or deadline.
        candidates = [
            b
            for b in row["bombs"]
            if b["defuser"]["status"] == "resolved"
            and b["defuser"]["target_life"] == player["pawn_life"]
            and b["defuser"]["handle"] == player["pawn_handle"]
        ]
        if len(candidates) != 1:
            return unavailable("native_defuser_identity_uncorroborated")
        fields = candidates[0]["fields"]
        if (
            native(fields["m_bBombTicking"], "bool")["value"] is not True
            or native(fields["m_bBombDefused"], "bool")["value"] is not False
        ):
            return unavailable("native_attempt_objective_state_conflict")
        deadline = native(fields["m_flDefuseCountDown"], game_time=True)
        duration = native(fields["m_flDefuseLength"])
        bomb = native(fields["m_flC4Blow"], game_time=True)
        kind = "defusing"
    elif phase == "preplant":
        if owned is None:
            return unavailable("ownership_context_unavailable")
        has = c4.possession(owned[0], owned[1])
        if has["value"] is False:
            return no_attempt()
        if has["value"] is not True:
            return unavailable("native_planter_ownership_uncorroborated")
        lives = {
            e["entity_life"]
            for e in owned[1]
            if e["owner"]["target_life"] == player["pawn_life"]
            and c4.item_type(e) == "c4"
        }
        items = [item for item in row["items"] if item["entity_life"] in lives]
        if len(items) != 1:
            return unavailable("native_arming_entity_unavailable")
        fields = items[0]["fields"]
        arming = native(fields["m_bStartedArming"], "bool")
        if arming["value"] is False:
            return no_attempt()
        if arming["value"] is not True:
            return unavailable("native_attempt_flag_unavailable")
        deadline = native(fields["m_fArmedTime"], game_time=True)
        duration = native(fields["m_flArmingLength"])
        bomb, kind = None, "planting"
    else:
        return unavailable("objective_phase_unavailable")

    def timing_unavailable(reason):
        return {**unavailable(reason), "attempt_state": o.fact(kind)}

    if pause["status"] != "observed":
        return timing_unavailable(pause["reason"])
    if now["status"] != "observed":
        return timing_unavailable(now["reason"])
    left, status = remaining(deadline, now)
    if left["status"] != "observed":
        return timing_unavailable(left["reason"])
    state = (
        "paused_" + kind
        if pause["value"] == "paused"
        else "expired_" + kind
        if status["value"] == "expired"
        else kind
    )
    progress = o.fact(reason="native_attempt_duration_unavailable")
    if duration["status"] == "observed" and duration["value"] > 0:
        if left["value"] <= duration["value"]:
            progress = o.fact(1 - left["value"] / duration["value"])
        else:
            return unavailable("native_attempt_duration_conflict")
    margin = o.fact(reason="not_a_defuse_attempt", status="not_applicable")
    if bomb is not None:
        margin = (
            o.fact(bomb["value"] - deadline["value"])
            if bomb["status"] == "observed"
            else bomb
        )
    return {
        "attempt_state": o.fact(state),
        "attempt_remaining_seconds": left,
        "attempt_progress": progress,
        "defuse_deadline_margin_seconds": margin,
    }


def initial_context(
    anchor,
    base,
    objective_row,
    public_state,
    ownership_state,
    row,
    prior=None,
    witness=None,
):
    if row is None:
        return {
            r: {k: o.fact(reason="missing_exact_clock_sample") for k in PLAYER}
            for r in ("self", "ally")
        }
    phase = o.initial_context(anchor, base, objective_row, public_state)["global"][
        "objective_phase"
    ]["value"]
    players = {p["actor_alias"]: p for p in row["players"]}
    objectives = {p["actor_alias"]: p for p in objective_row["players"]}
    owners = (
        {p["actor_alias"]: p for p in ownership_state["players"]}
        if ownership_state
        else {}
    )
    pause = pause_state(row)
    result = {}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        player = players.get(alias)
        now = aligned_now(row, player, witness)
        clock = (
            o.fact(
                "aligned_native_round_origin"
                if witness
                else "aligned_native_simulation"
            )
            if now["status"] == "observed"
            else now
        )
        guarded = (
            now if pause["status"] == "observed" else o.fact(reason=pause["reason"])
        )
        round_left = round_state = o.fact(reason="objective_phase_unavailable")
        bomb_left = bomb_state = o.fact(reason="objective_phase_unavailable")
        if phase == "preplant":
            bomb_left = bomb_state = o.fact(
                reason="no_active_planted_phase", status="not_applicable"
            )
            if row["rules"]:
                fields = row["rules"]["fields"]
                start = native(fields["m_fRoundStartTime"], game_time=True)
                duration = native(fields["m_iRoundTime"], "integer")
                deadline = (
                    o.fact(start["value"] + duration["value"])
                    if start["status"] == duration["status"] == "observed"
                    else start
                    if start["status"] != "observed"
                    else duration
                )
                round_left, round_state = remaining(deadline, guarded)
        elif phase == "planted":
            round_left = round_state = o.fact(
                reason="round_clock_superseded_by_bomb", status="not_applicable"
            )
            active = [
                b
                for b in row["bombs"]
                if native(b["fields"]["m_bBombTicking"], "bool")["value"] is True
                and native(b["fields"]["m_bBombDefused"], "bool")["value"] is False
            ]
            if len(active) == 1:
                bomb_left, bomb_state = remaining(
                    native(active[0]["fields"]["m_flC4Blow"], game_time=True), guarded
                )
            else:
                bomb_left = bomb_state = o.fact(
                    reason="native_planted_entity_uncorroborated"
                )
        if pause["value"] == "paused":
            if round_state["value"] == "active":
                round_state = o.fact("paused")
            if bomb_state["value"] == "active":
                bomb_state = o.fact("paused")
        owned = (
            (owners.get(alias), ownership_state["entities"])
            if ownership_state
            else None
        )
        result[role] = {
            "pause_state": pause,
            "clock_state": clock,
            "round_remaining_seconds": round_left,
            "round_clock_state": round_state,
            "bomb_remaining_seconds": bomb_left,
            "bomb_clock_state": bomb_state,
            **attempt(player, objectives.get(alias), owned, row, phase, now, pause),
        }
    if prior is not None:
        old_base, old_objective, old_public, _old_ownership, old_row = prior
        if old_row is not None and all(
            old_row[k] == row[k] for k in ("instance", "epoch")
        ):
            old_phase = o.initial_context(anchor, old_base, old_objective, old_public)[
                "global"
            ]["objective_phase"]["value"]
            if phase == old_phase and phase in ("preplant", "planted"):
                previous = initial_context(anchor, *prior)
                old_players = {x["actor_alias"]: x for x in old_row["players"]}
                for role, alias in (
                    ("self", anchor["actor_alias"]),
                    ("ally", anchor["ally_alias"]),
                ):
                    current_player, old_player = (
                        players.get(alias),
                        old_players.get(alias),
                    )
                    if (
                        current_player is None
                        or old_player is None
                        or current_player["pawn_life"] is None
                        or current_player["pawn_life"] != old_player["pawn_life"]
                    ):
                        continue
                    before = previous[role]["attempt_state"]["value"]
                    if result[role]["attempt_state"]["value"] == "none" and before in (
                        "planting",
                        "defusing",
                        "paused_planting",
                        "paused_defusing",
                        "expired_planting",
                        "expired_defusing",
                    ):
                        kind = "defusing" if before.endswith("defusing") else "planting"
                        result[role].update(
                            no_attempt(
                                "native_attempt_interrupted", "interrupted_" + kind
                            )
                        )
    return copy.deepcopy(result)


def prefix_inputs(support, objective, ownership, data, seq):
    support = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:seq]},
    }
    objective_frames = [
        r for r in objective["observations"]["frames"] if r["source_sequence"] <= seq
    ]
    objective = {
        **objective,
        "observations": {
            "frames": objective_frames,
            "events": objective["observations"]["events"][
                : objective_frames[-1]["event_sequence"] if objective_frames else 0
            ],
        },
    }
    cuts = [r for r in ownership["observations"]["cuts"] if r["source_sequence"] <= seq]
    ownership = {
        **ownership,
        "observations": {
            "cuts": cuts,
            "states": ownership["observations"]["states"][
                : cuts[-1]["state_sequence"] if cuts else 0
            ],
        },
    }
    data = {
        **data,
        "observations": {
            "frames": [
                r for r in data["observations"]["frames"] if r["source_sequence"] <= seq
            ]
        },
    }
    return support, objective, ownership, data


def initial_features(
    support, objective, ownership, data, anchor_id, actor, tick, frame
):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    support, objective, ownership, data = prefix_inputs(
        support, objective, ownership, data, seq
    )
    o.validate_source(objective, support)
    c4.validate_source(ownership, support, objective)
    validate_source(data, support, objective, ownership)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    anchor = next(x for x in selected if x["anchor_id"] == anchor_id)
    public = o.public_prefixes(objective)
    alignments, _ = alignment_prefixes(support, objective, data)
    prior = (
        context_inputs(support, objective, ownership, data, public, seq - 1)
        if seq > 1
        else None
    )
    return initial_context(
        anchor,
        *context_inputs(support, objective, ownership, data, public, seq),
        prior=prior,
        witness=alignments.get(seq),
    )


def context_inputs(support, objective, ownership, data, public, seq):
    obj = next(
        (r for r in objective["observations"]["frames"] if r["source_sequence"] == seq),
        None,
    )
    row = next(
        (r for r in data["observations"]["frames"] if r["source_sequence"] == seq), None
    )
    cut = next(
        (r for r in ownership["observations"]["cuts"] if r["source_sequence"] == seq),
        None,
    )
    state = (
        ownership["observations"]["states"][cut["state_sequence"] - 1] if cut else None
    )
    return (
        support["observations"]["frames"][seq - 1],
        obj,
        public.get(seq, {}),
        state,
        row,
    )


def build_report(
    support, objective, ownership, data, anchor_report, previous_report, references
):
    validate_source(data, support, objective, ownership)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        selected
        == [
            {k: v for k, v in x.items() if k != "later_observation"}
            for x in anchor_report["anchors"]
        ],
        "clock anchor mismatch",
    )
    a.require(
        len(previous_report["rows"]) == len(selected),
        "clock previous row count mismatch",
    )
    rows, matrix, counts = [], {}, {k: Counter() for k in FIELDS}
    index = {r["source_sequence"]: r for r in data["observations"]["frames"]}
    objectives = {r["source_sequence"]: r for r in objective["observations"]["frames"]}
    owners = {
        r["source_sequence"]: ownership["observations"]["states"][
            r["state_sequence"] - 1
        ]
        for r in ownership["observations"]["cuts"]
    }
    public = o.public_prefixes(objective)
    alignments, witnesses = alignment_prefixes(support, objective, data)
    for anchor, previous in zip(selected, previous_report["rows"], strict=True):
        a.require(
            all(anchor[k] == previous[k] for k in o.KEYS),
            "clock previous anchor mismatch",
        )
        seq = anchor["source_sequence"]
        prior = (
            (
                support["observations"]["frames"][seq - 2],
                objectives.get(seq - 1),
                public.get(seq - 1, {}),
                owners.get(seq - 1),
                index.get(seq - 1),
            )
            if seq > 1
            else None
        )
        initial = initial_context(
            anchor,
            support["observations"]["frames"][seq - 1],
            objectives.get(seq),
            public.get(seq, {}),
            owners.get(seq),
            index.get(seq),
            prior=prior,
            witness=alignments.get(seq),
        )
        key = anchor["instance"], anchor["actor_alias"], anchor["ally_alias"]
        entry = matrix.setdefault(
            key,
            {
                "anchors": 0,
                "canonical_interval_matched": 0,
                "fact_status_counts": {k: Counter() for k in FIELDS},
            },
        )
        entry["anchors"] += 1
        entry["canonical_interval_matched"] += (
            previous["competitive_round_status"] == "canonical_interval_matched"
        )
        for field in FIELDS:
            role, name = field.split(".")
            f = initial[role][name]
            status = f["status"] + (":" + f["reason"] if f["reason"] else "")
            counts[field][status] += 1
            entry["fact_status_counts"][field][status] += 1
        rows.append(
            {
                **{k: anchor[k] for k in o.KEYS},
                "competitive_round_status": previous["competitive_round_status"],
                "initial": initial,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    report = {
        "schema_id": "stratai.support_objective_clock_context@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": references,
        "clock_witnesses": witnesses,
        "rows": rows,
        "matrix": [
            {"instance": k[0], "actor_alias": k[1], "ally_alias": k[2], **v}
            for k, v in sorted(matrix.items())
        ],
        "summary": {
            "anchors": len(rows),
            "matrix_rows": len(matrix),
            "canonical_interval_matched": sum(
                v["canonical_interval_matched"] for v in matrix.values()
            ),
            "fact_status_counts": counts,
            "eligible_decisions": 0,
        },
    }
    a.schema_validator(REPORT).validate(report)
    return report


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 512 * 1024 * 1024,
        "invalid objective clock source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    ownership, objective, support, ref, base = c4.load_source(path.parent / c4.DATA)
    a.require(
        data["references"]
        == {
            "support_source": ref,
            "objective_source": p.file_reference(path.parent / o.DATA),
            "c4_source": p.file_reference(path.parent / c4.DATA),
        },
        "clock source hash mismatch",
    )
    validate_source(data, support, objective, ownership)
    a.require(path.read_bytes() == raw, "clock source changed")
    return data, ownership, objective, support, base


def report_inputs(source, output):
    data, ownership, objective, support, _ = load_source(source)
    refs = {
        **data["references"],
        "clock_source": p.file_reference(source),
        **{
            key: p.file_reference(output / name)
            for key, name in (
                ("anchors", a.REPORT),
                ("physical_context", p.REPORT),
                ("navigation_context", n.REPORT),
                ("objective_context", o.REPORT),
                ("c4_context", c4.REPORT),
            )
        },
    }
    return (
        support,
        objective,
        ownership,
        data,
        json.loads((output / a.REPORT).read_bytes()),
        json.loads((output / c4.REPORT).read_bytes()),
        refs,
    )


def materialize(source, output, evaluator, maps_root):
    validate_output_directory(output)
    load_source(source)
    previous = c4.materialize(source.parent / c4.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(report["summary"]["anchors"] > 0, "empty clock context")
    a.require(
        all(
            report["summary"]["fact_status_counts"]["self." + key].get("observed", 0)
            > 0
            for key in ("round_remaining_seconds", "bomb_remaining_seconds")
        ),
        "no accredited round or planted bomb countdowns",
    )
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous}


def validate_artifact(output, source, evaluator, maps_root):
    c4.validate_artifact(output, source.parent / c4.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    a.require(
        report == build_report(*report_inputs(source, output)),
        "clock report recomputation mismatch",
    )
    return report["summary"]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--evaluator", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    args = parser.parse_args()
    function = validate_artifact if args.validate else materialize
    result = (
        function(args.output, args.source, args.evaluator, args.maps_root)
        if args.validate
        else function(args.source, args.output, args.evaluator, args.maps_root)
    )
    print(json.dumps(result))


if __name__ == "__main__":
    main()
