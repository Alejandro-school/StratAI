require('dotenv').config({ path: __dirname + '/../.env' });
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
const http = require('http');


// Credenciales del bot en .env
const BOT_USERNAME = process.env.BOT_USERNAME;
const BOT_PASSWORD = process.env.BOT_PASSWORD;
const BOT_SHARED_SECRET = process.env.BOT_SHARED_SECRET;

if (!BOT_USERNAME || !BOT_PASSWORD || !BOT_SHARED_SECRET) {
  console.error('❌ Error: Faltan credenciales del bot en .env');
  process.exit(1);
}

// Directorio donde se guardan las demos
const DEMOS_DIR = path.join(__dirname, '../../data/demos');
if (!fs.existsSync(DEMOS_DIR)) {
  fs.mkdirSync(DEMOS_DIR, { recursive: true });
}

// Control de reintentos por sharecode
const reintentosSharecode = {};

// Cola para procesar sharecodes secuencialmente
const queue = new PQueue({
  concurrency: 1,
  interval: 2000,
  intervalCap: 1
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
 * Solicita al GC la URL de la demo correspondiente a un sharecode.
 */
GlobalOffensive.prototype.requestGameAsync = function (shareCodeStr, intentosMaximos = 2) {
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
              console.warn('⚠️ Sin sesión GC. Reintentando en 2s...');
              return setTimeout(solicitarDemo, 2000);
          }

          // Enviar petición al GC
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
                      match.roundstatsall.some(round => round.map && round.map.startsWith('http'))
              );

              if (!partidasValidas.length) {
                  console.warn('⚠️ No se encontró URL válida para la demo. Reintentamos...');
                  if (intentos < intentosMaximos) {
                      setTimeout(solicitarDemo, 1000);
                  } else {
                      reject(new Error('No se encontró la URL de la demo (partida caducada).'));
                  }
                  this.removeListener('matchList', onMatchList);
                  return;
              }

              // Se toma la primera partida válida
              const matchData = partidasValidas[0];
              const demoUrl = matchData.roundstatsall.find(round => round.map && round.map.startsWith('http')).map;

              // Extraer duración de la partida
              const lastRound = matchData.roundstatsall[matchData.roundstatsall.length - 1];
              const matchDuration = lastRound ? lastRound.match_duration || 0 : 0;

              this.removeListener('matchList', onMatchList);

              console.log(`✅ URL de la demo: ${demoUrl}`);
              console.log(`🕒 Duración de la partida: ${matchDuration} segundos`);

              resolve({
                  demoUrl: demoUrl,
                  matchDuration: matchDuration,
                  matchID: shareCodeDecoded.matchId
              });
          };

          this.once('matchList', onMatchList);
      };

      solicitarDemo();
  });
};

/**
 * descargarFicheroHTTP:
 * ---------------------
 * Descarga el fichero de demo y lo guarda en filePath.
 */
const { pipeline } = require('stream');

