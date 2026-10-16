import React from 'react';
import { act, cleanup, fireEvent, render, renderHook, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import NavigationFrame from './NavigationFrame';
import useCoachChat from '../../hooks/useCoachChat';
import { makeReviewNote, reviewStorageKey, saveReviewNotes } from '../../features/replay2d/domain/coachReviewNotes';

vi.mock('../../auth/useAuth', () => ({
  useAuth: () => ({
    user: { steam_id: '76561198000000001', username: 'Synthetic account A' },
    setUser: vi.fn(),
  }),
}));

const ACCOUNT_A = { steam_id: '76561198000000001', username: 'Synthetic account A' };
const PRIVATE_MESSAGE = 'Private synthetic chat for logout regression';
const PRIVATE_REVIEW = 'Private synthetic replay correction';
const REVIEW_BUNDLE = {
  case_id: 'synthetic-security-case', round: 1, instrument: 'synthetic-review@1',
  context_sha256: 'a'.repeat(64), source_html_sha256: 'b'.repeat(64),
  variants: [{ code: 'V1', reading: {} }],
};
let navigation;

beforeEach(() => {
  sessionStorage.clear();
  // Node also exposes a localStorage global; use an isolated browser-style store.
  const values = new Map();
  const browserStorage = {
    get length() { return values.size; },
    key: (index) => [...values.keys()][index] ?? null,
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: (key) => values.delete(key),
    clear: () => values.clear(),
  };
  vi.stubGlobal('localStorage', browserStorage);
  vi.useFakeTimers();
  navigation = {
    href: '/dashboard',
    assign: vi.fn((url) => { navigation.href = url; }),
    replace: vi.fn((url) => { navigation.href = url; }),
  };
  const browserWindow = window;
  vi.stubGlobal('window', new Proxy(browserWindow, {
    get(target, key) {
      if (key === 'localStorage') return browserStorage;
      return key === 'location' ? navigation : Reflect.get(target, key, target);
    },
  }));
});

afterEach(() => {
  cleanup();
  vi.clearAllTimers();
  vi.useRealTimers();
  localStorage.clear();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  sessionStorage.clear();
});

function persistPrivateBrowserData() {
  const chat = renderHook(() => useCoachChat(ACCOUNT_A));
  act(() => chat.result.current.submitMessage(PRIVATE_MESSAGE));
  chat.unmount();
  // Legacy data can remain in a browser upgraded from the previous version.
  sessionStorage.setItem('stratai_coach_chat_history', JSON.stringify([{ text: PRIVATE_MESSAGE }]));
  const review = makeReviewNote(REVIEW_BUNDLE, 'V1', { section: 'general' },
    PRIVATE_REVIEW, 'Synthetic reason', null,
    { id: 'synthetic-note', createdAt: '2026-10-15T12:00:00Z' });
  expect(saveReviewNotes(REVIEW_BUNDLE, [review], localStorage)).toBeNull();
  localStorage.setItem('stratai_theme', 'dark');
}

async function clickLogout() {
  render(<MemoryRouter initialEntries={['/dashboard']}>
    <NavigationFrame><div>Private dashboard</div></NavigationFrame>
  </MemoryRouter>);
  fireEvent.click(screen.getByRole('button', { name: /Abrir.*usuario/ }));
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: /Cerrar sesi/ }));
  });
}

function storageValues(storage) {
  return Array.from({ length: storage.length }, (_, index) => storage.getItem(storage.key(index))).join('\n');
}

describe('logout clears private browser data only after server confirmation', () => {
  it('removes chat histories and local replay corrections on successful logout', async () => {
    persistPrivateBrowserData();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200 }));

    await clickLogout();

    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/auth/steam/logout'),
      expect.objectContaining({ method: 'POST', credentials: 'include' }));
    expect(storageValues(sessionStorage)).not.toContain(PRIVATE_MESSAGE);
    expect(sessionStorage.getItem('stratai_coach_chat_history')).toBeNull();
    expect(localStorage.getItem(reviewStorageKey(REVIEW_BUNDLE))).toBeNull();
    expect(storageValues(localStorage)).not.toContain(PRIVATE_REVIEW);
    expect(localStorage.getItem('stratai_theme')).toBe('dark');
    expect(navigation.href).toBe('/');
  });

  it.each([403, 500])('does not navigate or discard private data after HTTP %i', async (status) => {
    persistPrivateBrowserData();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status }));
    vi.spyOn(console, 'error').mockImplementation(() => {});

    await clickLogout();

    expect(navigation.href).toBe('/dashboard');
    expect(navigation.assign).not.toHaveBeenCalled();
    expect(navigation.replace).not.toHaveBeenCalled();
    expect(storageValues(sessionStorage)).toContain(PRIVATE_MESSAGE);
    expect(storageValues(localStorage)).toContain(PRIVATE_REVIEW);
  });

  it('does not pretend to log out when the network request fails', async () => {
    persistPrivateBrowserData();
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Synthetic offline error')));
    vi.spyOn(console, 'error').mockImplementation(() => {});

    await clickLogout();

    expect(navigation.href).toBe('/dashboard');
    expect(storageValues(sessionStorage)).toContain(PRIVATE_MESSAGE);
    expect(storageValues(localStorage)).toContain(PRIVATE_REVIEW);
  });
});
