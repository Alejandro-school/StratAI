import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./useAuth";
import { API_URL } from "../utils/api";
import "../styles/Auth/botInstructions.css";

const STATUS_LABEL = {
  friend: "Ya eres amigo",
  pending: "Solicitud enviada (pendiente)",
  not_friend: "Enviar solicitud de amistad",
  unknown: "Comprobar estado",
  blocked: "Bloqueado en Steam",
};

export default function BotInstructions({ userSteamId: propUserSteamId, botSteamId: propBotSteamId }) {
  const navigate = useNavigate();
  const { user } = useAuth();
  const userSteamId = propUserSteamId || user?.steam_id || user?.steamid;
  
  const [botSteamId, setBotSteamId] = useState(propBotSteamId || "");
  const [status, setStatus] = useState("unknown"); // friend|pending|not_friend|unknown
  const [serviceDown, setServiceDown] = useState(false);
  const [source, setSource] = useState(null); // live|cache|none
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(false);
  const [message, setMessage] = useState("");

  const canSend = useMemo(
    () => !serviceDown && (status === "not_friend" || status === "unknown"),
    [serviceDown, status]
  );

  const checkStatus = useCallback(async (silent = false) => {
    if (!silent) {
      setChecking(true);
      setMessage("");
    }
    try {
      const res = await fetch(`${API_URL}/steam/check-friend-status`, {
        method: "GET",
        credentials: "include",
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.detail || data?.error || "Error al comprobar estado");
      setStatus(data?.status || (data?.is_friend ? "friend" : "not_friend"));
      if (data?.bot_steam_id) setBotSteamId(data.bot_steam_id);
      setServiceDown(Boolean(data?.service_down));
      setSource(data?.source || null);
      if (data?.is_friend || data?.status === "friend") {
        navigate("/dashboard", { replace: true });
      }
    } catch (e) {
      if (!silent) setMessage(e.message);
    } finally {
      if (!silent) setChecking(false);
    }
  }, [navigate]);

  async function handleSendFriendRequest() {
    if (!canSend || !userSteamId) return;
    setLoading(true);
    setMessage("");
    try {
      const res = await fetch(`${API_URL}/steam/send-friend-request`, {
        method: "POST",
        credentials: "include",
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.detail || data?.error || "No se pudo enviar la solicitud");
      // UX: marcamos como pending para no spamear mientras Steam responde
      setStatus(data?.status || "pending");
      if (data?.bot_steam_id) setBotSteamId(data.bot_steam_id);
      if (data?.status === "friend") {
        navigate("/dashboard", { replace: true });
        return;
      }
      setMessage(data?.message || "Solicitud enviada. Acepta la invitación en Steam.");
    } catch (e) {
      setMessage(`Error: ${e.message}`);
    } finally {
      setLoading(false);
    }
  }

  function openSteamAdd() {
    // Abre Steam directamente en “Añadir amigo”
    if (botSteamId) {
      window.location.href = `steam://friends/add/${botSteamId}`;
    }
  }

  async function copyBotId() {
    if (!botSteamId) return;
    await navigator.clipboard.writeText(botSteamId);
    setMessage("ID del bot copiado al portapapeles.");
    setTimeout(() => setMessage(""), 2000);
  }

  useEffect(() => {
    checkStatus(true);
    const interval = window.setInterval(() => checkStatus(true), 5000);
    return () => window.clearInterval(interval);
  }, [checkStatus]);

  return (
    <div className="bot-card">
      {/* Banner de servicio */}
      {serviceDown && (
        <div className="bot-banner">
          <span>Steam/GC o el bot están inestables. Mostrando estado en caché.</span>
          <button className="ghost-btn" onClick={() => checkStatus()} disabled={checking}>
            {checking ? "Actualizando..." : "Reintentar"}
          </button>
        </div>
      )}

      <header className="bot-header">
        <h2>Conecta el bot de Steam</h2>
        <p className="subtitle">
          Añade el bot como amigo para detectar tus partidas automáticamente.
        </p>
      </header>

      <ol className="bot-steps">
        <li>Pulsa el botón para enviar la solicitud de amistad.</li>
        <li>Ve a Steam y acepta la invitación.</li>
        <li>Vuelve aquí: el estado cambiará a <strong>Amigo</strong> automáticamente.</li>
      </ol>

      <div className="actions">
        <button
          className={`cta ${(!canSend || loading) ? "disabled" : ""}`}
          onClick={handleSendFriendRequest}
          disabled={!canSend || loading}
          aria-busy={loading}
        >
          {loading ? "Enviando..." : STATUS_LABEL[status] || "Enviar solicitud"}
        </button>

        <button className="secondary" onClick={openSteamAdd} disabled={!botSteamId}>
          Abrir en Steam
        </button>

        <button className="ghost-btn" onClick={() => checkStatus()} disabled={checking}>
          {checking ? "Comprobando..." : "Comprobar estado"}
        </button>
      </div>

      <div className="bot-id">
        <span>También puedes añadirlo manualmente con este ID:</span>
        <code>{botSteamId || "—"}</code>
        <button className="copy" onClick={copyBotId} disabled={!botSteamId}>Copiar</button>
      </div>

      {!!source && (
        <p className="source-note">
          Fuente del estado: <strong>{source}</strong>
          {status === "friend" && " · Todo listo"}
        </p>
      )}

      {message && <p className="feedback">{message}</p>}
    </div>
  );
}
