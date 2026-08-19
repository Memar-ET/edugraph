import path from 'node:path'

import react from '@vitejs/plugin-react'
// vitest/config re-exports defineConfig with the `test` key merged into
// Vite's UserConfig type -- importing from plain 'vite' here makes `test`
// below a type error under `tsc -b`.
import { defineConfig } from 'vitest/config'

// API_PROXY_TARGET lets docker-compose point the dev-server proxy at the
// `api` service by its container DNS name (http://api:8080) while local
// `npm run dev` outside Docker keeps working against http://localhost:8080.
const apiProxyTarget = process.env.API_PROXY_TARGET ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@features': path.resolve(__dirname, './src/features'),
      '@components': path.resolve(__dirname, './src/components'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@lib': path.resolve(__dirname, './src/lib'),
      '@stores': path.resolve(__dirname, './src/stores'),
    },
  },
  server: {
    port: 5173,
    host: true,
    // Enable polling when running inside Docker on Windows: NTFS bind-mounts
    // don't propagate inotify events to the Linux container, so HMR only fires
    // if Vite polls the filesystem. CHOKIDAR_USEPOLLING is set by docker-compose.
    watch: process.env.CHOKIDAR_USEPOLLING === 'true'
      ? { usePolling: true, interval: 500 }
      : {},
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
        // Vite's dev proxy (http-proxy under the hood) has no timeout
        // budget of its own by default, but the underlying Node HTTP
        // machinery still cuts a proxied request off well under two
        // minutes -- long enough for ordinary requests, too short for
        // the AI tutor's local-LLM generation (which legitimately takes
        // 60-150s+ under load: pkg/ai/ai.go's own Go->ai-service client
        // budgets 210s for exactly this reason). Without an explicit
        // timeout here, Vite returns its own false 500 to the browser
        // while the real backend request keeps running and succeeds
        // moments later -- same class of bug already documented in
        // CLAUDE.md for ApproveAndPromote over Supabase. `timeout` caps
        // inactivity on the client<->proxy socket; `proxyTimeout` caps
        // the proxy<->target leg. Both set above Go's 210s client budget.
        timeout: 220_000,
        proxyTimeout: 220_000,
      },
      '/ws': {
        target: apiProxyTarget.replace(/^http/, 'ws'),
        ws: true,
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./tests/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      exclude: ['node_modules/', 'src/main.tsx'],
    },
  },
})
