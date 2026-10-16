"""Isolated CS3 experiment; never a product coach or a training dataset.

The candidate is validated once. A/B/C receive the same causal, locally aliased
archive, approved principle bank and query limits. Private byte/source receipts
stay outside the model view. Mechanical citation checks are not tactical review.
"""

from collections import Counter
from copy import deepcopy
import json

from ai_coach.interpretation import candidate_context as candidate
from ai_coach.interpretation import common_context as common
from ai_coach.support.anchors import require

VERSION = "1.0.0-experiment.1"
SECTIONS = ("situation", "evolution", "interpretation", "uncertainty")
QUERY_LIMIT = 32
QUERY_BYTES = 512_000
PROMPT = """Produce a Spanish contextual reading with situation, evolution,
interpretation and uncertainty. Use only the supplied facts and causal queries.
For each claim use kind observed/inferred/limit, text, evidence (consulted IDs),
and principles (supplied IDs). Do not treat positions as intent, event positions
as current contacts, partial counts as exact, unknown eligibility as knives,
or loss as poor defense. Integrate a proven planted C4. Several supported
hypotheses are permitted. Explain material omissions and missing evidence.
CT repeated attack patterns and adaptation require comparable prior rounds;
these are not supplied. Do not recommend a correction without a supported,
viable alternative. No recommendations are implemented by this experiment.
Return JSON with exactly the four named arrays. Each claim has exactly kind,
text, evidence and principles. Never use case IDs, human answers or outcomes.
"""

