require('dotenv').config();
const GlobalOffensive = require('globaloffensive');
const SteamUser = require('steam-user');
const SteamTotp = require('steam-totp');
const redis = require('redis');
const PQueue = require('p-queue').default;

const client = new SteamUser();
let csgo = new GlobalOffensive(client);

// 📦 Conectar con Redis
const redisClient = redis.createClient();
redisClient.connect()
  .then(() => console.log('✅ Conectado a Redis'))
  .catch(console.error);

const BOT_USERNAME = process.env.BOT_USERNAME;
const BOT_PASSWORD = process.env.BOT_PASSWORD;
const BOT_SHARED_SECRET = process.env.BOT_SHARED_SECRET;

// 📂 Configurar la cola de procesamiento
const queue = new PQueue({
  concurrency: 1,      // Solo 1 tarea a la vez
  interval: 3000,      // Intervalo de 3 segundos
  intervalCap: 1       // Máximo 1 tarea por intervalo
});

// 🔐 Generar código 2FA
function generarAuthCode() {
  return SteamTotp.generateAuthCode(BOT_SHARED_SECRET);
}

// 🚀 Iniciar sesión en Steam
function iniciarSesionSteam() {
  client.logOn({
    accountName: BOT_USERNAME,
    password: BOT_PASSWORD,
    twoFactorCode: generarAuthCode()
  });

  client.on('loggedOn', () => {
    console.log('✅ Bot conectado a Steam');
    client.setPersona(SteamUser.EPersonaState.Online);
    client.gamesPlayed(730);  // Simula jugar CS2
  });

  client.on('error', (err) => {
    console.error(`❌ Error en el cliente de Steam: ${err.message}`);
  });

  // ✅ Confirmar conexión al Game Coordinator
  csgo.on('connectedToGC', () => {
    console.log('🎮 Conectado al Game Coordinator de CS2');
  });

  csgo.on('disconnectedFromGC', (reason) => {
    console.error(`❌ Desconectado del Game Coordinator: ${reason}`);
  });
}


// 🔄 Monitorear cambios en los ShareCodes de Redis
async function monitorearShareCodes() {
  const subscriber = redisClient.duplicate();
  await subscriber.connect();

  console.log('📡 Escuchando cambios en Redis para nuevos ShareCodes...');

  await subscriber.configSet('notify-keyspace-events', 'KEA');

  await subscriber.subscribe('__keyevent@0__:rpush', async (key) => {
    console.log(`🔔 Detectado cambio en Redis con clave: ${key}`);

    if (key.startsWith('sharecodes:')) {
      const steamID = key.split(':')[1];
      const sharecodes = await redisClient.lRange(`sharecodes:${steamID}`, 0, -1);

      if (sharecodes.length === 0) {
        console.warn(`⚠️ No se encontraron ShareCodes para SteamID: ${steamID}`);
        return;
      }

      console.log(`📥 Nuevos ShareCodes detectados para SteamID ${steamID}: ${sharecodes}`);

      // ➕ Añadir cada ShareCode a la cola
      sharecodes.forEach((code) => {
        queue.add(() => obtenerEstadisticasDesdeShareCode(code, steamID));
      });
    }
  });
}

// 📊 Obtener estadísticas desde el ShareCode con control de velocidad
async function obtenerEstadisticasDesdeShareCode(sharecode, steamID) {
  return new Promise((resolve, reject) => {
    console.log(`🔍 Procesando ShareCode ${sharecode} para SteamID ${steamID}`);

    let matchInfo;
    try {
      matchInfo = decodeSharecode(sharecode);
    } catch (error) {
      console.error(`❌ Error al decodificar ShareCode: ${error.message}`);
      return reject(error);
    }

    let timeout;

    // 🟦 Escuchar la respuesta del Game Coordinator
    csgo.on('matchList', (matches) => {
      clearTimeout(timeout);

      if (!matches || matches.length === 0) {
        console.error('❌ No se encontraron partidas.');
        return reject(new Error('No se encontraron partidas.'));
      }

      console.log(`📊 Partida encontrada para el ShareCode: ${sharecode}`);
      resolve(matches);
    });

    // ⏳ Timeout si no hay respuesta
    timeout = setTimeout(() => {
      console.error(`⏳ Timeout: No se recibió respuesta para el ShareCode ${sharecode}`);
      reject(new Error(`Timeout al procesar el ShareCode ${sharecode}`));
    }, 15000);  // 15 segundos

    // 🟩 Solicitud al Game Coordinator
    csgo.requestGame(sharecode, (err, data) => {
      if (err) {
        console.error(`❌ Error al solicitar datos del match: ${err}`);
        clearTimeout(timeout);
        return reject(err);
      }

      console.log(`📨 Solicitud enviada para el ShareCode: ${sharecode}`);
    });
  });
}

// 🔑 Decodificar ShareCode
function decodeSharecode(sharecode) {
  const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
  const cleaned = sharecode.replace("CSGO-", "").replace(/-/g, "");
  let value = 0n;

  for (let i = cleaned.length - 1; i >= 0; i--) {
    const index = ALPHABET.indexOf(cleaned[i]);
    if (index === -1) {
      throw new Error(`❌ Carácter inválido en ShareCode: ${cleaned[i]}`);
    }
    value = value * 64n + BigInt(index);
  }

  const token = Number(value & 0xFFFFn);
  value >>= 16n;

  const outcomeId = Number(value & 0xFFFFFFFFn);
  value >>= 32n;

  const matchId = value.toString();

  return {
    matchId,
    outcomeId,
    token
  };
}

// 🚀 Iniciar el proceso
iniciarSesionSteam();
monitorearShareCodes();
