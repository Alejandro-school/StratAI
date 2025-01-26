/**
 * steamDownloader.js
 * ------------------
 * Módulo principal que administra:
 *  - Conexión a Steam y Game Coordinator (GC) de CS:GO/CS2.
 *  - Decodificación de sharecodes y descarga secuencial de .dem.
 *  - Comprobación si una demo sigue disponible. Si no, la marcamos "descartada".
 *  - Tras descargar, llama a Go para parsear la demo y guardar stats en Redis.
 */

require('dotenv').config();
const SteamUser = require('steam-user');
const GlobalOffensive = require('globaloffensive');
const SteamTotp = require('steam-totp');
const { default: PQueue } = require('p-queue');
const fs = require('fs');
const path = require('path');
const https = require('https');
const { ShareCode } = require('globaloffensive-sharecode');
const Language = require('globaloffensive/language.js');
const Protos = require('globaloffensive/protobufs/generated/_load.js');
const redisClient = require('./redisClient');
const unbzip2Stream = require('unbzip2-stream');
const axios = require('axios');

// Credenciales del bot en .env
const BOT_USERNAME = process.env.BOT_USERNAME;
const BOT_PASSWORD = process.env.BOT_PASSWORD;
const BOT_SHARED_SECRET = process.env.BOT_SHARED_SECRET;

// Verificación básica
if (!BOT_USERNAME || !BOT_PASSWORD || !BOT_SHARED_SECRET) {
  console.error('❌ Error: Faltan credenciales del bot en .env');
  process.exit(1);
}

// Directorio de demos
const DEMOS_DIR = path.join(__dirname, '../../data/demos');
if (!fs.existsSync(DEMOS_DIR)) {
  fs.mkdirSync(DEMOS_DIR, { recursive: true });
}

// Mapa para controlar reintentos por cada sharecode
const reintentosSharecode = {};

/*
Cola de descargas/procesos. 
 concurrency=2 => procesar hasta 2 sharecodes simultáneamente
 interval=3000, intervalCap=2 => en cada 3s se inician 2 tareas como máximo.
 Ajusta si quieres algo más rápido o más lento para no colapsar el GC.
*/
const queue = new PQueue({
  concurrency: 10,
  interval: 2000,
  intervalCap: 10
});

// Inicializamos el cliente de Steam y la instancia de CSGO
const client = new SteamUser();
const csgo = new GlobalOffensive(client);

/**
 * Genera el código 2FA (Steam Guard)
 */
function generarAuthCode() {
  return SteamTotp.generateAuthCode(BOT_SHARED_SECRET);
}

/**
 * iniciarSesionSteam:
 * -------------------
 * Inicia sesión en la cuenta del bot y se conecta al GC de CS:GO.
 */
function iniciarSesionSteam() {
  client.logOn({
    accountName: BOT_USERNAME,
    password: BOT_PASSWORD,
    twoFactorCode: generarAuthCode(),
  });

  client.on('loggedOn', () => {
    console.log('✅ Bot conectado a Steam');
    client.setPersona(SteamUser.EPersonaState.Online);
    // Jugando "CS:GO" (AppID=730)
    client.gamesPlayed(730);
  });

  client.on('error', (err) => {
    console.error(`❌ Error en cliente Steam: ${err.message}`);
  });

  client.on('disconnected', (eresult, msg) => {
    console.error(`❌ Bot desconectado: ${msg}. Reintentando en 10s...`);
    setTimeout(() => iniciarSesionSteam(), 10000);
  });

  // Eventos del Game Coordinator
  csgo.on('connectedToGC', () => {
    console.log('🎮 Conectado al GC de CS:GO');
    console.log(`🟢 Estado GC: ${csgo.haveGCSession}`);
  });

  csgo.on('disconnectedFromGC', (reason) => {
    console.error(`❌ Desconectado del GC: ${reason}`);
  });
}

/**
 * requestGameAsync:
 * -----------------
 * Envía una solicitud al GC para obtener la URL de la demo de un sharecode dado.
 * Si no la encuentra, lanza error => esa demo se considera "no disponible".
 */
