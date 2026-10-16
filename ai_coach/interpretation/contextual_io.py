"""Offline contextual replay pipeline. Retained TEMP outputs, no promotion."""

import argparse
import json
from collections import Counter, deque
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import (
    coaching,
    context,
    detector,
    ledger_io,
    spatial_reference,
    temporal,
    temporal_detector,
)
from ai_coach.support import physical
from ai_coach.support.anchors import require

ARTIFACTS = (
    "contexts.jsonl",
    "interpretations.jsonl",
    "coaching_readiness.jsonl",
    "contextual_report.md",
)
MANIFEST = "contextual_manifest.json"
DETECTORS = {module.CONFIGURATION: module for module in (detector, temporal_detector)}

REASONS = {
    "fresh_carrier_approach_required_after_review": "tras la revisión falta una nueva aproximación del portador",
    "sustained_carrier_retreat": "el portador se aleja de forma continuada",
    "sustained_ally_retreat": "un aliado se aleja de forma continuada",
    "sustained_divergent_ally_movement": "un aliado mantiene una dirección hacia otra referencia",
    "carrier_approach_fully_reversed": "el portador ha deshecho la aproximación que respaldaba la lectura",
    "c4_carrier_changed_or_unavailable": "ha cambiado el portador de C4 o su tenencia ya no está verificada",
    "ally_identity_changed": "ha cambiado la identidad o vida de un aliado",
    "fewer_than_two_supporting_survivors": "quedan menos de dos supervivientes que respaldaban esta lectura",
    "fewer_than_two_traceable_survivors": "no hay varios supervivientes con trayectoria verificada",
    "temporal_continuity_lost": "se ha interrumpido la secuencia observada",
    "current_ally_information_unavailable": "faltan datos actuales de un aliado; no se considera muerto por esa ausencia",
    "carrier_has_no_joint_approach_evidence": "falta aproximación conjunta respaldada del portador de C4",
    "no_second_ally_approaching_same_reference": "falta otro aliado aproximándose a la misma referencia",
    "c4_trajectory_not_continuously_observed": "la tenencia de C4 no está verificada durante el tramo utilizado",
}


def check_intervals(intervals):
    require(0 < len(intervals) <= 1000, "select 1 to 1000 explicit intervals")
    previous = 0
    for start, end in intervals:
        require(
            type(start) is int
            and type(end) is int
            and 0 < start <= end
            and end > previous,
            "intervals require positive endpoints and increasing unique end cuts",
        )
        require(end - start <= 4096, "context interval exceeds 4097 samples")
        previous = end


def episodes(support, sources, *, side, intervals):
    check_intervals(intervals)
    bounds = iter(intervals)
    start, end = next(bounds)
    history = deque(maxlen=max(b - a + 1 for a, b in intervals))
    for row in temporal.iter_records(
        support, side=side, through_sequence=intervals[-1][1], **sources
    ):
        history.append(row)
        if row["cut"]["sequence"] == end:
            yield context.interval(history, start_sequence=start, end_sequence=end)
            next_bounds = next(bounds, None)
            if next_bounds is None:
                break
            start, end = next_bounds


def load_inputs(source, map_vmap):
    support, sources, refs, _ = ledger_io.load_sources(source)
    base_path = source / support["base_manifest"]["path"]
    raw = base_path.read_bytes()
    require(
        physical.file_reference(base_path)["sha256"]
        == support["base_manifest"]["sha256"],
        "base manifest changed after validation",
    )
    map_name = json.loads(raw)["lineage"]["map_name"]
    reference = (
        spatial_reference.extract_vmap(map_vmap, map_name=map_name)
        if map_vmap
        else None
    )
    # Explicit asset naming prevents accepting Mirage coordinates for another map.
    if map_vmap:
        require(
            Path(map_vmap).name == map_name + "_texto.vmap", "map asset name mismatch"
        )
    return support, sources, refs, map_name, reference


