import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  server: {
    // The compose gateway serves both APIs on one origin, so the dev server
    // forwards to it and the browser keeps seeing same-origin requests.
    proxy: { '/api': 'http://localhost:8080' },
  },
  build: {
    // The CSP allows nothing but 'self', so no asset may become a data: URI.
    assetsInlineLimit: 0,
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['src/test/setup.ts'],
    restoreMocks: true,
    unstubGlobals: true,
  },
});
