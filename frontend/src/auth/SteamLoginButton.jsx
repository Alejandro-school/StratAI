import React from 'react';
import { FaSteam } from 'react-icons/fa';


// Configurar la URL base del backend
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

function SteamLoginButton() {
  const handleLogin = () => {
    window.location.href = `${API_URL}/auth/steam/login`; // Redirigir a Steam
  };

  return (
    <div className="login-container">
      <h2>Welcome to CS2 Coach AI</h2>
      <p>Analyze your gameplay with advanced statistics.</p>
      <button className="login-btn" onClick={handleLogin}>
        <FaSteam /> Login with Steam
      </button>
    </div>
  );
}

export default SteamLoginButton;
