"""CS2 factual interface. No tactical hypotheses, human text or recommendations.

The retained candidate is evidence, not a model prompt. ``build`` exposes a
bounded situation, referenced changes and dated events. ``sample`` and
``evidence`` only return details within that same causal frontier. Native place
enrichment, when supplied, remains explicitly outside the original instrument.
"""

from copy import deepcopy
from functools import cache

from ai_coach.interpretation import candidate_context as candidate
from ai_coach.interpretation import native_place_context as places
from ai_coach.support.anchors import require, schema_validator

SCHEMA_ID = "stratai.common_factual_context@1"
GROUPS = ("frame", "objective", "players", "resources", "information")
UNITS = {
    "position": "world_units_xyz",
    "view_angles": "degrees_pitch_yaw",
    "sampled_displacement": "source_declared_sampled_displacement",
    "ammo_in_magazine": "count",
    "ammo_reserve": "count",
    "round_elapsed_seconds": "seconds",
    "configured_duration_seconds": "seconds",
    "round_remaining_seconds": "seconds",
    "bomb_remaining_seconds": "seconds",
}
LIMITATIONS = [
    "Sampled distribution is not a physical route, role or effective support.",
    "Event positions are historical; perception and communication are not proven.",
    "Native place supplements were not part of the original review instrument.",
    "Prior-round patterns are not supplied by this single-round evidence adapter.",
    "Factual completeness does not prove tactical interpretation or coaching.",
]


@cache
def _validator():
    return schema_validator("common_factual_context.json")


def _pointer(document, path):
    value = document
    for part in path.lstrip("/").split("/"):
        part = part.replace("~1", "/").replace("~0", "~")
        value = value[int(part)] if isinstance(value, list) else value[part]
    return value


def _fact(
    group,
    key,
    raw,
    paths,
    cut,
    index,
    *,
    subject=None,
    status=None,
    scope="own_ally",
    document="candidate",
    original=True,
    unit=None,
    derivation=None,
):
    wrapped = isinstance(raw, dict) and {"status", "value", "reason"} <= raw.keys()
    state = status or (
        raw.get("status", "observed") if isinstance(raw, dict) else "observed"
    )
    reason = raw.get("reason") if isinstance(raw, dict) else None
    return {
        "key": key,
        "group": group,
        "subject": subject,
        "status": state,
        "value": deepcopy(raw["value"] if wrapped else raw),
        "reason": reason,
        "unit": unit,
        "perspective": scope,
        "original_instrument": original,
        "derivation": derivation,
        "availability_cut": deepcopy(cut),
        "available_index": index,
        "evidence": [{"document": document, "pointer": path} for path in paths],
    }


