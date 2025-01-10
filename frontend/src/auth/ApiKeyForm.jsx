import React, { useState } from 'react';
import '../styles/users.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

function ApiKeyForm() {
  const [apiKey, setApiKey] = useState('');
  const [message, setMessage] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await fetch(`${API_URL}/steam/save-api-key`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({ user_api_key: apiKey }),
        credentials: 'include', // Incluir cookies de sesión
      });

      if (response.ok) {
        const data = await response.json();
        setMessage(data.message);
      } else {
        const errorData = await response.json();
        setMessage(`Error: ${errorData.detail}`);
      }
    } catch (error) {
      setMessage(`Error: ${error.message}`);
    }
  };

  return (
    <div className="api-key-container">
      <h2>Introduce tu clave API de Steam</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          placeholder="Clave API de Steam"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          required
        />
        <button type="submit">Guardar Clave</button>
      </form>
      {message && <p>{message}</p>}
    </div>
  );
}

export default ApiKeyForm;
