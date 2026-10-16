import copy
import json

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import attempt_origin as s
from ai_coach.support import objective as o
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF
from ai_coach.tests.test_support_objective_clock import build as clock_report
from ai_coach.tests.test_support_objective_clock import origin_fixture, scalar


def inputs():
    args = origin_fixture()
    _support, _objective, ownership, clock = args
    native = clock["observations"]["frames"]
    by_frame = {r["frame"]: r for r in native}
    entity = ownership["observations"]["states"][0]["entities"][0]
    item = {
        "entity_life": entity["entity_life"],
        "handle": entity["handle"],
        "created_frame": 99,
        "current": True,
        "fields": {
            "m_bStartedArming": scalar(False),
            "m_fArmedTime": scalar(0, declared="GameTime_t"),
            "m_hOwnerEntity": scalar(
                entity["owner"]["handle"], "uint64", "CHandle< C_BaseEntity >"
            ),
        },
        "definition": entity["definition"],
        "owner": entity["owner"],
        "inventory_status": "observed",
        "inventory_slots": [
            ownership["observations"]["states"][0]["players"][0]["slots"][0]
        ],
    }
    deliveries, frames = [], []

    def emit(frame, kind, name, value):
        previous = (
            item["fields"][name]
            if kind != "created"
            else scalar(declared=value["declared_type"])
        )
        deliveries.append(
            {
                "sequence": len(deliveries) + 1,
                "frame": frame,
                "tick": frame + 100,
                "entity_life": item["entity_life"],
                "kind": kind,
                "property": name,
                "previous": copy.deepcopy(previous),
                "value": copy.deepcopy(value),
            }
        )
        item["fields"][name] = copy.deepcopy(value)

    for frame in [99, *range(102, 241)]:
        if frame == 99:
            for name, value in list(item["fields"].items()):
                emit(frame, "created", name, value)
        if frame == 102:
            emit(frame, "update", "m_bStartedArming", scalar(True))
            emit(frame, "update", "m_fArmedTime", scalar(106.25, declared="GameTime_t"))
        source = by_frame.get(frame)
        network = {
            kind: [copy.deepcopy(source["network"][kind])]
            if source
            else [
                {
                    "frame": frame,
                    "tick": frame + 100,
                    "order": frame * 100 + (1 if kind == "tick" else 2),
                    "value": scalar(frame + 6500, "uint32"),
                }
            ]
            for kind in ("tick", "packet_tick")
        }
        network["interval"] = []
        frames.append(
            {
                "sequence": len(frames) + 1,
                "frame": frame,
                "tick": frame + 100,
                "instance": 1,
                "epoch": 0,
                "phase": "playing",
                "delivery_sequence": len(deliveries),
                "network": network,
                "interval": copy.deepcopy(native[0]["network"]["interval"])
                if frame >= 100
                else None,
                "rules": copy.deepcopy(native[0]["rules"]),
                "items": [copy.deepcopy(item)],
            }
        )
    for row in native:
        row["items"][0]["fields"]["m_bStartedArming"] = scalar(row["frame"] >= 102)
        row["items"][0]["fields"]["m_fArmedTime"] = scalar(
            106.25 if row["frame"] >= 102 else 0, declared="GameTime_t"
        )
    cuts = []
    for row in native:
        available = [r for r in frames if r["frame"] <= row["frame"]]
        cuts.append(
            {
                "source_sequence": row["source_sequence"],
                **{k: row[k] for k in ("frame", "tick", "instance", "epoch")},
                "clock_order": row["order"],
                "frame_sequence": len(available),
                "delivery_sequence": available[-1]["delivery_sequence"],
            }
        )
    data = {
        "schema_id": "stratai.support_attempt_origin_observations@1",
        **{
            k: clock[k]
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
        "references": {**clock["references"], "clock_source": REF},
        "capture_contract": "native_arming_delivery_and_every_active_FrameDone@1",
        "observations": {"deliveries": deliveries, "frames": frames, "cuts": cuts},
    }
    return [*args, data]


def features(args):
    anchor = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0][0]
    return s.initial_features(
        *args, anchor["anchor_id"], ACTOR, anchor["t0_tick"], anchor["t0_frame"]
    )


