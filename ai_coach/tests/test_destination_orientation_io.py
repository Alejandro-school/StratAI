"""Retained orientation artifacts must recompute from their original bindings."""

import json
import stat
import tempfile
from pathlib import Path
from types import SimpleNamespace

import pytest

from ai_coach.interpretation import candidate_context
from ai_coach.interpretation import destination_orientation_io as io
from ai_coach.tests.test_native_place_context import (
    encode,
    export_fixture,
    replace_snapshot,
    sha,
)


@pytest.mark.parametrize("with_native", [True, False])
def test_roundtrip_retains_separate_outputs_and_preserves_original_inputs(
    tmp_path, with_native
):
    candidate, args, _, _ = export_fixture(tmp_path / "source")
    before = args["candidate_path"].read_bytes()
    if not with_native:
        args = {key: args[key] for key in ("candidate_path", "candidate_sha256")}
    output = tmp_path / "candidate"
    output.mkdir()
    manifest = io.materialize(output, **args)
    assert io.validate(output) == manifest
    value = json.loads((output / io.DATA).read_bytes())
    assert value["status"] == ("hypothesis" if with_native else "abstain")
    assert manifest["preparation_prediction_replaced"] is False
    assert manifest["training_allowed"] is False
    assert manifest["information_delta"] == (
        "native_place_property_added"
        if with_native
        else "none_native_source_unavailable"
    )
    assert args["candidate_path"].read_bytes() == before
    assert json.loads(before) == candidate
    assert (
        "ai_coach/contracts/review_extra_context.schema.json"
        in manifest["producer_sha256"]
    )
    retained = {p.name: p.read_bytes() for p in output.iterdir()}
    with pytest.raises(ValueError, match="empty"):
        io.materialize(output, **args)
    assert retained == {p.name: p.read_bytes() for p in output.iterdir()}


@pytest.mark.parametrize("kind", ["non_temp", "temp_root", "occupied", "reparse"])
def test_output_guard_rejects_before_any_source_read(tmp_path, monkeypatch, kind):
    allowed = tmp_path / "allowed"
    allowed.mkdir()
    monkeypatch.setattr(tempfile, "gettempdir", lambda: str(allowed))
    if kind == "temp_root":
        output = allowed
    else:
        output = (tmp_path if kind == "non_temp" else allowed) / "output"
        output.mkdir()
    if kind == "occupied":
        (output / "keep.bin").write_bytes(b"keep")
    if kind == "reparse":
        real_lstat = Path.lstat

        def as_link(path, *args, **kwargs):
            if path == output:
                return SimpleNamespace(st_mode=stat.S_IFLNK, st_file_attributes=0)
            return real_lstat(path, *args, **kwargs)

        monkeypatch.setattr(Path, "lstat", as_link)

    def forbidden_read(*_args, **_kwargs):
        pytest.fail("source read preceded output preflight")

    monkeypatch.setattr(io, "_load", forbidden_read)
    with pytest.raises(ValueError):
        io.materialize(
            output, candidate_path="must-not-open", candidate_sha256="0" * 64
        )
    if kind == "occupied":
        assert (output / "keep.bin").read_bytes() == b"keep"


def test_incomplete_native_binding_rejected_before_source_read(tmp_path, monkeypatch):
    output = tmp_path / "output"
    output.mkdir()

    def forbidden(*_args, **_kwargs):
        pytest.fail("incomplete binding reached reader")

    monkeypatch.setattr(io, "_load", forbidden)
    with pytest.raises(ValueError, match="incomplete native"):
        io.materialize(
            output,
            candidate_path="absent",
            candidate_sha256="0" * 64,
            target_id="unbound",
        )
    assert not list(output.iterdir())


@pytest.mark.parametrize("artifact", [io.DATA, io.NATIVE])
def test_rehashed_output_forgery_fails_independent_source_recomputation(
    tmp_path, artifact
):
    _, args, _, _ = export_fixture(tmp_path / "source")
    output = tmp_path / "output"
    output.mkdir()
    io.materialize(output, **args)
    value = json.loads((output / artifact).read_bytes())
    if artifact == io.DATA:
        value["preferred_site"] = "B"
        value["alternatives"][0]["site"] = "A"
        id_field = "orientation_id"
    else:
        value["observations"][0]["value"] = "BombsiteB"
        id_field = "native_place_context_id"
    value[id_field] = candidate_context.content_hash(
        {k: v for k, v in value.items() if k != id_field}
    )
    raw = io._bytes(value)
    (output / artifact).write_bytes(raw)
    manifest = json.loads((output / io.MANIFEST).read_bytes())
    manifest["artifacts"][artifact] = {"sha256": sha(raw), "bytes": len(raw)}
    (output / io.MANIFEST).write_bytes(encode(manifest))
    with pytest.raises(ValueError, match="source recomputation"):
        io.validate(output)


def test_changed_original_candidate_fails_bound_byte_hash(tmp_path):
    _, args, _, _ = export_fixture(tmp_path / "source")
    output = tmp_path / "output"
    output.mkdir()
    io.materialize(output, **args)
    args["candidate_path"].write_bytes(b"{}")
    with pytest.raises(ValueError, match="checksum"):
        io.validate(output)


def test_unverified_native_current_read_emits_abstention_not_stale_destination(
    tmp_path,
):
    _, args, manifest, snapshot = export_fixture(tmp_path / "source")
    snapshot["actors"][0]["place"] = {"status": "read_error", "value": None}
    replace_snapshot(args, manifest, snapshot)
    output = tmp_path / "output"
    output.mkdir()
    io.materialize(output, **args)
    io.validate(output)
    value = json.loads((output / io.DATA).read_bytes())
    assert value["status"] == "abstain"
    assert value["reasons"] == ["current_native_place_value_unavailable"]
    assert (
        value["observed_facts"]["native_places"][0]["last_delivery"]["value"]
        == "BombsiteA"
    )


def test_changed_dependency_or_extra_artifact_rejects_prior_materialization(
    tmp_path, monkeypatch
):
    _, args, _, _ = export_fixture(tmp_path / "source")
    output = tmp_path / "output"
    output.mkdir()
    manifest = io.materialize(output, **args)
    fingerprints = dict(manifest["producer_sha256"])
    fingerprints["ai_coach/contracts/review_extra_context.schema.json"] = "0" * 64
    with monkeypatch.context() as patch:
        patch.setattr(io, "_code", lambda: fingerprints)
        with pytest.raises(ValueError, match="producer hash"):
            io.validate(output)
    (output / "human_answer.json").write_bytes(b"{}")
    with pytest.raises(ValueError, match="inventory"):
        io.validate(output)


def test_scope_gate_forgery_rejected_even_without_prediction_change(tmp_path):
    _, args, _, _ = export_fixture(tmp_path / "source")
    output = tmp_path / "output"
    output.mkdir()
    manifest = io.materialize(output, **args)
    manifest["preparation_prediction_replaced"] = True
    (output / io.MANIFEST).write_bytes(encode(manifest))
    with pytest.raises(ValueError, match="scope gate"):
        io.validate(output)
