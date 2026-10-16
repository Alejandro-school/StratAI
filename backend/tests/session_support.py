from backend.app.auth import sessions


class MemorySessions:
    def __init__(self):
        self.values = {}

    async def set(self, key, value, **_options):
        if key in self.values:
            return False
        self.values[key] = value
        return True

    async def get(self, key):
        return self.values.get(key)

    async def expire(self, key, _seconds):
        return key in self.values

    async def delete(self, key):
        return int(self.values.pop(key, None) is not None)


def install_session_store(monkeypatch):
    store = sessions.SessionStore(MemorySessions())
    monkeypatch.setattr(sessions, "session_store", store)
    return store
