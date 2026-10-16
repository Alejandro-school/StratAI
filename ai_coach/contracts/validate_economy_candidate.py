"""Independent validation of the isolated economic observation candidate.

No demo parsing, publication, mutation or tactical judgement occurs here. The
schemas are local and self-contained; validation never resolves network refs.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import stat
from collections import Counter
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

from ai_coach.contracts.runtime_contracts import RuntimeContractError

VERSION = "1.0.0-candidate.1"
DATA_FILE = "observed_economy_candidate.json"
MANIFEST_FILE = "economy_candidate_manifest.json"
SCHEMAS = Path(__file__).resolve().parent
ACTOR_SOURCES = {
    "money": "m_pInGameMoneyServices.m_iAccount",
    "spent_native": "m_pInGameMoneyServices.m_iCashSpentThisRound",
    "armor": "m_ArmorValue",
    "equipment_value_native": "m_unCurrentEquipmentValue",
    "helmet": "m_pItemServices.m_bHasHelmet",
    "kit": "m_pItemServices.m_bHasDefuser",
    "buy_zone": "m_bInBuyZone",
    "inventory": "m_pWeaponServices.m_hMyWeapons",
}
SUMMARY_KEYS = (
    "round_instances",
    "participants",
    "initial_snapshots",
    "freeze_snapshots",
    "end_snapshots",
    "initial_actors",
    "prepurchase_available",
    "money_observed",
    "money_zero",
    "inventory_observed",
    "execution_events",
    "other_events",
    "prior_allied_drop_links",
    "unresolved_pickups",
)


def _require(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeContractError(message)


def schema_validator(filename: str) -> Draft202012Validator:
    _require(
        filename
        in {
            "observed_economy_candidate.schema.json",
            "economy_candidate_manifest.schema.json",
            "observed_economy_candidate_v2.schema.json",
            "economy_candidate_manifest_v2.schema.json",
        },
        "unknown economy schema",
    )
    schema = json.loads((SCHEMAS / filename).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def _schema(value: Any, filename: str) -> None:
    error = next(schema_validator(filename).iter_errors(value), None)
    if error:
        raise RuntimeContractError(
            f"{filename}: invalid value at {list(error.absolute_path)}"
        )


def _pairs(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result = {}
    for key, value in pairs:
        _require(key not in result, "duplicate JSON key")
        result[key] = value
    return result


def _read(path: Path) -> tuple[dict[str, Any], bytes]:
    _require(
        path.stat().st_size <= 64 * 1024 * 1024, "economic JSON exceeds size limit"
    )
    payload = path.read_bytes()
    try:
        value = json.loads(
            payload,
            object_pairs_hook=_pairs,
            parse_constant=lambda _: (_ for _ in ()).throw(ValueError("nonfinite")),
        )
    except (ValueError, UnicodeError) as exc:
        raise RuntimeContractError("invalid economic JSON") from exc
    _require(isinstance(value, dict), "economic JSON must be an object")
    return value, payload


def _safe_file(root: Path, relative: str) -> Path:
    part = Path(relative)
    _require(
        not part.is_absolute()
        and not part.drive
        and "\\" not in relative
        and ":" not in relative
        and ".." not in part.parts,
        "unsafe artifact path",
    )
    path = root / part
    for candidate in [
        *reversed(root.parents),
        root,
        *[root.joinpath(*part.parts[:i]) for i in range(1, len(part.parts) + 1)],
    ]:
        info = candidate.lstat()
        _require(
            not stat.S_ISLNK(info.st_mode)
            and not (
                getattr(info, "st_file_attributes", 0)
                & stat.FILE_ATTRIBUTE_REPARSE_POINT
            ),
            "artifact uses link/reparse point",
        )
    _require(
        path.is_file() and path.resolve().is_relative_to(root.resolve()),
        "artifact escapes candidate root",
    )
    return path


def _fact(fact: dict, tick: int, source: str) -> None:
    _require(fact["source"] == source, "economic fact provenance mismatch")
    if fact["status"] == "observed":
        _require(
            fact["observation_tick"] == tick == fact["availability_tick"],
            "economic fact availability differs from observation frontier",
        )


def _expected_prebuy(
    actor: dict, snapshot: dict, round_: dict, events: list[dict]
) -> tuple[str, str]:
    if snapshot["phase"] != "initial_available":
        return "unavailable", "not_initial_snapshot" if actor[
            "connected"
        ] else "participant_unavailable"
    if (
        round_["freeze_end_tick"] is not None
        and round_["freeze_end_tick"] <= snapshot["tick"]
        and actor["connected"]
    ):
        return "unavailable", "freeze_closed_before_initial"
    if not actor["connected"]:
        return "unavailable", "participant_unavailable"
    if snapshot["tick"] != round_["start_tick"]:
        return "unavailable", "initial_sample_after_round_start"
    if any(
        e["actor_alias"] == actor["actor_alias"]
        and e["callback_sequence"] < snapshot["callback_sequence"]
        for e in events
    ):
        return "unavailable", "activity_at_or_before_initial"
    if any(
        actor["facts"][k]["status"] != "observed"
        for k in ("money", "inventory", "spent_native")
    ):
        return "unavailable", "required_initial_fact_unavailable"
    if actor["facts"]["spent_native"]["value"] != 0:
        return "unavailable", "spend_at_initial_observation"
    return "available", "observed_at_round_start_without_prior_activity"


def validate_data(data: dict) -> dict[str, int]:
    _schema(data, "observed_economy_candidate.schema.json")
    _require(
        data["source_alias"] == "demo-" + data["source_sha256"][:20],
        "economic source alias/checksum mismatch",
    )
    _require(math.isfinite(data["tick_rate"]), "invalid tick rate")
    aliases = {p["participant_alias"] for p in data["participants"]}
    keys = {p["actor_key"] for p in data["participants"]}
    _require(
        len(aliases) == len(keys) == len(data["participants"]),
        "duplicate economic participant",
    )
    counts = Counter({k: 0 for k in SUMMARY_KEYS})
    counts.update(round_instances=len(data["rounds"]), participants=len(aliases))
    all_sequences: set[int] = set()
    for index, round_ in enumerate(data["rounds"], 1):
        _require(
            round_["instance"] == index,
            "economic round instances must preserve encounter order",
        )
        start, freeze, end = (
            round_[k] for k in ("start_tick", "freeze_end_tick", "end_tick")
        )
        _require(freeze is None or freeze >= start, "freeze precedes round")
        _require(
            end is None or end >= (freeze if freeze is not None else start),
            "end precedes economic boundaries",
        )
        events = sorted(
            round_["execution"]["events"] + round_["reconciliation"]["other_events"],
            key=lambda e: e["callback_sequence"],
        )
        by_id = {}
        drops = {}
        previous_tick = start
        for group, execution in (
            (round_["execution"]["events"], True),
            (round_["reconciliation"]["other_events"], False),
        ):
            _require(
                group == sorted(group, key=lambda e: e["callback_sequence"]),
                "events are out of order",
            )
            for event in group:
                in_window = freeze is not None and start < event["tick"] < freeze
                _require(
                    in_window == execution, "event crossed freeze partition/boundary"
                )
        for event in events:
            seq = event["callback_sequence"]
            _require(
                seq not in all_sequences and seq > 0,
                "duplicate economic callback sequence",
            )
            all_sequences.add(seq)
            _require(
                event["tick"] >= previous_tick,
                "economic callbacks move backwards in time",
            )
            previous_tick = event["tick"]
            _require(
                event["event_id"] == f"e{seq:08d}" and event["event_id"] not in by_id,
                "invalid economic event identity",
            )
            _require(
                event["actor_alias"] in aliases, "economic event participant is unknown"
            )
            if event["kind"] != "pickup":
                _require(
                    event["attribution"] == "callback_only"
                    and event["prior_drop_event_id"] is None,
                    "invented non-pickup attribution",
                )
            elif event["prior_drop_event_id"] is None:
                _require(
                    event["attribution"] == "purchase_or_pickup_unresolved",
                    "pickup became purchase without evidence",
                )
                counts["unresolved_pickups"] += 1
            else:
                prior = by_id.get(event["prior_drop_event_id"])
                _require(
                    prior is not None
                    and prior["kind"] == "drop"
                    and prior["actor_alias"] != event["actor_alias"]
                    and prior["side"] == event["side"] != "unknown"
                    and prior["entity_alias"] == event["entity_alias"] is not None
                    and prior["tick"] <= event["tick"],
                    "invalid allied drop reference",
                )
                _require(
                    drops.get(event["entity_alias"]) == prior,
                    "drop reference reused or superseded",
                )
                _require(
                    event["attribution"] == "prior_allied_drop_same_entity",
                    "drop attribution mismatch",
                )
                counts["prior_allied_drop_links"] += 1
            by_id[event["event_id"]] = event
            if event["kind"] == "drop" and event["entity_alias"] is not None:
                drops[event["entity_alias"]] = event
            if event["kind"] == "pickup":
                drops.pop(event["entity_alias"], None)
        counts["execution_events"] += len(round_["execution"]["events"])
        counts["other_events"] += len(round_["reconciliation"]["other_events"])
        snapshots = [
            ("initial_snapshots", "initial_available", round_["initial_state"], start),
            (
                "freeze_snapshots",
                "freeze_end",
                round_["execution"]["freeze_snapshot"],
                freeze,
            ),
            (
                "end_snapshots",
                "round_end",
                round_["reconciliation"]["round_end_snapshot"],
                end,
            ),
        ]
        for count, phase, snapshot, boundary in snapshots:
            if snapshot is None:
                continue
            _require(
                boundary is not None
                and snapshot["phase"] == phase
                and snapshot["tick"] >= boundary,
                "snapshot phase/frontier mismatch",
            )
            seq = snapshot["callback_sequence"]
            _require(
                seq > 0 and seq not in all_sequences,
                "snapshot callback sequence reused",
            )
            all_sequences.add(seq)
            counts[count] += 1
            seen = set()
            for key, fact in snapshot["context"].items():
                _fact(
                    fact,
                    snapshot["tick"],
                    "m_iScore" if key.startswith("score_") else "convar:" + key,
                )
            for actor in snapshot["actors"]:
                who = actor["actor_alias"]
                _require(
                    who in aliases and who not in seen,
                    "duplicate/unknown economic snapshot actor",
                )
                seen.add(who)
                for key, fact in actor["facts"].items():
                    _fact(fact, snapshot["tick"], ACTOR_SOURCES[key])
                    if not actor["connected"]:
                        _require(
                            fact["status"] == "unavailable",
                            "disconnected player has observed facts",
                        )
                inventory = actor["facts"]["inventory"]
                if inventory["status"] == "observed":
                    entities = [item["entity_alias"] for item in inventory["value"]]
                    _require(
                        entities == sorted(set(entities)),
                        "duplicate/unsorted inventory entity",
                    )
                    counts["inventory_observed"] += 1
                money = actor["facts"]["money"]
                if money["status"] == "observed":
                    counts["money_observed"] += 1
                    counts["money_zero"] += int(money["value"] == 0)
                expected = _expected_prebuy(actor, snapshot, round_, events)
                _require(
                    (actor["prepurchase_status"], actor["prepurchase_reason"])
                    == expected,
                    "prepurchase availability is not backed by the observed frontier",
                )
                counts["prepurchase_available"] += int(
                    actor["prepurchase_status"] == "available"
                )
                counts["initial_actors"] += int(phase == "initial_available")
        for item in round_["reconciliation"]["comparisons"]:
            _require(
                item["actor_alias"] in aliases, "comparison references unknown actor"
            )
            _require(
                (item["delta"] is None) == (item["status"] == "unavailable"),
                "comparison has inconsistent availability",
            )
        timeline = events + [s for _, _, s, _ in snapshots if s is not None]
        ordered = sorted(timeline, key=lambda item: item["callback_sequence"])
        _require(
            [item["tick"] for item in ordered]
            == sorted(item["tick"] for item in ordered),
            "snapshot/event chronology mismatch",
        )
    _require(counts["initial_actors"] > 0, "candidate has no initial actor records")
    return dict(counts)


def initial_team_state(
    data: dict, instance: int, actor_alias: str, t0_tick: int
) -> dict:
    """Read only initial same-side facts at the exact anchor; never a finding.

    This narrow accessor excludes all execution/reconciliation compartments,
    opponent economy and missing/late initial states. It grants no eligibility.
    """
    if data.get("contract_version") == "1.0.0-candidate.2":
        from ai_coach.contracts.economy_initial_frontiers import (
            initial_team_state as read_v2,
        )

        return read_v2(data, instance, actor_alias, t0_tick)
    _require(data.get("contract_version") == VERSION, "unknown initial economy version")
    round_ = next((r for r in data["rounds"] if r["instance"] == instance), None)
    _require(
        round_ is not None and t0_tick == round_["start_tick"],
        "unknown/changed economic T0",
    )
    snapshot = round_["initial_state"]
    _require(
        snapshot is not None and snapshot["tick"] == t0_tick,
        "initial state unavailable at T0",
    )
    actor = next(
        (p for p in snapshot["actors"] if p["actor_alias"] == actor_alias), None
    )
    _require(actor is not None, "actor absent from initial state")
    _require(actor["side"] in {"CT", "T"}, "unknown initial team side")
    team = [p for p in snapshot["actors"] if p["side"] == actor["side"]]
    _require(
        len(team) == 5 and all(p["prepurchase_status"] == "available" for p in team),
        "complete prebuy team not observed",
    )
    for item in [
        *snapshot["context"].values(),
        *[f for p in team for f in p["facts"].values()],
    ]:
        _require(
            item["status"] != "observed" or item["availability_tick"] <= t0_tick,
            "future initial fact",
        )
    return copy.deepcopy(
        {
            "instance": instance,
            "t0_tick": t0_tick,
            "side": actor["side"],
            "context": snapshot["context"],
            "actors": team,
        }
    )


def _validate_reconciliation(data: dict, base: dict, root: Path) -> int:
    """Recompute diagnostics using the hashed canonical aggregate, never features."""
    _require(
        base.get("schema_id") == "stratai.canonical_manifest@3"
        and base.get("export_format_version") == "3.8.0",
        "unsupported canonical economy base",
    )
    _require(
        base.get("match_id") == data["source_alias"]
        and base.get("demo_checksum_sha256") == data["source_sha256"],
        "canonical economy base identity mismatch",
    )
    references = [
        a
        for a in base.get("artifacts", [])
        if a.get("path") == "derived/economy_players.json"
    ]
    _require(len(references) == 1, "canonical economic aggregate missing/duplicated")
    reference = references[0]
    aggregate, payload = _read(
        _safe_file(root, f"match_{data['source_alias']}/canonical/{reference['path']}")
    )
    _require(
        len(payload) == reference["bytes"]
        and hashlib.sha256(payload).hexdigest() == reference["sha256"],
        "canonical economy integrity mismatch",
    )
    _require(
        aggregate.get("match_id") == data["source_alias"]
        and len(aggregate.get("players", [])) == reference["record_count"],
        "canonical economy rows/identity mismatch",
    )
    bridge = {p["actor_key"]: p["participant_alias"] for p in data["participants"]}
    legacy = {}
    for player in aggregate["players"]:
        actor_key = hashlib.sha256(
            (data["source_sha256"] + "\n" + player["player_id"]).encode()
        ).hexdigest()
        alias = bridge.get(actor_key)
        _require(alias is not None, "canonical economic participant has no bridge")
        key = (player["round_number"], alias)
        _require(key not in legacy, "ambiguous canonical economic player/round")
        legacy[key] = player
    occurrences = Counter(r["round_number"] for r in data["rounds"])
    verified = 0
    for round_ in data["rounds"]:
        expected = []
        if occurrences[round_["round_number"]] == 1:
            for (number, alias), player in legacy.items():
                if number != round_["round_number"]:
                    continue
                for metric, snapshot, field in (
                    (
                        "initial_minus_legacy_initial_money",
                        round_["initial_state"],
                        "round_start_observed",
                    ),
                    (
                        "end_frame_minus_legacy_round_end_money",
                        round_["reconciliation"]["round_end_snapshot"],
                        "round_end_observed",
                    ),
                ):
                    actors = snapshot["actors"] if snapshot else []
                    actor = next((a for a in actors if a["actor_alias"] == alias), None)
                    delta = None
                    if actor and actor["facts"]["money"]["status"] == "observed":
                        legacy_fact = player["money"][field]
                        _require(
                            isinstance(legacy_fact, dict)
                            and legacy_fact.get("status") == "observed",
                            "canonical economic money unavailable",
                        )
                        native = legacy_fact.get("amount")
                        _require(
                            type(native) is int, "canonical economic money unavailable"
                        )
                        delta = actor["facts"]["money"]["value"] - native
                    expected.append(
                        {
                            "actor_alias": alias,
                            "metric": metric,
                            "delta": delta,
                            "status": "unavailable"
                            if delta is None
                            else "diagnostic_different_observation_frontiers",
                        }
                    )
        expected.sort(key=lambda item: (item["actor_alias"], item["metric"]))
        _require(
            expected == round_["reconciliation"]["comparisons"],
            "economic reconciliation differs from independent computation",
        )
        verified += len(expected)
    return verified


def validate_candidate(
    path: Path, *, expected_sha256: str | None = None, require_observed: bool = False
) -> dict:
    _require(path.name == MANIFEST_FILE, "unexpected economic manifest filename")
    path = _safe_file(path.parent.absolute(), path.name)
    manifest, _ = _read(path)
    is_v2 = manifest.get("contract_version") == "1.0.0-candidate.2"
    _schema(
        manifest,
        "economy_candidate_manifest_v2.schema.json"
        if is_v2
        else "economy_candidate_manifest.schema.json",
    )
    if expected_sha256 is not None:
        _require(
            manifest["source_sha256"] == expected_sha256, "wrong economic source demo"
        )
    _require(
        manifest["artifact"]["path"] == DATA_FILE,
        "unexpected economic artifact filename",
    )
    expected_base = f"match_{manifest['source_alias']}/canonical/manifest.json"
    _require(
        manifest["base_manifest"]["path"] == expected_base,
        "unexpected canonical base path",
    )
    loaded = {}
    for name in ("artifact", "base_manifest"):
        reference = manifest[name]
        target = _safe_file(path.parent, reference["path"])
        value, payload = _read(target)
        _require(
            len(payload) == reference["bytes"]
            and hashlib.sha256(payload).hexdigest() == reference["sha256"],
            "economic artifact integrity mismatch",
        )
        loaded[name] = value
    data = loaded["artifact"]
    for key in (
        "source_alias",
        "source_sha256",
        "build_identifier",
        "contract_version",
    ):
        _require(
            data.get(key) == manifest[key], "economic manifest/data provenance differs"
        )
    from ai_coach.contracts import economy_initial_frontiers as v2

    summary = v2.validate_data(data) if is_v2 else validate_data(data)
    _require(summary == manifest["summary"], "economic coverage counts differ")
    recomputed = _validate_reconciliation(data, loaded["base_manifest"], path.parent)
    extra = (
        v2.validate_round_links(data, loaded["base_manifest"], path.parent)
        if is_v2
        else {}
    )
    if require_observed:
        if is_v2:
            _require(
                summary["round_start_probes"] > 0
                and summary["competitive_round_links"] > 0
                and extra["initial_coverage"]["competitive_complete_teams"] > 0,
                "economic v2 smoke has empty competitive initial records",
            )
        _require(
            all(
                summary[k] > 0
                for k in (
                    "initial_actors",
                    "money_observed",
                    "inventory_observed",
                    "execution_events",
                    "prepurchase_available",
                )
            ),
            "economic smoke has empty affected records",
        )
    return {
        "ok": True,
        "schema_id": "stratai.economy_candidate_validation@2"
        if is_v2
        else "stratai.economy_candidate_validation@1",
        "source_alias": manifest["source_alias"],
        "source_sha256": manifest["source_sha256"],
        "summary": summary,
        "independently_recomputed_comparisons": recomputed,
        "training_allowed": False,
        **extra,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--expected-source-sha256")
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    try:
        report = validate_candidate(
            args.manifest,
            expected_sha256=args.expected_source_sha256,
            require_observed=args.require_observed,
        )
    except (OSError, RuntimeContractError) as exc:
        parser.exit(1, f"economic candidate rejected: {exc}\n")
    print(json.dumps(report, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
