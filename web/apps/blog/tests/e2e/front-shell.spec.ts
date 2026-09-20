import { expect, test, type Page } from '@playwright/test'

const routes = ['/archives', '/categories', '/reading', '/series', '/search', '/tags', '/talks', '/friends', '/message', '/about', '/photos/0']

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
    tags: ['测试标签', '系列'],
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
    if (path === '/api/v1/public/series') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 'OK',
          message: '操作成功',
          data: [
            {
              id: 3,
              seriesName: '阅读系列',
              seriesDesc: '从零搭建阅读体验',
              cover: '',
              articleCount: 3,
              updateTime: '2026-09-18T10:00:00+08:00'
            }
          ]
        })
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

test.beforeEach(async ({ page }) => {
  // The blog shell loads KaTeX from a CDN. Tests must not depend on external
  // network availability or wait for that script during navigation.
  await page.route('https://cdnjs.cloudflare.com/**', (route) => route.abort())
})
test.describe('blog front shell', () => {
  test('renders the new editorial shell and navigates between pages', async ({ page }, testInfo) => {
    await page.goto('/')

    await expect(page.locator('.site-header')).toBeVisible()
    await expect(page.locator('.plaza-hero')).toBeVisible()
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



function searchHitFixture(id: number, title: string, content: string) {
  return { id, articleTitle: title, articleContent: content, status: 1, isDelete: 0 }
}

async function mockContentDiscovery(page: Page, options: { searchFails?: boolean } = {}): Promise<void> {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname
    const fulfill = (data: unknown, status = 200) => route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(status >= 400
        ? { code: 'ERROR', message: 'request failed', data: null }
        : { code: 'OK', message: '操作成功', data })
    })

    if (path === '/api/v1/public/') {
      await fulfill({
        viewCount: 0,
        articleCount: 0,
        talkCount: 0,
        categoryCount: 0,
        tagCount: 0,
        websiteConfigDTO: {
          name: 'Stellar Beacon',
          englishName: 'Stellar Beacon',
          author: '测试作者',
          authorIntro: '',
          notice: '',
          multiLanguage: 0
        }
      })
      return
    }
    if (path === '/api/v1/public/authors') {
      await fulfill({
        items: [{ id: 1, handle: 'test-author', nickname: '测试作者', avatar: '', intro: '公共空间作者', articleCount: 1, talkCount: 0, seriesCount: 0 }],
        total: 1,
        page: 1,
        pageSize: 12
      })
      return
    }
    if (path === '/api/v1/public/authors/test-author') {
      await fulfill({ id: 1, handle: 'test-author', nickname: '测试作者', avatar: '', intro: '公共空间作者', website: 'https://example.com', articleCount: 1, talkCount: 1, seriesCount: 1 })
      return
    }
    if (path === '/api/v1/public/authors/test-author/articles') {
      await fulfill({ items: [{ id: 101, articleTitle: '公共空间的第一篇文章', articleContent: '由社区作者共同发布的公开内容。', categoryName: '工程实践', createTime: '2026-09-18T10:00:00+08:00' }], total: 1, page: 1, pageSize: 12 })
      return
    }
    if (path === '/api/v1/public/authors/test-author/talks') {
      await fulfill({ items: [{ id: 51, content: '作者的一条公开随想', createTime: '2026-09-18T10:00:00+08:00', commentCount: 0 }], total: 1, page: 1, pageSize: 12 })
      return
    }
    if (path === '/api/v1/public/authors/test-author/series') {
      await fulfill({ items: [{ id: 3, seriesName: '阅读系列', seriesDesc: '从零搭建阅读体验', articleCount: 3 }], total: 1, page: 1, pageSize: 12 })
      return
    }
    if (path === '/api/v1/public/feed') {
      await fulfill({
        items: [{
          id: 101,
          articleTitle: '公共空间的第一篇文章',
          articleContent: '由社区作者共同发布的公开内容。',
          articleCover: '',
          categoryName: '工程实践',
          status: 1,
          moderationStatus: 'visible',
          createTime: '2026-09-18T10:00:00+08:00',
          author: { id: 1, handle: 'test-author', nickname: '测试作者', avatar: '' },
          likeCount: 0,
          favoriteCount: 0
        }],
        total: 1,
        page: 1,
        pageSize: 12
      })
      return
    }    if (path === '/api/v1/public/categories') {
      await fulfill([
        { id: 1, categoryName: '工程实践', articleCount: 9 },
        { id: 2, categoryName: '系统设计', articleCount: 4 }
      ])
      return
    }
    if (path === '/api/v1/public/tags') {
      await fulfill([
        { id: 1, tagName: 'Go', count: 6 },
        { id: 2, tagName: 'Vue', count: 3 }
      ])
      return
    }
    if (path === '/api/v1/public/tags/top') {
      await fulfill([])
      return
    }
    if (path === '/api/v1/public/series') {
      await fulfill([
        {
          id: 3,
          seriesName: '阅读系列',
          seriesDesc: '从零搭建阅读体验',
          cover: '',
          articleCount: 3,
          updateTime: '2026-09-18T10:00:00+08:00'
        }
      ])
      return
    }
    if (path === '/api/v1/public/albums' || path === '/api/v1/public/comments/top') {
      await fulfill([])
      return
    }
    if (path === '/api/v1/public/series/3') {
      await fulfill({
        series: { id: 3, seriesName: '阅读系列' },
        articles: [
          { id: 7, articleTitle: '系列第一篇' },
          { id: 8, articleTitle: '系列第二篇' },
          { id: 9, articleTitle: '系列第三篇' }
        ]
      })
      return
    }
    if (path === '/api/v1/public/articles/featured') {
      await fulfill({ topArticle: null, featuredArticles: [] })
      return
    }
    if (path === '/api/v1/public/articles') {
      await fulfill({ items: [], records: [], total: 0, count: 0, page: 1, pageSize: 12 })
      return
    }
    if (path === '/api/v1/public/articles/search') {
      if (options.searchFails) {
        await fulfill(null, 500)
        return
      }
      const keywords = url.searchParams.get('keywords') || ''
      if (keywords.includes('无结果')) {
        await fulfill({ items: [], total: 0, page: 1, pageSize: 20 })
        return
      }
      if (keywords.includes('xss')) {
        await fulfill({
          items: [searchHitFixture(99, '危险 <img src=x onerror="window.__searchXss = true">', '正文 <mark>高亮</mark>')],
          total: 1,
          page: 1,
          pageSize: 5
        })
        return
      }
      const current = Math.max(1, Number(url.searchParams.get('current') || 1) || 1)
      const size = Math.max(1, Number(url.searchParams.get('size') || 20) || 20)
      const total = 25
      const items = []
      for (let index = 0; index < size; index += 1) {
        const id = (current - 1) * size + index + 1
        if (id > total) break
        items.push(searchHitFixture(id, `第 ${id} 篇匹配文章`, '正文包含 <mark>关键词</mark> 高亮'))
      }
      await fulfill({ items, total, page: current, pageSize: size })
      return
    }
    if (path === '/api/v1/public/articles/by-category' || path === '/api/v1/public/articles/by-tag') {
      await fulfill({
        items: [{
          id: 31,
          articleTitle: '目录文章',
          articleContent: '目录正文',
          categoryName: '工程实践',
          status: 1,
          createTime: '2026-09-18T10:00:00+08:00',
          updateTime: '2026-09-18T10:00:00+08:00',
          tags: [],
          author: { nickname: '作者', avatar: '', website: '' },
          likeCount: 0,
          favoriteCount: 0
        }],
        records: [],
        total: 1,
        count: 1,
        page: 1,
        pageSize: 12
      })
      return
    }
    await fulfill(null)
  })
}

