// backend/node-service/redisClient.js

const Redis = require('ioredis');

// Conectar a Redis (puerto y host por defecto)
const redis = new Redis({
    host: 'localhost',
    port: 6379,
});

redis.on('connect', () => {
    console.log('Conectado a Redis');
});

redis.on('error', (err) => {
    console.error('Error de conexión a Redis:', err);
});

module.exports = redis;
