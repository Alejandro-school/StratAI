import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

const API_URL = 'http://localhost:8000';

function SteamLoginSuccess() {
  const navigate = useNavigate();

  useEffect(() => {
    const checkLogin = async () => {
      try {
        const response = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: 'include',
        });

        if (response.ok) {
          navigate('/HistoryCodeForm'); // Redirige al Dashboard si el login fue exitoso
        } else {
          navigate('/');
        }
      } catch {
        navigate('/');
      }
    };

    checkLogin();
  }, [navigate]);

  return <div>Cargando...</div>;
}

export default SteamLoginSuccess;
