"""Retain an orientation candidate and its native enrichment in isolated TEMP."""

import argparse
import hashlib
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import destination_orientation, native_place_context
from ai_coach.support.anchors import require

ROOT = Path(__file__).resolve().parents[2]
DATA = "destination_orientation.json"
NATIVE = "native_place_context.json"
REPORT = "destination_orientation_report.md"
MANIFEST = "destination_orientation_manifest.json"
SCHEMA_ID = "stratai.destination_orientation_manifest@1"
PRODUCER_FILES = (
    "ai_coach/interpretation/destination_orientation_io.py",
    "ai_coach/interpretation/destination_orientation.py",
    "ai_coach/interpretation/native_place_context.py",
    "ai_coach/interpretation/candidate_context.py",
    "ai_coach/interpretation/context.py",
    "ai_coach/interpretation/temporal.py",
    "ai_coach/support/anchors.py",
    "ai_coach/economy/candidate_io.py",
    "ai_coach/contracts/accelerated_support_schema.py",
    "ai_coach/contracts/destination_orientation.schema.json",
    "ai_coach/contracts/native_place_context.schema.json",
    "ai_coach/contracts/candidate_context.schema.json",
    "ai_coach/contracts/contextual_input.schema.json",
    "ai_coach/contracts/contextual_review_packet_v3.schema.json",
    "ai_coach/contracts/review_kill_context.schema.json",
    "ai_coach/contracts/review_extra_context.schema.json",
    "ai_coach/contracts/review_bomb_context.schema.json",
)
ARGUMENTS = {
    "candidate_path",
    "candidate_sha256",
    "export_dir",
    "manifest_sha256",
    "targets_sha256",
    "target_id",
}


def _bytes(value):
    return (
        json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2, allow_nan=False)
        + "\n"
    ).encode("utf-8")


def _sha(raw):
    return hashlib.sha256(raw).hexdigest()


def _code():
    return {name: _sha((ROOT / name).read_bytes()) for name in PRODUCER_FILES}


def _arguments(arguments):
    require(set(arguments) <= ARGUMENTS, "unexpected orientation argument")
    result = {}
    for key, value in arguments.items():
        if value is None:
            continue
        require(
            isinstance(value, (str, Path)), "orientation argument must be path/text"
        )
        result[key] = (
            str(Path(value).resolve())
            if key in {"candidate_path", "export_dir"}
            else str(value)
        )
    require(
        {"candidate_path", "candidate_sha256"} <= set(result),
        "candidate input required",
    )
    native = {"export_dir", "manifest_sha256", "targets_sha256", "target_id"}
    require(
        not (native & set(result)) or native <= set(result),
        "incomplete native input binding",
    )
    return result


def _load(arguments):
    candidate = json.loads(
        native_place_context._read(
            arguments["candidate_path"], arguments["candidate_sha256"]
        )
    )
    if "export_dir" in arguments:
        native = native_place_context.load(
            candidate,
            export_dir=arguments["export_dir"],
            manifest_sha256=arguments["manifest_sha256"],
            targets_sha256=arguments["targets_sha256"],
            target_id=arguments["target_id"],
            candidate_context_sha256=arguments["candidate_sha256"],
        )
    else:
        native = native_place_context.unavailable(candidate)
    # infer validates the full unchanged candidate exactly once.
    return native, destination_orientation.infer(candidate, native)


def render(value):
    if value["status"] == "hypothesis":
        result = f"{value['preferred_site']} más probable; {value['alternatives'][0]['site']} sigue siendo posible, sin excluirla."
    elif value["status"] == "out_of_scope":
        result = "Fuera del alcance de esta hipótesis de orientación preplantado."
    else:
        result = (
            "Orientación indeterminada para la regla candidata y sus datos disponibles."
        )
    return "\n".join(
        [
            "# Hipótesis de orientación",
            "",
            result,
            "",
            "La preferencia es cualitativa. No afirma intención, ruta viable ni preparación de entrada.",
            "",
            "Motivo: " + ", ".join(value["reasons"]) + ".",
            "",
            "## Hechos utilizados",
            "",
            "```json",
            json.dumps(
                value["observed_facts"], ensure_ascii=False, sort_keys=True, indent=2
            ),
            "```",
            "",
            "El lugar nominal es la propiedad nativa leída en el corte. No certifica una frontera geométrica.",
            "La edad de su última entrega no es un umbral de caducidad.",
            "",
            "Las predicciones de preparación congeladas no se sustituyen ni se incorporan como entrada.",
            "",
        ]
    )


