"""
Redis-based rate limiter for FastAPI.
Uses a sliding window counter approach with Redis.
No external dependencies beyond redis.asyncio (already used).
"""
import time
import logging
from typing import Any, Callable, Optional, cast
from functools import wraps
from fastapi import Request, HTTPException
import redis.asyncio as aioredis


class RateLimiter:
    """Simple sliding-window rate limiter backed by Redis."""

    def __init__(self, redis_url: str = "redis://localhost:6379") -> None:
        self._redis: aioredis.Redis = aioredis.from_url(redis_url, decode_responses=True)  # type: ignore[assignment]

    async def is_rate_limited(
        self, key: str, max_requests: int, window_seconds: int
    ) -> tuple[bool, int]:
        """
        Check if a key has exceeded the rate limit.
        Returns (is_limited, remaining_requests).
        """
        now = time.time()
        window_start = now - window_seconds
        pipe_key = f"ratelimit:{key}"

        try:
            pipe = self._redis.pipeline()
            # Remove old entries outside the window
            pipe.zremrangebyscore(pipe_key, 0, window_start)
            # Count current entries
            pipe.zcard(pipe_key)
            # Add current request
            pipe.zadd(pipe_key, {str(now): now})
            # Set expiry on the key
            pipe.expire(pipe_key, window_seconds)
            results: list[Any] = cast(list[Any], await pipe.execute())

            current_count: int = int(results[1])
            _ = max(0, max_requests - current_count - 1)

            if current_count >= max_requests:
                return True, 0
            return False, max_requests - int(current_count) - 1
        except Exception as e:
            logging.warning(f"Rate limiter Redis error: {e}. Allowing request.")
            return False, max_requests  # Fail open

    def limit(self, max_requests: int, window_seconds: int, key_func: Optional[Callable[[Request], str]] = None) -> Callable[..., Any]:
        """
        Decorator for FastAPI route handlers.

        Usage:
            @router.get("/login")
            @rate_limiter.limit(5, 60)  # 5 requests per 60 seconds
            async def login(request: Request):
                ...
        """
        def decorator(func: Callable[..., Any]) -> Callable[..., Any]:
            @wraps(func)
            async def wrapper(*args: Any, **kwargs: Any) -> Any:
                request: Optional[Request] = kwargs.get("request")
                if request is None:
                    for arg in args:
                        if isinstance(arg, Request):
                            request = arg
                            break

                if request is None:
                    return await func(*args, **kwargs)

                # Build rate limit key
                if key_func:
                    rate_key = key_func(request)
                else:
                    # Use steam_id for authenticated users (per-user), IP for unauthenticated (per-IP)
                    steam_id = request.session.get("steam_id") if hasattr(request, "session") else None
                    if steam_id:
                        rate_key = f"{func.__name__}:user:{steam_id}"
                    else:
                        client_ip = request.client.host if request.client else "unknown"
                        rate_key = f"{func.__name__}:{client_ip}"

                is_limited, _ = await self.is_rate_limited(
                    rate_key, max_requests, window_seconds
                )

                if is_limited:
                    raise HTTPException(
                        status_code=429,
                        detail=f"Rate limit exceeded. Try again in {window_seconds} seconds.",
                    )

                response = await func(*args, **kwargs)
                return response

            return wrapper
        return decorator


# Singleton instance - import this in route files
_limiter = None

def get_rate_limiter(redis_url: str = "redis://localhost:6379") -> RateLimiter:
    global _limiter
    if _limiter is None:
        _limiter = RateLimiter(redis_url)
    return _limiter
