const config = require("./config");
const SteamUser = require("steam-user");
const GlobalOffensive = require("globaloffensive");
const SteamTotp = require("steam-totp");
const { default: PQueue } = require("p-queue");
const fs = require("fs");
const fsPromises = require("fs").promises;
const path = require("path");
const https = require("https");
const { ShareCode } = require("globaloffensive-sharecode");
const Language = require("globaloffensive/language.js");
const Protos = require("globaloffensive/protobufs/generated/_load.js");
const { redisClient, ensureRedis } = require("./redisClient");
const unbzip2Stream = require("unbzip2-stream");
const axios = require("axios");
const http = require("http");
const { EventEmitter } = require("events");
const { botPool } = require("./botPool");

// ============================================================
// PROGRESS EVENT EMITTER (Fase 5: SSE para onboarding)
// ============================================================
// Emite eventos de progreso por steamID para consumir via SSE.
// Stages: "gc_resolving" → "downloading" → "processing" → "completed" | "error"
// ============================================================
const progressEmitter = new EventEmitter();
progressEmitter.setMaxListeners(100); // Hasta 100 clientes SSE simultáneos

// Credenciales del bot desde config
const BOT_USERNAME = config.bot.username;
const BOT_PASSWORD = config.bot.password;
const BOT_SHARED_SECRET = config.bot.sharedSecret;

if (!BOT_USERNAME || !BOT_PASSWORD || !BOT_SHARED_SECRET) {
  console.error("❌ Error: Faltan credenciales del bot en .env");
  process.exit(1);
}

// Directorio donde se guardan las demos
const DEMOS_DIR = path.join(__dirname, "../../data/demos");
if (!fs.existsSync(DEMOS_DIR)) {
  fs.mkdirSync(DEMOS_DIR, { recursive: true });
}

// Control de reintentos por sharecode (con limpieza automática)
const reintentosSharecode = new Map();

// Limpia entradas antiguas cada 10 minutos para evitar memory leak
setInterval(() => {
  const ahora = Date.now();
  for (const [code, data] of reintentosSharecode.entries()) {
    if (ahora - data.timestamp > 600000) { // 10 minutos
      reintentosSharecode.delete(code);
    }
  }
}, 600000);

// ============================================================
// PIPELINE DE 3 COLAS (GC → Download → Go)
// ============================================================
// 1. gcQueue: Resuelve sharecode → demo URL via GC (multiplexado)
// 2. downloadQueue: Descarga + descomprime demos en paralelo
// 3. goQueue: Procesa demos con Go service en paralelo
//
// Esto permite que el GC resuelva URLs mientras se descargan demos
// anteriores, maximizando throughput.
// ============================================================

// Cola 1: GC URL resolution (multiplexada con correlación por matchId)
const gcQueue = new PQueue({
  concurrency: config.gcQueue.concurrency, // 3 concurrent GC requests
  timeout: config.gcQueue.timeout,
  throwOnTimeout: false,
});

gcQueue.on('error', (err) => {
  console.error('❌ Error en cola GC:', err.message);
});

gcQueue.on('active', () => {
  console.log(`🔄 [GC Queue] Activas: ${gcQueue.pending + 1}, En espera: ${gcQueue.size}`);
});

// Cola 2: HTTP downloads (paralelas, independientes del GC)
const downloadQueue = new PQueue({
  concurrency: config.downloadQueue.concurrency, // 6 descargas paralelas
  timeout: config.downloadQueue.timeout,
  throwOnTimeout: false,
});

downloadQueue.on('error', (err) => {
  console.error('❌ Error en cola de descargas:', err.message);
});

downloadQueue.on('active', () => {
  console.log(`⬇️ [Download Queue] Activas: ${downloadQueue.pending + 1}, En espera: ${downloadQueue.size}`);
});

// Cola 3: Go processing (paralelo, no bloquea descargas)
const goQueue = new PQueue({
  concurrency: config.goQueue.concurrency,  // 10 por defecto (config.js)
  timeout: config.goQueue.timeout,
  throwOnTimeout: false,
});

goQueue.on('error', (err) => {
  console.error('❌ Error en cola Go:', err.message);
});

