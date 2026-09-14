import { defineConfig } from 'vitest/config';
export default defineConfig({
  esbuild: { jsx: 'automatic' },
  test: {
    environment: 'jsdom',
    include: ['src/**/__tests__/*.compat.test.jsx'],
    setupFiles: ['./scripts/setup-compat-tests.mjs'],
    pool: 'threads',
    poolOptions: { threads: { singleThread: true } },
    isolate: false,
    testTimeout: 15000,
  },
});
