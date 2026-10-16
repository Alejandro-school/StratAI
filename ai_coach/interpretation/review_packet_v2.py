"""Versioned review cards with causal eligibility and native round clocks."""

import argparse
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import contextual_io, inventory, review_packet as v1
from ai_coach.interpretation import round_context as rounds
from ai_coach.interpretation import temporal_detector, spatial_reference
from ai_coach.support import anchors as a
from ai_coach.support import physical

TEMPLATE = Path(__file__).with_suffix(".html")


def build(value, support, sources, quantities, map_name, optional):
    packet = v1.build(value, sources, quantities, map_name)
    packet["schema_id"] = "stratai.contextual_review_packet@2"
    packet["round_context"] = rounds.sequence(value, support, sources, optional)
    states = {r["eligibility"]["kind"] for r in packet["round_context"]}
    packet["analysis_scope"] = (
        "excluded"
        if states <= {"knife", "warmup"}
        else "eligible"
        if states == {"bomb_objective"}
        else "pending_context"
    )
    a.schema_validator("contextual_review_packet_v2.json").validate(packet)
    return packet


def interpret(packet, *, map_reference=None):
    """The new entry point never sends excluded/unknown rounds to the detector."""
    a.schema_validator("contextual_review_packet_v2.json").validate(packet)
    a.require(
        packet["analysis_scope"] == "eligible"
        and all(
            r["eligibility"]["analysis_allowed"]
            and r["eligibility"]["kind"] == "bomb_objective"
            for r in packet["round_context"]
        ),
        "round not eligible for tactical analysis",
    )
    return temporal_detector.interpret(
        packet["context"], map_name=packet["map_name"], map_reference=map_reference
    )


def render(packet, radar=None):
    # v1 remains byte-for-byte reproducible, including the 36 frozen pilot cards.
    base = {
        k: v for k, v in packet.items() if k not in ("round_context", "analysis_scope")
    }
    base["schema_id"] = "stratai.contextual_review_packet@1"
    html = v1.render(base, radar)
    supplement = TEMPLATE.read_text(encoding="utf-8").replace(
        "__ROUND_JSON__",
        json.dumps(
            {"frames": packet["round_context"], "scope": packet["analysis_scope"]},
            ensure_ascii=False,
        ).replace("<", "\\u003c"),
    )
    return html.replace("</html>", supplement + "</html>")


def compute(source, *, side, intervals, radar=None):
    support, sources, refs, map_name, _ = contextual_io.load_inputs(source, None)
    quantities, ref = inventory.load_source(source, sources["c4_source"])
    if ref:
        refs[inventory.DATA] = ref
    optional, extra_refs = rounds.load_optional(source, support, sources)
    refs.update(extra_refs)
    return [
        build(value, support, sources, quantities, map_name, optional)
        for value in contextual_io.episodes(
            support, sources, side=side, intervals=intervals
        )
    ], refs


def manifest(side, intervals, refs, radar, artifacts, scopes):
    return {
        "schema_id": "stratai.contextual_review_packet_manifest@2",
        "side": side,
        "intervals": [list(pair) for pair in intervals],
        "source_artifacts": refs,
        "radar": physical.file_reference(radar) if radar else None,
        "templates": [physical.file_reference(p) for p in (v1.TEMPLATE, TEMPLATE)],
        "artifacts": artifacts,
        "analysis_scopes": scopes,
        "training_allowed": False,
    }


def analyze(source, output, *, side, intervals, map_vmap=None):
    """New analyses explicitly use the gate; the frozen legacy runner is unchanged."""
    output = validate_output_directory(output)
    packets, refs = compute(source, side=side, intervals=intervals)
    reference = (
        spatial_reference.extract_vmap(map_vmap, map_name=packets[0]["map_name"])
        if map_vmap
        else None
    )
    rows = []
    for p in packets:
        rows.append(
            {
                "context_id": p["context"]["context_id"],
                "scope": p["analysis_scope"],
                "interpretation": interpret(p, map_reference=reference)
                if p["analysis_scope"] == "eligible"
                else None,
            }
        )
    result = {
        "schema_id": "stratai.round_gated_analysis@1",
        "source_artifacts": refs,
        "detector_configuration": temporal_detector.CONFIGURATION,
        "map_reference": reference,
        "rows": rows,
        "training_allowed": False,
    }
    with (output / "round_gated_analysis.json").open("x", encoding="utf-8") as stream:
        json.dump(result, stream, ensure_ascii=False, indent=2)
    return result


def materialize(source, output, *, side, intervals, radar=None):
    output = validate_output_directory(output)
    packets, refs = compute(source, side=side, intervals=intervals, radar=radar)
    artifacts = []
    for i, packet in enumerate(packets, 1):
        for suffix, body in (
            ("json", json.dumps(packet, ensure_ascii=False, sort_keys=True)),
            ("html", render(packet, radar)),
        ):
            path = output / f"case_{i:03d}.{suffix}"
            with path.open("x", encoding="utf-8") as stream:
                stream.write(body)
            artifacts.append(physical.file_reference(path))
    result = manifest(
        side, intervals, refs, radar, artifacts, [p["analysis_scope"] for p in packets]
    )
    with (output / "review_packet_manifest.json").open("x", encoding="utf-8") as stream:
        json.dump(result, stream, indent=2)
    return result


def validate(source, output, *, radar=None):
    saved = json.loads((output / "review_packet_manifest.json").read_bytes())
    a.require(
        saved["schema_id"] == "stratai.contextual_review_packet_manifest@2",
        "packet version mismatch",
    )
    packets, refs = compute(
        source, side=saved["side"], intervals=saved["intervals"], radar=radar
    )
    artifacts = []
    for i, packet in enumerate(packets, 1):
        path, html = (output / f"case_{i:03d}.{ext}" for ext in ("json", "html"))
        a.require(
            json.loads(path.read_bytes()) == packet, "packet facts differ from sources"
        )
        a.require(
            html.read_text(encoding="utf-8") == render(packet, radar),
            "packet view differs",
        )
        artifacts.extend([physical.file_reference(path), physical.file_reference(html)])
    expected = manifest(
        saved["side"],
        saved["intervals"],
        refs,
        radar,
        artifacts,
        [p["analysis_scope"] for p in packets],
    )
    a.require(saved == expected, "packet manifest or sources changed")
    return saved


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("materialize", "validate", "analyze"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--radar", type=Path)
    parser.add_argument("--map-vmap", type=Path)
    parser.add_argument("--side", choices=("T", "CT"), default="T")
    parser.add_argument("--intervals", nargs="+")
    args = parser.parse_args()
    if args.action == "analyze":
        a.require(bool(args.intervals), "explicit packet intervals required")
        result = analyze(
            args.source,
            args.output,
            side=args.side,
            intervals=[tuple(int(x) for x in v.split(":")) for v in args.intervals],
            map_vmap=args.map_vmap,
        )
        print(json.dumps({"ok": True, "scopes": [r["scope"] for r in result["rows"]]}))
        return
    if args.action == "materialize":
        a.require(bool(args.intervals), "explicit packet intervals required")
        result = materialize(
            args.source,
            args.output,
            side=args.side,
            intervals=[tuple(int(x) for x in v.split(":")) for v in args.intervals],
            radar=args.radar,
        )
    else:
        result = validate(args.source, args.output, radar=args.radar)
    print(
        json.dumps(
            {
                "ok": True,
                "episodes": len(result["intervals"]),
                "analysis_scopes": result["analysis_scopes"],
            }
        )
    )


if __name__ == "__main__":
    main()