def build(args):
    previous = clock_report(*args[:4])
    return s.build_report(
        *args,
        previous,
        {**previous["references"], "attempt_source": REF, "clock_context": REF},
    )


def test_true_transition_before_support_sample_has_native_duration_and_progress():
    args = inputs()
    result = features(args)
    duration = 106.25 - 6602 / 64
    assert result["self"]["planting_origin_state"] == o.fact("native_transition")
    assert result["self"]["planting_duration_seconds"] == o.fact(duration)
    assert result["self"]["planting_elapsed_seconds"] == o.fact(6 / 64)
    assert result["self"]["planting_progress"] == o.fact(1 - 3 / duration)
    assert result["ally"]["planting_progress"]["status"] == "not_applicable"
    report = build(args)
    assert report["summary"]["anchors"] == 2
    assert (
        report["summary"]["fact_status_counts"]["self.planting_progress"]["observed"]
        == 1
    )
    assert report["rows"][0]["origin_reference"]["self"]["frame"] == 102
    assert all(
        not report[k]
        for k in ("comparison_allowed", "training_allowed", "product_supported")
    )
    assert all(
        token not in json.dumps(result)
        for token in (
            ACTOR,
            ALLY,
            ENEMY,
            "entity_life",
            "pawn_handle",
            "delivery_sequence",
        )
    )


@pytest.mark.parametrize(
    "mode",
    [
        "paused",
        "pause_missing",
        "tick_absent",
        "tick_disagrees",
        "multiple_packet_ticks",
        "interval_change",
        "rules_life",
        "round",
        "epoch",
        "rewind",
        "native_type",
        "undeclared_deadline",
        "inventory_missing",
        "owner_serial",
        "duplicate_slot",
        "frame_gap",
    ],
)
def test_positive_origin_is_invalidated_by_intermediate_ambiguity(mode):
    args = inputs()
    row = next(r for r in args[4]["observations"]["frames"] if r["frame"] == 103)
    if mode == "paused":
        row["rules"]["fields"]["m_bGamePaused"] = scalar(True)
    elif mode == "pause_missing":
        row["rules"]["fields"]["m_bGamePaused"] = scalar()
    elif mode == "tick_absent":
        row["network"]["tick"] = []
    elif mode == "tick_disagrees":
        row["network"]["tick"][0]["value"]["value"] += 1
    elif mode == "multiple_packet_ticks":
        message = copy.deepcopy(row["network"]["packet_tick"][0])
        message["order"] += 1
        message["value"]["value"] += 1
        row["network"]["packet_tick"].append(message)
    elif mode == "interval_change":
        row["interval"]["order"] += 1
    elif mode == "rules_life":
        row["rules"]["entity_life"] = "support-clock-object-999999"
    elif mode == "round":
        row["instance"] += 1
    elif mode == "epoch":
        row["epoch"] += 1
    elif mode == "rewind":
        for kind in ("tick", "packet_tick"):
            row["network"][kind][0]["value"]["value"] = 6600
    elif mode == "native_type":
        row["network"]["tick"][0]["value"] = scalar(6603, "float32")
    elif mode == "undeclared_deadline":
        for d in args[4]["observations"]["deliveries"]:
            if d["property"] == "m_fArmedTime":
                d["previous"]["declared_type"] = d["value"]["declared_type"] = None
        for f in args[4]["observations"]["frames"]:
            f["items"][0]["fields"]["m_fArmedTime"]["declared_type"] = None
        for f in args[3]["observations"]["frames"]:
            f["items"][0]["fields"]["m_fArmedTime"]["declared_type"] = None
    elif mode == "inventory_missing":
        row["items"][0]["inventory_status"] = "absent"
    elif mode == "owner_serial":
        row["items"][0]["owner"]["handle"] += 16384
    elif mode == "duplicate_slot":
        row["items"][0]["inventory_slots"] *= 2
    elif mode == "frame_gap":
        args[4]["observations"]["frames"].remove(row)
        for seq, r in enumerate(args[4]["observations"]["frames"], 1):
            r["sequence"] = seq
        for cut in args[4]["observations"]["cuts"]:
            if cut["frame"] > 103:
                cut["frame_sequence"] -= 1
    assert features(args)["self"]["planting_progress"]["status"] == "unavailable"


