"""Local-only, bounded semantic CS3 experiment. Never a product recommendation."""

from copy import deepcopy
from dataclasses import asdict, dataclass
import ipaddress
import json
import time
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

import jsonschema

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation import replay_model_context as model_context
from ai_coach.support.anchors import require

LEGACY_VERSION = "1.0.0-local.1"
DICTIONARY_VERSION = "1.1.0-local.1"
VERSION = "1.2.0-local.1"
TEMPLATE_VERSION = "1.3.0-local.1"
HYPOTHESIS_VERSION = "1.4.0-local.1"
BOUNDED_HYPOTHESIS_VERSION = "1.5.0-local.1"
FOCUSED_HYPOTHESIS_VERSION = "1.6.0-local.1"
REGION_HYPOTHESIS_VERSION = "1.7.0-local.1"
CONTEXTUAL_VERSION = "1.8.0-local.1"
GROUNDED_VERSION = "1.9.0-local.1"
CONTEXTUAL_VERSIONS = (CONTEXTUAL_VERSION, GROUNDED_VERSION)
HYPOTHESIS_VERSIONS = (HYPOTHESIS_VERSION, BOUNDED_HYPOTHESIS_VERSION, FOCUSED_HYPOTHESIS_VERSION, REGION_HYPOTHESIS_VERSION)
TEMPLATE_VERSIONS = (TEMPLATE_VERSION, *HYPOTHESIS_VERSIONS)
ADAPTER_VERSIONS = (LEGACY_VERSION, DICTIONARY_VERSION, VERSION, *TEMPLATE_VERSIONS, *CONTEXTUAL_VERSIONS)
MODEL = "qwen3.5:9b"
DIGEST = "56671c2ab9385f9cfcb404638e32cd62d88e3501d44822208363c010179a3c90"
PROTOCOL = """Use the original reading instructions and the common factual input.
You can consult earlier samples, with exactly the supplied query/byte limits.
Return a JSON object with action, indices and reading. To request samples:
{"action":"consult","indices":[0],"reading":null}.
To finish: {"action":"finish","indices":[],"reading":{...four arrays...}}.
Consult only when useful; sample indices are not seconds. No other operations.
An explicit query limit/error is missing information; do not invent its answer.
The reading arrays are situation, evolution, interpretation and uncertainty.
Every claim has kind, text, evidence and principles, as in the original prompt.
Use only evidence IDs actually delivered to you and supplied principle IDs.
"""


def compact(value):
    return json.dumps(value, ensure_ascii=False, allow_nan=False, separators=(",", ":"))


@dataclass(frozen=True)
class Config:
    endpoint: str = "http://127.0.0.1:11434"
    model: str = MODEL
    digest: str = DIGEST
    num_ctx: int = 65_536
    num_predict: int = 8_192
    max_calls: int = 8
    max_request_bytes: int = 2_000_000
    timeout_seconds: int = 240
    think: bool = True
    seed: int = 17

    def __post_init__(self):
        parsed = urlsplit(self.endpoint)
        require(parsed.scheme == "http" and parsed.hostname is not None, "local HTTP endpoint required")
        require(ipaddress.ip_address(parsed.hostname).is_loopback, "endpoint must be loopback")
        require(not parsed.username and not parsed.password and not parsed.query and not parsed.fragment, "endpoint credentials/query forbidden")
        require(parsed.path in ("", "/") and parsed.port is not None, "endpoint must be a local origin with port")
        require(self.model == MODEL and self.digest == DIGEST, "unsealed local model")
        require(1 <= self.max_calls <= 8 and 1 <= self.num_predict < self.num_ctx <= 262_144, "invalid generation limits")


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ValueError("local model redirect forbidden")


