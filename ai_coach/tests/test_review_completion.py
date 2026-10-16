"""Review facts remain causal; a possible destination never certifies a plan."""

import json
from copy import deepcopy

import pytest
from jsonschema import ValidationError

from ai_coach.interpretation import (
    context,
    evaluation,
    inventory,
    review,
    review_packet,
)
from ai_coach.tests.test_contextual_interpretation import (
    episode,
    infer,
)
from ai_coach.tests.test_contextual_interpretation import (
    inputs as tactical_inputs,
)
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, REF
from ai_coach.tests.test_support_objective import scalar
from ai_coach.tests.test_temporal_description import inputs


def grenade_inputs(n=5):
    source, sources = inputs(n)
    own = sources["c4_source"]
    for state in own["observations"]["states"]:
        state["entities"][0].update(
            class_name="CSmokeGrenade", definition=scalar(45, "uint64")
        )
    amounts = {
        **{
            k: own[k]
            for k in (
                "source_alias",
                "source_sha256",
                "build_identifier",
                "contract_version",
                "comparison_allowed",
                "training_allowed",
                "product_supported",
            )
        },
        "schema_id": "stratai.support_inventory_observations@1",
        "c4_source": REF,
        "capture_contract": "same_support_FrameDone_native_reserve@1",
        "observations": {"cuts": [], "states": []},
    }
    for cut, state in zip(
        own["observations"]["cuts"], own["observations"]["states"], strict=True
    ):
        amounts["observations"]["cuts"].append(
            {
                **{
                    k: cut[k]
                    for k in (
                        "source_sequence",
                        "frame",
                        "tick",
                        "instance",
                        "epoch",
                        "state_sequence",
                    )
                },
                "c4_state_sequence": cut["state_sequence"],
            }
        )
        amounts["observations"]["states"].append(
            {
                "sequence": state["sequence"],
                "first_cut": state["first_cut"],
                "items": [
                    {
                        **{
                            k: state["entities"][0][k]
                            for k in ("entity_life", "handle")
                        },
                        "fields": {
                            inventory.RESERVE[0]: scalar(0, "int32"),
                            inventory.RESERVE[1]: {
                                "status": "absent",
                                "value": None,
                                "native_type": None,
                            },
                        },
                    }
                ],
            }
        )
    return source, sources, amounts


def ctx(source, sources, end=3):
    return context.from_sources(
        source, side="CT", start_sequence=1, end_sequence=end, **sources
    )


def test_current_inventory_quantities_zero_reserve_and_missing_are_distinct():
    source, sources, quantities = grenade_inputs()
    inventory.validate_source(quantities, sources["c4_source"])
    quantities["observations"]["states"][0]["items"][0]["fields"][
        inventory.RESERVE[0]
    ] = scalar(1, "int32")
    quantities["observations"]["states"][2]["items"][0]["fields"][
        inventory.RESERVE[0]
    ] = {"status": "absent", "value": None, "native_type": None}
    value = ctx(source, sources)
    before = deepcopy((value, sources, quantities))
    rows = inventory.sequence(value, sources["c4_source"], quantities)
    assert [r[ACTOR]["items"][0]["count"] for r in rows] == [2, 1, None]
    assert rows[2][ACTOR]["items"][0]["minimum_count"] == 1
    assert rows[2][ACTOR]["complete"] and not rows[2][ACTOR]["quantity_complete"]
    assert rows[2][ALLY]["items"] == [] and rows[2][ALLY]["quantity_complete"]
    assert (value, sources, quantities) == before


@pytest.mark.parametrize(
    "value,native", [(-1, "int32"), (True, "bool"), (1.5, "float32"), (0, "uint64")]
)
def test_invalid_primary_quantity_never_becomes_zero_or_falls_back(value, native):
    item = {
        "fields": {
            inventory.RESERVE[0]: scalar(value, native),
            inventory.RESERVE[1]: scalar(0, "int32"),
        }
    }
    assert inventory.quantity(item) == (None, "native_quantity_invalid")
    item["fields"][inventory.RESERVE[0]] = {
        "status": "absent",
        "value": None,
        "native_type": None,
    }
    assert inventory.quantity(item) == (1, inventory.RESERVE[1])


