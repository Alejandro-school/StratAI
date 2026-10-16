"""Factual candidate input with the same data boundary as review web @5/@9.

``build(packet, case_id=..., instrument=..., kills=..., extra=..., bomb=...)``
accepts the existing packet @3 and structured supplements, never rendered HTML,
human readings or predictions. It retains the full prefix and produces a final
factual snapshot and dated events. The function does not mutate its inputs.

``load(packet_path, packet_sha256=..., ...)`` verifies each file against an
explicit expected SHA-256 before decoding it and records byte-level receipts.
In-memory ``build`` records content hashes only, not verified-file claims.
``validate`` recomputes every derived field; file receipts are provenance, not
an independent authentication of an upstream producer or a human response.
"""

import hashlib
import json
import math
import re
from collections import Counter
from copy import deepcopy
from functools import cache
from itertools import pairwise
from pathlib import Path

from ai_coach.interpretation import context
from ai_coach.support.anchors import require, schema_validator

SCHEMA_ID = "stratai.candidate_context@1"
INSTRUMENTS = ("stratai.review_web@5", "stratai.review_web@9")
SUPPLEMENT_SCHEMAS = {
    "kills": "review_kill_context.json",
    "extra": "review_extra_context.json",
    "bomb": "review_bomb_context.json",
}
LIMITATIONS = [
    "Factual development input; no tactical label, recommendation or training.",
    "Own/ally sampled positions do not prove a traversable route or intent.",
    "Event positions and weapons describe their event, not current enemy state.",
    "No individual perception or communication is accredited.",
    "Unknown counts, objective state and round eligibility remain unknown.",
    "Byte hashes bind supplied artifacts; upstream source recomputation is separate.",
]


@cache
def _validator(name):
    return schema_validator(name)


def content_hash(value):
    """Canonical UTF-8 JSON hash; this is distinct from the file-byte hash."""
    raw = json.dumps(
        value,
        ensure_ascii=False,
        sort_keys=True,
        separators=(",", ":"),
        allow_nan=False,
    ).encode("utf-8")
    return hashlib.sha256(raw).hexdigest()


def _counts(row, side):
    require(row["side"] == side, "team count side differs")
    known = row["observed_alive"] + row["observed_dead"]
    require(known <= 5, "team count exceeds 5v5 scope")
    if row["unknown"] is not None:
        require(known + row["unknown"] == 5, "team count total differs")
    if row["status"] == "observed":
        require(
            row["alive"] == row["observed_alive"] and row["unknown"] == 0,
            "observed team count differs",
        )
    else:
        require(row["alive"] is None, "partial count asserted as exact")


