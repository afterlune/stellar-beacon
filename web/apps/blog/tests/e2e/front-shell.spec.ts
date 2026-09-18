import { expect, test, type Page } from '@playwright/test'

const routes = ['/archives', '/tags', '/talks', '/friends', '/message', '/about', '/photos/0']

function articleFixture(id: number, title: string) {
  return {
    id,
    articleTitle: title,
    articleContent: '正文内容',
    articleContentHtml: `<h2 id="article-section-${id}">第一节</h2><p>正文内容</p><h2>第二节</h2>${Array.from({ length: 16 }, (_, index) => `<p>第 ${index + 1} 段测试内容，用于验证移动端目录和系列阅读进度。</p>`).join('')}`,
    articleCover: '',
    categoryName: '测试分类',
    status: 1,
    createTime: '2026-09-18T10:00:00+08:00',
    updateTime: '2026-09-18T10:00:00+08:00',
    seriesId: 3,
    seriesOrder: id - 6,
    tags: [{ id: 1, tagName: '测试标签' }],
    author: { nickname: '测试作者', avatar: '', website: '' },
    likeCount: 0,
    favoriteCount: 0,
    relatedArticles: [
      { id: 20, articleTitle: '标签相关文章', categoryName: '另一分类' },
      { id: 21, articleTitle: '分类相关文章', categoryName: '测试分类' }
    ],
    preArticleCard: { id: 0, articleContent: '' },
    nextArticleCard: { id: 0, articleContent: '' }
  }
}

async function mockArticleReading(page: Page, onContinuationEvent?: (event: { articleId: number; eventType: string; targetType?: string; targetId?: number; placement?: string; position?: number }) => void): Promise<void> {
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const continuationMatch = path.match(/^\/api\/v1\/public\/articles\/(\d+)\/continuation-events$/)
    if (continuationMatch) {
      const body = route.request().postDataJSON() as { eventType?: string; targetType?: string; targetId?: number; placement?: string; position?: number }
      onContinuationEvent?.({ articleId: Number(continuationMatch[1]), eventType: String(body?.eventType || ''), targetType: body?.targetType, targetId: body?.targetId, placement: body?.placement, position: body?.position })
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 'OK', message: '操作成功', data: null }) })
      return
    }
    if (path === '/api/v1/public/reports/visit') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 'OK', message: '操作成功', data: null }) })
      return
    }
    const articleMatch = path.match(/^\/api\/v1\/public\/articles\/(\d+)$/)
    if (articleMatch) {
      const id = Number(articleMatch[1])
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 'OK', message: '操作成功', data: articleFixture(id, `系列第${id - 6}篇`) })
      })
      return
    }
    if (path === '/api/v1/public/series/3') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 'OK',
          message: '操作成功',
          data: {
            series: { id: 3, seriesName: '阅读系列' },
            articles: [
              { id: 7, articleTitle: '系列第一篇' },
              { id: 8, articleTitle: '系列第二篇' },
              { id: 9, articleTitle: '系列第三篇' }
            ]
          }
        })
      })
      return
    }
    if (path === '/api/v1/public/comments') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 'OK', message: '操作成功', data: { items: [], total: 0, page: 1, pageSize: 7 } }) })
      return
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 'OK', message: '操作成功', data: null }) })
  })
}

test.describe('blog front shell', () => {
  test('renders the new editorial shell and navigates between pages', async ({ page }, testInfo) => {
    await page.goto('/')

    await expect(page.locator('.site-header')).toBeVisible()
    await expect(page.locator('.home-hero')).toBeVisible()
    await expect(page.locator('.ambient-grid')).toBeVisible()
    await expect(page.locator('#footer')).toBeVisible()

    if (testInfo.project.name === 'mobile') {
      // Opening the drawer transforms the app during the same pointer frame;
      // invoke the real button handler without a pointer retry against the
      // newly transformed overlay.
      await page.locator('[data-dia="menu"]').evaluate((element) => (element as HTMLElement).click())
      await expect(page.locator('#App-Mobile-Profile')).toHaveCSS('opacity', '1')
      await page.locator('#App-Mobile-Profile').getByText(/^(about|关于)$/i).click()
    } else {
      await page.locator('[data-menu="About"]').click()
    }
    await expect(page).toHaveURL(/\/about$/)
    await expect(page.locator('.post-header')).toBeVisible()
  })

  test('keeps the static routes free of horizontal overflow', async ({ page }) => {
    for (const route of routes) {
      await page.goto(route)
      await expect(page.locator('#App-Container')).toBeVisible()
      const fits = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)
      expect(fits, `${route} should not overflow horizontally`).toBe(true)
    }
  })
})

test.describe('mobile front shell', () => {
  test('opens the navigation drawer and follows a route', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'This case only applies to the mobile project')
    await page.goto('/')
    await page.locator('[data-dia="menu"]').evaluate((element) => (element as HTMLElement).click())
    await expect(page.locator('#App-Mobile-Profile')).toHaveCSS('opacity', '1')

    await page.locator('#App-Mobile-Profile').getByText(/^(about|关于)$/i).click()
    await expect(page).toHaveURL(/\/about$/)
  })
})

