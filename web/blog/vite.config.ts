import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const apiTarget = process.env.VITE_API_TARGET || process.env.VUE_APP_API_TARGET || 'https://localhost:7777'

function manualChunks(id: string): string | undefined {
  const normalized = id.replace(/\\/g, '/')
  if (!normalized.includes('/node_modules/')) return undefined
  if (normalized.includes('/node_modules/markdown-it')) return 'vendor-markdown'
  if (normalized.includes('/node_modules/element-plus')) return 'vendor-element-plus'
  if (normalized.includes('/node_modules/prismjs')) return 'vendor-prism'
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
    port: 8080,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        secure: false,
        rewrite: (path) => path.replace(/^\/api/, '')
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
