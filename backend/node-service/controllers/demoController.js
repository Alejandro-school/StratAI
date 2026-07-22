/**
 * demoController.js
 * -----------------
 * Controlador Express que expone endpoints para gestionar demos:
 * - /start-download: iniciar descarga de demos
 * - /fetch-match-info: obtener metadata de una partida por sharecode
 * - /demos-without-info: listar demos sin match_info.json
 */

const express = require('express');
const axios = require('axios');
const { queue, procesarShareCode, enqueueShareCode, fetchMatchInfoBySharecode, findDemosWithoutMatchInfo, progressEmitter, gcQueue, downloadQueue, goQueue } = require('../services/steamDownloader');
const { fetchSharecodesForUser } = require('../services/cronJob');
const { redisClient } = require('../services/redisClient');
const { internalOnly } = require('../middleware/security');
const config = require('../services/config');

const router = express.Router();

// Validación de formato sharecode
const SHARECODE_REGEX = /^CSGO-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}$/;

function isValidSharecode(code) {
  return typeof code === 'string' && SHARECODE_REGEX.test(code);
}

/**
 * POST /fetch-new-matches
 * On-demand match detection for a single user.
 * Called by FastAPI proxy (authenticated) or internally.
 * Rate-limited: max 1 call per user per 60 seconds.
 */
router.post('/fetch-new-matches', internalOnly, async (req, res) => {
  try {
    const { steam_id } = req.body;

    if (!steam_id || !/^\d{17}$/.test(steam_id)) {
      return res.status(400).json({ error: 'Invalid or missing steam_id' });
    }

    // Rate limit: 1 call per user per 60s
    const cooldownKey = `fetch_cooldown:${steam_id}`;
    const cooldownTTL = await redisClient.ttl(cooldownKey);
    if (cooldownTTL > 0) {
      return res.status(429).json({
        triggered: false,
        reason: 'cooldown',
        retry_after: cooldownTTL
      });
    }

    // Set cooldown
    await redisClient.set(cooldownKey, '1', { EX: 60 });

    // Fetch new sharecodes directly from Steam API (no Python proxy needed)
    let sharecodes = [];
    try {
      sharecodes = await fetchSharecodesForUser(steam_id);
    } catch (err) {
      console.error(`❌ [FETCH] Error fetching sharecodes for ${steam_id}:`, err.message);
      // Clear cooldown on error so user can retry
      await redisClient.del(cooldownKey);
      return res.status(502).json({
        triggered: false,
        reason: 'steam_api_error',
        message: err.message
      });
    }

    if (!sharecodes.length) {
      return res.json({
        triggered: false,
        reason: 'no_new_matches',
        message: 'No new matches found'
      });
    }

    // Enqueue new sharecodes for processing (priority 10 = higher than default)
    const statusMap = await redisClient.hGetAll(`sharecode_status:${steam_id}`);
    let enqueued = 0;
    for (const code of sharecodes) {
      const status = statusMap[code];
      if (!status || status === 'pending') {
        enqueueShareCode(code, steam_id, { priority: 10 });
        enqueued++;
      }
    }

    console.log(`🔍 [FETCH] ${steam_id}: ${sharecodes.length} new codes, ${enqueued} enqueued`);

    return res.json({
      triggered: true,
      new_codes_count: sharecodes.length,
      enqueued_count: enqueued
    });
  } catch (error) {
    console.error('❌ Error in /fetch-new-matches:', error);
    return res.status(500).json({ error: error.message });
  }
});

/**
 * POST /start-download
 * Permite forzar manualmente la descarga de demos indicando
 * un steam_id y sharecodes en el cuerpo de la petición.
 */
router.post('/start-download', internalOnly, async (req, res) => {
  try {
    const { steam_id, sharecodes } = req.body;
    
    if (!steam_id) {
      return res.status(400).json({ error: 'Falta steam_id' });
    }

    // Si vienen sharecodes manuales, los procesamos
    if (Array.isArray(sharecodes) && sharecodes.length > 0) {
      // Validar formato de cada sharecode
      const invalidCodes = sharecodes.filter(code => !isValidSharecode(code));
      if (invalidCodes.length > 0) {
        return res.status(400).json({ 
          error: 'Sharecodes con formato inválido',
          invalid_codes: invalidCodes
        });
      }
      
      // priority=10 para onboarding manual (mayor que cron que usa 0)
      const priority = req.body.priority === true ? 10 : 0;
      for (const code of sharecodes) {
        enqueueShareCode(code, steam_id, { priority });
      }
    }

    return res.json({ 
      message: 'Descarga iniciada en segundo plano.',
      queued_count: Array.isArray(sharecodes) ? sharecodes.length : 0
    });
  } catch (error) {
    console.error('❌ Error en /start-download:', error);
    return res.status(500).json({ error: error.message || error.toString() });
  }
});

