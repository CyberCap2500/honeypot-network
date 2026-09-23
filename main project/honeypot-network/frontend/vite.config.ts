// ©AngelaMos | 2026
// vite.config.ts

import { resolve } from 'node:path'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  css: {
    modules: {
      localsConvention: 'camelCaseOnly',
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8184',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8184',
        ws: true,
      },
    },
  },
})
