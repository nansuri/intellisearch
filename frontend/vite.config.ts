import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// In dev, /api is proxied to the Go API on :8088. When the backend is down or
// briefly unavailable, Vite's proxy would otherwise answer with an HTML error
// page — respond with the standard JSON envelope instead so the UI can show a
// friendly message (see parseEnvelope in src/services/api.ts).
export default defineConfig({
  plugins: [vue()],
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
    },
  },
})