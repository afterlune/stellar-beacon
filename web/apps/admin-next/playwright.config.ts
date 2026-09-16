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
    // 后台的默认界面语言跟随浏览器语言，而 E2E 断言用的是中文选择器
    // （`getByRole('button', { name: '新增' })` 等）。显式钉住 browser locale，
    // 否则 Desktop Chrome 的 en-US 会让整个套件渲染成英文而全部失败。
    locale: 'zh-CN',
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
