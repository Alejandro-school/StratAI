import json
import os
import threading
from pathlib import Path
from typing import Any

from .user_aggregates import ensure_user_directory

_LOCK = threading.RLock()


def _annotation_path(steam_id: str, match_id: str) -> Path:
    user_dir = ensure_user_directory(str(steam_id))
    replay_dir = user_dir / "replay_annotations"
    replay_dir.mkdir(exist_ok=True)
    return replay_dir / f"{match_id}.json"


def load_annotations(steam_id: str, match_id: str) -> list[dict[str, Any]]:
    path = _annotation_path(steam_id, match_id)
    with _LOCK:
        if not path.exists():
            return []
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return []
    return payload if isinstance(payload, list) else []


def save_annotations(steam_id: str, match_id: str, annotations: list[dict[str, Any]]) -> None:
    path = _annotation_path(steam_id, match_id)
    temporary = path.with_suffix(".tmp")
    with _LOCK:
        temporary.write_text(
            json.dumps(annotations, ensure_ascii=False, separators=(",", ":")),
            encoding="utf-8",
        )
        os.replace(temporary, path)
