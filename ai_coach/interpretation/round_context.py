"""Causal round eligibility and clocks, separate from tactical interpretation."""

import json

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as clock
from ai_coach.support import physical

DATA = "support_round_context_observations.json"
CONVARS = (
    "mp_give_player_c4",
    *[
        f"mp_{side}_default_{kind}"
        for side in ("t", "ct")
        for kind in ("primary", "secondary", "grenades", "melee")
    ],
)
CLOCK_FIELDS = (
    "pause_state",
    "clock_state",
    "round_remaining_seconds",
    "round_clock_state",
    "bomb_remaining_seconds",
    "bomb_clock_state",
)


def validate_source(data, support):
    a.schema_validator(DATA).validate(data)
    for key in ("source_alias", "source_sha256", "build_identifier"):
        a.require(data[key] == support[key], "round context identity mismatch")
    messages = data["observations"]["messages"]
    previous_order = previous_frame = previous_session = 0
    for seq, message in enumerate(messages, 1):
        a.require(
            message["sequence"] == seq
            and message["order"] > previous_order
            and message["frame"] >= previous_frame
            and message["session"] >= previous_session,
            "round message order",
        )
        previous_order, previous_frame, previous_session = (
            message[k] for k in ("order", "frame", "session")
        )
    cuts = data["observations"]["frames"]
    bases = support["observations"]["frames"]
    a.require(len(cuts) == len(bases), "round context coverage")
    cursor, last_order, last_session = 0, 0, 0
    for cut, base in zip(cuts, bases, strict=True):
        a.require(
            cut["source_sequence"] == base["sequence"]
            and all(cut[k] == base[k] for k in ("frame", "tick", "instance", "epoch")),
            "round context exact cut mismatch",
        )
        a.require(
            cut["order"] > last_order and cut["session"] >= last_session,
            "round cut order",
        )
        while cursor < len(messages) and messages[cursor]["order"] < cut["order"]:
            message = messages[cursor]
            a.require(
                message["frame"] <= cut["frame"]
                and message["session"] <= cut["session"]
                and (
                    message["frame"] != cut["frame"] or message["tick"] == cut["tick"]
                ),
                "future round configuration",
            )
            cursor += 1
        a.require(cut["message_sequence"] == cursor, "round configuration frontier")
        for value in cut["rules"].values():
            clock.validate_scalar(value)
        last_order, last_session = cut["order"], cut["session"]


def load_optional(directory, support, sources):
    result, refs = {}, {}
    for filename in (DATA, clock.DATA):
        path = directory / filename
        if not path.exists():
            result[filename] = None
            continue
        a.candidate_io.source_io._safe_file(directory.absolute(), filename)
        a.require(path.stat().st_size < 512 * 1024 * 1024, "oversized round source")
        before = physical.file_reference(path)
        data = json.loads(path.read_bytes())
        if filename == DATA:
            a.require(
                data["support_source"] == physical.file_reference(directory / a.DATA),
                "round support hash mismatch",
            )
            validate_source(data, support)
        else:
            for key, file in (
                ("support_source", a.DATA),
                ("objective_source", o.DATA),
                ("c4_source", "support_c4_observations.json"),
            ):
                a.require(
                    data["references"][key]
                    == physical.file_reference(directory / file),
                    "round clock provenance mismatch",
                )
            clock.validate_source(
                data, support, sources["objective_source"], sources["c4_source"]
            )
        a.require(
            before == physical.file_reference(path),
            "round source changed while reading",
        )
        result[filename], refs[filename] = data, before
    return result, refs


def configurations(data):
    """Only delivered assignments; no default values or future backfill."""
    if data is None:
        return {}
    messages = data["observations"]["messages"]
    states, values, cursor, session = {}, {}, 0, None
    for cut in data["observations"]["frames"]:
        if cut["session"] != session:
            values = {}
        session = cut["session"]
        while cursor < cut["message_sequence"]:
            message = messages[cursor]
            if message["session"] == session:
                for entry in message["entries"]:
                    values[entry["name"]] = entry["value"]
            cursor += 1
        states[cut["source_sequence"]] = {"cut": cut, "values": dict(values)}
    return states


def knife_inventory(sample, ownership):
    """Require complete, exactly owned inventories for every living ally."""
    seq = sample["cut"]["sequence"]
    cut = ownership["observations"]["cuts"][seq - 1]
    state = ownership["observations"]["states"][cut["state_sequence"] - 1]
    people = {p["actor_alias"]: p for p in state["players"]}
    entities = {e["entity_life"]: e for e in state["entities"]}
    if len(sample["actors"]) != 5:
        return False
    living = 0
    for actor in sample["actors"]:
        if actor["alive"]["value"] is False:
            continue
        if actor["alive"]["value"] is not True or actor["exclusion"] is not None:
            return False
        living += 1
        p = people.get(actor["actor_alias"])
        if p is None or any(
            p[k] != actor["identity"][k] for k in ("pawn_life", "pawn_handle")
        ):
            return False
        if p["inventory_status"] != "observed" or p["inventory_length"] != len(
            p["slots"]
        ):
            return False
        knives = set()
        for ref in p["slots"]:
            if ref["status"] == "invalid_sentinel":
                continue
            e = entities.get(ref["target_life"])
            if (
                ref["status"] != "resolved"
                or e is None
                or e["handle"] != ref["handle"]
                or e["entity_life"] in knives
                or e["class_name"] != "CKnife"
                or e["owner"]["status"] != "resolved"
                or e["owner"]["target_life"] != p["pawn_life"]
                or e["owner"]["handle"] != p["pawn_handle"]
            ):
                return False
            knives.add(e["entity_life"])
        if not knives:
            return False
    return living > 0