@pytest.mark.parametrize(
    "mode",
    [
        "forged_baseline",
        "future_delivery",
        "predecessor",
        "snapshot",
        "cut_frontier",
        "clock_join",
        "unknown_field",
        "life_reuse",
    ],
)
def test_malformed_or_fabricated_sources_are_rejected(mode):
    args = inputs()
    obs = args[4]["observations"]
    if mode == "forged_baseline":
        obs["deliveries"][3]["kind"] = "creation_replay"
    elif mode == "future_delivery":
        obs["deliveries"][3]["frame"] = 104
    elif mode == "predecessor":
        obs["deliveries"][3]["previous"] = scalar(True)
    elif mode == "snapshot":
        obs["frames"][2]["items"][0]["fields"]["m_fArmedTime"]["value"] += 1
    elif mode == "cut_frontier":
        obs["cuts"][2]["delivery_sequence"] -= 1
    elif mode == "clock_join":
        obs["cuts"][2]["clock_order"] += 1
    elif mode == "unknown_field":
        obs["frames"][2]["oracle"] = True
    elif mode == "life_reuse":
        obs["frames"][2]["items"][0]["handle"] += 16384
    with pytest.raises((ValueError, ValidationError)):
        features(args)


def test_malformed_future_does_not_change_initial_features():
    args = inputs()
    before = features(args)
    args[4]["observations"]["deliveries"].append({"frame": 109, "oracle": "malformed"})
    args[4]["observations"]["frames"].append({"frame": 109, "oracle": "malformed"})
    args[4]["observations"]["cuts"].append(
        {"source_sequence": 4, "oracle": "malformed"}
    )
    assert features(args) == before
    with pytest.raises((ValueError, ValidationError)):
        s.validate_source(args[4], args[0], args[3])


def test_reader_recomputes_instead_of_trusting_serialized_progress(
    tmp_path, monkeypatch
):
    args = inputs()
    report = build(args)
    refs = report["references"]
    monkeypatch.setattr(s.c, "validate_artifact", lambda *x: None)
    monkeypatch.setattr(
        s, "report_inputs", lambda *x: (*args, clock_report(*args[:4]), refs)
    )
    (tmp_path / s.REPORT).write_text(json.dumps(report))
    assert (
        s.validate_artifact(tmp_path, tmp_path / s.DATA, None, None)
        == report["summary"]
    )
    report["rows"][0]["initial"]["self"]["planting_progress"]["value"] += 0.1
    (tmp_path / s.REPORT).write_text(json.dumps(report))
    with pytest.raises(ValueError, match="recomputation"):
        s.validate_artifact(tmp_path, tmp_path / s.DATA, None, None)


def rewrite_events(args, events, baseline_active=False, baseline_deadline=0):
    obs = args[4]["observations"]
    first = obs["frames"][0]["items"][0]
    values = copy.deepcopy(first["fields"])
    values["m_bStartedArming"] = scalar(baseline_active)
    values["m_fArmedTime"] = scalar(baseline_deadline, declared="GameTime_t")
    source = [(99, "created", name, value) for name, value in values.items()] + events
    deliveries = []
    for frame, kind, name, value in sorted(source, key=lambda x: x[0]):
        previous = (
            scalar(declared=value["declared_type"])
            if kind == "created"
            else copy.deepcopy(values[name])
        )
        deliveries.append(
            {
                "sequence": len(deliveries) + 1,
                "frame": frame,
                "tick": frame + 100,
                "entity_life": first["entity_life"],
                "kind": kind,
                "property": name,
                "previous": previous,
                "value": value,
            }
        )
        values[name] = copy.deepcopy(value)
    obs["deliveries"] = deliveries
    for row in obs["frames"]:
        available = [d for d in deliveries if d["frame"] <= row["frame"]]
        row["delivery_sequence"] = len(available)
        for d in available:
            row["items"][0]["fields"][d["property"]] = copy.deepcopy(d["value"])
    for row in args[3]["observations"]["frames"]:
        latest = next(r for r in reversed(obs["frames"]) if r["frame"] <= row["frame"])
        for name in ("m_bStartedArming", "m_fArmedTime"):
            row["items"][0]["fields"][name] = copy.deepcopy(
                latest["items"][0]["fields"][name]
            )
    for cut in obs["cuts"]:
        cut["delivery_sequence"] = len(
            [d for d in deliveries if d["frame"] <= cut["frame"]]
        )