def _packet(packet):
    _validator("contextual_review_packet_v3.json").validate(packet)
    value = context.validate(packet["context"])
    samples = value["samples"]
    cuts = [sample["cut"] for sample in samples]
    side = value["perspective"]["side"]
    require(
        len({(c["round_number"], c["instance"], c["epoch"]) for c in cuts}) == 1,
        "candidate crosses round, instance or epoch",
    )
    require(
        all(
            a["tick"] < b["tick"] and a["frame"] < b["frame"] for a, b in pairwise(cuts)
        ),
        "candidate cuts not increasing",
    )
    require(
        len({c["tick_rate"] for c in cuts}) == 1
        and type(cuts[0]["tick_rate"]) in (int, float)
        and math.isfinite(cuts[0]["tick_rate"])
        and cuts[0]["tick_rate"] > 0,
        "candidate tick rate unavailable or changing",
    )
    for name in ("inventory", "objective", "round_context", "team_status"):
        require(len(packet[name]) == len(samples), name + " sample count differs")
    continuity_start = cuts[0]["sequence"]
    for i, sample in enumerate(samples):
        seq = sample["cut"]["sequence"]
        if i and (
            sample["continuity"] != "adjacent_samples"
            or sample["cut"]["tick"] - cuts[i - 1]["tick"]
            != sample["cut"]["sample_step"]
            or sample["cut"]["sample_step"] != cuts[i - 1]["sample_step"]
        ):
            continuity_start = seq
        actors = {a["actor_alias"]: a for a in sample["actors"]}
        inventory = packet["inventory"][i]
        require(set(inventory) == set(actors), "inventory roster differs")
        for alias, item in inventory.items():
            require(item["source_sequence"] == seq, "inventory cut differs")
            old = item["last_known"]
            require(
                old is None
                or (
                    old["source_sequence"] <= seq
                    and old["tick"] <= sample["cut"]["tick"]
                ),
                "future last-known inventory",
            )
            if old is not None:
                require(
                    continuity_start <= old["source_sequence"] < seq
                    and old["identity"] == actors[alias]["identity"]
                    and actors[alias]["exclusion"] is None,
                    "last-known inventory identity or continuity differs",
                )
                previous_index = old["source_sequence"] - cuts[0]["sequence"]
                previous_inventory = packet["inventory"][previous_index].get(alias)
                require(
                    previous_inventory is not None
                    and previous_inventory["status"] == "current"
                    and previous_inventory["items"] == old["items"]
                    and cuts[previous_index]["tick"] == old["tick"],
                    "last-known inventory origin differs",
                )
            if item["status"] == "current":
                require(
                    actors[alias]["alive"]["value"] is True,
                    "current inventory belongs to unavailable actor",
                )
        for name in ("objective", "round_context", "team_status"):
            require(packet[name][i]["source_sequence"] == seq, name + " cut differs")
        objective = packet["objective"][i]
        carriers = [
            a["actor_alias"] for a in sample["actors"] if a["has_c4"]["value"] is True
        ]
        require(
            all(actors[alias]["alive"]["value"] is True for alias in carriers),
            "current carrier is not alive",
        )
        require(
            sorted(objective["carrier_aliases"]) == sorted(carriers),
            "objective carrier identity differs",
        )
        round_row = packet["round_context"][i]
        for origin in (
            round_row["eligibility"]["basis_sequence"],
            round_row["clock_origin_sequence"],
            packet["team_status"][i]["roster_start_sequence"],
        ):
            require(origin is None or origin <= seq, "future factual origin")
        eligibility = round_row["eligibility"]
        require(
            eligibility["analysis_allowed"]
            is (eligibility["kind"] == "bomb_objective"),
            "eligibility status differs",
        )
        for name, team in (
            ("allies", side),
            ("opponents", "CT" if side == "T" else "T"),
        ):
            _counts(packet["team_status"][i][name], team)
        own = packet["team_status"][i]["allies"]
        if own["status"] == "observed":
            require(
                len(actors) == 5
                and own["alive"]
                == sum(a["alive"]["value"] is True for a in actors.values()),
                "ally count and packet life differ",
            )
            require(
                own["observed_dead"]
                == sum(a["alive"]["value"] is False for a in actors.values()),
                "ally dead count and packet life differ",
            )
    kinds = {row["eligibility"]["kind"] for row in packet["round_context"]}
    scope = (
        "excluded"
        if kinds <= {"knife", "warmup"}
        else "eligible"
        if kinds == {"bomb_objective"}
        else "pending_context"
    )
    require(packet["analysis_scope"] == scope, "packet analysis scope differs")
    return samples, side


def _participant(player, sample, labels, side):
    if player is None:
        return
    require(player["label"].startswith(player["side"]), "event label side differs")
    if player["position"] is not None:
        require(
            all(math.isfinite(v) for v in player["position"].values()),
            "nonfinite event position",
        )
    if player["side"] == side:
        actors = {labels[a["actor_alias"]]: a for a in sample["actors"]}
        actor = actors.get(player["label"])
        require(
            actor is not None
            and actor["identity"] is not None
            and actor["identity"]["identity_status"] == "observed"
            and actor["identity"]["pawn_life"] == player["life"],
            "event own actor identity differs",
        )


def _events(packet, supplements, labels):
    samples = packet["context"]["samples"]
    cuts = [s["cut"] for s in samples]
    side = packet["context"]["perspective"]["side"]
    streams = [("kill", supplements["kills"]["events"])]
    if supplements["extra"] is not None:
        streams += [
            ("damage", supplements["extra"]["damage"]),
            ("c4", supplements["extra"]["c4"]),
        ]
    rows, seen = [], set()
    for kind, events in streams:
        for event in events:
            require(event["id"] not in seen, "duplicate candidate event")
            seen.add(event["id"])
            index = event["available_index"]
            require(0 <= index < len(samples), "event availability outside prefix")
            cut = cuts[index]
            require(event["available_sequence"] == cut["sequence"], "event cut differs")
            require(event["round"] == cut["round_number"], "event round differs")
            require(
                cuts[0]["tick"] <= event["tick"] <= cut["tick"]
                and (index == 0 or cuts[index - 1]["tick"] < event["tick"]),
                "event is future, stale or not first available cut",
            )
            require(
                index == 0 or samples[index]["continuity"] == "adjacent_samples",
                "event crosses sample gap",
            )
            clock = packet["round_context"][index]
            require(
                clock["eligibility"]["kind"] not in ("knife", "warmup"),
                "event in excluded round",
            )
            elapsed = clock["round_elapsed_seconds"]
            if event["elapsed_seconds"] is not None and elapsed["status"] == "observed":
                require(
                    event["elapsed_seconds"] <= elapsed["value"] + 1e-8,
                    "event clock after available cut",
                )
            players = (
                [event["actor"]]
                if kind == "c4"
                else [event["attacker"], event["victim"]]
            )
            for player in players:
                _participant(player, samples[index], labels, side)
                if index:
                    _participant(player, samples[index - 1], labels, side)
            if kind == "c4":
                require(event["actor"]["side"] == side, "opponent objective event")
            if kind == "kill" and event["victim"]["side"] == side:
                victim = next(
                    a
                    for a in samples[index]["actors"]
                    if labels[a["actor_alias"]] == event["victim"]["label"]
                )
                require(
                    victim["alive"]["value"] is False, "event victim death unconfirmed"
                )
            rows.append(
                {
                    "kind": kind,
                    "event": deepcopy(event),
                    "availability_cut": deepcopy(cut),
                    "demo_seconds_since_event": (cuts[-1]["tick"] - event["tick"])
                    / cut["tick_rate"],
                    "position_meaning": "event_position_not_current_or_confirmed_perception",
                }
            )
    return sorted(rows, key=lambda r: (r["event"]["tick"], r["kind"], r["event"]["id"]))