def render(contexts, readings):
    lines = [
        "# Interpretación de la replay",
        "",
        "Prototipo provisional. Perspectiva analítica de estados propios y aliados.",
        "Las referencias del mapa local no acreditan zona ocupada, ruta ni compatibilidad con la build.",
        "",
    ]
    for value, reading in zip(contexts, readings):
        cut = reading["cut"]
        lines.extend(
            [
                f"## Ronda {cut['round_number']} · tick {cut['tick']} · lado {reading['side']}",
                "",
                (
                    f"Secuencias {reading['interval']['start_sequence']}–{reading['interval']['end_sequence']}: "
                    f"{len(value['samples'])} muestras conservadas. Lectura: {reading['status']}."
                ),
                "",
                f"Actualización: {reading['update']['kind']}.",
                "",
            ]
        )
        if not reading["hypotheses"]:
            lines += [
                (
                    "Información insuficiente: "
                    + ", ".join(reading["abstention_reasons"])
                    if reading["configuration"] == detector.CONFIGURATION
                    else "No se sostiene una preparación actual: "
                    + ", ".join(
                        REASONS.get(r, r) for r in reading["abstention_reasons"]
                    )
                )
                + ".",
                "",
            ]
        for h in reading["hypotheses"]:
            lines += [h["statement"], "", "Señales:", ""]
            for s in h["signals"]:
                alias = s["actor_alias"]
                label = alias.removeprefix("support-actor-")
                if s["kind"] == "sampled_approach_to_landmark":
                    movement = s["value"]
                    text = (
                        f"{label}: distancia horizontal a la referencia "
                        f"{movement['start_distance']:.1f} → {movement['end_distance']:.1f} unidades, "
                        f"en {movement['sample_count']} muestras."
                    )
                elif s["kind"] == "current_c4_possession":
                    text = f"{label}: tenencia C4 observada en el corte actual."
                elif s["kind"] == "temporal_possibility_maintained":
                    facts = s["value"]
                    text = (
                        f"{label}: posibilidad conservada desde la muestra {facts['seed_sequence']}; "
                        f"alejamiento acumulado sin avance intermedio: {facts['away_seconds']:.3f} s. "
                        "Una pausa no añade tiempo contrario ni confirma ejecución."
                    )
                else:
                    text = f"{label}: detrás del portador respecto a la referencia, con utilidad equipada y sin desplazamiento horizontal resuelto."
                lines.append("- " + text)
            lines += ["", "Funciones posibles:", ""]
            for role in h["possible_functions"]:
                function = (
                    "portador de C4 observado"
                    if role["function"] == "c4_carrier"
                    else "posible preparación de utilidad"
                )
                lines.append(
                    f"- {role['actor_alias'].removeprefix('support-actor-')}: {function}."
                )
            lines += [""]
        for event in reading.get("temporal_events", []):
            lines += [
                f"Revisión en muestra {event['sequence']}: "
                + REASONS.get(event["reason"], event["reason"])
                + ".",
                "",
            ]
        lines += [
            "No se acredita intención, lanzamiento, cobertura efectiva ni una alternativa mejor.",
            "Recomendaciones y misiones: abstención por falta de alternativas viables y comparación respaldada.",
            f"Referencia causal: `{value['context_id']}`.",
            "",
        ]
    return "\n".join(lines)


def compute(
    support,
    sources,
    *,
    side,
    intervals,
    map_name,
    map_reference,
    configuration=temporal_detector.CONFIGURATION,
):
    require(configuration in DETECTORS, "unknown detector configuration")
    implementation = DETECTORS[configuration]
    previous = None
    for value in episodes(support, sources, side=side, intervals=intervals):
        reading = implementation.interpret(
            value, map_name=map_name, map_reference=map_reference, previous=previous
        )
        yield value, reading, coaching.assess_readiness(reading)
        previous = reading


