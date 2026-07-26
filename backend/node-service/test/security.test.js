process.env.SESSION_SECRET_KEY = 'test-secret';
process.env.ALLOW_UNSIGNED_LOCAL_INTERNAL = 'true';

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const {
  isValidSteamId,
  securityHeaders,
  requestSizeLimit,
  internalOnly,
} = require('../middleware/security');
const { canonicalServiceRequest } = require('../services/serviceAuth');
const config = require('../services/config');

const serviceVector = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '../../testdata/service_auth_vectors.json'), 'utf8'),
);

test('derives independent development secrets compatible with FastAPI', () => {
  assert.equal(
    config.internalServiceSecret,
    crypto.createHash('sha256').update('test-secret:internal-service').digest('hex'),
  );
  assert.equal(
    config.credentialEncryptionKey,
    crypto.createHash('sha256').update('test-secret:credential-encryption').digest('hex'),
  );
  assert.notEqual(config.internalServiceSecret, config.credentialEncryptionKey);
});

function createResponse() {
  return {
    headers: new Map(),
    statusCode: null,
    body: null,
    setHeader(name, value) {
      this.headers.set(name, value);
    },
    removeHeader(name) {
      this.headers.delete(name);
    },
    status(code) {
      this.statusCode = code;
      return this;
    },
    json(body) {
      this.body = body;
      return this;
    },
  };
}

test('validates Steam IDs strictly', () => {
  assert.equal(isValidSteamId('76561198000000000'), true);
  assert.equal(isValidSteamId('7656119800000000'), false);
  assert.equal(isValidSteamId('7656119800000000x'), false);
  assert.equal(isValidSteamId(76561198000000000), false);
});

test('adds security headers and calls next', () => {
  const response = createResponse();
  response.headers.set('X-Powered-By', 'Express');
  let called = false;

  securityHeaders({}, response, () => {
    called = true;
  });

  assert.equal(called, true);
  assert.equal(response.headers.get('X-Frame-Options'), 'DENY');
  assert.equal(response.headers.has('X-Powered-By'), false);
});

test('rejects oversized requests', () => {
  const response = createResponse();
  let called = false;

  requestSizeLimit(100)(
    { headers: { 'content-length': '101' } },
    response,
    () => {
      called = true;
    },
  );

  assert.equal(called, false);
  assert.equal(response.statusCode, 413);
});

test('allows explicitly configured unsigned localhost requests', async () => {
  const response = createResponse();
  let called = false;

  await internalOnly({ ip: '127.0.0.1', headers: {} }, response, () => {
    called = true;
  });

  assert.equal(called, true);
});

test('rejects malformed HMAC signatures without throwing', async () => {
  const response = createResponse();
  const timestamp = String(Math.floor(Date.now() / 1000));

  await assert.doesNotReject(
    internalOnly(
      {
        ip: '203.0.113.10',
        headers: {
          'x-service-timestamp': timestamp,
          'x-service-signature': 'aa',
        },
        method: 'POST',
        originalUrl: '/internal/v2/discovery',
      },
      response,
      () => assert.fail('A malformed signature must not be accepted'),
    ),
  );

  assert.equal(response.statusCode, 403);
});

test('matches the cross-language canonical HMAC vector', () => {
  const canonical = canonicalServiceRequest(
    serviceVector.method,
    serviceVector.path_and_query,
    serviceVector.timestamp,
    serviceVector.nonce,
    Buffer.from(serviceVector.body),
  );
  const signature = crypto
    .createHmac('sha256', serviceVector.secret)
    .update(canonical)
    .digest('hex');
  assert.equal(signature, serviceVector.signature);

  const tampered = canonicalServiceRequest(
    serviceVector.method,
    '/process-demo?x=2',
    serviceVector.timestamp,
    serviceVector.nonce,
    Buffer.from(serviceVector.body),
  );
  assert.notEqual(
    crypto.createHmac('sha256', serviceVector.secret).update(tampered).digest('hex'),
    serviceVector.signature,
  );
});
