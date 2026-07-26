import hashlib
import hmac
import time
import uuid

from ..config import INTERNAL_SERVICE_SECRET

SIGNATURE_VERSION = "v1"


def canonical_service_request(
    method: str,
    path_and_query: str,
    timestamp: str,
    nonce: str,
    body: bytes = b"",
) -> str:
    body_hash = hashlib.sha256(body).hexdigest()
    return "\n".join(
        [
            SIGNATURE_VERSION,
            method.upper(),
            path_and_query,
            timestamp,
            nonce,
            body_hash,
        ]
    )


def sign_service_request(canonical: str, secret: str = INTERNAL_SERVICE_SECRET) -> str:
    return hmac.new(secret.encode(), canonical.encode(), hashlib.sha256).hexdigest()


def build_service_headers(method: str, path_and_query: str, body: bytes = b"") -> dict[str, str]:
    timestamp = str(int(time.time()))
    nonce = uuid.uuid4().hex
    canonical = canonical_service_request(method, path_and_query, timestamp, nonce, body)
    signature = sign_service_request(canonical)
    return {
        "X-Service-Version": SIGNATURE_VERSION,
        "X-Service-Timestamp": timestamp,
        "X-Service-Nonce": nonce,
        "X-Service-Signature": signature,
    }