GlobalOffensive.prototype.requestGameAsync = function (shareCodeStr, intentosMaximos = 5) {
  return new Promise((resolve, reject) => {
    let shareCodeDecoded;
    try {
      shareCodeDecoded = new ShareCode(shareCodeStr).decode();
      console.log(`📝 ShareCode decodificado: ${JSON.stringify(shareCodeDecoded)}`);
    } catch (err) {
      return reject(new Error(`No se pudo decodificar el sharecode: ${shareCodeStr} => ${err.message}`));
    }

    let intentos = 0;
    const solicitarDemo = () => {
      if (intentos >= intentosMaximos) {
        return reject(new Error('❌ Máximo de reintentos para obtener la URL de la demo.'));
      }
      intentos++;

      if (!this.haveGCSession) {
        console.warn('⚠️ Sin sesión GC. Reintentando en 5s...');
        return setTimeout(solicitarDemo, 5000);
      }

      // Petición al GC
      this._send(
        Language.MatchListRequestFullGameInfo,
        Protos.CMsgGCCStrike15_v2_MatchListRequestFullGameInfo,
        {
          matchid: shareCodeDecoded.matchId,
          outcomeid: shareCodeDecoded.outcomeId,
          token: shareCodeDecoded.token
        }
      );

      const onMatchList = (matches) => {
        const partidasValidas = matches.filter(
          match =>
            match.roundstatsall &&
            Array.isArray(match.roundstatsall) &&
            match.roundstatsall.some(
              round => round.map && round.map.startsWith('http')
            )
        );
        if (!partidasValidas.length) {
          console.warn('⚠️ No se encontró URL válida para la demo. Reintentamos en 5s...');
          if (intentos < intentosMaximos) {
            setTimeout(solicitarDemo, 5000);
          } else {
            reject(new Error('No se encontró la URL de la demo (partida caducada).'));
          }
          this.removeListener('matchList', onMatchList);
          return;
        }

        // Tomamos la última con map válido
        const demoUrl = partidasValidas
          .flatMap(match => match.roundstatsall)
          .reverse()
          .find(round => round.map && round.map.startsWith('http'))
          .map;

        this.removeListener('matchList', onMatchList);
        const demoUrlSegura = demoUrl.replace(/^http:/, 'https:');
        console.log(`✅ URL de la demo: ${demoUrlSegura}`);
        resolve(demoUrlSegura);
      };

      this.once('matchList', onMatchList);
    };

    solicitarDemo();
  });
};

/**
 * descargarFicheroHTTP:
 * ---------------------
 * Descarga un fichero .dem (o .bz2) vía HTTPS y lo guarda en filePath.
 * Si es .bz2, lo descomprime "al vuelo".
 */
