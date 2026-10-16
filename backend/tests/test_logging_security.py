import json
import logging

import pytest

from backend.app.logging_config import JSONFormatter


@pytest.mark.parametrize("message", [
    'HTTP Request: GET https://api.steampowered.com/profile?key=SENTINEL_PRIVATE&steamids=123',
    '{"auth_code": "SENTINEL_PRIVATE", "ok": true}',
    "steamidkey=SENTINEL_PRIVATE&match=1",
    "x-service-signature: SENTINEL_PRIVATE",
    "Authorization: Bearer SENTINEL_PRIVATE",
    "Cookie: session=SENTINEL_PRIVATE; other=x",
    "Set-Cookie: session=SENTINEL_PRIVATE; HttpOnly",
    "request failed: access_token=SENTINEL_PRIVATE",
])
def test_formatter_redacts_credentials(message):
    record = logging.LogRecord("httpx", logging.WARNING, __file__, 1, "%s", (message,), None)
    output = JSONFormatter().format(record)
    assert "SENTINEL_PRIVATE" not in output
    assert "REDACTED" in json.loads(output)["message"]


def test_formatter_preserves_non_sensitive_context():
    record = logging.LogRecord("app", logging.INFO, __file__, 1, "Job %s failed", ("abc",), None)
    assert json.loads(JSONFormatter().format(record))["message"] == "Job abc failed"


def test_uvicorn_access_logs_redact_openid_callback_in_real_logger_configuration():
    import subprocess
    import sys

    probe = '''
import logging
import uvicorn
from backend.app.logging_config import configure_logging
uvicorn.Config("backend.app.main:app")
configure_logging()
logging.getLogger("uvicorn.access").info('%s - "%s %s HTTP/%s" %d',
    '127.0.0.1:1234', 'GET', '/auth/steam/callback?openid.sig=SENTINEL_PRIVATE', '1.1', 200)
'''
    result = subprocess.run([sys.executable, "-c", probe], capture_output=True, text=True, check=True)
    assert "SENTINEL_PRIVATE" not in result.stdout + result.stderr
    assert "REDACTED" in result.stdout + result.stderr
