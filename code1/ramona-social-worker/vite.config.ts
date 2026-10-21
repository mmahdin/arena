import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

/**
 * `@/` alias resolves to `src/`. Kept dependency-free (no node:path import) so
 * the project type-checks without Node type definitions.
 */
const srcDir = new URL("./src", import.meta.url).pathname;

export default defineConfig({
  base: "./",
  plugins: [react()],
  resolve: {
    alias: { "@": srcDir },
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    allowedHosts: true,
  },
  preview: {
    host: "0.0.0.0",
    port: 4173,
    allowedHosts: true,
  },
});
