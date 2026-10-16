import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { API_URL } from '../utils/api';
import '../styles/Auth/codeForm.css';

const HistoryCodeForm = () => {
  const navigate = useNavigate();
  const [authCode, setAuthCode] = useState('');
  const [knownCode, setKnownCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const requestRef = useRef(null);

  useEffect(() => () => requestRef.current?.abort(), []);

  const waitForValidation = async (signal) => {
    for (let attempt = 0; attempt < 90; attempt += 1) {
      signal.throwIfAborted();
      const response = await fetch(`${API_URL}/steam/pipeline-status`, {
        credentials: 'include',
        signal,
      });
      if (response.status === 401) {
        throw new Error('Tu sesión ha caducado. Vuelve a iniciar sesión con Steam.');
      }
      if (!response.ok) {
        throw new Error('No se pudo comprobar el historial. Inténtalo de nuevo en unos instantes.');
      }
      const status = await response.json();
      if (status.configured) return;
      if (status.credential_status === 'needs_credentials') {
        throw new Error(
          'Steam ha rechazado los códigos. Genera un código de autenticación nuevo y comprueba la última partida compartida.',
        );
      }
      if (status.credential_status === 'discovery_failed') {
        throw new Error('No se pudo consultar Steam. Vuelve a intentarlo en unos instantes.');
      }
      const terminalDiscovery = status.jobs?.find(
        (job) => job.stage === 'failed' && job.error_code === 'steam_credentials_invalid',
      );
      if (terminalDiscovery) {
        throw new Error('Las credenciales de historial ya no son válidas.');
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    throw new Error(
      'La validación está tardando más de lo normal. El trabajo sigue guardado; puedes volver a intentarlo desde esta pantalla.',
    );
  };

  const handleGetAndSaveShareCodes = async () => {
    if (requestRef.current) return;
    if (!authCode.trim() || !knownCode.trim()) {
      setError('Introduce ambos códigos para continuar.');
      return;
    }
    setLoading(true);
    setError('');
    const controller = new AbortController();
    requestRef.current = controller;
    try {
      const response = await fetch(`${API_URL}/steam/onboarding`, {
        method: 'POST',
        credentials: 'include',
        signal: controller.signal,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          auth_code: authCode.trim(),
          known_code: knownCode.trim(),
        }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        const detail = Array.isArray(data.detail)
          ? data.detail.map((item) => item.msg).join('. ')
          : data.detail || data.error;
        throw new Error(detail || 'No se pudo iniciar el descubrimiento.');
      }
      await waitForValidation(controller.signal);
      if (!controller.signal.aborted) navigate('/dashboard', { replace: true });
    } catch (err) {
      if (!controller.signal.aborted) {
        setError(err.message || 'No se pudo guardar la configuración.');
      }
    } finally {
      if (!controller.signal.aborted) setLoading(false);
      if (requestRef.current === controller) requestRef.current = null;
    }
  };

  return (
    <div className="codeform-page">
      <div className="codeform-wrapper">
        {/* Columna Izquierda - Formulario */}
        <div className="form-container">
          <header className="form-header">
            <h2>Conecta tu historial de partidas</h2>
            <p className="subtitle">Conecta tu última partida y sincroniza las siguientes automáticamente.</p>
          </header>

          <div className="input-group">
            <label htmlFor="authCode">
              <span className="label-text">Código de autenticación</span>
              <span className="label-hint">Código de acceso al historial de CS2, no tu contraseña ni un código de Steam Guard</span>
            </label>
            <input
              id="authCode"
              type="password"
              value={authCode}
              onChange={(e) => setAuthCode(e.target.value)}
              placeholder="Ej: 8TRL-ZC9DA-VHYU"
              className="form-input"
              autoComplete="new-password"
            />
          </div>

          <div className="input-group">
            <label htmlFor="knownCode">
              <span className="label-text">Último código compartido</span>
              <span className="label-hint">Código de tu partida más reciente</span>
            </label>
            <input
              id="knownCode"
              type="text"
              value={knownCode}
              onChange={(e) => setKnownCode(e.target.value)}
              placeholder="CSGO-XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"
              className="form-input"
              autoComplete="off"
            />
          </div>

          <button 
            className="submit-btn" 
            onClick={handleGetAndSaveShareCodes} 
            disabled={loading || !authCode.trim() || !knownCode.trim()}
          >
            {loading ? 'Procesando...' : 'Guardar y Continuar'}
          </button>

          {error && <p role="alert" className="error-message">{error}</p>}
        </div>

        {/* Columna Derecha - Video Tutorial */}
        <div className="video-container">
          <div className="video-header">
            <h3>Tutorial</h3>
            <p>Aprende cómo obtener tus códigos</p>
          </div>
          <video 
            autoPlay
            loop
            muted
            playsInline
            controls 
            className="tutorial-video"
          >
            <source src="/videos/Instruccions.mp4" type="video/mp4" />
            Tu navegador no soporta video HTML5.
          </video>
        </div>
      </div>
    </div>
  );
};

export default HistoryCodeForm;
