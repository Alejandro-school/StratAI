/**
 * friendController.js
 * -------------------
 * Verificación/solicitud de amistad + health del bot.
 */
const express = require('express');
const config = require('../services/config');
const { redisClient, ensureRedis } = require('../services/redisClient');
const { client, csgo } = require('../services/steamDownloader');
const { internalOnly, isValidSteamId } = require('../middleware/security');
const SteamUser = require('steam-user');

const router = express.Router();

let friendsReady = false;
let friendsListTimeout = null;

client.on('friendsList', () => {
  friendsReady = true;
  if (friendsListTimeout) clearTimeout(friendsListTimeout);
  console.log('👥 friendsList cargada');
});

client.on('friendRelationship', async (steamID, relationship) => {
  friendsReady = true;
  const sid = steamID.getSteamID64();
  // Update Redis cache in real-time when friend status changes
  try {
    await ensureRedis();
    let status = 'not_friend';
    let ttl = config.ttl.friendStatus;
    if (relationship === SteamUser.EFriendRelationship.Friend) {
      status = 'friend';
    } else if (relationship === SteamUser.EFriendRelationship.RequestRecipient) {
      status = 'pending';
      ttl = config.ttl.friendStatusPending;
    }
    await Promise.all([
      redisClient.set(`friend_status:${sid}`, status, { EX: ttl }),
      redisClient.set(`friend_status_ts:${sid}`, new Date().toISOString(), { EX: ttl }),
    ]);
    console.log(`👥 friendRelationship: ${sid} → ${status}`);
  } catch (err) {
    console.error(`❌ Error updating friend status: ${err.message}`);
  }
});

// Fallback: if friendsList hasn't loaded after 10s, allow checks with cache
friendsListTimeout = setTimeout(() => {
  if (!friendsReady) {
    console.warn('⚠️ friendsList timeout (10s) — allowing checks with cache fallback');
    friendsReady = true;
  }
}, 10000);

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
router.get('/steam/status', internalOnly, async (_req, res) => {
  res.json(botHealth());
});

// -------- CHECK FRIEND ----------
router.get('/steam/check-friend', internalOnly, async (req, res) => {
  const { steam_id } = req.query;
  if (!steam_id || !isValidSteamId(steam_id)) {
    return res.status(400).json({ error: 'Steam ID inválido (se requieren 17 dígitos)' });
  }

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

  // Use short TTL for pending (re-checked via friendRelationship event), long for others
  const ttl = (status === 'pending') ? config.ttl.friendStatusPending : config.ttl.friendStatus;
  await Promise.all([
    redisClient.set(`friend_status:${steam_id}`, status, { EX: ttl }),
    redisClient.set(`friend_status_ts:${steam_id}`, new Date().toISOString(), { EX: ttl })
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
      // exponential backoff ante rate limit/transitorios
      if (attempt < maxAttempts) {
        const delay = Math.min(1000 * Math.pow(2, attempt - 1), 16000);
        console.warn(`♻️ addFriend reintento ${attempt}/${maxAttempts} en ${delay}ms: ${err.message}`);
        await new Promise(r => setTimeout(r, delay));
      } else {
        throw err;
      }
    }
  }
}

router.post('/steam/send-friend-request', internalOnly, async (req, res) => {
  const { steam_id } = req.body || {};
  if (!steam_id || !isValidSteamId(steam_id)) {
    return res.status(400).json({ error: 'Steam ID inválido (se requieren 17 dígitos)' });
  }

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
    // Batch Redis writes — use short TTL for pending (re-checked quickly via friendRelationship event)
    await Promise.all([
      redisClient.set(`friend_status:${steam_id}`, 'pending', { EX: config.ttl.friendStatusPending }),
      redisClient.set(`friend_status_ts:${steam_id}`, new Date().toISOString(), { EX: config.ttl.friendStatusPending })
    ]);
    return res.json({ message: 'Solicitud de amistad enviada', status: 'pending' });
  } catch (err) {
    console.error(`❌ Error al enviar solicitud a ${steam_id}: ${err.message}`);
    return res.status(500).json({ error: 'Error al enviar la solicitud' });
  }
});

module.exports = router;
