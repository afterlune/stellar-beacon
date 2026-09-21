import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), ['VITE_', 'VUE_APP_'])
  const apiTarget = env.VITE_API_TARGET || env.VUE_APP_API_TARGET || 'https://localhost:7777'

  return {
    base: env.VITE_BASE_URL || '/',
    plugins: [vue()],
    // Tocbot 4.x ships a legacy UMD wrapper that probes `global` during
    // module evaluation. Map it to the browser global so direct navigation
    // to Article/About remains a normal Vue route in Vite dev and production.
    define: {
      global: 'globalThis'
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    server: {
      host: '0.0.0.0',
      port: 8080,
      watch: {
        // Playwright streams videos and traces into the project while the dev
        // server runs; do not let the watcher churn on its own output.
        ignored: ['**/test-results/**', '**/playwright-report/**', '**/dist/**']
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
      sourcemap: false
    }
  }
})
