import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import useCoachChat from './useCoachChat';

const ACCOUNT_A = { steam_id: '76561198000000001', username: 'Synthetic account A' };
const ACCOUNT_B = { steam_id: '76561198000000002', username: 'Synthetic account B' };
const PRIVATE_MESSAGE = 'Private coaching note owned by synthetic account A';

beforeEach(() => {
  sessionStorage.clear();
  vi.useFakeTimers();
});

afterEach(() => {
  cleanup();
  vi.clearAllTimers();
  vi.useRealTimers();
  sessionStorage.clear();
});

describe('coach chat account isolation', () => {
  it('does not restore legacy history whose owner cannot be established', () => {
    sessionStorage.setItem('stratai_coach_chat_history', JSON.stringify([
      { id: 'legacy-private-note', sender: 'user', text: PRIVATE_MESSAGE },
    ]));

    const { result } = renderHook(() => useCoachChat(ACCOUNT_B));

    expect(result.current.messages).toEqual([]);
  });

  it('preserves the same account history when the dashboard remounts', () => {
    const first = renderHook(() => useCoachChat(ACCOUNT_A));
    act(() => first.result.current.submitMessage(PRIVATE_MESSAGE));
    first.unmount();

    const restored = renderHook(() => useCoachChat({ ...ACCOUNT_A }));

    expect(restored.result.current.messages.some((message) => message.text === PRIVATE_MESSAGE)).toBe(true);
  });

  it('does not restore account A history when account B mounts in the same tab', () => {
    const first = renderHook(() => useCoachChat(ACCOUNT_A));
    act(() => first.result.current.submitMessage(PRIVATE_MESSAGE));
    first.unmount();

    const second = renderHook(() => useCoachChat(ACCOUNT_B));

    expect(second.result.current.messages).toEqual([]);
  });

  it('never renders the previous identity history while changing accounts', () => {
    const renders = [];
    const { result, rerender } = renderHook(({ user }) => {
      const chat = useCoachChat(user);
      renders.push({ owner: user?.steam_id, texts: chat.messages.map((message) => message.text) });
      return chat;
    }, { initialProps: { user: ACCOUNT_A } });
    act(() => result.current.submitMessage(PRIVATE_MESSAGE));

    rerender({ user: ACCOUNT_B });

    expect(result.current.messages).toEqual([]);
    expect(renders.filter((render) => render.owner === ACCOUNT_B.steam_id)
      .every((render) => !render.texts.includes(PRIVATE_MESSAGE))).toBe(true);
  });

  it('hides private messages as soon as the authenticated identity is cleared', () => {
    const { result, rerender } = renderHook(({ user }) => useCoachChat(user), {
      initialProps: { user: ACCOUNT_A },
    });
    act(() => result.current.submitMessage(PRIVATE_MESSAGE));

    rerender({ user: null });

    expect(result.current.messages).toEqual([]);
    expect(result.current.isAiTyping).toBe(false);
  });

  it('does not deliver a pending account A analysis into account B conversation', () => {
    const { result, rerender } = renderHook(({ user }) => useCoachChat(user), {
      initialProps: { user: ACCOUNT_A },
    });
    act(() => result.current.sendProactiveGreeting([
      { map_name: 'de_synthetic_private_map', result: 'W' },
    ]));
    expect(result.current.isAiTyping).toBe(true);

    rerender({ user: ACCOUNT_B });
    act(() => vi.advanceTimersByTime(600));

    expect(result.current.messages).toEqual([]);
    expect(result.current.isAiTyping).toBe(false);
  });
});
