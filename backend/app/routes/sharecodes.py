import json
import logging
import re
import uuid
from typing import Any, AsyncIterator

import httpx
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException, Request, status
from pydantic import BaseModel, ConfigDict, Field, field_validator
from redis.exceptions import RedisError
from starlette.responses import StreamingResponse

from ..auth.dependencies import SteamUser, require_steam_user
from ..auth import sessions
from ..config import NODE_SERVICE_URL, PIPELINE_NAMESPACE, REDIS_URL
from ..security.credentials import encrypt_credential
from ..security.service_auth import build_service_headers

router = APIRouter()
redis = aioredis.from_url(REDIS_URL, decode_responses=True)
SHARECODE_PATTERN = re.compile(
    r"^CSGO-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}$"
)


class OnboardingRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    auth_code: str = Field(min_length=6, max_length=128)
    known_code: str = Field(min_length=1, max_length=128)

    @field_validator("auth_code", "known_code", mode="before")
    @classmethod
    def strip_value(cls, value: Any) -> Any:
        return value.strip() if isinstance(value, str) else value

    @field_validator("known_code")
    @classmethod
    def validate_known_code(cls, value: str) -> str:
        if not SHARECODE_PATTERN.fullmatch(value):
            raise ValueError("Formato de sharecode inválido")
        return value


def _json_bytes(payload: dict[str, Any]) -> bytes:
    return json.dumps(payload, separators=(",", ":"), sort_keys=True).encode()


async def _reserve_discovery(steam_id: str) -> None:
    # Shared across replicas and both entry points; a new credential version
    # must not let repeated submissions bypass the queue's deduplication.
    acquired = await redis.set(
        f"{PIPELINE_NAMESPACE}:user:{steam_id}:discovery-cooldown",
        "1",
        nx=True,
        ex=10,
    )
    if not acquired:
        raise HTTPException(
            status_code=429,
            detail="Espera unos segundos antes de volver a sincronizar el historial.",
            headers={"Retry-After": "10"},
        )


async def _node_post(path: str, payload: dict[str, Any]) -> dict[str, Any]:
    body = _json_bytes(payload)
    headers = {
        **build_service_headers("POST", path, body),
        "Content-Type": "application/json",
    }
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            response = await client.post(
                f"{NODE_SERVICE_URL}{path}",
                content=body,
                headers=headers,
            )
            response.raise_for_status()
            return response.json()
    except httpx.HTTPStatusError as exc:
        detail = "El pipeline rechazó la solicitud"
        try:
            detail = exc.response.json().get("error", detail)
        except ValueError:
            pass
        raise HTTPException(status_code=exc.response.status_code, detail=detail) from exc
    except (httpx.RequestError, ValueError) as exc:
        logging.error("Node pipeline unavailable: %s", type(exc).__name__)
        raise HTTPException(status_code=503, detail="Pipeline no disponible") from exc


@router.post("/steam/onboarding", status_code=status.HTTP_202_ACCEPTED)
async def onboarding(
    payload: OnboardingRequest,
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    credentials_key = f"{PIPELINE_NAMESPACE}:user:{user.steam_id}:credentials"
    credential_version = uuid.uuid4().hex
    try:
        await _reserve_discovery(user.steam_id)
        await redis.hset(
            credentials_key,
            mapping={
                "auth_code": encrypt_credential(payload.auth_code),
                "known_code": payload.known_code,
                "initial_code": payload.known_code,
                "status": "pending_validation",
                "validated_at": "",
                "version": credential_version,
                "discovery_error_code": "",
            },
        )
        await redis.sadd(f"{PIPELINE_NAMESPACE}:users", user.steam_id)
    except RedisError as exc:
        raise HTTPException(status_code=503, detail="No se pudo guardar la configuración.") from exc
    return await _node_post(
        "/internal/v2/discovery",
        {
            "steam_id": user.steam_id,
            "priority": 1,
            "credential_version": credential_version,
        },
    )


@router.post("/steam/discovery", status_code=status.HTTP_202_ACCEPTED)
async def discovery(user: SteamUser = Depends(require_steam_user)) -> dict[str, Any]:
    try:
        auth_code, known_code, credential_version = await redis.hmget(
            f"{PIPELINE_NAMESPACE}:user:{user.steam_id}:credentials",
            "auth_code", "known_code", "version",
        )
        if not auth_code or not known_code:
            raise HTTPException(status_code=409, detail="Conecta primero tu historial de partidas.")
        await _reserve_discovery(user.steam_id)
    except RedisError as exc:
        raise HTTPException(status_code=503, detail="Configuración no disponible.") from exc
    return await _node_post(
        "/internal/v2/discovery",
        {
            "steam_id": user.steam_id,
            "priority": 1,
            "credential_version": credential_version,
        },
    )


@router.get("/steam/pipeline-status")
async def pipeline_status(user: SteamUser = Depends(require_steam_user)) -> dict[str, Any]:
    path = f"/internal/v2/pipeline-status/{user.steam_id}"
    headers = build_service_headers("GET", path)
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get(f"{NODE_SERVICE_URL}{path}", headers=headers)
            response.raise_for_status()
            return response.json()
    except httpx.HTTPStatusError as exc:
        raise HTTPException(status_code=exc.response.status_code, detail="Estado no disponible") from exc
    except (httpx.RequestError, ValueError) as exc:
        raise HTTPException(status_code=503, detail="Pipeline no disponible") from exc


@router.get("/steam/download-progress")
async def download_progress(
    request: Request,
    user: SteamUser = Depends(require_steam_user),
) -> StreamingResponse:
    path = f"/internal/v2/events/{user.steam_id}"
    headers = build_service_headers("GET", path)
    last_event_id = request.headers.get("last-event-id")
    if last_event_id:
        headers["Last-Event-ID"] = last_event_id
    sid = request.session.get("sid")

    async def session_is_active() -> bool:
        profile = await sessions.session_store.get(sid, refresh=False)
        return bool(profile and profile["steam_id"] == user.steam_id)

    async def event_stream() -> AsyncIterator[bytes]:
        try:
            if not await session_is_active():
                return
            # Node emits a heartbeat every 25 seconds. Bound a silent upstream
            # and never let passive progress events extend the session's TTL.
            async with httpx.AsyncClient(timeout=httpx.Timeout(35.0, connect=5.0)) as client:
                async with client.stream(
                    "GET",
                    f"{NODE_SERVICE_URL}{path}",
                    headers=headers,
                ) as response:
                    response.raise_for_status()
                    async for chunk in response.aiter_bytes():
                        if not await session_is_active():
                            return
                        yield chunk
        except HTTPException:
            # Session storage failures close the stream without disclosing data.
            return
        except httpx.HTTPError as exc:
            logging.error("Progress stream failed: %s", type(exc).__name__)
            payload = json.dumps(
                {
                    "stage": "failed",
                    "error_code": "progress_stream_unavailable",
                }
            )
            yield f"event: pipeline\ndata: {payload}\n\n".encode()

    return StreamingResponse(
        event_stream(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-store",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )
