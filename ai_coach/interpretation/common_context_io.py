"""Materialize and validate the CS2 factual interface in isolated TEMP."""

import argparse
import hashlib
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import (
    candidate_context_io,
    common_context,
)
from ai_coach.support.anchors import require

ROOT = Path(__file__).resolve().parents[2]
DATA = "common_context.json"
EVIDENCE = "candidate_evidence.json"
NATIVE = "native_evidence.json"
REPORT = "common_context_report.md"
MANIFEST = "common_context_manifest.json"
PRODUCER_FILES = (
    *candidate_context_io.PRODUCER_FILES,
    "ai_coach/interpretation/common_context.py",
    "ai_coach/interpretation/common_context_io.py",
    "ai_coach/interpretation/native_place_context.py",
    "ai_coach/contracts/native_place_context.schema.json",
    "ai_coach/contracts/common_factual_context.schema.json",
)


def _sha(raw):
    return hashlib.sha256(raw).hexdigest()


def _bytes(value):
    return (
        json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2, allow_nan=False)
        + "\n"
    ).encode("utf-8")


def code_hashes():
    return {name: _sha((ROOT / name).read_bytes()) for name in PRODUCER_FILES}


def _read(path, expected):
    raw = Path(path).read_bytes()
    require(
        isinstance(expected, str) and _sha(raw) == expected,
        "common source byte hash differs",
    )
    return json.loads(raw), raw


def render(value):
    """A compact factual overview with references; no tactical summary is invented."""
    cut = value["cut"]
    lines = [
        "# Entrada factual común",
        "",
        f"Ronda {cut['round_number']}, tick {cut['tick']}, instrumento {value['source']['instrument']}.",
        "",
        "Situación y límites acreditados al corte. Las posiciones de eventos son históricas y no prueban percepción.",
        "",
        "| Bloque | Sujeto | Dato | Estado | Valor | Referencia |",
        "|---|---|---|---|---|---|",
    ]
    for group in value["situation"].values():
        for f in group:
            if f["key"] in (
                "identity",
                "exclusion",
                "inventory",
                "sampled_displacement",
                "cut",
                "perspective",
            ):
                continue
            payload = json.dumps(f["value"], ensure_ascii=False, separators=(",", ":"))
            # No source data are truncated in the machine input or detail reader.
            payload = payload.replace("|", "\\|").replace("\n", " ")
            lines.append(
                f"| {f['group']} | {f['subject'] or 'equipo'} | {f['key']} | {f['status']} | {payload} | `{f['evidence'][0]['pointer']}` |"
            )
    lines.extend(
        [
            "",
            f"Historial: {value['history']['sample_count']} muestras, {len(value['evolution'])} transiciones y {len(value['events'])} eventos.",
            f"Incertidumbres locales: {len(value['uncertainty'])}. Suplementos posteriores: {len(value['later_supplements'])}.",
            "",
            "El JSON conserva todos los recursos e incertidumbres. sample() consulta historia hasta el corte; evidence() resuelve sólo referencias citadas.",
            "El historial entre rondas no está suministrado. El nombre nativo posterior se mantiene separado del instrumento original.",
            "",
        ]
    )
    return "\n".join(lines).encode("utf-8")


def materialize(
    output,
    *,
    candidate_path,
    candidate_sha256,
    through_index=None,
    native_path=None,
    native_sha256=None,
):
    output = validate_output_directory(Path(output))  # before any source read
    require(
        (native_path is None) == (native_sha256 is None),
        "native path/hash pair incomplete",
    )
    code = code_hashes()
    value, raw = _read(candidate_path, candidate_sha256)
    # Byte receipts must correspond to the embedded validated inputs as well.
    for name, receipt in value["input_files"].items():
        supplied, _ = _read(receipt["path"], receipt["sha256"])
        expected = value["packet"] if name == "packet" else value["supplements"][name]
        require(supplied == expected, "candidate evidence source differs")
    native, native_raw = (
        (None, None) if native_path is None else _read(native_path, native_sha256)
    )
    result = common_context.build(value, through_index=through_index, native=native)
    bodies = {DATA: _bytes(result), EVIDENCE: raw, REPORT: render(result)}
    if native_raw is not None:
        bodies[NATIVE] = native_raw
    require(code_hashes() == code, "common producer changed during materialization")
    manifest = {
        "schema_id": "stratai.common_factual_manifest@1",
        "common_context_id": result["common_context_id"],
        "producer_sha256": code,
        "input_files": {
            "candidate": {
                "path": str(Path(candidate_path).resolve()),
                "sha256": candidate_sha256,
            }
        },
        "through_index": result["through_index"],
        "artifacts": {
            name: {"sha256": _sha(body), "bytes": len(body)}
            for name, body in bodies.items()
        },
        "training_allowed": False,
        "product_supported": False,
        "predictions_created": 0,
    }
    if native_path is not None:
        manifest["input_files"]["native"] = {
            "path": str(Path(native_path).resolve()),
            "sha256": native_sha256,
        }
    bodies[MANIFEST] = _bytes(manifest)
    for name, body in bodies.items():
        with (output / name).open("xb") as f:
            f.write(body)
    return manifest


