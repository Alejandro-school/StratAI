"""Artifact integrity and output safety for factual candidate materialization."""

import hashlib
import json
import tempfile

import pytest

from ai_coach.interpretation import candidate_context_io as io
from ai_coach.tests.test_candidate_context import fixture


def source_files(root, historical=False):
    root.mkdir()
    packet, inputs = fixture()
    arguments = {"case_id": "E03", "instrument": inputs["instrument"]}
    payloads = {"packet": packet, "kills": inputs["kills"]}
    if historical:
        arguments["instrument"] = "stratai.review_web@5"
    else:
        payloads.update(extra=inputs["extra"], bomb=inputs["bomb"])
    for key, value in payloads.items():
        path = root / (key + ".json")
        raw = json.dumps(value, ensure_ascii=False).encode("utf-8")
        path.write_bytes(raw)
        arguments[key + "_path"] = path
        arguments[key + "_sha256"] = hashlib.sha256(raw).hexdigest()
    return arguments


@pytest.mark.parametrize("historical", [False, True])
def test_roundtrip_preserves_instrument_and_hash_bound_input(tmp_path, historical):
    args = source_files(tmp_path / "sources", historical)
    output = tmp_path / "candidate"
    output.mkdir()
    manifest = io.materialize(output, **args)
    assert io.validate(output) == manifest
    value = json.loads((output / io.DATA).read_bytes())
    assert value["instrument"] == args["instrument"]
    assert (value["supplements"]["extra"] is None) is historical
    assert manifest["predictions_created"] == 0
    assert (
        "ai_coach/contracts/candidate_context.schema.json"
        in manifest["producer_sha256"]
    )
    for key, receipt in value["input_files"].items():
        assert receipt["sha256"] == args[key + "_sha256"]
        assert (
            hashlib.sha256(args[key + "_path"].read_bytes()).hexdigest()
            == receipt["sha256"]
        )
    before = {p.name: p.read_bytes() for p in output.iterdir()}
    with pytest.raises(ValueError, match="empty"):
        io.materialize(output, **args)
    assert before == {p.name: p.read_bytes() for p in output.iterdir()}


@pytest.mark.parametrize("kind", ["non_temp", "temp_root", "occupied"])
def test_unsafe_output_rejected_before_reading_sources(tmp_path, monkeypatch, kind):
    allowed = tmp_path / "allowed"
    allowed.mkdir()
    monkeypatch.setattr(tempfile, "gettempdir", lambda: str(allowed))
    if kind == "non_temp":
        output = tmp_path / "outside"
        output.mkdir()
    elif kind == "temp_root":
        output = allowed
    else:
        output = allowed / "occupied"
        output.mkdir()
        (output / "keep.txt").write_bytes(b"retain me")
    before = {p.name: p.read_bytes() for p in output.iterdir() if p.is_file()}

    def forbidden_read(**_kwargs):
        pytest.fail("unsafe output was accepted before source read")

    monkeypatch.setattr(io.candidate_context, "load", forbidden_read)
    with pytest.raises(ValueError):
        io.materialize(output, packet_path="must-not-be-read.json")
    assert before == {p.name: p.read_bytes() for p in output.iterdir() if p.is_file()}


def test_forged_snapshot_rejected_even_if_artifact_hash_is_updated(tmp_path):
    args = source_files(tmp_path / "sources")
    output = tmp_path / "candidate"
    output.mkdir()
    io.materialize(output, **args)
    data_path = output / io.DATA
    value = json.loads(data_path.read_bytes())
    value["snapshot"]["objective"]["phase"] = "postplant"
    raw = (
        json.dumps(value, sort_keys=True, indent=2, ensure_ascii=False).encode("utf-8")
        + b"\n"
    )
    data_path.write_bytes(raw)
    manifest_path = output / io.MANIFEST
    manifest = json.loads(manifest_path.read_bytes())
    manifest["artifacts"][io.DATA] = {
        "bytes": len(raw),
        "sha256": hashlib.sha256(raw).hexdigest(),
    }
    manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
    with pytest.raises(ValueError, match="facts differ from verified sources"):
        io.validate(output)


def test_changed_source_rejected_on_artifact_validation(tmp_path):
    args = source_files(tmp_path / "sources")
    output = tmp_path / "candidate"
    output.mkdir()
    io.materialize(output, **args)
    args["packet_path"].write_bytes(b"{}")
    with pytest.raises(ValueError, match="packet file hash differs"):
        io.validate(output)


def test_changed_contract_rejects_prior_artifact(tmp_path, monkeypatch):
    args = source_files(tmp_path / "sources")
    output = tmp_path / "candidate"
    output.mkdir()
    manifest = io.materialize(output, **args)
    fingerprints = dict(manifest["producer_sha256"])
    fingerprints["ai_coach/contracts/candidate_context.schema.json"] = "0" * 64
    monkeypatch.setattr(io, "_code", lambda: fingerprints)
    with pytest.raises(ValueError, match="producer hash differs"):
        io.validate(output)


def test_extra_artifact_rejected(tmp_path):
    args = source_files(tmp_path / "sources")
    output = tmp_path / "candidate"
    output.mkdir()
    io.materialize(output, **args)
    (output / "human-answer.json").write_bytes(b"{}")
    with pytest.raises(ValueError, match="inventory differs"):
        io.validate(output)
