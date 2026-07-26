import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { API_URL } from '../utils/api';
import '../styles/Auth/codeForm.css';

const HistoryCodeForm = () => {
  const navigate = useNavigate();
  const [authCode, setAuthCode] = useState('');
  const [knownCode, setKnownCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const waitForValidation = async () => {
    for (let attempt = 0; attempt < 90; attempt += 1) {
      const response = await fetch(`${API_URL}/steam/pipeline-status`, {
        credentials: 'include',
      });
      if (response.ok) {
        const status = await response.json();
        if (status.credential_status === 'configured') return;
        if (status.credential_status === 'needs_credentials') {
          throw new Error(
            'Steam ha rechazado los códigos. Genera un código de autenticación nuevo y comprueba la última partida compartida.',
          );
        }
        if (status.credential_status === 'discovery_failed') {
          const reason = status.discovery_error_code || 'pipeline_error';
          throw new Error(
            `No se pudo consultar Steam (${reason}). El trabajo agotó sus reintentos; vuelve a intentarlo.`,
          );
        }
        const terminalDiscovery = status.jobs?.find(
          (job) => job.stage === 'failed' && job.error_code === 'steam_credentials_invalid',
        );
        if (terminalDiscovery) {
          throw new Error('Las credenciales de historial ya no son válidas.');
        }
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    throw new Error(
      'La validación está tardando más de lo normal. El trabajo sigue guardado; puedes volver a intentarlo desde esta pantalla.',
    );
  };

  const handleGetAndSaveShareCodes = async () => {
    if (!authCode.trim() || !knownCode.trim()) {
      setError('Introduce ambos códigos para continuar.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const response = await fetch(`${API_URL}/steam/onboarding`, {
        method: 'POST',
        credentials: 'include',
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
      await waitForValidation();
      navigate('/bot-instructions', {
        replace: true,
        state: { discoveryJobId: data.discovery_job_id },
      });
    } catch (err) {
      setError(err.message || 'No se pudo guardar la configuración.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="codeform-page">
      <div className="codeform-wrapper">
        {/* Columna Izquierda - Formulario */}
        <div className="form-container">
          <header className="form-header">
            <h2>Configuración de Códigos</h2>
            <p className="subtitle">Ingresa tus códigos de Steam para detectar partidas automáticamente</p>
          </header>

          <div className="input-group">
            <label htmlFor="authCode">
              <span className="label-text">Código de autenticación</span>
              <span className="label-hint">Tu código de autenticación de Steam</span>
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
              placeholder="Ej: CSGO-XXXX-XXXX-XXXX"
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

          {error && <p className="error-message">{error}</p>}
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
