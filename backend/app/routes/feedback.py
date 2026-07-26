# backend/app/routes/feedback.py
# Feedback submission and history endpoints with SQLite persistence.

import os
import time
import aiosqlite
from datetime import datetime, timezone
from typing import Any
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field

from ..auth.dependencies import SteamUser, require_steam_user

router = APIRouter()

VALID_CATEGORIES = {"bug", "sugerencia", "ux", "otro"}
DB_PATH = os.path.join(os.path.dirname(__file__), "..", "..", "data", "feedback.db")
DB_PATH = os.path.normpath(DB_PATH)
MAX_PER_HOUR = 5


class FeedbackInput(BaseModel):
    category: str = Field(..., min_length=1, max_length=20)
    message: str = Field(..., min_length=10, max_length=2000)


async def _init_db():
    """Create feedback table if it doesn't exist."""
    os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            CREATE TABLE IF NOT EXISTS feedback (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                steam_id TEXT NOT NULL,
                username TEXT,
                category TEXT NOT NULL,
                message TEXT NOT NULL,
                created_at TEXT NOT NULL,
                status TEXT NOT NULL DEFAULT 'pending'
            )
        """)
        await db.execute("""
            CREATE INDEX IF NOT EXISTS idx_feedback_steam_id
            ON feedback (steam_id)
        """)
        await db.commit()


_db_initialized = False


async def _ensure_db():
    global _db_initialized
    if not _db_initialized:
        await _init_db()
        _db_initialized = True


@router.post("/feedback")
async def submit_feedback(
    body: FeedbackInput,
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    steam_id = user.steam_id

    if body.category not in VALID_CATEGORIES:
        raise HTTPException(
            status_code=422,
            detail=f"Categoría inválida. Opciones: {', '.join(sorted(VALID_CATEGORIES))}",
        )

    await _ensure_db()

    # Rate limiting: max submissions per hour
    async with aiosqlite.connect(DB_PATH) as db:
        one_hour_ago = datetime.now(timezone.utc).isoformat(timespec="seconds")
        # SQLite datetime comparison works with ISO strings
        cursor = await db.execute(
            """
            SELECT COUNT(*) FROM feedback
            WHERE steam_id = ? AND created_at > datetime(?, '-1 hour')
            """,
            (steam_id, one_hour_ago),
        )
        (count,) = await cursor.fetchone()
        if count >= MAX_PER_HOUR:
            raise HTTPException(
                status_code=429,
                detail=f"Límite alcanzado. Máximo {MAX_PER_HOUR} envíos por hora.",
            )

        now = datetime.now(timezone.utc).isoformat(timespec="seconds")
        await db.execute(
            """
            INSERT INTO feedback (steam_id, username, category, message, created_at)
            VALUES (?, ?, ?, ?, ?)
            """,
            (steam_id, user.username, body.category, body.message, now),
        )
        await db.commit()

    return {"success": True, "message": "Feedback enviado correctamente."}


@router.get("/feedback")
async def get_feedback(
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    steam_id = user.steam_id

    await _ensure_db()

    async with aiosqlite.connect(DB_PATH) as db:
        db.row_factory = aiosqlite.Row
        cursor = await db.execute(
            """
            SELECT id, category, message, created_at, status
            FROM feedback
            WHERE steam_id = ?
            ORDER BY created_at DESC
            LIMIT 50
            """,
            (steam_id,),
        )
        rows = await cursor.fetchall()

    items = [
        {
            "id": row["id"],
            "category": row["category"],
            "message": row["message"],
            "created_at": row["created_at"],
            "status": row["status"],
        }
        for row in rows
    ]

    return {"feedback": items}
