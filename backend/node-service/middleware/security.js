/**
 * security.js
 * -----------
 * Security middleware for the Node.js service.
 * - Security headers
 * - Steam ID validation
 * - Rate limiting helpers
 * - Request size limiting
 * - HMAC service-to-service authentication
 */

const crypto = require('crypto');
const config = require('../services/config');
const { ensureRedis, redisClient } = require('../services/redisClient');

const STEAM_ID_REGEX = /^\d{17}$/;

const HMAC_MAX_AGE_SECONDS = 30; // Reject requests older than 30 seconds
const SIGNATURE_VERSION = 'v1';

/**
 * Validates a Steam ID format (17-digit number).
 * @param {string} steamId
 * @returns {boolean}
 */
function isValidSteamId(steamId) {
  return typeof steamId === 'string' && STEAM_ID_REGEX.test(steamId);
}

/**
 * Express middleware to add security headers to all responses.
 */
function securityHeaders(req, res, next) {
  // Prevent MIME type sniffing
  res.setHeader('X-Content-Type-Options', 'nosniff');

  // Prevent clickjacking
  res.setHeader('X-Frame-Options', 'DENY');

  // XSS protection (legacy browsers)
  res.setHeader('X-XSS-Protection', '1; mode=block');

  // Referrer policy
  res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');

  // Don't disclose server technology
  res.removeHeader('X-Powered-By');

  // HSTS
  res.setHeader('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');

  // Restrict permissions
  res.setHeader('Permissions-Policy', 'camera=(), microphone=(), geolocation=(), payment=()');

  next();
}

/**
 * Express middleware to limit JSON body size.
 * Protects against DoS via oversized payloads.
 */
function requestSizeLimit(maxBytes = 1024 * 1024) {
  return (req, res, next) => {
    const contentLength = parseInt(req.headers['content-length'] || '0', 10);
    if (contentLength > maxBytes) {
      return res.status(413).json({ error: 'Request body too large' });
    }
    next();
  };
}

/**
 * Middleware to validate that requests come from trusted sources:
 * 1. Localhost (internal service calls) — always allowed
 * 2. HMAC-signed requests from FastAPI — verified via shared secret
 */
function isLocalRequest(req) {
  const ip = req.ip || req.connection?.remoteAddress || '';
  return (
    ip === '127.0.0.1' ||
    ip === '::1' ||
    ip === '::ffff:127.0.0.1' ||
    ip === 'localhost'
  );
}

function canonicalServiceRequest(req) {
  const version = req.headers['x-service-version'];
  const timestamp = req.headers['x-service-timestamp'];
  const nonce = req.headers['x-service-nonce'];
  const body = req.rawBody || Buffer.alloc(0);
  const bodyHash = crypto.createHash('sha256').update(body).digest('hex');
  return [
    version,
    req.method.toUpperCase(),
    req.originalUrl,
    timestamp,
    nonce,
    bodyHash,
  ].join('\n');
}

async function internalOnly(req, res, next) {
  if (config.allowUnsignedLocalInternal && isLocalRequest(req)) {
    return next();
  }

  const version = req.headers['x-service-version'];
  const timestamp = req.headers['x-service-timestamp'];
  const nonce = req.headers['x-service-nonce'];
  const signature = req.headers['x-service-signature'];
  if (
    version !== SIGNATURE_VERSION
    || typeof timestamp !== 'string'
    || typeof nonce !== 'string'
    || typeof signature !== 'string'
    || !/^[a-f0-9]{64}$/.test(signature)
    || !/^[a-f0-9]{32}$/.test(nonce)
  ) {
    return res.status(403).json({ error: 'Invalid service signature' });
  }

  const parsedTimestamp = Number.parseInt(timestamp, 10);
  const age = Math.abs(Math.floor(Date.now() / 1000) - parsedTimestamp);
  if (!Number.isFinite(parsedTimestamp) || age > HMAC_MAX_AGE_SECONDS) {
    return res.status(403).json({ error: 'Expired service signature' });
  }

  const expected = crypto
    .createHmac('sha256', config.internalServiceSecret)
    .update(canonicalServiceRequest(req))
    .digest('hex');
  const signatureBuffer = Buffer.from(signature, 'hex');
  const expectedBuffer = Buffer.from(expected, 'hex');
  if (!crypto.timingSafeEqual(signatureBuffer, expectedBuffer)) {
    return res.status(403).json({ error: 'Invalid service signature' });
  }

  try {
    await ensureRedis();
    const nonceKey = `${config.pipelineNamespace}:service-nonce:${nonce}`;
    const acquired = await redisClient.set(nonceKey, '1', { NX: true, EX: 60 });
    if (!acquired) {
      return res.status(409).json({ error: 'Replayed service request' });
    }
  } catch (error) {
    return res.status(503).json({ error: 'Service authentication unavailable' });
  }

  return next();
}

module.exports = {
  isValidSteamId,
  securityHeaders,
  requestSizeLimit,
  internalOnly,
  canonicalServiceRequest,
};
