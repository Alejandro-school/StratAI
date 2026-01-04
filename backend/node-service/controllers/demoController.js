/**
 * demoController.js
 * -----------------
 * Controlador Express que expone endpoints para gestionar demos:
 * - /start-download: iniciar descarga de demos
 * - /fetch-match-info: obtener metadata de una partida por sharecode
 * - /demos-without-info: listar demos sin match_info.json
 */

const express = require('express');
const { queue, procesarShareCode, fetchMatchInfoBySharecode, findDemosWithoutMatchInfo } = require('../services/steamDownloader');

const router = express.Router();

// Validación de formato sharecode
const SHARECODE_REGEX = /^CSGO-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}-[A-Za-z0-9]{5}$/;

function isValidSharecode(code) {
  return typeof code === 'string' && SHARECODE_REGEX.test(code);
}

/**
 * POST /start-download
 * Permite forzar manualmente la descarga de demos indicando
 * un steam_id y sharecodes en el cuerpo de la petición.
 */
router.post('/start-download', async (req, res) => {
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
      
      for (const code of sharecodes) {
        queue.add(() => procesarShareCode(code, steam_id));
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
router.post('/fetch-match-info', async (req, res) => {
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
router.get('/demos-without-info', async (req, res) => {
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
router.post('/fetch-match-info-batch', async (req, res) => {
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

module.exports = router;
