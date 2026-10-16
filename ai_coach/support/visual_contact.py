"""Exact support visual inputs; physics diagnostics never unlock hidden state."""

import argparse
import copy
import json
import math
import subprocess
from bisect import bisect_right
from collections import Counter
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.support import anchors as a
from ai_coach.support import native_motion as m
from ai_coach.support import navigation as n
from ai_coach.support import objective as o
from ai_coach.support import objective_position as z
from ai_coach.support import physical as p
from ai_coach.support import weapon_timing as t

DATA = "support_visual_contact_observations.json"
REPORT = "support_visual_contact_context.json"
PROBES = [
    "m_vecX",
    "m_vecY",
    "m_vecZ",
    "m_pCameraServices.m_iFOV",
    "m_pCameraServices.m_iFOVStart",
    "m_pCameraServices.m_flFOVTime",
    "m_pCameraServices.m_flFOVRate",
    "m_flFlashDuration",
    "m_flFlashMaxAlpha",
    "m_bIsScoped",
]
COORDINATES = ["CBodyComponent.m_cell" + axis for axis in "XYZ"] + [
    "CBodyComponent.m_vec" + axis for axis in "XYZ"
]
PLAYER = (
    "native_eye_offset",
    "reference_fov_degrees",
    "flash_clear",
    "smoke_clear",
    "visual_contact",
)
FIELDS = tuple(role + "." + name for role in ("self", "ally") for name in PLAYER)
DIAGNOSTICS = (
    "static_ray",
    "reference_fov",
    "smoke_occlusion",
    "flash_impairment",
    "legal_contact",
)
LIMITS = [
    *t.LIMITS,
    "static_collision_mesh_does_not_accredit_rendered_or_dynamic_occlusion",
    "smoke_volume_and_delivery_coverage_require_positive_source",
    "native_flash_duration_and_max_alpha_are_not_current_rendered_opacity",
    "canonical_fov_proxy_is_not_a_recording_of_the_players_viewport",
    "ally_contact_is_not_subjective_self_knowledge",
    "contact_does_not_establish_attention_trade_opportunity_or_tactical_superiority",
]