def _supplements(packet, supplements, case_id, instrument):
    value = packet["context"]
    samples = value["samples"]
    require(instrument in INSTRUMENTS, "unsupported review instrument")
    require(supplements["kills"] is not None, "kill supplement required")
    if instrument == "stratai.review_web@5":
        require(
            supplements["extra"] is None and supplements["bomb"] is None,
            "instrument @5 excludes extra and bomb supplements",
        )
    else:
        require(
            supplements["extra"] is not None and supplements["bomb"] is not None,
            "instrument @9 requires extra and bomb supplements",
        )
    refs = []
    for name, supplement in supplements.items():
        if supplement is None:
            continue
        _validator(SUPPLEMENT_SCHEMAS[name]).validate(supplement)
        require(supplement["case_id"] == case_id, "supplement case differs")
        require(
            all(supplement[k] == value[k] for k in ("source_alias", "source_sha256")),
            "supplement source differs",
        )
        require(
            supplement["side"] == value["perspective"]["side"],
            "supplement side differs",
        )
        require(
            supplement["frame_count"] == len(samples), "supplement frame count differs"
        )
        refs.append(supplement["source_refs"])
    for field in ("support_sha256", "canonical_manifest_sha256"):
        require(len({r[field] for r in refs}) == 1, "supplement provenance differs")
    extra, kills, bomb = (supplements[k] for k in ("extra", "kills", "bomb"))
    if extra is not None:
        require(
            extra["source_refs"]["combat_events_sha256"]
            == kills["source_refs"]["combat_sha256"]
            and extra["source_refs"]["combat_events_record_count"]
            == kills["source_refs"]["combat_record_count"],
            "combat supplement binding differs",
        )
    if bomb is not None:
        require(len(bomb["frames"]) == len(samples), "bomb frame count differs")
        for i, (row, sample) in enumerate(zip(bomb["frames"], samples, strict=True)):
            cut = sample["cut"]
            require(
                row["index"] == i
                and row["source_sequence"] == cut["sequence"]
                and row["tick"] == cut["tick"]
                and row["round"] == cut["round_number"],
                "bomb exact cut differs",
            )
            if row["position"] is not None:
                require(
                    all(math.isfinite(v) for v in row["position"].values()),
                    "nonfinite bomb position",
                )
            if row["status"] == "dropped_position":
                require(
                    bomb["side"] == "T"
                    and packet["round_context"][i]["eligibility"]["kind"]
                    == "bomb_objective"
                    and packet["objective"][i]["planted"] is False
                    and not packet["objective"][i]["carrier_aliases"],
                    "ground C4 conflicts with packet objective",
                )
        require(
            dict(Counter(r["status"] for r in bomb["frames"])) == bomb["counts"],
            "bomb counts differ",
        )


