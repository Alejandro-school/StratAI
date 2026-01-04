/**
 * backend/node-service/index.js
 */
require('dotenv').config();
const express = require('express');
const config = require('./services/config');
const friendController = require('./controllers/friendController');
const demoController = require('./controllers/demoController');
const { iniciarSesionSteam, monitorearShareCodes, getRedisSubscriber, queue, client } = require('./services/steamDownloader');
const { iniciarCron } = require('./services/cronJob');
const { ensureRedis, redisClient } = require('./services/redisClient');

const app = express();
app.use(express.json());
app.use(friendController);
app.use(demoController);

let server;

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
  server = app.listen(config.server.port, () => {
    console.log(`🚀 Servidor Node.js corriendo en http://localhost:${config.server.port}`);
  });
})();

// Graceful shutdown
async function gracefulShutdown(signal) {
  console.log(`\n🛑 Recibido ${signal}. Cerrando servicio...`);
  
  try {
    // 1) Esperar a que la cola se vacíe (máx 60s)
    console.log('⏳ Esperando finalización de tareas en cola...');
    await Promise.race([
      queue.onIdle(),
      new Promise(resolve => setTimeout(resolve, 60000))
    ]);
    
    // 2) Cerrar servidor HTTP
    if (server) {
      await new Promise((resolve) => server.close(resolve));
      console.log('✅ Servidor HTTP cerrado');
    }
    
    // 3) Desconectar Steam
    if (client.steamID) {
      client.logOff();
      console.log('✅ Sesión Steam cerrada');
    }
    
    // 4) Cerrar Redis subscriber (monitor de sharecodes)
    const subscriber = getRedisSubscriber();
    if (subscriber?.isOpen) {
      await subscriber.quit();
      console.log('✅ Subscriber Redis cerrado');
    }
    
    // 5) Cerrar Redis principal
    if (redisClient.isOpen) {
      await redisClient.quit();
      console.log('✅ Conexión Redis cerrada');
    }
    
    console.log('👋 Servicio cerrado correctamente');
    process.exit(0);
  } catch (err) {
    console.error('❌ Error durante shutdown:', err);
    process.exit(1);
  }
}

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));
