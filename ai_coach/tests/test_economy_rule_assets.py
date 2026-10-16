from __future__ import annotations

import copy
import json
from pathlib import Path

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import rule_assessment as domain
from ai_coach.economy import rule_assets as assets
from ai_coach.economy.candidate_io import sha
from ai_coach.economy.rule_asset_text import economic_defaults, vdata
from ai_coach.tests.test_economy_native_sources import bindings, native_fixture
from ai_coach.tests.test_economy_opportunities import source_fixture


def source_set(tmp_path):
    directory = tmp_path / "sources"
    directory.mkdir()
    refs = []
    for revision, client in [("a" * 40, "10001"), ("b" * 40, "10002")]:
        texts = {
            "version": f"ClientVersion={client}\nServerVersion={client}\nPatchVersion=1.40.0.0\nProductName=cs2\nappID=730\nServerAppID=2347773\nSourceRevision={client}\nVersionDate=Jan 01 2025\nVersionTime=00:00:00\n",
            "weapons": '{ "45" = { m_szName = "weapon_smokegrenade" m_nPrice = 300 m_nKillAward = 300 } "50" = { m_szName = "item_kevlar" m_nPrice = 650 } }',
            "competitive": "mp_buytime 20\nmp_freezetime 15\ncash_team_loser_bonus 1400\n",
        }
        for kind, text in texts.items():
            payload = text.encode()
            filename = f"{revision}-{kind}.txt"
            (directory / filename).write_bytes(payload)
            source_path = assets.SOURCE_PATHS[kind]
            refs.append(
                {
                    "revision": revision,
                    "kind": kind,
                    "repository_path": source_path,
                    "url": f"https://raw.githubusercontent.com/SteamTracking/GameTracking-CS2/{revision}/{source_path}",
                    "file": filename,
                    "bytes": len(payload),
                    "sha256": sha(payload),
                }
            )
    registry = {
        "schema_id": "stratai.economy_rule_source_registry@1",
        "contract_version": assets.VERSION,
        "repository": "https://github.com/SteamTracking/GameTracking-CS2",
        "status": "historical_reference_only",
        **assets.GATES,
        "sources": refs,
    }
    registry_path = tmp_path / "registry.json"
    registry_path.write_text(json.dumps(registry))
    loaded, digest, docs, _ = assets.load_sources(directory, registry_path)
    package = assets.build_package(loaded, digest, docs)
    return directory, registry_path, loaded, digest, docs, package


def report_fixture(tmp_path):
    _, _, registry, digest, docs, package = source_set(tmp_path)
    assets.validate_package(package, registry, digest, docs)
    source = source_fixture()
    data = native_fixture(source)
    bound = bindings(data)
    package_hash = sha(json.dumps(package).encode())
    report = domain.build_report(data, source, package, package_hash, bound)
    return report, data, source, package, package_hash, bound


def test_round_trip_and_independent_validator(tmp_path, monkeypatch):
    directory, path, registry, digest, docs, package = source_set(tmp_path)
    result = assets.validate_package(package, registry, digest, docs)
    assert result["source_values_rechecked"] == 12
    monkeypatch.setattr(
        assets, "build_package", lambda *a: pytest.fail("validator called constructor")
    )
    assets.validate_package(package, registry, digest, docs)
    package_path = tmp_path / assets.PACKAGE
    package_path.write_text(json.dumps(package))
    assert assets.load_package(package_path, directory, path)[0] == package


@pytest.mark.parametrize(
    "field", ["price", "pointer", "default", "version", "coverage", "gate"]
)
def test_package_forgery_rejected(tmp_path, field):
    _, _, registry, digest, docs, p = source_set(tmp_path)
    if field == "price":
        p["releases"][0]["items"][0]["price"]["value"] = 301
    elif field == "pointer":
        p["releases"][0]["items"][0]["price"]["pointer"] = "/50/m_nPrice"
    elif field == "default":
        p["releases"][0]["defaults"][0]["value"] = "99"
    elif field == "version":
        p["releases"][0]["version"]["ClientVersion"] = "99999"
    elif field == "coverage":
        p["releases"][0]["items"].pop()
    else:
        p["comparison_allowed"] = True
    with pytest.raises(RuntimeContractError):
        assets.validate_package(p, registry, digest, docs)


