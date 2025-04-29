import React from 'react';
import { FaSteam } from 'react-icons/fa';
import '../styles/Start/landing.css';  // Aseguramos que use los nuevos estilos

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

function SteamLoginButton() {
  const handleLogin = () => {
    window.location.href = `${API_URL}/auth/steam/login`;
  };

  return (
    <button className="steam-login-btn" onClick={handleLogin}>
      <FaSteam size={28} style={{ marginRight: '10px' }} />
      Inicia sesión con Steam
    </button>
  );
}

export default SteamLoginButton;