def materialize(
    source,
    output,
    *,
    side,
    intervals,
    map_vmap=None,
    configuration=temporal_detector.CONFIGURATION,
):
    output = validate_output_directory(output)
    check_intervals(intervals)
    require(configuration in DETECTORS, "unknown detector configuration")
    support, sources, refs, map_name, reference = load_inputs(source, map_vmap)
    all_contexts, all_readings, counts = [], [], Counter()
    streams = []
    try:
        streams = [
            (output / name).open("x", encoding="utf-8", newline="\n")
            for name in ARTIFACTS[:3]
        ]
        for value, reading, readiness in compute(
            support,
            sources,
            side=side,
            intervals=intervals,
            map_name=map_name,
            map_reference=reference,
            configuration=configuration,
        ):
            for stream, row in zip(streams, (value, reading, readiness)):
                stream.write(
                    json.dumps(row, sort_keys=True, ensure_ascii=False, allow_nan=False)
                    + "\n"
                )
            counts["episodes"] += 1
            counts["samples_in_episodes"] += len(value["samples"])
            counts[reading["status"]] += 1
            counts["hypotheses"] += len(reading["hypotheses"])
            counts["coaching_abstentions"] += 1
            counts.update("abstention:" + x for x in reading["abstention_reasons"])
            # Retain only report inputs; the caller explicitly bounds episode count/size.
            all_contexts.append(value)
            all_readings.append(reading)
    finally:
        for stream in streams:
            stream.close()
    with (output / ARTIFACTS[-1]).open("x", encoding="utf-8", newline="\n") as stream:
        stream.write(render(all_contexts, all_readings))
    manifest = {
        "schema_id": "stratai.contextual_manifest@1",
        "configuration": configuration,
        "source_alias": support["source_alias"],
        "source_sha256": support["source_sha256"],
        "source_artifacts": refs,
        "base_manifest": support["base_manifest"],
        "map_reference": reference,
        "map_name": map_name,
        "side": side,
        "intervals": [list(pair) for pair in intervals],
        "counts": dict(sorted(counts.items())),
        "artifacts": [physical.file_reference(output / name) for name in ARTIFACTS],
        "training_allowed": False,
        "product_supported": False,
    }
    with (output / MANIFEST).open("x", encoding="utf-8") as stream:
        json.dump(manifest, stream, indent=2, ensure_ascii=False)
    return manifest


def validate(source, output, *, map_vmap=None):
    """Recompute from original producer inputs; a rehashed output alone is insufficient."""
    manifest = json.loads((output / MANIFEST).read_bytes())
    support, sources, refs, map_name, reference = load_inputs(source, map_vmap)
    require(
        manifest["configuration"] in DETECTORS
        and manifest["source_artifacts"] == refs
        and manifest["base_manifest"] == support["base_manifest"]
        and manifest["source_alias"] == support["source_alias"]
        and manifest["source_sha256"] == support["source_sha256"]
        and manifest["map_name"] == map_name
        and manifest["map_reference"] == reference,
        "contextual inputs/configuration changed",
    )
    require(
        manifest["training_allowed"] is False
        and manifest["product_supported"] is False,
        "invalid release boundary",
    )
    require(
        manifest["artifacts"]
        == [physical.file_reference(output / name) for name in ARTIFACTS],
        "contextual artifact changed",
    )
    counts, values, readings = Counter(), [], []
    streams = [(output / name).open(encoding="utf-8") for name in ARTIFACTS[:3]]
    try:
        for value, reading, readiness in compute(
            support,
            sources,
            side=manifest["side"],
            intervals=manifest["intervals"],
            map_name=map_name,
            map_reference=reference,
            configuration=manifest["configuration"],
        ):
            for stream, row in zip(streams, (value, reading, readiness)):
                require(
                    json.loads(next(stream, "null")) == row,
                    "contextual recomputation differs",
                )
            values.append(value)
            readings.append(reading)
            counts.update(
                {
                    "episodes": 1,
                    "samples_in_episodes": len(value["samples"]),
                    reading["status"]: 1,
                    "hypotheses": len(reading["hypotheses"]),
                    "coaching_abstentions": 1,
                }
            )
            counts.update("abstention:" + x for x in reading["abstention_reasons"])
        require(all(not stream.read() for stream in streams), "extra contextual rows")
    finally:
        for stream in streams:
            stream.close()
    require(manifest["counts"] == dict(counts), "contextual counts differ")
    require(
        (output / ARTIFACTS[-1]).read_text(encoding="utf-8")
        == render(values, readings),
        "contextual report differs",
    )
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("materialize", "validate"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--map-vmap", type=Path)
    parser.add_argument("--side", choices=("T", "CT"), default="T")
    parser.add_argument(
        "--configuration",
        choices=tuple(DETECTORS),
        default=temporal_detector.CONFIGURATION,
    )
    parser.add_argument(
        "--intervals", nargs="+", help="Explicit source sequences start:end"
    )
    args = parser.parse_args()
    if args.action == "materialize":
        require(bool(args.intervals), "explicit intervals required")
        intervals = [
            tuple(int(x) for x in value.split(":")) for value in args.intervals
        ]
        result = materialize(
            args.source,
            args.output,
            side=args.side,
            intervals=intervals,
            map_vmap=args.map_vmap,
            configuration=args.configuration,
        )
    else:
        result = validate(args.source, args.output, map_vmap=args.map_vmap)
    print(json.dumps({"ok": True, "counts": result["counts"]}, sort_keys=True))


if __name__ == "__main__":
    main()