def _objective(packet, bomb, labels):
    row = packet["objective"][-1]
    ground = None if bomb is None else bomb["frames"][-1]
    carriers = [labels[a] for a in row["carrier_aliases"]]
    phase, state, reasons = "unknown", "unresolved", []
    if ground is not None and ground["status"] == "state_conflict":
        state, reasons = "conflicting", ["replay_ground_and_packet_objective_conflict"]
    elif len(carriers) > 1 or (row["planted"] is True and carriers):
        state, reasons = "conflicting", ["planted_or_carrier_facts_conflict"]
    elif row["planted"] is True:
        phase, state = "postplant", "planted"
    elif row["planted"] is False and len(carriers) == 1:
        phase, state = "preplant", "carried"
    elif (
        row["planted"] is False
        and ground is not None
        and ground["status"] == "dropped_position"
    ):
        phase, state = "preplant", "dropped"
    elif row["planted"] is False:
        state, reasons = (
            "not_planted_unresolved",
            ["no_current_carrier_or_ground_confirmation"],
        )
    else:
        reasons = ["public_planted_state_unavailable"]
    return {
        "phase": phase,
        "state": state,
        "planted": row["planted"],
        "carrier_labels": carriers,
        "ground": deepcopy(ground),
        "reasons": reasons,
    }


def _seal(result):
    result["candidate_context_id"] = content_hash(
        {k: v for k, v in result.items() if k != "candidate_context_id"}
    )
    _validator("candidate_context.json").validate(result)
    return result


def build(packet, *, case_id, instrument, kills, extra=None, bomb=None):
    """Join validated factual inputs; @5 keeps only the historical kill layer."""
    require(
        isinstance(case_id, str) and re.fullmatch(r"E[0-9]{2}", case_id),
        "invalid case ID",
    )
    samples, side = _packet(packet)
    supplements = {"kills": kills, "extra": extra, "bomb": bomb}
    _supplements(packet, supplements, case_id, instrument)
    aliases = sorted({a["actor_alias"] for row in samples for a in row["actors"]})
    labels = {alias: side + str(i + 1) for i, alias in enumerate(aliases)}
    events = _events(packet, supplements, labels)
    snapshot = {
        "cut": deepcopy(samples[-1]["cut"]),
        "actors": [
            dict(
                deepcopy(actor),
                label=labels[actor["actor_alias"]],
                inventory=deepcopy(packet["inventory"][-1][actor["actor_alias"]]),
            )
            for actor in samples[-1]["actors"]
        ],
        "objective": _objective(packet, bomb, labels),
        "round_context": deepcopy(packet["round_context"][-1]),
        "team_status": deepcopy(packet["team_status"][-1]),
    }
    inputs = {
        "packet": packet,
        **{k: v for k, v in supplements.items() if v is not None},
    }
    return _seal(
        {
            "schema_id": SCHEMA_ID,
            "case_id": case_id,
            "instrument": instrument,
            "packet": deepcopy(packet),
            "supplements": deepcopy(supplements),
            "snapshot": snapshot,
            "events": events,
            "input_content_hashes": {k: content_hash(v) for k, v in inputs.items()},
            "input_files": {},
            "perception_proven": False,
            "training_allowed": False,
            "predictions_included": False,
            "limitations": list(LIMITATIONS),
        }
    )


def load(
    packet_path,
    *,
    packet_sha256,
    case_id,
    instrument,
    kills_path,
    kills_sha256,
    extra_path=None,
    extra_sha256=None,
    bomb_path=None,
    bomb_sha256=None,
):
    """Read hash-bound JSON inputs only; never rewrite source or response files."""
    values, receipts = {}, {}
    for name, path, expected in (
        ("packet", packet_path, packet_sha256),
        ("kills", kills_path, kills_sha256),
        ("extra", extra_path, extra_sha256),
        ("bomb", bomb_path, bomb_sha256),
    ):
        require((path is None) == (expected is None), "path/hash pair incomplete")
        if path is None:
            values[name] = None
            continue
        require(
            isinstance(expected, str) and re.fullmatch(r"[0-9a-f]{64}", expected),
            "invalid expected SHA-256",
        )
        path = Path(path).resolve()
        raw = path.read_bytes()
        require(
            hashlib.sha256(raw).hexdigest() == expected, name + " file hash differs"
        )
        values[name] = json.loads(raw)
        receipts[name] = {"path": str(path), "sha256": expected, "bytes": len(raw)}
    result = build(
        values.pop("packet"), case_id=case_id, instrument=instrument, **values
    )
    result["input_files"] = receipts
    return _seal(result)


def validate(value):
    """Check content ID and recompute derived facts without trusting saved output."""
    _validator("candidate_context.json").validate(value)
    expected = build(
        value["packet"],
        case_id=value["case_id"],
        instrument=value["instrument"],
        **value["supplements"],
    )
    expected["input_files"] = deepcopy(value["input_files"])
    _seal(expected)
    require(expected == value, "candidate content or derived facts differ")
    if value["input_files"]:
        require(
            set(value["input_files"]) == set(value["input_content_hashes"]),
            "file receipts incomplete",
        )
    return value
