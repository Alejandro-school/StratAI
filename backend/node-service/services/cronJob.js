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
 - Tienes un set en Redis con los steam_ids "activos" en la clave "all_steam_ids".
 - Cada usuario guardó en Redis su "authCode" (clave: steam_id:authCode)
   y su "knownCode" (clave: steam_id:knownCode) que se usan para la API:
   GET /steam/all-sharecodes?auth_code=...&known_code=...
*/

async function fetchAllUsersSharecodes() {
  console.log('⏰ [CRON] Iniciando consulta de nuevos sharecodes.');
  try {
    // Obtenemos la lista de steam_ids activos desde el set "all_steam_ids"
    const steamIDs = await redisClient.sMembers('all_steam_ids');
    if (!steamIDs.length) {
      console.log('⚠️ [CRON] No hay usuarios activos en all_steam_ids.');
      return;
    }
    // Para cada steam_id, solicitamos nuevos sharecodes a la API FastAPI
    for (const steam_id of steamIDs) {
      const authCode = await redisClient.get(`${steam_id}:authCode`);
      const knownCode = await redisClient.get(`${steam_id}:knownCode`);
      if (!authCode || !knownCode) {
        console.log(`⚠️ [CRON] Usuario ${steam_id} no tiene authCode/knownCode en Redis.`);
        continue;
      }
      console.log(`🔎 [CRON] Obteniendo sharecodes para ${steam_id} ...`);
      try {
        await axios.get('http://127.0.0.1:8000/steam/all-sharecodes', {
          params: {
            auth_code: authCode,
            known_code: knownCode
          },
          withCredentials: true,
          headers: {
            // Importante: necesitas mandarle el header Cookie con `session=...`
            'Cookie': `session=${steam_id}`
          }
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
  cron.schedule('*/2 * * * *', fetchAllUsersSharecodes);
  console.log('🕒 [CRON] Tarea programada: cada 5 minutos.');
}

module.exports = {
  iniciarCron
};
