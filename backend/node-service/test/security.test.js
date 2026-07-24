process.env.SESSION_SECRET_KEY = 'test-secret';

const test = require('node:test');
const assert = require('node:assert/strict');
const {
  isValidSteamId,
  securityHeaders,
  requestSizeLimit,
  internalOnly,
} = require('../middleware/security');

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

test('allows localhost requests', () => {
  const response = createResponse();
  let called = false;

  internalOnly({ ip: '127.0.0.1', headers: {} }, response, () => {
    called = true;
  });

  assert.equal(called, true);
});

test('rejects malformed HMAC signatures without throwing', () => {
  const response = createResponse();
  const timestamp = String(Math.floor(Date.now() / 1000));

  assert.doesNotThrow(() => {
    internalOnly(
      {
        ip: '203.0.113.10',
        headers: {
          'x-service-timestamp': timestamp,
          'x-service-signature': 'aa',
        },
      },
      response,
      () => assert.fail('A malformed signature must not be accepted'),
    );
  });

  assert.equal(response.statusCode, 403);
});
