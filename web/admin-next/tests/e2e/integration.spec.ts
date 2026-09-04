import { expect, test } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const allowLogin = process.env.E2E_ADMIN_ALLOW_LOGIN === '1'

test.describe('admin-next isolated integration', () => {
  test('logs in and refreshes core migrated menu routes without blank pages @integration', async ({ page }) => {
    test.skip(
      !realIntegration || !allowLogin || !process.env.E2E_ADMIN_EMAIL || !process.env.E2E_ADMIN_PASSWORD,
      'set E2E_REAL_INTEGRATION=1, E2E_ADMIN_ALLOW_LOGIN=1 and admin credentials to run the write-capable login flow'
    )

    const email = process.env.E2E_ADMIN_EMAIL || ''
    const password = process.env.E2E_ADMIN_PASSWORD || ''
    if (!email || !password) throw new Error('E2E_ADMIN_EMAIL and E2E_ADMIN_PASSWORD are required')

    const pageErrors: string[] = []
    page.on('pageerror', (error) => pageErrors.push(error.message))
    page.on('console', (message) => {
      if (message.type() === 'error') pageErrors.push(message.text())
    })

    await page.goto('/login')
    await page.getByTestId('login-username').locator('input').fill(email)
    await page.getByTestId('login-password').locator('input').fill(password)
    await page.getByTestId('login-submit').click()
    await page.waitForURL((url) => url.pathname === '/')
    await expect(page.locator('.admin-shell')).toBeVisible()
    const home = page.getByRole('main')
    await expect(home).toContainText('会话状态')
    await expect(home).toContainText('已认证')
    await expect(home).toContainText('迁移策略')
    await expect(home).toContainText('渐进式')

    const routes = [
      '/article-list',
      '/categories',
      '/tags',
      '/comments',
      '/users',
      '/roles',
      '/operation/log',
      '/exception/log',
      '/quartz',
      '/albums',
      '/talk-list'
    ]
    for (const route of routes) {
      const response = await page.goto(route, { waitUntil: 'domcontentloaded' })
      expect(response?.status(), route).toBe(200)
      await expect(page.locator('.admin-content')).toBeVisible()
      await expect(page.locator('.admin-content')).not.toContainText('页面不存在')
      await page.reload()
      await expect(page.locator('.admin-content')).toBeVisible()
      await expect(page.locator('.admin-content')).not.toContainText('页面不存在')
    }

    expect(pageErrors).toEqual([])
  })
})
