import React, { useState } from 'react';
import axios from 'axios';

const HistoryCodeForm = () => {
  const [authCode, setAuthCode] = useState('');
  const [lastCode, setLastCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // 🔄 Obtener y guardar los Share Codes
  const handleGetAndSaveShareCodes = async () => {
    setLoading(true);
    try {
      // 📥 Obtener los share codes desde el backend
      const response = await axios.get('http://localhost:8000/steam/all-sharecodes', {
        params: { auth_code: authCode, last_code: lastCode },
        withCredentials: true,
      });

      const shareCodes = response.data.sharecodes;

      // 💾 Guardar los share codes en el backend
      await axios.post('http://localhost:8000/steam/save-sharecodes', {
        sharecodes: shareCodes,
      }, { withCredentials: true });

      setError('');
      
      // 🚀 Redirigir al Dashboard después de guardar
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

      {/* 🔑 Auth Code */}
      <label>Auth Code:</label>
      <input
        type="text"
        value={authCode}
        onChange={(e) => setAuthCode(e.target.value)}
        placeholder="Ej: 8TRL-ZC6HV-VKWG"
      />
      <br />

      {/* 🔄 Último Share Code */}
      <label>Último Share Code:</label>
      <input
        type="text"
        value={lastCode}
        onChange={(e) => setLastCode(e.target.value)}
        placeholder="Ej: CSGO-EFCsv-SuLwK-Jkaqu-xmBU5-LB6iL"
      />
      <br />

      {/* 📥 Botón para obtener y guardar Share Codes */}
      <button onClick={handleGetAndSaveShareCodes} disabled={loading}>
        {loading ? 'Procesando...' : 'Obtener y Guardar Share Codes'}
      </button>

      {/* ⚠️ Error */}
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
};

export default HistoryCodeForm;
