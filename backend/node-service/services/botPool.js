/**
 * botPool.js
 * ----------
 * Pool de bots Steam para paralelizar GC queries.
 * 
 * Cada bot tiene su propia conexión Steam + GC session independiente.
 * El pool distribuye requests con round-robin entre bots con GC session activa.
 * 
 * Configuración via .env:
 *   BOT_ACCOUNTS=user1:pass1:secret1,user2:pass2:secret2,...
 *   GC_PER_BOT=3  (GC requests concurrentes por bot)
 * 
 * Si BOT_ACCOUNTS no está definido, usa las credenciales legacy individuales
 * (BOT_USERNAME, BOT_PASSWORD, BOT_SHARED_SECRET) como bot único.
 */

const SteamUser = require("steam-user");
const GlobalOffensive = require("globaloffensive");
const SteamTotp = require("steam-totp");
const config = require("./config");
const { redisClient, ensureRedis } = require("./redisClient");

class BotInstance {
  constructor(username, password, sharedSecret, id) {
    this.id = id;
    this.username = username;
    this.password = password;
    this.sharedSecret = sharedSecret;
    this.client = new SteamUser();
    this.csgo = new GlobalOffensive(this.client);
    this.ready = false;
    this.friendsListReady = false;
    this.activeRequests = 0;
    this._setupListeners();
  }

  _setupListeners() {
    this.client.on("loggedOn", () => {
      console.log(`✅ [Bot ${this.id}] ${this.username} conectado a Steam`);
      this.client.setPersona(SteamUser.EPersonaState.Online);
      this.client.gamesPlayed(730);
    });

    this.client.on("error", (err) => {
      console.error(`❌ [Bot ${this.id}] Error: ${err.message}`);
      this.ready = false;
    });

    this.client.on("disconnected", (eresult, msg) => {
      console.error(`❌ [Bot ${this.id}] Desconectado: ${msg}. Reconectando en ${config.retry.steamReconnect / 1000}s...`);
      this.ready = false;
      setTimeout(() => this.login(), config.retry.steamReconnect);
    });

    this.csgo.on("connectedToGC", () => {
      console.log(`🎮 [Bot ${this.id}] Conectado al GC (${this.username})`);
      this.ready = true;
    });

    this.csgo.on("disconnectedFromGC", (reason) => {
      console.error(`❌ [Bot ${this.id}] Desconectado del GC: ${reason}`);
      this.ready = false;
    });

    // Detect when friends list is fully loaded
    this.client.on("friendsList", () => {
      console.log(`👥 [Bot ${this.id}] friendsList loaded`);
      this.friendsListReady = true;
    });

    // Real-time detection of friend relationship changes (accept/decline/remove)
    this.client.on("friendRelationship", async (steamID, relationship) => {
      const sid = steamID.getSteamID64();
      const relName = Object.keys(SteamUser.EFriendRelationship)
        .find(k => SteamUser.EFriendRelationship[k] === relationship) || relationship;
      console.log(`👥 [Bot ${this.id}] Friend relationship changed: ${sid} → ${relName}`);

      try {
        await ensureRedis();
        let status = "not_friend";
        let ttl = config.ttl.friendStatus;
        if (relationship === SteamUser.EFriendRelationship.Friend) {
          status = "friend";
        } else if (relationship === SteamUser.EFriendRelationship.RequestRecipient) {
          status = "pending";
          ttl = config.ttl.friendStatusPending;
        }
        await Promise.all([
          redisClient.set(`friend_status:${sid}`, status, { EX: ttl }),
          redisClient.set(`friend_status_ts:${sid}`, new Date().toISOString(), { EX: ttl }),
        ]);
        console.log(`✅ [Bot ${this.id}] Redis friend_status:${sid} → ${status}`);
      } catch (err) {
        console.error(`❌ [Bot ${this.id}] Failed to update friend status in Redis: ${err.message}`);
      }
    });
  }