goQueue.on('active', () => {
  console.log(`🔄 [GoQueue] Activas: ${goQueue.pending + 1}, En espera: ${goQueue.size}`);
});

// Referencia legacy para compatibilidad con exports
const queue = gcQueue;

// ============================================================
// BOT POOL INTEGRATION (Fase 4)
// ============================================================
// Si BOT_ACCOUNTS está configurado, usa pool multi-bot.
// Si no, usa las credenciales individuales (bot singleton legacy).
// El pool se inicializa en index.js antes de iniciar sesión.
// ============================================================

// Inicializar pool (parsea config, no hace login aún)
botPool.initialize();

// Referencias al bot primario (para compatibilidad con código legacy)
const primaryBot = botPool.getPrimaryBot();
const client = primaryBot ? primaryBot.client : new SteamUser();
const csgo = primaryBot ? primaryBot.csgo : new GlobalOffensive(client);

// Flag para evitar duplicar listeners
let steamListenersSetup = false;

/**
 * Genera el código 2FA (Steam Guard) — solo para bot legacy
 */
function generarAuthCode() {
  return SteamTotp.generateAuthCode(BOT_SHARED_SECRET);
}

/**
 * setupSteamListeners:
 * --------------------
 * En modo pool: Los listeners ya están configurados en cada BotInstance.
 * En modo legacy (1 bot): Configura listeners sobre el client singleton.
 * En ambos modos, este setup es idempotente.
 */
function setupSteamListeners() {
  if (steamListenersSetup) return;
  steamListenersSetup = true;

  // Si hay pool multi-bot, los listeners ya están en cada BotInstance
  if (botPool.bots.length > 1) return;

  // Modo legacy: configurar listeners sobre el singleton
  client.on("loggedOn", () => {
    console.log("✅ Bot conectado a Steam");
    client.setPersona(SteamUser.EPersonaState.Online);
    client.gamesPlayed(730);
  });

  client.on("error", (err) => {
    console.error(`❌ Error en cliente Steam: ${err.message}`);
  });

  client.on("disconnected", (eresult, msg) => {
    console.error(`❌ Bot desconectado: ${msg}. Reintentando en ${config.retry.steamReconnect / 1000}s...`);
    setTimeout(() => iniciarSesionSteam(), config.retry.steamReconnect);
  });

  csgo.on("connectedToGC", () => {
    console.log("🎮 Conectado al GC de CS:GO");
    console.log(`🟢 Estado GC: ${csgo.haveGCSession}`);
  });

  csgo.on("disconnectedFromGC", (reason) => {
    console.error(`❌ Desconectado del GC: ${reason}`);
  });
}

/**
 * iniciarSesionSteam:
 * -------------------
 * En modo pool: hace loginAll() en todos los bots.
 * En modo legacy: logea el bot singleton.
 */
async function iniciarSesionSteam() {
  setupSteamListeners();

  if (botPool.bots.length > 1) {
    // Multi-bot: login escalonado
    await botPool.loginAll();
  } else {
    // Legacy: login directo
    client.logOn({
      accountName: BOT_USERNAME,
      password: BOT_PASSWORD,
      twoFactorCode: generarAuthCode(),
    });
  }
}

// ============================================================
// GC MULTIPLEXADO: Correlación de respuestas por matchId
// ============================================================
// Permite enviar múltiples requests al GC simultáneamente.
// Cada respuesta se demuxa por matchId, resolviendo la Promise correcta.
// ============================================================

// Map de requests pendientes: matchId → { resolve, reject, timeout, retries }
const pendingGCRequests = new Map();
let gcListenerSetup = false;

// ============================================================
// FASE 3: Caché de Demo URLs en Redis
// ============================================================
// Key: demo_url_cache:{matchId} → JSON { demoUrl, matchDuration, matchDate, matchTime, mapName }
// TTL: 12 horas (urls de Valve CDN expiran, conservador)
// Evita GC requests duplicados si múltiples usuarios jugaron la misma partida.
// ============================================================

