"""RoundStart is a probe, not proof that delayed callbacks are in the future."""

import copy
import hashlib
import json

import pytest

from ai_coach.contracts import economy_initial_frontiers as v2
from ai_coach.contracts import validate_economy_candidate as v1
from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.tests.test_economy_candidate import (
    candidate_fixture,
    event,
    missing,
    observed,
)


def fixture():
    data = candidate_fixture()
    data.update(
        schema_id="stratai.observed_economy_candidate@2",
        contract_version="1.0.0-candidate.2",
    )
    round_ = data["rounds"][0]
    round_["initial_state"]["callback_sequence"] = 10
    round_["execution"]["events"] = [event(20, 150)]
    probe = copy.deepcopy(round_["initial_state"])
    probe.update(phase="round_start_probe", callback_sequence=5)
    for actor in probe["actors"]:
        actor.update(
            prepurchase_status="unavailable", prepurchase_reason="not_initial_snapshot"
        )
    data["initial_frontiers"] = [
        {
            "instance": 1,
            "round_start_probe": probe,
            "native_rules": {
                k: observed(v, v2.RULE_SOURCES[k])
                for k, v in {
                    "match_started": True,
                    "warmup": False,
                    "freeze": True,
                }.items()
            },
            "official_end_tick": 350,
            "pre_probe_callbacks": [],
            "round_link": {
                "status": "verified",
                "reason": "unique_identity_start_official_end_and_roster",
                "canonical_round_id": data["source_alias"] + ":round:001",
                "source": v2.LINK_SOURCE,
            },
        }
    ]
    return data


def refresh_frame_statuses(data):
    for round_ in data["rounds"]:
        snapshot = round_["initial_state"]
        for actor in snapshot["actors"]:
            actor["prepurchase_status"], actor["prepurchase_reason"] = (
                v1._expected_prebuy(actor, snapshot, round_, v2._events(round_))
            )


def write_fixture(root, data):
    def write(path, value):
        target = root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        payload = json.dumps(value, separators=(",", ":")).encode()
        target.write_bytes(payload)
        return {
            "path": path,
            "bytes": len(payload),
            "sha256": hashlib.sha256(payload).hexdigest(),
        }

    alias = data["source_alias"]
    prefix = f"match_{alias}/canonical/"
    rows = [
        {
            "round_id": f"{alias}:round:001",
            "round_number": 1,
            "start_tick": 100,
            "end_tick": 350,
            "winner_side": "ct",
        }
    ]
    players, comparisons = [], []
    for i in range(1, 11):
        players.append(
            {
                "round_id": rows[0]["round_id"],
                "round_number": 1,
                "player_id": f"steam:{i}",
                "side": "ct" if i <= 5 else "t",
                "money": {
                    "round_start_observed": {"status": "observed", "amount": 800},
                    "round_end_observed": {"status": "observed", "amount": 300},
                },
            }
        )
        for metric, delta in (
            ("end_frame_minus_legacy_round_end_money", None),
            ("initial_minus_legacy_initial_money", -800),
        ):
            comparisons.append(
                {
                    "actor_alias": f"p{i:04d}",
                    "metric": metric,
                    "delta": delta,
                    "status": "unavailable"
                    if delta is None
                    else "diagnostic_different_observation_frontiers",
                }
            )
    data["rounds"][0]["reconciliation"]["comparisons"] = comparisons
    refs = []
    for name, key, items in (
        ("core/rounds.json", "rounds", rows),
        ("derived/economy_players.json", "players", players),
    ):
        ref = write(prefix + name, {"match_id": alias, key: items})
        ref.update(path=name, record_count=len(items))
        refs.append(ref)
    base = write(
        prefix + "manifest.json",
        {
            "schema_id": "stratai.canonical_manifest@3",
            "export_format_version": "3.8.0",
            "match_id": alias,
            "demo_checksum_sha256": data["source_sha256"],
            "artifacts": refs,
        },
    )
    artifact = write(v1.DATA_FILE, data)
    manifest = {
        k: data[k]
        for k in (
            "contract_version",
            "source_alias",
            "source_sha256",
            "build_identifier",
            "training_allowed",
            "product_supported",
        )
    }
    manifest.update(
        schema_id="stratai.economy_candidate_manifest@2",
        base_export_version="3.8.0",
        artifact=artifact,
        base_manifest=base,
        summary=v2.validate_data(data),
    )
    write(v1.MANIFEST_FILE, manifest)
    return root / v1.MANIFEST_FILE


