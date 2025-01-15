const { downloadDemoFromSharecode } = require('../services/steamDownloader');

async function downloadDemos(req, res) {
  const { steamId, sharecodes } = req.body;

  if (!steamId || !sharecodes || !Array.isArray(sharecodes)) {
    return res.status(400).json({ message: 'Faltan datos o el formato es incorrecto.' });
  }

  try {
    // Descarga cada demo en secuencia
    for (const sharecode of sharecodes) {
      await downloadDemoFromSharecode(steamId, sharecode);
    }

    return res.status(200).json({ message: '✅ Descarga de demos finalizada.' });
  } catch (error) {
    console.error(`❌ Error al descargar demos: ${error.message}`);
    return res.status(500).json({ message: '❌ Error al descargar las demos.' });
  }
}

module.exports = { downloadDemos };