class LocalClient:
    def __init__(self, config=Config()):
        self.config = config
        self._opener = build_opener(ProxyHandler({}), NoRedirect())

    def request(self, method, path, body=None):
        require(path in ("/api/version", "/api/tags", "/api/show", "/api/chat", "/api/ps"), "unknown local operation")
        data = None if body is None else compact(body).encode("utf-8")
        require(data is None or len(data) <= self.config.max_request_bytes, "request byte limit exceeded")
        request = Request(self.config.endpoint.rstrip("/") + path, data=data, method=method, headers={"Content-Type": "application/json"})
        try:
            with self._opener.open(request, timeout=self.config.timeout_seconds) as response:
                raw = response.read(4_000_001)
            require(len(raw) <= 4_000_000, "local response byte limit exceeded")
            value = json.loads(raw)
            require(isinstance(value, dict) and "error" not in value, "local model response error")
            return value
        except HTTPError as error:
            detail = error.read(8192).decode("utf-8", errors="replace")
            raise ValueError(f"local HTTP {error.code}: {detail}") from error

    def pin(self):
        rows = self.request("GET", "/api/tags")["models"]
        row = next((r for r in rows if r.get("name") == self.config.model), None)
        require(row is not None and row.get("digest") == self.config.digest, "local model digest changed or unavailable")
        require(not row.get("remote_host") and not row.get("remote_model"), "remote model forbidden")
        return row

    def preflight(self):
        row = self.pin()
        show = self.request("POST", "/api/show", {"model": self.config.model})
        require(not show.get("remote_host") and not show.get("remote_model"), "remote model forbidden")
        require("completion" in show.get("capabilities", []), "model cannot generate readings")
        return {"version": self.request("GET", "/api/version"), "model": row, "show": show,
                "configuration": asdict(self.config), "network_scope": "loopback only; redirects and proxies disabled",
                "provider_cost_usd": 0, "downloads": 0}

    def chat(self, body):
        self.pin()
        result = self.request("POST", "/api/chat", body)
        self.pin()
        return result


def schema(access):
    if getattr(access, "contextual_selection", False):
        return access.envelope_schema()
    fact_ids = sorted(access._consulted)
    principle_ids = [p["id"] for p in (*comparison.BANK, *comparison.EXAMPLES)]
    claim = {"type": "object", "additionalProperties": False,
             "required": ["kind", "text", "evidence", "principles"],
             "properties": {"kind": {"enum": ["observed", "inferred", "limit"]},
                            "text": {"type": "string", "minLength": 1},
                            "evidence": {"type": "array", "items": {"enum": fact_ids}},
                            "principles": {"type": "array", "items": {"enum": principle_ids}}}}
    reading = {"type": "object", "additionalProperties": False, "required": list(comparison.SECTIONS),
               "properties": {key: {"type": "array", "items": claim} for key in comparison.SECTIONS}}
    if getattr(access, "reference_selection", False):
        reference = {"enum": fact_ids}
        reading["properties"] = {key: {"type": "array", "maxItems": 256, "items": reference}
                                 for key in comparison.SECTIONS}
        reading["properties"]["interpretation"]["items"] = {
            "type": "object", "additionalProperties": False,
            "required": ["principle_id", "evidence"], "properties": {
                "principle_id": {"enum": [p["id"] for p in comparison.BANK]},
                "evidence": {"type": "array", "minItems": 1, "maxItems": 32, "items": reference}}}
        if getattr(access, "hypothesis_selection", False):
            interpretation = reading["properties"]["interpretation"]
            interpretation["items"] = access.interpretation_schema(interpretation["items"])
        if getattr(access, "hypothesis_adapter_version", None) in (BOUNDED_HYPOTHESIS_VERSION, FOCUSED_HYPOTHESIS_VERSION, REGION_HYPOTHESIS_VERSION):
            return access.envelope_schema(reading, access.initial()["through_index"])
    return {"type": "object", "additionalProperties": False, "required": ["action", "indices", "reading"],
            "properties": {"action": {"enum": ["consult", "finish"]},
                           "indices": {"type": "array", "maxItems": comparison.QUERY_LIMIT,
                                       "items": {"type": "integer", "minimum": 0, "maximum": access.initial()["through_index"]}},
                           "reading": {"anyOf": [{"type": "null"}, reading]}}}


