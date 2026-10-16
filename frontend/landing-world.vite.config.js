import { mergeConfig } from 'vite';
import { fileURLToPath } from 'node:url';
import baseConfig from './vite.config.js';

export default environment => mergeConfig(baseConfig(environment), {
  build: {
    outDir: '../output/landing-world-build-20261015',
    emptyOutDir: false,
    rollupOptions: { input: fileURLToPath(new URL('./landing-world.html', import.meta.url)) },
  },
});