@pytest.mark.parametrize(
    "mode",
    [
        "baseline_active",
        "creation_replay",
        "stale_deadline",
        "zero_duration",
        "negative_duration",
        "multiple_starts",
        "deadline_changed",
        "cancelled",
        "restart",
        "deadline_before_flag",
        "expired",
    ],
)
def test_start_duration_and_attempt_boundaries(mode):
    args = inputs()
    flag = lambda frame, value: (frame, "update", "m_bStartedArming", scalar(value))
    deadline = lambda frame, value: (
        frame,
        "update",
        "m_fArmedTime",
        scalar(value, declared="GameTime_t"),
    )
    events = [flag(102, True), deadline(102, 106.25)]
    baseline = False
    initial_deadline = 0
    if mode == "baseline_active":
        events = []
        baseline = True
        initial_deadline = 106.25
    elif mode == "creation_replay":
        events = [
            (99, "creation_replay", "m_bStartedArming", scalar(True)),
            (
                99,
                "creation_replay",
                "m_fArmedTime",
                scalar(106.25, declared="GameTime_t"),
            ),
        ]
    elif mode == "stale_deadline":
        events = [flag(102, True)]
        initial_deadline = 106.25
    elif mode == "zero_duration":
        events = [flag(102, True), deadline(102, 6602 / 64)]
    elif mode == "negative_duration":
        events = [flag(102, True), deadline(102, 6601 / 64)]
    elif mode == "multiple_starts":
        events = [
            flag(102, True),
            flag(102, False),
            flag(102, True),
            deadline(102, 106.25),
        ]
    elif mode == "deadline_changed":
        events += [deadline(103, 107)]
    elif mode == "cancelled":
        events += [flag(105, False)]
    elif mode == "restart":
        events += [flag(104, False), flag(105, True), deadline(105, 107)]
    elif mode == "deadline_before_flag":
        events = list(reversed(events))
    elif mode == "expired":
        events = [flag(102, True), deadline(102, 6607 / 64)]
    rewrite_events(args, events, baseline, initial_deadline)
    result = features(args)["self"]
    if mode == "cancelled":
        assert result["planting_progress"]["status"] == "not_applicable"
    elif mode == "expired":
        assert result["planting_progress"] == o.fact(1)
        assert result["planting_origin_state"] == o.fact("expired")
    elif mode in ("restart", "deadline_before_flag"):
        assert result["planting_progress"]["status"] == "observed"
        expected = 107 - 6605 / 64 if mode == "restart" else 106.25 - 6602 / 64
        assert result["planting_duration_seconds"] == o.fact(expected)
    else:
        assert result["planting_progress"]["status"] == "unavailable"
        if mode in ("baseline_active", "creation_replay"):
            assert (
                result["planting_progress"]["reason"]
                == "baseline_active_without_start_transition"
            )


def test_unrelated_player_clocks_do_not_change_own_or_ally_planting_progress():
    args = inputs()
    before = features(args)
    for row in args[3]["observations"]["frames"]:
        for player in row["players"]:
            if player["actor_alias"] == ENEMY:
                player["fields"]["m_flSimulationTime"] = scalar(
                    88888, declared="float32"
                )
                player["fields"]["m_nTickBase"] = scalar(77777, "int")
    assert features(args) == before


def test_full_owner_handle_is_not_restricted_to_clock_numeric_range():
    item = copy.deepcopy(inputs()[4]["observations"]["frames"][1]["items"][0])
    item["owner"]["handle"] = 1500000000
    item["fields"]["m_hOwnerEntity"] = scalar(1500000000, "uint64")
    assert s.owner_key(item, [item]) == (item["owner"]["target_life"], 1500000000)
