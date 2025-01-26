/**
 * cronJob.js
 * ----------
 * Tarea en segundo plano para buscar automáticamente nuevos sharecodes
 * de cada usuario y almacenarlos en Redis. El bot (steamDownloader) se
 * encargará de descargar y parsear.
 */

const cron = require('node-cron');
const axios = require('axios');
const redisClient = require('./redisClient');

/*
Para que este cron funcione, asumimos que:
 - Tienes una lista/set en Redis con los steam_ids "activos" 
   (p.ej. "all_steam_ids") o un método para obtenerlos.
 - Cada usuario guardó en Redis su "authCode" (steamid:authCode)
   y su "lastCode" (steamid:lastCode) que se usan para la API:
   GET /steam/all-sharecodes?auth_code=...&last_code=...
*/

async function fetchAllUsersSharecodes() {
  console.log('⏰ [CRON] Iniciando consulta de nuevos sharecodes.');

  try {
    // 1) Obtenemos la lista de steam_ids activos. 
    //    Ejemplo: un set "all_steam_ids". Ajusta si usas otra estrategia.
    const steamIDs = await redisClient.sMembers('all_steam_ids');
    if (!steamIDs.length) {
      console.log('⚠️ [CRON] No hay usuarios activos en all_steam_ids.');
      return;
    }

    // 2) Para cada steam_id, pedimos a la API de FastAPI que obtenga
    //    "all-sharecodes" con su auth_code y last_code
    for (const steam_id of steamIDs) {
      // Recuperamos authCode y lastCode de Redis
      const authCode = await redisClient.get(`${steam_id}:authCode`);
      const lastCode = await redisClient.get(`${steam_id}:lastCode`);
      if (!authCode || !lastCode) {
        console.log(`⚠️ [CRON] Usuario ${steam_id} no tiene authCode/lastCode en Redis.`);
        continue;
      }

      // Llamamos a la ruta Python => /steam/all-sharecodes
      // (Esa ruta ya se encarga de meter en Redis los nuevos sharecodes si los hay)
      console.log(`🔎 [CRON] Obteniendo sharecodes para ${steam_id} ...`);
      try {
        await axios.get('http://localhost:8000/steam/all-sharecodes', {
          params: {
            auth_code: authCode,
            last_code: lastCode
          },
          // Si tu FastAPI necesita cookies, adaptas "withCredentials" y "headers"
          // withCredentials: true
        });
      } catch (err) {
        console.error(`❌ [CRON] Error al obtener sharecodes para ${steam_id}:`, err.message);
      }
    }
  } catch (err) {
    console.error('❌ [CRON] Error general comprobando partidas:', err);
  }

  console.log('✅ [CRON] Finalizada la consulta de sharecodes.');
}


/*Programamos la tarea con node-cron:
 - '/5 * * * *' => se ejecuta cada 5 minutos
 - Ajusta según tu preferencia (p.ej. cada minuto '/1 * * * *')
*/
function iniciarCron() {
  cron.schedule('*/5 * * * *', fetchAllUsersSharecodes);
  console.log('🕒 [CRON] Tarea programada: cada 5 minutos.');
}

module.exports = {
  iniciarCron
};
