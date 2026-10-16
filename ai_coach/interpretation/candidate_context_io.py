"""Materialize auditable factual inputs in isolated TEMP, without predictions."""

import argparse
import hashlib
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import candidate_context
from ai_coach.support.anchors import require

DATA = "candidate_context.json"
REPORT = "candidate_context_report.md"
MANIFEST = "candidate_context_manifest.json"
SCHEMA_ID = "stratai.candidate_context_manifest@1"
ARGUMENTS = {
    "packet_path",
    "packet_sha256",
    "case_id",
    "instrument",
    "kills_path",
    "kills_sha256",
    "extra_path",
    "extra_sha256",
    "bomb_path",
    "bomb_sha256",
}
ROOT = Path(__file__).resolve().parents[2]
PRODUCER_FILES = (
    "ai_coach/interpretation/candidate_context_io.py",
    "ai_coach/interpretation/candidate_context.py",
    "ai_coach/interpretation/context.py",
    "ai_coach/interpretation/temporal.py",
    "ai_coach/support/anchors.py",
    "ai_coach/economy/candidate_io.py",
    "ai_coach/contracts/accelerated_support_schema.py",
    "ai_coach/contracts/candidate_context.schema.json",
    "ai_coach/contracts/contextual_input.schema.json",
    "ai_coach/contracts/contextual_review_packet_v3.schema.json",
    "ai_coach/contracts/review_kill_context.schema.json",
    "ai_coach/contracts/review_extra_context.schema.json",
    "ai_coach/contracts/review_bomb_context.schema.json",
)


def _bytes(value):
    return (
        json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2).encode("utf-8")
        + b"\n"
    )


def _sha(raw):
    return hashlib.sha256(raw).hexdigest()


def _code():
    return {name: _sha((ROOT / name).read_bytes()) for name in PRODUCER_FILES}


def _arguments(arguments):
    require(set(arguments) <= ARGUMENTS, "unrecognized candidate input argument")
    result = {}
    for key, value in arguments.items():
        if value is None:
            continue
        require(
            isinstance(value, (str, Path)), "candidate input argument must be text/path"
        )
        result[key] = (
            str(Path(value).resolve()) if key.endswith("_path") else str(value)
        )
    return result


def render(value):
    """Diagnostic artifact only; event positions are never called current contacts."""
    return "\n".join(
        [
            "# Contexto factual para interpretación",
            "",
            (
                "Datos de la ficha en su corte, con incertidumbre conservada. "
                "Este artefacto no emite interpretación táctica ni recomendaciones."
            ),
            "",
            f"Caso: {value['case_id']}. Instrumento: {value['instrument']}.",
            "",
            f"Identificador: `{value['candidate_context_id']}`.",
            "",
            "Estado factual al corte:",
            "",
            "```json",
            json.dumps(value["snapshot"], ensure_ascii=False, sort_keys=True, indent=2),
            "```",
            "",
            (
                "Los eventos fechados y sus edades están en candidate_context.json. "
                "Las posiciones rivales de eventos no acreditan ubicación actual, "
                "percepción individual o comunicación."
            ),
            "",
        ]
    )


def materialize(output, **load_arguments):
    # This guard must run before source reads or expensive work.
    output = validate_output_directory(Path(output))
    arguments = _arguments(load_arguments)
    code = _code()
    # load validates hash-bound inputs, derives fields and validates its result.
    value = candidate_context.load(**arguments)
    bodies = {DATA: _bytes(value), REPORT: render(value).encode("utf-8")}
    require(_code() == code, "candidate producer changed during materialization")
    for name, raw in bodies.items():
        with (output / name).open("xb") as stream:
            stream.write(raw)
    manifest = {
        "schema_id": SCHEMA_ID,
        "candidate_context_id": value["candidate_context_id"],
        "input_arguments": arguments,
        "producer_sha256": code,
        "artifacts": {
            name: {"sha256": _sha(raw), "bytes": len(raw)}
            for name, raw in bodies.items()
        },
        "training_allowed": False,
        "product_supported": False,
        "predictions_created": 0,
    }
    with (output / MANIFEST).open("xb") as stream:
        stream.write(_bytes(manifest))
    return manifest


def validate(output):
    output = Path(output)
    require(
        {p.name for p in output.iterdir()} == {DATA, REPORT, MANIFEST},
        "candidate output inventory differs",
    )
    for name in (DATA, REPORT, MANIFEST):
        require(
            (output / name).is_file() and not (output / name).is_symlink(),
            "candidate artifact is not a regular file",
        )
    saved = json.loads((output / MANIFEST).read_bytes())
    require(
        set(saved)
        == {
            "schema_id",
            "candidate_context_id",
            "input_arguments",
            "producer_sha256",
            "artifacts",
            "training_allowed",
            "product_supported",
            "predictions_created",
        },
        "candidate manifest fields differ",
    )
    require(saved["schema_id"] == SCHEMA_ID, "candidate manifest version differs")
    require(
        saved["training_allowed"] is False
        and saved["product_supported"] is False
        and type(saved["predictions_created"]) is int
        and saved["predictions_created"] == 0,
        "candidate capability gate differs",
    )
    require(saved["producer_sha256"] == _code(), "candidate producer hash differs")
    arguments = _arguments(saved["input_arguments"])
    require(arguments == saved["input_arguments"], "candidate input arguments differ")
    # Rebuild from verified sources; load already returns a validated value.
    expected = candidate_context.load(**arguments)
    require(
        saved["candidate_context_id"] == expected["candidate_context_id"],
        "candidate context identity differs",
    )
    bodies = {DATA: _bytes(expected), REPORT: render(expected).encode("utf-8")}
    require(set(saved["artifacts"]) == set(bodies), "candidate artifact list differs")
    for name, expected_raw in bodies.items():
        raw = (output / name).read_bytes()
        require(raw == expected_raw, f"{name}: facts differ from verified sources")
        require(
            saved["artifacts"][name] == {"sha256": _sha(raw), "bytes": len(raw)},
            f"{name}: artifact checksum differs",
        )
    return saved


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("materialize", "validate"))
    parser.add_argument("--output", type=Path, required=True)
    for name in sorted(ARGUMENTS):
        parser.add_argument("--" + name.replace("_", "-"))
    args = parser.parse_args()
    if args.action == "validate":
        require(
            not any(getattr(args, key) for key in ARGUMENTS),
            "validation uses only retained input arguments",
        )
        result = validate(args.output)
    else:
        for key in (
            "packet_path",
            "packet_sha256",
            "case_id",
            "instrument",
            "kills_path",
            "kills_sha256",
        ):
            require(bool(getattr(args, key)), f"missing candidate argument {key}")
        result = materialize(
            args.output, **{key: getattr(args, key) for key in ARGUMENTS}
        )
    print(
        json.dumps(
            {
                "ok": True,
                "schema_id": result["schema_id"],
                "candidate_context_id": result["candidate_context_id"],
            }
        )
    )


if __name__ == "__main__":
    main()