test.describe('content discovery', () => {
  test.describe.configure({ timeout: 90_000 })

  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      document.cookie = 'locale=cn; path=/'
      sessionStorage.setItem('token', 'e2e-reading-token')
    })
  })

  test('home presents the public plaza and links to an author channel', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })

    await expect(page.locator('.plaza-hero')).toBeVisible()
    await expect(page.getByText('测试作者').first()).toBeVisible()
    await expect(page.getByTestId('public-feed')).toContainText('公共空间的第一篇文章')

    const authorLink = page.getByRole('link', { name: /测试作者/ }).first()
    await expect(authorLink).toHaveAttribute('href', '/u/test-author')
    await expect(page.getByRole('link', { name: '公共空间的第一篇文章' })).toHaveAttribute('href', '/articles/101')
  })

  test('author page presents identity and switches content channels', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/u/test-author', { waitUntil: 'domcontentloaded' })

    await expect(page.getByRole('heading', { name: '测试作者' })).toBeVisible()
    await expect(page.getByText('@test-author')).toBeVisible()
    await expect(page.getByRole('link', { name: '公共空间的第一篇文章' })).toHaveAttribute('href', '/articles/101')

    await page.getByRole('button', { name: '随想' }).click()
    await expect(page.getByText('作者的一条公开随想')).toBeVisible()
  })

  test('category overview links into a category article list', async ({ page }) => {    await mockContentDiscovery(page)
    await page.goto('/categories', { waitUntil: 'domcontentloaded' })

    const grid = page.getByTestId('categories-grid')
    await expect(grid).toBeVisible()
    await grid.getByRole('link', { name: /工程实践/ }).click()

    await expect(page).toHaveURL(/\/categories\/1\?name=/)
    await expect(page.locator('.post-title')).toContainText('工程实践')
    await expect(page.locator('.tag-article')).toContainText('目录文章')
  })

  test('tag route serves its article list and keeps the legacy path working', async ({ page }) => {
    await mockContentDiscovery(page)

    await page.goto('/tags/1?tagName=Go', { waitUntil: 'domcontentloaded' })
    await expect(page.locator('.post-title')).toContainText('Go')
    await expect(page.locator('.tag-article')).toContainText('目录文章')

    await page.goto('/article-list/1?tagName=Go', { waitUntil: 'domcontentloaded' })
    await expect(page).toHaveURL(/\/tags\/1\?tagName=Go$/)
    await expect(page.locator('.tag-article')).toContainText('目录文章')
  })

  test('search page paginates results and restores state from the URL', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/search?q=关键词&page=2', { waitUntil: 'domcontentloaded' })

    const results = page.getByTestId('search-results')
    await expect(results).toContainText('共找到 25 篇')
    await expect(page.locator('.search-page__list li')).toHaveCount(5)
    await expect(page.locator('.paginator .active')).toContainText('2')

    await page.locator('.paginator li').first().click()
    await expect.poll(() => new URL(page.url()).searchParams.get('page')).toBeNull()
    await expect(page.locator('.search-page__list li')).toHaveCount(20)
  })

  test('search page offers matching topic shortcuts', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/search?q=阅读', { waitUntil: 'domcontentloaded' })

    const topics = page.getByTestId('search-topics')
    await expect(topics).toBeVisible()
    await expect(topics).toContainText('阅读系列')
    await expect(topics.getByRole('link', { name: /阅读系列/ })).toHaveAttribute('href', '/series/3')
  })

  test('search page reports empty results and backend failures', async ({ page }) => {
    const options: { searchFails?: boolean } = {}
    await mockContentDiscovery(page, options)

    await page.goto('/search?q=无结果', { waitUntil: 'domcontentloaded' })
    await expect(page.getByTestId('search-results')).toContainText('没有找到匹配的文章。')

    options.searchFails = true
    await page.goto('/search?q=关键词', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('alert')).toContainText('搜索暂时不可用，请稍后再试。')
  })

  test('search highlights escape untrusted markup', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/search?q=xss', { waitUntil: 'domcontentloaded' })

    const first = page.locator('.search-page__list li').first()
    await expect(first).toContainText('<img src=x onerror="window.__searchXss = true">')
    await expect(first.locator('img')).toHaveCount(0)
    expect(await page.evaluate(() => (window as unknown as { __searchXss?: boolean }).__searchXss)).toBeUndefined()
  })

  test('search modal previews five hits, shows topics and opens the full search', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.locator('[data-dia="search"]').click()

    const input = page.locator('#search-input')
    await expect(input).toBeVisible()
    await input.fill('阅读')

    await expect(page.locator('#search-menu li')).toHaveCount(5)
    await expect(page.getByTestId('search-topic-preview')).toContainText('阅读系列')
    await expect(page.locator('.search-hit-title mark').first()).toBeVisible()

    await page.locator('.search-view-all').click()
    await expect(page).toHaveURL(/\/search\?q=/)
    await expect(page.getByTestId('search-results')).toContainText('共找到 25 篇')
  })

  test('desktop navigation links into categories and series', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'Desktop header navigation only')
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })

    await page.locator('[data-menu="Categories"]').click()
    await expect(page).toHaveURL(/\/categories$/)
    await expect(page.getByTestId('categories-grid')).toContainText('工程实践')

    await page.locator('[data-menu="Series"]').click()
    await expect(page).toHaveURL(/\/series$/)
    await expect(page.locator('.series-page')).toContainText('阅读系列')
  })

  test('mobile drawer links into categories and series', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'Mobile drawer navigation only')
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })

    const drawer = page.locator('#App-Mobile-Profile')
    await page.locator('[data-dia="menu"]').evaluate((element) => (element as HTMLElement).click())
    await expect(drawer).toHaveCSS('opacity', '1')

    const drawerNav = drawer.locator('ul').last()
    await expect(drawerNav.getByText('分类', { exact: true })).toBeVisible()
    await expect(drawerNav.getByText('系列', { exact: true })).toBeVisible()

    await drawerNav.getByText('系列', { exact: true }).click()
    await expect(page).toHaveURL(/\/series$/)
    await expect(page.locator('.series-page')).toContainText('阅读系列')
  })
})


