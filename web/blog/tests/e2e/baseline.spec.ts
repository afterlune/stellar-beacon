import { expect, Page, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  // The baseline validates the current frontend shell and routing independently
  // from mutable backend data. The real API is exercised by the integration
  // smoke scripts and by the deployed-browser run with E2E_BASE_URL.
  await page.route('**/*', async (route) => {
    const requestURL = new URL(route.request().url())
    if (!requestURL.pathname.startsWith('/api')) {
      await route.continue()
      return
    }

    let data: unknown = []
    if (requestURL.pathname === '/api') {
      data = {
        viewCount: 0,
        articleCount: 0,
        talkCount: 0,
        categoryCount: 0,
        tagCount: 0,
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
    } else if (requestURL.pathname.endsWith('/articles/topAndFeatured')) {
      data = { topArticle: { articleContent: '' }, featuredArticles: [] }
    } else if (
      requestURL.pathname.endsWith('/articles/all') ||
      requestURL.pathname.endsWith('/articles/categoryId') ||
      requestURL.pathname.endsWith('/archives/all')
    ) {
      data = { records: [], count: 0 }
    } else if (
      requestURL.pathname.endsWith('/categories/all') ||
      requestURL.pathname.endsWith('/tags/topTen') ||
      requestURL.pathname.endsWith('/comments/topSix') ||
      requestURL.pathname.endsWith('/photos/albums')
    ) {
      data = []
    } else if (requestURL.pathname.endsWith('/agent/features')) {
      data = {
        publicChat: false,
        vitals: false,
        galaxy: false,
        dreams: false,
        capsules: false,
        radio: false,
        videos: false,
        ttsEnabled: false
      }
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ flag: true, code: 20000, message: 'ok', data })
    })
  })
})

test.describe('blog baseline', () => {
  test('homepage renders the SPA shell with optional Agent widgets closed', async ({ page }) => {
    const pageErrors = capturePageErrors(page)
    const response = await page.goto('/')
    expect(response?.status()).toBe(200)
    await expect(page.locator('#app')).toBeVisible()
    await expect(page.locator('#App-Container')).toBeVisible()
    await expect(page.locator('.agent-radio')).toHaveCount(0)
    await expect(page.locator('.agent-videos')).toHaveCount(0)
    expect(pageErrors()).toEqual([])
  })

  test('history deep link keeps the SPA shell and route path', async ({ page }) => {
    const pageErrors = capturePageErrors(page)
    const response = await page.goto('/archives')
    expect(response?.status()).toBe(200)
    await expect(page.locator('#App-Container')).toBeVisible()
    expect(new URL(page.url()).pathname).toBe('/archives')
    expect(pageErrors()).toEqual([])
  })

  test('homepage first screen stays within the browser baseline budget', async ({ page }) => {
    const response = await page.goto('/', { waitUntil: 'load' })
    expect(response?.status()).toBe(200)
    const timing = await page.evaluate(() => {
      const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined
      const paint = performance.getEntriesByName('first-contentful-paint')[0]
      return {
        domContentLoaded: navigation?.domContentLoadedEventEnd ?? 0,
        loadEvent: navigation?.loadEventEnd ?? 0,
        firstContentfulPaint: paint?.startTime ?? 0
      }
    })
    expect(timing.domContentLoaded).toBeGreaterThan(0)
    expect(timing.domContentLoaded).toBeLessThan(5_000)
    expect(timing.loadEvent).toBeLessThan(5_000)
    expect(timing.firstContentfulPaint).toBeGreaterThan(0)
    expect(timing.firstContentfulPaint).toBeLessThan(5_000)
  })

  test('Agent routes fail closed while their public flags are disabled', async ({ page }) => {
    const pageErrors = capturePageErrors(page)
    for (const path of ['/agent', '/agent/chat', '/galaxy', '/dreams']) {
      const response = await page.goto(path)
      expect(response?.status()).toBe(200)
      await expect(page.locator('[data-agent-route-state="disabled"]')).toBeVisible()
      expect(new URL(page.url()).pathname).toBe(path)
    }
    expect(pageErrors()).toEqual([])
  })
})

function capturePageErrors(page: Page): () => string[] {
  const errors: string[] = []
  page.on('pageerror', (error: Error) => errors.push(error.stack || error.message))
  page.on('console', (message: { type(): string; text(): string }) => {
    if (message.type() === 'error') errors.push(message.text())
  })
  return () => errors
}
