/**
 * backend/node-service/index.js
 */
require('dotenv').config();
const express = require('express');
const cors = require('cors');
const config = require('./services/config');
const friendController = require('./controllers/friendController');
const demoController = require('./controllers/demoController');
const { securityHeaders, requestSizeLimit } = require('./middleware/security');
const { iniciarSesionSteam, monitorearShareCodes, getRedisSubscriber, gcQueue, downloadQueue, goQueue, botPool } = require('./services/steamDownloader');
const { ensureRedis, redisClient } = require('./services/redisClient');
const { startCronJob, stopCronJob } = require('./services/cronJob');

const app = express();

// ── Security Middleware ──
app.use(securityHeaders);
app.use(requestSizeLimit(1024 * 1024)); // 1 MB max body
app.disable('x-powered-by');

// ── CORS ── Whitelist approach
const allowedOrigins = (process.env.ALLOWED_ORIGINS || '')
  .split(',')
  .map(o => o.trim())
  .filter(Boolean);

// Default dev origins if none configured
if (allowedOrigins.length === 0) {
  allowedOrigins.push('http://localhost:3000', 'http://localhost:8000', 'http://127.0.0.1:3000');
}

app.use(cors({
  origin: (origin, callback) => {
    // Allow requests with no origin (internal service calls, curl, etc.)
    if (!origin) return callback(null, true);
    if (allowedOrigins.includes(origin)) return callback(null, true);
    callback(new Error(`CORS: origin ${origin} not allowed`));
  },
  credentials: true,
  methods: ['GET', 'POST', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization'],
}));

app.use(express.json({ limit: '1mb' }));
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
    // 3) Start periodic match detection cron
    startCronJob();
  } else {
    console.warn('⚠️ Redis NO está READY: desactivo monitor de ShareCodes y cron hasta reconexión.');
  }

  // 3) Arranca sesión Steam
  iniciarSesionSteam();

  // 4) Levanta el servidor HTTP
  server = app.listen(config.server.port, () => {
    console.log(`🚀 Servidor Node.js corriendo en http://localhost:${config.server.port}`);
  });
})();

// Graceful shutdown
async function gracefulShutdown(signal) {
  console.log(`\n🛑 Recibido ${signal}. Cerrando servicio...`);
  
  try {
    // 0) Stop cron job
    stopCronJob();
    
    // 1) Esperar a que las 3 colas se vacíen (máx 60s)
    console.log('⏳ Esperando finalización de tareas en colas (GC + Download + Go)...');
    await Promise.race([
      Promise.all([gcQueue.onIdle(), downloadQueue.onIdle(), goQueue.onIdle()]),
      new Promise(resolve => setTimeout(resolve, 60000))
    ]);
    
    // 2) Cerrar servidor HTTP
    if (server) {
      await new Promise((resolve) => server.close(resolve));
      console.log('✅ Servidor HTTP cerrado');
    }
    
    // 3) Desconectar todos los bots Steam (pool-aware)
    botPool.logoffAll();
    console.log('✅ Sesión(es) Steam cerrada(s)');
    
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
