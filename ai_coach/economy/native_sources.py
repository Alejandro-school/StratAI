"""Verify opt-in native purchase sources against the immutable v2 candidate."""

from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.contracts import validate_economy_candidate as base
from ai_coach.economy.candidate_io import CONTRACTS, load_source, sha

require = base._require
MANIFEST = "economy_native_sources_manifest.json"
DATA = "economy_native_sources.json"
SCHEMAS = {
    "economy_native_sources.schema.json",
    "economy_native_sources_manifest.schema.json",
    "economy_native_reconstruction.schema.json",
}
ARRAYS = {
    "sellback": (
        "m_pBuyServices.m_vecSellbackPurchaseEntries",
        ("m_unDefIdx", "m_nCost", "m_nPrevArmor", "m_bPrevHelmet", "m_hItem"),
    ),
    "purchases_round": (
        "m_pActionTrackingServices.m_weaponPurchasesThisRound.m_weaponPurchases",
        ("m_nItemDefIndex", "m_nCount"),
    ),
    "purchases_match": (
        "m_pActionTrackingServices.m_weaponPurchasesThisMatch.m_weaponPurchases",
        ("m_nItemDefIndex", "m_nCount"),
    ),
}


def schema_validator(filename: str) -> Draft202012Validator:
    require(filename in SCHEMAS, "unknown native economy schema")
    schema = json.loads((CONTRACTS / filename).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def check_schema(value: dict, filename: str) -> None:
    error = next(schema_validator(filename).iter_errors(value), None)
    require(
        error is None,
        f"invalid {filename}: {list(error.absolute_path) if error else ''}",
    )


def validate_fact(fact: dict, tick: int, source: str) -> None:
    require(fact["source"] == source, "native property provenance differs")
    if fact["status"] == "observed":
        require(
            fact["value"] is not None
            and fact["observation_tick"] == fact["availability_tick"] == tick,
            "native observation availability differs",
        )
    else:
        require(
            fact["value"]
            is fact["observation_tick"]
            is fact["availability_tick"]
            is None,
            "missing native fact became observed",
        )


def validate_array(array: dict, root: str, members: tuple, tick: int) -> None:
    if array["length"] is None:
        require(
            array["status"] == "unavailable"
            and array["entries"] == []
            and array["reason"]
            in {"native_array_unavailable", "native_array_shape_unsupported"},
            "missing array became empty",
        )
        return
    require(
        len(array["entries"]) == array["length"], "native array cardinality differs"
    )
    complete = True
    for index, entry in enumerate(array["entries"]):
        for member in members:
            validate_fact(entry[member], tick, f"{root}.{index:04d}.{member}")
            complete &= entry[member]["status"] == "observed"
    expected = (
        ("observed", "native_array_and_members_observed")
        if complete
        else ("partial", "native_member_unavailable")
    )
    require(
        (array["status"], array["reason"]) == expected,
        "native array availability differs",
    )


def summary(data: dict) -> dict:
    observations = data["observations"]
    rows = observations["snapshots"]
    return {
        "snapshots": len(rows),
        "snapshot_checks": observations["snapshot_checks"],
        "version_observations": len(observations["version_observations"]),
        "sellback_observed": sum(r["sellback"]["status"] == "observed" for r in rows),
        "round_purchases_observed": sum(
            r["purchases_round"]["status"] == "observed" for r in rows
        ),
        "match_purchases_observed": sum(
            r["purchases_match"]["status"] == "observed" for r in rows
        ),
        "native_purchase_events": observations["generic_event_counts"].get(
            "item_purchase", 0
        ),
    }


def validate_data(data: dict, source: dict) -> dict:
    check_schema(data, "economy_native_sources.schema.json")
    for key in ("source_alias", "source_sha256", "build_identifier"):
        require(data[key] == source[key], "native/v2 identity or build differs")
    rounds = {r["instance"]: r for r in source["rounds"]}
    actors = {p["participant_alias"] for p in source["participants"]}
    previous_tick = -1
    checked_boundaries = set()
    observations = data["observations"]
    for index, row in enumerate(observations["snapshots"], 1):
        require(
            row["observation_id"] == f"n{index:08d}",
            "native snapshot identity/order differs",
        )
        require(
            row["instance"] in rounds and row["actor_alias"] in actors,
            "native actor/round not reconciled",
        )
        round_ = rounds[row["instance"]]
        require(
            previous_tick <= row["tick"] and row["tick"] >= round_["start_tick"],
            "native tick order differs",
        )
        if round_["freeze_end_tick"] is not None:
            require(
                row["tick"] <= round_["freeze_end_tick"], "post-freeze native snapshot"
            )
        elif round_["end_tick"] is not None:
            require(
                row["tick"] <= round_["end_tick"], "native observation after round end"
            )
        previous_tick = row["tick"]
        boundary = (row["instance"], row["actor_alias"], row["tick"])
        reference = (
            round_["initial_state"]
            if row["tick"] == round_["start_tick"]
            else round_["execution"]["freeze_snapshot"]
            if row["tick"] == round_["freeze_end_tick"]
            else None
        )
        if (
            reference is not None
            and reference["tick"] == row["tick"]
            and boundary not in checked_boundaries
        ):
            matching = [
                a for a in reference["actors"] if a["actor_alias"] == row["actor_alias"]
            ]
            require(len(matching) == 1, "native boundary actor differs from v2")
            expected = matching[0]
            require(
                row["side"] == expected["side"]
                and row["connected"] == expected["connected"],
                "native boundary roster differs from v2",
            )
            for name in ("money", "spent_native"):
                require(
                    row[name]["status"] == expected["facts"][name]["status"]
                    and row[name]["value"] == expected["facts"][name]["value"],
                    "native boundary balance differs from v2",
                )
            checked_boundaries.add(boundary)
        for field, prop in (
            ("money", "m_iAccount"),
            ("spent_native", "m_iCashSpentThisRound"),
        ):
            validate_fact(row[field], row["tick"], "m_pInGameMoneyServices." + prop)
        for name, (root, members) in ARRAYS.items():
            validate_array(row[name], root, members, row["tick"])
    for row in observations["version_observations"]:
        for name, fact in row["fields"].items():
            native_name = "protocol" if name == "network_protocol" else name
            validate_fact(fact, row["tick"], row["source"] + "." + native_name)
    require(
        observations["snapshot_checks"] >= len(observations["snapshots"]),
        "native checks count differs",
    )
    return summary(data)


def load_candidate(
    path: Path, *, require_observed: bool = False
) -> tuple[dict, dict, dict]:
    path = base._safe_file(path.parent.absolute(), path.name)
    require(path.name == MANIFEST, "unexpected native source manifest")
    manifest, manifest_raw = base._read(path)
    check_schema(manifest, "economy_native_sources_manifest.schema.json")
    source_ref, artifact_ref = manifest["source_candidate"], manifest["artifact"]
    source_path = base._safe_file(path.parent, source_ref["path"])
    require(
        source_path.stat().st_size == source_ref["bytes"]
        and sha(source_path.read_bytes()) == source_ref["sha256"],
        "native/v2 source hash differs",
    )
    source, source_manifest_hash, _ = load_source(source_path)
    data_path = base._safe_file(path.parent, artifact_ref["path"])
    data, raw = base._read(data_path)
    require(
        len(raw) == artifact_ref["bytes"] and sha(raw) == artifact_ref["sha256"],
        "native artifact integrity differs",
    )
    require(
        data.get("source_candidate_manifest_sha256")
        == source_manifest_hash
        == source_ref["sha256"],
        "native source changed during load",
    )
    for key in (
        "contract_version",
        "source_alias",
        "source_sha256",
        "build_identifier",
        "comparison_allowed",
        "training_allowed",
        "product_supported",
    ):
        require(data.get(key) == manifest[key], "native manifest/data metadata differs")
    counts = validate_data(data, source)
    require(manifest["summary"] == counts, "native summary differs")
    require(path.read_bytes() == manifest_raw, "native manifest changed during load")
    if require_observed:
        require(
            counts["snapshots"] > 0
            and counts["version_observations"] > 0
            and counts["snapshot_checks"] > 0,
            "native source smoke is empty",
        )
    return (
        data,
        source,
        {
            "source_manifest_sha256": sha(manifest_raw),
            "source_data_sha256": sha(raw),
            "source_candidate_manifest_sha256": source_manifest_hash,
        },
    )
