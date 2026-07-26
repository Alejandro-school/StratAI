/**
 * backend/node-service/index.js
 */
require('dotenv').config();
const express = require('express');
const config = require('./services/config');
const friendController = require('./controllers/friendController');
const demoController = require('./controllers/demoController');
const { securityHeaders, requestSizeLimit } = require('./middleware/security');
const { iniciarSesionSteam, botPool } = require('./services/steamGateway');
const { ensureRedis, redisClient } = require('./services/redisClient');
const { isPipelineReady, startPipelineV2, stopPipelineV2 } = require('./services/pipelineV2');

const app = express();

// ── Security Middleware ──
app.use(securityHeaders);
app.use(requestSizeLimit(1024 * 1024)); // 1 MB max body
app.disable('x-powered-by');

// ── CORS ── Whitelist approach
app.use(express.json({
  limit: '1mb',
  verify: (req, _res, buffer) => {
    req.rawBody = Buffer.from(buffer);
  },
}));
app.get('/health', (_req, res) => res.json({ status: 'ok', service: 'steam-pipeline' }));
app.get('/ready', (_req, res) => {
  const ready = redisClient.isReady && (isPipelineReady() || !config.pipelineV2Enabled);
  return res.status(ready ? 200 : 503).json({ status: ready ? 'ready' : 'not_ready' });
});
app.use(friendController);
app.use(demoController);

let server;
let isShuttingDown = false;

(async () => {
  if (
    config.env === 'production'
    && (
      new Set([
        config.sessionSecret,
        config.internalServiceSecret,
        config.credentialEncryptionKey,
      ]).size !== 3
      || [
        config.sessionSecret,
        config.internalServiceSecret,
        config.credentialEncryptionKey,
      ].some((secret) => secret.length < 32)
    )
  ) {
    throw new Error('Session, service and credential secrets must be independent and at least 32 characters');
  }

  // 1) Conectar a Redis con reintentos
  await ensureRedis();

  if (redisClient.isReady) {
    console.log('✅ Conectado a Redis (cliente principal)');
    // 2) Solo ahora inicia consumidores dependientes de Redis
    // 3) Start periodic match detection cron
  } else {
    throw new Error('Redis is required for the v2 pipeline');
  }

  // 3) Arranca sesión Steam
  await iniciarSesionSteam();
  await startPipelineV2();

  // 4) Levanta el servidor HTTP
  server = app.listen(config.server.port, () => {
    console.log(`🚀 Servidor Node.js corriendo en http://localhost:${config.server.port}`);
  });
  server.requestTimeout = 30_000;
  server.headersTimeout = 35_000;
  server.keepAliveTimeout = 5_000;
})().catch(async (error) => {
  console.error('Node service startup failed:', error);
  await stopPipelineV2().catch(() => {});
  if (redisClient.isOpen) await redisClient.quit().catch(() => {});
  process.exit(1);
});

// Graceful shutdown
async function gracefulShutdown(signal) {
  if (isShuttingDown) return;
  isShuttingDown = true;
  console.log(`\n🛑 Recibido ${signal}. Cerrando servicio...`);
  
  try {
    // 2) Cerrar servidor HTTP
    if (server) {
      await new Promise((resolve) => server.close(resolve));
      console.log('✅ Servidor HTTP cerrado');
    }
    await stopPipelineV2();
    
    // 3) Desconectar todos los bots Steam (pool-aware)
    botPool.logoffAll();
    console.log('✅ Sesión(es) Steam cerrada(s)');
    
    // 4) Cerrar Redis principal
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
