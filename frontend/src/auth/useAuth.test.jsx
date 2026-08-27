import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import { afterEach, vi } from 'vitest';
import { AuthProvider, useAuth } from './useAuth';

const SessionConsumer = () => {
  const { user } = useAuth();
  return <div>{user?.username || 'Sin sesión'}</div>;
};

describe('AuthProvider', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps checking until the active StrictMode request finishes', async () => {
    let resolveActiveRequest;
    let requestCount = 0;

    globalThis.fetch = vi.fn((_url, options) => {
      requestCount += 1;
      if (requestCount === 1) {
        return new Promise((_resolve, reject) => {
          options.signal.addEventListener('abort', () => {
            reject(new DOMException('Aborted', 'AbortError'));
          });
        });
      }

      return new Promise((resolve) => {
        resolveActiveRequest = () => resolve({
          ok: true,
          json: async () => ({ authenticated: true, username: 'Kerchak' }),
        });
      });
    });

    render(
      <React.StrictMode>
        <AuthProvider>
          <SessionConsumer />
        </AuthProvider>
      </React.StrictMode>,
    );

    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(2));
    expect(screen.getByRole('status')).toHaveTextContent('Comprobando sesión');
    expect(screen.queryByText('Sin sesión')).not.toBeInTheDocument();

    await act(async () => resolveActiveRequest());
    expect(await screen.findByText('Kerchak')).toBeInTheDocument();
  });
});
