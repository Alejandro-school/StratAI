export function createSecurityHeaders({ development = false, apiUrl = '' } = {}) {
  let apiOrigin = "'self'";
  if (apiUrl && (!apiUrl.startsWith('/') || apiUrl.startsWith('//'))) {
    const url = new URL(apiUrl);
    const local = ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname);
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password ||
        (!development && url.protocol === 'http:' && !local)) {
      throw new Error('VITE_API_URL must use HTTPS (HTTP is allowed only for local development)');
    }
    apiOrigin = url.origin;
  }
  const policy = [
    "default-src 'self'",
    `script-src 'self' 'wasm-unsafe-eval'${development ? " 'unsafe-inline' 'unsafe-eval'" : ''}`,
    "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
    "font-src 'self' https://fonts.gstatic.com",
    "img-src 'self' https://avatars.steamstatic.com https://steamcdn-a.akamaihd.net data: blob:",
    `connect-src 'self' ${apiOrigin} https://www.gstatic.com blob:${development ? ' ws://localhost:3000 ws://127.0.0.1:3000' : ''}`,
    "media-src 'self' blob:", "worker-src 'self' blob:",
    "object-src 'none'", "base-uri 'self'", "frame-ancestors 'none'",
  ].join('; ');
  return {
    'Content-Security-Policy': policy,
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
    'Permissions-Policy': 'camera=(), microphone=(), geolocation=(), payment=()',
  };
}
