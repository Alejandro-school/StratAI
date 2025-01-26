// SteamLoginSuccess.jsx
import React from 'react';

// Si tras loguearte en Steam tu backend te redirige aquí,
// simplemente muestras un “Cargando” o “Has iniciado sesión con éxito”
// pero la verdadera decisión de a dónde navegar la hace el SessionHandler
function SteamLoginSuccess() {
  return (
    <div>
      <h2>¡Te has logueado correctamente con Steam!</h2>
      <p>Un momento mientras comprobamos tu estado...</p>
    </div>
  );
}

export default SteamLoginSuccess;
