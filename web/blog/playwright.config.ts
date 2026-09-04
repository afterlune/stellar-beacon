import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8080'
const useLocalServer = !process.env.E2E_BASE_URL

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  // The baseline shares one Vite dev server and measures cold startup. A
  // single worker avoids concurrent module compilation turning the budget and
  // feature-gate assertions into machine-load dependent flakes.
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
        command: 'npm run serve -- --host 127.0.0.1 --port 8080',
        url: baseURL,
        reuseExistingServer: true,
        timeout: 120_000,
        stdout: 'pipe',
        stderr: 'pipe'
      }
    : undefined,
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] }
    }
  ]
})
