from __future__ import annotations

import copy
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import rule_context
from ai_coach.economy import session_payload as domain
from ai_coach.economy.candidate_io import sha
from ai_coach.tests.test_economy_rule_context import context_fixture, write_context


def fixture():
    context, _, _ = context_fixture()
    native = context["observations"]["sessions"][0]
    native["configuration_present"] = True
    native["configuration_data"] = {"present": True, "bytes": 570, "sha256": "a" * 64}
    fields = []
    for path, kind in sorted(domain.probe()[0]["fields"].items()):
        integer = kind == "int32"
        fields.append(
            {
                "path": path,
                "occurrence": 0,
                "node_index": len(fields) + 2,
                "present": True,
                "native_type": 2 if integer else 1,
                "status": "observed_native_integer"
                if integer
                else "observed_reference_name_hash",
                "value": 0 if integer else None,
                "reference_name_sha256": None if integer else sha(b"fixture-reference"),
            }
        )
    blocks = []
    for i, origin in enumerate(("manifest", "configuration_data")):
        blocks.append(
            {
                "source_path": domain.PATHS[i],
                "source": copy.deepcopy(native[origin]),
                "status": "decoded" if i else "absent",
                "format": "vbkv_alt_end_11" if i else "unverified",
                "compression": "none" if i else "unverified",
                "checksum_verified": bool(i),
                "nodes": 12 if i else 0,
                "scalars": 9 if i else 0,
                "economic_names_seen": [],
                "fields": fields if i else [],
            }
        )
    data = dict(
        schema_id="stratai.economy_session_payload@1",
        contract_version="1.0.0-candidate.1",
        **{k: context[k] for k in rule_context.IDENTITY},
        **rule_context.GATES,
        source_context={"path": rule_context.MANIFEST, "bytes": 10, "sha256": "a" * 64},
        probe_sha256=domain.probe()[1],
        economic_probe_sha256=rule_context.probe()[1],
        exact_build_asset_binding="not_accredited",
        observations={"sessions": [{"sequence": 1, "tick": 0, "blocks": blocks}]},
    )
    return data, context


def test_verified_fields_and_absence():
    data, context = fixture()
    assert domain.validate_data(data, context, require_observed=True) == {
        "sessions": 1,
        "decoded_blocks": 1,
        "native_integers": 5,
        "reference_name_hashes": 4,
        "exact_build_asset_binding": "not_accredited",
    }
    field = data["observations"]["sessions"][0]["blocks"][1]["fields"][0]
    field.update(
        present=False,
        node_index=None,
        native_type=None,
        value=None,
        status="not_seen_in_decoded_tree",
    )
    assert domain.validate_data(data, context)["native_integers"] == 4


