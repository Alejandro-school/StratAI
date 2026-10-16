import copy
import math
from pathlib import Path

import pytest

from ai_coach.support import anchors as a
from ai_coach.support import navigation as n
from ai_coach.support.navigation_verify import GraphVerifier
from ai_coach.tests.test_support_anchors import REF, fixture
from ai_coach.tests.test_support_physical import build as physical_report


def vector(x=0, y=0, z=0):
    return {"X": x, "Y": y, "Z": z}


def graph():
    nodes = [
        {
            "ID": i + 1,
            "Position": vector(x),
            "Corners": [
                vector(x - 50, -50),
                vector(x + 50, -50),
                vector(x + 50, 50),
                vector(x - 50, 50),
            ],
        }
        for i, x in enumerate((0, 200, 400))
    ]
    return {
        "Version": "nav_area_connections_with_polygons@2",
        "AssetReference": "de_mirage/de_mirage.nav",
        "AssetSHA256": "b" * 64,
        "AvailabilityTick": 0,
        "Nodes": nodes,
        "Edges": [
            {"FromID": 1, "ToID": 2, "Weight": 200},
            {"FromID": 2, "ToID": 3, "Weight": 200},
        ],
    }


def trace(
    origin,
    target,
    distance=None,
    points=None,
    reason="",
    offsets=(0, 0),
    reachable=True,
):
    return {
        "Measurement": {
            "DirectDistance": math.dist(tuple(origin.values()), tuple(target.values())),
            "Reachable": reachable,
            "PathDistance": distance,
            "OriginAssociationDistance": offsets[0],
            "TargetAssociationDistance": offsets[1],
            "UnavailableReason": reason,
        },
        "Points": points or [],
    }


def data_fixture():
    support = fixture()
    anchors = a.build_report(support, REF)
    selected = anchors["anchors"]
    qs = n.queries(selected)
    g = graph()
    results = [
        trace(
            q["origin"],
            q["target"],
            12,
            [q["origin"], q["target"]],
            offsets=(0, 12) if i == 0 else (12, 0),
        )
        for i, q in enumerate(qs)
    ]
    refs = {
        k: REF
        for k in (
            "physical_source",
            "support_source",
            "anchors",
            "physical_context",
            "base_manifest",
            "evaluator",
        )
    }
    data = {
        "schema_id": "stratai.support_navigation_observations@1",
        "contract_version": n.VERSION,
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "map_name": "de_mirage",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": refs,
        "asset": {
            "reference": g["AssetReference"],
            "sha256": "b" * 64,
            "bytes": 100,
            "magic": 0xFEEDFACE,
            "format_version": 35,
            "subversion": 1,
            "status": "candidate_loaded",
            "demo_binding": "not_accredited",
        },
        "limitations": n.LIMITS.copy(),
        "algorithm": "directed_dijkstra_stable_ties@1",
        "units": "world_units",
        "association_method": "containing_nav_polygon_xy_then_nearest_center_3d@2",
        "endpoint_tolerance": 96,
        "endpoint_vertical_tolerance": 128,
        "anchor_keys": [{k: row[k] for k in n.KEYS} for row in selected],
        "queries": qs,
        "graph": g,
        "results": results,
    }
    manifest = {
        "lineage": {
            "map_name": "de_mirage",
            "input_hashes": {
                "nav_mesh": {
                    "status": "observed",
                    "value": "b" * 64,
                    "source": "map_manager:de_mirage/de_mirage.nav",
                }
            },
        }
    }
    return support, anchors, data, manifest, {**refs, "navigation_source": REF}


def test_same_polygon_is_a_centre_diagnostic_not_a_physical_claim():
    s, anchors, data, manifest, refs = data_fixture()
    report = n.build_report(data, s, anchors, physical_report(s), refs, manifest)
    assert report["summary"]["candidate_connected"] == 2
    assert report["summary"]["physical_connections_observed"] == 0
    for row in report["rows"]:
        assert row["candidate_diagnostic"]["association"]["same_polygon"]
        assert row["initial"]["reachable"] is None
        assert (
            n.initial_features(
                s, row["anchor_id"], row["actor_alias"], row["t0_tick"], row["t0_frame"]
            )
            == row["initial"]
        )


def test_independent_directed_path_and_no_path():
    v = GraphVerifier(graph())
    start, end = vector(), vector(400)
    q = {"origin": start, "target": end}
    v.verify(q, trace(start, end, 400, [start, vector(200), end]))
    v.verify(
        {"origin": end, "target": start},
        trace(end, start, reason="no_navigable_connection", reachable=False),
    )
    bad = trace(start, end, 400, [start, end])
    with pytest.raises(ValueError, match="segment"):
        v.verify(q, bad)


@pytest.mark.parametrize(
    "origin,target,reason",
    [
        (9999, 0, "origin_outside_nav_tolerance"),
        (0, 9999, "target_outside_nav_tolerance"),
        (9999, 9999, "both_endpoints_outside_nav_tolerance"),
    ],
)
def test_outside_coverage_is_unknown_not_false(origin, target, reason):
    start, end = vector(origin), vector(target)
    v = GraphVerifier(graph())
    t = trace(start, end, reason=reason, reachable=None, offsets=(None, None))
    v.verify({"origin": start, "target": end}, t)
    t["Measurement"]["Reachable"] = False
    with pytest.raises(ValueError):
        v.verify({"origin": start, "target": end}, t)


