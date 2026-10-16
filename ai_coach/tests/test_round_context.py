from copy import deepcopy

import pytest

from ai_coach.interpretation import review_packet_v2 as packet
from ai_coach.interpretation import round_context as r
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as clock
from ai_coach.tests.test_contextual_interpretation import inputs, episode
from ai_coach.tests.test_support_objective_clock import clock_data, scalar
from ai_coach.tests.test_support_anchors import REF


def configured(source):
    frames = source["observations"]["frames"]
    values = {
        k: "0"
        if k == "mp_give_player_c4"
        else "weapon_knife"
        if k.endswith("melee")
        else ""
        for k in r.CONVARS
    }
    return {
        "schema_id": "stratai.support_round_context_observations@1",
        "contract_version": "1.0.0-candidate.1",
        **{k: source[k] for k in ("source_alias", "source_sha256", "build_identifier")},
        "support_source": REF,
        "capture_contract": "same_support_FrameDone_public_round_configuration@1",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "observations": {
            "messages": [
                {
                    "sequence": 1,
                    "order": 1,
                    "frame": frames[0]["frame"],
                    "tick": frames[0]["tick"],
                    "session": 1,
                    "entries": [{"name": k, "value": v} for k, v in values.items()],
                }
            ],
            "frames": [
                {
                    "source_sequence": f["sequence"],
                    **{k: f[k] for k in ("frame", "tick", "instance", "epoch")},
                    "order": i + 2,
                    "message_sequence": 1,
                    "session": 1,
                    "rules": {
                        k: scalar(k != "m_bWarmupPeriod")
                        for k in ("m_bTCantBuy", "m_bCTCantBuy", "m_bWarmupPeriod")
                    },
                }
                for i, f in enumerate(frames)
            ],
        },
    }


def knife_fixture():
    source, sources = inputs()
    data = configured(source)
    sample = deepcopy(episode(source, sources)["samples"][0])
    own = sources["c4_source"]
    state = own["observations"]["states"][0]
    owner, entity = deepcopy(state["players"][0]), deepcopy(state["entities"][0])
    sample["actors"] = [deepcopy(sample["actors"][0]) for _ in range(5)]
    state["players"], state["entities"] = [], []
    for i, actor in enumerate(sample["actors"]):
        alias, life, handle = f"actor-{i}", f"pawn-{i}", 100 + i
        actor.update(
            actor_alias=alias,
            identity={"pawn_life": life, "pawn_handle": handle},
            has_c4=o.fact(False),
        )
        p, e = deepcopy(owner), deepcopy(entity)
        p.update(
            actor_alias=alias,
            pawn_life=life,
            pawn_handle=handle,
            inventory_status="observed",
            inventory_length=1,
        )
        e.update(entity_life=f"item-{i}", handle=200 + i, class_name="CKnife")
        e["owner"].update(status="resolved", target_life=life, handle=handle)
        p["slots"] = [
            {
                **deepcopy(owner["slots"][0]),
                "status": "resolved",
                "target_life": e["entity_life"],
                "handle": e["handle"],
            }
        ]
        state["players"].append(p)
        state["entities"].append(e)
    return (
        sample,
        sources["objective_source"]["observations"]["frames"][0],
        r.configurations(data)[1],
        own,
    )


@pytest.mark.parametrize("native_flag", ["0", "false"])
def test_positive_knife_configuration_excludes_without_tactical_label(native_flag):
    args = knife_fixture()
    args[2]["values"]["mp_give_player_c4"] = native_flag
    assert r.eligibility(*args)["kind"] == "knife"
    assert r.eligibility(*args)["analysis_allowed"] is False


@pytest.mark.parametrize(
    "mutation",
    [
        "no_config",
        "absent_c4_flag",
        "null_secondary",
        "missing_rule",
        "rifle",
        "partial",
        "wrong_owner",
        "duplicate",
        "unknown_alive",
    ],
)
def test_no_knife_exclusion_from_missing_c4_or_equipped_knife(mutation):
    sample, obj, conf, own = knife_fixture()
    if mutation == "no_config":
        conf = None
    elif mutation == "absent_c4_flag":
        del conf["values"]["mp_give_player_c4"]
    elif mutation == "null_secondary":
        conf["values"]["mp_t_default_secondary"] = None
    elif mutation == "missing_rule":
        conf["cut"]["rules"]["m_bTCantBuy"] = scalar()
    elif mutation == "rifle":
        own["observations"]["states"][0]["entities"][0]["class_name"] = "CAK47"
    elif mutation == "partial":
        own["observations"]["states"][0]["players"][0]["inventory_status"] = "absent"
    elif mutation == "wrong_owner":
        own["observations"]["states"][0]["entities"][0]["owner"]["handle"] = 999
    elif mutation == "duplicate":
        p = own["observations"]["states"][0]["players"][0]
        p["slots"] *= 2
        p["inventory_length"] = 2
    else:
        sample["actors"][0]["alive"] = o.fact()
    assert r.eligibility(sample, obj, conf, own)["kind"] == "unknown"


def test_bomb_or_warmup_and_conflicts_override_knife():
    args = knife_fixture()
    args[0]["actors"][0]["has_c4"] = o.fact(True)
    assert r.eligibility(*args)["kind"] == "unknown"
    assert r.eligibility(args[0], args[1], None, args[3])["kind"] == "bomb_objective"
    args[1]["rules"]["m_bWarmupPeriod"]["value"] = True
    assert r.eligibility(*args)["kind"] == "warmup"


