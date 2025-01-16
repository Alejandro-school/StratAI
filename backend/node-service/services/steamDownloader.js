require('dotenv').config();
const GlobalOffensive = require('globaloffensive');
const SteamUser = require('steam-user');
const express = require('express');
const SteamTotp = require('steam-totp');
const redis = require('redis');
const PQueue = require('p-queue').default;
const fs = require('fs');
const path = require('path');
const https = require('https');

// Configuración básica de credenciales
const BOT_USERNAME = process.env.BOT_USERNAME;
const BOT_PASSWORD = process.env.BOT_PASSWORD;
const BOT_SHARED_SECRET = process.env.BOT_SHARED_SECRET;

// Carpeta donde se guardarán las demos
const DEMOS_DIR = path.join(__dirname, '../../data/demos');
if (!fs.existsSync(DEMOS_DIR)) {
  fs.mkdirSync(DEMOS_DIR, { recursive: true });
}

// Creamos una cola para gestionar descargas secuenciales
const queue = new PQueue({ 
  concurrency: 1,     // procesar de 1 en 1
  interval: 3000,     // cada 3 segundos
  intervalCap: 1      // máximo 1 tarea por intervalo
});

// Inicializamos clientes de Steam y Redis
const client = new SteamUser();
const csgo = new GlobalOffensive(client);
const redisClient = redis.createClient();

// Servidor Express
const app = express();
app.use(express.json());

// Función para generar el código 2FA
function generarAuthCode() {
  return SteamTotp.generateAuthCode(BOT_SHARED_SECRET);
}

// Iniciar sesión de Steam
function iniciarSesionSteam() {
  client.logOn({
    accountName: BOT_USERNAME,
    password: BOT_PASSWORD,
    twoFactorCode: generarAuthCode(),
  });

  client.on('loggedOn', () => {
    console.log('✅ Bot conectado a Steam');
    client.setPersona(SteamUser.EPersonaState.Online);
    // Jugando CS:GO (AppID=730)
    client.gamesPlayed(730);
  });

  client.on('error', (err) => {
    console.error(`❌ Error en el cliente de Steam: ${err.message}`);
  });

  client.on('disconnected', (eresult, msg) => {
    console.error(`❌ Bot desconectado: ${msg}. Reintentando en 10 segundos...`);
    setTimeout(() => iniciarSesionSteam(), 10000);
  });

  csgo.on('connectedToGC', () => {
    console.log('🎮 Conectado al Game Coordinator de CS2/CS:GO');
  });

  csgo.on('disconnectedFromGC', (reason) => {
    console.error(`❌ Desconectado del Game Coordinator: ${reason}`);
  });
}

// Decodificar el share code (transforma CSGO-XXXXX-XXXXX-XXXXX en matchId, outcomeId y token)
function decodeShareCode(sharecode) {
  const cleaned = sharecode.replace('CSGO-', '').replace(/-/g, '');
  const ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';

  let result = 0n;
  for (let i = 0; i < cleaned.length; i++) {
    const index = ALPHABET.indexOf(cleaned[i]);
    if (index === -1) {
      throw new Error(`Carácter inválido en ShareCode: ${cleaned[i]}`);
    }
    result = result * 64n + BigInt(index);
  }

  const token = Number(result & 0xFFFFn);
  result >>= 16n;

  const outcomeId = (result & 0xFFFFFFFFn).toString();
  result >>= 32n;

  const matchId = result.toString();

  return { matchId, outcomeId, token };
}

// Realiza la petición al GC para obtener la URL de la demo
function requestMatchListPorSharecode(sharecode, intentos = 3) {
  return new Promise((resolve, reject) => {
    console.log(`➡️  Solicitando partida con ShareCode: ${sharecode}`);

    const { matchId, outcomeId, token } = decodeShareCode(sharecode);
    console.log(`🔍 matchId: ${matchId}, outcomeId: ${outcomeId}, token: ${token}`);

    csgo.requestGame({ matchId, outcomeId, token }, (err, response) => {
      if (err) {
          console.error(`❌ Error al solicitar la partida: ${err.message}`);
          return reject(err);
      }
  
      if (!response || !response.url) {
          console.error('❌ No se encontró la URL de la demo en la respuesta del GC. Respuesta:', response);
          return reject(new Error('No se encontró la URL de la demo.'));
      }
  
      console.log(`🔗 URL de la demo recibida: ${response.url}`);  // 🔍 Verificar la URL
      resolve(response.url);
  });
  
  
  });
}

// Descarga un fichero .dem.bz2 desde una URL HTTPS
function descargarFicheroHTTP(url, filePath) {
  return new Promise((resolve, reject) => {
    // Si ya existe, no lo descargamos de nuevo
    if (fs.existsSync(filePath)) {
      console.log(`📁 La demo ya existe en: ${filePath}`);
      return resolve(filePath);
    }

    console.log(`⬇️  Descargando demo desde: ${url}`);
    const fileStream = fs.createWriteStream(filePath);

    const req = https.get(url, (res) => {
      if (res.statusCode !== 200) {
        console.error(`❌ Error al descargar la demo. Código HTTP: ${res.statusCode}`);
        fileStream.close();
        fs.unlink(filePath, () => {});
        return reject(new Error(`HTTP status: ${res.statusCode}`));
      }

      res.pipe(fileStream);

      fileStream.on('finish', () => {
        fileStream.close();
        console.log(`✅ Demo guardada en: ${filePath}`);
        resolve(filePath);
      });
    });

    req.on('error', (err) => {
      console.error('❌ Error en la solicitud HTTPS:', err);
      fileStream.close();
      fs.unlink(filePath, () => {});
      reject(err);
    });
  });
}

