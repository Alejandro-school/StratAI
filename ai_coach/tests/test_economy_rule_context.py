from __future__ import annotations

import copy
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import rule_context as domain
from ai_coach.economy.candidate_io import sha
from ai_coach.tests.test_economy_native_sources import (
    bindings,
    fact,
    native_fixture,
    write_native,
)
from ai_coach.tests.test_economy_opportunities import source_fixture
from ai_coach.tests.test_economy_rule_assets import source_set


def context_fixture(source=None):
    source = source or source_fixture()
    spec, spec_hash = domain.probe()
    native = native_fixture(source)
    data = {
        "schema_id": "stratai.economy_rule_context@1",
        "contract_version": "1.0.0-candidate.1",
        **{k: source[k] for k in domain.IDENTITY},
        **domain.GATES,
        "probe_sha256": spec_hash,
        "source_native_manifest_sha256": "b" * 64,
        "observations": {"messages": [], "boundaries": [], "sessions": []},
    }
    data["observations"]["sessions"] = [
        {
            "sequence": 1,
            "tick": 0,
            "configuration_present": False,
            "manifest": {"present": False, "bytes": None, "sha256": None},
            "configuration_data": {"present": False, "bytes": None, "sha256": None},
        }
    ]
    data["observations"]["messages"] = [
        {
            "sequence": 2,
            "tick": 0,
            "entries": [
                {"name": "mp_buytime", "value": "10"},
                {"name": "mp_buytime", "value": "20"},
                {"name": "mp_startmoney", "value": "0"},
                {"name": "mp_maxmoney", "value": None},
            ],
        },
        {
            "sequence": 4,
            "tick": 100,
            "entries": [
                {"name": "mp_buytime", "value": "0"},
                {"name": "mp_startmoney", "value": "800"},
            ],
        },
    ]
    for sequence, tick, phase in [
        (3, 100, "round_start_probe"),
        (5, 200, "freeze_end_probe"),
    ]:
        state = {name: domain.EMPTY.copy() for name in spec["convars"]}
        for message in data["observations"]["messages"]:
            if message["sequence"] < sequence:
                for entry in message["entries"]:
                    state[entry["name"]] = {
                        "value": entry["value"],
                        "source_sequence": message["sequence"],
                        "source_tick": message["tick"],
                    }
        data["observations"]["boundaries"].append(
            {
                "sequence": sequence,
                "tick": tick,
                "instance": 1,
                "phase": phase,
                "convars": state,
                "rules": {
                    name: fact(False if boolean else 0, name, tick)
                    for name, boolean in spec["rules"].items()
                },
            }
        )
    return data, native, source


def report_fixture(tmp_path):
    data, native, source = context_fixture()
    package = source_set(tmp_path)[-1]
    package_hash = sha(json.dumps(package).encode())
    bound = {
        "context_manifest_sha256": "d" * 64,
        "context_data_sha256": "e" * 64,
        **bindings(native),
    }
    report = domain.build_report(data, native, package, package_hash, bound)
    return report, data, native, source, package, package_hash, bound


def test_native_context_roundtrip_and_validator_does_not_call_builder(
    tmp_path, monkeypatch
):
    args = report_fixture(tmp_path)
    monkeypatch.setattr(
        domain, "build_report", lambda *a: pytest.fail("constructor used by validator")
    )
    result = domain.validate_report(*args)
    assert result["summary"] == {
        "configuration_checks": 112,
        "native_property_checks": 8,
        "observed_parameters": 4,
        "observed_properties": 8,
        "approved_rule_packages": 0,
    }
    report = args[0]
    start = {
        r["name"]: r
        for r in report["configuration_checks"]
        if r["phase"] == "round_start_probe"
    }
    assert start["mp_startmoney"]["value"] == "0"
    assert start["mp_buytime"]["value"] == "20"
    assert start["mp_maxmoney"]["status"] == "native_value_absent"
    assert (
        start["cash_team_loser_bonus"]["status"] == "not_seen_in_probed_message_prefix"
    )
    assert start["cash_team_loser_bonus"]["reference_values"] == ["1400"]
    assert start["cash_team_loser_bonus"]["value"] is None
    assert report["identity"]["build_binding"] == "unverified"


@pytest.mark.parametrize(
    "mutation",
    [
        "value",
        "source_tick",
        "source_sequence",
        "sequence",
        "tick",
        "coverage",
        "name",
        "rules",
        "property_type",
        "property_time",
        "opaque",
        "probe",
        "gate",
        "identity",
    ],
)
def test_context_forgery_rejected(mutation):
    data, _, source = context_fixture()
    boundary = data["observations"]["boundaries"][0]
    if mutation in ("value", "source_tick", "source_sequence"):
        boundary["convars"]["mp_buytime"][mutation] = (
            "800" if mutation == "value" else 99
        )
    elif mutation == "sequence":
        data["observations"]["messages"][1]["sequence"] = 3
    elif mutation == "tick":
        boundary["tick"] += 1
    elif mutation == "coverage":
        data["observations"]["boundaries"].pop()
    elif mutation == "name":
        data["observations"]["messages"][0]["entries"][0]["name"] = "hostname"
    elif mutation == "rules":
        boundary["rules"].pop(next(iter(boundary["rules"])))
    elif mutation in ("property_type", "property_time"):
        target = boundary["rules"]["m_pGameRules.m_iNumConsecutiveCTLoses"]
        target["value" if mutation == "property_type" else "availability_tick"] = (
            True if mutation == "property_type" else 101
        )
    elif mutation == "opaque":
        data["observations"]["sessions"][0]["manifest"]["sha256"] = "a" * 64
    elif mutation == "probe":
        data["probe_sha256"] = "a" * 64
    elif mutation == "gate":
        data["comparison_allowed"] = True
    else:
        data["source_sha256"] = "c" * 64
    with pytest.raises(RuntimeContractError):
        domain.validate_data(data, source)


