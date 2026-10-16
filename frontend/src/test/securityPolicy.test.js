import { describe, expect, it } from 'vitest';
import { createSecurityHeaders } from '../../securityPolicy';

function header(headers, name) {
  const entry = Object.entries(headers).find(([key]) => key.toLowerCase() === name.toLowerCase());
  return entry?.[1];
}

function directives(headers) {
  return Object.fromEntries(header(headers, 'Content-Security-Policy')
    .split(';')
    .map((directive) => directive.trim().split(/\s+/))
    .filter(([name]) => name)
    .map(([name, ...sources]) => [name, sources]));
}

describe('frontend HTTP security policy', () => {
  it('limits production scripts while retaining the required Draco WebAssembly capability', () => {
    const policy = directives(createSecurityHeaders());

    expect(policy['script-src']).toContain("'self'");
    expect(policy['script-src']).toContain("'wasm-unsafe-eval'");
    expect(policy['script-src']).not.toContain("'unsafe-inline'");
    expect(policy['script-src']).not.toContain("'unsafe-eval'");
    expect(policy['script-src']).not.toContain('*');
    expect(policy['object-src']).toEqual(["'none'"]);
    expect(policy['base-uri']).toEqual(["'self'"]);
    expect(policy['frame-ancestors']).toEqual(["'none'"]);
  });

  it('returns real HTTP headers for MIME handling, framing and referrer restrictions', () => {
    const headers = createSecurityHeaders();

    expect(header(headers, 'X-Content-Type-Options')).toBe('nosniff');
    expect(header(headers, 'X-Frame-Options')).toBe('DENY');
    expect(['no-referrer', 'same-origin', 'strict-origin', 'strict-origin-when-cross-origin'])
      .toContain(header(headers, 'Referrer-Policy'));
  });

  it('allows the configured API origin without granting other origins or wildcard WebSockets', () => {
    const policy = directives(createSecurityHeaders({ apiUrl: 'https://api.example.test:9443/v1' }));
    const sources = policy['connect-src'];

    expect(sources).toContain("'self'");
    expect(sources).toContain('https://api.example.test:9443');
    expect(sources).toContain('https://www.gstatic.com');
    expect(sources).toContain('blob:');
    expect(sources.every((source) => !source.includes('*'))).toBe(true);
    expect(sources.some((source) => /^wss?:/.test(source))).toBe(false);
    expect(sources.some((source) => source.includes('localhost') || source.includes('127.0.0.1'))).toBe(false);
  });

  it('supports an API behind a same-origin relative path', () => {
    const sources = directives(createSecurityHeaders({ apiUrl: '/api' }))['connect-src'];

    expect(sources).toContain("'self'");
    expect(sources).not.toContain('/api');
    expect(sources.some((source) => source.includes('localhost') || source.includes('127.0.0.1'))).toBe(false);
  });

  it.each(['http://localhost:8000', 'http://127.0.0.1:8000'])
    ('permits an explicitly configured local HTTP API: %s', (apiUrl) => {
      const sources = directives(createSecurityHeaders({ apiUrl }))['connect-src'];

      expect(sources).toContain(apiUrl);
      expect(sources.every((source) => !source.includes('*'))).toBe(true);
    });

  it.each([
    'http://192.168.1.20:8000',
    'http://10.0.0.2:8000',
    'http://api.example.test',
    'http://0.0.0.0:8000',
    'http://localhost.attacker.example:8000',
    'http://127.0.0.1.attacker.example:8000',
    'https://user:password@api.example.test',
    'https://user@api.example.test',
    '//untrusted.example/api',
    'javascript:alert(1)',
    'data:text/plain,synthetic',
    'ftp://files.example.test/api',
  ])('rejects an unsafe production API URL: %s', (apiUrl) => {
    expect(() => createSecurityHeaders({ apiUrl })).toThrow();
  });

  it('adds only explicit local HMR socket origins in development', () => {
    const production = directives(createSecurityHeaders());
    const development = directives(createSecurityHeaders({ development: true }));

    expect(development['connect-src']).toContain('ws://localhost:3000');
    expect(development['connect-src']).toContain('ws://127.0.0.1:3000');
    expect(development['connect-src']).not.toContain('ws:');
    expect(development['connect-src']).not.toContain('wss:');
    expect(production['connect-src'].some((source) => /^wss?:/.test(source))).toBe(false);
  });
});
