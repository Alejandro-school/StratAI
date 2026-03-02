// src/auth/SteamLoginSuccess.jsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";
import { API_URL } from "../utils/api";

export default function SteamLoginSuccess() {
  const navigate = useNavigate();
  const { setUser } = useAuth();

  useEffect(() => {
    const run = async () => {
      try {
        const saveRes = await fetch(`${API_URL}/steam/save-steam-id`, {
          method: "POST",
          credentials: "include",
        });
        if (!saveRes.ok) throw new Error(`Save failed: ${saveRes.status}`);

        const statusRes = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: "include",
        });
        if (!statusRes.ok) throw new Error(`Status failed: ${statusRes.status}`);
        const status = await statusRes.json();
        if (status.authenticated) setUser(status);
        else throw new Error("Not authenticated after login");

        const codesRes = await fetch(`${API_URL}/steam/check-sharecodes`, {
          credentials: "include",
        });
        if (!codesRes.ok) throw new Error(`Codes check failed: ${codesRes.status}`);
        const { exists } = await codesRes.json();
        if (!exists) {
          navigate("/history-code", { replace: true });
          return;
        }

        const friendRes = await fetch(`${API_URL}/steam/check-friend-status`, {
          credentials: "include",
        });
        if (!friendRes.ok) throw new Error(`Friend check failed: ${friendRes.status}`);
        const { is_friend } = await friendRes.json();
        if (!is_friend) {
          navigate("/bot-instructions", { replace: true });
          return;
        }

        navigate("/dashboard", { replace: true });
      } catch (err) {
        console.error("SteamLoginSuccess sequence failed:", err);
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
