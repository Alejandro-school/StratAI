/**
 * index.js
 * --------
 * Punto de entrada principal de la aplicación Node.js:
 *  - Inicializa Express.
 *  - Llama a iniciarSesionSteam().
 *  - Arranca la monitorización de sharecodes en Redis.
 *  - Registra los controladores (rutas).
 *  - Expone el servidor HTTP en el puerto correspondiente.
 */

require('dotenv').config();
const express = require('express');
const friendController = require('./controllers/friendController');
const demoController = require('./controllers/demoController');
const { iniciarSesionSteam, monitorearShareCodes } = require('./services/steamDownloader');
const { iniciarCron } = require('./services/cronJob');

// Inicializamos Express
const app = express();
app.use(express.json());

// Registramos las rutas
app.use(friendController);
app.use(demoController);


// Iniciamos sesión en Steam
iniciarSesionSteam();

// Comenzamos a escuchar los cambios de ShareCodes en Redis
monitorearShareCodes();



// 3. Iniciar la tarea CRON
iniciarCron();

// Ponemos a escuchar el servidor en el puerto 4000 (o el configurado en .env)
const PORT = process.env.PORT || 4000;
app.listen(PORT, () => {
  console.log(`🚀 Servidor Node.js corriendo en http://localhost:${PORT}`);
});
