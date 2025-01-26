import React, { useState } from 'react';
import axios from 'axios';
import '../styles/Auth/codeForm.css';  // Aseguramos que use los nuevos estilos

const HistoryCodeForm = () => {
  const [authCode, setAuthCode] = useState('');
  const [lastCode, setLastCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showAuthGuide, setShowAuthGuide] = useState(false);
  const [showLastGuide, setShowLastGuide] = useState(false);

  const handleGetAndSaveShareCodes = async () => {
    setLoading(true);
    try {
      const response = await axios.get('http://localhost:8000/steam/all-sharecodes', {
        params: { auth_code: authCode, last_code: lastCode },
        withCredentials: true,
      });

      const shareCodes = response.data.sharecodes;
      await axios.post('http://localhost:8000/steam/save-sharecodes', {
        sharecodes: shareCodes,
        auth_code: authCode,
        last_code: lastCode,
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
            <p>1. Inicia sesión en Steama través de Google.</p>
            <p>2. Haz click en Soporte y accede a Counter-strike 2 </p>
            <p>3. Accede a Administrar mis códigos de autenticación.</p>
            <p>4. Obten el código de acceso al historial de partidas:</p>
            <img src="/images/codes.png" alt="Cómo obtener el código de autenticación" style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }} />
            <p>5. Copia el código de autenticación:</p>
            <img src="/images/codes2.png" alt="Cómo obtener el código de autenticación" style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }} />
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
            <p>1. Abre CS2 y ve al historial de partidas.</p>
            <p>2. Busca la demo mas antigua disponible para analizar el máximo posible</p>
            <p>3. Verifica que el botón de descargar se encuentra disponible y no ha caducado la demo. </p>
            <p>4. Copia el código y pegalo en el navegador o un bloc de notas. </p>
            <img src="/images/code-share.png" alt="Cómo obtener el código de autenticación" style={{ width: '100%', maxWidth: '400px', borderRadius: '8px' }} />
            <p>5. Copia el código que empieza por CSGO-XXX, se encuentra al final del enlace.</p>
          </div>
        )}
        <input
          type="text"
          value={lastCode}
          onChange={(e) => setLastCode(e.target.value)}
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
