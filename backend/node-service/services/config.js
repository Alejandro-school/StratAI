/**
 * config.js
 * ---------
 * Configuración centralizada para el servicio Node.js
 */
require("dotenv").config({ path: __dirname + "/../.env" });

module.exports = {
  // Credenciales del bot Steam
  bot: {
    username: process.env.BOT_USERNAME,
    password: process.env.BOT_PASSWORD,
    sharedSecret: process.env.BOT_SHARED_SECRET,
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
    friendStatus: 86400, // 24 horas
    matchData: 3600, // 1 hora
  },

  // Tiempos de reintento (en milisegundos)
  retry: {
    gcSessionDelay: 5000, // Delay si no hay sesión GC
    steamReconnect: 10000, // Delay reconexión Steam
    gcRetryDelay: 2000, // Delay reintento GC
  },

  // Cola principal (solicitar demo + descargar)
  queue: {
    concurrency: parseInt(process.env.DOWNLOAD_CONCURRENCY || "3", 10), // 3 descargas paralelas
    interval: 500,     // Menos delay entre tareas
    intervalCap: 2,    // 2 tareas por intervalo
    timeout: 300000,   // 5 min timeout (demos grandes)
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

  // Cron
  cron: {
    interval: process.env.CRON_INTERVAL || "*/5 * * * *",
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
};
