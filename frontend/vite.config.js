import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath } from 'node:url';
import { createSecurityHeaders } from './securityPolicy';

const DEFERRED_3D_CHUNK_WARNING_LIMIT_KB = 800;

export default defineConfig(({ command, mode, isPreview }) => {
  const env = loadEnv(mode, fileURLToPath(new URL('.', import.meta.url)), 'VITE_');
  const development = command === 'serve' && !isPreview;
  const headers = createSecurityHeaders({ development, apiUrl: env.VITE_API_URL || (development ? 'http://localhost:8000' : '') });
  return {
  plugins: [react(), {
    name: 'stratai-security-policy',
    transformIndexHtml(html) {
      // frame-ancestors is HTTP-only; the meta policy still protects static builds.
      const metaPolicy = headers['Content-Security-Policy'].replace(/; frame-ancestors[^;]*/, '');
      return html.replace('__STRATAI_CSP__', metaPolicy);
    },
    generateBundle() {
      this.emitFile({ type: 'asset', fileName: 'security-headers.json', source: JSON.stringify(headers, null, 2) });
    },
  }],
  server: {
    headers,
    host: '0.0.0.0',
    port: 3000,
    strictPort: true,
  },
  preview: {
    headers,
    host: '0.0.0.0',
    port: 3000,
    strictPort: true,
  },
  build: {
    outDir: 'build',
    emptyOutDir: true,
    target: 'es2020',
    chunkSizeWarningLimit: DEFERRED_3D_CHUNK_WARNING_LIMIT_KB,
  },
  test: {
    // Bound jsdom workers to avoid memory/CPU contention on developer machines.
    maxWorkers: 4,
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
    css: true,
  },
};
});