async function getCachedDemoUrl(matchId) {
  try {
    const cached = await redisClient.get(`demo_url_cache:${matchId}`);
    if (cached) {
      console.log(`⚡ [Cache HIT] matchId=${matchId} — skipping GC request`);
      return JSON.parse(cached);
    }
  } catch { /* cache miss, continue to GC */ }
  return null;
}

async function setCachedDemoUrl(matchId, data) {
  try {
    await redisClient.set(`demo_url_cache:${matchId}`, JSON.stringify(data), {
      EX: config.ttl.demoUrl,
    });
  } catch (err) {
    console.warn(`⚠️ [Cache] Error guardando URL para matchId=${matchId}: ${err.message}`);
  }
}

/**
 * setupGCListener:
 * ----------------
 * Configura un listener GLOBAL y PERSISTENTE para matchList events.
 * En modo pool: se registra en TODOS los bots.
 * Demuxa respuestas por matchId a las Promises correctas.
 */
function setupGCListener() {
  if (gcListenerSetup) return;
  gcListenerSetup = true;

  const onMatchList = (matches) => {
    if (!matches || !matches.length) return;

    for (const match of matches) {
      const matchId = match.matchid?.toString();
      if (!matchId) continue;

      const pending = pendingGCRequests.get(matchId);
      if (!pending) continue;

      // Verificar que tiene URL de demo válida
      const hasUrl = match.roundstatsall &&
        Array.isArray(match.roundstatsall) &&
        match.roundstatsall.some((round) => round.map && round.map.startsWith("http"));

      if (!hasUrl) {
        // Sin URL: reintentar si quedan intentos
        if (pending.retries < pending.maxRetries) {
          pending.retries++;
          console.warn(`⚠️ [GC Mux] matchId=${matchId} sin URL. Reintento ${pending.retries}/${pending.maxRetries}`);
          setTimeout(() => sendGCRequest(pending.decoded), 1000);
        } else {
          clearTimeout(pending.timeoutId);
          pendingGCRequests.delete(matchId);
          pending.reject(new Error("No se encontró la URL de la demo (partida caducada)."));
        }
        continue;
      }

      // Extraer datos
      const demoUrl = match.roundstatsall.find(
        (round) => round.map && round.map.startsWith("http")
      ).map;
      const lastRound = match.roundstatsall[match.roundstatsall.length - 1];
      const matchDuration = lastRound ? lastRound.match_duration || 0 : 0;
      const matchTime = match.matchtime || 0;
      const matchDate = matchTime > 0 ? new Date(matchTime * 1000).toISOString() : "";
      const mapName = lastRound ? lastRound.reservation?.game_map_key || "" : "";

      // Resolver la Promise correspondiente
      clearTimeout(pending.timeoutId);
      pendingGCRequests.delete(matchId);

      // Decrementar activeRequests del bot que procesó la respuesta
      if (pending.botInstance) {
        pending.botInstance.activeRequests = Math.max(0, pending.botInstance.activeRequests - 1);
      }

      console.log(`✅ [GC Mux] matchId=${matchId} → URL resuelta (${pendingGCRequests.size} pendientes)`);

      const result = {
        demoUrl,
        matchDuration,
        matchDate,
        matchTime,
        mapName,
        matchID: matchId,
      };

      // Guardar en caché para futuros requests (Fase 3)
      setCachedDemoUrl(matchId, result);

      pending.resolve(result);
    }
  };

  // Registrar listener en TODOS los bots del pool
  for (const bot of botPool.bots) {
    bot.csgo.on("matchList", onMatchList);
  }
}

/**
 * sendGCRequest:
 * Envía un request protobuf al GC usando el bot con menor carga del pool.
 * Retorna el BotInstance utilizado (para tracking de activeRequests).
 */
