from __future__ import annotations

import copy
import json

import pytest

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import entry_identity as domain
from ai_coach.tests.test_economy_ownership import fixture as ownership_fixture


def scalar(value, native_type=None):
    if value is None:
        return {"status": "absent", "native_type": None, "value": None}
    return {
        "status": "observed",
        "native_type": native_type or ("bool" if type(value) is bool else "int64"),
        "value": value,
    }


def fixture():
    own, identity, candidate = ownership_fixture()
    item = copy.deepcopy(own["observations"]["states"][0]["pawns"][0]["slots"][0])
    entry = {
        "index": 0,
        "fields": {
            k: scalar(v)
            for k, v in zip(domain.MEMBERS, (7, 0, -1, False, item["handle"]))
        },
        "item": item,
        "entity_definition": scalar(7),
    }
    states = [
        {
            "sequence": n,
            "first_cut": n,
            "ownership_state_sequence": n,
            "pawns": [
                {
                    "entity_alias": p["entity_alias"],
                    "status": "observed",
                    "length": 1 if i == 0 else 0,
                    "entries": [copy.deepcopy(entry)] if i == 0 else [],
                }
                for i, p in enumerate(s["pawns"])
            ],
        }
        for n, s in enumerate(own["observations"]["states"], 1)
    ]
    data = {
        k: v
        for k, v in own.items()
        if k not in ("schema_id", "source_identity", "observations", "cut_contract")
    }
    data.update(
        schema_id="stratai.economy_entry_observations@1",
        source_ownership={
            "path": domain.ownership.DATA,
            "bytes": 1,
            "sha256": "b" * 64,
        },
        cut_contract="fresh_sellback_read_in_same_ownership_FrameDone_no_parser_advance",
        source_property="m_pBuyServices.m_vecSellbackPurchaseEntries",
        observations={
            "cuts": [{"ownership_cut": n, "state_sequence": n} for n in (1, 2)],
            "states": states,
        },
    )
    return data, own, identity, candidate


def first(data):
    return data["observations"]["states"][0]["pawns"][0]["entries"][0]


def report(args):
    return domain.build_report(
        *args,
        {"path": domain.DATA, "bytes": 1, "sha256": "c" * 64},
        require_observed=True,
    )


def status(args):
    return report(args)["states"][0]["records"][0]["status"]


def test_exact_common_cut_link_and_later_owner_disagreement_are_not_transactions():
    args = fixture()
    before = copy.deepcopy(args)
    result = report(args)
    assert result["summary"]["status_counts"]["exact_link"] == 1
    assert result["summary"]["status_counts"]["ownership_disagreement"] == 1
    assert result["summary"]["exact_unique_objects"] == 1
    assert result["states"][0]["records"][0]["recorded_cost"] == 0
    assert result["summary"]["verified_itemized_transactions"] == 0
    assert args == before
    result["states"][0]["records"][0]["cause"] = "purchase"
    with pytest.raises(RuntimeContractError):
        domain.validate_report(
            result, *args, {"path": domain.DATA, "bytes": 1, "sha256": "c" * 64}
        )


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("comparison_allowed",), True),
        (("training_allowed",), True),
        (("product_supported",), True),
        (("source_sha256",), "d" * 64),
        (("observations", "cuts", 0, "ownership_cut"), 2),
        (("observations", "cuts", 0, "state_sequence"), 2),
        (("observations", "states", 0, "first_cut"), 2),
        (("observations", "states", 0, "ownership_state_sequence"), 2),
        (("observations", "states", 0, "pawns", 0, "entries", 0, "index"), 1),
        (("observations", "states", 0, "pawns", 0, "length"), 2),
        (
            ("observations", "states", 0, "pawns", 0, "entries", 0, "item", "handle"),
            (20001 << 14) | 5,
        ),
        (
            (
                "observations",
                "states",
                0,
                "pawns",
                0,
                "entries",
                0,
                "fields",
                "m_nCost",
                "native_type",
            ),
            "bool",
        ),
    ],
)
def test_rejects_forged_cuts_presence_provenance_and_series(path, value):
    args = fixture()
    target = args[0]
    for k in path[:-1]:
        target = target[k]
    target[path[-1]] = value
    with pytest.raises(RuntimeContractError):
        report(args)


