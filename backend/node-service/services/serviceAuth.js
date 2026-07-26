const crypto = require('crypto');
const config = require('./config');

const SIGNATURE_VERSION = 'v1';

function canonicalServiceRequest(method, pathAndQuery, timestamp, nonce, body = Buffer.alloc(0)) {
  const rawBody = Buffer.isBuffer(body) ? body : Buffer.from(body);
  const bodyHash = crypto.createHash('sha256').update(rawBody).digest('hex');
  return [
    SIGNATURE_VERSION,
    method.toUpperCase(),
    pathAndQuery,
    timestamp,
    nonce,
    bodyHash,
  ].join('\n');
}

function buildServiceHeaders(method, pathAndQuery, body = Buffer.alloc(0)) {
  const rawBody = Buffer.isBuffer(body) ? body : Buffer.from(body);
  const timestamp = String(Math.floor(Date.now() / 1000));
  const nonce = crypto.randomBytes(16).toString('hex');
  const canonical = canonicalServiceRequest(method, pathAndQuery, timestamp, nonce, rawBody);
  const signature = crypto
    .createHmac('sha256', config.internalServiceSecret)
    .update(canonical)
    .digest('hex');
  return {
    'X-Service-Version': SIGNATURE_VERSION,
    'X-Service-Timestamp': timestamp,
    'X-Service-Nonce': nonce,
    'X-Service-Signature': signature,
  };
}

module.exports = { buildServiceHeaders, canonicalServiceRequest };
