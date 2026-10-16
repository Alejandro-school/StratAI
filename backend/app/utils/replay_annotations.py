import json
import os
import threading
import time
from contextlib import contextmanager
from pathlib import Path
from typing import Any

from .user_aggregates import ensure_user_directory

_LOCK = threading.RLock()
MAX_ANNOTATIONS_PER_MATCH = 500
MAX_MATCH_FILES_PER_USER = 200
MAX_USER_BYTES = 10 * 1024 * 1024


class AnnotationQuotaExceeded(ValueError):
    pass


@contextmanager
def _user_lock(directory: Path):
    """Coordinate quota checks and mutations across threads and web workers."""
    with _LOCK:
        descriptor = os.open(directory / ".annotations.lock", os.O_CREAT | os.O_RDWR, 0o600)
        locked = False
        try:
            if os.fstat(descriptor).st_size == 0:
                os.write(descriptor, b"0")
            deadline = time.monotonic() + 5
            while not locked:
                try:
                    os.lseek(descriptor, 0, os.SEEK_SET)
                    if os.name == "nt":
                        import msvcrt
                        msvcrt.locking(descriptor, msvcrt.LK_NBLCK, 1)
                    else:
                        import fcntl
                        fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    locked = True
                except OSError:
                    if time.monotonic() >= deadline:
                        raise TimeoutError("Annotation storage busy")
                    time.sleep(0.02)
            yield
        finally:
            if locked:
                if os.name == "nt":
                    import msvcrt
                    os.lseek(descriptor, 0, os.SEEK_SET)
                    msvcrt.locking(descriptor, msvcrt.LK_UNLCK, 1)
                else:
                    import fcntl
                    fcntl.flock(descriptor, fcntl.LOCK_UN)
            os.close(descriptor)


def _annotation_path(steam_id: str, match_id: str) -> Path:
    user_dir = ensure_user_directory(str(steam_id))
    replay_dir = user_dir / "replay_annotations"
    replay_dir.mkdir(exist_ok=True)
    return replay_dir / f"{match_id}.json"


def load_annotations(steam_id: str, match_id: str) -> list[dict[str, Any]]:
    path = _annotation_path(steam_id, match_id)
    with _user_lock(path.parent):
        return _read_annotations(path)


def _read_annotations(path):
    if not path.exists():
        return []
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, list):
        raise ValueError("Invalid annotation storage")
    return payload


def save_annotations(steam_id: str, match_id: str, annotations: list[dict[str, Any]]) -> None:
    mutate_annotations(steam_id, match_id, lambda _current: annotations)


def mutate_annotations(steam_id: str, match_id: str, mutate) -> list[dict[str, Any]]:
    path = _annotation_path(steam_id, match_id)
    with _user_lock(path.parent):
        annotations = mutate(_read_annotations(path))
        data = json.dumps(annotations, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        files = list(path.parent.glob("*.json"))
        used = sum(item.stat().st_size for item in files if item != path)
        if annotations and (len(annotations) > MAX_ANNOTATIONS_PER_MATCH
                            or used + len(data) > MAX_USER_BYTES
                            or (not path.exists() and len(files) >= MAX_MATCH_FILES_PER_USER)):
            raise AnnotationQuotaExceeded("Annotation storage quota exceeded")
        if not annotations:
            path.unlink(missing_ok=True)
            return annotations
        temporary = path.with_suffix(".tmp")
        temporary.write_bytes(data)
        os.replace(temporary, path)
        return annotations
