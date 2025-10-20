const cron = require('node-cron');
const axios = require('axios');
const { redisClient, ensureRedis } = require('./redisClient'); // 👈

const CRON_INTERVAL = process.env.CRON_INTERVAL || '*/5 * * * *';

async function fetchAllUsersSharecodes() {
  console.log('⏰ [CRON] Iniciando consulta de nuevos sharecodes.');
  try {
    await ensureRedis(); // 👈 garantiza conexión

    const steamIDs = await redisClient.sMembers('all_steam_ids'); // 👈 ya funciona
    if (!steamIDs.length) {
      console.log('⚠️ [CRON] No hay usuarios activos en all_steam_ids.');
      return;
    }

    for (const steam_id of steamIDs) {
      const authCode  = await redisClient.get(`${steam_id}:authCode`);
      const knownCode = await redisClient.get(`${steam_id}:knownCode`);
      if (!authCode || !knownCode) {
        console.log(`⚠️ [CRON] Usuario ${steam_id} sin authCode/knownCode.`);
        continue;
      }
      try {
        await axios.get('http://127.0.0.1:8000/steam/all-sharecodes', {
          params: { auth_code: authCode, known_code: knownCode },
          withCredentials: true,
          headers: { Cookie: `session=${steam_id}` }
        });
      } catch (err) {
        console.error(`❌ [CRON] Error con ${steam_id}:`, err.message);
      }
    }
  } catch (err) {
    console.error('❌ [CRON] Error general comprobando sharecodes:', err);
  }
  console.log('✅ [CRON] Finalizada la consulta de sharecodes.');
}

function iniciarCron() {
  cron.schedule(CRON_INTERVAL, fetchAllUsersSharecodes);
  console.log(`🕒 [CRON] Tarea programada: ${CRON_INTERVAL}.`);
}

module.exports = { iniciarCron };
