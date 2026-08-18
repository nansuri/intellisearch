import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

// In dev, /api is proxied to the Go API on :8088. When the backend is down or
// briefly unavailable, Vite's proxy would otherwise answer with an HTML error
// page — respond with the standard JSON envelope instead so the UI can show a
// friendly message (see parseEnvelope in src/services/api.ts).
export default defineConfig({
  plugins: [
    vue(),
    // PWA: service worker + precached app shell so the site can be installed
    // ("Add to Home Screen" on Android/iOS). The web manifest itself is served
    // by the Go backend (/manifest.webmanifest) so it carries the live
    // site_settings branding — hence manifest: false here.
    VitePWA({
      registerType: 'autoUpdate',
      manifest: false,
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,ico}'],
        // SPA shell for navigations; API/uploads/manifest are never intercepted.
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/uploads\//, /^\/manifest\.webmanifest$/],
        runtimeCaching: [],
      },
      devOptions: { enabled: false },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8088',
        changeOrigin: true,
        configure(proxy) {
          proxy.on('error', (_err, _req, res) => {
            res.writeHead(503, { 'Content-Type': 'application/json' })
            res.end(JSON.stringify({ data: null, errorCode: 'HTTP_PROXY', errorMessage: 'Backend is not reachable — start the API (run-local.sh) and retry.' }))
          })
        },
      },
      '/manifest.webmanifest': {
        target: 'http://localhost:8088',
        changeOrigin: true,
      },
    },
  },
})
