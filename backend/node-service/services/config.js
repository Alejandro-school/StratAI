/**
 * config.js
 * ---------
 * Configuración centralizada para el servicio Node.js
 */
// Load node-service/.env first (bot credentials, Redis)
require("dotenv").config({ path: __dirname + "/../.env" });
// Then load backend/.env for shared keys (STEAM_API_KEY, SESSION_SECRET_KEY)
// dotenv won't override vars already set by the first call
require("dotenv").config({ path: __dirname + "/../../.env" });

module.exports = {
  // Credenciales del bot Steam
  bot: {
    username: process.env.BOT_USERNAME,
    password: process.env.BOT_PASSWORD,
    sharedSecret: process.env.BOT_SHARED_SECRET,
  },

  // Steam API
  steam: {
    apiKey: process.env.STEAM_API_KEY || "",
  },

  // Redis
  redis: {
    url: process.env.REDIS_URL || "redis://127.0.0.1:6379",
    connectTimeout: parseInt(process.env.REDIS_CONNECT_TIMEOUT || "10000", 10),
    maxRetries: parseInt(process.env.REDIS_MAX_RETRIES || "50", 10),
    keepAlive: 5000,
  },

  // TTLs para Redis (en segundos)
  ttl: {
    friendStatus: 86400, // 24 horas (for confirmed "friend" / "not_friend")
    friendStatusPending: 300, // 5 minutos (for "pending" — re-check frequently)
    matchData: 3600, // 1 hora
    demoUrl: 43200, // 12 horas — URLs de Valve CDN expiran, TTL conservador
  },

  // Tiempos de reintento (en milisegundos)
  retry: {
    gcSessionDelay: 5000, // Delay si no hay sesión GC
    steamReconnect: 10000, // Delay reconexión Steam
    gcRetryDelay: 2000, // Delay reintento GC
  },

  // Cola GC: resuelve sharecode → demo URL (serial o multiplexado)
  gcQueue: {
    concurrency: parseInt(process.env.GC_CONCURRENCY || "3", 10), // 3 concurrent GC requests (multiplexadas por matchId)
    timeout: 60000,    // 1 min timeout (solo GC, sin I/O pesado)
  },

  // Cola de descargas HTTP: descarga + descomprime demos (paralela, independiente del GC)
  downloadQueue: {
    concurrency: parseInt(process.env.DOWNLOAD_CONCURRENCY || "3", 10), // 3 descargas paralelas (Valve CDN throttlea con más)
    timeout: 300000,   // 5 min timeout (demos grandes ~300MB)
    maxRetries: 4,            // Reintentos por descarga individual
    retryBaseDelay: 2000,     // Delay base para backoff exponencial (2s, 4s, 8s, 16s)
  },

  // Cola de procesamiento Go (separada para no bloquear descargas)
  goQueue: {
    concurrency: parseInt(process.env.GO_CONCURRENCY || "10", 10), // 10 procesos Go paralelos (optimizado para 5800X3D)
    timeout: 600000,   // 10 min timeout (raycasting es lento)
  },

  // Servidor
  server: {
    port: parseInt(process.env.PORT || "4000", 10),
  },

  // URLs de servicios
  services: {
    goService: process.env.GO_SERVICE_URL || "http://localhost:8080",
    pythonService: process.env.PYTHON_SERVICE_URL || "http://127.0.0.1:8000",
  },

  // Timeouts para peticiones HTTP (en milisegundos)
  http: {
    timeout: 30000,
    goTimeout: 600000, // 10 min para Go (raycasting)
  },

  // Pool de bots Steam (Fase 4)
  // Formato en .env: BOT_ACCOUNTS=user1:pass1:secret1,user2:pass2:secret2,...
  // Si no se define, usa las credenciales individuales de bot.username/password/sharedSecret
  botPool: {
    accounts: process.env.BOT_ACCOUNTS || "",
    gcConcurrencyPerBot: parseInt(process.env.GC_PER_BOT || "3", 10), // GC requests por bot
  },

  // Cron job for periodic match detection
  cron: {
    interval: process.env.CRON_INTERVAL || "*/5 * * * *", // Every 5 minutes
    userDelay: parseInt(process.env.CRON_USER_DELAY || "2000", 10), // Delay between users (ms)
    enabled: process.env.CRON_ENABLED !== "false", // Enabled by default
  },
};
