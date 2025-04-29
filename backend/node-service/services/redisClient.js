/**
 * redisClient.js
 * ---------------
 * Módulo que inicializa y exporta el cliente de Redis para
 * ser utilizado en toda la aplicación Node.js.
 */

const redis = require('redis');

// Creamos el cliente principal de Redis
const redisClient = redis.createClient({
  // Si necesitas configurar host/puerto/contraseña, agrega opciones aquí
  host: 'localhost', 
  port: 6379,
});

redisClient.on('error', (err) => {
  console.error('❌ Error en Redis Client:', err);
});

// Iniciamos la conexión al arrancar
(async () => {
  try {
    await redisClient.connect();
    console.log('✅ Conectado a Redis (cliente principal)');
  } catch (err) {
    console.error('❌ Error al conectar con Redis:', err);
    process.exit(1);
  }
})();

module.exports = redisClient;
