// cronJob.js
const cron = require('node-cron');
const axios = require('axios');
const redisClient = require('./redisClient');

// Permite configurar el intervalo (por defecto cada 5 minutos)
const CRON_INTERVAL = process.env.CRON_INTERVAL || '*/5 * * * *';

async function fetchAllUsersSharecodes() {
  console.log('⏰ [CRON] Iniciando consulta de nuevos sharecodes.');
  try {
    // Obtenemos los steam_ids activos desde el set "all_steam_ids"
    const steamIDs = await redisClient.sMembers('all_steam_ids');
    if (!steamIDs.length) {
      console.log('⚠️ [CRON] No hay usuarios activos en all_steam_ids.');
      return;
    }
    // Procesamos cada usuario
    for (const steam_id of steamIDs) {
      const authCode = await redisClient.get(`${steam_id}:authCode`);
      const knownCode = await redisClient.get(`${steam_id}:knownCode`);
      if (!authCode || !knownCode) {
        console.log(`⚠️ [CRON] Usuario ${steam_id} no tiene authCode/knownCode en Redis.`);
        continue;
      }
      console.log(`🔎 [CRON] Obteniendo sharecodes para ${steam_id} con knownCode: ${knownCode} ...`);
      try {
        const response = await axios.get('http://127.0.0.1:8000/steam/all-sharecodes', {
          params: {
            auth_code: authCode,
            known_code: knownCode
          },
          withCredentials: true,
          headers: {
            // Se envía la cookie "session" con el steam_id
            'Cookie': `session=${steam_id}`
          }
        });
        // Si el endpoint indica que hay más sharecodes (has_more true), se reconsulta inmediatamente
        if (response.data && response.data.has_more) {
          console.log(`🔄 [CRON] Más sharecodes disponibles para ${steam_id}. Reconsulta inmediata.`);
          const updatedKnownCode = await redisClient.get(`${steam_id}:knownCode`);
          await axios.get('http://127.0.0.1:8000/steam/all-sharecodes', {
            params: {
              auth_code: authCode,
              known_code: updatedKnownCode
            },
            withCredentials: true,
            headers: {
              'Cookie': `session=${steam_id}`
            }
          });
        }
      } catch (err) {
        console.error(`❌ [CRON] Error al obtener sharecodes para ${steam_id}:`, err.message);
      }
    }
  } catch (err) {
    console.error('❌ [CRON] Error general comprobando sharecodes:', err);
  }
  console.log('✅ [CRON] Finalizada la consulta de sharecodes.');
}

function iniciarCron() {
  cron.schedule(CRON_INTERVAL, fetchAllUsersSharecodes);
  console.log(`🕒 [CRON] Tarea programada: cada ${CRON_INTERVAL}.`);
}

module.exports = {
  iniciarCron
};