function sendGCRequest(decoded) {
  const bot = botPool.getAvailableBot();
  if (!bot) {
    console.warn(`⚠️ [GC Pool] Ningún bot disponible. Reintentando en ${config.retry.gcRetryDelay}ms...`);
    setTimeout(() => sendGCRequest(decoded), config.retry.gcRetryDelay);
    return null;
  }

  bot.activeRequests++;
  bot.csgo._send(
    Language.MatchListRequestFullGameInfo,
    Protos.CMsgGCCStrike15_v2_MatchListRequestFullGameInfo,
    {
      matchid: decoded.matchId,
      outcomeid: decoded.outcomeId,
      token: decoded.token,
    }
  );

  // Almacenar referencia al bot en el pending request para decrementar después
  const matchId = decoded.matchId.toString();
  const pending = pendingGCRequests.get(matchId);
  if (pending) {
    pending.botInstance = bot;
  }

  return bot;
}

/**
 * requestGameAsync:
 * -----------------
 * Solicita al GC la URL de la demo. Usa el sistema multiplexado:
 * - Registra una Promise indexada por matchId
 * - Envía el request al GC
 * - El listener global resuelve la Promise cuando llega la respuesta
 * - Soporta concurrent requests (cada uno vuelve con su matchId)
 */
GlobalOffensive.prototype.requestGameAsync = function (
  shareCodeStr,
  maxRetries = 2
) {
  // Asegurar que el listener global está activo
  setupGCListener();

  // Decodificar primero para extraer matchId (necesario para cache check)
  let decoded;
  try {
    decoded = new ShareCode(shareCodeStr).decode();
  } catch (err) {
    return Promise.reject(new Error(`No se pudo decodificar el sharecode: ${shareCodeStr} => ${err.message}`));
  }

  const matchId = decoded.matchId.toString();

  // FASE 3: Check caché ANTES de hacer GC request
  return getCachedDemoUrl(matchId).then((cached) => {
    if (cached) return cached;

    return new Promise((resolve, reject) => {
      // Si ya hay un request pendiente para este matchId, reusar
      if (pendingGCRequests.has(matchId)) {
        const existing = pendingGCRequests.get(matchId);
        existing.promise.then(resolve).catch(reject);
        return;
      }

      // Timeout individual (30s)
      const timeoutId = setTimeout(() => {
        pendingGCRequests.delete(matchId);
        reject(new Error(`Timeout: sin respuesta del GC para matchId=${matchId} en 30s`));
      }, 30000);

      // Crear Promise compartible para deduplicación
      let resolveOuter, rejectOuter;
      const promise = new Promise((res, rej) => {
        resolveOuter = res;
        rejectOuter = rej;
      });
      promise.then(resolve).catch(reject);

      pendingGCRequests.set(matchId, {
        resolve: resolveOuter,
        reject: rejectOuter,
        promise,
        timeoutId,
        retries: 0,
        maxRetries,
        decoded,
      });

      // Esperar GC session antes de enviar (pool-aware)
      const availableBot = botPool.getAvailableBot();
      if (!availableBot) {
        console.warn(`⚠️ [GC Mux] Ningún bot con sesión GC. Esperando...`);
        const waitGC = () => {
          if (botPool.getAvailableBot()) {
            sendGCRequest(decoded);
          } else {
            setTimeout(waitGC, config.retry.gcRetryDelay);
          }
        };
        waitGC();
        return;
      }

      sendGCRequest(decoded);
    });
  });
};

/**
 * descargarFicheroHTTP:
 * ---------------------
 * Descarga el fichero de demo y lo guarda en filePath.
 * Incluye retry con backoff exponencial para errores transitorios
 * (ECONNRESET, ETIMEDOUT, ECONNREFUSED, socket hang up).
 */
const { pipeline } = require("stream");

const RETRYABLE_CODES = new Set(["ECONNRESET", "ETIMEDOUT", "ECONNREFUSED", "EPIPE", "EAI_AGAIN"]);

function isRetryableError(err) {
  if (RETRYABLE_CODES.has(err.code)) return true;
  if (err.message && err.message.includes("socket hang up")) return true;
  if (err.message && err.message.includes("aborted")) return true;
  return false;
}

/**
 * Check if error is a CDN URL expiration (403/410).
 * These require re-resolving the demo URL via GC, not simple retries.
 */
function isCdnExpiredError(err) {
  const msg = err.message || "";
  return msg.includes("HTTP status: 403") || msg.includes("HTTP status: 410");
}