test.describe('article reading experience', () => {
  test('tracks visible continuation impressions and clicks', async ({ page }) => {
    test.setTimeout(60_000)
    const events: Array<{ articleId: number; eventType: string; targetType?: string; targetId?: number; placement?: string; position?: number }> = []
    await mockArticleReading(page, (event) => events.push(event))

    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    await page.getByTestId('series-context').evaluate((element) => (element as HTMLElement).scrollIntoView({ block: 'center' }))
    await expect.poll(() => events.filter((event) => event.articleId === 8 && event.eventType === 'series_impression').length).toBe(1)
    await page.getByTestId('series-next').evaluate((element) => (element as HTMLElement).click())
    await expect(page).toHaveURL(/\/articles\/9$/)
    await expect.poll(() => events.filter((event) => event.articleId === 8 && event.eventType === 'series_click').length).toBe(1)
    expect(events.find((event) => event.eventType === 'series_click')).toMatchObject({ targetType: 'article', targetId: 9, placement: 'series_next', position: 0 })

    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    await page.getByTestId('related-articles').evaluate((element) => (element as HTMLElement).scrollIntoView({ block: 'center' }))
    await expect.poll(() => events.filter((event) => event.articleId === 8 && event.eventType === 'related_impression').length).toBe(1)
    await page.getByTestId('related-articles').locator('a').first().evaluate((element) => (element as HTMLElement).click())
    await expect.poll(() => events.filter((event) => event.articleId === 8 && event.eventType === 'related_click').length).toBe(1)
    expect(events.find((event) => event.eventType === 'related_click')).toMatchObject({ targetType: 'article', targetId: 20, placement: 'related', position: 1 })
  })
  test('opens the mobile reader toc and jumps to a section', async ({ page }, testInfo) => {
    test.setTimeout(60_000)
    test.skip(testInfo.project.name !== 'mobile', 'This case only applies to the mobile project')
    await mockArticleReading(page)
    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('heading', { name: '第一节' })).toBeVisible()

    await page.locator('.Ob-Navigator-ball').click()
    const tocAction = page.locator('#Ob-Navigator-toc')
    await expect(tocAction).toBeVisible()
    await tocAction.click()

    const drawer = page.getByTestId('article-reader-drawer')
    await expect(drawer).toBeVisible()
    await expect(drawer).toContainText('第一节')
    await expect(drawer).toContainText('阅读系列')
    await drawer.getByRole('button', { name: '第二节' }).click()

    await expect(drawer).not.toBeVisible()
    await expect.poll(() => page.evaluate(() => window.location.hash)).toBe('#article-heading-2')
  })

  test('persists completed series progress and offers the next article', async ({ page }) => {
    test.setTimeout(60_000)
    await mockArticleReading(page)
    await page.goto('/articles/7', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('heading', { name: '第一节' })).toBeVisible()
    await page.waitForTimeout(3200)
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))

    await expect.poll(() => page.evaluate(() => {
      const raw = localStorage.getItem('stellar-beacon.reader.series-progress.v1')
      if (!raw) return false
      const progress = JSON.parse(raw) as Record<string, { completedArticleIds?: number[] }>
      return progress['3']?.completedArticleIds?.includes(7) || false
    })).toBe(true)

    await page.goto('/series/3', { waitUntil: 'domcontentloaded' })
    await expect(page.getByTestId('series-progress')).toContainText(/已读 1\/3|Read 1\/3/)
    const continueLink = page.getByTestId('series-continue')
    await expect(continueLink).toContainText('系列第二篇')
    await expect(continueLink).toHaveAttribute('href', '/articles/8')
  })
  test('ignores malformed local series progress', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('stellar-beacon.reader.series-progress.v1', '{broken'))
    await mockArticleReading(page)
    await page.goto('/series/3', { waitUntil: 'domcontentloaded' })
    await expect(page.getByTestId('series-progress')).toContainText(/已读 0\/3|Read 0\/3/)
    await expect(page.getByTestId('series-continue')).toContainText('系列第一篇')
  })
  test('keeps series navigation separate from related reading', async ({ page }) => {
    test.setTimeout(60_000)
    await mockArticleReading(page)
    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })

    const series = page.getByTestId('series-context')
    await expect(series).toBeVisible()
    await expect(series).toContainText('阅读系列')
    await expect(series).toContainText(/系列进度 2\/3|Collection progress 2\/3/)
    await expect(page.getByTestId('series-previous')).toContainText('系列第一篇')
    await expect(page.getByTestId('series-next')).toContainText('系列第三篇')

    const related = page.getByTestId('related-articles')
    await expect(related).toContainText('标签相关文章')
    await expect(related).toContainText('分类相关文章')
    await expect(related).not.toContainText('系列第一篇')

    await page.goto('/articles/7', { waitUntil: 'domcontentloaded' })
    await expect(page.getByTestId('series-previous')).toHaveCount(0)
    await expect(page.getByTestId('series-next')).toContainText('系列第二篇')

    await page.goto('/articles/9')
    await expect(page.getByTestId('series-previous')).toContainText('系列第二篇')
    await expect(page.getByTestId('series-next')).toHaveCount(0)
  })
})
