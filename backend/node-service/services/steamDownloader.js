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

// Cola para descargas (secuencial para evitar duplicados con Steam GC)
const queue = new PQueue({
  concurrency: 1,  // IMPORTANTE: 1 para evitar race conditions con Steam GC
  interval: config.queue.interval,
  intervalCap: config.queue.intervalCap,
  timeout: config.queue.timeout,
  throwOnTimeout: false,
});

queue.on('error', (err) => {
  console.error('❌ Error en cola de descargas:', err.message);
});

// Cola SEPARADA para procesamiento Go (10 paralelos para batch processing)
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

// Inicializamos el cliente de Steam y la instancia de CSGO
const client = new SteamUser();
const csgo = new GlobalOffensive(client);

// Flag para evitar duplicar listeners
let steamListenersSetup = false;

/**
 * Genera el código 2FA (Steam Guard)
 */
function generarAuthCode() {
  return SteamTotp.generateAuthCode(BOT_SHARED_SECRET);
}

/**
 * setupSteamListeners:
 * --------------------
 * Configura los event listeners de Steam/CSGO UNA SOLA VEZ
 */
function setupSteamListeners() {
  if (steamListenersSetup) return;
  steamListenersSetup = true;

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

  // Eventos del Game Coordinator
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
 * Inicia sesión en la cuenta del bot y se conecta al GC de CS:GO.
 */
function iniciarSesionSteam() {
  // Setup listeners una sola vez
  setupSteamListeners();

  client.logOn({
    accountName: BOT_USERNAME,
    password: BOT_PASSWORD,
    twoFactorCode: generarAuthCode(),
  });
}

/**
 * requestGameAsync:
 * -----------------
 * Solicita al GC la URL de la demo correspondiente a un sharecode.
 */
GlobalOffensive.prototype.requestGameAsync = function (
  shareCodeStr,
  intentosMaximos = 2
) {
  return new Promise((resolve, reject) => {
    // Timeout global para evitar promesas colgadas
    const globalTimeout = setTimeout(() => {
      cleanup();
      reject(new Error('Timeout: sin respuesta del GC en 30s'));
    }, 30000);

    const cleanup = () => {
      clearTimeout(globalTimeout);
      this.removeListener('matchList', onMatchList);
    };

    let shareCodeDecoded;
    try {
      shareCodeDecoded = new ShareCode(shareCodeStr).decode();
      console.log(
        `📝 ShareCode decodificado: ${JSON.stringify(shareCodeDecoded)}`
      );
    } catch (err) {
      cleanup();
      return reject(
        new Error(
          `No se pudo decodificar el sharecode: ${shareCodeStr} => ${err.message}`
        )
      );
    }

    let intentos = 0;
    const solicitarDemo = () => {
      if (intentos >= intentosMaximos) {
        cleanup();
        return reject(
          new Error("❌ Máximo de reintentos para obtener la URL de la demo.")
        );
      }
      intentos++;

      if (!this.haveGCSession) {
        console.warn(`⚠️ Sin sesión GC. Reintentando en ${config.retry.gcRetryDelay / 1000}s...`);
        return setTimeout(solicitarDemo, config.retry.gcRetryDelay);
      }

      // Enviar petición al GC
      this._send(
        Language.MatchListRequestFullGameInfo,
        Protos.CMsgGCCStrike15_v2_MatchListRequestFullGameInfo,
        {
          matchid: shareCodeDecoded.matchId,
          outcomeid: shareCodeDecoded.outcomeId,
          token: shareCodeDecoded.token,
        }
      );

      this.once("matchList", onMatchList);
    };

    const onMatchList = (matches) => {
      const partidasValidas = matches.filter(
        (match) =>
          match.roundstatsall &&
          Array.isArray(match.roundstatsall) &&
          match.roundstatsall.some(
            (round) => round.map && round.map.startsWith("http")
          )
      );

      if (!partidasValidas.length) {
        console.warn(
          "⚠️ No se encontró URL válida para la demo. Reintentamos..."
        );
        this.removeListener("matchList", onMatchList);
        if (intentos < intentosMaximos) {
          setTimeout(solicitarDemo, 1000);
        } else {
          cleanup();
          reject(
            new Error("No se encontró la URL de la demo (partida caducada).")
          );
        }
        return;
      }

      // Se toma la primera partida válida
      const matchData = partidasValidas[0];
      const demoUrl = matchData.roundstatsall.find(
        (round) => round.map && round.map.startsWith("http")
      ).map;

      // Extraer duración de la partida
      const lastRound =
        matchData.roundstatsall[matchData.roundstatsall.length - 1];
      const matchDuration = lastRound ? lastRound.match_duration || 0 : 0;

      // Extraer fecha de la partida (matchtime es Unix timestamp)
      const matchTime = matchData.matchtime || 0;
      const matchDate =
        matchTime > 0 ? new Date(matchTime * 1000).toISOString() : "";

      // Extraer mapa si está disponible
      const mapName = lastRound
        ? lastRound.reservation?.game_map_key || ""
        : "";

      cleanup();

      console.log(`✅ URL de la demo: ${demoUrl}`);
      console.log(`🕒 Duración de la partida: ${matchDuration} segundos`);
      console.log(`📅 Fecha de la partida: ${matchDate}`);

      resolve({
        demoUrl: demoUrl,
        matchDuration: matchDuration,
        matchDate: matchDate,
        matchTime: matchTime,
        mapName: mapName,
        matchID: shareCodeDecoded.matchId,
      });
    };

    solicitarDemo();
  });
};

/**
 * descargarFicheroHTTP:
 * ---------------------
 * Descarga el fichero de demo y lo guarda en filePath.
 */
const { pipeline } = require("stream");

function descargarFicheroHTTP(url, filePath) {
  return new Promise((resolve, reject) => {
    console.log(`⬇️ Descargando: ${url}`);

    if (fs.existsSync(filePath)) {
      console.log(`📂 Ya existe: ${filePath}`);
      return resolve(filePath);
    }

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
            console.error("❌ Error en la descarga/descompresión:", err);
            fs.unlink(filePath, () => {});
            return reject(err);
          }
          console.log(`✅ Demo guardada en: ${filePath}`);
          resolve(filePath);
        });
      } else {
        pipeline(res, fileStream, (err) => {
          if (err) {
            console.error("❌ Error en la descarga:", err);
            fs.unlink(filePath, () => {});
            return reject(err);
          }
          console.log(`✅ Demo guardada en: ${filePath}`);
          resolve(filePath);
        });
      }
    });

    req.on("error", (err) => {
      console.error(`❌ Error al descargar: ${err}`);
      fileStream.close();
      fs.unlink(filePath, () => {});
      reject(err);
    });
  });
}