async function descargarFicheroHTTP(url, filePath, attempt = 0) {
  if (fs.existsSync(filePath)) {
    console.log(`📂 Ya existe: ${filePath}`);
    return filePath;
  }

  const maxRetries = config.downloadQueue.maxRetries || 4;
  const baseDelay = config.downloadQueue.retryBaseDelay || 2000;

  try {
    await _downloadOnce(url, filePath);
    return filePath;
  } catch (err) {
    // Limpiar archivo parcial
    try { fs.unlinkSync(filePath); } catch {}

    if (isRetryableError(err) && attempt < maxRetries) {
      const delay = baseDelay * Math.pow(2, attempt); // 2s, 4s, 8s, 16s
      console.warn(`♻️ [Download] Retry ${attempt + 1}/${maxRetries} para ${path.basename(filePath)} en ${delay / 1000}s (${err.code || err.message})`);
      await new Promise((r) => setTimeout(r, delay));
      return descargarFicheroHTTP(url, filePath, attempt + 1);
    }

    throw err;
  }
}

function _downloadOnce(url, filePath) {
  return new Promise((resolve, reject) => {
    console.log(`⬇️ Descargando: ${url}`);

    const esBz2 = url.endsWith(".bz2");
    const fileStream = fs.createWriteStream(filePath);
    const lib = url.startsWith("https") ? https : http;

    const req = lib.get(url, { rejectUnauthorized: false }, (res) => {
      if (res.statusCode !== 200) {
        fileStream.close();
        fs.unlink(filePath, () => {});
        return reject(new Error(`HTTP status: ${res.statusCode}`));
      }

      if (esBz2) {
        pipeline(res, unbzip2Stream(), fileStream, (err) => {
          if (err) {
            fs.unlink(filePath, () => {});
            return reject(err);
          }
          console.log(`✅ Demo guardada en: ${filePath}`);
          resolve(filePath);
        });
      } else {
        pipeline(res, fileStream, (err) => {
          if (err) {
            fs.unlink(filePath, () => {});
            return reject(err);
          }
          console.log(`✅ Demo guardada en: ${filePath}`);
          resolve(filePath);
        });
      }
    });

    req.on("error", (err) => {
      fileStream.close();
      fs.unlink(filePath, () => {});
      reject(err);
    });
  });
}

/**
 * procesarShareCode:
 * ------------------
 * Orquesta el pipeline de 3 colas para un sharecode:
 * 1. gcQueue: Resuelve sharecode → demo URL via GC
 * 2. downloadQueue: Descarga + descomprime la demo
 * 3. goQueue: Analiza la demo con el servicio Go
 *
 * Cada fase se encola en la cola correspondiente, permitiendo
 * que GC resuelva el siguiente sharecode mientras se descarga el anterior.
 */
