import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: true,
    allowedHosts: [
      'www.xpan9.tech',
      'xarvis.xpan9.tech',
      'localhost',
      '127.0.0.1'
    ],
    proxy: {
      '/v1': {
        target: 'http://xarvis-core:8088',
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: 'ws://xarvis-core:8088',
        ws: true,
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  }
})

