from copy import deepcopy

import pytest

from ai_coach.interpretation import review_packet_v3 as packet
from ai_coach.interpretation import team_status as t
from ai_coach.tests.test_contextual_interpretation import inputs, episode


def sample():
    source, sources = inputs()
    context = episode(source, sources)
    for f in source["observations"]["frames"]:
        template = f["players"][0]
        f["players"] = [
            dict(
                template,
                actor_alias=f"actor-{i}",
                side="T" if i < 5 else "CT",
                pawn_life=f"pawn-{i}",
                pawn_handle=i + 100,
                connected=True,
                identity_status="observed",
                alive_status="observed",
                alive=i in (0, 1, 5),
            )
            for i in range(10)
        ]
    return context, source


def test_current_counts_and_opposite_perspective():
    context, source = sample()
    row = t.sequence(context, source)[-1]
    assert row["allies"]["alive"] == 2 and row["opponents"]["alive"] == 1
    assert row["opponents"]["observed_dead"] == 4 and row["opponents"]["unknown"] == 0
    context["perspective"]["side"] = "CT"
    reverse = t.sequence(context, source)[-1]
    assert (
        reverse["allies"] == row["opponents"] and reverse["opponents"] == row["allies"]
    )


@pytest.mark.parametrize(
    "change",
    [
        "missing",
        "disconnected",
        "unknown_alive",
        "bad_bool",
        "bad_identity",
        "duplicate_alias",
        "duplicate_pawn",
        "duplicate_handle",
        "sentinel",
        "missing_alias",
    ],
)
def test_uncertainty_is_not_death_or_stale_state(change):
    context, source = sample()
    players = source["observations"]["frames"][4]["players"]
    p = players[5]
    if change == "missing":
        players.pop(5)
    elif change == "disconnected":
        p["connected"] = False
    elif change == "unknown_alive":
        p["alive_status"] = "unavailable"
        p["alive"] = None
    elif change == "bad_bool":
        p["alive"] = 1
    elif change == "bad_identity":
        p["identity_status"] = "unavailable"
    elif change == "duplicate_alias":
        players[6]["actor_alias"] = p["actor_alias"]
    elif change == "duplicate_pawn":
        players[6]["pawn_life"] = p["pawn_life"]
    elif change == "duplicate_handle":
        players[6]["pawn_handle"] = p["pawn_handle"]
    elif change == "sentinel":
        p["pawn_handle"] = 16383
    else:
        p["actor_alias"] = None
    final = t.sequence(context, source)[-1]["opponents"]
    assert final["status"] == "partial" and final["alive"] is None
    assert final["unknown"] >= 1 and final["observed_alive"] == 0
    assert final["observed_dead"] <= 4


def test_hidden_positions_resources_and_future_payload_have_no_effect():
    context, source = sample()
    expected = t.sequence(context, source)
    for frame in source["observations"]["frames"][:5]:
        for p in frame["players"]:
            p["position"] = object()
            p["position_status"] = object()
            p["health"] = object()
            p["weapon"] = object()
    source["observations"]["frames"][5] = {"sequence": 6, "players": object()}
    assert t.sequence(context, source) == expected
    assert "actor-" not in str(expected) and "pawn-" not in str(expected)


def test_counts_change_at_death_cut_without_future_backfill():
    context, source = sample()
    for frame in source["observations"]["frames"][:3]:
        frame["players"][6]["alive"] = True
    rows = t.sequence(context, source)
    assert [r["opponents"]["alive"] for r in rows] == [2, 2, 2, 1, 1]
    del source["observations"]["frames"][0]["players"][6]
    assert t.sequence(context, source)[0]["opponents"]["alive"] is None


def test_roster_changes_do_not_cross_round_and_side_boundaries():
    context, source = sample()
    for frame in source["observations"]["frames"][2:]:
        frame["players"][5]["actor_alias"] = "replacement"
    assert (
        t.sequence(context, source)[-1]["opponents"]["reason"]
        == "roster_exceeds_5v5_scope"
    )
    for f in source["observations"]["frames"][2:]:
        f["instance"] += 1
    for s in context["samples"][2:]:
        s["cut"]["instance"] += 1
    assert t.sequence(context, source)[-1]["opponents"]["alive"] == 1
    for f in source["observations"]["frames"][3:]:
        f["players"][0]["side"], f["players"][5]["side"] = "CT", "T"
    assert t.sequence(context, source)[-1]["allies"]["alive"] is None
    for f in source["observations"]["frames"][3:]:
        f["epoch"] += 1
    for s in context["samples"][3:]:
        s["cut"]["epoch"] += 1
    assert t.sequence(context, source)[-1]["allies"]["alive"] == 2


def test_partial_roster_never_claims_exact_5v5_count():
    context, source = sample()
    for f in source["observations"]["frames"]:
        f["players"].pop()
    final = t.sequence(context, source)[-1]["opponents"]
    assert final["alive"] is None and final["unknown"] == 1


def test_wrong_cut_or_source_rejected():
    context, source = sample()
    source["observations"]["frames"][4]["tick"] += 1
    with pytest.raises(ValueError, match="cut differs"):
        t.sequence(context, source)
    source["source_sha256"] = "f" * 64
    with pytest.raises(ValueError, match="source differs"):
        t.sequence(context, source)


def test_versioned_packet_preserves_v2_and_refuses_render_misalignment():
    source, sources = inputs()
    context = episode(source, sources)
    optional = {packet.rounds.DATA: None, packet.rounds.clock.DATA: None}
    base = packet.v2.build(context, source, sources, None, "de_mirage", optional)
    value = packet.build(context, source, sources, None, "de_mirage", optional)
    stripped = {
        k: v for k, v in value.items() if k not in ("team_status", "team_status_scope")
    }
    stripped["schema_id"] = base["schema_id"]
    assert stripped == base
    html = packet.render(value)
    assert "Resumen de hechos" in html and "Plantado sin confirmar" in html
    corrupted = deepcopy(value)
    corrupted["team_status"][-1]["source_sequence"] += 1
    with pytest.raises(ValueError, match="count samples"):
        packet.render(corrupted)


def test_materializer_refuses_unsafe_or_nonempty_output_before_input(
    tmp_path, monkeypatch
):
    monkeypatch.setattr(
        packet,
        "compute",
        lambda *a, **kw: pytest.fail("read input before output guard"),
    )
    dest = tmp_path / "nonempty"
    dest.mkdir()
    (dest / "kept").write_text("keep")
    with pytest.raises(ValueError):
        packet.materialize(tmp_path / "missing", dest, side="T", intervals=[(1, 2)])
    assert (dest / "kept").read_text() == "keep"
