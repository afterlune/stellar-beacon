import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.BLOG_BASE_URL || 'http://127.0.0.1:8080'
const useLocalServer = !process.env.BLOG_BASE_URL

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: [['line'], ['html', { outputFolder: 'test-results/playwright-report', open: 'never' }]],
  use: {
    baseURL,
    locale: 'zh-CN',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    launchOptions: {
      // A system-wide proxy client can intercept loopback traffic and stall
      // navigations; the suite only talks to its local server.
      args: ['--no-proxy-server'],
      ...(process.env.BLOG_PLAYWRIGHT_CHROMIUM_PATH
        ? { executablePath: process.env.BLOG_PLAYWRIGHT_CHROMIUM_PATH }
        : {})
    }
  },
  outputDir: 'test-results/artifacts',
  webServer: useLocalServer
    ? {
        // Serve the production bundle: the dev server rewrites every module on
        // demand, which makes the article routes slow and flaky under test.
        command: 'npm run build && npm run preview -- --host 127.0.0.1 --port 8080',
        url: baseURL,
        reuseExistingServer: true,
        timeout: 180_000,
        stdout: 'pipe',
        stderr: 'pipe'
      }
    : undefined,
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'] } },
    { name: 'mobile', use: { ...devices['Pixel 5'] } }
  ]
})
