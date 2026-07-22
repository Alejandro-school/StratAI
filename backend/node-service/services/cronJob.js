/**
 * cronJob.js
 * ----------
 * Periodic match detection for all registered users.
 * 
 * Every CRON_INTERVAL (default: 5 minutes), iterates over all_steam_ids
 * and fetches new sharecodes from Steam's API. New codes are pushed to
 * Redis which triggers the existing rpush → GC → Download → Go pipeline.
 * 
 * Calls Steam API directly (no Python proxy) to avoid session auth issues.
 */

const cron = require("node-cron");
const axios = require("axios");
const config = require("./config");
const { redisClient, ensureRedis } = require("./redisClient");
const { enqueueShareCode } = require("./steamDownloader");

const STEAM_API_URL = "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/";
const MAX_CODES_PER_USER = 50;
const MAX_RETRIES = 3;

let cronTask = null;
let isRunning = false;

/**
 * Fetch all new sharecodes for a single user directly from Steam API.
 * Returns the list of new sharecodes found.
 */
async function fetchSharecodesForUser(steamId) {
  const apiKey = config.steam.apiKey;
  if (!apiKey) {
    console.warn("⚠️ [CRON] STEAM_API_KEY not set — skipping sharecode fetch");
    return [];
  }

  const [authCode, knownCode] = await redisClient.mGet([
    `${steamId}:authCode`,
    `${steamId}:knownCode`,
  ]);

  if (!authCode || !knownCode) {
    return []; // User hasn't configured credentials yet
  }

  const sharecodes = [];
  let currentCode = knownCode;

  try {
    while (sharecodes.length < MAX_CODES_PER_USER) {
      let data = null;

      for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
        try {
          const resp = await axios.get(STEAM_API_URL, {
            params: {
              key: apiKey,
              steamid: steamId,
              steamidkey: authCode,
              knowncode: currentCode,
            },
            timeout: 10000,
          });
          data = resp.data;
          break;
        } catch (err) {
          if (err.response?.status === 412) {
            // Auth code expired or invalid — skip this user
            console.warn(`⚠️ [CRON] 412 for ${steamId}: auth code may be expired`);
            return sharecodes;
          }
          if (err.response?.status === 429) {
            // Rate limited — back off and retry
            const delay = 2000 * (attempt + 1);
            console.warn(`⚠️ [CRON] 429 for ${steamId}, waiting ${delay}ms...`);
            await new Promise((r) => setTimeout(r, delay));
            continue;
          }
          // Other errors — retry with backoff
          if (attempt < MAX_RETRIES - 1) {
            await new Promise((r) => setTimeout(r, 1000 * (attempt + 1)));
            continue;
          }
          console.error(`❌ [CRON] Failed to fetch sharecodes for ${steamId}: ${err.message}`);
          return sharecodes;
        }
      }

      if (!data) break;

      const nextCode = data?.result?.nextcode?.trim();
      if (!nextCode || ["n/a", "null", "none"].includes(nextCode.toLowerCase())) {
        break; // No more matches
      }

      sharecodes.push(nextCode);
      currentCode = nextCode;

      // Small delay between Steam API calls
      await new Promise((r) => setTimeout(r, 100));
    }
  } catch (err) {
    console.error(`❌ [CRON] Unexpected error for ${steamId}: ${err.message}`);
  }

  // Update Redis with latest known code
  if (sharecodes.length > 0) {
    await redisClient.set(`${steamId}:knownCode`, sharecodes[sharecodes.length - 1]);
    console.log(`📌 [CRON] ${steamId}: ${sharecodes.length} new sharecodes found`);
  }

  return sharecodes;
}

/**
 * Main cron tick: process all registered users.
 */
async function cronTick() {
  if (isRunning) {
    console.log("⏭️ [CRON] Previous tick still running, skipping...");
    return;
  }

  isRunning = true;
  const startTime = Date.now();

  try {
    await ensureRedis();

    // Get all registered users
    const steamIds = await redisClient.sMembers("all_steam_ids");
    if (!steamIds || steamIds.length === 0) {
      return;
    }

    console.log(`⏰ [CRON] Starting match detection for ${steamIds.length} users...`);

    let totalNewCodes = 0;
    let usersWithNewCodes = 0;

    for (const steamId of steamIds) {
      try {
        const newCodes = await fetchSharecodesForUser(steamId);

        if (newCodes.length > 0) {
          // Check which codes are actually new (not already processed)
          const statusMap = await redisClient.hGetAll(`sharecode_status:${steamId}`);
          let enqueued = 0;

          for (const code of newCodes) {
            const status = statusMap[code];
            if (!status || status === "pending") {
              enqueueShareCode(code, steamId, { priority: 0 });
              enqueued++;
            }
          }

          if (enqueued > 0) {
            totalNewCodes += enqueued;
            usersWithNewCodes++;
          }
        }

        // Stagger between users to avoid Steam API rate limits
        if (steamIds.indexOf(steamId) < steamIds.length - 1) {
          await new Promise((r) => setTimeout(r, config.cron.userDelay));
        }
      } catch (err) {
        console.error(`❌ [CRON] Error processing user ${steamId}: ${err.message}`);
      }
    }

    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
    if (totalNewCodes > 0) {
      console.log(`✅ [CRON] Completed in ${elapsed}s: ${totalNewCodes} new codes from ${usersWithNewCodes} users`);
    } else {
      console.log(`✅ [CRON] Completed in ${elapsed}s: no new matches`);
    }
  } catch (err) {
    console.error(`❌ [CRON] Fatal error in tick: ${err.message}`);
  } finally {
    isRunning = false;
  }
}

/**
 * Start the cron job.
 */
function startCronJob() {
  if (!config.cron.enabled) {
    console.log("ℹ️ [CRON] Disabled via CRON_ENABLED=false");
    return;
  }

  if (!config.steam.apiKey) {
    console.warn("⚠️ [CRON] STEAM_API_KEY not set — cron job will not fetch sharecodes");
  }

  const interval = config.cron.interval;
  console.log(`⏰ [CRON] Starting match detection job (schedule: ${interval})`);

  cronTask = cron.schedule(interval, cronTick);

  // Run once immediately on startup (after a short delay for services to initialize)
  setTimeout(() => {
    console.log("⏰ [CRON] Running initial match detection...");
    cronTick();
  }, 15000); // 15s after startup
}

/**
 * Stop the cron job (for graceful shutdown).
 */
function stopCronJob() {
  if (cronTask) {
    cronTask.stop();
    cronTask = null;
    console.log("🛑 [CRON] Job stopped");
  }
}

module.exports = { startCronJob, stopCronJob, cronTick, fetchSharecodesForUser };