# Authored general principles, not labels extracted from human free text.
BANK = (
    {"id": "P01", "tags": ["always"], "text": "Una lectura integra situación, evolución, interpretación e incertidumbre; un destino correcto no compensa una omisión relevante.", "source": "docs/AI_COACH_CONTEXTUAL_REPRESENTATION_CRITERION.md"},
    {"id": "P02", "tags": ["always"], "text": "Distribución y desplazamiento no prueban intención, ruta viable ni apoyo efectivo.", "source": "docs/canonical-common-factual-context-v1.md"},
    {"id": "P03", "tags": ["postplant"], "text": "La planta acreditada cambia el objetivo de la situación y debe integrarse; abstenerse de detectar preparación no demuestra comprender postplant.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
    {"id": "P04", "tags": ["unknown_eligibility"], "text": "Elegibilidad unknown no acredita ronda de cuchillos ni vuelve falsos los demás hechos observados.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
    {"id": "P05", "tags": ["partial_counts"], "text": "Un recuento parcial no certifica un enfrentamiento exacto ni transforma desconocidos en cero.", "source": "docs/canonical-common-factual-context-v1.md"},
    {"id": "P06", "tags": ["events"], "text": "Un acontecimiento fechado describe ese momento; no acredita contacto rival actual ni percepción del jugador.", "source": "docs/canonical-common-factual-context-v1.md"},
    {"id": "P07", "tags": ["CT"], "text": "Evaluar efecto de defensa, patrones T y adaptación exige antecedentes comparables; perder no demuestra mala defensa. Una corrección necesita alternativa viable respaldada.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
    {"id": "P08", "tags": ["always"], "text": "Movilización inicial puede requerir más evolución para interpretar; no existe un mínimo universal de segundos aprobado.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
)
EXAMPLES = (
    {"id": "X01", "principles": ["P01", "P02"], "premise": "Si está acreditado un compañero en A y es viable reunirse para apoyarse", "reading": "A puede ser más probable por ese apoyo; B sigue posible. Llevar C4 no fundamenta por sí solo esa preferencia.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
    {"id": "X02", "principles": ["P03", "P05"], "premise": "Si el C4 está plantado y el número de rivales es parcial", "reading": "Integrar postplant y explicar que el recuento no prueba 1v1.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
    {"id": "X03", "principles": ["P04", "P08"], "premise": "Si sólo hay movilización o elegibilidad unknown", "reading": "Conservar los hechos; declarar lo que falta, sin inventar cuchillos o un umbral temporal universal.", "source": "docs/AI_COACH_CONTEXT_COMPARISON_REVIEW.md"},
)

# Removed only from the model view; complete values/references remain private.
PRIVATE_KEYS = {
    "references", "source_refs", "source_sha256", "source_alias", "source",
    "source_sequence", "available_sequence", "basis_sequence", "sequence",
    "tick", "frame", "round", "round_number", "instance", "epoch", "id",
    "pawn_life", "pawn_handle", "controller_handle", "actor_alias", "life",
    "label", "entity_life", "build_identifier", "context_id", "canonical_manifest_sha256",
}


def encoded(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, allow_nan=False).encode("utf-8")


class Session:
    """Private evidence owner with independently bounded per-method consultation."""

    def __init__(self, value, *, through_index=None):
        result = common.build(value, through_index=through_index)
        end = result["through_index"]
        samples = value["packet"]["context"]["samples"][:end + 1]
        sequence_indices = {s["cut"]["sequence"]: i for i, s in enumerate(samples)}
        first_tick = samples[0]["cut"]["tick"]
        rate = samples[-1]["cut"]["tick_rate"]
        self._archive = []
        self._private = {}
        self._source_hash = candidate.content_hash(value)
        self._common_hash = result["common_context_id"]
        aliases, lives = {}, {}
        for row in samples:
            for a in row["actors"]:
                aliases.setdefault(a["actor_alias"], "J" + str(len(aliases) + 1))
                if a["identity"] is not None:
                    life = a["identity"]["pawn_life"]
                    lives.setdefault(life, aliases[a["actor_alias"]])
        side = value["packet"]["context"]["perspective"]["side"]
        labels = {side + str(i + 1): aliases[a] for i, a in enumerate(sorted(aliases))}
        identities = {}
        other = {}
        absent = {}
        items = {}

        def scrub(raw):
            if isinstance(raw, list):
                return [scrub(v) for v in raw]
            if isinstance(raw, dict):
                if "pawn_life" in raw and "pawn_handle" in raw:
                    identity = candidate.content_hash(raw)
                    identities.setdefault(identity, "L" + str(len(identities) + 1))
                    return {"local_life": identities[identity]}
                extra = {}
                if "entity_life" in raw:
                    life = raw["entity_life"]
                    items.setdefault(life, "I" + str(len(items) + 1))
                    extra["local_item_life"] = items[life]
                for name in ("source_sequence", "basis_sequence", "available_sequence", "sequence"):
                    if name in raw:
                        extra[name.replace("sequence", "sample_index")] = sequence_indices.get(raw[name])
                if "tick" in raw:
                    extra["seconds_in_window"] = None if raw["tick"] is None else (raw["tick"] - first_tick) / rate
                if "frame" in raw:
                    extra["frame_offset"] = None if raw["frame"] is None else raw["frame"] - samples[0]["cut"]["frame"]
                if "life" in raw and "side" in raw:
                    life = raw["life"]
                    if life in lives:
                        extra["participant"] = lives[life]
                    elif life is not None:
                        token = (raw["side"], life)
                        other.setdefault(token, "O" + str(len(other) + 1))
                        extra["participant"] = other[token]
                    else:
                        extra["participant"] = None
                return {**{k: scrub(v) for k, v in raw.items() if k not in PRIVATE_KEYS}, **extra}
            if isinstance(raw, str):
                if raw.startswith("support-actor-") and raw not in aliases:
                    absent.setdefault(raw, "U" + str(len(absent) + 1))
                    return absent[raw]
                return aliases.get(raw, labels.get(raw, raw))
            return raw

        def expose(f, evidence_id):
            self._private[evidence_id] = deepcopy(f)
            return {
                "id": evidence_id, "key": f["key"], "group": f["group"],
                "subject": aliases.get(f["subject"], f["subject"]),
                "status": f["status"], "value": scrub(f["value"]),
                "reason": f["reason"], "unit": f["unit"],
                "perspective": f["perspective"],
                "original_instrument": f["original_instrument"],
                "available_index": f["available_index"],
            }

        for i in range(end + 1):
            facts = [f for group in common._at(value, i).values() for f in group]
            facts = [f for f in facts if f["key"] != "build_identifier"]
            self._archive.append([expose(f, f"F{i}:{j}") for j, f in enumerate(facts)])
        events = []
        for i, f in enumerate(result["events"]):
            f = deepcopy(f)
            f["key"] = f["value"]["id"].split("-")[0]
            event = expose(f, f"E{i}")
            event["seconds_since_event"] = f["seconds_since_event"]
            events.append(event)
        times = [(s["cut"]["tick"] - samples[0]["cut"]["tick"]) / rate for s in samples]
        boundaries = [r["to_index"] for r in result["evolution"] if r["comparison"] == "boundary_only"]
        self._view = {
            "version": VERSION, "side": side, "instrument": value["instrument"],
            "through_index": end, "sample_times_seconds": times,
            "situation": self._archive[-1], "events": events,
            "evolution": {"sample_count": end + 1, "duration_seconds": times[-1], "boundaries": boundaries},
            "prior_rounds_status": "not_supplied", "native_status": "not_supplied",
            "bank": {"principles": deepcopy(list(BANK)), "examples": deepcopy(list(EXAMPLES))},
            "consultation": {"max_queries": QUERY_LIMIT, "max_response_bytes": QUERY_BYTES,
                             "operation": "sample(index)", "frontier": end},
        }
        self._events = events

    def consultation(self):
        return Consultation(self)

    def receipt(self):
        return {"candidate_hash": self._source_hash, "common_hash": self._common_hash,
                "input_hash": candidate.content_hash(self._view), "version": VERSION,
                "training_allowed": False, "product_supported": False}


class Consultation:
    def __init__(self, session):
        self._session = session
        self._consulted = {f["id"] for f in session._view["situation"] + session._events}
        self._queries = []
        self._bytes = 0

    def initial(self):
        return deepcopy(self._session._view)

    def sample(self, index):
        require(type(index) is int and 0 <= index <= self._session._view["through_index"], "outside causal frontier")
        require(len(self._queries) < QUERY_LIMIT, "query budget exhausted")
        result = deepcopy(self._session._archive[index])
        size = len(encoded(result))
        require(self._bytes + size <= QUERY_BYTES, "response byte budget exhausted")
        self._bytes += size
        self._queries.append({"index": index, "bytes": size, "response_hash": candidate.content_hash(result)})
        self._consulted.update(f["id"] for f in result)
        return result

    def verify(self, reading):
        require(isinstance(reading, dict) and set(reading) == set(SECTIONS), "reading sections differ")
        allowed = {p["id"] for p in BANK} | {p["id"] for p in EXAMPLES}
        claims = 0
        for section in SECTIONS:
            require(isinstance(reading[section], list), "claims must be arrays")
            for claim in reading[section]:
                require(isinstance(claim, dict) and set(claim) == {"kind", "text", "evidence", "principles"}, "claim fields differ")
                require(claim["kind"] in ("observed", "inferred", "limit"), "unknown claim kind")
                require(isinstance(claim["text"], str) and bool(claim["text"].strip()), "empty claim")
                for key, permitted in (("evidence", self._consulted), ("principles", allowed)):
                    require(isinstance(claim[key], list) and all(isinstance(x, str) and x in permitted for x in claim[key]), "unconsulted or unknown citation")
                require(claim["evidence"] or (claim["kind"] == "limit" and claim["principles"]), "unsupported claim")
                claims += 1
        require(claims > 0, "empty reading")
        return {"citation_integrity": True, "claims": claims, "semantic_truth": "not_evaluated", "tactical_quality": "not_evaluated", "queries": deepcopy(self._queries), "response_bytes": self._bytes}


def claim(kind, text, facts=(), principles=()):
    return {"kind": kind, "text": text, "evidence": [f["id"] for f in facts], "principles": list(principles)}


def baseline(consultation, *, with_principles=False):
    """Factual reference; B retrieves relevant principles, not tactical labels."""
    view = consultation.initial()
    facts = view["situation"]
    start = consultation.sample(0)
    reading = {section: [] for section in SECTIONS}
    # All final facts remain explicit, including false, zero and local unknowns.
    reading["situation"] = [claim("observed" if f["status"] in ("observed", "derived", "current") else "limit",
        f'{f["subject"] or "Contexto"}: {f["key"]} = {json.dumps(f["value"], ensure_ascii=False)}; estado {f["status"]}.', [f]) for f in facts]
    old = {(f["group"], f["subject"], f["key"]): f for f in start}
    for f in facts:
        before = old.get((f["group"], f["subject"], f["key"]))
        if before is not None and (before["status"], before["value"]) != (f["status"], f["value"]):
            reading["evolution"].append(claim("observed", f'{f["subject"] or "Contexto"}: {f["key"]} cambió entre inicio y corte: {json.dumps(before["value"], ensure_ascii=False)} → {json.dumps(f["value"], ensure_ascii=False)}. La comparación de extremos no describe los pasos intermedios.', [before, f]))
    reading["evolution"].extend(claim("observed", f'Acontecimiento {e["key"]}, hace {e["seconds_since_event"]} s: {json.dumps(e["value"], ensure_ascii=False)}.', [e]) for e in view["events"])
    reading["uncertainty"].append(claim("limit", "Esta referencia factual no produce una interpretación táctica ni acredita rutas, intención o apoyo efectivo.", principles=["P01", "P02"]))
    if with_principles:
        tags = {"always", view["side"]}
        if view["events"]:
            tags.add("events")
        for f in facts:
            if f["key"] == "objective_state" and f["value"]["phase"] == "postplant":
                tags.add("postplant")
            if f["key"] == "eligibility" and f["status"] == "unknown":
                tags.add("unknown_eligibility")
            if f["key"].endswith("_count") and f["status"] != "observed":
                tags.add("partial_counts")
        selected = set()
        for p in BANK:
            if tags.intersection(p["tags"]):
                selected.add(p["id"])
                reading["uncertainty"].append(claim("limit", p["text"], principles=[p["id"]]))
        for example in EXAMPLES:
            if set(example["principles"]) <= selected:
                reading["uncertainty"].append(claim("limit",
                    f'Ejemplo condicional: {example["premise"]}. {example["reading"]}',
                    principles=[example["id"], *example["principles"]]))
        reading["uncertainty"].append(claim("limit", "Los principios recuperados son criterios condicionales; no demuestran que su premisa se cumpla en este episodio.", principles=["P01", "P02"]))
    return reading


def compare_local(session):
    results = {}
    for method in ("A", "B"):
        access = session.consultation()
        reading = baseline(access, with_principles=method == "B")
        results[method] = {"reading": reading, "verification": access.verify(reading), "receipt": session.receipt()}
    require(results["A"]["receipt"]["input_hash"] == results["B"]["receipt"]["input_hash"], "unequal inputs")
    results["C"] = {"status": "pending_provider_model_data_cost", "executed": False}
    return results


def counts(session):
    return {"samples": len(session._archive), "facts": sum(map(len, session._archive)),
            "events": len(session._events), "local_statuses": dict(Counter(f["status"] for f in session._view["situation"]))}
