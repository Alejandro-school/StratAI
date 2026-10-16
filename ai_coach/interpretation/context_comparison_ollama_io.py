"""Retain isolated local-model attempts without modifying factual packages."""

import argparse
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import candidate_context as candidate
from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation import context_comparison_io as factual_io
from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.support.anchors import require

ROOT = Path(__file__).resolve().parents[2]
DATA = "local_result.json"
MANIFEST = "local_manifest.json"


def code_hashes():
    return {**factual_io.code_hashes(), **{name: factual_io.sha(ROOT / name) for name in (
        "ai_coach/interpretation/context_comparison_ollama.py",
        "ai_coach/interpretation/context_comparison_ollama_io.py")}}


def load_session(package, expected_sha256):
    """Equivalent factual reader, returning its one rebuilt Session for reuse."""
    package = Path(package)
    require(factual_io.sha(package / factual_io.MANIFEST) == expected_sha256, "sealed factual manifest differs")
    manifest = json.loads((package / factual_io.MANIFEST).read_bytes())
    require(manifest["code_hashes"] == factual_io.code_hashes(), "factual producer changed")
    require(set(p.name for p in package.iterdir()) == set(manifest["files"]) | {factual_io.MANIFEST}, "factual inventory differs")
    require(all(factual_io.sha(package / name) == digest for name, digest in manifest["files"].items()), "factual artifact bytes differ")
    source = Path(manifest["source_path"])
    require(factual_io.sha(source) == manifest["source_sha256"], "factual original source differs")
    value = json.loads((package / "candidate_evidence.json").read_bytes())
    require(json.loads(source.read_bytes()) == value and candidate.content_hash(value) == manifest["source_content_hash"], "factual source copy differs")
    session = comparison.Session(value, through_index=manifest["through_index"])
    expected = {"model_input.json": session.consultation().initial(), "causal_samples.json": session._archive,
                "private_citations.json": session._private,
                "semantic_prompt.json": {"version": comparison.VERSION, "prompt": comparison.PROMPT},
                factual_io.DATA: comparison.compare_local(session)}
    require(all(json.loads((package / name).read_bytes()) == result for name, result in expected.items()), "factual reconstruction differs")
    require(manifest["version"] == comparison.VERSION and manifest["receipt"] == session.receipt() and manifest["counts"] == comparison.counts(session), "factual receipt differs")
    require(manifest["calls_external_executed"] == 0 and manifest["semantic_evaluation"] == "pending", "factual evaluation gate changed")
    require(manifest["training_allowed"] is manifest["product_supported"] is False, "factual product gate changed")
    return session, manifest


def materialize(output, *, package, package_sha256, client=None):
    output = validate_output_directory(output)
    session, factual = load_session(package, package_sha256)
    source_hashes = code_hashes()
    factual_io.write(output / "preflight.json", {"package": str(Path(package)), "package_sha256": package_sha256,
                                              "code_hashes": source_hashes, "receipt": session.receipt()})
    ordinal = 0

    def record(entry):
        nonlocal ordinal
        factual_io.write(output / f"call-{ordinal:03d}-{entry['kind']}.json", entry)
        ordinal += 1

    result = local.interpret(session, client or local.LocalClient(), record=record)
    audit = local.verify_replay(session, result)
    factual_io.write(output / DATA, result)
    factual_io.write(output / "replay.json", audit)
    require(source_hashes == code_hashes() and factual_io.sha(Path(package) / factual_io.MANIFEST) == package_sha256, "producer/source changed during local inference")
    manifest = {"version": local.VERSION, "package": str(Path(package)), "package_sha256": package_sha256,
                "input_receipt": session.receipt(), "code_hashes": source_hashes,
                "files": {p.name: factual_io.sha(p) for p in sorted(output.iterdir())},
                "factual_counts": factual["counts"], "status": result["status"], "local_calls": result["local_calls"],
                "semantic_truth": "not_evaluated", "tactical_quality": "not_evaluated",
                "provider_cost_usd": 0, "external_calls": 0, "training_allowed": False, "product_supported": False}
    factual_io.write(output / MANIFEST, manifest)
    return manifest


def validate(output):
    output = Path(output)
    manifest = json.loads((output / MANIFEST).read_bytes())
    require(manifest["version"] == local.VERSION and manifest["code_hashes"] == code_hashes(), "local producer changed")
    require(set(p.name for p in output.iterdir()) == set(manifest["files"]) | {MANIFEST}, "local inventory differs")
    require(all(factual_io.sha(output / name) == digest for name, digest in manifest["files"].items()), "local artifact bytes differ")
    session, factual = load_session(manifest["package"], manifest["package_sha256"])
    result = json.loads((output / DATA).read_bytes())
    require(local.verify_replay(session, result) == json.loads((output / "replay.json").read_bytes()), "local replay differs")
    require(manifest["input_receipt"] == session.receipt() and manifest["factual_counts"] == factual["counts"], "local input differs")
    require(manifest["status"] == result["status"] and manifest["local_calls"] == result["local_calls"], "local outcome differs")
    require(manifest["external_calls"] == manifest["provider_cost_usd"] == 0 and manifest["training_allowed"] is manifest["product_supported"] is False, "local gate differs")
    return manifest


def main():
    parser = argparse.ArgumentParser(description="CS3 isolated local reading; never training or product coaching")
    parser.add_argument("--package", required=True)
    parser.add_argument("--package-sha256", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    result = materialize(args.output, package=args.package, package_sha256=args.package_sha256)
    print(json.dumps({"status": result["status"], "local_calls": result["local_calls"], "external_calls": 0, "provider_cost_usd": 0}))


if __name__ == "__main__":
    main()
