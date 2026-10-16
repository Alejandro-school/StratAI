const CHAT_PREFIX = 'stratai_coach_chat_history';
const REVIEW_PREFIX = 'stratai:coach-review:';
const MAX_HISTORY_BYTES = 512 * 1024;

export function readCoachHistory(owner) {
  try {
    // The old unscoped history cannot be attributed to the current account.
    sessionStorage.removeItem(CHAT_PREFIX);
    if (!owner) return [];
    const raw = sessionStorage.getItem(`${CHAT_PREFIX}:${owner}`);
    if (!raw || raw.length > MAX_HISTORY_BYTES) return [];
    const value = JSON.parse(raw);
    return Array.isArray(value) ? value.filter(message => message &&
      ['user', 'ai'].includes(message.sender) && typeof message.text === 'string' &&
      message.text.length <= 20000).slice(-100) : [];
  } catch {
    return [];
  }
}

export function saveCoachHistory(owner, messages) {
  if (!owner) return;
  try {
    const value = JSON.stringify(messages.slice(-100));
    if (value.length <= MAX_HISTORY_BYTES) sessionStorage.setItem(`${CHAT_PREFIX}:${owner}`, value);
  } catch { /* Storage can be disabled; the in-memory conversation still works. */ }
}

export function clearPrivateBrowserData() {
  for (const storageName of ['sessionStorage', 'localStorage']) {
    try {
      const storage = window[storageName];
      for (let index = storage.length - 1; index >= 0; index -= 1) {
        const key = storage.key(index);
        if (key?.startsWith(CHAT_PREFIX) || key?.startsWith(REVIEW_PREFIX)) storage.removeItem(key);
      }
    } catch { /* Browser storage may be unavailable. */ }
  }
}
