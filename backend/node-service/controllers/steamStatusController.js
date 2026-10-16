const express = require('express');
const { redisClient } = require('../services/redisClient');
const { botPool } = require('../services/steamGateway');
const { internalOnly } = require('../middleware/security');

const router = express.Router();

// Operational diagnostics only. History access is authorized by the user's
// session and history code, independently of the bot's social relationships.
router.get('/steam/status', internalOnly, (_req, res) => {
  const bot = botPool.getPrimaryBot();
  res.json({
    logged_in: Boolean(bot?.client?.steamID),
    bot_steam_id: bot?.client?.steamID?.getSteamID64() || null,
    gc_session: Boolean(bot?.csgo?.haveGCSession),
    redis_ready: Boolean(redisClient.isReady),
  });
});

module.exports = router;