def _bodies(native, prediction):
    return {
        NATIVE: _bytes(native),
        DATA: _bytes(prediction),
        REPORT: render(prediction).encode("utf-8"),
    }


def materialize(output, **load_arguments):
    output = validate_output_directory(Path(output))
    arguments = _arguments(load_arguments)
    code = _code()
    native, prediction = _load(arguments)
    bodies = _bodies(native, prediction)
    require(_code() == code, "orientation producer changed during materialization")
    manifest = {
        "schema_id": SCHEMA_ID,
        "rule_id": destination_orientation.RULE_ID,
        "orientation_id": prediction["orientation_id"],
        "input_arguments": arguments,
        "producer_sha256": code,
        "artifacts": {
            name: {"sha256": _sha(raw), "bytes": len(raw)}
            for name, raw in bodies.items()
        },
        "information_delta": "native_place_property_added"
        if native["status"] == "observed"
        else "none_native_source_unavailable",
        "preparation_prediction_replaced": False,
        "training_allowed": False,
        "product_supported": False,
    }
    for name, raw in {**bodies, MANIFEST: _bytes(manifest)}.items():
        with (output / name).open("xb") as stream:
            stream.write(raw)
    return manifest


def validate(output):
    output = Path(output)
    require(
        {path.name for path in output.iterdir()} == {DATA, NATIVE, REPORT, MANIFEST},
        "orientation inventory differs",
    )
    for name in (DATA, NATIVE, REPORT, MANIFEST):
        require(
            (output / name).is_file() and not (output / name).is_symlink(),
            "orientation artifact not regular",
        )
    manifest = json.loads((output / MANIFEST).read_bytes())
    require(
        set(manifest)
        == {
            "schema_id",
            "rule_id",
            "orientation_id",
            "input_arguments",
            "producer_sha256",
            "artifacts",
            "information_delta",
            "preparation_prediction_replaced",
            "training_allowed",
            "product_supported",
        },
        "orientation manifest fields differ",
    )
    require(
        manifest["schema_id"] == SCHEMA_ID
        and manifest["rule_id"] == destination_orientation.RULE_ID,
        "orientation contract differs",
    )
    require(manifest["producer_sha256"] == _code(), "orientation producer hash differs")
    require(
        manifest["training_allowed"] is False
        and manifest["product_supported"] is False
        and manifest["preparation_prediction_replaced"] is False,
        "orientation scope gate differs",
    )
    arguments = _arguments(manifest["input_arguments"])
    require(arguments == manifest["input_arguments"], "orientation arguments differ")
    native, prediction = _load(arguments)
    require(
        manifest["orientation_id"] == prediction["orientation_id"],
        "orientation identity differs",
    )
    require(
        manifest["information_delta"]
        == (
            "native_place_property_added"
            if native["status"] == "observed"
            else "none_native_source_unavailable"
        ),
        "orientation information delta differs",
    )
    bodies = _bodies(native, prediction)
    require(
        set(manifest["artifacts"]) == set(bodies), "orientation artifact list differs"
    )
    for name, raw in bodies.items():
        require(
            (output / name).read_bytes() == raw,
            "orientation differs from source recomputation",
        )
        require(
            manifest["artifacts"][name] == {"sha256": _sha(raw), "bytes": len(raw)},
            "orientation artifact hash differs",
        )
    return manifest


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
            "validation uses retained bindings",
        )
        result = validate(args.output)
    else:
        result = materialize(
            args.output, **{key: getattr(args, key) for key in ARGUMENTS}
        )
    print(
        json.dumps(
            {
                "ok": True,
                "schema_id": result["schema_id"],
                "orientation_id": result["orientation_id"],
            }
        )
    )


if __name__ == "__main__":
    main()
