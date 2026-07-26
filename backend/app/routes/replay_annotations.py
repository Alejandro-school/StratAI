from datetime import datetime, timezone
from typing import Literal, Optional
from uuid import uuid4

from fastapi import APIRouter, Depends, HTTPException, Response
from pydantic import BaseModel, ConfigDict, Field

from ..auth.dependencies import SteamUser, require_steam_user
from ..middleware.security import sanitize_match_id
from ..utils.replay_annotations import load_annotations, save_annotations

router = APIRouter(tags=["replay-annotations"])


class AnnotationPoint(BaseModel):
    model_config = ConfigDict(extra="forbid")

    x: float = Field(ge=-20000, le=20000)
    y: float = Field(ge=-20000, le=20000)


class AnnotationCreate(BaseModel):
    model_config = ConfigDict(extra="forbid")

    round: int = Field(ge=1, le=100)
    start_tick: int = Field(ge=0)
    end_tick: int = Field(ge=0)
    type: Literal["arrow", "circle", "freehand", "note"]
    points: list[AnnotationPoint] = Field(min_length=1, max_length=500)
    text: str = Field(default="", max_length=500)
    color: str = Field(default="#63d7ff", pattern=r"^#[0-9a-fA-F]{6}$")

class AnnotationPatch(BaseModel):
    model_config = ConfigDict(extra="forbid")

    start_tick: Optional[int] = Field(default=None, ge=0)
    end_tick: Optional[int] = Field(default=None, ge=0)
    type: Optional[Literal["arrow", "circle", "freehand", "note"]] = None
    points: Optional[list[AnnotationPoint]] = Field(default=None, min_length=1, max_length=500)
    text: Optional[str] = Field(default=None, max_length=500)
    color: Optional[str] = Field(default=None, pattern=r"^#[0-9a-fA-F]{6}$")

class Annotation(AnnotationCreate):
    id: str
    created_at: str
    updated_at: str


def _match_id(raw_match_id: str) -> str:
    try:
        sanitized = sanitize_match_id(raw_match_id)
    except ValueError as error:
        raise HTTPException(status_code=400, detail="Invalid match_id") from error
    if sanitized != raw_match_id:
        raise HTTPException(status_code=400, detail="Invalid match_id")
    return sanitized


def _model_dict(model: BaseModel, *, exclude_unset: bool = False) -> dict:
    if hasattr(model, "model_dump"):
        return model.model_dump(exclude_unset=exclude_unset)
    return model.dict(exclude_unset=exclude_unset)


def _validate_range(annotation: dict) -> None:
    if annotation["end_tick"] < annotation["start_tick"]:
        raise HTTPException(status_code=422, detail="end_tick must be greater than or equal to start_tick")


@router.get("/match/{match_id}/replay/annotations", response_model=list[Annotation])
async def get_replay_annotations(
    match_id: str,
    user: SteamUser = Depends(require_steam_user),
) -> list[dict]:
    return load_annotations(user.steam_id, _match_id(match_id))


@router.post("/match/{match_id}/replay/annotations", response_model=Annotation, status_code=201)
async def create_replay_annotation(
    match_id: str,
    payload: AnnotationCreate,
    user: SteamUser = Depends(require_steam_user),
) -> dict:
    steam_id = user.steam_id
    safe_match_id = _match_id(match_id)
    annotation = _model_dict(payload)
    _validate_range(annotation)
    now = datetime.now(timezone.utc).isoformat()
    annotation.update(id=str(uuid4()), created_at=now, updated_at=now)
    annotations = load_annotations(steam_id, safe_match_id)
    annotations.append(annotation)
    save_annotations(steam_id, safe_match_id, annotations)
    return annotation


@router.patch("/match/{match_id}/replay/annotations/{annotation_id}", response_model=Annotation)
async def update_replay_annotation(
    match_id: str,
    annotation_id: str,
    payload: AnnotationPatch,
    user: SteamUser = Depends(require_steam_user),
) -> dict:
    steam_id = user.steam_id
    safe_match_id = _match_id(match_id)
    annotations = load_annotations(steam_id, safe_match_id)
    index = next((i for i, item in enumerate(annotations) if item.get("id") == annotation_id), None)
    if index is None:
        raise HTTPException(status_code=404, detail="Annotation not found")
    updated = {**annotations[index], **_model_dict(payload, exclude_unset=True)}
    _validate_range(updated)
    updated["updated_at"] = datetime.now(timezone.utc).isoformat()
    annotations[index] = updated
    save_annotations(steam_id, safe_match_id, annotations)
    return updated


@router.delete("/match/{match_id}/replay/annotations/{annotation_id}", status_code=204)
async def delete_replay_annotation(
    match_id: str,
    annotation_id: str,
    user: SteamUser = Depends(require_steam_user),
) -> Response:
    steam_id = user.steam_id
    safe_match_id = _match_id(match_id)
    annotations = load_annotations(steam_id, safe_match_id)
    remaining = [item for item in annotations if item.get("id") != annotation_id]
    if len(remaining) == len(annotations):
        raise HTTPException(status_code=404, detail="Annotation not found")
    save_annotations(steam_id, safe_match_id, remaining)
    return Response(status_code=204)
