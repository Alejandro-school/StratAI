// src/components/SteamLoginButton.jsx
import React from 'react';
import { FaSteam } from 'react-icons/fa';
import '../styles/Landing/landing.css';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

// 👉 Construye la URL de login con (o sin) plan
export const buildSteamLoginUrl = (plan) => {
  const base = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');
  return `${base}/auth/steam/login${plan ? `?plan=${plan}` : ''}`;
};

// 👉 Navegación coherente en todo el proyecto
export const steamLogin = (plan) => {
  window.location.assign(buildSteamLoginUrl(plan));
};

function SteamLoginButton({ className = 'steam-login-btn', children }) {
  const handleLogin = () => steamLogin();

  return (
    <button className={className} onClick={handleLogin}>
      <FaSteam size={28} style={{ marginRight: '10px' }} />
      {children || 'Inicia sesión con Steam'}
    </button>
  );
}

export default SteamLoginButton;
