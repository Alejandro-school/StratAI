import pytest
from fastapi import HTTPException
from redis.exceptions import ConnectionError

from backend.app.middleware.rate_limit import RateLimiter


@pytest.mark.asyncio
async def test_rate_limit_fails_closed_when_shared_store_is_unavailable():
    class UnavailableRedis:
        def pipeline(self):
            raise ConnectionError("synthetic unavailable store")

    limiter = RateLimiter("redis://127.0.0.1:1")
    limiter._redis = UnavailableRedis()
    with pytest.raises(HTTPException) as failure:
        await limiter.is_rate_limited("synthetic-user", 60, 60)
    assert failure.value.status_code == 503