@pytest.mark.parametrize(
    ("handle", "ref_status", "expected"),
    [
        (0, "entity_unavailable", "reference_unresolved"),
        ((20001 << 14) | 5, "serial_mismatch", "reference_unresolved"),
        ((2 << 14) | 1, "kind_mismatch", "reference_unresolved"),
        (16777215, "invalid_sentinel", "special_entry_without_entity"),
        (4294967295, "invalid_sentinel", "special_entry_without_entity"),
    ],
)
def test_full_handle_and_special_sentinels_are_explicit(handle, ref_status, expected):
    args = fixture()
    e = first(args[0])
    e["fields"]["m_hItem"] = scalar(handle, "uint32")
    e["item"] = {"handle": handle, "status": ref_status, "target_alias": None}
    e["entity_definition"] = scalar(None)
    assert status(args) == expected
    assert first(args[0])["fields"]["m_hItem"]["value"] == handle


@pytest.mark.parametrize("armor", [-1, 4294967295, 0, 100])
def test_armor_metadata_is_preserved_without_physical_interpretation(armor):
    args = fixture()
    first(args[0])["fields"]["m_nPrevArmor"] = scalar(
        armor, "uint32" if armor >= 0 else "int32"
    )
    assert status(args) == "exact_link"
    assert first(args[0])["fields"]["m_nPrevArmor"]["value"] == armor


def test_late_actor_is_not_backfilled():
    args = fixture()
    args[1]["observations"]["states"][0]["pawns"][0]["actor_alias"] = None
    assert status(args) == "actor_unavailable"


@pytest.mark.parametrize("cross_pawn", [False, True])
def test_duplicate_entries_or_claims_block_exclusive_link(cross_pawn):
    args = fixture()
    pawns = args[0]["observations"]["states"][0]["pawns"]
    e = copy.deepcopy(first(args[0]))
    if cross_pawn:
        pawns[1].update(length=1, entries=[e])
    else:
        e["index"] = 1
        pawns[0]["entries"].append(e)
        pawns[0]["length"] = 2
    assert status(args) == "identity_ambiguous"


@pytest.mark.parametrize("which", ["cost", "other_list", "handle"])
def test_partial_or_missing_list_and_missing_scalar_do_not_become_complete(which):
    args = fixture()
    pawns = args[0]["observations"]["states"][0]["pawns"]
    if which == "other_list":
        pawns[1].update(status="absent", length=None)
    else:
        pawns[0]["status"] = "partial"
        e = first(args[0])
        e["fields"]["m_nCost" if which == "cost" else "m_hItem"] = scalar(None)
        if which == "handle":
            e["item"] = {"handle": None, "status": "absent", "target_alias": None}
            e["entity_definition"] = scalar(None)
    assert status(args) == "partial_list"


@pytest.mark.parametrize(
    ("field", "value", "expected"),
    [
        ("entity_definition", None, "definition_unavailable"),
        ("entity_definition", 9, "definition_disagreement"),
        ("m_nCost", -1, "invalid_member_value"),
        ("m_bPrevHelmet", 2, "invalid_member_value"),
    ],
)
def test_definition_and_member_disagreements_abstain(field, value, expected):
    args = fixture()
    e = first(args[0])
    if field == "entity_definition":
        e[field] = scalar(value)
    else:
        e["fields"][field] = scalar(value)
    assert status(args) == expected


def test_two_cuts_in_same_tick_keep_distinct_cost_and_owner():
    args = fixture()
    args[1]["observations"]["cuts"][1]["demo_tick"] = 1
    args[2]["observations"]["clocks"][1]["demo_tick"] = 1
    args[0]["observations"]["states"][1]["pawns"][0]["entries"][0]["fields"][
        "m_nCost"
    ] = scalar(300)
    result = report(args)
    assert [s["demo_tick"] for s in result["states"]] == [1, 1]
    assert [s["records"][0]["recorded_cost"] for s in result["states"]] == [0, 300]
    assert result["states"][1]["records"][0]["status"] == "ownership_disagreement"


