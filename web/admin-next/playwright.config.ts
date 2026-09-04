import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8082'
const useLocalServer = !process.env.E2E_BASE_URL

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  // All baseline cases share one Vite dev server and a stateful API mock.
  // Serial workers make refresh/menu assertions deterministic.
  workers: 1,
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: [['line'], ['html', { outputFolder: 'test-results/playwright-report', open: 'never' }]],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  },
  outputDir: 'test-results/artifacts',
  webServer: useLocalServer
    ? {
        command: 'npm run serve -- --host 127.0.0.1 --port 8082',
        url: baseURL,
        reuseExistingServer: true,
        timeout: 120_000,
        stdout: 'pipe',
        stderr: 'pipe'
      }
    : undefined,
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
})
