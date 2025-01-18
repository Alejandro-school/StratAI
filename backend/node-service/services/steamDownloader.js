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
const { ShareCode } = require('globaloffensive-sharecode');
const Language = require('globaloffensive/language.js');
const Protos = require('globaloffensive/protobufs/generated/_load.js');



// Configuración básica de credenciales
const BOT_USERNAME = process.env.BOT_USERNAME;
const BOT_PASSWORD = process.env.BOT_PASSWORD;
const BOT_SHARED_SECRET = process.env.BOT_SHARED_SECRET;

if (!BOT_USERNAME || !BOT_PASSWORD || !BOT_SHARED_SECRET) {
  console.error('❌ Error: Faltan credenciales en el archivo .env');
  process.exit(1);
}


// Carpeta donde se guardarán las demos
const DEMOS_DIR = path.join(__dirname, '../../data/demos');
if (!fs.existsSync(DEMOS_DIR)) {
  fs.mkdirSync(DEMOS_DIR, { recursive: true });
}

// Creamos una cola para gestionar descargas secuenciales
const queue = new PQueue({ 
  concurrency: 1,     // procesar de 1 en 1
  interval: 5000,     // cada 3 segundos
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
    console.log(`🟢 Estado de la sesión con GC: ${csgo.haveGCSession}`);
  });
  
  csgo.on('disconnectedFromGC', (reason) => {
    console.error(`❌ Desconectado del Game Coordinator: ${reason}`);
  });
}

// Realiza la petición al GC para obtener la URL de la demo
// Solicitar la demo con reintentos controlados
// 🔄 Función optimizada para solicitar la demo al GC con reintentos
GlobalOffensive.prototype.requestGameAsync = function (shareCodeOrDetails, intentosMaximos = 5) {
  return new Promise((resolve, reject) => {
    let shareCodeDecoded;

    // 🔍 Decodificar el ShareCode
    if (typeof shareCodeOrDetails === 'string') {
      try {
        shareCodeDecoded = (new ShareCode(shareCodeOrDetails)).decode();
        console.log(`📝 ShareCode decodificado correctamente:
        🔑 Match ID: ${shareCodeDecoded.matchId}
        📊 Outcome ID: ${shareCodeDecoded.outcomeId}
        🛡️ Token: ${shareCodeDecoded.token}`);
      } catch (err) {
        console.error(`❌ Error al decodificar el ShareCode: ${err.message}`);
        return reject(err);
      }
    }

    let intentos = 0;

    const solicitarDemo = () => {
      if (intentos >= intentosMaximos) {
        console.error('❌ Se alcanzó el máximo de reintentos.');
        return reject(new Error('No se pudo obtener la URL de la demo.'));
      }

      intentos++;
      console.log(`🔄 Intento ${intentos} de ${intentosMaximos}`);

      if (!this.haveGCSession) {
        console.warn('⚠️ Sin sesión activa con el GC. Reintentando en 5s...');
        return setTimeout(solicitarDemo, 5000);
      }

      // 🔔 Solicitar demo al GC
      this._send(Language.MatchListRequestFullGameInfo, Protos.CMsgGCCStrike15_v2_MatchListRequestFullGameInfo, {
        matchid: shareCodeDecoded.matchId,
        outcomeid: shareCodeDecoded.outcomeId,
        token: shareCodeDecoded.token
      });



      const respuestaGC = (matches) => {
        // Filtrar solo los matches que tienen un 'map' no nulo y que empieza con 'http'
        const partidasValidas = matches.filter(match => 
          match.roundstatsall &&                      // Asegura que roundstatsall existe
          Array.isArray(match.roundstatsall) &&      // Verifica que es un arreglo
          match.roundstatsall.some(round =>          // Revisa si al menos una ronda tiene un map válido
            round.map && typeof round.map === 'string' && round.map.startsWith('http')
          )
        );
      
        console.log(partidasValidas);
      
        // ⚠️ Si no hay mapas válidos, reintentar
        if (!partidasValidas.length) {
          console.warn(`⚠️ No se encontró URL válida. Reintentando en 5s...`);
          if (intentos < intentosMaximos) {
            return setTimeout(solicitarDemo, 5000);
          }
          return reject(new Error('No se encontró la URL de la demo.'));
        }
      
        // ✅ Extraer la última URL válida
        const demoUrl = partidasValidas
          .flatMap(match => match.roundstatsall)
          .reverse()
          .find(round => round.map && typeof round.map === 'string' && round.map.startsWith('http')).map;
      
          const demoUrlSegura = demoUrl.replace(/^http:/, 'https:');
          console.log(`✅ URL de la demo detectada: ${demoUrlSegura}`);
          this.removeListener('matchList', respuestaGC);
          resolve(demoUrlSegura);
      };
      
      // 🎯 Escuchar respuesta solo una vez
      this.once('matchList', respuestaGC);
    };

    // 🚀 Iniciar solicitud
    solicitarDemo();
  });
};

// 📥 Función para descargar el archivo .dem.bz2
function descargarFicheroHTTP(url, filePath) {
  return new Promise((resolve, reject) => {
    console.log(`⬇️ Descargando desde: ${url}`);

    if (fs.existsSync(filePath)) {
      console.log(`📂 La demo ya existe en: ${filePath}`);
      return resolve(filePath);
    }

    const fileStream = fs.createWriteStream(filePath);
    const req = https.get(url, { rejectUnauthorized: false }, (res) => {
      if (res.statusCode !== 200) {
        console.error(`❌ Error al descargar. HTTP: ${res.statusCode}`);
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
      console.error(`❌ Error en la descarga: ${err}`);
      fileStream.close();
      fs.unlink(filePath, () => {});
      reject(err);
    });
  });
}

// 🔄 Procesar ShareCode mejorado
async function procesarShareCode(sharecode, steamID) {
  console.log(`🔍 Procesando ShareCode ${sharecode} para SteamID ${steamID}...`);

  if (!csgo.haveGCSession) {
    console.warn('⚠️ No conectado al GC. Reintentando...');
    return setTimeout(() => procesarShareCode(sharecode, steamID), 5000);
  }

  try {
    const demoUrl = await csgo.requestGameAsync(sharecode);
    const cleanedCode = sharecode.replace(/CSGO-|-/g, '');
    const filePath = path.join(DEMOS_DIR, `match_${cleanedCode}.dem`);
    await descargarFicheroHTTP(demoUrl, filePath);
    console.log(`✅ Demo descargada correctamente: ${filePath}`);
  } catch (err) {
    console.error(`❌ Error en el proceso del ShareCode: ${sharecode}`, err);
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

      queue.on('active', (task) => {
        console.log('🔄 Procesando tarea en la cola...');
      });

      // Cuando termine la cola
      queue.onIdle().then(() => {
        console.log('✅ Todas las tareas en la cola han sido procesadas.');
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

  // Levantar el servidor en puerto 4000
  const PORT = process.env.PORT || 4000;
  app.listen(PORT, () => {
    console.log(`🚀 Servidor Node.js corriendo en http://localhost:${PORT}`);
  });
})();
