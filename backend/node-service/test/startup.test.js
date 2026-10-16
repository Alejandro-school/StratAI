const test = require('node:test');
const assert = require('node:assert/strict');
const { effectiveSettings } = require('../../../scripts/check-demo-services.cjs');

test('startup checks recognize the same Redis instance behind localhost aliases', () => {
  assert.deepEqual(effectiveSettings({ REDIS_URL: 'redis://localhost:6379' }), effectiveSettings({ REDIS_URL: 'redis://127.0.0.1:6379/0' }));
});

test('startup checks distinguish Redis database, credentials and transport', () => {
  const plain = effectiveSettings({ REDIS_URL: 'redis://localhost:6379/0' });
  for (const REDIS_URL of ['redis://localhost:6379/1', 'rediss://localhost:6379/0', 'redis://user:password@localhost:6379/0']) {
    assert.notEqual(effectiveSettings({ REDIS_URL }).redis, plain.redis);
  }
});

test('startup checks use the existing cross-service key derivation', () => {
  const derived = effectiveSettings({ SESSION_SECRET_KEY: 'local-session-test' });
  assert.notEqual(derived.serviceSecret, derived.encryptionKey);
  assert.deepEqual(effectiveSettings({ INTERNAL_SERVICE_SECRET: derived.serviceSecret, CREDENTIAL_ENCRYPTION_KEY: derived.encryptionKey }), derived);
});
