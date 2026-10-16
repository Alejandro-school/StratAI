const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { createRequire } = require('node:module');
const dotenv = require('dotenv');

const root = path.resolve(__dirname, '..');
const serviceRequire = createRequire(path.join(root, 'backend/node-service/package.json'));

function readEnv(file) {
  return fs.existsSync(file) ? dotenv.parse(fs.readFileSync(file)) : {};
}

function effectiveSettings(env) {
  const derive = purpose => env.SESSION_SECRET_KEY
    ? crypto.createHash('sha256').update(`${env.SESSION_SECRET_KEY}:${purpose}`).digest('hex') : '';
  const redis = new URL(env.REDIS_URL || 'redis://localhost:6379');
  if (redis.hostname === 'localhost') redis.hostname = '127.0.0.1';
  if (!redis.port) redis.port = '6379';
  if (!redis.pathname || redis.pathname === '/') redis.pathname = '/0';
  return {
    redis: redis.href,
    namespace: env.PIPELINE_NAMESPACE || 'stratai:v2',
    serviceSecret: env.INTERNAL_SERVICE_SECRET || derive('internal-service'),
    encryptionKey: env.CREDENTIAL_ENCRYPTION_KEY || derive('credential-encryption'),
  };
}

async function check() {
  const backendEnv = { ...readEnv(path.join(root, 'backend/.env')), ...process.env };
  const nodeEnv = {
    ...readEnv(path.join(root, 'backend/.env')),
    ...readEnv(path.join(root, 'backend/node-service/.env')),
    ...process.env,
  };
  const backend = effectiveSettings(backendEnv);
  const node = effectiveSettings(nodeEnv);
  const go = effectiveSettings({
    ...backendEnv,
    REDIS_URL: backendEnv.REDIS_URL || (backendEnv.REDIS_ADDR ? `redis://${backendEnv.REDIS_ADDR}` : undefined),
  });
  if (go.redis !== backend.redis) throw new Error('Go usa un Redis diferente. Configura el mismo REDIS_URL en los servicios.');
  for (const key of Object.keys(backend)) {
    if (backend[key] !== node[key]) throw new Error(`Configuración diferente entre FastAPI y Node: ${key}. Revisa ambos .env.`);
  }
  for (const key of ['SESSION_SECRET_KEY', 'STEAM_API_KEY']) {
    if (!backendEnv[key] || /^(your_|generate_|replace-)/i.test(backendEnv[key])) {
      throw new Error(`Falta configurar ${key} en backend/.env.`);
    }
  }
  if (!nodeEnv.BOT_ACCOUNTS && !['BOT_USERNAME', 'BOT_PASSWORD', 'BOT_SHARED_SECRET'].every(key => nodeEnv[key] && !/^your_/i.test(nodeEnv[key]))) {
    throw new Error('Faltan las credenciales de la cuenta técnica en backend/node-service/.env.');
  }
  if (nodeEnv.PIPELINE_V2_ENABLED === 'false') throw new Error('El pipeline de demos debe estar habilitado.');
  const { createClient } = serviceRequire('redis');
  const redis = createClient({ url: backend.redis, socket: { connectTimeout: 5000, reconnectStrategy: false } });
  redis.on('error', () => {});
  try {
    await redis.connect();
    if (await redis.ping() !== 'PONG') throw new Error('Redis no está preparado.');
  } catch {
    throw new Error('Redis no responde. Arráncalo o configura el mismo REDIS_URL en los servicios.');
  } finally {
    if (redis.isOpen) redis.destroy();
  }
  console.log('Configuración compartida y Redis verificados. Arranque completo habilitado.');
}

if (require.main === module) {
  check().catch(error => {
    console.error(error instanceof TypeError ? 'Configuración de servicios inválida. Revisa los archivos .env.' : error.message);
    process.exitCode = 1;
  });
}

module.exports = { effectiveSettings };
