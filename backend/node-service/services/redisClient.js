/**
 * backend/node-service/redisClient.js
 * Cliente Redis robusto para node-redis v4
 */
require('dotenv').config();
const { createClient } = require('redis');

const REDIS_URL = process.env.REDIS_URL || 'redis://127.0.0.1:6379';
const CONNECT_TIMEOUT = parseInt(process.env.REDIS_CONNECT_TIMEOUT || '10000', 10);
const MAX_RETRIES = parseInt(process.env.REDIS_MAX_RETRIES || '50', 10);

const client = createClient({
  url: REDIS_URL,
  socket: {
    connectTimeout: CONNECT_TIMEOUT,
    keepAlive: 5000,
    noDelay: true
  }
});

// Logs útiles
client.on('error', (err) => console.error('❌ [Redis] Error:', err?.message || err));
client.on('reconnecting', () => console.warn('🔁 [Redis] Reintentando conexión...'));
client.on('end', () => console.warn('🛑 [Redis] Conexión cerrada.'));
client.on('connect', () => console.log('🧩 [Redis] Socket conectado; esperando READY...'));
client.on('ready', () => console.log('✅ [Redis] READY'));

// ---- FIX: evita llamar connect() si ya está abierto o en progreso ----
let retries = 0;
let connecting = false;

async function connectWithRetry() {
  // si ya está abierto o estamos conectando, no hagas nada
  if (client.isOpen || connecting) return client;

  connecting = true;
  try {
    while (!client.isOpen) {
      try {
        await client.connect();               // intentamos conectar
        retries = 0;                          // reset de reintentos
        break;
      } catch (err) {
        // si el socket ya está abierto, salimos silenciosamente
        if (String(err?.message || '').includes('Socket already opened')) {
          break;
        }
        retries += 1;
        console.error(`❌ [Redis] connect() falló (#${retries}): ${err?.message || err}`);
        if (retries > MAX_RETRIES) {
          console.error('⛔ [Redis] Máximo de reintentos alcanzado. Sigo en modo degradado.');
          break;
        }
        const delay = Math.min(1000 * Math.pow(2, retries), 15000);
        await new Promise((r) => setTimeout(r, delay));
      }
    }
    return client;
  } finally {
    connecting = false;
  }
}

module.exports = {
  redisClient: client,
  ensureRedis: connectWithRetry
};
