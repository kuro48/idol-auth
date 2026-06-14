import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'node',
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/login': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/register': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/legal': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/account/restore': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
