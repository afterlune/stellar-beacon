import { expect, test, type Page } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const allowLogin = process.env.E2E_ADMIN_ALLOW_LOGIN === '1'

test('article admin use cases keep their API and page-size contracts @integration @article-admin', async ({ page, request }) => {
  test.setTimeout(90_000)
  test.skip(
    !realIntegration || !allowLogin || !process.env.E2E_ADMIN_EMAIL || !process.env.E2E_ADMIN_PASSWORD,
    'requires the isolated integration stack and its admin credentials'
  )

  page.setDefaultTimeout(10_000)
  const suffix = String(Date.now())
  const articleTitle = `e2e-admin-${suffix}`
  const importTitle = `e2e-import-${suffix}`
  let articleID = 0
  let importedArticleID = 0
  const token = await login(page)
  const headers = { Authorization: `Bearer ${token}` }

  try {
    const createResponse = await request.post('/api/v1/admin/articles', {
      headers,
      data: {
        articleTitle,
        articleContent: `isolated article ${suffix}`,
        articleContentHtml: '',
        categoryName: '',
        tagNames: [],
        status: 3,
        type: 1
      }
    })
    expect((await createResponse.json()).code).toBe('OK')

    const listResponse = await request.get(`/api/v1/admin/articles?keywords=${encodeURIComponent(articleTitle)}`, { headers })
    const listPayload = await listResponse.json() as PageResult
    expect(listResponse.status()).toBe(200)
    expect(listPayload.code).toBe('OK')
    expect(listPayload.data.page).toBe(1)
    expect(listPayload.data.pageSize).toBe(12)
    articleID = listPayload.data.items.find((item) => item.articleTitle === articleTitle)?.id || 0
    expect(articleID).toBeGreaterThan(0)

    const detailResponse = await request.get(`/api/v1/admin/articles/${articleID}`, { headers })
    const detailPayload = await detailResponse.json() as Result<ArticleDetail>
    expect(detailPayload.code).toBe('OK')
    expect(detailPayload.data.articleTitle).toBe(articleTitle)
    expect(detailPayload.data.articleContent).toBe(`isolated article ${suffix}`)

    const featuredResponse = await request.put('/api/v1/admin/articles/featured', {
      headers,
      data: { id: articleID, isTop: 0, isFeatured: 1 }
    })
    const featuredPayload = await featuredResponse.json() as Result<unknown>
    expect(featuredPayload.code).toBe('OPERATION_FAILED')
    expect(featuredPayload.message).toBe('只有公开且审核可见的文章可以置顶或推荐')

    const trashResponse = await request.put('/api/v1/admin/articles/trash', {
      headers,
      data: { ids: [articleID], isDelete: 1 }
    })
    expect((await trashResponse.json()).code).toBe('OK')
    const trashListResponse = await request.get(`/api/v1/admin/articles?keywords=${encodeURIComponent(articleTitle)}&isDelete=1`, { headers })
    const trashList = await trashListResponse.json() as PageResult
    expect(trashList.data.items.some((item) => item.id === articleID)).toBe(true)

    const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64')
    const imageResponse = await request.post('/api/v1/admin/articles/images', {
      headers,
      multipart: { file: { name: `article-${suffix}.png`, mimeType: 'image/png', buffer: png } }
    })
    const imagePayload = await imageResponse.json() as Result<string>
    expect(imagePayload.code).toBe('OK')
    expect(imagePayload.data).toContain('.png')

    const importResponse = await request.post('/api/v1/admin/articles/import', {
      headers,
      multipart: {
        file: {
          name: `${importTitle}.md`,
          mimeType: 'text/markdown',
          buffer: Buffer.from(`isolated import ${suffix}`)
        }
      }
    })
    expect((await importResponse.json()).code).toBe('OK')
    const importedListResponse = await request.get(`/api/v1/admin/articles?keywords=${encodeURIComponent(importTitle)}`, { headers })
    const importedList = await importedListResponse.json() as PageResult
    importedArticleID = importedList.data.items.find((item) => item.articleTitle === importTitle)?.id || 0
    expect(importedArticleID).toBeGreaterThan(0)

    const exportResponse = await request.post('/api/v1/admin/articles/export', {
      headers,
      data: [articleID, importedArticleID]
    })
    const exportPayload = await exportResponse.json() as Result<string[]>
    expect(exportPayload.code).toBe('OK')
    expect(exportPayload.data).toHaveLength(2)
    expect(exportPayload.data.every((url) => url.endsWith('.md'))).toBe(true)

    // A persisted pre-migration value of 10 must be normalized to 12 by the UI.
    await page.addInitScript(() => {
      localStorage.setItem('stellar-beacon.admin.table.article-list', JSON.stringify({ pageSize: 10 }))
    })
    const articleListRequest = page.waitForRequest((candidate) => {
      const url = new URL(candidate.url())
      return url.pathname.endsWith('/admin/articles') && candidate.method() === 'GET'
    })
    await page.goto('/article-list', { waitUntil: 'domcontentloaded' })
    const uiListURL = new URL((await articleListRequest).url())
    expect(uiListURL.searchParams.get('size')).toBe('12')
    await expect(page.locator('.arco-table')).toBeVisible()
  } finally {
    const remainingIDs = new Set([articleID, importedArticleID].filter((id) => id > 0))
    for (const title of [articleTitle, importTitle]) {
      const response = await request.get(`/api/v1/admin/articles?keywords=${encodeURIComponent(title)}`, { headers })
      if (!response.ok()) continue
      const payload = await response.json() as PageResult
      for (const item of payload.data?.items || []) {
        if (item.articleTitle === title && item.id > 0) remainingIDs.add(item.id)
      }
    }
    if (remainingIDs.size > 0) {
      const cleanup = await request.delete('/api/v1/admin/articles/batch-delete', {
        headers,
        data: [...remainingIDs]
      })
      expect((await cleanup.json()).code).toBe('OK')
    }
  }
})

async function login(page: Page): Promise<string> {
  await page.goto('/login', { waitUntil: 'domcontentloaded' })
  await page.getByTestId('login-username').locator('input').fill(process.env.E2E_ADMIN_EMAIL || '')
  await page.getByTestId('login-password').locator('input').fill(process.env.E2E_ADMIN_PASSWORD || '')
  await page.getByTestId('login-submit').click()
  await page.waitForURL((url) => url.pathname === '/')
  await expect(page.locator('.admin-shell')).toBeVisible()
  const token = await page.evaluate(() => sessionStorage.getItem('token') || '')
  expect(token).not.toBe('')
  return token
}

interface Result<T> {
  code: string
  message: string
  data: T
}

interface ArticleListItem {
  id: number
  articleTitle: string
}

interface PageResult extends Result<{
  items: ArticleListItem[]
  total: number
  page: number
  pageSize: number
}> {}

interface ArticleDetail {
  id: number
  articleTitle: string
  articleContent: string
}
