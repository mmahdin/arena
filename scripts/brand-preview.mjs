import { createServer } from 'vite';
import react from '@vitejs/plugin-react';
import { pathToFileURL } from 'node:url';

/**
 * Serve the isolated Kenar identity gallery.
 *
 * It mounts the real screens with synthetic props only: no backend, no session,
 * no conversation API. Useful for visual review of the identity without wiring
 * a tester or the Go API.
 */
export async function startBrandPreview({ host = '127.0.0.1', port = 5174 } = {}) {
  const server = await createServer({
    configFile: false,
    plugins: [react(), {
      name: 'kenar-brand-preview-entry',
      configureServer(vite) {
        vite.middlewares.use((request, response, next) => {
          const path = request.url?.split('?')[0];
          if (path !== '/' && path !== '/index.html') return next();
          response.statusCode = 302;
          response.setHeader('Location', '/scripts/fixtures/brand-preview.html');
          response.end();
        });
      },
    }],
    server: {
      host,
      port,
      strictPort: true,
      // The fixture contains no private data; allow the Arena preview proxy.
      allowedHosts: ['.e2b.app'],
    },
  });
  await server.listen();
  return server;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const value = (flag, fallback) => {
    const index = process.argv.indexOf(flag);
    return index >= 0 ? process.argv[index + 1] : fallback;
  };
  const server = await startBrandPreview({
    host: value('--host', '127.0.0.1'),
    port: Number(value('--port', '5174')),
  });
  server.printUrls();
  const close = async () => { await server.close(); process.exit(0); };
  process.on('SIGTERM', close);
  process.on('SIGINT', close);
}