def test_v2_reads_first_probe_and_preserves_v1_reader(tmp_path):
    data = fixture()
    path = write_fixture(tmp_path, data)
    report = v1.validate_candidate(path, require_observed=True)
    assert report["independently_recomputed_round_links"] == 1
    assert report["initial_coverage"]["competitive_complete_teams"] == 2
    v1.validate_data(v2.frame_projection(data))
    team = v2.initial_team_state(data, 1, "p0001", 100)
    assert len(team["actors"]) == 5
    assert all(a["observation_source"] == "round_start_probe" for a in team["actors"])
    assert set(team) == {"instance", "t0_tick", "side", "context", "actors"}


def test_smoke_requires_nonempty_competitive_affected_records(tmp_path):
    data = fixture()
    frontier = data["initial_frontiers"][0]
    frontier["official_end_tick"] = None
    frontier["round_link"].update(
        status="excluded", reason="official_end_unavailable", canonical_round_id=None
    )
    path = write_fixture(tmp_path, data)
    assert (
        v1.validate_candidate(path)["initial_coverage"]["competitive_complete_teams"]
        == 0
    )
    with pytest.raises(RuntimeContractError, match="empty competitive"):
        v1.validate_candidate(path, require_observed=True)


@pytest.mark.parametrize("excluded", [False, True])
def test_common_initial_accessor_cannot_bypass_v2_round_gate(excluded):
    data = fixture()
    if excluded:
        data["initial_frontiers"][0]["round_link"]["status"] = "excluded"
        with pytest.raises(RuntimeContractError, match="competitive round"):
            v1.initial_team_state(data, 1, "p0001", 100)
    else:
        assert v1.initial_team_state(data, 1, "p0001", 100) == v2.initial_team_state(
            data, 1, "p0001", 100
        )


@pytest.mark.parametrize("seq", [3, 7, 11])
@pytest.mark.parametrize("kind", ["pickup", "drop", "refund"])
def test_same_tick_callbacks_block_before_probe_after_probe_and_after_frame(seq, kind):
    data = fixture()
    activity = event(seq, 100, kind=kind)
    data["rounds"][0]["reconciliation"]["other_events"] = [activity]
    if seq < 5:
        data["initial_frontiers"][0]["pre_probe_callbacks"] = [
            {
                k: activity[k]
                for k in ("actor_alias", "kind", "tick", "callback_sequence")
            }
        ]
    refresh_frame_statuses(data)
    v2.validate_data(data)
    with pytest.raises(RuntimeContractError, match="complete prebuy"):
        v2.initial_team_state(data, 1, "p0001", 100)
    assert v2.coverage(data)["initial_actors_accredited"] == 9


def test_future_refunds_opponent_economy_and_later_round_do_not_change_initial():
    data = fixture()
    before = v2.initial_team_state(data, 1, "p0001", 100)
    for compartment in (
        data["rounds"][0]["initial_state"],
        data["initial_frontiers"][0]["round_start_probe"],
    ):
        for actor in compartment["actors"][5:]:
            actor["facts"]["money"] = missing(v1.ACTOR_SOURCES["money"])
    data["rounds"][0]["execution"]["events"].append(event(30, 170, kind="refund"))
    data["rounds"][0]["reconciliation"]["comparisons"] = [{"future_only": 9999}]
    data["rounds"].append({"instance": 2, "future_only": "arbitrary"})
    assert v2.initial_team_state(data, 1, "p0001", 100) == before
    before["actors"][0]["facts"]["money"]["value"] = 9999
    assert (
        v2.initial_team_state(data, 1, "p0001", 100)["actors"][0]["facts"]["money"][
            "value"
        ]
        == 0
    )


def test_native_activity_before_first_round_is_not_lost_by_reset():
    data = fixture()
    data["initial_frontiers"][0]["pre_probe_callbacks"] = [
        {"actor_alias": "p0001", "kind": "refund", "tick": 100, "callback_sequence": 2}
    ]
    v2.validate_data(data)
    assert v2.coverage(data)["abstentions"] == {"activity_in_initial_tick": 1}


def test_missing_probe_falls_back_without_merging_or_backdating():
    data = fixture()
    actor = data["initial_frontiers"][0]["round_start_probe"]["actors"][0]
    actor["facts"]["inventory"] = missing(v1.ACTOR_SOURCES["inventory"])
    v2.validate_data(data)
    team = v2.initial_team_state(data, 1, "p0001", 100)
    assert team["actors"][0]["observation_source"] == "first_frame_done"
    frame = data["rounds"][0]["initial_state"]
    frame["tick"] = 101
    for fact in [
        *frame["context"].values(),
        *[f for a in frame["actors"] for f in a["facts"].values()],
    ]:
        if fact["status"] == "observed":
            fact["observation_tick"] = fact["availability_tick"] = 101
    refresh_frame_statuses(data)
    v2.validate_data(data)
    with pytest.raises(RuntimeContractError, match="complete prebuy"):
        v2.initial_team_state(data, 1, "p0001", 100)


