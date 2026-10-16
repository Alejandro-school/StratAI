import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import HistoryCodeForm from './HistoryCodeForm';

const sharecode = 'CSGO-AAAAA-BBBBB-CCCCC-DDDDD-EEEEE';
const response = (body, status = 200) => Promise.resolve({
  ok: status < 400, status, json: async () => body,
});

function renderForm() {
  return render(
    <MemoryRouter initialEntries={['/history-code']}>
      <Routes>
        <Route path="/history-code" element={<HistoryCodeForm />} />
        <Route path="/dashboard" element={<div>Dashboard listo</div>} />
        <Route path="/bot-instructions" element={<div>Paso de amistad obsoleto</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

function submit() {
  fireEvent.change(screen.getByLabelText(/Código de autenticación/), { target: { value: 'AUTH-CODE-TEST' } });
  fireEvent.change(screen.getByLabelText(/Último código compartido/), { target: { value: sharecode } });
  fireEvent.click(screen.getByRole('button', { name: 'Guardar y Continuar' }));
}

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe('History setup without Steam friendship', () => {
  it('validates the history and enters the dashboard directly', async () => {
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response({ status: 'queued', discovery_job_id: 'job-1' }, 202))
      .mockImplementationOnce(() => response({ configured: true, credential_status: 'configured' }));
    vi.stubGlobal('fetch', fetchMock);
    renderForm();
    submit();

    expect(await screen.findByText('Dashboard listo')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      credentials: 'include', method: 'POST',
      body: JSON.stringify({ auth_code: 'AUTH-CODE-TEST', known_code: sharecode }),
    });
  });

  it('stops immediately when the session has expired during validation', async () => {
    vi.stubGlobal('fetch', vi.fn()
      .mockImplementationOnce(() => response({ status: 'queued' }, 202))
      .mockImplementationOnce(() => response({ detail: 'Usuario no autenticado.' }, 401)));
    renderForm();
    submit();

    expect(await screen.findByRole('alert')).toHaveTextContent(/sesión/i);
    expect(screen.queryByText('Dashboard listo')).not.toBeInTheDocument();
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('cancels validation requests when leaving the form', async () => {
    let validationSignal;
    vi.stubGlobal('fetch', vi.fn()
      .mockImplementationOnce(() => response({ status: 'queued' }, 202))
      .mockImplementationOnce((_url, options) => {
        validationSignal = options?.signal;
        return new Promise(() => {});
      }));
    const view = renderForm();
    submit();
    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
    view.unmount();
    expect(validationSignal?.aborted).toBe(true);
  });
});
