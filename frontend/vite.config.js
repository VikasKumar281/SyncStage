import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In development the app is served from :5173 and the API lives on :8080.
// Proxying /api keeps the browser on a single origin, which means no CORS
// preflights and — more importantly — an EventSource connection that behaves
// exactly like it will in production.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
        changeOrigin: true,
        // Server-Sent Events must not be buffered by the dev proxy.
        configure: (proxy) => {
          proxy.on('proxyRes', (proxyRes) => {
            if (proxyRes.headers['content-type']?.includes('text/event-stream')) {
              proxyRes.headers['cache-control'] = 'no-cache, no-transform';
            }
          });
        },
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
