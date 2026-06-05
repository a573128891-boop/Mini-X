import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: 'dist',
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'https://alluring-nurturing-production.up.railway.app',
        changeOrigin: true,
      },
      '/ws': {
        target: 'https://alluring-nurturing-production.up.railway.app',
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
