const cron = require('node-cron');
const axios = require('axios');
const config = require('./config');
const { redisClient, ensureRedis } = require('./redisClient');

async function fetchAllUsersSharecodes() {
  console.log('⏰ [CRON] Iniciando consulta de nuevos sharecodes.');
  try {
    await ensureRedis();

    const steamIDs = await redisClient.sMembers('all_steam_ids');
    if (!steamIDs.length) {
      console.log('⚠️ [CRON] No hay usuarios activos en all_steam_ids.');
      return;
    }

    for (const steam_id of steamIDs) {
      // Batch Redis reads con mGet
      const [authCode, knownCode] = await redisClient.mGet([
        `${steam_id}:authCode`,
        `${steam_id}:knownCode`
      ]);
      if (!authCode || !knownCode) {
        console.log(`⚠️ [CRON] Usuario ${steam_id} sin authCode/knownCode.`);
        continue;
      }
      try {
        const response = await axios.get(`${config.services.pythonService}/steam/all-sharecodes`, {
          params: { auth_code: authCode, known_code: knownCode },
          withCredentials: true,
          headers: { Cookie: `session=${steam_id}` },
          timeout: config.http.timeout
        });
        
        // Log si se encontraron nuevos sharecodes
        if (response.data?.new_codes?.length > 0) {
          console.log(`🆕 [CRON] ${response.data.new_codes.length} nuevos sharecodes para ${steam_id}`);
        }
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
  cron.schedule(config.cron.interval, fetchAllUsersSharecodes);
  console.log(`🕒 [CRON] Tarea programada: ${config.cron.interval}.`);
}

module.exports = { iniciarCron };
