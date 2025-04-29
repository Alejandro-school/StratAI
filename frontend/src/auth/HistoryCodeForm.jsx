import React, { useState } from 'react';
import axios from 'axios';
import '../styles/Auth/codeForm.css';

const HistoryCodeForm = () => {
  // Renombramos "knowCode" a "knownCode"
  const [authCode, setAuthCode] = useState('');
  const [knownCode, setKnownCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showAuthGuide, setShowAuthGuide] = useState(false);
  const [showLastGuide, setShowLastGuide] = useState(false);

  const handleGetAndSaveShareCodes = async () => {
    setLoading(true);
    try {
      // Ajustamos el nombre del parámetro de la query y el body a "known_code"
      const response = await axios.get('http://localhost:8000/steam/all-sharecodes', {
        params: { auth_code: authCode, known_code: knownCode },
        withCredentials: true,
      });

      const shareCodes = response.data.sharecodes;

      await axios.post('http://localhost:8000/steam/save-sharecodes', {
        sharecodes: shareCodes,
        auth_code: authCode,
        known_code: knownCode,
      }, { withCredentials: true });

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
    <div className="container">
      <div className="form-container">
        <h2>Códigos</h2>
        
        <label>
          Auth Code:
          <span className="guide-link" onClick={() => setShowAuthGuide(!showAuthGuide)}>
            ¿Cómo obtener?
          </span>
        </label>
        {showAuthGuide && (
          <div className="guide">
            <p>1. Inicia sesión en Steam a través de Google.</p>
            <p>2. Haz click en Soporte y accede a Counter-strike 2</p>
            <p>3. Accede a “Administrar mis códigos de autenticación.”</p>
            <p>4. 4. Crea el código de autenticación si aún no lo tienes creado</p>
            <img
              src="/images/codes.png"
              alt="Cómo obtener el código de autenticación"
              style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }}
            />
            <p>5. Copia el código de autenticación:</p>
            <img
              src="/images/codes2.png"
              alt="Cómo obtener el código de autenticación"
              style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }}
            />
          </div>
        )}
        <input
          type="text"
          value={authCode}
          onChange={(e) => setAuthCode(e.target.value)}
          placeholder="Ej: 8TRL-ZC9DA-VHYU"
        />

        <label>
          Último Share Code:
          <span className="guide-link" onClick={() => setShowLastGuide(!showLastGuide)}>
            ¿Cómo obtener?
          </span>
        </label>
        {showLastGuide && (
          <div className="guide">
            <p>1. Inicia sesión en Steam a través de Google.</p>
            <p>2. Haz click en Soporte y accede a Counter-strike 2</p>
            <p>3. Accede a “Administrar mis códigos de autenticación.”</p>
            <p>4. Crea el código de autenticación si aún no lo tienes creado</p>
            <img
              src="/images/codes.png"
              alt="Cómo obtener el código de autenticación"
              style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }}
            />
                      
            <p>5. Copia el código de tu última partida completada. </p>

            <img
              src="/images/code-share.png"
              alt="Cómo obtener el último share code"
              style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }}
            />
          </div>
        )}
        <input
          type="text"
          value={knownCode}
          onChange={(e) => setKnownCode(e.target.value)}
          placeholder="Ej: CSGO-XXXX-XXXX"
        />

        <button onClick={handleGetAndSaveShareCodes} disabled={loading}>
          {loading ? 'Procesando...' : 'Obtener y Guardar Share Codes'}
        </button>

        {error && <p className="error-message">{error}</p>}
      </div>
    </div>
  );
};

export default HistoryCodeForm;
