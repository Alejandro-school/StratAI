import hashlib
import hmac
import json
from pathlib import Path

import pytest

from backend.app.security.credentials import decrypt_credential, encrypt_credential
from backend.app.security.service_auth import (
    canonical_service_request,
    sign_service_request,
)
from backend.app.auth.dependencies import SteamUser
from backend.app.routes import sharecodes


def test_cross_language_service_auth_vector() -> None:
    vector_path = Path(__file__).parents[1] / "testdata" / "service_auth_vectors.json"
    vector = json.loads(vector_path.read_text(encoding="utf-8"))
    body = vector["body"].encode()
    canonical = canonical_service_request(
        vector["method"],
        vector["path_and_query"],
        vector["timestamp"],
        vector["nonce"],
        body,
    )

    assert hashlib.sha256(body).hexdigest() == vector["body_sha256"]
    assert sign_service_request(canonical, vector["secret"]) == vector["signature"]
    assert hmac.compare_digest(
        sign_service_request(canonical, vector["secret"]),
        vector["signature"],
    )
    assert (
        sign_service_request(canonical.replace("?x=1", "?x=2"), vector["secret"])
        != vector["signature"]
    )


def test_credential_encryption_roundtrip_and_tamper_detection() -> None:
    secret = "SENSITIVE-STEAM-HISTORY-CODE"
    envelope = encrypt_credential(secret)

    assert secret not in envelope
    assert decrypt_credential(envelope) == secret

    replacement = "A" if envelope[-1] != "A" else "B"
    with pytest.raises(ValueError):
        decrypt_credential(envelope[:-1] + replacement)


@pytest.mark.asyncio
async def test_onboarding_keeps_credentials_out_of_internal_payload(monkeypatch) -> None:
    stored: dict[str, str] = {}
    outbound: dict = {}

    class FakeRedis:
        async def hset(self, _key, mapping):
            stored.update(mapping)

        async def sadd(self, *_args):
            return 1

    async def fake_node_post(_path: str, payload: dict):
        outbound.update(payload)
        return {"status": "queued", "discovery_job_id": "job-1"}

    monkeypatch.setattr(sharecodes, "redis", FakeRedis())
    monkeypatch.setattr(sharecodes, "_node_post", fake_node_post)
    auth_code = "SENSITIVE-AUTH-CODE"
    result = await sharecodes.onboarding(
        sharecodes.OnboardingRequest(
            auth_code=auth_code,
            known_code="N/A",
        ),
        SteamUser(steam_id="76561198000000000"),
    )

    assert result["discovery_job_id"] == "job-1"
    assert auth_code not in stored["auth_code"]
    assert decrypt_credential(stored["auth_code"]) == auth_code
    assert auth_code not in json.dumps(outbound)
