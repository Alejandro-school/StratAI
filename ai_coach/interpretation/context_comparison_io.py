"""Guarded temporary CS3 packages. No corpus writes or network transport."""

import hashlib
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import candidate_context as candidate
from ai_coach.interpretation import common_context_io as common_io
from ai_coach.interpretation import context_comparison as comparison
from ai_coach.support.anchors import require

DATA = "comparison.json"
MANIFEST = "comparison_manifest.json"
ROOT = Path(__file__).resolve().parents[2]


def sha(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def code_hashes():
    return {**common_io.code_hashes(), **{p: sha(ROOT / p) for p in (
        "ai_coach/interpretation/context_comparison.py",
        "ai_coach/interpretation/context_comparison_io.py",
    )}}


def write(path, value):
    with Path(path).open("x", encoding="utf-8") as stream:
        json.dump(value, stream, indent=2, ensure_ascii=False, allow_nan=False)
        stream.write("\n")


def materialize(output, *, candidate_path, candidate_sha256, through_index=None):
    output = validate_output_directory(output)
    path = Path(candidate_path)
    require(sha(path) == candidate_sha256, "candidate byte hash differs")
    value = json.loads(path.read_bytes())
    session = comparison.Session(value, through_index=through_index)
    write(output / "candidate_evidence.json", value)
    write(output / "model_input.json", session.consultation().initial())
    write(output / "causal_samples.json", session._archive)
    write(output / "private_citations.json", session._private)
    write(output / "semantic_prompt.json", {"version": comparison.VERSION, "prompt": comparison.PROMPT})
    write(output / DATA, comparison.compare_local(session))
    files = {p.name: sha(p) for p in sorted(output.iterdir())}
    manifest = {"version": comparison.VERSION, "source_path": str(path),
                "source_sha256": candidate_sha256, "through_index": through_index,
                "source_content_hash": candidate.content_hash(value),
                "counts": comparison.counts(session), "code_hashes": code_hashes(),
                "files": files, "receipt": session.receipt(),
                "calls_external_executed": 0, "semantic_evaluation": "pending",
                "training_allowed": False, "product_supported": False}
    write(output / MANIFEST, manifest)
    require(sha(path) == candidate_sha256, "source changed during materialization")
    return manifest


def validate(output):
    output = Path(output)
    manifest = json.loads((output / MANIFEST).read_bytes())
    require(manifest["version"] == comparison.VERSION, "experiment version differs")
    require(manifest["code_hashes"] == code_hashes(), "producer changed")
    require(set(p.name for p in output.iterdir()) == set(manifest["files"]) | {MANIFEST}, "package inventory differs")
    require(all(sha(output / name) == h for name, h in manifest["files"].items()), "artifact bytes differ")
    value = json.loads((output / "candidate_evidence.json").read_bytes())
    require(candidate.content_hash(value) == manifest["source_content_hash"], "source content differs")
    source = Path(manifest["source_path"])
    require(sha(source) == manifest["source_sha256"], "original source differs")
    require(json.loads(source.read_bytes()) == value, "source copy differs")
    session = comparison.Session(value, through_index=manifest["through_index"])
    expected = {"model_input.json": session.consultation().initial(),
                "causal_samples.json": session._archive, "private_citations.json": session._private,
                "semantic_prompt.json": {"version": comparison.VERSION, "prompt": comparison.PROMPT},
                DATA: comparison.compare_local(session)}
    require(all(json.loads((output / name).read_bytes()) == v for name, v in expected.items()), "rebuilt comparison differs")
    require(manifest["counts"] == comparison.counts(session) and manifest["receipt"] == session.receipt(), "receipt differs")
    require(manifest["calls_external_executed"] == 0 and manifest["semantic_evaluation"] == "pending", "unexecuted semantic evaluation asserted")
    require(manifest["training_allowed"] is False and manifest["product_supported"] is False, "experimental gate changed")
    return manifest