test.describe('reading hub', () => {
  test.describe.configure({ timeout: 90_000 })

  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      document.cookie = 'locale=cn; path=/'
      sessionStorage.setItem('token', 'e2e-reading-token')
    })
  })

  test('records visited articles and lists them in the reading hub', async ({ page }) => {
    await mockArticleReading(page)
    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('heading', { name: '第一节' })).toBeVisible()

    await page.goto('/studio/library/reading', { waitUntil: 'domcontentloaded' })
    const recent = page.getByTestId('reading-recent')
    await expect(recent).toBeVisible()
    await expect(recent).toContainText('系列第2篇')
    await expect(recent.getByRole('link', { name: /系列第2篇/ })).toHaveAttribute('href', '/articles/8')
  })

  test('offers a continue card for the series in progress', async ({ page }) => {
    await mockArticleReading(page)
    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('heading', { name: '第一节' })).toBeVisible()

    await page.goto('/studio/library/reading', { waitUntil: 'domcontentloaded' })
    const card = page.getByTestId('reading-continue')
    await expect(card).toBeVisible()
    await expect(card).toContainText('阅读系列')
    await expect(card).toContainText('已读 0/3')
    await expect(card.getByRole('link')).toHaveAttribute('href', '/articles/8')
  })

  test('clears the saved reading history', async ({ page }) => {
    await mockArticleReading(page)
    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('heading', { name: '第一节' })).toBeVisible()

    await page.goto('/studio/library/reading', { waitUntil: 'domcontentloaded' })
    await expect(page.getByTestId('reading-recent')).toBeVisible()

    await page.getByTestId('reading-clear').click()
    await page.getByTestId('reading-clear-confirm').click()

    await expect(page.getByTestId('reading-recent')).toHaveCount(0)
    await expect(page.getByTestId('reading-cleared')).toBeVisible()
    await expect.poll(() => page.evaluate(() => localStorage.getItem('stellar-beacon.reader.history.v1'))).toBeNull()
  })

  test('ignores malformed reading history', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('stellar-beacon.reader.history.v1', '{broken'))
    await mockArticleReading(page)
    await page.goto('/studio/library/reading', { waitUntil: 'domcontentloaded' })
    await expect(page.getByTestId('reading-empty')).toBeVisible()
  })

  test('opens the private reading hub from the header', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })

    await page.locator('[data-dia="reading"]').click()
    await expect(page).toHaveURL(/\/studio\/library\/reading$/)
    await expect(page.locator('.reading-page')).toBeVisible()
  })
})

