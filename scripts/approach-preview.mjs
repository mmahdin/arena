import { createServer } from 'vite';
import react from '@vitejs/plugin-react';
import { pathToFileURL } from 'node:url';

/** Serve only the isolated design fixture; no backend or tester session is implied. */
export async function startApproachPreview({ host = '127.0.0.1', port = 5173 } = {}) {
  const server = await createServer({
    configFile: false,
    plugins: [react(), {
      name: 'hamrah-approach-preview-entry',
      configureServer(vite) {
        vite.middlewares.use((request, response, next) => {
          const path = request.url?.split('?')[0];
          if (path !== '/' && path !== '/index.html') return next();
          response.statusCode = 302;
          response.setHeader('Location', '/scripts/fixtures/approach-preview.html');
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
  const server = await startApproachPreview({ host: value('--host', '127.0.0.1'), port: Number(value('--port', '5173')) });
  server.printUrls();
  const close = async () => { await server.close(); process.exit(0); };
  process.on('SIGTERM', close);
  process.on('SIGINT', close);
}