def validate_source(data, support, timing):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(
            data[key] == support[key] == timing[key],
            "visual source identity mismatch",
        )
    obs = data["observations"]
    loaded = bool(data["geometry"]["assets"])
    roles = [x["role"] for x in data["geometry"]["assets"]]
    a.require(
        not loaded or (roles.count("physics_gltf") == 1 and "physics_buffer" in roles),
        "visual geometry asset coverage incomplete",
    )
    deliveries, states, cuts = (obs[k] for k in ("deliveries", "states", "cuts"))
    created, last, destroyed, delivery_frames = {}, {}, set(), []
    for seq, d in enumerate(deliveries, 1):
        a.require(
            d["sequence"] == seq
            and (not delivery_frames or d["frame"] >= delivery_frames[-1]),
            "visual delivery order mismatch",
        )
        delivery_frames.append(d["frame"])
        life, name, kind = d["entity_life"], d["property"], d["kind"]
        a.require(life not in destroyed, "visual delivery after destruction")
        z.validate_scalar(d["previous"])
        z.validate_scalar(d["value"])
        if kind == "destroyed":
            a.require(
                life in created
                and name == ""
                and d["previous"]["status"] == "absent"
                and d["value"]["status"] == "absent",
                "invalid visual destruction",
            )
            destroyed.add(life)
            continue
        a.require(name in PROBES, "visual field outside probe contract")
        key = life, name
        if kind == "created":
            a.require(
                key not in last
                and created.setdefault(life, d["frame"]) == d["frame"]
                and d["previous"]["status"] == "absent",
                "invalid visual creation",
            )
        else:
            a.require(
                key in last and d["previous"] == last[key],
                "visual predecessor mismatch",
            )
            a.require(
                (kind == "creation_replay") == (d["frame"] == created[life]),
                "visual creation replay misclassified",
            )
        last[key] = d["value"]
    available, ended, frontier, state_frontier, identities = {}, set(), 0, 0, {}
    previous_cuts = timing["observations"]["cuts"]
    a.require(
        len(cuts) == len(support["observations"]["frames"]) == len(previous_cuts),
        "visual cut coverage mismatch",
    )
    for seq, (cut, base, previous) in enumerate(
        zip(cuts, support["observations"]["frames"], previous_cuts, strict=True), 1
    ):
        a.require(
            cut["source_sequence"]
            == seq
            == base["sequence"]
            == previous["source_sequence"]
            and all(
                cut[k] == base[k] == previous[k]
                for k in ("frame", "tick", "instance", "epoch")
            )
            and cut["clock_order"] == previous["clock_order"],
            "visual exact cut mismatch",
        )
        a.require(
            cut["geometry_loaded"] == loaded, "visual geometry provenance mismatch"
        )
        end = bisect_right(delivery_frames, cut["frame"])
        a.require(cut["delivery_sequence"] == end, "visual delivery frontier mismatch")
        for d in deliveries[frontier:end]:
            if d["kind"] == "destroyed":
                ended.add(d["entity_life"])
            else:
                available[d["entity_life"], d["property"]] = d
        frontier = end
        a.require(
            state_frontier <= cut["state_sequence"] <= len(states),
            "visual state frontier mismatch",
        )
        new_states = set(range(state_frontier + 1, cut["state_sequence"] + 1))
        used = set()
        a.require(
            len(cut["players"]) == len(base["players"]), "visual player census mismatch"
        )
        for player, actor in zip(cut["players"], base["players"], strict=True):
            a.require(
                player["actor_alias"] == actor["actor_alias"],
                "visual actor order mismatch",
            )
            present = [
                player[k] is not None
                for k in ("pawn_handle", "pawn_life", "state_sequence")
            ]
            a.require(all(present) or not any(present), "partial visual pawn reference")
            if not any(present):
                a.require(
                    player["coordinates"] == {}, "visual coordinates without pawn"
                )
                continue
            a.require(
                actor["connected"]
                and actor["identity_status"] == "observed"
                and actor["alive"] is True
                and actor["alive_status"] == "observed"
                and actor["position_status"] == "observed"
                and actor["position"] is not None
                and (player["pawn_life"], player["pawn_handle"])
                == (actor["pawn_life"], actor["pawn_handle"]),
                "visual pawn is not current support actor",
            )
            state_seq = player["state_sequence"]
            a.require(
                1 <= state_seq <= cut["state_sequence"], "visual state outside prefix"
            )
            used.add(state_seq)
            state = states[state_seq - 1]
            a.require(
                state["sequence"] == state_seq
                and state["first_cut"] <= seq
                and (state["instance"], state["epoch"])
                == (cut["instance"], cut["epoch"])
                and state["handle"] == player["pawn_handle"],
                "visual state provenance mismatch",
            )
            if state_seq in new_states:
                a.require(state["first_cut"] == seq, "visual first cut mismatch")
            life = state["entity_life"]
            a.require(
                life in created
                and life not in ended
                and state["created_frame"] == created[life],
                "visual state lifetime mismatch",
            )
            identity = (
                state["handle"],
                state["class_name"],
                state["created_frame"],
                player["pawn_life"],
            )
            a.require(
                identities.setdefault(life, identity) == identity,
                "visual pawn identity changed",
            )
            a.require(
                identities.setdefault((player["pawn_life"], "native_life"), life)
                == life,
                "visual native entity substituted within support life",
            )
            a.require(
                set(state["fields"]) == set(state["field_sequences"]) == set(PROBES),
                "visual probe fields mismatch",
            )
            for name, value in state["fields"].items():
                d = available.get((life, name))
                a.require(
                    d is not None
                    and value == d["value"]
                    and state["field_sequences"][name] == d["sequence"],
                    "visual snapshot not equal to delivered prefix",
                )
            a.require(
                set(player["coordinates"]) == set(COORDINATES),
                "visual coordinate census mismatch",
            )
            for value in player["coordinates"].values():
                z.validate_scalar(value)
            validate_coordinates(player["coordinates"], actor)
        a.require(new_states <= used, "unused visual state at first cut")
        validate_rays(cut, base, states)
        state_frontier = cut["state_sequence"]
    a.require(state_frontier == len(states), "visual trailing state")


def native_float(value, declared=("CNetworkedQuantizedFloat", "float32")):
    if (
        value["status"] == "observed"
        and value["native_type"] in ("float32", "float64")
        and value["declared_type"] in declared
        and type(value["value"]) in (int, float)
        and math.isfinite(value["value"])
    ):
        return value["value"]
    return None


def cell_position(coords):
    xyz = []
    for axis in "XYZ":
        cell, offset = (
            coords["CBodyComponent.m_cell" + axis],
            coords["CBodyComponent.m_vec" + axis],
        )
        value = z.raw_integer(cell)
        fraction = native_float(offset, ("CNetworkedQuantizedFloat",))
        if (
            value is None
            or value > 65535
            or cell["declared_type"] != "uint16"
            or fraction is None
        ):
            return None
        xyz.append(value * 512 - 16384 + fraction)
    return xyz