@pytest.mark.parametrize("field", ["bytes", "url", "kind", "duplicate", "missing"])
def test_source_integrity_and_completeness(tmp_path, field):
    directory, path, registry, _, _, _ = source_set(tmp_path)
    if field == "bytes":
        (directory / registry["sources"][0]["file"]).write_bytes(b"bad")
    elif field == "url":
        registry["sources"][0]["url"] = registry["sources"][0]["url"].replace(
            "a" * 40, "c" * 40
        )
    elif field == "kind":
        registry["sources"][0]["repository_path"] = "game/csgo/another.inf"
    elif field == "duplicate":
        registry["sources"].append(copy.deepcopy(registry["sources"][0]))
    else:
        registry["sources"].pop()
    path.write_text(json.dumps(registry))
    with pytest.raises(RuntimeContractError):
        assets.load_sources(directory, path)


@pytest.mark.parametrize(
    "text",
    [
        '{"45" = {m_szName="x" m_nPrice=true}}',
        '{"45" = {m_szName="x" m_nPrice=-1}}',
        '{"45" = {m_szName="x" m_nPrice=3 m_nPrice=4}}',
        '{"45" = {m_szName="x"} "045" = {m_szName="x"}}',
    ],
)
def test_bad_price_definitions_rejected(text):
    with pytest.raises(RuntimeContractError):
        assets.item_facts(text)


def test_native_zero_missing_nested_and_typed_strings_are_distinct():
    rows = assets.item_facts(
        '{"45" = {m_szName="x" m_nPrice=0 nested={m_nKillAward=900} note=resource:"a{b}.vdata"}}'
    )
    assert rows[0]["price"]["value"] == 0
    assert rows[0]["kill_award"]["value"] is None
    assert rows[0]["kill_award"]["status"] == "unavailable"


@pytest.mark.parametrize(
    "text", ["{ x = 1 } extra", "{ x = [1, 2 }", "{ x = 1 x = 2 }", "{ x = 1 @ }"]
)
def test_broken_vdata_fails_closed(text):
    with pytest.raises(RuntimeContractError):
        vdata(text)


def test_incomplete_sources_are_counted_without_imputing_price_checks(tmp_path):
    _, data, source, package, digest, bound = report_fixture(tmp_path)
    data["observations"]["snapshots"][1]["sellback"]["status"] = "partial"
    data["observations"]["snapshots"][3]["connected"] = False
    report = domain.build_report(data, source, package, digest, bound)
    domain.validate_report(report, data, source, package, digest, bound)
    assert report["source_coverage"] == {
        "snapshots": 4,
        "connected_complete_lists": 2,
        "disconnected_snapshots": 1,
        "connected_incomplete_lists": 1,
    }
    assert report["price_checks"] == []


def test_later_cost_cannot_change_earlier_comparison_or_initial_configuration(tmp_path):
    report, data, source, package, digest, bound = report_fixture(tmp_path)
    data["observations"]["snapshots"][-1]["sellback"]["entries"][0]["m_nCost"][
        "value"
    ] = 999
    changed = domain.build_report(data, source, package, digest, bound)
    domain.validate_report(changed, data, source, package, digest, bound)
    assert changed["price_checks"][:-1] == report["price_checks"][:-1]
    assert changed["configuration_checks"] == report["configuration_checks"]
    assert changed["price_checks"][-1]["result"] == "different_reference"
    assert changed["price_checks"][-1]["player_error"] is False


def test_cfg_defaults_do_not_execute_includes_or_accept_duplicate_rules():
    assert economic_defaults(
        "// cash_team_loser_bonus 999\nexec server.cfg\nmp_startmoney 0 // source default\n"
    ) == [{"name": "mp_startmoney", "value": "0", "line": 3}]
    with pytest.raises(RuntimeContractError):
        economic_defaults("mp_startmoney 800\nmp_startmoney 999")


def test_same_patch_two_builds_never_approve_rules(tmp_path):
    report, *inputs = report_fixture(tmp_path)
    validation = domain.validate_report(report, *inputs)
    assert len(report["identity"]["candidate_revisions"]) == 2
    assert report["identity"]["build_binding"] == "unverified"
    assert not report["comparison_allowed"]
    assert validation["summary"]["equal_reference_prices"] == 2
    assert all(
        not row["default_applied_to_demo"] for row in report["configuration_checks"]
    )


