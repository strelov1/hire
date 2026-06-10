import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  resolve: {
    alias: {
      $lib: path.resolve('./src/lib'),
    },
  },
  server: {
    port: 5173,
    strictPort: false,
    // Proxy the API so the browser only ever talks to this origin. That makes
    // dev match the same-origin production deployment, so the SameSite=Lax auth
    // cookie is sent and no CORS is needed. Target overridable via VITE_API_URL.
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