@pytest.mark.parametrize(
    "mutation",
    [
        "tick",
        "sequence",
        "source_hash",
        "source_size",
        "source_identity",
        "probe",
        "economic_probe",
        "gate",
        "exact_binding",
        "path",
        "private_field",
        "raw_reference",
        "integer_type",
        "integer_null",
        "reference_null",
        "absent_value",
        "field_coverage",
        "partial_decode",
        "crc",
        "nodes",
        "scalars",
        "economic_names",
        "manifest_decode",
        "compression",
        "sessions",
        "read_limit",
    ],
)
def test_rejects_forged_semantics_and_provenance(mutation):
    data, context = fixture()
    row = data["observations"]["sessions"][0]
    block = row["blocks"][1]
    field = block["fields"][0]
    reference = block["fields"][2]
    if mutation in ("tick", "sequence"):
        row[mutation] += 1
    elif mutation == "source_hash":
        block["source"]["sha256"] = "b" * 64
    elif mutation == "source_size":
        block["source"]["bytes"] += 1
    elif mutation == "source_identity":
        data["source_alias"] = "demo-" + "b" * 20
    elif mutation == "probe":
        data["probe_sha256"] = "b" * 64
    elif mutation == "economic_probe":
        data["economic_probe_sha256"] = "b" * 64
    elif mutation == "gate":
        data["comparison_allowed"] = True
    elif mutation == "exact_binding":
        data["exact_build_asset_binding"] = "exact"
    elif mutation == "path":
        field["path"] = "//hostname"
    elif mutation == "private_field":
        field["hostname"] = "private"
    elif mutation == "raw_reference":
        reference["reference_name_sha256"] = "private-reference"
    elif mutation == "integer_type":
        field["native_type"] = 1
    elif mutation == "integer_null":
        field["value"] = None
    elif mutation == "reference_null":
        reference["reference_name_sha256"] = None
    elif mutation == "absent_value":
        field["present"] = False
    elif mutation == "field_coverage":
        block["fields"].pop()
    elif mutation == "partial_decode":
        block["status"] = "malformed"
    elif mutation == "crc":
        block["checksum_verified"] = False
    elif mutation == "nodes":
        block["nodes"] = 4097
    elif mutation == "scalars":
        block["scalars"] = 13
    elif mutation == "economic_names":
        block["economic_names_seen"] = ["private"]
    elif mutation == "manifest_decode":
        row["blocks"][0] = copy.deepcopy(block)
    elif mutation == "compression":
        block["compression"] = "unverified"
    elif mutation == "sessions":
        data["observations"]["sessions"].append(copy.deepcopy(row))
    elif mutation == "read_limit":
        block["status"] = "read_limit_exceeded"
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, context)


def test_future_observations_do_not_change_t0_even_at_same_tick():
    data, _ = fixture()
    before = copy.deepcopy(domain.available_sessions(data, 0, 1))
    future = copy.deepcopy(data["observations"]["sessions"][0])
    future["sequence"] = 2
    data["observations"]["sessions"].append(future)
    assert domain.available_sessions(data, 0, 1) == before
    future["sequence"], future["tick"] = 1, 1
    assert domain.available_sessions(data, 0, 1) == before
    assert domain.available_sessions(data, -1, 5) == []


def test_repeated_reference_occurrences_are_not_collapsed():
    data, context = fixture()
    block = data["observations"]["sessions"][0]["blocks"][1]
    second = copy.deepcopy(block["fields"][2])
    second.update(occurrence=1, node_index=11, reference_name_sha256=sha(b"second"))
    block["fields"].insert(3, second)
    block["scalars"] += 1
    assert domain.validate_data(data, context)["reference_name_hashes"] == 5
    second["occurrence"] = 0
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, context)
    second["occurrence"] = 1
    second["node_index"] = block["fields"][2]["node_index"]
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, context)


def test_loader_checks_complete_context_chain(tmp_path):
    context_path = write_context(tmp_path)
    context, _, _, _ = rule_context.load_candidate(context_path)
    data, _ = fixture()
    data["source_context"] = {
        "path": context_path.name,
        "bytes": context_path.stat().st_size,
        "sha256": sha(context_path.read_bytes()),
    }
    native = context["observations"]["sessions"][0]
    data["observations"]["sessions"][0]["blocks"][1] = {
        **data["observations"]["sessions"][0]["blocks"][0],
        "source_path": domain.PATHS[1],
        "source": native["configuration_data"],
    }
    path = context_path.with_name(domain.DATA)
    path.write_text(json.dumps(data))
    assert domain.load_candidate(path)[2]["decoded_blocks"] == 0
    with pytest.raises(RuntimeContractError):
        domain.load_candidate(path, require_observed=True)
    data["source_context"]["path"] = "../" + context_path.name
    path.write_text(json.dumps(data))
    with pytest.raises(RuntimeContractError):
        domain.load_candidate(path)
    data["source_context"]["path"] = context_path.name
    data["source_context"]["sha256"] = "0" * 64
    path.write_text(json.dumps(data))
    with pytest.raises(RuntimeContractError):
        domain.load_candidate(path)