def validate_coordinates(coords, actor):
    xyz = cell_position(coords)
    if xyz is None:
        return
    # Replay X/Y use Go math.Round (ties away from zero), not Python bankers rounding.
    rounded = [math.copysign(math.floor(abs(x) + 0.5), x) for x in xyz[:2]]
    a.require(
        actor["position"] is not None
        and rounded == [actor["position"]["x"], actor["position"]["y"]]
        and math.isclose(xyz[2], actor["position"]["z"], rel_tol=0, abs_tol=1e-6),
        "visual native coordinate/replay mismatch",
    )


def eye_offset(state):
    values = [
        native_float(state["fields"]["m_vec" + axis], ("CNetworkedQuantizedFloat",))
        for axis in "XYZ"
    ]
    return values if all(x is not None for x in values) else None


def diagnostic_eye(player, state):
    position, offset = cell_position(player["coordinates"]), eye_offset(state)
    if position is None or offset is None:
        return None
    return [x + y for x, y in zip(position, offset, strict=True)]


def validate_rays(cut, base, states):
    expected = []
    eyes = {}
    for i, (player, actor) in enumerate(
        zip(cut["players"], base["players"], strict=True)
    ):
        if player["state_sequence"] is None or actor["side"] not in ("T", "CT"):
            continue
        eyes[i] = diagnostic_eye(player, states[player["state_sequence"] - 1])
        for j, target in enumerate(base["players"]):
            if (
                cut["players"][j]["state_sequence"] is not None
                and target["side"] in ("T", "CT")
                and actor["side"] != target["side"]
            ):
                expected.append((i, j))
    a.require(
        [(r["observer_index"], r["subject_index"]) for r in cut["rays"]] == expected,
        "visual ray census/order mismatch",
    )
    for ray in cut["rays"]:
        usable = (
            cut["geometry_loaded"]
            and eyes[ray["observer_index"]] is not None
            and eyes[ray["subject_index"]] is not None
        )
        a.require(
            (ray["clear"] is not None) == usable,
            "visual ray without exact inputs/geometry",
        )


def reference_fov(state):
    fields = state["fields"]
    target, start = (
        fields["m_pCameraServices." + name] for name in ("m_iFOV", "m_iFOVStart")
    )
    values = [
        z.raw_integer(x) if x["declared_type"] == "uint32" else None
        for x in (target, start)
    ]
    if any(x is None for x in values):
        return o.fact(reason="native_camera_fov_unavailable")
    if not 0 < values[0] < 180 or not 0 < values[1] < 180:
        return o.fact(reason="native_camera_fov_default_or_range_unaccredited")
    if values[0] != values[1]:
        return o.fact(reason="native_camera_fov_transition_unaccredited")
    scoped = o.native(fields["m_bIsScoped"])
    if scoped["value"] is not False:
        return o.fact(reason="scoped_optics_or_scope_state_unaccredited")
    return o.fact(values[0])


def fov_contains(origin, target, angles, fov):
    """Same declared 16:9 rectangular proxy; no guessed FOV/default or angles."""
    if (
        origin is None
        or target is None
        or angles["status"] != "observed"
        or fov["status"] != "observed"
    ):
        return None
    yaw, pitch = angles["value"]
    if (
        not all(math.isfinite(x) for x in (yaw, pitch))
        or not -180 <= yaw < 360
        or not -180 <= pitch < 360
    ):
        return None
    pitch = (pitch + 180) % 360 - 180
    if not -90 <= pitch <= 90:
        return None
    pitch, yaw = math.radians(pitch), math.radians(yaw)
    forward = [
        math.cos(pitch) * math.cos(yaw),
        math.cos(pitch) * math.sin(yaw),
        -math.sin(pitch),
    ]
    right = [-math.sin(yaw), math.cos(yaw), 0]
    up = [
        math.sin(pitch) * math.cos(yaw),
        math.sin(pitch) * math.sin(yaw),
        math.cos(pitch),
    ]
    delta = [x - y for x, y in zip(target, origin, strict=True)]
    dot = lambda v: sum(x * y for x, y in zip(v, delta, strict=True))
    depth = dot(forward)
    if depth <= 0:
        return False
    vertical = math.tan(math.radians(fov["value"]) / 2) / (4 / 3)
    return (
        abs(dot(right) / depth) <= vertical * (16 / 9)
        and abs(dot(up) / depth) <= vertical
    )