def test_configuration_prefix_ignores_future_and_clears_new_session():
    source, _ = inputs()
    data = configured(source)
    r.validate_source(data, source)
    before = r.configurations(data)[1]
    message = deepcopy(data["observations"]["messages"][0])
    message.update(sequence=2, order=4, frame=999)
    message["entries"][0]["value"] = "1"
    data["observations"]["messages"].append(message)
    assert r.configurations(data)[1] == before
    data["observations"]["frames"][1]["session"] = 2
    assert r.configurations(data)[2]["values"] == {}
    data["observations"]["frames"][1]["message_sequence"] = 2
    with pytest.raises(ValueError):
        r.validate_source(data, source)


def test_clocks_card_causal_future_independent_and_not_duration_minus_elapsed():
    source, sources = inputs()
    clocks = clock_data(source, sources["objective_source"], sources["c4_source"])
    optional = {r.DATA: None, clock.DATA: clocks}
    value = episode(source, sources)
    result = packet.build(value, source, sources, None, "de_mirage", optional)
    assert result["analysis_scope"] == "eligible"
    assert (
        result["round_context"][0]["native_clock"]["round_remaining_seconds"]["value"]
        < 20
    )
    assert result["round_context"][0]["configured_duration_seconds"]["value"] > 100
    sources["objective_source"]["observations"]["frames"][5:] = [None]
    clocks["observations"]["frames"][5:] = [None]
    source["observations"]["frames"][5:] = [None]
    assert r.sequence(value, source, sources, optional) == result["round_context"]
    rendered = packet.render(result)
    assert "Restante de ronda:" in rendered and "Tramo: " in rendered


@pytest.mark.parametrize(
    "mutation", ["missing", "paused", "misaligned", "missing_pause"]
)
def test_clock_fallbacks_keep_unavailable_or_pause_explicit(mutation):
    source, sources = inputs()
    data = clock_data(source, sources["objective_source"], sources["c4_source"])
    for row in data["observations"]["frames"]:
        if mutation == "paused":
            row["network"]["paused"]["value"] = scalar(True, "bool")
        elif mutation == "missing_pause":
            row["network"]["paused"] = None
            row["rules"]["fields"]["m_bGamePaused"] = scalar()
        elif mutation == "misaligned":
            for player in row["players"]:
                player["fields"]["m_flSimulationTime"] = scalar(-1)
    optional = {r.DATA: None, clock.DATA: None if mutation == "missing" else data}
    facts = r.sequence(episode(source, sources), source, sources, optional)[-1][
        "native_clock"
    ]
    if mutation == "paused":
        assert facts["round_clock_state"]["value"] == "paused"
    else:
        assert facts["round_remaining_seconds"]["status"] == "unavailable"


def test_unknown_card_cannot_call_detector(monkeypatch):
    source, sources = inputs()
    value = episode(source, sources)
    for f in value["samples"]:
        for actor in f["actors"]:
            actor["has_c4"] = o.fact(False)
    # Build a valid packet first, then check the gate independent of detector internals.
    value = episode(source, sources)
    p = packet.build(
        value, source, sources, None, "de_mirage", {r.DATA: None, clock.DATA: None}
    )
    p["analysis_scope"] = "pending_context"
    monkeypatch.setattr(
        packet.temporal_detector,
        "interpret",
        lambda *a, **kw: pytest.fail("detector called"),
    )
    with pytest.raises(ValueError, match="not eligible"):
        packet.interpret(p)


def test_non_temporary_output_rejected_before_sources():
    with pytest.raises(ValueError):
        packet.materialize(None, packet.TEMPLATE.parent, side="T", intervals=[(1, 5)])


def test_c4_prefix_proof_survives_gaps_and_conflicts_until_round_changes(monkeypatch):
    source, sources = inputs()
    value = episode(source, sources)
    for sample in value["samples"][1:]:
        for actor in sample["actors"]:
            actor["has_c4"] = o.fact(False)
    optional = {r.DATA: None, clock.DATA: None}
    facts = r.sequence(value, source, sources, optional)
    assert all(f["eligibility"]["kind"] == "bomb_objective" for f in facts)
    assert all(f["eligibility"]["basis_sequence"] == 1 for f in facts)
    # Even corroborated knife inventory later cannot erase earlier C4 evidence.
    optional[r.DATA] = configured(source)
    monkeypatch.setattr(r, "knife_inventory", lambda *args: True)
    facts = r.sequence(value, source, sources, optional)
    assert all(f["eligibility"]["kind"] == "unknown" for f in facts)
    source["observations"]["frames"][-2]["epoch"] += 1
    facts = r.sequence(value, source, sources, optional)
    assert facts[-1]["eligibility"]["kind"] == "knife"


def test_gated_pipeline_omits_excluded_and_unknown_interpretations(
    monkeypatch, tmp_path
):
    source, sources = inputs()
    p = packet.build(
        episode(source, sources),
        source,
        sources,
        None,
        "de_mirage",
        {r.DATA: None, clock.DATA: None},
    )
    excluded, unknown = deepcopy(p), deepcopy(p)
    excluded["analysis_scope"], unknown["analysis_scope"] = (
        "excluded",
        "pending_context",
    )
    monkeypatch.setattr(
        packet, "compute", lambda *a, **kw: ([excluded, unknown, p], {})
    )
    calls = []
    monkeypatch.setattr(
        packet,
        "interpret",
        lambda value, **kw: calls.append(value) or {"status": "checked"},
    )
    result = packet.analyze(None, tmp_path, side="T", intervals=[(1, 5)])
    assert len(calls) == 1
    assert [r["interpretation"] for r in result["rows"]] == [
        None,
        None,
        {"status": "checked"},
    ]