def eligibility(sample, objective, configuration, ownership, prior_bomb=False):
    result = {
        "kind": "unknown",
        "analysis_allowed": False,
        "reason": "round_kind_unconfirmed",
        "basis_sequence": sample["cut"]["sequence"],
    }
    warmup = o.native(objective["rules"]["m_bWarmupPeriod"])["value"]
    if warmup is True:
        return {**result, "kind": "warmup", "reason": "native_warmup"}
    if warmup is not False:
        return result
    planted = o.native(objective["rules"]["m_bBombPlanted"])["value"]
    bomb = planted is True or any(
        p["has_c4"]["value"] is True for p in sample["actors"]
    )
    configured = False
    if configuration:
        values, rules = configuration["values"], configuration["cut"]["rules"]
        configured = (
            values.get("mp_give_player_c4") in ("0", "false")
            and all(
                values.get(f"mp_{s}_default_{k}") == ""
                for s in ("t", "ct")
                for k in ("primary", "secondary", "grenades")
            )
            and all(
                values.get(f"mp_{s}_default_melee") == "weapon_knife"
                for s in ("t", "ct")
            )
            and all(
                clock.native(rules[k], "bool")["value"] is True
                for k in ("m_bTCantBuy", "m_bCTCantBuy")
            )
            and clock.native(rules["m_bWarmupPeriod"], "bool")["value"] is False
        )
    if configured:
        if (
            bomb
            or prior_bomb
            or planted is not False
            or not knife_inventory(sample, ownership)
        ):
            return {**result, "reason": "knife_configuration_not_corroborated"}
        return {
            **result,
            "kind": "knife",
            "reason": "native_knife_configuration_and_inventory",
        }
    if bomb or prior_bomb:
        return {
            **result,
            "kind": "bomb_objective",
            "analysis_allowed": True,
            "reason": "public_plant_or_ally_c4_in_round_prefix",
        }
    return result


def sequence(value, support, sources, optional):
    end = value["requested_interval"]["end_sequence"]
    support = {
        **support,
        "observations": {"frames": support["observations"]["frames"][:end]},
    }
    original = sources["objective_source"]
    frames = original["observations"]["frames"][:end]
    objectives = {
        **original,
        "observations": {
            "frames": frames,
            "events": original["observations"]["events"][
                : frames[-1]["event_sequence"]
            ],
        },
    }
    public = o.public_prefixes(objectives)
    config_data = optional[DATA]
    if config_data:
        cuts = config_data["observations"]["frames"][:end]
        config_data = {
            "observations": {
                "frames": cuts,
                "messages": config_data["observations"]["messages"][
                    : cuts[-1]["message_sequence"]
                ],
            }
        }
    configurations_by_seq = configurations(config_data)
    clocks = optional[clock.DATA]
    if clocks:
        clocks = {
            **clocks,
            "observations": {"frames": clocks["observations"]["frames"][:end]},
        }
    witnesses = (
        clock.alignment_prefixes(support, objectives, clocks)[0] if clocks else {}
    )
    clock_rows = (
        {r["source_sequence"]: r for r in clocks["observations"]["frames"]}
        if clocks
        else {}
    )
    result, previous_key, prior_bomb = [], None, None
    for sample in value["samples"]:
        seq = sample["cut"]["sequence"]
        base = support["observations"]["frames"][seq - 1]
        objective = objectives["observations"]["frames"][seq - 1]
        config = configurations_by_seq.get(seq)
        key = (
            base["epoch"],
            base["instance"],
            config["cut"]["session"] if config else None,
        )
        if key != previous_key:
            prior_bomb = None
        if o.native(objective["rules"]["m_bWarmupPeriod"])["value"] is True:
            prior_bomb = None
        elif o.native(objective["rules"]["m_bWarmupPeriod"])["value"] is False and (
            o.native(objective["rules"]["m_bBombPlanted"])["value"] is True
            or any(p["has_c4"]["value"] is True for p in sample["actors"])
        ):
            prior_bomb = seq
        gate = eligibility(sample, objective, config, sources["c4_source"], prior_bomb)
        if gate["kind"] == "bomb_objective":
            gate["basis_sequence"] = prior_bomb
        previous_key = key
        native_clocks = []
        row = clock_rows.get(seq)
        witness = witnesses.get(seq)
        for actor in sample["actors"]:
            if actor["alive"]["value"] is not True or actor["exclusion"] is not None:
                continue
            alias = actor["actor_alias"]
            anchor = {"actor_alias": alias, "ally_alias": alias}
            facts = clock.initial_context(
                anchor, base, objective, public.get(seq, {}), None, row, witness=witness
            )["self"]
            native_clocks.append({k: facts[k] for k in CLOCK_FIELDS})
        # Agreement among all current legal actor clocks; never consult rivals.
        facts = {}
        for field in CLOCK_FIELDS:
            observed = [
                c[field] for c in native_clocks if c[field]["status"] == "observed"
            ]
            if observed and all(f == observed[0] for f in observed):
                facts[field] = observed[0]
            elif observed:
                facts[field] = o.fact(reason="legal_actor_clocks_disagree")
            else:
                facts[field] = (
                    native_clocks[0][field]
                    if native_clocks
                    else o.fact(reason="no_current_ally_clock")
                )
        result.append(
            {
                "source_sequence": seq,
                "eligibility": gate,
                "round_elapsed_seconds": o.elapsed(
                    public.get(seq, {}).get("freeze"), objective, base
                ),
                "configured_duration_seconds": o.native(
                    objective["rules"]["m_iRoundTime"], integer=True
                ),
                "native_clock": facts,
                "clock_origin_sequence": witness["source_sequence"]
                if witness
                else None,
                "configuration_frontier": config["cut"]["message_sequence"]
                if config
                else None,
            }
        )
    return result
