import { defineConfig } from 'vitest/config';

// The viewer is served by `pf serve`: pages at /view/..., built assets at
// /view/static/<file>. Assets are emitted flat into dist/ so that each one is
// a single path segment under /view/static/.
export default defineConfig({
  base: '/view/static/',
  build: {
    outDir: 'dist',
    assetsDir: '',
    emptyOutDir: true,
    sourcemap: false,
    // Phaser 3 alone is ~1.2 MB minified; it is served from the local binary.
    chunkSizeWarningLimit: 2000,
    // Keep everything in external files: the CSP forbids inline scripts.
    modulePreload: { polyfill: false },
    // Never inline sounds as data: URLs: the stage fetches them, and the CSP
    // only allows same-origin connections.
    assetsInlineLimit: (file: string) => (file.endsWith('.mp3') ? false : undefined),
  },
  server: {
    // The lobby embeds ../scripts/*.pf (sample bots) at build time.
    fs: { allow: ['..'] },
    // `npm run dev` against a running `pf serve` on the default address.
    proxy: {
      '/matches': 'http://127.0.0.1:8080',
      '/healthz': 'http://127.0.0.1:8080',
    },
  },
  test: {
    include: ['test/**/*.test.ts'],
    environment: 'node',
  },
});
