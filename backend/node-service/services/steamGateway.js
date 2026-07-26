const GlobalOffensive = require('globaloffensive');
const SteamUser = require('steam-user');
const { ShareCode } = require('globaloffensive-sharecode');
const Language = require('globaloffensive/language.js');
const Protos = require('globaloffensive/protobufs/generated/_load.js');
const config = require('./config');
const { botPool } = require('./botPool');
const { normalizeGcDemoUrl } = require('./demoDownload');
const { redisClient } = require('./redisClient');

botPool.initialize();

const primaryBot = botPool.getPrimaryBot();
const client = primaryBot?.client || new SteamUser();
const csgo = primaryBot?.csgo || new GlobalOffensive(client);
const pendingRequests = new Map();
let listenerConfigured = false;

function cacheKey(matchId) {
  return `${config.pipelineNamespace}:demo-url:${matchId}`;
}

async function readCachedResult(matchId) {
  const cached = await redisClient.get(cacheKey(matchId));
  if (!cached) return null;
  try {
    const result = JSON.parse(cached);
    return {
      ...result,
      demoUrl: normalizeGcDemoUrl(result.demoUrl).href,
    };
  } catch {
    await redisClient.del(cacheKey(matchId));
    return null;
  }
}

async function cacheResult(matchId, result) {
  await redisClient.set(cacheKey(matchId), JSON.stringify(result), {
    EX: config.ttl.demoUrl,
  });
}

function sendRequest(pending) {
  const bot = botPool.getAvailableBot();
  if (!bot) {
    clearTimeout(pending.timeout);
    pending.reject(new Error('No Steam GC session is available'));
    pendingRequests.delete(pending.matchId);
    return;
  }
  pending.bot = bot;
  bot.activeRequests += 1;
  bot.csgo._send(
    Language.MatchListRequestFullGameInfo,
    Protos.CMsgGCCStrike15_v2_MatchListRequestFullGameInfo,
    {
      matchid: pending.decoded.matchId,
      outcomeid: pending.decoded.outcomeId,
      token: pending.decoded.token,
    },
  );
}

function findDemoUrl(rounds) {
  for (const round of rounds || []) {
    if (typeof round.map !== 'string') continue;
    try {
      return normalizeGcDemoUrl(round.map).href;
    } catch {
      // Non-URL map names and unapproved hosts are not downloadable demos.
    }
  }
  return null;
}

function configureMatchListeners() {
  if (listenerConfigured) return;
  listenerConfigured = true;

  const onMatchList = (matches) => {
    for (const match of matches || []) {
      const matchId = match.matchid?.toString();
      const pending = pendingRequests.get(matchId);
      if (!pending) continue;

      const demoUrl = findDemoUrl(match.roundstatsall);
      if (!demoUrl) {
        if (pending.retries < pending.maxRetries) {
          pending.retries += 1;
          if (pending.bot) {
            pending.bot.activeRequests = Math.max(0, pending.bot.activeRequests - 1);
          }
          setTimeout(() => sendRequest(pending), 1000);
        } else {
          clearTimeout(pending.timeout);
          pendingRequests.delete(matchId);
          const error = new Error('Steam GC did not return a demo URL');
          error.pipelineCode = 'demo_url_unavailable';
          pending.reject(error);
        }
        continue;
      }

      const lastRound = match.roundstatsall[match.roundstatsall.length - 1];
      const matchTime = match.matchtime || 0;
      const result = {
        demoUrl,
        matchDuration: lastRound?.match_duration || 0,
        matchDate: matchTime > 0 ? new Date(matchTime * 1000).toISOString() : '',
        matchTime,
        mapName: lastRound?.reservation?.game_map_key || '',
        matchID: matchId,
      };
      clearTimeout(pending.timeout);
      pendingRequests.delete(matchId);
      if (pending.bot) {
        pending.bot.activeRequests = Math.max(0, pending.bot.activeRequests - 1);
      }
      cacheResult(matchId, result).catch(() => {});
      pending.resolve(result);
    }
  };

  for (const bot of botPool.bots) {
    bot.csgo.on('matchList', onMatchList);
  }
}

async function resolveSharecode(sharecode, maxRetries = 2) {
  configureMatchListeners();
  const decoded = new ShareCode(sharecode).decode();
  const matchId = decoded.matchId.toString();
  const cached = await readCachedResult(matchId);
  if (cached) return cached;
  if (pendingRequests.has(matchId)) return pendingRequests.get(matchId).promise;

  let resolvePromise;
  let rejectPromise;
  const promise = new Promise((resolve, reject) => {
    resolvePromise = resolve;
    rejectPromise = reject;
  });
  const pending = {
    matchId,
    decoded,
    maxRetries,
    retries: 0,
    promise,
    resolve: resolvePromise,
    reject: rejectPromise,
    bot: null,
  };
  pending.timeout = setTimeout(() => {
    pendingRequests.delete(matchId);
    if (pending.bot) {
      pending.bot.activeRequests = Math.max(0, pending.bot.activeRequests - 1);
    }
    rejectPromise(new Error(`Steam GC timed out for match ${matchId}`));
  }, 30000);
  pendingRequests.set(matchId, pending);
  sendRequest(pending);
  return promise;
}

async function iniciarSesionSteam() {
  if (botPool.bots.length === 0) {
    throw new Error('No Steam bot credentials are configured');
  }
  await botPool.loginAll();
}

module.exports = {
  botPool,
  client,
  csgo,
  findDemoUrl,
  iniciarSesionSteam,
  resolveSharecode,
};