async function procesarShareCode(sharecode, steamID, maxReintentos = 3) {
  console.log(`\n🔍 Procesando ShareCode ${sharecode} (SteamID: ${steamID})`);

  // LOCK ATÓMICO: Prevenir procesamiento duplicado con SETNX
  const lockKey = `lock:sharecode:${sharecode}`;
  const lockAcquired = await redisClient.setNX(lockKey, Date.now().toString());
  
  if (!lockAcquired) {
    console.log(`⏭️ Sharecode ${sharecode} ya está siendo procesado por otro worker`);
    return;
  }
  
  // TTL en el lock (10 min máximo para evitar locks fantasma)
  await redisClient.expire(lockKey, 600);

  // Inicializar reintentos
  if (!reintentosSharecode.has(sharecode)) {
    reintentosSharecode.set(sharecode, { count: 0, timestamp: Date.now() });
  }

  // Esperar sesión GC (pool-aware: cualquier bot con sesión activa)
  if (!botPool.getAvailableBot()) {
    console.warn(`⚠️ Ningún bot con sesión GC. Reintentamos en ${config.retry.gcSessionDelay / 1000}s...`);
    await redisClient.del(lockKey);
    await new Promise(resolve => setTimeout(resolve, config.retry.gcSessionDelay));
    return procesarShareCode(sharecode, steamID, maxReintentos);
  }

  try {
    // ── FASE 1: Resolver demo URL via GC ──
    progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "gc_resolving", matchID: null });
    const activeCsgo = botPool.getPrimaryBot()?.csgo || csgo;
    const { demoUrl, matchDuration, matchDate, matchTime, mapName, matchID } =
      await activeCsgo.requestGameAsync(sharecode);

    // Guardar metadata en Redis inmediatamente (no depende de descarga)
    const matchData = {
      matchID: matchID,
      matchDuration: matchDuration,
      matchDate: matchDate,
      matchTime: matchTime,
    };
    await redisClient.set(`match_data:${matchID}`, JSON.stringify(matchData), {
      EX: config.ttl.matchData,
    });
    console.log(`✅ [GC] URL resuelta para ${sharecode} → matchID=${matchID}`);

    // Liberar lock del GC y marcar sharecode como "downloading"
    await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, "downloading");

    // ── FASE 2: Encolar descarga HTTP (paralela, no bloquea GC) ──
    const cleanedCode = sharecode.replace(/CSGO-|-/g, "");
    const filename = `match_${cleanedCode}.dem`;
    const filePath = path.join(DEMOS_DIR, filename);

    downloadQueue.add(async () => {
      try {
        progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "downloading", matchID });
        await descargarFicheroHTTP(demoUrl, filePath);
        console.log(`✅ [Download] ${filename} descargada`);

        // ── FASE 3: Encolar procesamiento Go (paralelo) ──
        goQueue.add(async () => {
          try {
            progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "processing", matchID });
            console.log(`🔧 [Go] Procesando ${filename}...`);
            const goResponse = await axios.post(
              `${config.services.goService}/process-demo`,
              {
                demo_path: filePath,
                steam_id: steamID,
                match_id: matchID.toString(),
                match_date: matchDate,
                match_duration: matchDuration,
              },
              { timeout: config.http.goTimeout }
            );

            if (goResponse.data?.status === "success") {
              console.log(`✅ [Go] Stats de ${filename} procesadas`);
              
              const processedDemoData = {
                match_id: matchID.toString(),
                steam_id: steamID,
                map_name: mapName || goResponse.data?.map_name || "unknown",
                date: matchDate,
                duration: matchDuration,
                processed_at: new Date().toISOString()
              };
              
              await redisClient.rPush(
                `processed_demos:${steamID}`,
                JSON.stringify(processedDemoData)
              );
              
              await redisClient.del(`dashboard_stats:${steamID}`);
              console.log(`📊 [Pipeline] ${filename} completado: GC → Download → Go ✅`);
              progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "completed", matchID });
            } else {
              console.warn(`⚠️ [Go] Respuesta inesperada para ${filename}:`, goResponse.data);
            }
          } catch (err) {
            console.error(`❌ [Go] Error procesando ${filename}: ${err.message}`);
            progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "error", matchID, error: err.message });
          }
        });
      } catch (err) {
        // CDN URL expired (403/410): invalidate cache and re-enqueue for fresh GC resolution
        if (isCdnExpiredError(err)) {
          console.warn(`⚠️ [Download] CDN URL expired for ${filename} (${err.message}). Invalidating cache and re-enqueueing...`);
          await redisClient.del(`demo_url_cache:${matchID}`);
          await redisClient.del(`lock:sharecode:${sharecode}`);
          await redisClient.del(`lock:enqueue:${sharecode}`);
          await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, "pending");
          enqueueShareCode(sharecode, steamID, { priority: 5, maxRetries: 2 });
          progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "retrying_gc", matchID });
          return;
        }
        console.error(`❌ [Download] Error descargando ${filename} (agotados reintentos): ${err.message}`);
        progressEmitter.emit(`progress:${steamID}`, { sharecode, stage: "error", matchID, error: err.message });
      }
    });

    // Marcar sharecode como procesado (GC resuelto + descarga encolada)
    await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, "processed");
    reintentosSharecode.delete(sharecode);
    await redisClient.del(lockKey);

  } catch (err) {
    console.error(`❌ Error al procesar el ShareCode ${sharecode}:`, err);

    // Demo caducada: no reintentar
    if (
      err.message.includes("caducada") ||
      err.message.includes("No se encontró la URL")
    ) {
      console.log(`ℹ️ Marcando sharecode caducado: ${sharecode}`);
      await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, "caducado");
      reintentosSharecode.delete(sharecode);
      await redisClient.del(lockKey);
      return;
    }

    // Reintentos GC
    const retryData = reintentosSharecode.get(sharecode) || { count: 0, timestamp: Date.now() };
    retryData.count++;
    retryData.timestamp = Date.now();
    reintentosSharecode.set(sharecode, retryData);

    if (retryData.count < maxReintentos) {
      console.log(`♻️ Reintentando sharecode ${sharecode} (intento ${retryData.count} de ${maxReintentos})...`);
      await redisClient.del(lockKey);
      enqueueShareCode(sharecode, steamID, { maxRetries: maxReintentos });
    } else {
      console.error(`❌ Sharecode ${sharecode} alcanzó el máximo de reintentos.`);
      reintentosSharecode.delete(sharecode);
      await redisClient.del(lockKey);
    }
  }
}

