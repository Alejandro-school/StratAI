import React, { useState } from 'react';
import axios from 'axios';

const HistoryCodeForm = () => {
  const [authCode, setAuthCode] = useState('');
  const [lastCode, setLastCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleGetAndSaveShareCodes = async () => {
    setLoading(true);
    try {
      // 1. Obtener los sharecodes desde el backend
      const response = await axios.get('http://localhost:8000/steam/all-sharecodes', {
        params: { auth_code: authCode, last_code: lastCode },
        withCredentials: true,
      });

      // 2. Guardarlos en Redis
      const shareCodes = response.data.sharecodes;
      await axios.post('http://localhost:8000/steam/save-sharecodes', {
        sharecodes: shareCodes,
        auth_code: authCode,
        last_code: lastCode,
      }, { withCredentials: true });

      setError('');
      // Redirigir o actualizar la vista
      window.location.href = '/dashboard';
    } catch (err) {
      console.error(err);
      setError('❌ Error al obtener o guardar los share codes');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Obtener hasta 30 partidas</h2>
      <label>Auth Code:</label>
      <input
        type="text"
        value={authCode}
        onChange={(e) => setAuthCode(e.target.value)}
        placeholder="Ej: 8TRL-ZC6HV-VKWG"
      />
      <br />
      <label>Último Share Code:</label>
      <input
        type="text"
        value={lastCode}
        onChange={(e) => setLastCode(e.target.value)}
        placeholder="Ej: CSGO-XXXX-XXXX"
      />
      <br />
      <button onClick={handleGetAndSaveShareCodes} disabled={loading}>
        {loading ? 'Procesando...' : 'Obtener y Guardar Share Codes'}
      </button>
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
};

export default HistoryCodeForm;