@pytest.mark.parametrize(
    "mutation",
    [
        "missing_fact",
        "spent",
        "disconnected",
        "side_switch",
        "native_missing",
        "native_false",
        "excluded_link",
        "changed_t0",
    ],
)
def test_initial_access_abstains(mutation):
    data = fixture()
    frontier = data["initial_frontiers"][0]
    snapshots = [frontier["round_start_probe"], data["rounds"][0]["initial_state"]]
    for snap in snapshots:
        actor = snap["actors"][0]
        if mutation == "missing_fact":
            actor["facts"]["kit"] = missing(v1.ACTOR_SOURCES["kit"])
        if mutation == "spent":
            actor["facts"]["spent_native"]["value"] = 300
        if mutation == "disconnected":
            actor["connected"] = False
    if mutation == "side_switch":
        snapshots[0]["actors"][0]["side"] = "T"
    if mutation == "native_missing":
        frontier["native_rules"]["freeze"] = missing(v2.RULE_SOURCES["freeze"])
    if mutation == "native_false":
        frontier["native_rules"]["match_started"]["value"] = False
    if mutation == "excluded_link":
        frontier["round_link"]["status"] = "excluded"
    with pytest.raises(RuntimeContractError):
        v2.initial_team_state(
            data, 1, "p0001", 101 if mutation == "changed_t0" else 100
        )


@pytest.mark.parametrize(
    "mutation",
    [
        "future_fact",
        "future_context",
        "reuse_sequence",
        "probe_prebuy",
        "oracle",
        "reused_entity",
        "omit_preprobe",
        "wrong_native_source",
    ],
)
def test_v2_validator_rejects_fabricated_frontiers(mutation):
    data = fixture()
    f = data["initial_frontiers"][0]
    probe = f["round_start_probe"]
    if mutation == "future_fact":
        probe["actors"][0]["facts"]["money"]["availability_tick"] = 101
    if mutation == "future_context":
        probe["context"]["score_CT"]["availability_tick"] = 300
    if mutation == "reuse_sequence":
        probe["callback_sequence"] = 10
    if mutation == "probe_prebuy":
        probe["actors"][0]["prepurchase_status"] = "available"
    if mutation == "oracle":
        probe["winner"] = "CT"
    if mutation == "reused_entity":
        probe["actors"][0]["facts"]["inventory"]["value"] *= 2
    if mutation == "wrong_native_source":
        f["native_rules"]["freeze"]["source"] = "next_round"
    if mutation == "omit_preprobe":
        data["rounds"][0]["reconciliation"]["other_events"] = [event(3, 100)]
        refresh_frame_statuses(data)
    with pytest.raises(RuntimeContractError):
        v2.validate_data(data)


@pytest.mark.parametrize(
    "mutation",
    [
        "start",
        "official_end",
        "participant_side",
        "participant_identity",
        "same_number_restart",
        "duplicate_row",
        "claimed_exclusion",
    ],
)
def test_round_links_recomputed_even_after_rehash(tmp_path, mutation):
    data = fixture()
    path = write_fixture(tmp_path, data)
    base_path = tmp_path / f"match_{data['source_alias']}/canonical/manifest.json"
    base = json.loads(base_path.read_text())
    name = (
        "derived/economy_players.json"
        if mutation.startswith("participant")
        else "core/rounds.json"
    )
    target = base_path.parent / name
    body = json.loads(target.read_text())
    if mutation == "start":
        body["rounds"][0]["start_tick"] = 99
    if mutation == "official_end":
        body["rounds"][0]["end_tick"] = 351
    if mutation == "same_number_restart":
        body["rounds"][0].update(start_tick=90, end_tick=95)
    if mutation == "duplicate_row":
        body["rounds"] *= 2
    if mutation == "participant_side":
        body["players"][0]["side"] = "t"
    if mutation == "participant_identity":
        body["players"][0]["player_id"] = "steam:999"
    if mutation == "claimed_exclusion":
        data["initial_frontiers"][0]["round_link"].update(
            status="excluded",
            reason="canonical_boundaries_unmatched",
            canonical_round_id=None,
        )
    payload = json.dumps(body).encode()
    target.write_bytes(payload)
    for ref in base["artifacts"]:
        if ref["path"] == name:
            ref.update(
                bytes=len(payload),
                sha256=hashlib.sha256(payload).hexdigest(),
                record_count=len(
                    body["players" if mutation.startswith("participant") else "rounds"]
                ),
            )
    with pytest.raises(RuntimeContractError, match="round correspondence"):
        v2.validate_round_links(data, base, tmp_path)
    assert path.exists()