test.describe('front-end experience regressions', () => {
  test.describe.configure({ timeout: 90_000 })

  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      document.cookie = 'locale=cn; path=/'
      sessionStorage.setItem('token', 'e2e-reading-token')
    })
  })

  test('renders string-array article tags with their names', async ({ page }) => {
    await mockArticleReading(page)
    await page.goto('/articles/8', { waitUntil: 'domcontentloaded' })
    const labels = page.locator('.post-labels')
    await expect(labels).toContainText('测试标签')
    await expect(labels).toContainText('系列')
    await expect(labels.locator('li', { hasText: '# 测试标签' })).toHaveCount(1)
  })

  test('keeps a single h1 on the home page', async ({ page }) => {
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await expect(page.locator('.plaza-hero')).toBeVisible()
    await expect(page.locator('h1')).toHaveCount(1)
  })

  test('retries a rate-limited content request once', async ({ page }) => {
    let attempts = 0
    await page.route('**/api/v1/public/feed?**', async (route) => {
      attempts += 1
      if (attempts === 1) {
        await route.fulfill({
          status: 429,
          contentType: 'application/json',
          headers: { 'Retry-After': '1' },
          body: JSON.stringify({ code: 'RATE_LIMITED', message: '请求过于频繁', data: null })
        })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 'OK',
          message: '操作成功',
          data: {
            items: [articleFixture(31, '限流重试后的文章')],
            total: 1,
            page: 1,
            pageSize: 12
          }
        })
      })
    })
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('link', { name: '限流重试后的文章' })).toBeVisible()
    expect(attempts).toBeGreaterThan(1)
  })

  test('keeps the mobile drawer visually hidden until opened', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'mobile drawer only')
    await mockContentDiscovery(page)
    await page.goto('/', { waitUntil: 'domcontentloaded' })

    const sidebar = page.locator('.App-Mobile-sidebar')
    await expect(sidebar).toHaveCSS('visibility', 'hidden')

    await page.getByTestId('public-feed').getByRole('link', { name: '公共空间的第一篇文章' }).click()
    await expect(page).toHaveURL(/\/articles\/101$/)
    await expect(sidebar).toHaveCSS('visibility', 'hidden')

    await page.locator('[data-dia="menu"]').evaluate((element) => (element as HTMLElement).click())
    await expect(sidebar).toHaveCSS('visibility', 'visible')
  })
})