/**
 * POST /fetch-match-info
 * Obtiene la metadata de una partida (fecha, duración, etc.) por su sharecode
 * sin descargar la demo. Útil para demos ya existentes.
 * 
 * Body: { sharecode: "CSGO-xxxx-xxxx-xxxx-xxxx-xxxx" }
 */
router.post('/fetch-match-info', internalOnly, async (req, res) => {
  try {
    const { sharecode } = req.body;
    
    if (!sharecode) {
      return res.status(400).json({ error: 'Falta sharecode' });
    }
    
    if (!isValidSharecode(sharecode)) {
      return res.status(400).json({ error: 'Formato de sharecode inválido' });
    }

    const matchInfo = await fetchMatchInfoBySharecode(sharecode);
    return res.json({ 
      success: true, 
      match_info: matchInfo 
    });
  } catch (error) {
    console.error('❌ Error en /fetch-match-info:', error);
    return res.status(500).json({ error: error.message });
  }
});

/**
 * GET /demos-without-info
 * Lista las demos que no tienen su archivo match_info.json correspondiente.
 * Útil para identificar qué demos necesitan sus sharecodes para obtener la fecha.
 */
router.get('/demos-without-info', internalOnly, async (req, res) => {
  try {
    const missing = await findDemosWithoutMatchInfo();
    return res.json({ 
      count: missing.length,
      demos: missing 
    });
  } catch (error) {
    console.error('❌ Error en /demos-without-info:', error);
    return res.status(500).json({ error: error.message || error.toString() });
  }
});

/**
 * POST /fetch-match-info-batch
 * Procesa múltiples sharecodes para obtener la metadata.
 * Útil para llenar match_info.json de demos existentes.
 * 
 * Body: { sharecodes: ["CSGO-xxx...", "CSGO-yyy..."] }
 */
router.post('/fetch-match-info-batch', internalOnly, async (req, res) => {
  try {
    const { sharecodes } = req.body;
    
    if (!Array.isArray(sharecodes) || sharecodes.length === 0) {
      return res.status(400).json({ error: 'Falta array de sharecodes' });
    }
    
    // Validar formato de cada sharecode
    const invalidCodes = sharecodes.filter(code => !isValidSharecode(code));
    if (invalidCodes.length > 0) {
      return res.status(400).json({ 
        error: 'Sharecodes con formato inválido',
        invalid_codes: invalidCodes
      });
    }

    // Encolar procesamiento de cada sharecode
    for (const code of sharecodes) {
      queue.add(async () => {
        try {
          await fetchMatchInfoBySharecode(code);
          console.log(`✅ Match info obtenido para: ${code}`);
        } catch (err) {
          console.error(`❌ Error obteniendo match info para ${code}: ${err.message}`);
        }
      });
    }

    return res.json({ 
      message: `${sharecodes.length} sharecodes encolados para procesamiento.`,
      note: 'Los resultados se guardarán en match_info.json junto a cada demo.'
    });
  } catch (error) {
    console.error('❌ Error en /fetch-match-info-batch:', error);
    return res.status(500).json({ error: error.message || error.toString() });
  }
});

/**
 * GET /download-progress/:steamId
 * SSE (Server-Sent Events) endpoint para recibir progreso en tiempo real
 * del pipeline de descarga de un usuario.
 * 
 * Events: { sharecode, stage, matchID, error? }
 * Stages: gc_resolving → downloading → processing → completed | error
 */
router.get('/download-progress/:steamId', internalOnly, (req, res) => {
  const { steamId } = req.params;
  
  if (!steamId || !/^\d{17}$/.test(steamId)) {
    return res.status(400).json({ error: 'SteamID inválido' });
  }

  // SSE headers
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    Connection: 'keep-alive',
    'X-Accel-Buffering': 'no',
  });
  res.flushHeaders();

  // Heartbeat cada 30s para mantener la conexión
  const heartbeat = setInterval(() => {
    res.write(': heartbeat\n\n');
  }, 30000);

  const onProgress = (data) => {
    res.write(`data: ${JSON.stringify(data)}\n\n`);
  };

  progressEmitter.on(`progress:${steamId}`, onProgress);

  // Cleanup cuando el cliente cierra la conexión
  req.on('close', () => {
    clearInterval(heartbeat);
    progressEmitter.removeListener(`progress:${steamId}`, onProgress);
  });
});

/**
 * GET /pipeline-status
 * Devuelve el estado actual de las 3 colas del pipeline.
 */
router.get('/pipeline-status', internalOnly, (req, res) => {
  res.json({
    gcQueue: { active: gcQueue.pending, waiting: gcQueue.size },
    downloadQueue: { active: downloadQueue.pending, waiting: downloadQueue.size },
    goQueue: { active: goQueue.pending, waiting: goQueue.size },
  });
});

module.exports = router;
