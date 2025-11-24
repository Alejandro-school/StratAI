from fastapi import APIRouter, Request
from typing import Any

router = APIRouter()

@router.get("/auth/status")
async def auth_status(request: Request) -> dict[str, Any]:
    if "steam_id" in request.session:
        return {
            "authenticated": True,
            "steam_id": request.session["steam_id"],
            "username":  request.session["username"],
            "avatar":    request.session["avatar"],
        }
    return {"authenticated": False}