def test_future_entity_cannot_resolve_earlier_entry():
    args = fixture()
    rows = args[2]["observations"]["rows"]
    # Keep ownership self-consistent at the earlier cut with no such equipment.
    rows[-1]["frame"], rows[-1]["sequence"] = 2, 6
    args[1]["observations"]["cuts"][0]["identity_sequence"] = 3
    args[1]["observations"]["cuts"][1]["identity_sequence"] = 6
    s = args[1]["observations"]["states"][0]
    s["equipment"] = []
    s["pawns"][0].update(slots=[], inventory_length=0)
    with pytest.raises(RuntimeContractError, match="serial or lifetime"):
        report(args)


def test_load_rejects_changed_ownership_hash(tmp_path, monkeypatch):
    data, own, identity, candidate = fixture()
    source = tmp_path / domain.DATA
    source.write_text(json.dumps(data))
    (tmp_path / domain.ownership.DATA).write_text("{}")
    monkeypatch.setattr(
        domain.ownership,
        "load_source",
        lambda path: (
            own,
            identity,
            candidate,
            {"path": domain.ownership.DATA, "bytes": 2, "sha256": "d" * 64},
        ),
    )
    with pytest.raises(RuntimeContractError, match="ownership hash"):
        domain.load_source(source)


def test_output_preflight_happens_before_source_read(tmp_path, monkeypatch):
    (tmp_path / "sentinel").write_text("preserve")
    monkeypatch.setattr(
        domain, "load_source", lambda path: pytest.fail("source read before preflight")
    )
    with pytest.raises((ValueError, RuntimeContractError)):
        domain.materialize(tmp_path / "missing", tmp_path)


def test_ambiguous_actor_binding_blocks_link():
    args = fixture()
    pawns = args[1]["observations"]["states"][0]["pawns"]
    pawns[1]["actor_alias"] = pawns[0]["actor_alias"]
    assert status(args) == "identity_ambiguous"


def test_owner_without_inventory_does_not_accredit_holding():
    args = fixture()
    args[1]["observations"]["states"][0]["pawns"][0].update(
        slots=[], inventory_length=0
    )
    assert status(args) == "ownership_unaccredited"


def test_uninterpretable_other_handle_blocks_exclusivity():
    args = fixture()
    p = args[0]["observations"]["states"][0]["pawns"][1]
    e = copy.deepcopy(first(args[0]))
    e["fields"]["m_hItem"] = scalar(True)
    e["item"] = {"handle": None, "status": "unsupported_type", "target_alias": None}
    e["entity_definition"] = scalar(None)
    p.update(length=1, entries=[e])
    assert status(args) == "partial_list"


def test_unsupported_scalar_keeps_explicit_type_and_abstains():
    args = fixture()
    p = args[0]["observations"]["states"][0]["pawns"][0]
    p["status"] = "partial"
    first(args[0])["fields"]["m_nCost"] = {
        "status": "unsupported_type",
        "native_type": "float32",
        "value": None,
    }
    assert status(args) == "partial_list"


def test_entry_state_cannot_hide_a_later_ownership_state():
    args = fixture()
    args[0]["observations"]["cuts"][1]["state_sequence"] = 1
    with pytest.raises(RuntimeContractError, match="delivery states"):
        report(args)


def test_equal_fresh_payload_can_reuse_storage_but_retains_all_cuts():
    args = fixture()
    args[1]["observations"]["states"] = args[1]["observations"]["states"][:1]
    args[1]["observations"]["cuts"][1]["state_sequence"] = 1
    args[0]["observations"]["states"] = args[0]["observations"]["states"][:1]
    args[0]["observations"]["cuts"][1]["state_sequence"] = 1
    result = report(args)
    assert result["summary"]["cuts"] == 2
    assert result["summary"]["entry_state_records"] == 1
