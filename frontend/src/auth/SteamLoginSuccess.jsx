// src/auth/SteamLoginSuccess.jsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";
import { API_URL } from "../utils/api";

export default function SteamLoginSuccess() {
  const navigate = useNavigate();
  const { user, authError } = useAuth();

  useEffect(() => {
    const run = async () => {
      try {
        if (authError || !user?.authenticated) {
          throw new Error("Steam session was not established");
        }

        const pipelineRes = await fetch(`${API_URL}/steam/pipeline-status`, {
          credentials: "include",
        });
        if (!pipelineRes.ok) throw new Error(`Pipeline status failed: ${pipelineRes.status}`);
        const pipeline = await pipelineRes.json();
        if (!pipeline.configured) {
          navigate("/history-code", { replace: true });
          return;
        }

        let friendship = null;
        for (let attempt = 0; attempt < 6; attempt += 1) {
          const friendRes = await fetch(`${API_URL}/steam/check-friend-status`, {
            credentials: "include",
          });
          if (!friendRes.ok) throw new Error(`Friend check failed: ${friendRes.status}`);
          friendship = await friendRes.json();
          if (
            friendship.is_friend
            || (!friendship.service_down && friendship.status !== "unknown")
          ) {
            break;
          }
          await new Promise((resolve) => setTimeout(resolve, 1500));
        }
        if (!friendship?.is_friend) {
          navigate("/bot-instructions", { replace: true });
          return;
        }

        navigate("/dashboard", { replace: true });
      } catch (err) {
        console.error("SteamLoginSuccess sequence failed:", err);
        navigate("/", { replace: true });
      }
    };

    run();
  }, [authError, navigate, user]);

  return (
    <p style={{ color: "#fff", textAlign: "center", marginTop: "2rem" }}>
      Cargando…
    </p>
  );
}
