import { createServer } from 'vite';
import react from '@vitejs/plugin-react';
import { pathToFileURL } from 'node:url';

/**
 * Serve the isolated Kenar identity preview. It mounts the real screens with
 * fixture data and never touches a backend, a tester session or stored data.
 * This is design tooling; the shipped app does not include the preview bar.
 */
export async function startIdentityPreview({ host = '127.0.0.1', port = 5174 } = {}) {
  const server = await createServer({
    configFile: false,
    plugins: [react(), {
      name: 'kenar-identity-preview-entry',
      configureServer(vite) {
        vite.middlewares.use((request, response, next) => {
          const path = request.url?.split('?')[0];
          if (path !== '/' && path !== '/index.html') return next();
          response.statusCode = 302;
          response.setHeader('Location', '/scripts/fixtures/identity-preview.html');
          response.end();
        });
      },
    }],
    server: {
      host,
      port,
      strictPort: true,
      // The fixture contains only synthetic content; allow the Arena preview proxy.
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
  const server = await startIdentityPreview({ host: value('--host', '127.0.0.1'), port: Number(value('--port', '5174')) });
  server.printUrls();
  const close = async () => { await server.close(); process.exit(0); };
  process.on('SIGTERM', close);
  process.on('SIGINT', close);
}
