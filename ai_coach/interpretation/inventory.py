"""Current native grenade ownership and optional quantities, never replay cache."""

import json
from copy import deepcopy

from ai_coach.support import anchors as a
from ai_coach.support import c4, physical

DATA = "support_inventory_observations.json"
RESERVE = ("m_pReserveAmmo.0000", "m_iPrimaryReserveAmmoCount")
# Definition IDs from the pinned CS2 item definitions / demoinfocs v5.2.0.
GRENADES = {
    43: "flash",
    44: "HE",
    45: "humo",
    46: "molotov",
    47: "señuelo",
    48: "incendiaria",
}


def validate_source(source, ownership):
    a.schema_validator(DATA).validate(source)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(source[key] == ownership[key], "inventory source mismatch")
    cuts, states = (source["observations"][key] for key in ("cuts", "states"))
    original = ownership["observations"]
    a.require(
        len(cuts) == len(original["cuts"]) and len(cuts) > 0, "inventory cut coverage"
    )
    last, previous = 0, None
    for index, state in enumerate(states, 1):
        a.require(state["sequence"] == index, "inventory state sequence")
        ids = [item["entity_life"] for item in state["items"]]
        a.require(len(ids) == len(set(ids)), "duplicate inventory life")
    for cut, native in zip(cuts, original["cuts"], strict=True):
        a.require(
            all(
                cut[k] == native[k]
                for k in ("source_sequence", "frame", "tick", "instance", "epoch")
            ),
            "inventory clock mismatch",
        )
        a.require(
            cut["c4_state_sequence"] == native["state_sequence"],
            "inventory ownership frontier",
        )
        seq = cut["state_sequence"]
        a.require(
            seq in (last, last + 1) and 1 <= seq <= len(states),
            "inventory state frontier",
        )
        state = states[seq - 1]
        a.require(state["first_cut"] <= cut["source_sequence"], "future inventory")
        if seq != last:
            a.require(
                state["first_cut"] == cut["source_sequence"], "inventory first cut"
            )
        else:
            a.require(
                previous is not None
                and all(cut[k] == previous[k] for k in ("instance", "epoch")),
                "inventory crossed boundary",
            )
        entities = original["states"][native["state_sequence"] - 1]["entities"]
        a.require(
            {(e["entity_life"], e["handle"]) for e in entities}
            == {(i["entity_life"], i["handle"]) for i in state["items"]},
            "inventory entity coverage",
        )
        last, previous = seq, cut
    a.require(last == len(states), "unreferenced inventory state")


def load_source(directory, ownership):
    path = directory / DATA
    if not path.exists():
        return None, None
    a.candidate_io.source_io._safe_file(directory.absolute(), DATA)
    a.require(path.stat().st_size < 512 * 1024 * 1024, "oversized inventory")
    source = json.loads(path.read_bytes())
    a.require(
        source["c4_source"] == physical.file_reference(directory / c4.DATA),
        "inventory ownership hash mismatch",
    )
    validate_source(source, ownership)
    return source, physical.file_reference(path)


def quantity(item):
    """For a corroborated grenade entity: one held unit + native reserve.

    Mirrors the pinned SDK's grenade semantics, without its missing-value zero.
    A present invalid primary property cannot silently fall back to another.
    """
    if item is None:
        return None, "quantity_source_unavailable"
    for key in RESERVE:
        field = item["fields"][key]
        if field["status"] == "absent":
            continue
        value = field["value"]
        if (
            field["status"] == "observed"
            and field["native_type"] == "int32"
            and type(value) is int
            and value >= 0
        ):
            return 1 + value, key
        return None, "native_quantity_invalid"
    return None, "native_quantity_absent"


