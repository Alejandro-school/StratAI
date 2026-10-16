"""A local NAV graph diagnostic must never become physical-route evidence."""

import copy

import pytest

from ai_coach.contracts.spatial_candidate import (
    SpatialCandidateError,
    compact,
    decode,
    validate_candidate_example,
)
from ai_coach.datasets.build_examples import process_manifest
from ai_coach.datasets.spatial_candidate import (
    ROOT,
    audit_spatial_candidate,
    fact_status,
    load_spatial_candidate,
)
from ai_coach.tests.test_spatial_candidate import MAPS, _materialize, _source_fixture

NAV = "FLD-NAV-EXIT-PATH"


@pytest.fixture
def materialized(tmp_path):
    _source_fixture(tmp_path)
    path = _materialize(tmp_path)
    candidate = load_spatial_candidate(path, maps_root=MAPS)
    examples = []
    process_manifest(
        candidate.base, catalog_required_fields=[], example_sink=examples.append
    )
    return path, candidate, examples[0]


@pytest.mark.parametrize("reachability", [True, False, None, "missing"])
def test_nav_graph_values_survive_without_physical_credit(materialized, reachability):
    _, candidate, base = materialized
    baseline = candidate.attach(base)
    for join in candidate.joins.values():
        if join["field_id"] != NAV:
            continue
        key = (join["round_number"], join["t0_tick"], join["participant_alias"])
        if reachability == "missing":
            candidate.indexes[NAV].pop(key, None)
            join.update(
                status="missing",
                reason_codes=["no_exact_t0_sample"],
                row_line=None,
                sample_tick=None,
                availability_tick=None,
            )
        else:
            row = candidate.indexes[NAV][key][1]
            objective = row["candidates"][0]["objective_viability"]
            objective["reachability"] = {
                "status": "unavailable" if reachability is None else "derived",
                "value": reachability,
            }
            objective["reason_codes"] = (
                ["nav_graph_unavailable"] if reachability is None else []
            )
            join["status"], join["reason_codes"] = fact_status(NAV, row)

    example = candidate.attach(base)
    fact = example["observed"]["spatial_facts"][NAV]
    assert fact["status"] == "unavailable" and fact["values"] == {}
    assert candidate.field_status(example, NAV) == (
        "missing" if reachability == "missing" else "null"
    )
    metadata = example["spatial_evidence"][NAV]
    diagnostic = metadata["diagnostic"]
    assert diagnostic["scope"] == "partial_nav_graph"
    assert "nav_physical_connection_not_accredited" in metadata["reason_codes"]
    if reachability == "missing":
        assert diagnostic["status"] == "missing" and diagnostic["values"] == {}
        assert metadata["row_reference"] is None
    else:
        assert diagnostic["status"] == ("null" if reachability is None else "present")
        key = (
            base["join_keys"]["round_number"],
            base["t0_tick"],
            example["join_keys"]["actor_player_id"],
        )
        row = candidate.indexes[NAV][key][1]
        expected = {
            key: copy.deepcopy(row[key])
            for key in ("sample", "candidates", "posture_context")
        }
        for item in expected["candidates"]:
            item.pop("tactical_assessment")
        assert diagnostic["values"] == expected
        assert metadata["provenance"] == row["provenance"]
        assert metadata["row_reference"]["line"] == candidate.indexes[NAV][key][0]
    for field in ("FLD-GEOMETRIC-CLEARANCE-SECTORS", "FLD-GEOMETRIC-OCCLUSION"):
        assert (
            example["observed"]["spatial_facts"][field]
            == baseline["observed"]["spatial_facts"][field]
        )
        assert example["spatial_evidence"][field] == baseline["spatial_evidence"][field]
    assert example["training_allowed"] is False
    validate_candidate_example(example)


def test_nav_cannot_be_promoted_by_changing_status_or_using_legacy_manifest(
    materialized,
):
    path, candidate, base = materialized
    example = candidate.attach(base)
    assert example["observed"]["spatial_facts"][NAV]["status"] == "unavailable"
    forged = copy.deepcopy(example)
    forged["observed"]["spatial_facts"][NAV].update(
        status="derived", values={"reachable": True}
    )
    forged["spatial_evidence"][NAV]["status"] = "present"
    with pytest.raises(SpatialCandidateError):
        validate_candidate_example(forged)
    future = copy.deepcopy(example)
    future["spatial_evidence"][NAV]["diagnostic"]["values"]["posture_context"][
        "availability_tick"
    ] = example["t0_tick"] + 1
    with pytest.raises(SpatialCandidateError, match="temporal violation"):
        validate_candidate_example(future)
    injected = copy.deepcopy(example)
    injected["spatial_evidence"][NAV]["diagnostic"]["values"]["posture_context"][
        "oracle"
    ] = "future"
    with pytest.raises(SpatialCandidateError, match="schema mismatch"):
        validate_candidate_example(injected)
    manifest = decode(path.read_bytes())
    manifest.update(
        schema_id="stratai.canonical_spatial_candidate_manifest@1",
        export_format_version="3.9.0-candidate.1",
        contract_version="1.0.0",
        loader_schema_id="stratai.spatial_candidate_loader@1",
        validator_schema_id="stratai.spatial_candidate_validator@1",
    )
    path.write_bytes(compact(manifest))
    with pytest.raises(SpatialCandidateError, match="schema mismatch"):
        load_spatial_candidate(path, maps_root=MAPS)


def test_eligibility_uses_physical_availability_and_retains_diagnostic_counts(
    materialized,
):
    _, candidate, _ = materialized
    audit = audit_spatial_candidate(candidate)
    for family in audit["field_coverage"][NAV].values():
        assert family["present"] == 0 and family["null"] == 1
    for family in audit["spatial_by_family"].values():
        assert family[NAV]["present"] == 1
        assert family[NAV]["scope"] == "partial_nav_graph"
    catalog = decode((ROOT / "ai_coach/contracts/capability_catalog.json").read_bytes())
    required = {
        cap["id"]
        for cap in catalog["capabilities"]
        if NAV in cap["task_eligibility"]["required_fields"]
    }
    assert required
    for task in audit["task_eligibility"]:
        if task["capability_id"] in required:
            assert (
                NAV in task["missing_required_fields"] and task["status"] != "eligible"
            )
    assert audit["training_allowed"] is audit["product_supported"] is False