function descargarFicheroHTTP(url, filePath) {
  return new Promise((resolve, reject) => {
    console.log(`⬇️ Descargando: ${url}`);

    if (fs.existsSync(filePath)) {
      console.log(`📂 Ya existe: ${filePath}`);
      return resolve(filePath);
    }

    const esBz2 = url.endsWith('.bz2');
    const fileStream = fs.createWriteStream(filePath);
    const lib = url.startsWith('https') ? https : http;

    const req = lib.get(url, { rejectUnauthorized: false }, (res) => {
      if (res.statusCode !== 200) {
        fileStream.close();
        fs.unlink(filePath, () => {});
        return reject(new Error(`HTTP status: ${res.statusCode}`));
      }

      if (esBz2) {
        pipeline(
          res,
          unbzip2Stream(),
          fileStream,
          (err) => {
            if (err) {
              console.error('❌ Error en la descarga/descompresión:', err);
              fs.unlink(filePath, () => {});
              return reject(err);
            }
            console.log(`✅ Demo guardada en: ${filePath}`);
            resolve(filePath);
          }
        );
      } else {
        pipeline(
          res,
          fileStream,
          (err) => {
            if (err) {
              console.error('❌ Error en la descarga:', err);
              fs.unlink(filePath, () => {});
              return reject(err);
            }
            console.log(`✅ Demo guardada en: ${filePath}`);
            resolve(filePath);
          }
        );
      }
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
 * Procesa un sharecode:
 * 1. Solicita la URL y datos de la demo.
 * 2. Descarga la demo.
 * 3. Guarda datos básicos en Redis.
 * 4. Notifica a Go para analizar la demo.
 * 5. Marca el sharecode como "processed".
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
      // Paso 1: Solicitar la demo
      const { demoUrl, matchDuration, matchID } = await csgo.requestGameAsync(sharecode);

      // Paso 2: Descargar la demo
      const cleanedCode = sharecode.replace(/CSGO-|-/g, '');
      const filename = `match_${cleanedCode}.dem`;
      const filePath = path.join(DEMOS_DIR, filename);
      await descargarFicheroHTTP(demoUrl, filePath);

      // Paso 3: Guardar datos en Redis
      const matchData = {
          matchID: matchID,
          matchDuration: matchDuration
      };
      await redisClient.set(`match_data:${matchID}`, JSON.stringify(matchData), 'EX', 3600);
      console.log(`✅ Match data guardado en Redis: match_data:${matchID}`);

      // Paso 4: Notificar a Go para analizar la demo
      try {
          await axios.post('http://localhost:8080/process-downloaded-demo', {
              steam_id: steamID,
              filename,
              match_id: matchID
          });
          console.log(`✅ Stats de la demo ${filename} procesadas y almacenadas en Redis.`);
      } catch (err) {
          console.error(`❌ Error al llamar a Go: ${err.message}`);
      }

      // Paso 5: Marcar el sharecode como procesado
      await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, 'processed');
      reintentosSharecode[sharecode] = 0;

  } catch (err) {
      console.error(`❌ Error al procesar el ShareCode ${sharecode}:`, err);

      // Si la demo está caducada o no disponible, marcamos y no reintentamos
      if (err.message.includes('caducada') || err.message.includes('No se encontró la URL')) {
          console.log(`ℹ️ Marcando sharecode caducado: ${sharecode}`);
          await redisClient.hSet(`sharecode_status:${steamID}`, sharecode, 'caducado');
          return;
      }

      // Reintentos en caso de error
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
 * Se suscribe a eventos de Redis para detectar nuevos sharecodes en
 * la clave "sharecodes:{steamID}" y encola su procesamiento.
 */
async function monitorearShareCodes() {
  const subscriber = redisClient.duplicate();
  await subscriber.connect();
  console.log('📡 Escuchando rpush en Redis para nuevos ShareCodes...');

  // Activamos notificaciones de listas en Redis
  await subscriber.configSet('notify-keyspace-events', 'KEA');

  await subscriber.subscribe('__keyevent@0__:rpush', async (key) => {
    console.log(`\n🔔 rpush detectado en clave: ${key}`);
    if (key.startsWith('sharecodes:')) {
      const steamID = key.split(':')[1];

      // Leemos todos los sharecodes almacenados para el usuario
      const sharecodes = await redisClient.lRange(`sharecodes:${steamID}`, 0, -1);
      console.log(`📥 ShareCodes (SteamID: ${steamID}):`, sharecodes);

      // Procesamos solo los sharecodes que no estén marcados
      for (const code of sharecodes) {
        const status = await redisClient.hGet(`sharecode_status:${steamID}`, code);
        if (!status || status === 'pending') {
          queue.add(() => procesarShareCode(code, steamID));
        } else {
          console.log(`⚠️ Sharecode ${code} ya tiene estado "${status}". Se omite.`);
        }
      }

      // Log de estado de la cola
      queue.on('active', () => {
        console.log('🔄 Procesando siguiente tarea en la cola...');
      });
      queue.onIdle().then(() => {
        console.log('✅ Cola de descargas finalizada o en espera.');
      });
    }
  });
}

// Exportamos las funciones principales
module.exports = {
  client,
  csgo,
  queue,
  iniciarSesionSteam,
  monitorearShareCodes,
  procesarShareCode,
};