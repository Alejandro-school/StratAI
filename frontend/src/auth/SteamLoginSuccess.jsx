// src/auth/SteamLoginSuccess.jsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";

const API_URL = process.env.REACT_APP_API_URL || "http://localhost:8000";

export default function SteamLoginSuccess() {
  const navigate = useNavigate();
  const { setUser } = useAuth();          // ← actualizamos el contexto global

  useEffect(() => {
    const run = async () => {
      try {
        /* 1️⃣  Guarda el Steam ID en Redis (por si aún no está) */
        await fetch(`${API_URL}/steam/save-steam-id`, {
          method: "POST",
          credentials: "include",
        });

        /* 2️⃣  Trae el perfil para llenar el contexto */
        const statusRes = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: "include",
        });
        const status = await statusRes.json();
        if (status.authenticated) setUser(status);

        /* 3️⃣  ¿Tiene auth_code + known_code guardados? */
        const codesRes = await fetch(`${API_URL}/steam/check-sharecodes`, {
          credentials: "include",
        });
        const { exists } = await codesRes.json();
        if (!exists) {
          navigate("/history-code", { replace: true });
          return;
        }

        /* 4️⃣  ¿Es amigo del bot? */
        const friendRes = await fetch(`${API_URL}/steam/check-friend-status`, {
          credentials: "include",
        });
        const { is_friend } = await friendRes.json();
        if (!is_friend) {
          navigate("/bot-instructions", { replace: true });
          return;
        }

        /* 5️⃣  Todo OK → Dashboard */
        navigate("/dashboard", { replace: true });
      } catch (err) {
        console.error("SteamLoginSuccess error:", err);
        navigate("/", { replace: true });   // fallback
      }
    };

    run();
  }, [navigate, setUser]);

  return (
    <p style={{ color: "#fff", textAlign: "center", marginTop: "2rem" }}>
      Cargando…
    </p>
  );
}