def _at(value, index):
    packet = value["packet"]
    source = packet["context"]
    row = source["samples"][index]
    cut = row["cut"]
    result = {g: [] for g in GROUPS}

    def add(group, key, raw, path, **kw):
        paths = [path] if isinstance(path, str) else path
        result[group].append(_fact(group, key, raw, paths, cut, index, **kw))

    for key, raw, path in (
        ("map_name", packet["map_name"], "/packet/map_name"),
        (
            "build_identifier",
            source["build_identifier"],
            "/packet/context/build_identifier",
        ),
        ("perspective", source["perspective"], "/packet/context/perspective"),
        ("resource_scope", source["resource_scope"], "/packet/context/resource_scope"),
        ("cut", cut, f"/packet/context/samples/{index}/cut"),
        (
            "continuity",
            row["continuity"],
            f"/packet/context/samples/{index}/continuity",
        ),
        (
            "missing_previous_actors",
            row["missing_previous_actors"],
            f"/packet/context/samples/{index}/missing_previous_actors",
        ),
    ):
        add(
            "frame",
            key,
            raw,
            path,
            scope="analysis_frame",
            status="unavailable" if raw is None else "observed",
        )
    base = f"/packet/context/samples/{index}"
    for j, actor in enumerate(row["actors"]):
        alias = actor["actor_alias"]
        start = f"{base}/actors/{j}"
        for key in (
            "identity",
            "connected",
            "alive",
            "position",
            "exclusion",
            "sampled_displacement",
            "has_c4",
        ):
            group = "objective" if key == "has_c4" else "players"
            add(
                group,
                key,
                actor[key],
                [f"{start}/{key}", f"{start}/references"],
                subject=alias,
                unit=UNITS.get(key),
                status="unavailable"
                if key == "identity" and actor[key] is None
                else None,
            )
        for key, raw in actor["physical"].items():
            add(
                "resources",
                key,
                raw,
                [f"{start}/physical/{key}", f"{start}/references"],
                subject=alias,
                unit=UNITS.get(key),
            )
        inv = packet["inventory"][index][alias]
        state = inv["status"]
        if state == "current" and not (inv["complete"] and inv["quantity_complete"]):
            state = "partial"
        add(
            "resources",
            "inventory",
            inv,
            f"/packet/inventory/{index}/{alias}",
            subject=alias,
            status=state,
        )
        for key in ("health", "armor"):
            add(
                "resources",
                key,
                {
                    "status": "not_supplied",
                    "value": None,
                    "reason": "current_value_not_in_original_instrument",
                },
                ["/packet/context/resource_scope", f"{start}/identity"],
                subject=alias,
                unit="points",
                derivation="original_instrument_resource_coverage@1",
            )
    # Reuse the established objective semantics, selecting only this exact cut.
    aliases = sorted(
        {a["actor_alias"] for r in source["samples"][: index + 1] for a in r["actors"]}
    )
    labels = {
        a: source["perspective"]["side"] + str(i + 1) for i, a in enumerate(aliases)
    }
    bomb = value["supplements"]["bomb"]
    local_bomb = None if bomb is None else {"frames": [bomb["frames"][index]]}
    objective = candidate._objective(
        {"objective": [packet["objective"][index]]}, local_bomb, labels
    )
    paths = [f"/packet/objective/{index}"]
    paths.extend(f"{base}/actors/{i}/has_c4" for i in range(len(row["actors"])))
    if bomb is not None:
        paths.append(f"/supplements/bomb/frames/{index}")
    state = {
        "conflicting": "conflicting",
        "unresolved": "unavailable",
        "not_planted_unresolved": "partial",
    }.get(objective["state"], "derived")
    add(
        "objective",
        "objective_state",
        objective,
        paths,
        status=state,
        scope="objective_context",
        derivation="candidate_context._objective@1",
    )
    clock = packet["round_context"][index]
    for key in ("eligibility", "round_elapsed_seconds", "configured_duration_seconds"):
        add(
            "frame",
            key,
            clock[key],
            f"/packet/round_context/{index}/{key}",
            status="unknown"
            if key == "eligibility" and clock[key]["kind"] == "unknown"
            else None,
            scope="public_context",
            unit=UNITS.get(key),
        )
    for key, raw in clock["native_clock"].items():
        add(
            "objective" if key.startswith("bomb_") else "frame",
            key,
            raw,
            f"/packet/round_context/{index}/native_clock/{key}",
            scope="public_context",
            unit=UNITS.get(key),
        )
    for key in ("allies", "opponents"):
        add(
            "players",
            key + "_count",
            packet["team_status"][index][key],
            f"/packet/team_status/{index}/{key}",
            scope="public_aggregate",
            unit="players",
        )
    add(
        "information",
        "perception_proven",
        value["perception_proven"],
        "/perception_proven",
        scope="retrospective_not_perception",
    )
    return result


def _key(fact):
    return fact["group"], fact["subject"], fact["key"]


def _meaning(fact):
    value = deepcopy(fact["value"])
    if isinstance(value, dict):
        value.pop("source_sequence", None)
    return fact["status"], value, fact["reason"]


def _index(value, through_index):
    n = len(value["packet"]["context"]["samples"])
    index = n - 1 if through_index is None else through_index
    require(
        type(index) is int and 0 <= index < n, "common context exact index unavailable"
    )
    return index


