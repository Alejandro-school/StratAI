"""Offline Spanish presentation of verified own/ally replay cuts; no tactics."""

from __future__ import annotations

import argparse
import html
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import ledger_io, temporal
from ai_coach.support import anchors, physical

VERSION = "replay_description_es@1"
REPORT = "replay_report.md"
EVIDENCE = "replay_report_records.jsonl"
MANIFEST = "replay_report_manifest.json"


def select_records(support, sources, *, side, sequences):
    """Project a single side causally, preserving original records and IDs."""
    anchors.require(side in ("CT", "T"), "invalid report side")
    anchors.require(
        0 < len(sequences) <= 100
        and all(type(n) is int and n > 0 for n in sequences)
        and list(sequences) == sorted(set(sequences)),
        "report sequences must be positive, unique, ordered (maximum 100)",
    )
    selected = []
    for row in temporal.iter_records(
        support, side=side, through_sequence=sequences[-1], **sources
    ):
        if row["cut"]["sequence"] in sequences:
            temporal.validate_record(row)
            selected.append(row)
    anchors.require(len(selected) == len(sequences), "missing requested report cut")
    return selected


def cell(value):
    """Keep source strings inert inside Markdown tables."""
    return (
        html.escape(str(value), quote=True)
        .replace("|", "&#124;")
        .replace("\r", " ")
        .replace("\n", " ")
        .replace("`", "&#96;")
        .replace("[", "&#91;")
        .replace("]", "&#93;")
    )


def fact_text(fact):
    if fact["status"] != "observed":
        label = (
            "No aplicable" if fact["status"] == "not_applicable" else "No disponible"
        )
        return f"{label} ({cell(fact['reason'])})"
    value = fact["value"]
    if isinstance(value, bool):
        return "Sí" if value else "No"
    if isinstance(value, dict):
        return "; ".join(f"{cell(k)}={cell(v)}" for k, v in sorted(value.items()))
    return cell(value)


