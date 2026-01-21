/**
 * friendController.js
 * -------------------
 * Verificación/solicitud de amistad + health del bot.
 */
const express = require('express');
const config = require('../services/config');
const { redisClient, ensureRedis } = require('../services/redisClient');
const { client, csgo } = require('../services/steamDownloader');
const SteamUser = require('steam-user');

const router = express.Router();

let friendsReady = false;
client.on('friendsList', () => {
  friendsReady = true;
  console.log('👥 friendsList cargada');
});
client.on('friendRelationship', () => {
  // pequeños cambios en caliente
  friendsReady = true;
});

function isBotLoggedIn() {
  return !!client.steamID;
}

function botHealth() {
  return {
    logged_in: isBotLoggedIn(),
    bot_steam_id: client.steamID ? client.steamID.getSteamID64() : null,
    friends_ready: friendsReady,
    gc_session: !!csgo?.haveGCSession,
    redis_ready: !!redisClient?.isReady
  };
}

// -------- HEALTH ----------
router.get('/steam/status', async (_req, res) => {
  res.json(botHealth());
});

// -------- CHECK FRIEND ----------
router.get('/steam/check-friend', async (req, res) => {
  const { steam_id } = req.query;
  if (!steam_id) return res.status(400).json({ error: 'Falta el Steam ID' });

  await ensureRedis();

  // Batch Redis reads con mGet
  const [cached, cachedTs] = await redisClient.mGet([
    `friend_status:${steam_id}`,
    `friend_status_ts:${steam_id}`
  ]);

  const health = botHealth();
  const serviceDown = !(health.logged_in && health.friends_ready);

  if (serviceDown) {
    // No rompemos: devolvemos cache si existe
    return res.json({
      is_friend: cached === 'friend',
      status: cached || 'unknown',
      service_down: true,
      source: cached ? 'cache' : 'none',
      cached_at: cachedTs || null
    });
  }

  // Amigos cargados: lectura fiable
  const relationship = client.myFriends[steam_id];
  const isFriend = (relationship === SteamUser.EFriendRelationship.Friend);
  const status = isFriend ? 'friend' : (cached === 'pending' ? 'pending' : 'not_friend');

  // Batch Redis writes con Promise.all
  await Promise.all([
    redisClient.set(`friend_status:${steam_id}`, status, { EX: config.ttl.friendStatus }),
    redisClient.set(`friend_status_ts:${steam_id}`, new Date().toISOString(), { EX: config.ttl.friendStatus })
  ]);

  return res.json({
    is_friend: isFriend,
    status,
    service_down: false,
    source: 'live'
  });
});

// -------- SEND FRIEND REQUEST ----------
async function sendFriendWithBackoff(steamId, maxAttempts = 3) {
  let attempt = 0;
  while (attempt < maxAttempts) {
    attempt++;
    try {
      await new Promise((resolve, reject) => {
        client.addFriend(steamId, (err) => (err ? reject(err) : resolve()));
      });
      return;
    } catch (err) {
      // backoff simple ante rate limit/transitorios
      if (attempt < maxAttempts) {
        const delay = 1000 * attempt;
        console.warn(`♻️ addFriend reintento ${attempt}/${maxAttempts} en ${delay}ms: ${err.message}`);
        await new Promise(r => setTimeout(r, delay));
      } else {
        throw err;
      }
    }
  }
}

router.post('/steam/send-friend-request', async (req, res) => {
  const { steam_id } = req.body || {};
  if (!steam_id) return res.status(400).json({ error: 'Falta el Steam ID' });

  await ensureRedis();

  const health = botHealth();
  if (!(health.logged_in && health.friends_ready)) {
    return res.status(503).json({ error: 'Bot no operativo. Inténtalo más tarde.' });
  }

  const cached = await redisClient.get(`friend_status:${steam_id}`);
  if (cached === 'friend') {
    return res.json({ message: 'Ya sois amigos', status: 'friend' });
  }
  if (cached === 'pending') {
    return res.json({ message: 'Solicitud ya enviada', status: 'pending' });
  }

  try {
    await sendFriendWithBackoff(steam_id);
    // Batch Redis writes con Promise.all
    await Promise.all([
      redisClient.set(`friend_status:${steam_id}`, 'pending', { EX: config.ttl.friendStatus }),
      redisClient.set(`friend_status_ts:${steam_id}`, new Date().toISOString(), { EX: config.ttl.friendStatus })
    ]);
    return res.json({ message: 'Solicitud de amistad enviada', status: 'pending' });
  } catch (err) {
    console.error(`❌ Error al enviar solicitud a ${steam_id}: ${err.message}`);
    return res.status(500).json({ error: 'Error al enviar la solicitud' });
  }
});

module.exports = router;