def model_content(access, value, version):
    require(version in ADAPTER_VERSIONS, "unknown adapter version")
    if getattr(access, "reference_selection", False) and version != LEGACY_VERSION:
        wire_version = {DICTIONARY_VERSION: model_context.SHARED_VERSION,
                        VERSION: model_context.VERSION,
                        TEMPLATE_VERSION: model_context.TEMPLATE_VERSION,
                        HYPOTHESIS_VERSION: model_context.TEMPLATE_VERSION,
                        BOUNDED_HYPOTHESIS_VERSION: model_context.TEMPLATE_VERSION,
                        FOCUSED_HYPOTHESIS_VERSION: model_context.TEMPLATE_VERSION,
                        REGION_HYPOTHESIS_VERSION: model_context.TEMPLATE_VERSION,
                        CONTEXTUAL_VERSION: model_context.TEMPLATE_VERSION,
                        GROUNDED_VERSION: model_context.TEMPLATE_VERSION}[version]
        return compact(model_context.pack(value, version=wire_version))
    return compact(value)


def initial_messages(access, *, version=VERSION):
    require(version in ADAPTER_VERSIONS, "unknown adapter version")
    require(bool(getattr(access, "contextual_selection", False)) == (version in CONTEXTUAL_VERSIONS),
            "adapter requires matching contextual session")
    require(bool(getattr(access, "grounded_selection", False)) == (version == GROUNDED_VERSION),
            "adapter requires matching grounded session")
    if version in CONTEXTUAL_VERSIONS:
        from ai_coach.interpretation.replay_contextual_reading import PROMPT
        if version == GROUNDED_VERSION:
            from ai_coach.interpretation.replay_grounded_reading import PROMPT
        return [{"role":"system", "content":PROMPT + "\n" + model_context.TEMPLATE_INSTRUCTIONS},
                {"role":"user", "content":model_content(access, access.initial(), version)},
                {"role":"user", "content":compact(access.current_overview())}]
    require(bool(getattr(access, "hypothesis_selection", False)) == (version in HYPOTHESIS_VERSIONS),
            "adapter requires matching hypothesis session")
    require(not getattr(access, "hypothesis_selection", False) or access.hypothesis_adapter_version == version,
            "adapter requires matching hypothesis session version")
    if getattr(access, "reference_selection", False):
        from ai_coach.interpretation.replay_factual_reading import PROMPT
        if version == HYPOTHESIS_VERSION:
            from ai_coach.interpretation.replay_attack_hypothesis import PROMPT
        elif version == BOUNDED_HYPOTHESIS_VERSION:
            from ai_coach.interpretation.replay_attack_hypothesis import BOUNDED_PROMPT as PROMPT
        elif version == FOCUSED_HYPOTHESIS_VERSION:
            from ai_coach.interpretation.replay_attack_hypothesis import FOCUSED_PROMPT as PROMPT
        elif version == REGION_HYPOTHESIS_VERSION:
            from ai_coach.interpretation.replay_attack_region import PROMPT
        protocol = ('Return {"action":"finish","indices":[],"reading":{...four reference arrays...}}. '
                    'For a causal query return {"action":"consult","indices":[0],"reading":null}. '
                    'Use only delivered IDs; no free text or extra fields in any section.')
        instructions = (model_context.SHARED_INSTRUCTIONS if version == DICTIONARY_VERSION else
                        model_context.INSTRUCTIONS if version == VERSION else
                        model_context.TEMPLATE_INSTRUCTIONS if version in TEMPLATE_VERSIONS else "")
        system = PROMPT + protocol + ("\n" + instructions if instructions else "")
        if version in TEMPLATE_VERSIONS:
            system += ("\nExpert criterion: early-round utility may be routine preventive utility against a rush. "
                       "Its timing together with a nearby reload alone does not establish a reaction to an entry. "
                       "Native demo sound map points are review-only and are not additional model evidence. ")
            system += ("Do not infer a defensive reaction from that coincidence alone." if version in (FOCUSED_HYPOTHESIS_VERSION, REGION_HYPOTHESIS_VERSION) else
                       "Do not force an interpretation when delivered facts do not distinguish alternatives.")
        messages = [{"role": "system", "content": system},
                    {"role": "user", "content": model_content(access, access.initial(), version)}]
        if version in (FOCUSED_HYPOTHESIS_VERSION, REGION_HYPOTHESIS_VERSION):
            messages.append({"role": "user", "content": compact(access.spatial_review())})
        return messages
    return [{"role": "system", "content": comparison.PROMPT + "\n" + PROTOCOL},
            {"role": "user", "content": compact(access.initial())}]