async function mockStudioApi(page: Page, options: { draft?: unknown; profile?: any } = {}) {
  const saved = {
    article: null as any,
    talk: null as any,
    series: null as any,
    profile: {
      handle: 'test-author',
      nickname: '测试作者',
      avatar: '',
      intro: '',
      website: '',
      ...(options.profile || {})
    }
  }
  await page.addInitScript((draft) => {
    document.cookie = 'locale=cn; path=/'
    sessionStorage.setItem('token', 'e2e-studio-token')
    sessionStorage.setItem('userStore', JSON.stringify({
      userInfo: { userInfoId: 7, id: 7, nickname: '测试作者', handle: 'test-author' },
      token: 'e2e-studio-token'
    }))
    if (draft) localStorage.setItem('stellar-beacon:studio-draft:v1:7:article:new', JSON.stringify(draft))
  }, options.draft || null)

  await page.route('**/api/v1/auth/me/avatar', async (route) => {
    saved.profile.avatar = 'https://cdn.example.test/avatar.png'
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 'OK', message: '操作成功', data: saved.profile.avatar })
    })
  })

  await page.route('**/api/v1/studio/**', async (route) => {
    const url = new URL(route.request().url())
    const method = route.request().method()
    const respond = (data: unknown) => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 'OK', message: '操作成功', data })
    })
    if (url.pathname === '/api/v1/studio/dashboard' && method === 'GET') {
      await respond({ articleCount: 3, draftCount: 1, privateCount: 1, talkCount: 2, seriesCount: 1, favoriteCount: 4 })
      return
    }
    if (url.pathname === '/api/v1/studio/profile' && method === 'GET') {
      await respond(saved.profile)
      return
    }
    if (url.pathname === '/api/v1/studio/profile' && method === 'PUT') {
      saved.profile = { ...saved.profile, ...route.request().postDataJSON() }
      await respond(saved.profile)
      return
    }
    if (url.pathname === '/api/v1/studio/uploads' && method === 'POST') {
      await respond('https://cdn.example.test/studio-upload.png')
      return
    }
    if (url.pathname === '/api/v1/studio/articles' && method === 'GET') {
      await respond({ items: [{ id: 9, articleTitle: '我的私有草稿', status: 3, moderationStatus: 'visible', createTime: '2026-09-18T10:00:00+08:00' }], total: 1, page: 1, pageSize: 12 })
      return
    }
    if (url.pathname === '/api/v1/studio/articles/99' && method === 'GET') {
      await respond({
        id: 99,
        articleTitle: saved.article?.articleTitle || '已保存文章',
        articleContent: saved.article?.articleContent || '',
        articleContentHtml: saved.article?.articleContentHtml || '',
        articleCover: '',
        categoryId: 0,
        tagNames: [],
        seriesId: 0,
        seriesOrder: 0,
        scheduledAt: '',
        status: 3,
        type: 1,
        password: '',
        originalUrl: ''
      })
      return
    }
    if (url.pathname === '/api/v1/studio/articles' && method === 'POST') {
      saved.article = route.request().postDataJSON()
      await respond({ id: 99 })
      return
    }
    if (url.pathname === '/api/v1/studio/talks' && method === 'POST') {
      saved.talk = route.request().postDataJSON()
      await respond({ id: 88 })
      return
    }
    if (url.pathname === '/api/v1/studio/series' && method === 'GET') {
      await respond({ items: [], total: 0, page: 1, pageSize: 100 })
      return
    }
    if (url.pathname === '/api/v1/studio/series' && method === 'POST') {
      saved.series = route.request().postDataJSON()
      await respond({ id: 77 })
      return
    }
    if (url.pathname === '/api/v1/studio/categories' || url.pathname === '/api/v1/studio/tags') {
      await respond([])
      return
    }
    await respond(null)
  })
  return saved
}

