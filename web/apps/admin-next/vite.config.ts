import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const apiTarget = process.env.VITE_ADMIN_API_TARGET || process.env.VITE_API_TARGET || 'https://localhost:7777'

function manualChunks(id: string): string | undefined {
  const normalized = id.replace(/\\/g, '/')
  if (!normalized.includes('/node_modules/')) return undefined
  if (normalized.includes('/node_modules/@arco-design/')) return 'vendor-arco'
  if (normalized.includes('/node_modules/vue')) return 'vendor-vue'
  if (normalized.includes('/node_modules/@wangeditor-next/')) return 'vendor-editor'
  if (normalized.includes('/node_modules/echarts/') || normalized.includes('/node_modules/zrender/')) return 'vendor-echarts'
  if (normalized.includes('/node_modules/markdown-it') || normalized.includes('/node_modules/katex/')) return 'vendor-markdown'
  return 'vendor'
}

export default defineConfig({
  base: '/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@stellar-beacon/api-contract': fileURLToPath(new URL('../../packages/api-contract/src/index.ts', import.meta.url)),
      '@stellar-beacon/api-client': fileURLToPath(new URL('../../packages/api-client/src/index.ts', import.meta.url))
    }
  },
  server: {
    host: '127.0.0.1',
    port: 8082,
    watch: {
      // 编辑器/工具链在 Windows 上做原子保存时会先在目标目录写
      // `.Name.<pid>.<uuid>.tmpdir/Name.tmp` 再替换源文件。
      // 默认 watcher 会去 watch 这个临时目录并以 EBUSY 崩溃，
      // 同时导致替换失败（ReplaceFileW EIO），因此直接忽略。
      ignored: ['**/.*.tmpdir/**', '**/*.tmp', '**/*~']
    },
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        secure: false
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
