/**
 * friendController.js
 * -------------------
 * Controlador Express para manejar la lógica de amistad:
 *  - Verificación de si el bot ya es amigo del usuario
 *  - Envío de solicitudes de amistad
 */

const express = require('express');
const redisClient = require('../services/redisClient');
const {client } = require('../services/steamDownloader');
const SteamUser = require('steam-user');

const router = express.Router();

// Endpoint para verificar si el bot es amigo del usuario
router.get('/steam/check-friend', async (req, res) => {
  const { steam_id } = req.query;

  if (!steam_id) {
    console.error("❌ Steam ID no proporcionado.");
    return res.status(400).json({ error: 'Falta el Steam ID' });
  }

  console.log(`🔍 Verificando amistad para Steam ID: ${steam_id}`);

  // Utilizamos el objeto client.myFriends para comprobar la relación de amistad
  const relationship = client.myFriends[steam_id];
  const isFriend = (relationship === SteamUser.EFriendRelationship.Friend);

  // Cacheamos en Redis el resultado de amistad
  await redisClient.set(`friend_status:${steam_id}`, isFriend ? 'true' : 'false', { EX: 86400 });

  console.log(`✅ Resultado amistad: ${isFriend ? 'Es amigo' : 'No es amigo'}`);

  return res.json({ is_friend: isFriend });
});


// Endpoint para enviar solicitud de amistad
router.post('/steam/send-friend-request', async (req, res) => {
  const { steam_id } = req.body;

  if (!steam_id) {
    return res.status(400).json({ error: 'Falta el Steam ID' });
  }

  client.addFriend(steam_id, async (err) => {
    if (err) {
      console.error(`❌ Error al enviar solicitud a ${steam_id}: ${err.message}`);
      return res.status(500).json({ error: 'Error al enviar la solicitud' });
    }
    console.log(`✅ Solicitud de amistad enviada a ${steam_id}`);

    // Marcamos en Redis el estado como "pending" (1 día)
    await redisClient.set(`friend_status:${steam_id}`, 'pending', { EX: 86400 });
    return res.json({ message: 'Solicitud de amistad enviada correctamente' });
  });
});

module.exports = router;
