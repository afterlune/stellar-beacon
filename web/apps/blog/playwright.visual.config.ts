import { defineConfig, devices } from '@playwright/test'

// Visual/a11y/perf gate for the reader-facing blog. It expects a running
// environment (integration stack or preview + API) instead of booting one,
// because the assertions are about rendered content.
const baseURL = process.env.BLOG_VISUAL_BASE_URL || 'http://127.0.0.1:18080'

export default defineConfig({
  testDir: './tests/visual',
  timeout: Number(process.env.BLOG_VISUAL_TIMEOUT || 120_000),
  workers: 1,
  retries: 0,
  reporter: [
    ['list'],
    ['json', { outputFile: 'test-results/visual/visual-report.json' }]
  ],
  use: {
    baseURL,
    locale: 'zh-CN',
    trace: 'off',
    screenshot: 'off',
    video: 'off',
    launchOptions: { args: ['--no-proxy-server'] }
  },
  outputDir: 'test-results/visual/artifacts',
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    { name: 'mobile', use: { ...devices['Pixel 5'], viewport: { width: 390, height: 844 } } }
  ]
})
