const express = require('express');
const config = require('../services/config');
const { redisClient, ensureRedis } = require('../services/redisClient');
const { botPool } = require('../services/steamGateway');
const { addFriend, getFriendStatus } = require('../services/friendship');
const { internalOnly, isValidSteamId } = require('../middleware/security');

const router = express.Router();

function friendCacheKey(steamId) {
  return `${config.pipelineNamespace}:friend-status:${steamId}`;
}

function friendCacheTimestampKey(steamId) {
  return `${config.pipelineNamespace}:friend-status-ts:${steamId}`;
}

function getPrimaryBotState() {
  const bot = botPool.getPrimaryBot();
  const client = bot?.client;
  return {
    bot,
    client,
    botSteamId: client?.steamID?.getSteamID64() || null,
    loggedIn: Boolean(client?.steamID),
    friendsReady: Boolean(bot?.friendsListReady),
  };
}

async function readCachedStatus(steamId) {
  const [status, cachedAt] = await redisClient.mGet([
    friendCacheKey(steamId),
    friendCacheTimestampKey(steamId),
  ]);
  return { status, cachedAt };
}

async function cacheStatus(steamId, status) {
  const ttl = status === 'pending'
    ? config.ttl.friendStatusPending
    : config.ttl.friendStatus;
  const cachedAt = new Date().toISOString();
  await Promise.all([
    redisClient.set(friendCacheKey(steamId), status, { EX: ttl }),
    redisClient.set(friendCacheTimestampKey(steamId), cachedAt, { EX: ttl }),
  ]);
  return cachedAt;
}

function friendResponse(status, botSteamId, extra = {}) {
  return {
    is_friend: status === 'friend',
    status,
    bot_steam_id: botSteamId,
    ...extra,
  };
}

router.get('/steam/status', internalOnly, (_req, res) => {
  const state = getPrimaryBotState();
  res.json({
    logged_in: state.loggedIn,
    bot_steam_id: state.botSteamId,
    friends_ready: state.friendsReady,
    gc_session: Boolean(state.bot?.csgo?.haveGCSession),
    redis_ready: Boolean(redisClient.isReady),
  });
});

router.get('/steam/check-friend', internalOnly, async (req, res) => {
  const steamId = req.query.steam_id;
  if (!isValidSteamId(steamId)) {
    return res.status(400).json({ error: 'Steam ID inválido' });
  }

  await ensureRedis();
  const state = getPrimaryBotState();
  if (!state.loggedIn || !state.friendsReady) {
    const cached = await readCachedStatus(steamId);
    return res.json(friendResponse(cached.status || 'unknown', state.botSteamId, {
      service_down: true,
      source: cached.status ? 'cache' : 'none',
      cached_at: cached.cachedAt,
    }));
  }

  const status = getFriendStatus(state.client.myFriends[steamId]);
  const cachedAt = await cacheStatus(steamId, status);
  return res.json(friendResponse(status, state.botSteamId, {
    service_down: false,
    source: 'live',
    cached_at: cachedAt,
  }));
});

router.post('/steam/send-friend-request', internalOnly, async (req, res) => {
  const steamId = req.body?.steam_id;
  if (!isValidSteamId(steamId)) {
    return res.status(400).json({ error: 'Steam ID inválido' });
  }

  await ensureRedis();
  const state = getPrimaryBotState();
  if (!state.loggedIn || !state.friendsReady) {
    return res.status(503).json({
      error: 'La lista de amigos del bot todavía no está disponible',
      error_code: 'friends_not_ready',
      bot_steam_id: state.botSteamId,
    });
  }

  const currentStatus = getFriendStatus(state.client.myFriends[steamId]);
  if (currentStatus === 'friend' || currentStatus === 'pending') {
    await cacheStatus(steamId, currentStatus);
    return res.json(friendResponse(currentStatus, state.botSteamId, {
      message: currentStatus === 'friend' ? 'Ya sois amigos' : 'La solicitud ya está pendiente',
    }));
  }
  if (currentStatus === 'blocked') {
    return res.status(409).json({
      error: 'No se puede enviar la solicitud porque existe un bloqueo en Steam',
      error_code: 'steam_user_blocked',
      bot_steam_id: state.botSteamId,
    });
  }

  try {
    const result = await addFriend(state.client, steamId);
    const relationshipStatus = getFriendStatus(state.client.myFriends[steamId]);
    const status = relationshipStatus === 'friend' ? 'friend' : 'pending';
    await cacheStatus(steamId, status);
    return res.json(friendResponse(status, state.botSteamId, {
      message: result.duplicate
        ? 'La amistad o la solicitud ya existía'
        : 'Solicitud de amistad enviada',
    }));
  } catch (error) {
    const statusCode = error.eresult === 40 ? 409 : 502;
    return res.status(statusCode).json({
      error: error.eresult === 40
        ? 'No se puede enviar la solicitud porque existe un bloqueo en Steam'
        : 'Steam no ha podido procesar la solicitud',
      error_code: error.eresult === 40 ? 'steam_user_blocked' : 'steam_request_failed',
      bot_steam_id: state.botSteamId,
    });
  }
});

module.exports = router;