@pytest.mark.parametrize(
    "mutation",
    [
        "configuration",
        "property",
        "armor",
        "identity",
        "gate",
        "hash",
        "summary",
        "session",
    ],
)
def test_report_forgery_rejected(tmp_path, mutation):
    args = report_fixture(tmp_path)
    report = args[0]
    if mutation == "configuration":
        report["configuration_checks"][0]["value"] = "1400"
    elif mutation == "property":
        report["native_property_checks"] = []
    elif mutation == "armor":
        report["armor_snapshot_groups"][0]["snapshot_occurrences"] += 1
    elif mutation == "identity":
        report["identity"]["build_binding"] = "verified"
    elif mutation == "gate":
        report["comparison_allowed"] = True
    elif mutation == "hash":
        report["package_sha256"] = "a" * 64
    elif mutation == "summary":
        report["summary"]["observed_parameters"] += 1
    else:
        report["sessions"] = []
    with pytest.raises(RuntimeContractError):
        domain.validate_report(*args)


def test_future_same_tick_assignment_cannot_rewrite_initial_context(tmp_path):
    before, data, native, source, package, package_hash, bound = report_fixture(
        tmp_path
    )
    data = copy.deepcopy(data)
    data["observations"]["messages"][1]["entries"][0]["value"] = "99999"
    data["observations"]["boundaries"][1]["convars"]["mp_buytime"]["value"] = "99999"
    after = domain.build_report(data, native, package, package_hash, bound)
    domain.validate_report(after, data, native, source, package, package_hash, bound)
    assert before["configuration_checks"][:56] == after["configuration_checks"][:56]
    assert before["configuration_checks"][56:] != after["configuration_checks"][56:]


@pytest.mark.parametrize(
    "value,status",
    [
        ("false", "observed_native_parameter"),
        ("-30", "observed_native_parameter"),
        ("", "native_value_unsupported"),
        ("NaN", "native_value_unsupported"),
        ("Infinity", "native_value_unsupported"),
        (None, "native_value_absent"),
    ],
)
def test_native_numeric_presence_has_no_default(value, status):
    result = domain.configuration_row(
        {"instance": 1, "phase": "round_start_probe", "tick": 100, "sequence": 3},
        "mp_buytime",
        {"value": value, "source_sequence": 2, "source_tick": 0},
        ["20"],
    )
    assert result["status"] == status and result["value"] == value


def test_future_tick_remains_unusable_even_when_callback_precedes_boundary():
    result = domain.configuration_row(
        {"instance": 1, "phase": "round_start_probe", "tick": 100, "sequence": 3},
        "mp_buytime",
        {"value": "20", "source_sequence": 2, "source_tick": 101},
        ["20"],
    )
    assert (
        result["status"] == "source_tick_after_boundary"
        and result["comparison"] == "unavailable"
    )


def write_context(root):
    path = write_native(root)
    _, source, _ = domain.native_sources.load_candidate(path)
    data, _, _ = context_fixture(source)
    data["source_native_manifest_sha256"] = sha(path.read_bytes())
    raw = json.dumps(data).encode()
    (root / domain.DATA).write_bytes(raw)
    manifest = {
        "schema_id": "stratai.economy_rule_context_manifest@1",
        **{
            k: data[k]
            for k in (
                *domain.IDENTITY,
                *domain.GATES,
                "contract_version",
                "probe_sha256",
            )
        },
        "source_native": {
            "path": path.name,
            "bytes": path.stat().st_size,
            "sha256": sha(path.read_bytes()),
        },
        "artifact": {"path": domain.DATA, "bytes": len(raw), "sha256": sha(raw)},
        "summary": domain.validate_data(data, source),
    }
    target = root / domain.MANIFEST
    target.write_text(json.dumps(manifest))
    return target


@pytest.mark.parametrize("mutation", [None, "artifact", "native", "path", "probe"])
def test_load_context_checks_integrity_chain(tmp_path, mutation):
    path = write_context(tmp_path / "source")
    if mutation == "artifact":
        (path.parent / domain.DATA).write_text("{}")
    elif mutation == "native":
        (path.parent / "economy_native_sources.json").write_text("{}")
    elif mutation in ("path", "probe"):
        manifest = json.loads(path.read_text())
        if mutation == "path":
            manifest["artifact"]["path"] = "../outside.json"
        else:
            manifest["probe_sha256"] = "a" * 64
        path.write_text(json.dumps(manifest))
    if mutation is None:
        loaded = domain.load_candidate(path, require_observed=True)
        assert len(loaded[0]["observations"]["boundaries"]) == 2
    else:
        with pytest.raises(RuntimeContractError):
            domain.load_candidate(path, require_observed=True)


def test_materializer_preflight_preserves_nonempty_destination(tmp_path):
    keep = tmp_path / "keep"
    keep.write_text("preserve")
    with pytest.raises(RuntimeContractError):
        domain.materialize(
            tmp_path / "missing", tmp_path / "missing", tmp_path / "missing", tmp_path
        )
    assert keep.read_text() == "preserve"
