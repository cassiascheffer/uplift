// ABOUTME: Vite configuration for the Uplift web app
// ABOUTME: Configures Tailwind CSS and DaisyUI processing via Vite, proxies WebSocket to Go backend
import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [
    tailwindcss()
  ],
  root: 'src',
  publicDir: '../static',
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
  server: {
    port: 3000,
    proxy: {
      '/ws': {
        target: 'http://localhost:8080',
        ws: true,
      },
    },
  },
});
