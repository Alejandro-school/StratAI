/**
 * demoController.js
 * -----------------
 * Controlador Express que expone el endpoint /start-download
 * para disparar manualmente la descarga de demos, sin esperar
 * a que Redis notifique un rpush.
 */

const express = require('express');
const { queue, procesarShareCode } = require('../services/steamDownloader');

const router = express.Router();

/**
 * POST /start-download
 * Permite forzar manualmente la descarga de demos indicando
 * un steam_id y sharecodes en el cuerpo de la petición.
 */
router.post('/start-download', async (req, res) => {
  try {
    const { steam_id, sharecodes, autoPerfil } = req.body; // p.ej. autoPerfil=true
    
    if (!steam_id) {
      return res.status(400).json({ error: 'Falta steam_id' });
    }

    // Si vienen sharecodes manuales, los procesamos
    if (Array.isArray(sharecodes) && sharecodes.length > 0) {
      for (const code of sharecodes) {
        queue.add(() => procesarShareCode(code, steam_id));
      }
    }

    // Si el usuario pide "autoPerfil" para leer matches
    if (autoPerfil) {
      leerShareCodesEnPerfil(steam_id);
    }

    return res.json({ message: 'Descarga iniciada en segundo plano.' });
  } catch (error) {
    console.error('❌ Error en /start-download:', error);
    return res.status(500).json({ error: error.toString() });
  }
});

module.exports = router;
