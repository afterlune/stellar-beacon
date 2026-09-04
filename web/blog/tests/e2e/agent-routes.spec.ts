import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (!url.pathname.startsWith('/api')) {
      await route.continue()
      return
    }

    let data: unknown = []
    if (url.pathname === '/api') {
      data = {
        viewCount: 1,
        articleCount: 1,
        talkCount: 0,
        categoryCount: 1,
        tagCount: 1,
        websiteConfigDTO: {
          siteName: 'Benetnasch',
          siteURL: '',
          siteAvatar: '',
          siteIntro: '',
          author: 'Benetnasch',
          authorAvatar: '',
          authorIntro: '',
          notice: '',
          websiteCreateTime: new Date(0).toISOString()
        }
      }
    } else if (url.pathname === '/api/agent/features') {
      data = {
        publicChat: true,
        vitals: false,
        galaxy: true,
        dreams: true,
        capsules: false,
        radio: false,
        videos: false,
        ttsEnabled: false
      }
    } else if (url.pathname === '/api/galaxy') {
      data = {
        records: [{ articleId: 7, lifeStage: 'growing', x: 0.2, y: -0.4, updatedAt: new Date(0).toISOString() }],
        count: 1,
        hasMore: false
      }
    } else if (url.pathname === '/api/dreams') {
      data = {
        records: [
          {
            id: 'dream-1',
            title: '一颗慢慢醒来的星',
            content: '公开梦境内容',
            imageUrl: '/dream-placeholder.svg',
            imageStatus: 'placeholder',
            createdAt: new Date(0).toISOString(),
            updatedAt: new Date(0).toISOString()
          }
        ],
        count: 1,
        hasMore: false
      }
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ flag: true, code: 20000, message: 'ok', data })
    })
  })
})

test('enabled Agent features are available through independent routes', async ({ page }) => {
  const pageErrors: string[] = []
  page.on('pageerror', (error) => pageErrors.push(error.stack || error.message))
  page.on('console', (message) => {
    if (message.type() === 'error') pageErrors.push(message.text())
  })

  await page.goto('/agent')
  await expect(page.getByRole('heading', { name: 'Benetnasch Agent' })).toBeVisible()
  await expect(page.getByRole('link', { name: /公开对话/ })).toBeVisible()
  await expect(page.getByRole('link', { name: /内容星河/ })).toBeVisible()
  await expect(page.getByRole('link', { name: /梦境档案/ })).toBeVisible()
  await expect(page.getByText('一颗慢慢醒来的星')).toBeVisible()

  await page.goto('/agent/chat')
  await expect(page.getByRole('textbox', { name: '对话内容' })).toBeVisible()

  await page.goto('/galaxy')
  await expect(page.locator('.agent-route-header').getByRole('heading', { name: '内容星河' })).toBeVisible()
  await expect(page.locator('.agent-galaxy__canvas')).toBeVisible()
  await expect(page.locator('.agent-galaxy')).toHaveAttribute('data-point-cap', '2000')
  await expect.poll(async () => page.locator('.agent-galaxy').getAttribute('data-measured-fps'), { timeout: 5_000 }).not.toBeNull()
  const measuredFps = Number(await page.locator('.agent-galaxy').getAttribute('data-measured-fps'))
  expect(measuredFps).toBeGreaterThan(0)
  if (measuredFps < 45) await expect(page.locator('.agent-galaxy')).toHaveAttribute('data-performance-mode', 'limited')

  await page.goto('/agent/dreams')
  await expect(page.locator('.agent-route-header').getByRole('heading', { name: '梦境档案' })).toBeVisible()
  await expect(page.getByText('一颗慢慢醒来的星')).toBeVisible()
  expect(pageErrors).toEqual([])
})

test('galaxy switches to the low-performance renderer on constrained devices', async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'deviceMemory', { configurable: true, value: 2 })
    Object.defineProperty(navigator, 'hardwareConcurrency', { configurable: true, value: 2 })
  })
  await page.goto('/galaxy')
  await expect(page.locator('.agent-galaxy[data-performance-mode="limited"]')).toBeVisible()
  await expect(page.locator('.agent-galaxy')).toHaveAttribute('data-point-cap', '400')
  await expect(page.getByText('低性能渲染已启用')).toBeVisible()
})

test('galaxy respects the reduced-motion accessibility preference', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.goto('/galaxy')
  await expect(page.locator('.agent-galaxy[data-performance-mode="limited"]')).toBeVisible()
  await expect(page.locator('.agent-galaxy')).toHaveAttribute('data-point-cap', '400')
  await expect(page.getByText('低性能渲染已启用')).toBeVisible()
})