def build(value, *, through_index=None, native=None):
    candidate.validate(value)
    end = _index(value, through_index)
    samples = value["packet"]["context"]["samples"]
    previous, evolution = {}, []
    situation = None
    for i in range(end + 1):
        situation = _at(value, i)
        current = {_key(f): f for group in situation.values() for f in group}
        if i:
            changes = []
            for key in sorted(previous.keys() | current.keys(), key=str):
                before, after = previous.get(key), current.get(key)
                if (
                    before is None
                    or after is None
                    or _meaning(before) != _meaning(after)
                ):
                    changes.append(
                        {
                            "group": key[0],
                            "subject": key[1],
                            "key": key[2],
                            "before": [] if before is None else before["evidence"],
                            "after": [] if after is None else after["evidence"],
                        }
                    )
            old_actors = {
                a["actor_alias"]: a["identity"] for a in samples[i - 1]["actors"]
            }
            new_actors = {a["actor_alias"]: a["identity"] for a in samples[i]["actors"]}
            gap = (
                samples[i]["continuity"] != "adjacent_samples"
                or samples[i]["cut"]["tick"] - samples[i - 1]["cut"]["tick"]
                != samples[i]["cut"]["sample_step"]
            )
            evolution.append(
                {
                    "from_index": i - 1,
                    "to_index": i,
                    "cut": deepcopy(samples[i]["cut"]),
                    "continuity": samples[i]["continuity"],
                    "comparison": "boundary_only"
                    if gap or old_actors != new_actors
                    else "adjacent_observations",
                    "changes": changes,
                }
            )
        previous = current
    events = []
    for i, row in enumerate(value["events"]):
        available = row["event"]["available_index"]
        if available > end:
            continue
        fact = _fact(
            "information",
            row["event"]["id"],
            row["event"],
            [f"/events/{i}/event", f"/events/{i}/availability_cut"],
            row["availability_cut"],
            available,
            scope="retrospective_event_not_current_contact",
        )
        fact["seconds_since_event"] = (
            samples[end]["cut"]["tick"] - row["event"]["tick"]
        ) / samples[end]["cut"]["tick_rate"]
        events.append(fact)
    later = []
    if native is not None:
        places.validate(native, value)
        require(
            end == len(samples) - 1, "native supplement is only bound to the final cut"
        )
        for i, row in enumerate(native["observations"]):
            later.append(
                _fact(
                    "players",
                    "native_place",
                    row,
                    [f"/observations/{i}", "/provenance"],
                    samples[end]["cut"],
                    end,
                    subject=row["actor_alias"],
                    status=row["value_status"],
                    scope="native_property_not_geometric_zone",
                    document="native",
                    original=False,
                    unit="native_string",
                )
            )
    uncertainty = [
        {
            "group": f["group"],
            "subject": f["subject"],
            "key": f["key"],
            "status": f["status"],
            "reason": f["reason"],
            "evidence": f["evidence"],
        }
        for group in (*situation.values(), later)
        for f in group
        if f["status"] not in ("observed", "derived", "current")
    ]
    result = {
        "schema_id": SCHEMA_ID,
        "contract_version": "1.0.0",
        "source": {
            "candidate_context_id": value["candidate_context_id"],
            "instrument": value["instrument"],
            "source_demo_sha256": value["packet"]["context"]["source_sha256"],
            "native_content_id": None
            if native is None
            else native["native_place_context_id"],
        },
        "through_index": end,
        "cut": deepcopy(samples[end]["cut"]),
        "situation": situation,
        "evolution": evolution,
        "events": events,
        "uncertainty": uncertainty,
        "later_supplements": later,
        "history": {
            "sample_count": end + 1,
            "start_cut": deepcopy(samples[0]["cut"]),
            "prior_rounds_status": "not_supplied",
            "native_status": "not_supplied" if native is None else native["status"],
        },
        "interpretations": [],
        "recommendations": [],
        "limitations": list(LIMITATIONS),
        "training_allowed": False,
        "product_supported": False,
    }
    result["common_context_id"] = candidate.content_hash(result)
    _validator().validate(result)
    return result


def validate(result, value, *, native=None):
    _validator().validate(result)
    require(
        result == build(value, through_index=result["through_index"], native=native),
        "common context facts, references or summary differ",
    )
    return result


def sample(result, value, index, *, native=None):
    """Detailed history access never returns a sample after the selected cut."""
    require(
        type(index) is int and 0 <= index <= result["through_index"],
        "history request outside causal frontier",
    )
    validate(result, value, native=native)
    return deepcopy(_at(value, index))


def evidence(result, value, ref, *, native=None):
    """Only cited paths are accessible; raw retained evidence is not a prompt."""
    validate(result, value, native=native)
    allowed = []
    for group in result["situation"].values():
        allowed.extend(r for f in group for r in f["evidence"])
    allowed.extend(
        r for f in result["events"] + result["later_supplements"] for r in f["evidence"]
    )
    for change in result["evolution"]:
        allowed.extend(r for f in change["changes"] for r in f["before"] + f["after"])
    require(ref in allowed, "evidence request not cited within causal frontier")
    return deepcopy(
        _pointer(value if ref["document"] == "candidate" else native, ref["pointer"])
    )