@pytest.mark.parametrize("mode", ["missing", "mismatch", "conflict"])
def test_missing_or_conflicting_version_never_uses_price_reference(tmp_path, mode):
    _, data, source, package, digest, bound = report_fixture(tmp_path)
    if mode == "missing":
        data["observations"]["version_observations"] = []
    elif mode == "mismatch":
        data["observations"]["version_observations"][0]["fields"]["patch_version"][
            "value"
        ] = 99999
    else:
        extra = copy.deepcopy(data["observations"]["version_observations"][0])
        extra["fields"]["patch_version"]["value"] = 99999
        data["observations"]["version_observations"].append(extra)
    report = domain.build_report(data, source, package, digest, bound)
    domain.validate_report(report, data, source, package, digest, bound)
    assert not report["identity"]["candidate_revisions"]
    assert all(row["reference_price"] is None for row in report["price_checks"])


def test_disagreement_between_assets_is_not_resolved_by_observed_cost(tmp_path):
    _, data, source, package, digest, bound = report_fixture(tmp_path)
    package["releases"][0]["items"][0]["price"]["value"] = 200
    report = domain.build_report(data, source, package, digest, bound)
    domain.validate_report(report, data, source, package, digest, bound)
    assert report["price_checks"][0]["result"] == "reference_conflicting"


@pytest.mark.parametrize(
    "field",
    ["price", "result", "tick", "identity", "default", "source", "coverage", "gate"],
)
def test_forged_report_rejected(tmp_path, field):
    report, *inputs = report_fixture(tmp_path)
    if field == "price":
        report["price_checks"][0]["reference_price"] = 800
    elif field == "result":
        report["price_checks"][0]["result"] = "different_reference"
    elif field == "tick":
        report["price_checks"][0]["tick"] += 1
    elif field == "identity":
        report["identity"]["candidate_revisions"].pop()
    elif field == "default":
        report["configuration_checks"][0]["reference_default"] = "1"
    elif field == "source":
        report["source_data_sha256"] = "f" * 64
    elif field == "coverage":
        report["configuration_checks"].pop()
    else:
        report["price_checks"][0]["effective_shop_price"] = 300
    with pytest.raises(RuntimeContractError):
        domain.validate_report(report, *inputs)


def test_future_and_uncaptured_rules_are_not_filled_with_defaults(
    tmp_path, monkeypatch
):
    _, data, source, package, digest, bound = report_fixture(tmp_path)
    r = source["rounds"][0]
    r["initial_state"]["context"]["mp_buytime"]["availability_tick"] = (
        r["start_tick"] + 1
    )
    report = domain.build_report(data, source, package, digest, bound)
    monkeypatch.setattr(
        domain, "build_report", lambda *a: pytest.fail("validator called builder")
    )
    domain.validate_report(report, data, source, package, digest, bound)
    buy = next(x for x in report["configuration_checks"] if x["name"] == "mp_buytime")
    bonus = next(
        x
        for x in report["configuration_checks"]
        if x["name"] == "cash_team_loser_bonus"
    )
    assert (
        buy["native_status"] == "not_available_at_round_start"
        and buy["native_value"] is None
    )
    assert (
        bonus["native_status"] == "not_captured_by_source_contract"
        and bonus["native_value"] is None
    )


@pytest.mark.parametrize("module", [assets, domain])
def test_unsafe_output_rejected_before_reading_inputs(tmp_path, monkeypatch, module):
    output = tmp_path / "occupied"
    output.mkdir()
    (output / "keep").write_text("user content")
    if module is assets:
        monkeypatch.setattr(
            assets,
            "load_sources",
            lambda *a: pytest.fail("read input before preflight"),
        )
        with pytest.raises(RuntimeContractError):
            assets.materialize(Path("missing"), output)
    else:
        monkeypatch.setattr(
            domain.native_sources,
            "load_candidate",
            lambda *a, **kw: pytest.fail("read input before preflight"),
        )
        with pytest.raises(RuntimeContractError):
            domain.materialize(
                Path("missing"), Path("missing"), Path("missing"), output
            )
    assert (output / "keep").read_text() == "user content"