// Función que pide la URL oficial al GC y luego descarga la demo
async function descargarDemoDesdeSharecode(sharecode) {
  const demoUrl = await requestMatchListPorSharecode(sharecode);
  // A veces en la respuesta del GC, la URL puede venir directa en "demoUrl"
  // o dentro de "replays[0].url". node-globaloffensive la expone en "response.url"
  // (arriba en requestMatchListPorSharecode). Asumimos demoUrl es la url final

  const cleaned = sharecode.replace('CSGO-', '').replace(/-/g, '');
  const fileName = `match_${cleaned}.dem.bz2`;  // Nombre de la demo
  const filePath = path.join(DEMOS_DIR, fileName);

  await descargarFicheroHTTP(demoUrl, filePath);
  return filePath;
}

// Procesa un sharecode, asegurándose de que exista la sesión con GC
async function procesarShareCode(sharecode, steamID) {
  console.log(`\n🔍 Procesando ShareCode ${sharecode} para SteamID ${steamID}...`);

  if (!csgo.haveGCSession) {
    console.error('❌ No está conectado al Game Coordinator. Esperando conexión...');
    return setTimeout(() => procesarShareCode(sharecode, steamID), 5000);
  }

  // Normalizar sharecode por si viene sin formatear
  const cleaned = sharecode.replace('CSGO-', '').replace(/-/g, '');
  const finalSharecode = `CSGO-${cleaned.slice(0,5)}-${cleaned.slice(5,10)}-${cleaned.slice(10)}`;

  try {
    const filePath = await descargarDemoDesdeSharecode(finalSharecode);
    console.log(`✅ Demo descargada: ${filePath}`);
  } catch (err) {
    console.error(`❌ Error en la descarga del sharecode: ${finalSharecode}`, err);
  }
}

// Suscriptor de Redis para detectar cuando se añaden nuevos sharecodes
async function monitorearShareCodes() {
  const subscriber = redisClient.duplicate();
  await subscriber.connect();

  console.log('📡 Escuchando cambios en Redis para nuevos ShareCodes...');
  await subscriber.configSet('notify-keyspace-events', 'KEA');

  // Suscribirse a eventos de push en cualquier lista
  await subscriber.subscribe('__keyevent@0__:rpush', async (key) => {
    console.log(`\n🔔 Detectado cambio en Redis con clave: ${key}`);

    if (key.startsWith('sharecodes:')) {
      const steamID = key.split(':')[1];
      const sharecodes = await redisClient.lRange(`sharecodes:${steamID}`, 0, -1);

      if (!sharecodes || !sharecodes.length) {
        console.warn(`⚠️ No se encontraron ShareCodes para SteamID: ${steamID}`);
        return;
      }

      console.log(`📥 Nuevos ShareCodes detectados para SteamID ${steamID}: ${sharecodes}`);
      for (const code of sharecodes) {
        queue.add(() => procesarShareCode(code, steamID));
      }

      // Cuando termine la cola
      queue.onIdle().then(() => {
        console.log(`✅ Todos los ShareCodes para SteamID ${steamID} han sido procesados.`);
      });
    }
  });
}

// Endpoint opcional para iniciar la descarga inmediatamente sin esperar al evento de Redis
app.post('/start-download', async (req, res) => {
  try {
    const { steam_id, sharecodes } = req.body;
    if (!steam_id || !sharecodes) {
      return res.status(400).json({ error: 'Faltan parámetros steam_id o sharecodes' });
    }

    console.log(`\n🌀 Iniciando descarga manual para SteamID ${steam_id} con sharecodes:`, sharecodes);
    for (const code of sharecodes) {
      queue.add(() => procesarShareCode(code, steam_id));
    }
    
    // Esperamos a que la cola termine (opcional)
    queue.onIdle().then(() => {
      console.log(`✅ Finalizó la descarga manual para SteamID ${steam_id}.`);
    });

    return res.json({ message: 'Descarga iniciada en segundo plano.' });
  } catch (error) {
    console.error('❌ Error en /start-download:', error);
    return res.status(500).json({ error: error.toString() });
  }
});

// Inicializar todo
(async () => {
  try {
    await redisClient.connect();
    console.log('✅ Conectado a Redis');
  } catch (error) {
    console.error('❌ Error conectando a Redis:', error);
    process.exit(1);
  }

  // Iniciar sesión en Steam
  iniciarSesionSteam();

  // Suscribirse a los eventos de Redis
  monitorearShareCodes();

  // Levantar el servidor en puerto 3000
  const PORT = process.env.PORT || 3000;
  app.listen(PORT, () => {
    console.log(`🚀 Servidor Node.js corriendo en http://localhost:${PORT}`);
  });
})();
