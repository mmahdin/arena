import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Local synthetic-data only: same-origin API proxy avoids browser cross-origin cookies.
// Vite and the Go API should both be accessed as 127.0.0.1 (do not mix localhost).
//
// Host binding stays loopback by default. A hosted preview sandbox reaches the dev
// server from a different machine, so it can be opted into per-environment:
//   HAMRAH_HOST=0.0.0.0 HAMRAH_ALLOWED_HOSTS=.e2b.app npm run dev
// The API proxy always targets loopback, so the backend is never exposed either way.
const host = process.env.HAMRAH_HOST || '127.0.0.1';
const allowedHosts = process.env.HAMRAH_ALLOWED_HOSTS
  ? process.env.HAMRAH_ALLOWED_HOSTS.split(',').map((h) => h.trim()).filter(Boolean)
  : undefined;

export default defineConfig({
  plugins: [react()],
  server: {
    host,
    port: 5173,
    strictPort: true,
    ...(allowedHosts ? { allowedHosts } : {}),
    proxy: { '/api/v1': { target: 'http://127.0.0.1:8080', changeOrigin: false } },
  },
  preview: {
    host,
    ...(allowedHosts ? { allowedHosts } : {}),
  },
});