def request_body(access, messages, config):
    return {"model": config.model, "messages": deepcopy(messages), "stream": False,
            "think": config.think, "format": schema(access), "truncate": False, "shift": False,
            "keep_alive": "5m", "options": {"num_ctx": config.num_ctx, "num_predict": config.num_predict,
                                             "temperature": 0, "seed": config.seed, "presence_penalty": 0}}


def consume(access, response, messages, config, *, version=VERSION):
    require(response.get("model") == config.model and response.get("done") is True, "incomplete or different model response")
    require(response.get("done_reason") == "stop", "generation limit reached")
    require(type(response.get("prompt_eval_count")) is int and 0 < response["prompt_eval_count"] <= config.num_ctx, "invalid context accounting")
    message = response.get("message", {})
    require(not message.get("tool_calls"), "unexpected executable tool call")
    answer = json.loads(message["content"])
    jsonschema.validate(answer, schema(access))
    if getattr(access, "grounded_selection", False) and access.grounding_stage == "audit":
        return access.complete_audit(answer)
    if answer["action"] == "finish":
        require(not answer["indices"] and answer["reading"] is not None, "invalid finish envelope")
        if getattr(access, "grounded_selection", False):
            access.submit(answer["reading"])
            messages[:] = access.audit_messages()
            return None
        if getattr(access, "reference_selection", False):
            return access.render(answer["reading"])
        return answer["reading"], access.verify(answer["reading"])
    require(answer["reading"] is None and bool(answer["indices"]), "invalid consultation envelope")
    contextual = bool(getattr(access, "contextual_selection", False))
    if contextual:
        checkpoint = (deepcopy(access._queries), access._bytes, set(access._consulted))
    messages.append({"role": "assistant", "content": message["content"]})
    results = []
    for index in answer["indices"]:
        try:
            facts = (access.query(index, "history" if answer["action"] == "consult_history" else "full")
                     if contextual else access.sample(index))
            results.append({"index": index, "facts": facts})
        except ValueError as error:
            results.append({"index": index, "error": str(error)})
    reply = model_content(access, {"consultation_results": results}, version)
    if contextual:
        # A conservative byte-level bound for the pinned local tokenizer,
        # including chat framing. Refuse a whole reply, never trim its facts.
        upper = response["prompt_eval_count"] + len(message["content"].encode("utf-8")) + len(reply.encode("utf-8")) + 128
        if upper > config.num_ctx-config.num_predict:
            access._queries, access._bytes, access._consulted = checkpoint
            access._context_budget_refusals.append({"indices":list(answer["indices"]),
                "scope":"history" if answer["action"] == "consult_history" else "full",
                "reason":"not_delivered_context_budget", "conservative_token_bound":upper})
            results = [{"indices":answer["indices"], "error":
                "not_delivered_context_budget: no samples from this query were delivered; do not cite them. "
                "Use existing evidence or another available scope; finish with local uncertainty if needed."}]
            reply = model_content(access, {"consultation_results":results}, version)
    messages.append({"role": "user", "content": reply})
    return None