def native_facts(state):
    facts = {
        "native_eye_offset": o.fact(reason="native_eye_offset_unavailable"),
        "reference_fov_degrees": o.fact(reason="native_camera_fov_unavailable"),
        "flash_clear": o.fact(reason="rendered_flash_opacity_unaccredited"),
        "smoke_clear": o.fact(reason="smoke_volume_and_coverage_unaccredited"),
        "visual_contact": o.fact(reason="legal_visual_contact_inputs_unaccredited"),
    }
    if state is not None:
        offset = eye_offset(state)
        if offset is not None:
            facts["native_eye_offset"] = o.fact(dict(zip("xyz", offset, strict=True)))
        facts["reference_fov_degrees"] = reference_fov(state)
    return facts


def initial_context(anchor, data):
    cut = data["observations"]["cuts"][anchor["source_sequence"] - 1]
    initial, refs = {}, {}
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        player = next(x for x in cut["players"] if x["actor_alias"] == alias)
        seq = player["state_sequence"]
        state = data["observations"]["states"][seq - 1] if seq else None
        initial[role] = native_facts(state)
        refs[role] = {
            "source_sequence": cut["source_sequence"],
            "frame": cut["frame"],
            "state_sequence": seq,
            "pawn_life": player["pawn_life"],
        }
    return initial, refs


def prefix_source(data, seq):
    return m.prefix_source(data, seq)


def initial_features(support, timing, data, anchor_id, actor, tick, frame):
    observed = a.initial_features(support, anchor_id, actor, tick, frame)
    seq = observed["history"][-1]["source_sequence"]
    support = {
        **support,
        "observations": {
            "frames": [
                f for f in support["observations"]["frames"] if f["sequence"] <= seq
            ]
        },
    }
    timing, data = t.prefix_source(timing, seq), prefix_source(data, seq)
    validate_source(data, support, timing)
    anchor = next(
        x
        for x in a.select_anchors(
            support["observations"]["frames"], support["source_alias"]
        )[0]
        if x["anchor_id"] == anchor_id
    )
    return copy.deepcopy(initial_context(anchor, data)[0])


def diagnostic_counts(anchor, data, physical):
    seq = anchor["source_sequence"]
    cut, pf = (
        data["observations"]["cuts"][seq - 1],
        physical["observations"]["frames"][seq - 1],
    )
    result = {
        role: {name: Counter() for name in DIAGNOSTICS} for role in ("self", "ally")
    }
    for role, alias in (
        ("self", anchor["actor_alias"]),
        ("ally", anchor["ally_alias"]),
    ):
        index = next(
            i for i, x in enumerate(cut["players"]) if x["actor_alias"] == alias
        )
        player = cut["players"][index]
        if player["state_sequence"] is None:
            continue
        state = data["observations"]["states"][player["state_sequence"] - 1]
        origin, fov = diagnostic_eye(player, state), reference_fov(state)
        phys = pf["players"][index]["context"]
        angles = (
            phys["facts"]["view_angles"]
            if phys
            else o.fact(reason="observer_orientation_unavailable")
        )
        for ray in cut["rays"]:
            if ray["observer_index"] != index:
                continue
            target = cut["players"][ray["subject_index"]]
            target_state = data["observations"]["states"][target["state_sequence"] - 1]
            inside = fov_contains(
                origin, diagnostic_eye(target, target_state), angles, fov
            )
            result[role]["static_ray"][str(ray["clear"]).lower()] += 1
            result[role]["reference_fov"][str(inside).lower()] += 1
            for name in ("smoke_occlusion", "flash_impairment", "legal_contact"):
                result[role][name]["unavailable"] += 1
    return result