test.describe('studio workspace', () => {
  test('redirects anonymous visitors through login', async ({ page }) => {
    await page.goto('/studio/dashboard', { waitUntil: 'domcontentloaded' })
    await expect(page).toHaveURL(/login=1/)
  })

  test('shows non-blocking public identity guidance on the studio dashboard', async ({ page }) => {
    await mockStudioApi(page)
    await page.goto('/studio/dashboard', { waitUntil: 'domcontentloaded' })

    await expect(page.getByRole('heading', { name: '公开主页资料' })).toBeVisible()
    await expect(page.locator('.studio-profile-progress small')).toContainText('还缺：头像、简介')
    await expect(page.getByRole('link', { name: '完善公开资料 →' })).toBeVisible()
    await expect(page.getByRole('link', { name: '写一篇文章 →' })).toBeVisible()
  })

  test('saves public identity and warns before changing the handle', async ({ page }) => {
    const saved = await mockStudioApi(page)
    await page.goto('/studio/profile', { waitUntil: 'domcontentloaded' })

    await expect(page.getByPlaceholder('your-handle')).toHaveValue('test-author')
    await page.getByPlaceholder('your-handle').fill('new-author')
    await page.getByPlaceholder('作者昵称').fill('新作者')
    await page.getByPlaceholder('介绍你的关注领域、写作方向或正在做的事。').fill('关注系统设计与长期写作。')
    await page.getByPlaceholder('https://example.com').fill('https://example.com')
    await page.getByRole('button', { name: '保存公开资料' }).click()

    await expect(page.getByText('确认更换公开地址')).toBeVisible()
    await page.getByRole('button', { name: '确认更换' }).click()

    await expect.poll(() => saved.profile?.handle).toBe('new-author')
    expect(saved.profile?.nickname).toBe('新作者')
    expect(saved.profile?.intro).toBe('关注系统设计与长期写作。')
    expect(saved.profile?.website).toBe('https://example.com')
    await expect(page.locator('.studio-public-card h2')).toHaveText('新作者')
    await expect(page.locator('.studio-profile-progress strong')).toHaveText('3/4')
    await expect.poll(() => page.evaluate(() => JSON.parse(sessionStorage.getItem('userStore') || '{}')?.userInfo?.handle)).toBe('new-author')
  })

  test('uploads a cropped avatar from the public profile editor', async ({ page }) => {
    const saved = await mockStudioApi(page)
    await page.goto('/studio/profile', { waitUntil: 'domcontentloaded' })

    await page.locator('.avatar-cropper-img-input').setInputFiles({
      name: 'avatar.png',
      mimeType: 'image/png',
      buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64')
    })
    await expect(page.locator('.avatar-cropper-overlay')).toBeVisible()
    await page.getByRole('button', { name: '上传头像' }).click()

    await expect.poll(() => saved.profile?.avatar).toBe('https://cdn.example.test/avatar.png')
    await expect(page.locator('.studio-public-card img')).toHaveAttribute('src', 'https://cdn.example.test/avatar.png')
  })


  test('creates a draft in the dedicated article editor', async ({ page }) => {
    const saved = await mockStudioApi(page)
    await page.goto('/studio/articles', { waitUntil: 'domcontentloaded' })
    await expect(page.getByText('我的私有草稿')).toBeVisible()

    await page.getByRole('button', { name: '新建文章 +' }).click()
    await expect(page).toHaveURL(/\/studio\/articles\/new$/)
    await expect(page.locator('.studio-editor-page')).toBeVisible()
    await page.getByPlaceholder('写下一个清晰的标题').fill('新建的私有文章')
    await expect(page.locator('.w-e-toolbar')).toBeVisible()
    await page.getByRole('button', { name: 'Markdown / HTML' }).click()
    await page.locator('.studio-field--source textarea').fill('# 这是正文。\n<script>alert(1)</script>')
    await page.getByRole('button', { name: '预览' }).click()
    await expect(page.locator('.studio-article-preview h1')).toContainText('这是正文')
    await expect(page.locator('.studio-article-preview script')).toHaveCount(0)
    await page.getByRole('button', { name: '编辑' }).click()
    await page.getByRole('button', { name: '保存内容' }).click()

    await expect.poll(() => saved.article?.articleTitle).toBe('新建的私有文章')
    expect(saved.article?.articleContent).toContain('这是正文')
    expect(saved.article?.articleContentHtml).toBe('')
    expect(saved.article?.visibility).toBe('draft')
    await expect(page).toHaveURL(/\/studio\/articles\/99\/edit$/)
    await page.reload({ waitUntil: 'domcontentloaded' })
    await expect(page.getByPlaceholder('写下一个清晰的标题')).toHaveValue('新建的私有文章')
  })

  test('restores a local article draft before editing', async ({ page }) => {
    await mockStudioApi(page, {
      draft: {
        version: 1,
        savedAt: '2026-09-20T12:00:00+08:00',
        data: {
          id: 0,
          articleTitle: '本地恢复草稿',
          articleContent: '恢复后的正文',
          articleContentHtml: '',
          articleCover: '',
          categoryId: 0,
          tagIds: [],
          seriesId: 0,
          seriesOrder: 0,
          scheduledAt: '',
          visibility: 'draft',
          type: 1,
          password: '',
          originalUrl: ''
        }
      }
    })

    await page.goto('/studio/articles/new', { waitUntil: 'domcontentloaded' })
    await expect(page.getByText('恢复本地草稿')).toBeVisible()
    await page.getByRole('button', { name: '恢复草稿' }).click()
    await expect(page.getByPlaceholder('写下一个清晰的标题')).toHaveValue('本地恢复草稿')
    await expect(page.locator('.studio-field--source textarea')).toHaveValue('恢复后的正文')
  })

  test('uploads ordered talk images and serializes them as JSON', async ({ page }) => {
    const saved = await mockStudioApi(page)
    await page.goto('/studio/talks/new', { waitUntil: 'domcontentloaded' })
    await page.getByPlaceholder('记录此刻的想法…').fill('带图片的随想')
    await page.locator('input[type="file"]').setInputFiles({
      name: 'talk.png',
      mimeType: 'image/png',
      buffer: Buffer.from('image')
    })
    await expect(page.locator('.studio-talk-images__grid img')).toHaveCount(1)
    await page.getByRole('button', { name: '保存内容' }).click()

    await expect.poll(() => saved.talk?.content).toBe('带图片的随想')
    expect(JSON.parse(saved.talk.images)).toEqual(['https://cdn.example.test/studio-upload.png'])
  })

  test('uploads a series cover from the dedicated editor', async ({ page }) => {
    const saved = await mockStudioApi(page)
    await page.goto('/studio/series/new', { waitUntil: 'domcontentloaded' })
    await page.getByPlaceholder('给长期主题一个名字').fill('测试系列')
    await page.getByPlaceholder('说明这个系列关注什么。').fill('系列简介')
    await page.locator('input[type="file"]').setInputFiles({
      name: 'series.png',
      mimeType: 'image/png',
      buffer: Buffer.from('image')
    })
    await expect(page.locator('.studio-cover-field img')).toBeVisible()
    await page.getByRole('button', { name: '保存内容' }).click()

    await expect.poll(() => saved.series?.seriesName).toBe('测试系列')
    expect(saved.series?.cover).toBe('https://cdn.example.test/studio-upload.png')
    expect(saved.series?.visibility).toBe('draft')
  })
})