@pytest.mark.parametrize(
    "mutation", ["owner", "serial", "unknown_definition", "missing_slot", "duplicate"]
)
def test_ownership_gaps_do_not_mean_no_grenades(mutation):
    source, sources, quantities = grenade_inputs()
    value = ctx(source, sources)
    state = sources["c4_source"]["observations"]["states"][2]
    if mutation == "owner":
        state["entities"][0]["owner"]["target_life"] = "support-pawn-000099"
    elif mutation == "serial":
        state["entities"][0]["handle"] += 16384
    elif mutation == "unknown_definition":
        state["entities"][0]["definition"] = {
            "status": "absent",
            "value": None,
            "native_type": None,
        }
    elif mutation == "missing_slot":
        state["players"][0]["slots"][0]["status"] = "absent"
    else:
        state["players"][0]["slots"] *= 2
        state["players"][0]["inventory_length"] = 2
    row = inventory.sequence(value, sources["c4_source"], quantities)[-1][ACTOR]
    assert row["status"] == "unavailable" and not row["complete"]
    assert row["last_known"]["source_sequence"] == 2


@pytest.mark.parametrize("boundary", ["gap", "life", "dead", "round"])
def test_historical_inventory_never_crosses_boundary(boundary):
    source, sources, quantities = grenade_inputs()
    value = ctx(source, sources)
    state = sources["c4_source"]["observations"]["states"][2]
    state["players"][0].update(
        inventory_status="absent", inventory_length=None, slots=[]
    )
    last = value["samples"][-1]
    if boundary == "gap":
        last["continuity"] = "unavailable"
    elif boundary == "life":
        last["actors"][0]["identity"]["pawn_life"] = "support-pawn-000099"
    elif boundary == "dead":
        last["actors"][0]["alive"]["value"] = False
        last["actors"][0]["exclusion"] = "not_alive"
    else:
        last["cut"]["round_number"] += 1
    row = inventory.sequence(value, sources["c4_source"], quantities)[-1][ACTOR]
    assert row["last_known"] is None
    if boundary == "dead":
        assert row["status"] == "not_applicable"


def test_partial_inventory_can_prove_presence_but_not_absence():
    source, sources, quantities = grenade_inputs()
    value = ctx(source, sources)
    sources["c4_source"]["observations"]["states"][2]["players"][0][
        "inventory_length"
    ] = 2
    row = inventory.sequence(value, sources["c4_source"], quantities)[-1][ACTOR]
    assert row["status"] == "partial" and row["items"][0]["count"] == 1
    assert not row["complete"] and not row["quantity_complete"]


@pytest.mark.parametrize("mutation", ["clock", "future", "life", "unreferenced"])
def test_inventory_source_provenance_and_causal_frontier(mutation):
    _, sources, quantities = grenade_inputs()
    if mutation == "clock":
        quantities["observations"]["cuts"][0]["frame"] += 1
    elif mutation == "future":
        quantities["observations"]["states"][0]["first_cut"] = 2
    elif mutation == "life":
        quantities["observations"]["states"][0]["items"][0]["handle"] += 16384
    else:
        quantities["observations"]["states"].append(
            {"sequence": 6, "first_cut": 6, "items": []}
        )
    with pytest.raises(ValueError):
        inventory.validate_source(quantities, sources["c4_source"])


def test_packet_is_neutral_causal_and_renders_counts_without_inventing_inventory():
    source, sources, quantities = grenade_inputs()
    value = ctx(source, sources)
    packet = review_packet.build(value, sources, quantities, "de_mirage")
    html = review_packet.render(packet)
    display = json.loads(
        html.split('<script id="data" type="application/json">')[1].split("</script>")[
            0
        ]
    )
    assert not display["has_radar"] and len(display["frames"]) == 3
    assert [a["label"] for a in display["frames"][-1]["actors"]] == ["CT1", "CT2"]
    assert display["frames"][-1]["actors"][0]["inventory"]["items"][0]["count"] == 1
    expected = deepcopy(packet)
    sources["objective_source"]["observations"]["frames"] = sources["objective_source"][
        "observations"
    ]["frames"][:3]
    sources["c4_source"]["observations"]["states"][3:] = [None]
    sources["c4_source"]["observations"]["cuts"][3:] = [None]
    quantities["observations"]["cuts"][3:] = [None]
    quantities["observations"]["states"][3:] = [None]
    sources["c4_source"]["observations"]["states"][2]["players"][-1]["slots"] = [
        "enemy poison"
    ]
    sources["expert_answer"] = "B"
    assert review_packet.build(value, sources, quantities, "de_mirage") == expected
    old = review_packet.build(value, sources, None, "de_mirage")
    assert old["inventory"][-1][ACTOR]["items"][0]["count"] is None


