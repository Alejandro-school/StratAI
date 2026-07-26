import base64
import binascii
import hashlib
import os

from cryptography.exceptions import InvalidTag
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

from ..config import CREDENTIAL_ENCRYPTION_KEY

VERSION = "v1"


def _key() -> bytes:
    return hashlib.sha256(CREDENTIAL_ENCRYPTION_KEY.encode()).digest()


def encrypt_credential(value: str) -> str:
    nonce = os.urandom(12)
    ciphertext = AESGCM(_key()).encrypt(nonce, value.encode(), VERSION.encode())
    payload = base64.urlsafe_b64encode(nonce + ciphertext).decode().rstrip("=")
    return f"{VERSION}.{payload}"


def decrypt_credential(envelope: str) -> str:
    try:
        version, payload = envelope.split(".", 1)
        if version != VERSION:
            raise ValueError("Unsupported credential envelope version")
        padded = payload + "=" * (-len(payload) % 4)
        raw = base64.urlsafe_b64decode(padded)
        if len(raw) < 29:
            raise ValueError("Invalid credential envelope")
        return AESGCM(_key()).decrypt(raw[:12], raw[12:], VERSION.encode()).decode()
    except (InvalidTag, UnicodeDecodeError, binascii.Error) as exc:
        raise ValueError("Invalid credential envelope") from exc
