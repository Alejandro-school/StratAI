/**
 * backend/node-service/index.js
 */
require('dotenv').config();
const express = require('express');
const friendController = require('./controllers/friendController');
const demoController = require('./controllers/demoController');
const { iniciarSesionSteam, monitorearShareCodes } = require('./services/steamDownloader');
const { iniciarCron } = require('./services/cronJob');
const { ensureRedis, redisClient } = require('./services/redisClient'); // ← importa tu cliente

const app = express();
app.use(express.json());
app.use(friendController);
app.use(demoController);

const PORT = process.env.PORT || 4000;

(async () => {
  // 1) Conectar a Redis con reintentos
  await ensureRedis();

  if (redisClient.isReady) {
    console.log('✅ Conectado a Redis (cliente principal)');
    // 2) Solo ahora inicia consumidores dependientes de Redis
    monitorearShareCodes();
  } else {
    console.warn('⚠️ Redis NO está READY: desactivo monitor de ShareCodes hasta reconexión.');
  }

  // 3) Arranca sesión Steam y CRON (si dependen de Redis, ya hay conexión)
  iniciarSesionSteam();
  iniciarCron();

  // 4) Levanta el servidor HTTP
  app.listen(PORT, () => {
    console.log(`🚀 Servidor Node.js corriendo en http://localhost:${PORT}`);
  });
})();