function descargarFicheroHTTP(url, filePath) {
  return new Promise((resolve, reject) => {
    console.log(`⬇️ Descargando: ${url}`);
    if (fs.existsSync(filePath)) {
      console.log(`📂 Ya existe: ${filePath}`);
      return resolve(filePath);
    }
    const esBz2 = url.endsWith('.bz2');
    const fileStream = fs.createWriteStream(filePath);

    const req = https.get(url, { rejectUnauthorized: false }, (res) => {
      if (res.statusCode !== 200) {
        fileStream.close();
        fs.unlink(filePath, () => {});
        return reject(new Error(`HTTP status: ${res.statusCode}`));
      }
      // Descomprime si es .bz2
      if (esBz2) {
        res.pipe(unbzip2Stream()).pipe(fileStream);
      } else {
        res.pipe(fileStream);
      }
      fileStream.on('finish', () => {
        fileStream.close();
        console.log(`✅ Demo guardada en: ${filePath}`);
        resolve(filePath);
      });
    });

    req.on('error', (err) => {
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
 * 1) Verifica disponibilidad de la demo en el GC (requestGameAsync).
 * 2) Descarga el .dem.
 * 3) Llama a Go (POST /process-downloaded-demo) para parsear y guardar en Redis.
 */
async function procesarShareCode(sharecode, steamID, maxReintentos = 3) {
  console.log(`\n🔍 Procesando ShareCode ${sharecode} (SteamID: ${steamID})`);
  if (!reintentosSharecode[sharecode]) {
    reintentosSharecode[sharecode] = 0;
  }

  if (!csgo.haveGCSession) {
    console.warn('⚠️ No hay sesión GC. Reintentamos en 5s...');
    return setTimeout(() => procesarShareCode(sharecode, steamID, maxReintentos), 5000);
  }

  try {
    // 1) Petición al GC para ver si la demo está
    const demoUrl = await csgo.requestGameAsync(sharecode);
    // 2) Descargamos
    const cleanedCode = sharecode.replace(/CSGO-|-/g, '');
    const filename = `match_${cleanedCode}.dem`;
    const filePath = path.join(DEMOS_DIR, filename);

    await descargarFicheroHTTP(demoUrl, filePath);

    // 3) Llamar a Go para parsear
    try {
      await axios.post('http://localhost:8080/process-downloaded-demo', {
        steam_id: steamID,
        filename
      });
      console.log(`✅ Stats de la demo ${filename} procesadas y almacenadas en Redis.`);
    } catch (err) {
      console.error(`❌ Error al llamar a Go: ${err.message}`);
    }

    reintentosSharecode[sharecode] = 0;
  } catch (err) {
    console.error(`❌ Error al procesar el ShareCode ${sharecode}:`, err);

    // Si "No se encontró la URL" => es demo caducada
    if (err.message.includes('caducada') || err.message.includes('No se encontró la URL')) {
      // Podríamos marcar en Redis que este sharecode no está disponible, 
      // para no reintentarlo
      console.log(`ℹ️ Marcando sharecode caducado: ${sharecode}`);
      await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, 'caducado');
      return;
    }

    // Sino, reintentos
    reintentosSharecode[sharecode]++;
    if (reintentosSharecode[sharecode] < maxReintentos) {
      console.log(`♻️ Reintentando sharecode ${sharecode} (intento ${reintentosSharecode[sharecode]} de ${maxReintentos})...`);
      queue.add(() => procesarShareCode(sharecode, steamID, maxReintentos));
    } else {
      console.error(`❌ Sharecode ${sharecode} alcanzó el máximo de reintentos.`);
    }
  }
}

/**
 * monitorearShareCodes:
 * ---------------------
 * Suscribe a Redis para detectar rpush en sharecodes:steamID,
 * y encola la descarga+proceso para cada sharecode nuevo.
 */
async function monitorearShareCodes() {
  const subscriber = redisClient.duplicate();
  await subscriber.connect();
  console.log('📡 Escuchando rpush en Redis para nuevos ShareCodes...');

  // Activamos eventos de listas
  await subscriber.configSet('notify-keyspace-events', 'KEA');

  await subscriber.subscribe('__keyevent@0__:rpush', async (key) => {
    console.log(`\n🔔 rpush detectado en clave: ${key}`);
    if (key.startsWith('sharecodes:')) {
      const steamID = key.split(':')[1];
      const sharecodes = await redisClient.lRange(`sharecodes:${steamID}`, 0, -1);
      if (!sharecodes.length) {
        console.warn(`⚠️ No hay sharecodes en Redis para: ${steamID}`);
        return;
      }
      console.log(`📥 ShareCodes (SteamID: ${steamID}):`, sharecodes);

      for (const code of sharecodes) {
        // Opcional: comprobar si ya se marcó "caducado" en sharecode_status
        const status = await redisClient.hGet(`sharecode_status:${steamID}`, code);
        if (status === 'caducado') {
          console.log(`⚠️ Sharecode ${code} ya marcado como caducado. Saltamos.`);
          continue;
        }
        // Encolamos la descarga y parseo
        queue.add(() => procesarShareCode(code, steamID));
      }

      queue.on('active', () => {
        console.log('🔄 Procesando siguiente tarea en la cola...');
      });
      queue.onIdle().then(() => {
        console.log('✅ Cola de descargas finalizada o en espera.');
      });
    }
  });
}

// Exportar funciones principales
module.exports = {
  client,
  csgo,
  queue,
  iniciarSesionSteam,
  monitorearShareCodes,
  procesarShareCode,
};
