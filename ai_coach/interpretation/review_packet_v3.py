"""Review cards with current ally/opponent counts and a neutral factual summary."""

import argparse
import json
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import contextual_io, inventory, review_packet as v1
from ai_coach.interpretation import review_packet_v2 as v2
from ai_coach.interpretation import round_context as rounds, team_status
from ai_coach.support import anchors as a, physical

TEMPLATE = Path(__file__).with_suffix(".html")


def build(value, support, sources, quantities, map_name, optional):
    packet = v2.build(value, support, sources, quantities, map_name, optional)
    packet["schema_id"] = "stratai.contextual_review_packet@3"
    packet["team_status_scope"] = team_status.SCOPE
    packet["team_status"] = team_status.sequence(value, support)
    a.schema_validator("contextual_review_packet_v3.json").validate(packet)
    return packet


def render(packet, radar=None):
    a.schema_validator("contextual_review_packet_v3.json").validate(packet)
    base = {
        k: v for k, v in packet.items() if k not in ("team_status", "team_status_scope")
    }
    base["schema_id"] = "stratai.contextual_review_packet@2"
    a.require(
        len(packet["team_status"]) == len(packet["context"]["samples"])
        and all(
            row["source_sequence"] == sample["cut"]["sequence"]
            for row, sample in zip(
                packet["team_status"], packet["context"]["samples"], strict=True
            )
        ),
        "team count samples differ",
    )
    html = v2.render(base, radar)
    supplement = TEMPLATE.read_text(encoding="utf-8").replace(
        "__TEAM_JSON__",
        json.dumps(packet["team_status"], ensure_ascii=False).replace("<", "\\u003c"),
    )
    return html.replace("</html>", supplement + "</html>")


def compute(source, *, side, intervals):
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
    result = v2.manifest(side, intervals, refs, radar, artifacts, scopes)
    result["schema_id"] = "stratai.contextual_review_packet_manifest@3"
    result["templates"] = [
        physical.file_reference(p) for p in (v1.TEMPLATE, v2.TEMPLATE, TEMPLATE)
    ]
    result["team_status_scope"] = team_status.SCOPE
    return result


def materialize(source, output, *, side, intervals, radar=None):
    output = validate_output_directory(output)
    packets, refs = compute(source, side=side, intervals=intervals)
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
        saved["schema_id"] == "stratai.contextual_review_packet_manifest@3",
        "packet version differs",
    )
    packets, refs = compute(source, side=saved["side"], intervals=saved["intervals"])
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
    parser.add_argument("action", choices=("materialize", "validate"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--radar", type=Path)
    parser.add_argument("--side", choices=("T", "CT"), default="T")
    parser.add_argument("--intervals", nargs="+")
    args = parser.parse_args()
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
                "scopes": result["analysis_scopes"],
            }
        )
    )


if __name__ == "__main__":
    main()
