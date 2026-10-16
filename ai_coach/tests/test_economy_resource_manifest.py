from __future__ import annotations

import copy
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import resource_manifest as domain
from ai_coach.economy.candidate_io import sha
from ai_coach.tests.test_economy_rule_context import context_fixture, write_context


def fixture():
    context, _, _ = context_fixture()
    origin = context["observations"]["sessions"][0]
    origin["manifest"] = {"present": True, "bytes": 44, "sha256": "a" * 64}
    decoded = {
        "compression": "none",
        "encoded_bytes": 40,
        "decoded_bytes": 40,
        "decoded_sha256": "b" * 64,
        "outer_consumed_bits": 345,
        "outer_padding_bits": 7,
        "inner_consumed_bits": 316,
        "inner_padding_bits": 4,
        "type_name_hashes": [sha(b"vmdl")],
        "directory_name_hashes": [sha(b"models/")],
        "entries": [
            {
                "ordinal": i,
                "directory_index": 0,
                "type_index": 0,
                "reference_name_sha256": sha(b"models/fixture.vmdl"),
            }
            for i in range(2)
        ],
    }
    data = dict(
        schema_id="stratai.economy_resource_manifest@1",
        contract_version="1.0.0-candidate.1",
        **{k: context[k] for k in domain.rule_context.IDENTITY},
        **domain.rule_context.GATES,
        source_context={
            "path": domain.rule_context.MANIFEST,
            "bytes": 10,
            "sha256": "a" * 64,
        },
        codec_reference=domain.CODEC,
        native_format_version=None,
        exact_build_asset_binding="not_accredited",
        economic_rules="not_accredited",
        sessions=[
            {
                "sequence": origin["sequence"],
                "tick": origin["tick"],
                "source_path": "CSVCMsg_ServerInfo.game_session_manifest",
                "source": copy.deepcopy(origin["manifest"]),
                "status": "decoded",
                "decoded": decoded,
            }
        ],
    )
    return data, context


def test_reference_counts_duplicates_and_causal_availability():
    data, context = fixture()
    assert (
        domain.validate_data(data, context, require_observed=True)["reference_entries"]
        == 2
    )
    assert domain.available_sessions(data, -1, 1) == []
    assert domain.available_sessions(data, 0, 0) == []
    assert domain.available_sessions(data, 0, 1) == data["sessions"]


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("source_sha256",), "c" * 64),
        (("codec_reference",), "guessed"),
        (("comparison_allowed",), True),
        (("training_allowed",), True),
        (("product_supported",), True),
        (("native_format_version",), 10896),
        (("exact_build_asset_binding",), "exact"),
        (("sessions", 0, "sequence"), 2),
        (("sessions", 0, "tick"), 1),
        (("sessions", 0, "source", "sha256"), "b" * 64),
        (("sessions", 0, "source_path"), "other"),
        (("sessions", 0, "status"), "absent"),
        (("sessions", 0, "status"), "read_limit_exceeded"),
        (("sessions", 0, "decoded"), None),
        (("sessions", 0, "decoded", "outer_padding_bits"), 6),
        (("sessions", 0, "decoded", "inner_padding_bits"), 0),
        (("sessions", 0, "decoded", "encoded_bytes"), 41),
        (("sessions", 0, "decoded", "entries", 0, "ordinal"), 1),
        (("sessions", 0, "decoded", "entries", 0, "type_index"), 1),
        (("sessions", 0, "decoded", "entries", 0, "directory_index"), 1),
        (("sessions", 0, "decoded", "entries", 0, "asset_content_sha256"), "d" * 64),
        (("sessions", 0, "decoded", "raw"), "private"),
    ],
)
def test_rejects_forged_or_inconsistent_evidence(path, value):
    data, context = fixture()
    target = data
    for key in path[:-1]:
        target = target[key]
    target[path[-1]] = value
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, context)


@pytest.mark.parametrize(
    "status", ["absent", "malformed_or_unsupported", "read_limit_exceeded"]
)
def test_absence_and_abstentions_never_leak_partial_values(status):
    data, context = fixture()
    row = data["sessions"][0]
    row["status"], row["decoded"] = status, None
    if status == "absent":
        row["source"] = {"present": False, "bytes": None, "sha256": None}
    if status == "read_limit_exceeded":
        row["source"]["bytes"] = 1048577
    context["observations"]["sessions"][0]["manifest"] = copy.deepcopy(row["source"])
    assert domain.validate_data(data, context)["decoded_manifests"] == 0
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, context, require_observed=True)


def test_loader_binds_artifact_and_source_chain(tmp_path):
    data, _ = fixture()
    source = write_context(tmp_path)
    context, _, _, _ = domain.rule_context.load_candidate(source)
    data["sessions"][0].update(
        source=context["observations"]["sessions"][0]["manifest"],
        status="absent",
        decoded=None,
    )
    raw = source.read_bytes()
    data["source_context"] = {
        "path": source.name,
        "bytes": len(raw),
        "sha256": sha(raw),
    }
    path = tmp_path / domain.DATA
    path.write_text(json.dumps(data), encoding="utf-8")
    assert domain.load_candidate(path)[2]["reference_entries"] == 0
    data["source_context"]["path"] = "../outside.json"
    path.write_text(json.dumps(data), encoding="utf-8")
    with pytest.raises(RuntimeContractError):
        domain.load_candidate(path)
