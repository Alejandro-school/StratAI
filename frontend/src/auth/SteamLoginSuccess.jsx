// src/auth/SteamLoginSuccess.jsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

export default function SteamLoginSuccess() {
  const navigate = useNavigate();
  const { setUser } = useAuth();          // ← actualizamos el contexto global

  useEffect(() => {
    const run = async () => {
      try {
        /* 1️⃣  Guarda el Steam ID en Redis */
        console.log("[LoginSuccess] Step 1: Saving Steam ID...");
        const saveRes = await fetch(`${API_URL}/steam/save-steam-id`, {
          method: "POST",
          credentials: "include",
        });
        if (!saveRes.ok) throw new Error(`Save failed: ${saveRes.status}`);

        /* 2️⃣  Trae el perfil */
        console.log("[LoginSuccess] Step 2: Fetching status...");
        const statusRes = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: "include",
        });
        if (!statusRes.ok) throw new Error(`Status failed: ${statusRes.status}`);
        const status = await statusRes.json();
        if (status.authenticated) setUser(status);
        else throw new Error("Not authenticated after login");

        /* 3️⃣  ¿Tiene sharecodes? */
        console.log("[LoginSuccess] Step 3: Checking sharecodes...");
        const codesRes = await fetch(`${API_URL}/steam/check-sharecodes`, {
          credentials: "include",
        });
        if (!codesRes.ok) throw new Error(`Codes check failed: ${codesRes.status}`);
        const { exists } = await codesRes.json();
        console.log("[LoginSuccess] Exists:", exists);
        if (!exists) {
          console.log("[LoginSuccess] No codes, navigating to /history-code");
          navigate("/history-code", { replace: true });
          return;
        }

        /* 4️⃣  ¿Es amigo del bot? */
        console.log("[LoginSuccess] Step 4: Checking friend status...");
        const friendRes = await fetch(`${API_URL}/steam/check-friend-status`, {
          credentials: "include",
        });
        if (!friendRes.ok) throw new Error(`Friend check failed: ${friendRes.status}`);
        const { is_friend } = await friendRes.json();
        console.log("[LoginSuccess] Is Friend:", is_friend);
        if (!is_friend) {
          console.log("[LoginSuccess] Not friend, navigating to /bot-instructions");
          navigate("/bot-instructions", { replace: true });
          return;
        }

        /* 5️⃣  Todo OK → Dashboard */
        console.log("[LoginSuccess] All checks passed, navigating to /dashboard");
        navigate("/dashboard", { replace: true });
      } catch (err) {
        console.error("SteamLoginSuccess sequence failed:", err);
        // navigate("/", { replace: true }); // Comentado para Debug: ver el error en consola
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
