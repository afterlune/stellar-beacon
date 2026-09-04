import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const apiTarget = process.env.VITE_ADMIN_API_TARGET || process.env.VITE_API_TARGET || 'https://localhost:7777'
const preserveApiPrefix = process.env.VITE_ADMIN_API_PRESERVE_API_PREFIX === '1'

function manualChunks(id: string): string | undefined {
  const normalized = id.replace(/\\/g, '/')
  if (!normalized.includes('/node_modules/')) return undefined
  if (normalized.includes('/node_modules/@arco-design/')) return 'vendor-arco'
  if (normalized.includes('/node_modules/vue')) return 'vendor-vue'
  return 'vendor'
}

export default defineConfig({
  base: '/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@shared': fileURLToPath(new URL('../shared', import.meta.url))
    }
  },
  server: {
    host: '127.0.0.1',
    port: 8082,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        secure: false,
        ...(preserveApiPrefix ? {} : { rewrite: (path: string) => path.replace(/^\/api/, '') })
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: { manualChunks }
    }
  }
})