def render(rows):
    """Render only validated, ordered records from one match and perspective."""
    anchors.require(bool(rows), "empty report")
    first = rows[0]
    previous = -1
    for row in rows:
        temporal.validate_record(row)
        anchors.require(
            all(
                row[k] == first[k]
                for k in (
                    "source_alias",
                    "source_sha256",
                    "build_identifier",
                    "perspective",
                )
            )
            and row["cut"]["sequence"] > previous,
            "mixed or unordered report records",
        )
        previous = row["cut"]["sequence"]
    lines = [
        "# Descripción de la replay",
        "",
        f"Partida `{first['source_alias']}`. Lado analizado: **{first['perspective']['side']}**.",
        "",
        ("Hechos propios y aliados extraídos de la replay. Esta perspectiva analítica "
        "no demuestra qué vio, oyó o sabía cada jugador. Los alias identifican actores "
        "de esta replay; no equivalen a los nombres T1–T5 de las fichas anteriores."),
        "",
        ("Las posiciones y diferencias conservan las unidades del mapa y la precisión "
        "de la fuente. Cada diferencia corresponde a la muestra inmediatamente anterior "
        "del registro, aunque esa muestra no esté seleccionada en este informe. No "
        "representa el recorrido entre los cortes seleccionados ni velocidad instantánea."),
        "",
        ("Un valor cero o «No» es un dato observado. «No disponible» conserva la "
        "causa de abstención; «No aplicable» distingue recursos que no corresponden al arma."),
    ]
    for row in rows:
        cut = row["cut"]
        lines += [
            "",
            f"## Corte {cut['sequence']}",
            "",
            (f"Ronda {cut['round_number']}; instancia {cut['instance']}; época "
            f"{cut['epoch']}; tick {cut['tick']}; frame {cut['frame']}. "
            f"Paso de muestreo: {cut['sample_step']} ticks."),
            "",
            (f"Actores presentes en esta muestra: {len(row['actors'])}. "
            "Esta cifra no acredita una plantilla completa."),
            "",
            "| Actor | Vivo | Posición | Arma activa | Cargador | Reserva | C4 |",
            "|---|---|---|---|---|---|---|",
        ]
        for actor in row["actors"]:
            phys = actor["physical"]
            values = [
                actor["actor_alias"],
                fact_text(actor["alive"]),
                fact_text(actor["position"]),
                fact_text(phys["active_weapon"]),
                fact_text(phys["ammo_in_magazine"]),
                fact_text(phys["ammo_reserve"]),
                fact_text(actor["has_c4"]),
            ]
            lines.append("| " + " | ".join(values) + " |")
        lines += [
            "",
            ("| Actor | Diferencia frente a muestra anterior (x, y, z) | "
            "Caminando | Agachado | Recargando | Ángulos de vista |"),
            "|---|---|---|---|---|---|",
        ]
        for actor in row["actors"]:
            phys = actor["physical"]
            values = [actor["actor_alias"], fact_text(actor["sampled_displacement"])]
            values += [
                fact_text(phys[k])
                for k in ("walking", "ducking", "reloading", "view_angles")
            ]
            lines.append("| " + " | ".join(values) + " |")
        excluded = [a for a in row["actors"] if a["exclusion"]]
        for actor in excluded:
            lines += [
                "",
                f"Actor {actor['actor_alias']}: exclusión {cell(actor['exclusion'])}.",
            ]
        if row["missing_previous_actors"]:
            lines += [
                "",
                "Sin muestra actual: "
                + ", ".join(row["missing_previous_actors"])
                + ". La ausencia no demuestra muerte ni retirada.",
            ]
        lines += [
            "",
            (f"Registro de evidencia: `{row['record_id']}`. "
            f"Continuidad: `{row['continuity']}`."),
        ]
    lines += [
        "",
        "## Límites de la interpretación",
        "",
        ("No se acreditan contacto enemigo legal, información compartida, intención, "
        "plan táctico, zona semántica, ruta física, velocidad instantánea ni "
        "disponibilidad completa del arma. El spotting nativo y los iconos del radar "
        "no se convierten aquí en percepción atribuida."),
        "",
        ("E01–E04 permanecen sin resolver: defensa percibida, oportunidad en medio, "
        "fake y pérdida del jugador de información requieren los hechos de "
        "atribución y contexto que se enumeran en la evidencia. No se emiten "
        "recomendaciones ni se habilita entrenamiento o uso en producto."),
        "",
        (f"[Registros y referencias exactas]({EVIDENCE}) · "
        f"[Hashes y procedencia]({MANIFEST})"),
        "",
        f"SHA-256 de la demo: `{first['source_sha256']}`.",
        "",
    ]
    return "\n".join(lines)


def materialize(source, output, *, side, sequences):
    output = validate_output_directory(output)
    support, sources, refs, _ = ledger_io.load_sources(source)
    rows = select_records(support, sources, side=side, sequences=sequences)
    report = render(rows)
    with (output / REPORT).open("x", encoding="utf-8", newline="\n") as stream:
        stream.write(report)
    with (output / EVIDENCE).open("x", encoding="utf-8", newline="\n") as stream:
        for row in rows:
            stream.write(json.dumps(row, sort_keys=True, allow_nan=False) + "\n")
    manifest = {
        "presentation_version": VERSION,
        "temporal_contract": temporal.VERSION,
        "source_alias": support["source_alias"],
        "source_sha256": support["source_sha256"],
        "source_artifacts": refs,
        "side": side,
        "sequences": list(sequences),
        "records": len(rows),
        "actors": sum(len(r["actors"]) for r in rows),
        "hypotheses_unavailable": sum(len(r["hypotheses"]) for r in rows),
        "artifacts": [physical.file_reference(output / p) for p in (REPORT, EVIDENCE)],
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
    }
    with (output / MANIFEST).open("x", encoding="utf-8") as stream:
        json.dump(manifest, stream, indent=2, sort_keys=True)
        stream.write("\n")
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--side", choices=("CT", "T"), required=True)
    parser.add_argument("--sequences", nargs="+", type=int, required=True)
    args = parser.parse_args()
    print(
        json.dumps(
            materialize(
                args.source, args.output, side=args.side, sequences=args.sequences
            ),
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
