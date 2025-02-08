// steamGC.js
const EventEmitter = require('events');
// Suponiendo que la librería node-csgo ya carga los protos y la información necesaria:
const Protos = require('globaloffensive/protobufs/generated/_load.js');
const Language = require('globaloffensive/language.js');

class SteamGC extends EventEmitter {
  constructor(csgo) {
    super();
    this.csgo = csgo;
    this.registerHandlers();
  }

  registerHandlers() {
    // Ejemplo hipotético:
    // Suponemos que el GC emite un evento "matchInfo" en el que se incluye un arreglo "playerRanks".
    // Cada objeto en "playerRanks" tendría, por ejemplo, { steamid, premiereRank }.
    this.csgo.on('matchInfo', (data) => {
      if (data && Array.isArray(data.playerRanks)) {
        data.playerRanks.forEach(pr => {
          if (pr.steamid && pr.premiereRank) {
            // Emitimos un evento específico para este SteamID
            this.emit(`steamGC:rank:${pr.steamid}`, pr.premiereRank);
          }
        });
      }
    });
  }

  // getPlayerRank espera hasta timeout (5s por defecto) para recibir el dato vía GC.
  getPlayerRank(steamID, timeout = 5000) {
    return new Promise((resolve, reject) => {
      const eventKey = `steamGC:rank:${steamID}`;
      const onRank = (rank) => {
        clearTimeout(timer);
        resolve(rank);
      };
      this.once(eventKey, onRank);
      const timer = setTimeout(() => {
        this.removeListener(eventKey, onRank);
        resolve("N/A");
      }, timeout);
    });
  }
}

module.exports = SteamGC;
