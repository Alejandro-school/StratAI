import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const DEFERRED_3D_CHUNK_WARNING_LIMIT_KB = 800;

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    strictPort: true,
  },
  preview: {
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
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
    css: true,
  },
});
