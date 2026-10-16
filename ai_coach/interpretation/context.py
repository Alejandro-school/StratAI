"""Causal own/ally episodes. A requested interval is not an intention threshold."""

import json
import math
from copy import deepcopy
from functools import lru_cache
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.interpretation import temporal as t
from ai_coach.support.anchors import require

SCHEMA_ID = "stratai.contextual_input@1"
VERSION = "1.0.0-candidate.1"
SCHEMA_PATH = (
    Path(__file__).resolve().parents[1] / "contracts/contextual_input.schema.json"
)
FIELDS = ("record_id", "cut", "continuity", "actors", "missing_previous_actors")


def interval(records, *, start_sequence, end_sequence):
    """Keep every sample, including gaps, unavailable values and life boundaries.

    Input is a validated temporal stream, already projected by perspective.
    No interpolation, nearest sample joins, or case/answer/outcome fields.
    """
    require(
        type(start_sequence) is int and type(end_sequence) is int,
        "sequences must be integers",
    )
    require(0 < start_sequence <= end_sequence, "invalid context interval")
    selected = []
    head = None
    previous = None
    for row in records:
        seq = row["cut"]["sequence"]
        if seq > end_sequence:
            break
        if seq < start_sequence:
            continue
        t.validate_record(row)
        identity = t.pick(
            row, ("source_alias", "source_sha256", "build_identifier", "perspective")
        )
        require(head is None or identity == head, "mixed context source/perspective")
        if previous:
            require(seq == previous["cut"]["sequence"] + 1, "missing context record")
            require(
                row["previous_record_id"] == previous["record_id"],
                "broken context chain",
            )
        head = identity
        # Resource values retain provenance. Magazine/reserve are NOT readiness.
        selected.append(t.pick(row, FIELDS))
        previous = row
        if seq == end_sequence:
            break
    require(
        bool(selected)
        and selected[0]["cut"]["sequence"] == start_sequence
        and selected[-1]["cut"]["sequence"] == end_sequence,
        "missing exact interval",
    )
    result = {
        "schema_id": SCHEMA_ID,
        "contract_version": VERSION,
        **head,
        "requested_interval": {
            "start_sequence": start_sequence,
            "end_sequence": end_sequence,
        },
        "samples": selected,
        "resource_scope": "physical_active_weapon_only_ammunition_not_readiness",
        "objective_scope": "own_ally_c4_possession_not_world_bomb_location",
        "trajectory_kind": "sampled_world_positions_not_physical_route",
        "training_allowed": False,
        "product_supported": False,
    }
    result["context_id"] = t.digest(result)
    validate(result)
    return result


def from_sources(source, *, side, start_sequence, end_sequence, **sources):
    return interval(
        t.iter_records(source, side=side, through_sequence=end_sequence, **sources),
        start_sequence=start_sequence,
        end_sequence=end_sequence,
    )


@lru_cache(maxsize=1)
def validator():
    schema = json.loads(SCHEMA_PATH.read_bytes())
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def validate(value):
    validator().validate(value)
    require(
        value["context_id"]
        == t.digest({k: v for k, v in value.items() if k != "context_id"}),
        "context hash mismatch",
    )
    rows = value["samples"]
    bounds = value["requested_interval"]
    require(
        len(rows) == bounds["end_sequence"] - bounds["start_sequence"] + 1,
        "incomplete context",
    )
    for i, row in enumerate(rows):
        cut = row["cut"]
        require(
            cut["sequence"] == bounds["start_sequence"] + i, "context order mismatch"
        )
        require(
            len({a["actor_alias"] for a in row["actors"]}) == len(row["actors"]),
            "duplicate context actor",
        )
        for actor in row["actors"]:
            require(
                actor["identity"] is None
                or actor["identity"]["side"] == value["perspective"]["side"],
                "opponent in context",
            )
            for ref in actor["references"]:
                require(
                    ref["source_sequence"] == cut["sequence"]
                    and ref["availability_tick"] == cut["tick"]
                    and ref["availability_frame"] == cut["frame"]
                    and ref["epoch"] == cut["epoch"]
                    and ref["instance"] == cut["instance"]
                    and ref["actor_alias"] == actor["actor_alias"],
                    "context provenance mismatch",
                )
            if actor["position"]["status"] == "observed":
                require(
                    all(math.isfinite(x) for x in actor["position"]["value"].values()),
                    "nonfinite context position",
                )
    return value


def trajectories(value):
    """Continuous suffixes only: do not bridge an unavailable or changed identity."""
    validate(value)
    paths = {}
    previous_cut = None
    for row in value["samples"]:
        cut = row["cut"]
        continuous = (
            previous_cut is not None and row["continuity"] == "adjacent_samples"
        )
        if continuous:
            continuous = all(
                cut[k] == previous_cut[k]
                for k in (
                    "epoch",
                    "instance",
                    "round_number",
                    "tick_rate",
                    "sample_step",
                    "phase",
                )
            )
            continuous = (
                continuous
                and cut["phase"] == "playing"
                and cut["tick"] - previous_cut["tick"] == cut["sample_step"]
            )
        if not continuous:
            paths = {}
        current = set()
        for actor in row["actors"]:
            alias = actor["actor_alias"]
            current.add(alias)
            if (
                actor["exclusion"] is not None
                or actor["position"]["status"] != "observed"
                or actor["alive"]["value"] is not True
                or cut["phase"] != "playing"
            ):
                paths.pop(alias, None)
                continue
            old = paths.get(alias, [])
            if old and old[-1]["identity"] != actor["identity"]:
                old = []
            paths[alias] = [*old, {"cut": deepcopy(cut), **deepcopy(actor)}]
        paths = {alias: path for alias, path in paths.items() if alias in current}
        previous_cut = cut
    return paths