def validate(output):
    output = Path(output)
    require(
        (output / MANIFEST).is_file() and not (output / MANIFEST).is_symlink(),
        "common manifest is not regular",
    )
    saved = json.loads((output / MANIFEST).read_bytes())
    require(
        set(saved)
        == {
            "schema_id",
            "common_context_id",
            "producer_sha256",
            "input_files",
            "through_index",
            "artifacts",
            "training_allowed",
            "product_supported",
            "predictions_created",
        },
        "common manifest fields differ",
    )
    require(
        saved["schema_id"] == "stratai.common_factual_manifest@1",
        "common manifest version differs",
    )
    require(
        saved["training_allowed"] is False
        and saved["product_supported"] is False
        and type(saved["predictions_created"]) is int
        and saved["predictions_created"] == 0,
        "common capability boundary differs",
    )
    require(saved["producer_sha256"] == code_hashes(), "common producer hash differs")
    require(
        set(saved["input_files"]) in ({"candidate"}, {"candidate", "native"}),
        "common source list differs",
    )
    for receipt in saved["input_files"].values():
        require(
            set(receipt) == {"path", "sha256"}, "common source receipt fields differ"
        )
    sources = {
        k: _read(**{"path": r["path"], "expected": r["sha256"]})
        for k, r in saved["input_files"].items()
    }
    value, raw = sources["candidate"]
    for name, receipt in value["input_files"].items():
        supplied, _ = _read(receipt["path"], receipt["sha256"])
        require(
            supplied
            == (value["packet"] if name == "packet" else value["supplements"][name]),
            "candidate evidence source differs",
        )
    native, native_raw = sources.get("native", (None, None))
    result = common_context.build(
        value, through_index=saved["through_index"], native=native
    )
    require(
        saved["common_context_id"] == result["common_context_id"],
        "common identity differs",
    )
    expected = {DATA: _bytes(result), EVIDENCE: raw, REPORT: render(result)}
    if native_raw is not None:
        expected[NATIVE] = native_raw
    require(set(saved["artifacts"]) == set(expected), "common artifact list differs")
    require(
        {p.name for p in output.iterdir()} == set(expected) | {MANIFEST},
        "common inventory differs",
    )
    for name, body in expected.items():
        path = output / name
        require(
            path.is_file() and not path.is_symlink(), "common artifact is not regular"
        )
        require(
            path.read_bytes() == body
            and saved["artifacts"][name] == {"sha256": _sha(body), "bytes": len(body)},
            "common artifact facts or hash differ",
        )
    return saved


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("materialize", "validate"))
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--candidate-path", type=Path)
    parser.add_argument("--candidate-sha256")
    parser.add_argument("--through-index", type=int)
    parser.add_argument("--native-path", type=Path)
    parser.add_argument("--native-sha256")
    args = parser.parse_args()
    if args.action == "validate":
        require(
            args.candidate_path is None
            and args.candidate_sha256 is None
            and args.through_index is None
            and args.native_path is None
            and args.native_sha256 is None,
            "validation uses retained source arguments only",
        )
        result = validate(args.output)
    else:
        require(
            args.candidate_path is not None and args.candidate_sha256 is not None,
            "candidate path/hash required",
        )
        result = materialize(
            args.output,
            candidate_path=args.candidate_path,
            candidate_sha256=args.candidate_sha256,
            through_index=args.through_index,
            native_path=args.native_path,
            native_sha256=args.native_sha256,
        )
    print(
        json.dumps(
            {
                "ok": True,
                "schema_id": result["schema_id"],
                "common_context_id": result["common_context_id"],
            }
        )
    )


if __name__ == "__main__":
    main()