def interpret(session, client, *, record=lambda row: None, version=VERSION):
    """One attempt, without repairs, hidden retries or tactical auto-labels."""
    require(version in ADAPTER_VERSIONS, "unknown adapter version")
    input_version = getattr(session, "_version", "")
    require(not input_version.startswith("stratai.replay_context_input@")
            or input_version in ("stratai.replay_context_input@6", "stratai.replay_context_input@7", "stratai.replay_context_input@8", "stratai.replay_context_input@9"),
            "historical replay inputs are read-only; use protected input@6, @7, @8 or @9 for new generation")
    config = client.config
    access = session.consultation()
    messages = initial_messages(access, version=version)
    transcript = []
    reading, verification, failure = None, None, None
    start = time.monotonic()
    metadata = client.preflight()
    record({"kind": "preflight", "value": metadata})
    for index in range(config.max_calls):
        if getattr(access, "contextual_selection", False):
            access._remaining_model_calls = config.max_calls-index
        body = request_body(access, messages, config)
        entry = {"index": index, "request": body}
        record({"kind": "request", "value": entry})
        call_start = time.monotonic()
        try:
            response = client.chat(body)
            entry.update(response=response, seconds=round(time.monotonic() - call_start, 3))
            record({"kind": "response", "value": entry})
            finished = consume(access, response, messages, config, version=version)
            transcript.append(entry)
            if finished is not None:
                reading, verification = finished
                break
        except (ValueError, KeyError, jsonschema.ValidationError, OSError) as error:
            entry["failure"] = str(error)
            if entry not in transcript:
                transcript.append(entry)
            failure = str(error)
            record({"kind": "failure", "value": entry})
            break
    if reading is None and failure is None:
        failure = "model call budget exhausted"
    return {"version": version, "executed": bool(transcript), "status": "mechanically_valid" if verification else "rejected_or_incomplete",
            "reading": reading, "verification": verification, "failure": failure, "metadata": metadata,
            "transcript": transcript, "receipt": session.receipt(),
            "queries": deepcopy(access._queries), "response_bytes": access._bytes,
            "seconds": round(time.monotonic() - start, 3), "local_calls": len(transcript), "external_calls": 0,
            "provider_cost_usd": 0, "semantic_truth": "not_evaluated", "tactical_quality": "not_evaluated",
            "training_allowed": False, "product_supported": False,
            **({"contextual_runtime":access.runtime_state()} if getattr(access,"contextual_selection",False) else {})}


def verify_replay(session, result):
    """Offline replay verifies exactly which inputs/citations were available."""
    require(result["version"] in ADAPTER_VERSIONS and result["receipt"] == session.receipt(), "semantic receipt differs")
    config = Config(**result["metadata"]["configuration"])
    require(result["metadata"]["model"]["digest"] == config.digest, "model metadata digest differs")
    access = session.consultation()
    messages = initial_messages(access, version=result["version"])
    finished = None
    for index, entry in enumerate(result["transcript"]):
        if getattr(access, "contextual_selection", False):
            access._remaining_model_calls = config.max_calls-index
        require(finished is None, "calls after final reading")
        require(entry["request"] == request_body(access, messages, config), "model request differs from causal replay")
        if "failure" in entry:
            break
        finished = consume(access, entry["response"], messages, config, version=result["version"])
    require(len(result["transcript"]) <= config.max_calls, "call budget exceeded")
    require(access._queries == result["queries"] and access._bytes == result["response_bytes"], "consultation replay differs")
    if getattr(access, "contextual_selection", False):
        require(result["contextual_runtime"] == access.runtime_state(), "contextual query state differs")
    require(result["external_calls"] == result["provider_cost_usd"] == 0 and result["training_allowed"] is result["product_supported"] is False, "local experiment gate differs")
    if result["status"] == "mechanically_valid":
        require(finished is not None and finished == (result["reading"], result["verification"]), "reading replay differs")
    else:
        require(result["reading"] is None and result["verification"] is None and bool(result["failure"]), "rejection asserted as valid")
    return {"causal_replay": True, "citation_integrity": finished is not None,
            "semantic_truth": "not_evaluated", "tactical_quality": "not_evaluated"}