def build_report(support, physical, timing, data, previous, references):
    p.validate_source(physical, support)
    validate_source(data, support, timing)
    selected, _ = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )
    a.require(
        len(selected) == len(previous["rows"]), "visual previous row coverage mismatch"
    )
    rows, matrix = [], {}
    counts = {k: Counter() for k in FIELDS}
    diagnostics = {
        role: {name: Counter() for name in DIAGNOSTICS} for role in ("self", "ally")
    }
    for anchor, old in zip(selected, previous["rows"], strict=True):
        a.require(
            all(anchor[k] == old[k] for k in o.KEYS), "visual previous anchor mismatch"
        )
        initial, ref = initial_context(anchor, data)
        diag = diagnostic_counts(anchor, data, physical)
        for role in diagnostics:
            for name in DIAGNOSTICS:
                diagnostics[role][name].update(diag[role][name])
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
            old["competitive_round_status"] == "canonical_interval_matched"
        )
        for field in FIELDS:
            role, name = field.split(".")
            fact = initial[role][name]
            status = fact["status"] + (":" + fact["reason"] if fact["reason"] else "")
            counts[field][status] += 1
            entry["fact_status_counts"][field][status] += 1
        rows.append(
            {
                **{k: anchor[k] for k in o.KEYS},
                "competitive_round_status": old["competitive_round_status"],
                "initial": initial,
                "visual_reference": ref,
                "missing_context": LIMITS.copy(),
                "decision_eligible": False,
            }
        )
    source_counts = {name: Counter() for name in PROBES}
    for state in data["observations"]["states"]:
        for name, value in state["fields"].items():
            source_counts[name][value["status"]] += 1
    report = {
        "schema_id": "stratai.support_visual_contact_context@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": references,
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
            "native_field_state_counts": source_counts,
            "eligible_decisions": 0,
        },
        "evaluator_diagnostics": {
            "visibility_scope": "internal_physics_diagnostic_not_initial_information",
            "pair_counts_by_anchor_role": diagnostics,
        },
    }
    a.schema_validator(REPORT).validate(report)
    return report


def load_source(path):
    a.require(
        path.name == DATA and path.stat().st_size <= 2 * 1024**3,
        "invalid visual source",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    support, _, physical, _, _, _, timing, base = t.load_source(path.parent / t.DATA)
    expected = {
        **timing["references"],
        "timing_source": p.file_reference(path.parent / t.DATA),
    }
    a.require(data["references"] == expected, "visual source hash mismatch")
    validate_source(data, support, timing)
    a.require(path.read_bytes() == raw, "visual source changed")
    return support, physical, timing, data, base


def report_inputs(source, output):
    *sources, _ = load_source(source)
    previous = json.loads((output / t.REPORT).read_bytes())
    references = {
        **previous["references"],
        "visual_source": p.file_reference(source),
        "timing_context": p.file_reference(output / t.REPORT),
    }
    return *sources, previous, references


def verify_physics(source, data, maps_root, visual_evaluator):
    root = maps_root.resolve(strict=True)
    seen = set()
    for asset in data["geometry"]["assets"]:
        path = root / asset["reference"]
        a.require(not path.is_symlink(), "visual asset is a symlink")
        path = path.resolve(strict=True)
        a.require(
            path.is_relative_to(root) and path.is_file() and path not in seen,
            "visual asset escapes/duplicates maps root",
        )
        seen.add(path)
        a.require(
            a.digest(path.read_bytes()) == asset["sha256"],
            "visual geometry asset hash mismatch",
        )
    result = subprocess.run(
        [str(visual_evaluator), "--source", str(source), "--maps-root", str(root)],
        capture_output=True,
        text=True,
        check=True,
    )
    summary = json.loads(result.stdout)
    a.require(summary["mismatches"] == 0, "visual ray recomputation failed")
    return summary


def materialize(source, output, evaluator, maps_root, visual_evaluator):
    validate_output_directory(output)
    data = load_source(source)[3]
    physics = verify_physics(source, data, maps_root, visual_evaluator)
    previous = t.materialize(source.parent / t.DATA, output, evaluator, maps_root)
    report = build_report(*report_inputs(source, output))
    a.require(report["summary"]["anchors"] > 0, "empty visual context")
    with (output / REPORT).open("xb") as stream:
        stream.write(n.encode(report))
    return {"summary": report["summary"], "previous": previous, "physics": physics}


def validate_artifact(output, source, evaluator, maps_root, visual_evaluator):
    t.validate_artifact(output, source.parent / t.DATA, evaluator, maps_root)
    report = json.loads((output / REPORT).read_bytes())
    a.schema_validator(REPORT).validate(report)
    inputs = report_inputs(source, output)
    verify_physics(source, inputs[3], maps_root, visual_evaluator)
    a.require(report == build_report(*inputs), "visual report recomputation mismatch")
    return report["summary"]


def main():
    parser = argparse.ArgumentParser()
    for name in ("source", "output", "evaluator", "maps-root", "visual-evaluator"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--validate", action="store_true")
    args = parser.parse_args()
    result = (
        validate_artifact(
            args.output,
            args.source,
            args.evaluator,
            args.maps_root,
            args.visual_evaluator,
        )
        if args.validate
        else materialize(
            args.source,
            args.output,
            args.evaluator,
            args.maps_root,
            args.visual_evaluator,
        )
    )
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main()