/**
 * procesarShareCode:
 * ------------------
 * Procesa un sharecode:
 * 1. Solicita la URL y datos de la demo.
 * 2. Descarga la demo.
 * 3. Guarda datos básicos en Redis.
 * 4. Notifica a Go para analizar la demo.
 * 5. Marca el sharecode como "processed".
 */
async function procesarShareCode(sharecode, steamID, maxReintentos = 3) {
  console.log(`\n🔍 Procesando ShareCode ${sharecode} (SteamID: ${steamID})`);

  // LOCK ATÓMICO: Prevenir procesamiento duplicado con SETNX
  const lockKey = `lock:sharecode:${sharecode}`;
  const lockAcquired = await redisClient.setNX(lockKey, Date.now().toString());
  
  if (!lockAcquired) {
    // Otro worker ya está procesando este sharecode
    console.log(`⏭️ Sharecode ${sharecode} ya está siendo procesado por otro worker`);
    return;
  }
  
  // Establecer TTL en el lock (10 min máximo)
  await redisClient.expire(lockKey, 600);

  // Inicializar reintentos si no existe (usando Map)
  if (!reintentosSharecode.has(sharecode)) {
    reintentosSharecode.set(sharecode, { count: 0, timestamp: Date.now() });
  }

  // Esperar sesión GC con Promise en lugar de devolver setTimeout
  if (!csgo.haveGCSession) {
    console.warn(`⚠️ No hay sesión GC. Reintentamos en ${config.retry.gcSessionDelay / 1000}s...`);
    await redisClient.del(lockKey); // Liberar lock antes de reintentar
    await new Promise(resolve => setTimeout(resolve, config.retry.gcSessionDelay));
    return procesarShareCode(sharecode, steamID, maxReintentos);
  }

  try {
    // Paso 1: Solicitar la demo
    const { demoUrl, matchDuration, matchDate, matchTime, mapName, matchID } =
      await csgo.requestGameAsync(sharecode);

    // Paso 2: Descargar la demo
    const cleanedCode = sharecode.replace(/CSGO-|-/g, "");
    const filename = `match_${cleanedCode}.dem`;
    const filePath = path.join(DEMOS_DIR, filename);
    await descargarFicheroHTTP(demoUrl, filePath);

    // Paso 3: Guardar datos en Redis
    const matchData = {
      matchID: matchID,
      matchDuration: matchDuration,
      matchDate: matchDate,
      matchTime: matchTime,
    };
    await redisClient.set(`match_data:${matchID}`, JSON.stringify(matchData), {
      EX: config.ttl.matchData,
    });

    console.log(`✅ Match data guardado en Redis: match_data:${matchID}`);

    // Paso 4: Encolar procesamiento Go (PARALELO - no espera)
    // Esto permite que múltiples demos se procesen simultáneamente
    goQueue.add(async () => {
      try {
        console.log(`🔧 [Go] Iniciando procesamiento de ${filename}...`);
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
          console.log(`✅ [Go] Stats de ${filename} procesadas correctamente`);
          
          // IMPORTANTE: Registrar demo procesada en Redis para el Dashboard
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
          console.log(`📊 [Redis] Demo registrada en processed_demos:${steamID}`);
          
          // Invalidar caché del dashboard para que se recalcule con la nueva demo
          await redisClient.del(`dashboard_stats:${steamID}`);
          console.log(`🗑️ [Redis] Cache de dashboard invalidada para ${steamID}`);
        } else {
          console.warn(`⚠️ [Go] Respuesta inesperada para ${filename}:`, goResponse.data);
        }
      } catch (err) {
        console.error(`❌ [Go] Error procesando ${filename}: ${err.message}`);
      }
    });

    // Paso 5: Marcar el sharecode como procesado
    await redisClient.hSet(
      `sharecode_status:${steamID}`,
      sharecode,
      "processed"
    );
    // Limpieza del Map y lock tras éxito
    reintentosSharecode.delete(sharecode);
    await redisClient.del(lockKey);
  } catch (err) {
    console.error(`❌ Error al procesar el ShareCode ${sharecode}:`, err);

    // Si la demo está caducada o no disponible, marcamos y no reintentamos
    if (
      err.message.includes("caducada") ||
      err.message.includes("No se encontró la URL")
    ) {
      console.log(`ℹ️ Marcando sharecode caducado: ${sharecode}`);
      await redisClient.hSet(
        `sharecode_status:${steamID}`,
        sharecode,
        "caducado"
      );
      reintentosSharecode.delete(sharecode);
      await redisClient.del(lockKey);
      return;
    }

    // Reintentos en caso de error (usando Map)
    const retryData = reintentosSharecode.get(sharecode) || { count: 0, timestamp: Date.now() };
    retryData.count++;
    retryData.timestamp = Date.now();
    reintentosSharecode.set(sharecode, retryData);

    if (retryData.count < maxReintentos) {
      console.log(
        `♻️ Reintentando sharecode ${sharecode} (intento ${retryData.count} de ${maxReintentos})...`
      );
      queue.add(() => procesarShareCode(sharecode, steamID, maxReintentos));
    } else {
      console.error(
        `❌ Sharecode ${sharecode} alcanzó el máximo de reintentos.`
      );
      reintentosSharecode.delete(sharecode);
      await redisClient.del(lockKey);
    }
  }
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

    await redisSubscriber.configSet("notify-keyspace-events", "KEA");

    await redisSubscriber.subscribe("__keyevent@0__:rpush", async (key) => {
      if (!key.startsWith("sharecodes:")) return;

      const steamID = key.split(":")[1];
      console.log(`🔔 [Monitor] Nuevos sharecodes detectados para: ${steamID}`);
      
      try {
        // Obtener todos los sharecodes
        const sharecodes = await redisClient.lRange(`sharecodes:${steamID}`, 0, -1);
        if (!sharecodes.length) return;
        
        // Batch: obtener status de todos los sharecodes de una vez
        const statusMap = await redisClient.hGetAll(`sharecode_status:${steamID}`);
        
        let encolados = 0;
        for (const code of sharecodes) {
          const status = statusMap[code];
          if (!status || status === "pending") {
            queue.add(() => procesarShareCode(code, steamID));
            encolados++;
          }
        }
        
        if (encolados > 0) {
          console.log(`✅ [Monitor] ${encolados} sharecodes encolados para ${steamID}`);
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

  if (!csgo.haveGCSession) {
    throw new Error('No hay sesión GC activa. Asegúrate de que el bot esté conectado.');
  }

  try {
    const result = await csgo.requestGameAsync(sharecode);
    
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
  queue,
  goQueue,  // Nueva cola paralela para procesamiento Go (10 concurrent)
  iniciarSesionSteam,
  monitorearShareCodes,
  getRedisSubscriber,
  procesarShareCode,
  fetchMatchInfoBySharecode,
  findDemosWithoutMatchInfo,
};
