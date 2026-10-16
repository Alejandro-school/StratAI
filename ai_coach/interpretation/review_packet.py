"""Reproducible neutral replay cards with temporal native inventory. No predictions."""

import argparse
import base64
import json
from copy import deepcopy
from pathlib import Path

from ai_coach.economy.candidate_io import validate_output_directory
from ai_coach.interpretation import (
    context as context_contract,
)
from ai_coach.interpretation import (
    contextual_io,
    inventory,
)
from ai_coach.support import anchors as a
from ai_coach.support import physical

TEMPLATE = Path(__file__).with_suffix(".html")


def build(value, sources, quantities, map_name):
    context_contract.validate(value)
    objectives = {
        f["source_sequence"]: f
        for f in sources["objective_source"]["observations"]["frames"][
            : value["requested_interval"]["end_sequence"]
        ]
    }
    states = []
    for sample in value["samples"]:
        cut = sample["cut"]
        row = objectives[cut["sequence"]]
        a.require(
            all(row[k] == cut[k] for k in ("frame", "tick", "instance", "epoch")),
            "packet objective clock mismatch",
        )
        native = row["rules"]["m_bBombPlanted"]
        planted = (
            native["value"]
            if native["status"] == "observed"
            and native["native_type"] == "bool"
            and type(native["value"]) is bool
            else None
        )
        states.append(
            {
                "source_sequence": cut["sequence"],
                "planted": planted,
                "carrier_aliases": [
                    p["actor_alias"]
                    for p in sample["actors"]
                    if p["has_c4"]["value"] is True
                ],
                "position": None,
                "position_status": "unavailable",
            }
        )
    packet = {
        "schema_id": "stratai.contextual_review_packet@1",
        "context": deepcopy(value),
        "map_name": map_name,
        "inventory": inventory.sequence(value, sources["c4_source"], quantities),
        "objective": states,
        "predictions_included": False,
        "training_allowed": False,
    }
    a.schema_validator("contextual_review_packet.json").validate(packet)
    return packet


def render(packet, radar=None):
    context = packet["context"]
    aliases = sorted(
        {p["actor_alias"] for sample in context["samples"] for p in sample["actors"]}
    )
    labels = {
        alias: f"{context['perspective']['side']}{i + 1}"
        for i, alias in enumerate(aliases)
    }
    frames = []
    start = context["samples"][0]["cut"]["tick"]
    for sample, inventories, objective in zip(
        context["samples"], packet["inventory"], packet["objective"], strict=True
    ):
        actors = []
        for p in sample["actors"]:
            alias = p["actor_alias"]
            actors.append(
                {
                    "label": labels[alias],
                    "identity": p["identity"],
                    "alive": p["alive"]["value"],
                    "position": p["position"]["value"],
                    "weapon": p["physical"]["active_weapon"]["value"],
                    "clip": p["physical"]["ammo_in_magazine"]["value"],
                    "reserve": p["physical"]["ammo_reserve"]["value"],
                    "c4": p["has_c4"]["value"],
                    "inventory": inventories[alias],
                }
            )
        tick_rate = sample["cut"]["tick_rate"]
        frames.append(
            {
                "seconds": (sample["cut"]["tick"] - start) / tick_rate
                if tick_rate
                else None,
                "round": sample["cut"]["round_number"],
                "continuity": sample["continuity"],
                "actors": actors,
                "planted": objective["planted"],
                "carriers": [labels[a] for a in objective["carrier_aliases"]],
            }
        )
    display = {
        "map": packet["map_name"],
        "frames": frames,
        "has_radar": radar is not None,
    }
    # Only numeric map projection supported by this local radar, never nearest-place assignment.
    if radar is not None:
        a.require(
            packet["map_name"] == "de_mirage"
            and radar.name == "de_mirage_radar_psd.webp",
            "radar map mismatch",
        )
    uri = (
        "data:image/webp;base64," + base64.b64encode(radar.read_bytes()).decode()
        if radar
        else ""
    )
    return (
        TEMPLATE.read_text(encoding="utf-8")
        .replace(
            "__DISPLAY_JSON__",
            json.dumps(display, ensure_ascii=False).replace("<", "\\u003c"),
        )
        .replace("__RADAR_URI__", uri)
    )


def compute(source, *, side, intervals, radar):
    support, sources, refs, map_name, _ = contextual_io.load_inputs(source, None)
    quantities, ref = inventory.load_source(source, sources["c4_source"])
    if ref:
        refs[inventory.DATA] = ref
    packets = [
        build(value, sources, quantities, map_name)
        for value in contextual_io.episodes(
            support, sources, side=side, intervals=intervals
        )
    ]
    return packets, refs


def materialize(source, output, *, side, intervals, radar=None):
    output = validate_output_directory(output)
    packets, refs = compute(source, side=side, intervals=intervals, radar=radar)
    artifacts = []
    for i, packet in enumerate(packets, 1):
        for suffix, body in [
            ("json", json.dumps(packet, ensure_ascii=False, sort_keys=True)),
            ("html", render(packet, radar)),
        ]:
            path = output / f"case_{i:03d}.{suffix}"
            with path.open("x", encoding="utf-8") as stream:
                stream.write(body)
            artifacts.append(physical.file_reference(path))
    manifest = {
        "schema_id": "stratai.contextual_review_packet_manifest@1",
        "side": side,
        "intervals": [list(pair) for pair in intervals],
        "source_artifacts": refs,
        "radar": physical.file_reference(radar) if radar else None,
        "template_sha256": physical.file_reference(TEMPLATE)["sha256"],
        "artifacts": artifacts,
        "training_allowed": False,
    }
    with (output / "review_packet_manifest.json").open("x", encoding="utf-8") as stream:
        json.dump(manifest, stream, indent=2)
    return manifest


def validate(source, output, *, radar=None):
    manifest = json.loads((output / "review_packet_manifest.json").read_bytes())
    packets, refs = compute(
        source, side=manifest["side"], intervals=manifest["intervals"], radar=radar
    )
    a.require(refs == manifest["source_artifacts"], "packet source hashes changed")
    a.require(
        manifest["radar"] == (physical.file_reference(radar) if radar else None),
        "packet radar changed",
    )
    a.require(
        manifest["template_sha256"] == physical.file_reference(TEMPLATE)["sha256"]
        and manifest["training_allowed"] is False,
        "packet template/boundary changed",
    )
    expected = []
    for i, packet in enumerate(packets, 1):
        path = output / f"case_{i:03d}.json"
        a.require(
            json.loads(path.read_bytes()) == packet, "packet facts differ from sources"
        )
        html = output / f"case_{i:03d}.html"
        a.require(
            html.read_text(encoding="utf-8") == render(packet, radar),
            "packet view differs",
        )
        expected.extend([physical.file_reference(path), physical.file_reference(html)])
    a.require(
        manifest["artifacts"] == expected, "packet artifact coverage/hash mismatch"
    )
    return manifest


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
        intervals = [tuple(int(x) for x in v.split(":")) for v in args.intervals]
        result = materialize(
            args.source,
            args.output,
            side=args.side,
            intervals=intervals,
            radar=args.radar,
        )
    else:
        result = validate(args.source, args.output, radar=args.radar)
    print(json.dumps({"ok": True, "episodes": len(result["intervals"])}))


if __name__ == "__main__":
    main()
