"""Revocable sessions; cookies contain an opaque identifier, never profile data."""
import hashlib
import json
import math
import re
import secrets
import time

import redis.asyncio as aioredis
from fastapi import HTTPException
from redis.exceptions import RedisError

from ..config import PIPELINE_NAMESPACE, REDIS_URL, SESSION_MAX_AGE, SESSION_ABSOLUTE_MAX_AGE


class SessionStore:
    def __init__(self, redis_client, idle_seconds=SESSION_MAX_AGE,
                 absolute_seconds=SESSION_ABSOLUTE_MAX_AGE, clock=time.time):
        self.redis = redis_client
        self.idle_seconds = idle_seconds
        self.absolute_seconds = absolute_seconds
        self.clock = clock

    def _key(self, sid):
        if not isinstance(sid, str) or not re.fullmatch(r"[A-Za-z0-9_-]{43}", sid):
            return None
        return f"{PIPELINE_NAMESPACE}:session:{hashlib.sha256(sid.encode()).hexdigest()}"

    async def create(self, steam_id, username="", avatar=""):
        sid = secrets.token_urlsafe(32)
        value = json.dumps({"steam_id": steam_id, "username": username, "avatar": avatar,
                            "expires_at": self.clock() + self.absolute_seconds})
        try:
            created = await self.redis.set(self._key(sid), value, nx=True,
                                           ex=min(self.idle_seconds, self.absolute_seconds))
        except RedisError as error:
            raise HTTPException(503, "Servicio de sesiones no disponible.") from error
        if not created:
            raise HTTPException(503, "No se pudo crear la sesión.")
        return sid

    async def get(self, sid, *, refresh=True):
        key = self._key(sid)
        if key is None:
            return None
        try:
            raw = await self.redis.get(key)
            if raw is None:
                return None
            try:
                value = json.loads(raw)
                remaining = float(value["expires_at"]) - self.clock()
                valid = (math.isfinite(remaining) and remaining > 0
                         and re.fullmatch(r"\d{17}", value["steam_id"]) is not None)
            except (ValueError, TypeError, KeyError):
                valid = False
            if not valid:
                await self.redis.delete(key)
                return None
            if not refresh:
                return value
            # Expiry refresh cannot recreate a session deleted by logout.
            alive = await self.redis.expire(key, max(1, min(self.idle_seconds, math.ceil(remaining))))
            return value if alive else None
        except RedisError as error:
            raise HTTPException(503, "Servicio de sesiones no disponible.") from error

    async def revoke(self, sid):
        key = self._key(sid)
        if key is None:
            return
        try:
            await self.redis.delete(key)
        except RedisError as error:
            raise HTTPException(503, "No se pudo revocar la sesión.") from error


session_store = SessionStore(aioredis.from_url(REDIS_URL, decode_responses=True))


async def session_profile(request):
    profile = await session_store.get(request.session.get("sid"))
    if profile is None and request.session.get("steam_id"):
        # Legacy cookies have no revocation record and require a fresh login.
        request.session.clear()
    return profile
