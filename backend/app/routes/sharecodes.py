
# backend/app/routes/sharecodes.py
# --------------------------------
# Rutas de FastAPI para gestionar sharecodes en CS:GO/CS2.

import os
import logging
import httpx
import asyncio
import redis.asyncio as aioredis
from typing import Any
from fastapi import APIRouter, Request, HTTPException, Query

router = APIRouter()

# Redis (usar inyección de dependencias sería mejor, pero mantenemos simple para refactor)
redis = aioredis.from_url("redis://localhost", decode_responses=True)

@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(
    request: Request,
    auth_code: str = Query(..., alias="auth_code"),
    last_code: str | None = Query(None, alias="last_code"),
    known_code: str | None = Query(None, alias="known_code"),
) -> dict[str, Any]:
    """
    Obtiene sharecodes siguientes usando la API oficial de Steam.
    Reglas:
      - last_code/known_code es OBLIGATORIO si no hay uno previo en Redis.
      - Si ya hay uno guardado en Redis, puede omitirse en la query.
      - Un único fallback 'N/A' si Steam responde 412 para resincronizar.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Cargar API key en tiempo de petición (evita problemas de import/orden)
    STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")
    if not STEAM_API_KEY:
        raise HTTPException(status_code=400, detail="Falta Steam API Key")

    # Normaliza (sin cambiar mayúsculas/minúsculas)
    auth_code = auth_code.strip()

    # 1) Usa param de query si viene
    provided_code = (last_code or known_code)
    if provided_code:
        current_code = provided_code.strip()
    else:
        # 2) Si no viene, usa el último guardado en Redis
        saved_code = await redis.get(f"{steam_id}:knownCode")
        if saved_code:
            current_code = saved_code.strip()
        else:
            # 3) No hay en query ni en Redis -> obligatorio
            raise HTTPException(
                status_code=400,
                detail="Falta last_code/known_code y no hay valor previo en Redis. Indica tu último share code."
            )

    # Control
    sharecodes: list[str] = []
    max_retries = 5
    delay_seconds = 0.2
    max_total_codes = 50
    original_code = current_code
    resynced = False
    
    # data: dict[str, Any] = {} # declared but initialized inside loop logic typically or implicitly

    logging.info(f"[all-sharecodes] Usuario={steam_id} | code_inicial='{current_code}'")

    try:
        async with httpx.AsyncClient() as client:
            while True:
                success = False
                data = {}
                for attempt in range(max_retries):
                    # params en cada intento (respeta current_code actualizado)
                    params: dict[str, str] = {
                        "key": STEAM_API_KEY,
                        "steamid": steam_id,
                        "steamidkey": auth_code,
                        "knowncode": current_code,
                    }

                    resp = await client.get(
                        "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/",
                        params=params
                    )

                    logging.info(f"[all-sharecodes] HTTP {resp.status_code} con knowncode='{current_code}'")

                    # 412 -> code inválido/caducado: probar una sola vez con 'N/A'
                    if resp.status_code == 412:
                        logging.warning(
                            f"[all-sharecodes] 412 con knowncode='{current_code}'. "
                            f"Body: {resp.text[:300]}"
                        )
                        if not resynced and current_code != "N/A":
                            current_code = "N/A"
                            resynced = True
                            continue
                        raise HTTPException(
                            status_code=400,
                            detail=(
                                "Auth/ShareCode inválido: verifica que el AUTH CODE corresponde "
                                "a este SteamID y vuelve a generarlo en Steam Soporte. "
                                "Si el problema persiste, copia el último share code exacto desde Steam."
                            )
                        )

                    # 429 -> backoff incremental
                    if resp.status_code == 429:
                        wait = delay_seconds * (attempt + 1)
                        logging.warning(f"[all-sharecodes] 429 (rate limit). Reintentando en {wait:.2f}s…")
                        await asyncio.sleep(wait)
                        continue

                    resp.raise_for_status()
                    data = resp.json()
                    success = True
                    break

                if not success:
                    raise HTTPException(
                        status_code=429,
                        detail="Límite de reintentos alcanzado. Intente en unos minutos."
                    )

                next_code = data.get("result", {}).get("nextcode", "").strip()

                # Fin si vacío/nulo/N/A
                if not next_code or next_code.lower() in ("n/a", "null", "none"):
                    logging.info(f"[all-sharecodes] FIN | Usuario={steam_id} | motivo=respuesta vacía/nula")
                    break

                # Añadir y avanzar
                sharecodes.append(next_code)
                current_code = next_code

                # Límite de seguridad
                if len(sharecodes) >= max_total_codes:
                    logging.info(f"[all-sharecodes] LÍMITE | Usuario={steam_id} | códigos={max_total_codes}")
                    break

                await asyncio.sleep(delay_seconds)

        # Actualiza Redis con el último estado
        if sharecodes:
            await redis.set(f"{steam_id}:authCode", auth_code)
            await redis.set(f"{steam_id}:knownCode", sharecodes[-1])
            logging.info(f"[all-sharecodes] REDIS UPDATE | Usuario={steam_id} | nuevo_knownCode={sharecodes[-1]}")
        else:
            await redis.set(f"{steam_id}:knownCode", original_code)
            logging.info(f"[all-sharecodes] REDIS PRESERVE | Usuario={steam_id} | knownCode={original_code}")

        return {
            "sharecodes": sharecodes,
            "has_more": len(sharecodes) == max_total_codes
        }

    except httpx.HTTPError as e:
        logging.error(f"[all-sharecodes] ERROR HTTP | Usuario={steam_id} | {str(e)}")
        raise HTTPException(
            status_code=502,
            detail="Error de comunicación con los servidores de Steam. Intente nuevamente."
        )


@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request) -> dict[str, str]:
    """
    Recibe sharecodes, auth_code y known_code, y los almacena en Redis.
    El bot Node.js se activa automáticamente al detectar el rpush.
    """
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    auth_code = data.get("auth_code")
    known_code = data.get("known_code")

    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron sharecodes.")
    if not auth_code or not known_code:
        raise HTTPException(status_code=400, detail="Faltan auth_code o known_code.")

    # Actualiza lista de sharecodes del usuario
    await redis.delete(f"sharecodes:{steam_id}")
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)  # type: ignore

    # Guarda auth_code y known_code
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:knownCode", known_code)

    logging.info(f"[save-sharecodes] Almacenados {len(sharecodes)} sharecodes en Redis para {steam_id}")
    return {"message": "✅ ShareCodes guardados. El bot se activará automáticamente."}


@router.get("/steam/check-sharecodes")
async def check_sharecodes(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    existing_sharecodes: list[str] = await redis.lrange(f"sharecodes:{steam_id}", 0, -1)  # type: ignore
    return {"exists": bool(existing_sharecodes), "sharecodes": existing_sharecodes}  # type: ignore
