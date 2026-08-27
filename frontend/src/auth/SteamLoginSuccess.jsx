// src/auth/SteamLoginSuccess.jsx
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";
import { API_URL } from "../utils/api";

export default function SteamLoginSuccess() {
  const navigate = useNavigate();
  const { user, authError } = useAuth();
  const [callbackError, setCallbackError] = useState(null);

  useEffect(() => {
    const controller = new AbortController();

    const run = async () => {
      try {
        if (authError || !user?.authenticated) {
          throw new Error("Steam session was not established");
        }

        const pipelineRes = await fetch(`${API_URL}/steam/pipeline-status`, {
          credentials: "include",
          signal: controller.signal,
        });
        if (!pipelineRes.ok) throw new Error(`Pipeline status failed: ${pipelineRes.status}`);
        const pipeline = await pipelineRes.json();
        if (controller.signal.aborted) return;

        if (!pipeline.configured) {
          navigate("/history-code", { replace: true });
          return;
        }
        if (pipeline.temporary_interface_mode) {
          navigate("/dashboard", { replace: true });
          return;
        }

        let friendship = null;
        for (let attempt = 0; attempt < 6; attempt += 1) {
          const friendRes = await fetch(`${API_URL}/steam/check-friend-status`, {
            credentials: "include",
            signal: controller.signal,
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
          if (controller.signal.aborted) return;
        }
        if (!friendship?.is_friend) {
          navigate("/bot-instructions", { replace: true });
          return;
        }

        navigate("/dashboard", { replace: true });
      } catch (err) {
        if (err.name === "AbortError" || controller.signal.aborted) return;
        console.error("SteamLoginSuccess sequence failed:", err);
        setCallbackError(
          authError || !user?.authenticated
            ? "No se pudo confirmar la sesión de Steam. Vuelve a iniciar sesión."
            : "La sesión se inició, pero no pudimos preparar tu cuenta. Inténtalo de nuevo.",
        );
      }
    };

    run();
    return () => controller.abort();
  }, [authError, navigate, user]);

  if (callbackError) {
    return (
      <div role="alert" style={{ color: "#fff", textAlign: "center", marginTop: "2rem" }}>
        <p>{callbackError}</p>
        <button type="button" onClick={() => navigate("/", { replace: true })}>
          Volver al inicio
        </button>
      </div>
    );
  }

  return (
    <p style={{ color: "#fff", textAlign: "center", marginTop: "2rem" }}>
      Cargando…
    </p>
  );
}