  login() {
    this.client.logOn({
      accountName: this.username,
      password: this.password,
      twoFactorCode: SteamTotp.generateAuthCode(this.sharedSecret),
    });
  }

  get isAvailable() {
    return this.ready && this.csgo.haveGCSession;
  }
}

class BotPool {
  constructor() {
    /** @type {BotInstance[]} */
    this.bots = [];
    this._roundRobinIndex = 0;
  }

  /**
   * Inicializa el pool parseando BOT_ACCOUNTS o usando credenciales legacy.
   * No hace login automático — llamar loginAll() por separado.
   */
  initialize() {
    const accountsStr = config.botPool.accounts;

    if (accountsStr) {
      // Multi-bot: user1:pass1:secret1,user2:pass2:secret2,...
      const accounts = accountsStr.split(",").map(s => s.trim()).filter(Boolean);
      for (let i = 0; i < accounts.length; i++) {
        const parts = accounts[i].split(":");
        if (parts.length < 3) {
          console.error(`❌ [BotPool] Formato inválido en BOT_ACCOUNTS posición ${i}: esperado user:pass:secret`);
          continue;
        }
        const [username, password, ...secretParts] = parts;
        const sharedSecret = secretParts.join(":"); // por si el secret contiene ":"
        this.bots.push(new BotInstance(username, password, sharedSecret, i));
      }
      console.log(`🤖 [BotPool] ${this.bots.length} bots configurados desde BOT_ACCOUNTS`);
    } else {
      // Fallback: credenciales legacy (single bot)
      const { username, password, sharedSecret } = config.bot;
      if (username && password && sharedSecret) {
        this.bots.push(new BotInstance(username, password, sharedSecret, 0));
        console.log(`🤖 [BotPool] 1 bot configurado (credenciales legacy)`);
      } else {
        console.error("❌ [BotPool] No hay credenciales de bot configuradas");
      }
    }
  }

  /**
   * Login de todos los bots con delay escalonado para evitar rate limit de Steam.
   */
  async loginAll() {
    for (let i = 0; i < this.bots.length; i++) {
      this.bots[i].login();
      // Delay escalonado: 5s entre logins para evitar rate limit
      if (i < this.bots.length - 1) {
        await new Promise(resolve => setTimeout(resolve, 5000));
      }
    }
  }

  /**
   * Obtiene el bot con GC session activa y menor carga.
   * Fallback round-robin si todos tienen la misma carga.
   * @returns {BotInstance|null}
   */
  getAvailableBot() {
    const available = this.bots.filter(b => b.isAvailable);
    if (available.length === 0) return null;

    // Least-busy: bot con menos requests activos
    available.sort((a, b) => a.activeRequests - b.activeRequests);

    // Si todos tienen la misma carga, round-robin
    if (available[0].activeRequests === available[available.length - 1].activeRequests) {
      this._roundRobinIndex = (this._roundRobinIndex + 1) % available.length;
      return available[this._roundRobinIndex];
    }

    return available[0];
  }

  /**
   * Retorna el primer bot disponible (para compatibilidad con código legacy
   * que usa `csgo.requestGameAsync()`).
   * @returns {{ client: SteamUser, csgo: GlobalOffensive }}
   */
  getPrimaryBot() {
    return this.bots[0] || null;
  }

  /**
   * Estado de salud del pool.
   */
  getStatus() {
    return {
      totalBots: this.bots.length,
      readyBots: this.bots.filter(b => b.isAvailable).length,
      bots: this.bots.map(b => ({
        id: b.id,
        username: b.username,
        ready: b.ready,
        gcSession: b.csgo.haveGCSession,
        activeRequests: b.activeRequests,
      })),
    };
  }

  /**
   * Desconecta todos los bots.
   */
  logoffAll() {
    for (const bot of this.bots) {
      if (bot.client.steamID) {
        bot.client.logOff();
      }
    }
  }
}

// Singleton
const botPool = new BotPool();

module.exports = { botPool, BotInstance };