def at_sample(sample, ownership, quantities=None):
    """Project only own/ally actors in the supplied causal sample."""
    seq = sample["cut"]["sequence"]
    cut = ownership["observations"]["cuts"][seq - 1]
    a.require(
        cut["source_sequence"] == seq
        and all(
            cut[k] == sample["cut"][k] for k in ("frame", "tick", "epoch", "instance")
        ),
        "inventory sample clock mismatch",
    )
    state = ownership["observations"]["states"][cut["state_sequence"] - 1]
    a.require(state["first_cut"] <= seq, "future ownership")
    people = {p["actor_alias"]: p for p in state["players"]}
    entities = {e["entity_life"]: e for e in state["entities"]}
    amounts = {}
    if quantities is not None:
        qcut = quantities["observations"]["cuts"][seq - 1]
        a.require(
            all(
                qcut[k] == cut[k]
                for k in ("source_sequence", "frame", "tick", "instance", "epoch")
            )
            and qcut["c4_state_sequence"] == cut["state_sequence"],
            "quantity clock mismatch",
        )
        qstate = quantities["observations"]["states"][qcut["state_sequence"] - 1]
        a.require(qstate["first_cut"] <= seq, "future quantity")
        amounts = {i["entity_life"]: i for i in qstate["items"]}
    result = {}
    for actor in sample["actors"]:
        row = {
            "status": "unavailable",
            "items": [],
            "complete": False,
            "quantity_complete": False,
            "reason": actor["exclusion"],
            "source_sequence": seq,
        }
        result[actor["actor_alias"]] = row
        if actor["exclusion"] is not None:
            row["status"] = (
                "not_applicable" if actor["alive"]["value"] is False else "unavailable"
            )
            continue
        player = people.get(actor["actor_alias"])
        if (
            player is None
            or player["pawn_life"] != actor["identity"]["pawn_life"]
            or player["pawn_handle"] != actor["identity"]["pawn_handle"]
        ):
            row["reason"] = "current_pawn_unavailable"
            continue
        complete = player["inventory_status"] == "observed" and player[
            "inventory_length"
        ] == len(player["slots"])
        seen, items = set(), []
        conflict = False
        for ref in player["slots"]:
            if ref["status"] == "invalid_sentinel":
                continue
            if ref["status"] != "resolved":
                complete = False
                continue
            life = ref["target_life"]
            if life in seen:
                conflict = True
                continue
            seen.add(life)
            entity = entities.get(life)
            if entity is None or entity["handle"] != ref["handle"]:
                complete = False
                continue
            if c4.item_type(entity) not in ("c4", "other"):
                complete = False
                continue
            owner = entity["owner"]
            if (
                owner["status"] != "resolved"
                or owner["handle"] != player["pawn_handle"]
                or owner["target_life"] != player["pawn_life"]
            ):
                complete = False
                continue
            definition = entity["definition"]["value"]
            if definition in GRENADES:
                amount = amounts.get(life)
                a.require(
                    amount is None or amount["handle"] == entity["handle"],
                    "quantity identity mismatch",
                )
                count, source = quantity(amount)
                items.append(
                    {
                        "kind": GRENADES[definition],
                        "definition": definition,
                        "count": count,
                        "minimum_count": 1,
                        "quantity_source": source,
                        "entity_life": life,
                    }
                )
        if conflict:
            row["reason"] = "duplicate_inventory_reference"
            continue
        # Multiple entities of one definition may encode the same stack. Do not sum.
        if len({i["kind"] for i in items}) != len(items):
            row["reason"] = "ambiguous_grenade_stack"
            continue
        row.update(
            status="current" if complete else "partial" if items else "unavailable",
            items=sorted(items, key=lambda i: i["definition"]),
            complete=complete,
            quantity_complete=complete and all(i["count"] is not None for i in items),
            reason=None if complete else "inventory_coverage_incomplete",
        )
    return result


def sequence(value, ownership, quantities=None):
    """Historical lists are explicit, bounded by life/round/side/continuity."""
    memory, previous, result = {}, None, []
    for sample in value["samples"]:
        cut = sample["cut"]
        continuous = (
            previous is not None
            and sample["continuity"] == "adjacent_samples"
            and cut["sequence"] == previous["sequence"] + 1
            and all(
                cut[k] == previous[k]
                for k in (
                    "instance",
                    "epoch",
                    "round_number",
                    "sample_step",
                    "tick_rate",
                )
            )
            and cut["tick"] - previous["tick"] == cut["sample_step"]
            and cut["frame"] > previous["frame"]
        )
        if not continuous:
            memory.clear()
        current = at_sample(sample, ownership, quantities)
        for actor in sample["actors"]:
            alias = actor["actor_alias"]
            row = current[alias]
            key = actor["identity"]
            last = memory.get(alias)
            if actor["exclusion"] is not None or (last and last["identity"] != key):
                memory.pop(alias, None)
                last = None
            row["last_known"] = None
            if row["status"] == "current":
                memory[alias] = {
                    "identity": deepcopy(key),
                    "source_sequence": cut["sequence"],
                    "tick": cut["tick"],
                    "items": deepcopy(row["items"]),
                }
            elif last:
                row["last_known"] = deepcopy(last)
        memory = {k: v for k, v in memory.items() if k in current}
        result.append(current)
        previous = cut
    return result