def comparison_fixture():
    source, sources = tactical_inputs()
    pred = infer(episode(source, sources))
    manifest = evaluation.freeze(
        [{"source_sha256": pred["source_sha256"], "side": "T", **pred["interval"]}],
        development_matches=[],
        detector_configuration=pred["configuration"],
        exposure={pred["source_sha256"]: "unknown"},
    )
    eid = manifest["cases"][0]["episode_id"]
    human = {
        "schema_id": "stratai.contextual_review@2",
        "episode_id": eid,
        "context_id": pred["context_id"],
        "reviewer": "expert",
        "literal": "No se puede definir la estrategia. B es posible.",
        "strategy": {"status": "undetermined", "compatible_plans": []},
        "destinations": [{"destination": "B", "judgement": "possible"}],
        "roles": [],
        "exposure": {
            "prediction_seen": False,
            "prior_exposure": "unknown",
            "assisted_orientation": True,
        },
        "training_allowed": False,
    }
    return manifest, {eid: pred}, human


def test_possible_destination_is_not_strategy_agreement_and_preserves_literal():
    manifest, predictions, human = comparison_fixture()
    before = deepcopy((manifest, predictions, human))
    result = review.compare(manifest, predictions, [human])
    row = result["rows"][0]
    assert row["strategy_comparison"] == "plan_not_confirmed_by_review"
    assert row["emitted_destinations"] == [{"destination": "B", "review": "possible"}]
    assert row["literal"] == human["literal"] and row["agreement"] is None
    assert result["reviewed_episodes"] == 1 and result["agreement"] is None
    assert not result["independent_generalization_accredited"]
    assert (manifest, predictions, human) == before
    assert review.compare(manifest, predictions, [])["reviewed_episodes"] == 0


@pytest.mark.parametrize(
    "mutation",
    [
        "context",
        "duplicate",
        "strategy",
        "prediction",
        "configuration",
        "unknown_field",
    ],
)
def test_review_rejects_misbound_or_relabelled_results(mutation):
    manifest, predictions, human = comparison_fixture()
    reviews = [human]
    if mutation == "context":
        human["context_id"] = "0" * 64
    elif mutation == "duplicate":
        reviews *= 2
    elif mutation == "strategy":
        human["strategy"]["compatible_plans"] = ["prepare_entry"]
    elif mutation == "prediction":
        next(iter(predictions.values()))["hypotheses"] = []
    elif mutation == "configuration":
        manifest["detector_configuration"] = "other"
    else:
        human["winner"] = "T"
    with pytest.raises((ValueError, ValidationError)):
        review.compare(manifest, predictions, reviews)


def test_independent_reviewers_count_once_per_episode():
    manifest, predictions, human = comparison_fixture()
    second = deepcopy(human)
    second["reviewer"] = "second"
    second["strategy"] = {
        "status": "interpretable",
        "compatible_plans": ["prepare_entry"],
    }
    result = review.compare(manifest, predictions, [human, second])
    assert result["reviewed_episodes"] == 1 and len(result["rows"]) == 2
    assert result["rows"][1]["strategy_comparison"] == "plan_compatible"


def test_packet_io_recomputes_view_and_facts_and_rejects_overwrite(
    tmp_path, monkeypatch
):
    source, sources, quantities = grenade_inputs()
    packet = review_packet.build(ctx(source, sources), sources, quantities, "de_mirage")
    monkeypatch.setattr(
        review_packet, "compute", lambda *args, **kwargs: ([packet], {})
    )
    manifest = review_packet.materialize(
        tmp_path, tmp_path, side="CT", intervals=[(1, 3)]
    )
    assert review_packet.validate(tmp_path, tmp_path) == manifest
    with pytest.raises(ValueError):
        review_packet.materialize(tmp_path, tmp_path, side="CT", intervals=[(1, 3)])
    path = tmp_path / "case_001.json"
    changed = json.loads(path.read_bytes())
    changed["inventory"][0][ACTOR]["items"][0]["count"] = 99
    path.write_text(json.dumps(changed), encoding="utf-8")
    with pytest.raises(ValueError, match="facts differ"):
        review_packet.validate(tmp_path, tmp_path)
    path.write_text(json.dumps(packet), encoding="utf-8")
    (tmp_path / "case_001.html").write_text("invented facts", encoding="utf-8")
    with pytest.raises(ValueError, match="view differs"):
        review_packet.validate(tmp_path, tmp_path)


def test_packet_rejects_non_temp_before_reading_sources(monkeypatch):
    def forbidden(*args, **kwargs):
        raise AssertionError("read before guard")

    monkeypatch.setattr(review_packet, "compute", forbidden)
    with pytest.raises(ValueError):
        review_packet.materialize(
            review_packet.TEMPLATE.parent,
            review_packet.TEMPLATE.parent,
            side="T",
            intervals=[(1, 3)],
        )
