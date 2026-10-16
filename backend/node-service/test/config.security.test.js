const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { createRequire } = require('node:module');

const configFile = path.resolve(__dirname, '../services/config.js');
const securityFile = path.resolve(__dirname, '../middleware/security.js');
const syntheticSecrets = {
  SESSION_SECRET_KEY: 'synthetic-session-secret-000000000000000000000000',
  INTERNAL_SERVICE_SECRET: 'synthetic-internal-secret-1111111111111111111111',
  CREDENTIAL_ENCRYPTION_KEY: 'synthetic-credential-secret-222222222222222222',
};

function evaluateModule(filename, env, dependencies = {}) {
  const module = { exports: {} };
  const localRequire = createRequire(filename);
  vm.runInNewContext(fs.readFileSync(filename, 'utf8'), {
    module,
    exports: module.exports,
    __filename: filename,
    __dirname: path.dirname(filename),
    process: { env },
    Buffer,
    require(name) {
      if (name === 'dotenv') return { config() {} };
      return Object.hasOwn(dependencies, name) ? dependencies[name] : localRequire(name);
    },
  }, { filename });
  return module.exports;
}

function configuration(env = {}) {
  return evaluateModule(configFile, { ...syntheticSecrets, ...env });
}

test('unsigned local internal requests are disabled by default in development', () => {
  assert.equal(configuration({ NODE_ENV: 'development' }).allowUnsignedLocalInternal, false);
  assert.equal(configuration({}).allowUnsignedLocalInternal, false);
});

test('development requires an explicit opt-in for unsigned local requests', () => {
  const config = configuration({ NODE_ENV: 'development', ALLOW_UNSIGNED_LOCAL_INTERNAL: 'true' });
  assert.equal(config.env, 'development');
  assert.equal(config.allowUnsignedLocalInternal, true);
  assert.equal(configuration({ NODE_ENV: 'development', ALLOW_UNSIGNED_LOCAL_INTERNAL: 'false' }).allowUnsignedLocalInternal, false);
});

for (const environment of [
  { NODE_ENV: 'production' },
  { APP_ENV: 'production' },
  { APP_ENV: 'production', NODE_ENV: 'production' },
]) {
  test(`production rejects unsigned mode: ${JSON.stringify(environment)}`, () => {
    assert.throws(() => configuration({ ...environment, ALLOW_UNSIGNED_LOCAL_INTERNAL: 'true' }));
    assert.equal(configuration({ ...environment, ALLOW_UNSIGNED_LOCAL_INTERNAL: 'false' }).env, 'production');
    assert.equal(configuration(environment).allowUnsignedLocalInternal, false);
  });
}

for (const environment of [
  { NODE_ENV: 'development', APP_ENV: 'production' },
  { NODE_ENV: 'production', APP_ENV: 'development' },
  { NODE_ENV: 'test', APP_ENV: 'development' },
]) {
  test(`contradictory environment names are rejected: ${JSON.stringify(environment)}`, () => {
    assert.throws(() => configuration({ ...environment, ALLOW_UNSIGNED_LOCAL_INTERNAL: 'false' }));
  });
}

test('internalOnly still rejects unsigned loopback traffic if a runtime config mutation enables the flag in production', async () => {
  const config = configuration({ NODE_ENV: 'development', ALLOW_UNSIGNED_LOCAL_INTERNAL: 'true' });
  const { internalOnly } = evaluateModule(securityFile, {}, {
    '../services/config': config,
    '../services/redisClient': {
      async ensureRedis() { throw new Error('Unsigned traffic must not reach Redis'); },
      redisClient: {},
    },
  });
  config.env = 'production';
  config.allowUnsignedLocalInternal = true;
  const response = {
    statusCode: 200,
    status(code) { this.statusCode = code; return this; },
    json(body) { this.body = body; return this; },
  };
  let accepted = false;

  await internalOnly({ ip: '127.0.0.1', headers: {} }, response, () => { accepted = true; });

  assert.equal(accepted, false);
  assert.equal(response.statusCode, 403);
});
