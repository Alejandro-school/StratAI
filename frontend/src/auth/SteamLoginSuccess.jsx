// src/auth/SteamLoginSuccess.jsx
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";
import { API_URL } from "../utils/api";

export default function SteamLoginSuccess() {
  const navigate = useNavigate();
  const { user, authError } = useAuth();
  const [callbackError, setCallbackError] = useState(null);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setCallbackError(null);

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
  }, [authError, navigate, retry, user]);

  if (callbackError) {
    return (
      <div role="alert" style={{ color: "#fff", textAlign: "center", marginTop: "2rem" }}>
        <p>{callbackError}</p>
        {user?.authenticated && !authError && (
          <button type="button" onClick={() => setRetry((value) => value + 1)}>
            Reintentar
          </button>
        )}
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