def test_association_vertical_layers_edges_and_nearest_are_only_candidates():
    v = GraphVerifier(graph())
    assert v.associate((50, 50, 128))[2] == "polygon_xy_candidate"
    assert v.associate((50, 50, 129))[0] is None
    assert v.associate((90, 0, 0))[2] == "nearest_center_candidate"
    assert v.associate((100, 0, 0))[0] is None
    g = graph()
    layer = copy.deepcopy(g["Nodes"][0])
    layer["ID"] = 4
    layer["Position"]["Z"] = 100
    for c in layer["Corners"]:
        c["Z"] = 100
    g["Nodes"].append(layer)
    assert GraphVerifier(g).associate((0, 0, 80))[0] == 4


@pytest.mark.parametrize(
    "mutation",
    [
        "map",
        "hash",
        "reference",
        "format",
        "graph_hash",
        "future_asset",
        "missing_sample",
        "future_target",
        "actor",
        "life",
        "epoch",
        "side",
        "tick",
        "frame",
        "distance",
        "oracle",
        "gate",
    ],
)
def test_tampered_source_causal_join_and_claims_are_rejected(mutation):
    s, anchors, data, manifest, refs = data_fixture()
    if mutation == "map":
        data["map_name"] = "de_nuke"
    elif mutation == "hash":
        data["asset"]["sha256"] = "c" * 64
    elif mutation == "reference":
        data["asset"]["reference"] = "de_nuke/de_nuke.nav"
    elif mutation == "format":
        data["asset"]["subversion"] = 99
    elif mutation == "graph_hash":
        data["graph"]["AssetSHA256"] = "c" * 64
    elif mutation == "future_asset":
        data["graph"]["AvailabilityTick"] = 9999
    elif mutation == "missing_sample":
        data["results"].pop()
    elif mutation == "future_target":
        data["queries"][0]["target"]["X"] += 5
    elif mutation in ("actor", "life", "epoch", "side", "tick", "frame"):
        key = {
            "actor": "actor_alias",
            "life": "actor_life",
            "epoch": "epoch",
            "side": "side",
            "tick": "t0_tick",
            "frame": "t0_frame",
        }[mutation]
        data["anchor_keys"][0][key] = (
            "changed" if isinstance(data["anchor_keys"][0][key], str) else 9999
        )
    elif mutation == "distance":
        data["results"][0]["Measurement"]["PathDistance"] += 1
    elif mutation == "oracle":
        data["oracle"] = {}
    else:
        data["training_allowed"] = True
    with pytest.raises((ValueError, __import__("jsonschema").ValidationError)):
        n.build_report(data, s, anchors, physical_report(s), refs, manifest)


@pytest.mark.parametrize("kind", ["asset_absent", "unsupported_format"])
def test_unavailable_asset_retains_matrix_and_all_anchors(kind):
    s, anchors, data, manifest, refs = data_fixture()
    data["graph"], data["results"] = None, []
    data["asset"]["status"] = kind
    if kind == "asset_absent":
        for k in ("sha256", "bytes", "magic", "format_version", "subversion"):
            data["asset"][k] = None
    else:
        data["asset"]["format_version"] = 36
    report = n.build_report(data, s, anchors, physical_report(s), refs, manifest)
    assert report["summary"]["anchors"] == report["summary"]["unavailable"] == 2
    assert report["summary"]["physical_absences_accredited"] == 0


@pytest.mark.parametrize(
    "change", ["future", "enemy", "wrong_t0", "life", "epoch", "side"]
)
def test_initial_access_uses_only_unchanged_causal_self_ally_prefix(change):
    s, anchors, _, _, _ = data_fixture()
    row = anchors["anchors"][0]
    args = (row["anchor_id"], row["actor_alias"], row["t0_tick"], row["t0_frame"])
    expected = n.initial_features(s, *args)
    if change == "future":
        for f in s["observations"]["frames"][3:]:
            f["oracle_outcome"] = "ignored"
    elif change == "enemy":
        for f in s["observations"]["frames"]:
            f["players"][2]["position"]["x"] += 100
    elif change == "wrong_t0":
        args = (*args[:2], args[2] + 1, args[3])
    elif change == "life":
        s["observations"]["frames"][2]["players"][0]["pawn_life"] = (
            "support-pawn-999999"
        )
    elif change == "epoch":
        s["observations"]["frames"][2]["epoch"] += 1
    else:
        s["observations"]["frames"][2]["players"][0]["side"] = "T"
    if change in ("future", "enemy"):
        assert n.initial_features(s, *args) == expected
        assert "support-actor" not in json_text(expected)
    else:
        with pytest.raises(ValueError):
            n.initial_features(s, *args)


def json_text(value):
    import json

    return json.dumps(value)


def test_destination_protected_before_loading_or_executing(tmp_path, monkeypatch):
    (tmp_path / "retained").write_text("keep")
    monkeypatch.setattr(
        n.p, "materialize", lambda *args, **kwargs: pytest.fail("ran before protection")
    )
    with pytest.raises(ValueError):
        n.materialize(Path("missing"), tmp_path, Path("missing"), Path("missing"))
    with pytest.raises(ValueError):
        n.materialize(Path("missing"), Path.cwd(), Path("missing"), Path("missing"))
