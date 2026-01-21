
# steam_service.py
# ----------------
# Rutas de FastAPI para gestionar sharecodes en CS:GO/CS2.

import os
import json
import logging
import httpx
import asyncio
import requests
import redis.asyncio as aioredis
from fastapi import APIRouter, Request, HTTPException, Query
from typing import Any

router = APIRouter()

# Log básico
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Redis cliente (routes-level)
redis = aioredis.from_url("redis://localhost", decode_responses=True)  # type: ignore

def get_steam_data(steam_id: str) -> dict[str, Any]:
    """
    Obtiene avatar y rango (CS2 Premiere) vía API Steam.
    Lee la API key en tiempo de ejecución para evitar problemas de import.
    """
    STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")
    url_avatar = f"http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key={STEAM_API_KEY}&steamids={steam_id}"
    url_rank   = f"https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/?key={STEAM_API_KEY}&steamid={steam_id}"

    avatar_url: str | None = None
    rank: int | None = None

    # Avatar
    try:
        response = requests.get(url_avatar, timeout=5)
        if response.status_code == 200 and response.text.strip():
            avatar_response = response.json()
            if "response" in avatar_response and "players" in avatar_response["response"]:
                players = avatar_response["response"]["players"]
                if players:
                    avatar_url = players[0].get("avatarfull", None)
    except requests.exceptions.Timeout:
        logging.debug(f"Timeout al obtener avatar para {steam_id}")
    except requests.exceptions.JSONDecodeError:
        logging.debug(f"Respuesta no-JSON al obtener avatar para {steam_id}")
    except Exception as e:
        logging.debug(f"Error al obtener avatar para {steam_id}: {e}")

    # Rango CS2 Premiere
    try:
        response = requests.get(url_rank, timeout=5)
        if response.status_code == 200 and response.text.strip():
            rank_response = response.json()
            if "result" in rank_response and "rank_type" in rank_response["result"]:
                rank = rank_response["result"]["rank_type"]
    except requests.exceptions.Timeout:
        logging.debug(f"Timeout al obtener rango para {steam_id}")
    except requests.exceptions.JSONDecodeError:
        logging.debug(f"Respuesta no-JSON al obtener rango para {steam_id}")
    except Exception as e:
        logging.debug(f"Error al obtener rango para {steam_id}: {e}")

    return {"avatar": avatar_url, "rank": rank}


@router.post("/steam/save-steam-id")
async def save_steam_id(request: Request) -> dict[str, str]:
    """
    Guarda el Steam ID del usuario autenticado en Redis (set all_steam_ids).
    """
    logging.info(f"🔍 Cookies recibidas: {request.cookies}")
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    await redis.sadd("all_steam_ids", steam_id)  # type: ignore
    logging.info(f"✅ Steam ID {steam_id} agregado al set 'all_steam_ids'.")
    return {"message": "Steam ID guardado correctamente en Redis."}


@router.get("/steam/check-friend-status")
async def check_friend_status(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # 1) Cache rápida
    cached = await redis.get(f"friend_status:{steam_id}")   # friend|pending|not_friend|unknown
    cached_ts = await redis.get(f"friend_status_ts:{steam_id}")

    # 2) Intento live contra node-service
    try:
        async with httpx.AsyncClient(timeout=3.0) as client:
            response = await client.get(
                "http://localhost:4000/steam/check-friend",
                params={"steam_id": steam_id}
            )
            response.raise_for_status()
            data = response.json()
            # normalizamos
            is_friend = bool(data.get("is_friend", False))
            status = data.get("status", "unknown")
            service_down = bool(data.get("service_down", False))
            source = data.get("source", "live")

            # refresca cache si vino de live
            if source == "live":
                await redis.set(f"friend_status:{steam_id}", status, ex=86400)
                await redis.set(f"friend_status_ts:{steam_id}", 
                                data.get("cached_at") or __import__("datetime").datetime.utcnow().isoformat(), 
                                ex=86400)

            return {
                "is_friend": is_friend,
                "status": status,
                "bot_steam_id": data.get("bot_steam_id"),
                "service_down": service_down,
                "source": source,
                "cached_at": cached_ts
            }
    except httpx.RequestError as e:
        # node-service caído: devolvemos cache
        logging.warning(f"[check_friend_status] node-service caído: {e}")
        return {
            "is_friend": (cached == "friend"),
            "status": cached or "unknown",
            "service_down": True,
            "source": "cache",
            "cached_at": cached_ts
        }

@router.post("/steam/send-friend-request")
async def send_friend_request(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.post("http://localhost:4000/steam/send-friend-request",
                                     json={"steam_id": steam_id})
            resp.raise_for_status()
            data = resp.json()
            # guardamos pending en cache por UX
            await redis.set(f"friend_status:{steam_id}", data.get("status", "pending"), ex=86400)
            await redis.set(f"friend_status_ts:{steam_id}", 
                            __import__("datetime").datetime.utcnow().isoformat(), ex=86400)
            return data
    except httpx.RequestError as e:
        raise HTTPException(status_code=503, detail=f"Bot no operativo: {e}")
    except httpx.HTTPStatusError as e:
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)

@router.get("/steam/get-user-data")
async def get_user_data(request: Request) -> dict[str, Any]:
    """Obtiene datos del usuario autenticado desde Steam API (avatar y rango)."""
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    steam_data = get_steam_data(steam_id)
    return {"steam_id": steam_id, "avatar": steam_data.get("avatar"), "rank": steam_data.get("rank")}
