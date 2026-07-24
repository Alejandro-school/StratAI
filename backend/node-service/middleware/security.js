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

const STEAM_ID_REGEX = /^\d{17}$/;

// Shared secret for HMAC verification (must match Python SESSION_SECRET_KEY)
const SERVICE_SECRET = process.env.SESSION_SECRET_KEY || '';
const HMAC_MAX_AGE_SECONDS = 30; // Reject requests older than 30 seconds

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
function internalOnly(req, res, next) {
  const ip = req.ip || req.connection?.remoteAddress || '';
  const isLocal = (
    ip === '127.0.0.1' ||
    ip === '::1' ||
    ip === '::ffff:127.0.0.1' ||
    ip === 'localhost'
  );

  if (isLocal) return next();

  // Check HMAC signature for non-localhost (e.g., Docker, proxy scenarios)
  if (SERVICE_SECRET) {
    const timestamp = req.headers['x-service-timestamp'];
    const signature = req.headers['x-service-signature'];

    if (timestamp && signature) {
      // Reject stale requests (replay protection)
      const parsedTimestamp = Number.parseInt(timestamp, 10);
      const age = Math.abs(Math.floor(Date.now() / 1000) - parsedTimestamp);
      if (Number.isFinite(parsedTimestamp) && age <= HMAC_MAX_AGE_SECONDS) {
        const expected = crypto
          .createHmac('sha256', SERVICE_SECRET)
          .update(timestamp)
          .digest('hex');
        const signatureBuffer = Buffer.from(signature, 'hex');
        const expectedBuffer = Buffer.from(expected, 'hex');
        if (
          signatureBuffer.length === expectedBuffer.length
          && crypto.timingSafeEqual(signatureBuffer, expectedBuffer)
        ) {
          return next();
        }
      }
    }
  }

  return res.status(403).json({ error: 'Access denied: internal endpoint only' });
}

module.exports = {
  isValidSteamId,
  securityHeaders,
  requestSizeLimit,
  internalOnly,
};
