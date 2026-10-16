"""Local transport isolation, causal replay and honest model failure handling."""

from copy import deepcopy
import json
from threading import Thread
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import pytest

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation import context_comparison_io as factual_io
from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation import context_comparison_ollama_io as io
from ai_coach.tests.test_common_context import value


def session():
    return comparison.Session(value())


def finish(access):
    reading = {key: [] for key in comparison.SECTIONS}
    fact = access.initial()["situation"][0]
    reading["situation"] = [comparison.claim("observed", "Hecho de prueba", [fact])]
    return {"action": "finish", "indices": [], "reading": reading}


class FakeClient:
    config = local.Config()

    def __init__(self, answers):
        self.answers = iter(answers)
        self.bodies = []

    def preflight(self):
        from dataclasses import asdict
        return {"configuration": asdict(self.config), "model": {"digest": local.DIGEST}}

    def chat(self, body):
        self.bodies.append(body)
        answer = next(self.answers)
        if isinstance(answer, Exception):
            raise answer
        return {"model": local.MODEL, "done": True, "done_reason": "stop", "prompt_eval_count": 100,
                "message": {"role": "assistant", "content": json.dumps(answer), "thinking": "private test reasoning"}}


@pytest.mark.parametrize("endpoint", ["https://example.com:443", "http://localhost:11434", "http://127.0.0.1:11434/path",
                                    "http://127.0.0.1:11434?target=remote", "http://user:pass@127.0.0.1:11434"])
def test_only_loopback_origin_and_no_credentials(endpoint):
    with pytest.raises(ValueError):
        local.Config(endpoint=endpoint)


def test_local_query_and_reading_with_equal_initial_information():
    src = value()
    s = comparison.Session(src)
    client = FakeClient([{"action": "consult", "indices": [0], "reading": None}, finish(s.consultation())])
    result = local.interpret(s, client)
    assert result["status"] == "mechanically_valid" and result["local_calls"] == 2
    assert result["queries"][0]["index"] == 0
    assert json.loads(client.bodies[0]["messages"][1]["content"]) == s.consultation().initial()
    assert result["receipt"] == comparison.compare_local(s)["A"]["receipt"]
    assert result["external_calls"] == result["provider_cost_usd"] == 0
    assert result["tactical_quality"] == result["verification"]["tactical_quality"] == "not_evaluated"
    assert local.verify_replay(s, result)["causal_replay"]
    wire = json.dumps(client.bodies)
    assert src["packet"]["context"]["source_sha256"] not in wire and src["case_id"] not in wire


def test_invalid_citation_is_retained_and_never_repaired_or_scored():
    s = session()
    answer = finish(s.consultation())
    answer["reading"]["situation"][0]["evidence"] = ["F999:0"]
    client = FakeClient([answer])
    result = local.interpret(s, client)
    assert result["status"] == "rejected_or_incomplete" and result["reading"] is None
    assert json.loads(result["transcript"][0]["response"]["message"]["content"]) == answer
    assert len(client.bodies) == 1
    assert local.verify_replay(s, result)["citation_integrity"] is False


def test_future_query_cannot_deliver_data():
    s = session()
    client = FakeClient([{"action": "consult", "indices": [999], "reading": None}])
    result = local.interpret(s, client)
    assert result["queries"] == [] and result["response_bytes"] == 0
    assert result["status"] == "rejected_or_incomplete"


def test_offline_replay_detects_request_tampering():
    s = session()
    result = local.interpret(s, FakeClient([finish(s.consultation())]))
    changed = deepcopy(result)
    changed["transcript"][0]["request"]["messages"][1]["content"] = "more information"
    with pytest.raises(ValueError, match="causal replay"):
        local.verify_replay(s, changed)


def test_generation_length_or_http_failure_is_not_sufficient_reading():
    s = session()
    client = FakeClient([ValueError("context too long")])
    result = local.interpret(s, client)
    assert result["failure"] == "context too long" and result["local_calls"] == 1
    response = {"model": local.MODEL, "done": True, "done_reason": "length", "prompt_eval_count": 100}
    with pytest.raises(ValueError, match="generation limit"):
        local.consume(s.consultation(), response, [], client.config)


def test_query_budget_is_explicit_and_not_silent_truncation():
    s = session()
    client = FakeClient([{"action": "consult", "indices": [0] * 32, "reading": None},
                         {"action": "consult", "indices": [0], "reading": None}, finish(s.consultation())])
    result = local.interpret(s, client)
    assert len(result["queries"]) == 32
    last_user = json.loads(client.bodies[-1]["messages"][-1]["content"])
    assert last_user["consultation_results"] == [{"index": 0, "error": "query budget exhausted"}]
    assert local.verify_replay(s, result)["causal_replay"]


def test_network_redirects_and_proxies_are_not_used(monkeypatch):
    class Redirect(BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(302)
            self.send_header("Location", "https://example.com/leak")
            self.end_headers()

        def log_message(self, *args):
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Redirect)
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    monkeypatch.setenv("HTTP_PROXY", "http://invalid.example:9999")
    try:
        client = local.LocalClient(local.Config(endpoint=f"http://127.0.0.1:{server.server_port}"))
        with pytest.raises(ValueError, match="redirect forbidden"):
            client.request("GET", "/api/tags")
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


def test_model_digest_change_is_rejected_before_chat(monkeypatch):
    client = local.LocalClient()
    monkeypatch.setattr(client, "request", lambda *args: {"models": [{"name": local.MODEL, "digest": "changed"}]})
    with pytest.raises(ValueError, match="digest changed"):
        client.chat({})


def test_temp_guard_package_binding_and_offline_reader(tmp_path):
    candidate = tmp_path / "candidate.json"
    factual_io.write(candidate, value())
    factual = tmp_path / "facts"
    factual.mkdir()
    factual_io.materialize(factual, candidate_path=candidate, candidate_sha256=factual_io.sha(candidate))
    expected = factual_io.sha(factual / factual_io.MANIFEST)
    s, _ = io.load_session(factual, expected)
    output = tmp_path / "local"
    output.mkdir()
    client = FakeClient([finish(s.consultation())])
    io.materialize(output, package=factual, package_sha256=expected, client=client)
    assert io.validate(output)["status"] == "mechanically_valid"
    before = len(client.bodies)
    with pytest.raises(ValueError, match="empty"):
        io.materialize(output, package=factual, package_sha256=expected, client=client)
    assert len(client.bodies) == before
    with pytest.raises(ValueError, match="manifest differs"):
        io.load_session(factual, "0" * 64)
    result = output / io.DATA
    result.write_bytes(result.read_bytes() + b" ")
    with pytest.raises(ValueError, match="artifact bytes"):
        io.validate(output)