/**
 * enqueueShareCode:
 * -----------------
 * Encola un sharecode en el pipeline con prioridad opcional.
 * priority > 0 = alta prioridad (onboarding de usuario nuevo).
 * PQueue ejecuta primero las tareas con mayor priority.
 */
function enqueueShareCode(sharecode, steamID, { priority = 0, maxRetries = 3 } = {}) {
  gcQueue.add(() => procesarShareCode(sharecode, steamID, maxRetries), { priority });
}

/**
 * monitorearShareCodes:
 * ---------------------
 * Se suscribe a eventos de Redis para detectar nuevos sharecodes en
 * la clave "sharecodes:{steamID}" y encola su procesamiento.
 */
let redisSubscriber = null;  // Para cleanup en graceful shutdown

async function monitorearShareCodes() {
  await ensureRedis();
  
  // Crear subscriber con manejo de errores
  redisSubscriber = redisClient.duplicate();
  
  // Error handling para el subscriber
  redisSubscriber.on('error', (err) => {
    console.error('❌ [Monitor] Error en subscriber Redis:', err.message);
  });
  
  redisSubscriber.on('end', () => {
    console.warn('🛑 [Monitor] Subscriber Redis desconectado. Reconectando en 5s...');
    setTimeout(() => monitorearShareCodes(), config.retry.gcSessionDelay);
  });

  try {
    await redisSubscriber.connect();
    console.log("📡 Escuchando rpush en Redis para nuevos ShareCodes...");

    // Validate and configure keyspace events
    try {
      await redisSubscriber.configSet("notify-keyspace-events", "KEA");
    } catch (configErr) {
      // Some Redis configs don't allow CONFIG SET (e.g., managed Redis)
      // Verify the setting is already correct
      const currentConfig = await redisSubscriber.configGet("notify-keyspace-events");
      const value = currentConfig?.["notify-keyspace-events"] || "";
      if (!value.includes("K") || !value.includes("E")) {
        console.error("❌ [Monitor] Redis keyspace events not configured. Run: CONFIG SET notify-keyspace-events KEA");
        console.error("❌ [Monitor] Sharecode monitoring will NOT work without this.");
        return;
      }
      console.log("✅ [Monitor] Redis keyspace events already configured:", value);
    }

    await redisSubscriber.subscribe("__keyevent@0__:rpush", async (key) => {
      if (!key.startsWith("sharecodes:")) return;

      const steamID = key.split(":")[1];
      console.log(`🔔 [Monitor] Nuevos sharecodes detectados para: ${steamID}`);
      
      try {
        // Obtain all sharecodes
        const sharecodes = await redisClient.lRange(`sharecodes:${steamID}`, 0, -1);
        if (!sharecodes.length) return;
        
        // Batch: status check for all sharecodes at once
        const statusMap = await redisClient.hGetAll(`sharecode_status:${steamID}`);
        
        let encolados = 0;
        for (const code of sharecodes) {
          const status = statusMap[code];
          if (!status || status === "pending") {
            // Atomic dedup: only enqueue if we can set the lock
            const lockKey = `lock:enqueue:${code}`;
            const acquired = await redisClient.set(lockKey, "1", { NX: true, EX: 300 });
            if (acquired) {
              enqueueShareCode(code, steamID);
              encolados++;
            }
          }
        }
        
        if (encolados > 0) {
          console.log(`✅ [Monitor] ${encolados} sharecodes encolados en gcQueue para ${steamID}`);
        }
      } catch (err) {
        console.error(`❌ [Monitor] Error procesando sharecodes de ${steamID}:`, err.message);
      }
    });
  } catch (err) {
    console.error('❌ [Monitor] Error iniciando subscriber:', err.message);
    // Reintentar conexión
    setTimeout(() => monitorearShareCodes(), config.retry.gcSessionDelay);
  }
}

