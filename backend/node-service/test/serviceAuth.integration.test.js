process.env.SESSION_SECRET_KEY = 'test-session-secret-32-characters-long';
process.env.INTERNAL_SERVICE_SECRET = 'test-internal-secret-32-characters-long';
process.env.CREDENTIAL_ENCRYPTION_KEY = 'test-credential-secret-32-characters-long';
process.env.ALLOW_UNSIGNED_LOCAL_INTERNAL = 'false';
process.env.PIPELINE_NAMESPACE = `stratai:test:auth:${process.pid}`;

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('crypto');

const { internalOnly } = require('../middleware/security');
const { buildServiceHeaders, canonicalServiceRequest } = require('../services/serviceAuth');
const { ensureRedis, redisClient } = require('../services/redisClient');

function responseDouble() {
  return {
    statusCode: 200,
    body: null,
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

function lowerCaseHeaders(headers) {
  return Object.fromEntries(
    Object.entries(headers).map(([key, value]) => [key.toLowerCase(), value]),
  );
}

async function authenticate(request) {
  const response = responseDouble();
  let accepted = false;
  await internalOnly(request, response, () => {
    accepted = true;
  });
  return { accepted, response };
}

test(
  'HMAC middleware rejects replay, expiry and body tampering',
  { skip: process.env.REDIS_INTEGRATION !== 'true', timeout: 15000 },
  async () => {
    await ensureRedis();
    try {
      const method = 'POST';
      const requestPath = '/internal/v2/discovery?source=test';
      const body = Buffer.from('{"steam_id":"76561198000000000"}');
      const headers = lowerCaseHeaders(buildServiceHeaders(method, requestPath, body));
      const request = {
        ip: '203.0.113.10',
        method,
        originalUrl: requestPath,
        rawBody: body,
        headers,
      };

      const first = await authenticate(request);
      assert.equal(first.accepted, true);
      const replay = await authenticate(request);
      assert.equal(replay.accepted, false);
      assert.equal(replay.response.statusCode, 409);

      const tamperedHeaders = lowerCaseHeaders(buildServiceHeaders(method, requestPath, body));
      const tampered = await authenticate({
        ...request,
        rawBody: Buffer.from('{"steam_id":"76561198000000001"}'),
        headers: tamperedHeaders,
      });
      assert.equal(tampered.response.statusCode, 403);

      const timestamp = String(Math.floor(Date.now() / 1000) - 120);
      const nonce = crypto.randomBytes(16).toString('hex');
      const canonical = canonicalServiceRequest(
        method,
        requestPath,
        timestamp,
        nonce,
        body,
      );
      const signature = crypto
        .createHmac('sha256', process.env.INTERNAL_SERVICE_SECRET)
        .update(canonical)
        .digest('hex');
      const expired = await authenticate({
        ...request,
        headers: {
          'x-service-version': 'v1',
          'x-service-timestamp': timestamp,
          'x-service-nonce': nonce,
          'x-service-signature': signature,
        },
      });
      assert.equal(expired.response.statusCode, 403);
    } finally {
      const nonceKeys = await redisClient.keys(`${process.env.PIPELINE_NAMESPACE}:service-nonce:*`);
      if (nonceKeys.length) await redisClient.del(nonceKeys);
      await redisClient.quit();
    }
  },
);
