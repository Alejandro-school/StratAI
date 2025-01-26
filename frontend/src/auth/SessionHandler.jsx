import { useEffect, useState } from "react";
import { useNavigate, useLocation, Outlet } from "react-router-dom";
import axios from "axios";

const API_URL = 'http://localhost:8000';

function SessionHandler() {
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const verificarSesion = async () => {
      let nextRoute;
      
      // 1) Decidimos un "fallback" según la ruta actual
      //    - Si estoy en '/steam-login-success' o '/', asumo que iremos a '/dashboard' si todo está OK
      //    - Si estoy en otra ruta (p.ej. '/HistoryGames'), mantengo la ruta actual
      if (location.pathname === '/steam-login-success' || location.pathname === '/') {
        nextRoute = '/dashboard';
      } else {
        nextRoute = location.pathname;
      }
      
      try {
        // Comprobamos si está autenticado
        const authResp = await axios.get(`${API_URL}/auth/steam/status`, {
          withCredentials: true
        });
        const { authenticated } = authResp.data;

        if (!authenticated) {
          // No autenticado => al landing
          nextRoute = '/';
        } else {
          // Comprobar si es amigo del bot
          const friendResp = await fetch(`${API_URL}/steam/check-friend-status`, {
            credentials: 'include'
          });
          const friendData = await friendResp.json();
          if (!friendData.is_friend) {
            // Si no es amigo => /bot-instructions
            nextRoute = '/bot-instructions';
          } else {
            // Comprobar sharecodes
            const shareResp = await fetch(`${API_URL}/steam/check-sharecodes`, {
              credentials: 'include'
            });
            const shareData = await shareResp.json();

            if (!shareData.exists) {
              // Sin sharecodes => /HistoryCodeForm
              nextRoute = '/HistoryCodeForm';
            }
          }
        }
      } catch (err) {
        console.error('Error en la verificación de sesión:', err);
        nextRoute = '/';
      } finally {
        setLoading(false);

        // Redirigimos solo si la ruta final difiere de la actual
        if (location.pathname !== nextRoute) {
          navigate(nextRoute);
        }
      }
    };
  
    verificarSesion();
  }, [navigate, location.pathname]);

  if (loading) {
    return <p>Cargando...</p>;
  }

  return <Outlet />;
}

export default SessionHandler;