/**
 * Obtiene el subscriber para cleanup
 */
function getRedisSubscriber() {
  return redisSubscriber;
}

/**
 * fetchMatchInfoBySharecode:
 * --------------------------
 * Obtiene la metadata de una partida (incluyendo fecha) sin descargar la demo.
 * Útil para demos ya existentes que no tienen match_info.json
 * 
 * @param {string} sharecode - El sharecode de la partida (ej: CSGO-xxxx-xxxx-xxxx-xxxx-xxxx)
 * @returns {Promise<Object>} - Objeto con date, matchDuration, matchTime, matchID
 */
async function fetchMatchInfoBySharecode(sharecode) {
  console.log(`\n📅 Obteniendo metadata para sharecode: ${sharecode}`);

  if (!botPool.getAvailableBot()) {
    throw new Error('No hay sesión GC activa. Asegúrate de que al menos un bot esté conectado.');
  }

  try {
    const primaryCsgo = botPool.getPrimaryBot()?.csgo || csgo;
    const result = await primaryCsgo.requestGameAsync(sharecode);
    
    // Crear el fichero match_info.json (async)
    const cleanedCode = sharecode.replace(/CSGO-|-/g, '');
    const matchInfoPath = path.join(DEMOS_DIR, `match_${cleanedCode}_info.json`);
    
    const matchInfoData = {
      match_id: result.matchID.toString(),
      sharecode: sharecode,
      date: result.matchDate,
      date_unix: result.matchTime,
      duration_seconds: result.matchDuration,
      map_name: result.mapName || '',
      demo_file: `match_${cleanedCode}.dem`
    };
    
    await fsPromises.writeFile(matchInfoPath, JSON.stringify(matchInfoData, null, 2));
    console.log(`📄 Match info guardado: ${matchInfoPath}`);
    
    return matchInfoData;
  } catch (err) {
    console.error(`❌ Error obteniendo metadata: ${err.message}`);
    throw err;
  }
}

/**
 * fetchMatchInfoForAllDemos:
 * --------------------------
 * Busca demos en DEMOS_DIR que no tengan su match_info.json correspondiente
 * y necesitan una lista de sharecodes para procesarlos.
 * Devuelve las demos que necesitan sus sharecodes.
 */
async function findDemosWithoutMatchInfo() {
  const files = await fsPromises.readdir(DEMOS_DIR);
  const demFiles = files.filter(f => f.endsWith('.dem'));
  const missing = [];
  
  for (const demFile of demFiles) {
    const infoFile = demFile.replace('.dem', '_info.json');
    const infoPath = path.join(DEMOS_DIR, infoFile);
    
    try {
      await fsPromises.access(infoPath);
    } catch {
      // File doesn't exist
      missing.push(demFile);
    }
  }
  
  console.log(`📊 Demos sin match_info.json: ${missing.length} de ${demFiles.length}`);
  return missing;
}

// Exportamos las funciones principales
module.exports = {
  client,
  csgo,
  botPool,
  progressEmitter,
  queue,        // Legacy alias → gcQueue
  gcQueue,
  downloadQueue,
  goQueue,
  iniciarSesionSteam,
  monitorearShareCodes,
  getRedisSubscriber,
  procesarShareCode,
  enqueueShareCode,
  fetchMatchInfoBySharecode,
  findDemosWithoutMatchInfo,
};
