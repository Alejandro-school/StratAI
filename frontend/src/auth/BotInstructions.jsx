import React from 'react';
import '../styles/Auth/botInstructions.css';

const BotInstructions = ({ userSteamId, botSteamId }) => {
    const handleSendFriendRequest = async () => {
        try {
            const response = await fetch("http://localhost:8000/steam/send-friend-request", {
                method: "POST",
                credentials: "include",
                headers: {
                    "Content-Type": "application/json",
                },
                // Enviamos el steam_id del usuario en lugar del id del bot
                body: JSON.stringify({ steam_id: userSteamId })
            });            
            const data = await response.json();
            if (response.ok) {
                alert("✅ Solicitud de amistad enviada correctamente.");
            } else {
                alert(`❌ Error: ${data.error || data.detail}`);
            }
        } catch (error) {
            console.error("Error al enviar solicitud de amistad:", error);
            alert("❌ Error al conectar con el servidor.");
        }
    };

    return (
        <div className="bot-instructions-container">
            <h2>Instrucciones para conectar con el bot</h2>
            <p>
                Para que podamos detectar automáticamente tus partidas, primero debes añadir a nuestro bot de Steam como amigo.
            </p>
            <ol>
                <li>Haz clic en el botón de abajo para enviar una solicitud de amistad.</li>
                <li>Asegúrate de aceptar la solicitud en tu cuenta de Steam.</li>
                <li>Una vez aceptada, el sistema comenzará a detectar tus partidas automáticamente.</li>
            </ol>
            <button onClick={handleSendFriendRequest} className="add-friend-btn">
                Enviar solicitud de amistad
            </button>
            <p>También puedes agregarlo manualmente con este ID de Steam: <strong>{botSteamId}</strong></p>
        </div>
    );
};

export default BotInstructions;
