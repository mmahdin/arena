import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Local synthetic-data only: same-origin API proxy avoids browser cross-origin cookies.
// Vite and the Go API should both be accessed as 127.0.0.1 (do not mix localhost).
export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: { '/api/v1': { target: 'http://127.0.0.1:8080', changeOrigin: false } },
  },
  preview: { host: '127.0.0.1' },
});
