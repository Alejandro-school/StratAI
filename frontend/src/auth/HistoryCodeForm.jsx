import React, { useState } from 'react';
import axios from 'axios';
import '../styles/Auth/codeForm.css';

const HistoryCodeForm = () => {
  const [authCode, setAuthCode] = useState('');
  const [knownCode, setKnownCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleGetAndSaveShareCodes = async () => {
    setLoading(true);
    try {
      const { data } = await axios.get('http://localhost:8000/steam/all-sharecodes', {
        params: { auth_code: authCode, last_code: knownCode },
        withCredentials: true,
      });
  
      const shareCodes = Array.isArray(data?.sharecodes) ? data.sharecodes : [];
      const finalKnown = shareCodes.length ? shareCodes[shareCodes.length - 1] : knownCode;
  
      await axios.post(
        'http://localhost:8000/steam/save-sharecodes',
        {
          sharecodes: shareCodes,
          auth_code: authCode,
          known_code: finalKnown,
        },
        { withCredentials: true }
      );
  
      setError('');
      window.location.href = '/dashboard';
    } catch (err) {
      console.error(err);
      setError('❌ Error al obtener o guardar los share codes');
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
              <span className="label-text">Auth Code</span>
              <span className="label-hint">Tu código de autenticación de Steam</span>
            </label>
            <input
              id="authCode"
              type="text"
              value={authCode}
              onChange={(e) => setAuthCode(e.target.value)}
              placeholder="Ej: 8TRL-ZC9DA-VHYU"
              className="form-input"
            />
          </div>

          <div className="input-group">
            <label htmlFor="knownCode">
              <span className="label-text">Último Share Code</span>
              <span className="label-hint">Código de tu partida más reciente</span>
            </label>
            <input
              id="knownCode"
              type="text"
              value={knownCode}
              onChange={(e) => setKnownCode(e.target.value)}
              placeholder="Ej: CSGO-XXXX-XXXX-XXXX"
              className="form-input"
            />
          </div>

          <button 
            className="submit-btn" 
            onClick={handleGetAndSaveShareCodes} 
            disabled={loading}
          >
            {loading ? 'Procesando...' : 'Guardar y Continuar'}
          </button>

          {error && <p className="error-message">{error}</p>}
        </div>

        {/* Columna Derecha - Video Tutorial */}
        <div className="video-container">
          <div className="video-header">
            <h3>📹 Tutorial</h3>
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
