// SteamLoginSuccess.jsx
// Si tras loguearte en Steam tu backend te redirige aquí,
// simplemente muestras un “Cargando” o “Has iniciado sesión con éxito”
// pero la verdadera decisión de a dónde navegar la hace el SessionHandler
function SteamLoginSuccess() {
  const saveSteamId = async () => {
    try {
      const response = await fetch("http://localhost:8000/steam/save-steam-id", {
        method: "POST",
        credentials: "include", // Enviar cookies de sesión
      });
  
      const data = await response.json();
      if (response.ok) {
        console.log("✅ Steam ID guardado correctamente:", data.message);
      } else {
        console.error("❌ Error al guardar Steam ID:", data.detail);
      }
    } catch (error) {
      console.error("❌ Error en la solicitud:", error);
    }
  };
  saveSteamId();
}  

// Llamar a esta función después de iniciar sesión

export default SteamLoginSuccess;
