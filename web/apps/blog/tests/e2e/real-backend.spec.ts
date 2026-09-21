import { expect, test, type APIRequestContext, type APIResponse, type Browser, type BrowserContext, type Page } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const blogBaseURL = process.env.BLOG_BASE_URL || ''
const adminEmail = process.env.E2E_ADMIN_EMAIL || ''
const adminPassword = process.env.E2E_ADMIN_PASSWORD || ''
const userEmail = process.env.E2E_USER_EMAIL || ''
const userPassword = process.env.E2E_USER_PASSWORD || ''
const mailpitBaseURL = 'http://127.0.0.1:18025'

interface LoginSession {
  token: string
  userInfo: Record<string, unknown>
}

interface BrowserSession {
  context: BrowserContext
  page: Page
}

test.describe('blog real backend main chain @integration', () => {
  test.skip(
    !realIntegration || !blogBaseURL || !adminEmail || !adminPassword || !userEmail || !userPassword,
    'set E2E_REAL_INTEGRATION=1, BLOG_BASE_URL and isolated E2E credentials to run the real backend suite'
  )
  test.setTimeout(240_000)

  let runID = ''
  let fixtureArticleID = 0
  let fixtureAuthorID = 0
  let fixtureRootCommentID = 0
  let createdTalkID = 0
  let admin: LoginSession
  let user: LoginSession
  let authorSession: BrowserSession
  let readerSession: BrowserSession
  const createdCommentIDs: number[] = []

  test.beforeAll(async ({ request }, testInfo) => {
    if (testInfo.project.name !== 'desktop') return
    testInfo.setTimeout(240_000)
    runID = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
    admin = await loginByAPI(request, adminEmail, adminPassword)
    user = await loginByAPI(request, userEmail, userPassword)
    fixtureArticleID = await findFixtureArticleID(request)
    fixtureAuthorID = Number((await getAPIData(request, '/api/v1/public/authors/e2e-admin')).id)
    fixtureRootCommentID = await findRootCommentID(request, fixtureArticleID)
  })

  test.afterAll(async ({ request }) => {
    await authorSession?.context.close()
    await readerSession?.context.close()
    if (!admin?.token || !user?.token) return

    await putAPI(request, '/api/v1/auth/me/notification-preferences', admin.token, { notifyInteraction: 1 }).catch(() => undefined)
    const comments = await findCommentIDsByRun(request, admin.token, runID).catch(() => [])
    const commentIDs = Array.from(new Set([...createdCommentIDs, ...comments]))
    if (commentIDs.length) {
      await deleteAPI(request, '/api/v1/admin/comments', admin.token, commentIDs).catch(() => undefined)
    }
    if (fixtureArticleID > 0) {
      await putAPI(request, '/api/v1/auth/me/reactions', user.token, { articleId: fixtureArticleID, reaction: 'like', active: false }).catch(() => undefined)
      await putAPI(request, '/api/v1/auth/me/reactions', user.token, { articleId: fixtureArticleID, reaction: 'favorite', active: false }).catch(() => undefined)
    }
    if (fixtureAuthorID > 0) {
      await deleteAPI(request, `/api/v1/auth/me/following/${fixtureAuthorID}`, user.token).catch(() => undefined)
    }
    if (createdTalkID > 0) {
      await deleteAPI(request, '/api/v1/studio/talks', admin.token, [createdTalkID]).catch(() => undefined)
    }
  })

  test.beforeEach(async ({}, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'real backend acceptance runs once on desktop Chrome')
  })

  test('public routes load from the real API', async ({ page }) => {
    const routes = [
      '/',
      '/search?q=integration',
      `/articles/${fixtureArticleID}`,
      '/u/e2e-admin',
      '/talks',
      '/series'
    ]
    for (const route of routes) {
      const response = await page.goto(route, { waitUntil: 'domcontentloaded' })
      expect(response?.status(), route).toBe(200)
      await expect(page.locator('#App-Container'), route).toBeVisible()
      await expect(page.locator('body'), route).not.toContainText('页面不存在')
    }
  })

  test('publishes content and completes interaction notification flows', async ({ request, browser }) => {
    authorSession = await createBrowserSession(browser, admin)
    readerSession = await createBrowserSession(browser, user)
    const talkText = `integration e2e talk ${runID}`
    const rootComment = `integration e2e root ${runID}`
    const replyComment = `integration e2e reply ${runID}`
    const optedOutComment = `integration e2e opted-out ${runID}`
    const restoredComment = `integration e2e restored ${runID}`

    await readerSession.page.goto('/u/e2e-admin', { waitUntil: 'domcontentloaded' })
    await readerSession.page.getByRole('button', { name: '关注', exact: true }).click()
    await expect(readerSession.page.getByRole('button', { name: '已关注', exact: true })).toBeVisible()

    await authorSession.page.goto('/studio/talks/new', { waitUntil: 'domcontentloaded' })
    await authorSession.page.getByPlaceholder('记录此刻的想法…').fill(talkText)
    await authorSession.page.getByRole('button', { name: '保存内容', exact: true }).click()
    await authorSession.page.waitForURL(/\/studio\/talks\/\d+\/edit$/)
    createdTalkID = Number(new URL(authorSession.page.url()).pathname.match(/\/studio\/talks\/(\d+)\/edit$/)?.[1] || 0)
    expect(createdTalkID).toBeGreaterThan(0)

    await readerSession.page.goto('/notifications', { waitUntil: 'domcontentloaded' })
    await expect(readerSession.page.getByText(talkText).first()).toBeVisible()
    await readerSession.page.getByRole('button', { name: '发布', exact: true }).click()
    await expect(readerSession.page.getByText(talkText).first()).toBeVisible()
    await expect(readerSession.page.getByText(rootComment)).toHaveCount(0)

    await readerSession.page.goto(`/articles/${fixtureArticleID}`, { waitUntil: 'domcontentloaded' })
    await readerSession.page.getByTestId('article-like').click()
    await expect(readerSession.page.getByTestId('article-like')).toHaveAttribute('aria-pressed', 'true')
    await readerSession.page.getByTestId('article-favorite').click()
    await expect(readerSession.page.getByTestId('article-favorite')).toHaveAttribute('aria-pressed', 'true')

    await postRootComment(readerSession.page, rootComment)
    const fixtureRoot = readerSession.page.locator(`#comment-${fixtureRootCommentID}`)
    await expect(fixtureRoot).toBeVisible()
    await fixtureRoot.locator('.reply-button').click()
    const replyInput = fixtureRoot.locator('textarea:visible')
    await replyInput.fill(replyComment)
    await fixtureRoot.getByRole('button', { name: 'Reply', exact: true }).click()

    const rootID = await waitForPendingComment(request, admin.token, rootComment)
    const replyID = await waitForPendingComment(request, admin.token, replyComment)
    createdCommentIDs.push(rootID, replyID)
    await approveComments(request, admin.token, [rootID, replyID])

    await authorSession.page.goto('/notifications', { waitUntil: 'domcontentloaded' })
    await expect(authorSession.page.getByText(replyComment).first()).toBeVisible()
    await expect(authorSession.page.getByText(rootComment).first()).toBeVisible()
    await authorSession.page.getByRole('button', { name: '赞与收藏', exact: true }).click()
    await expect(authorSession.page.getByText('点赞', { exact: true }).first()).toBeVisible()
    await expect(authorSession.page.getByText('收藏', { exact: true }).first()).toBeVisible()

    await authorSession.page.getByRole('button', { name: '评论与回复', exact: true }).click()
    const commentLink = authorSession.page.locator(`a[href*="comment=${rootID}#comment-${rootID}"]`)
    await expect(commentLink).toBeVisible()
    await commentLink.click()
    await expect(authorSession.page).toHaveURL(new RegExp(`/articles/${fixtureArticleID}\\?comment=${rootID}#comment-${rootID}$`))
    await expect(authorSession.page.locator(`#comment-${rootID}`)).toHaveClass(/comment-focus/)

    await waitForMailpitMessage(request, rootComment)

    await setInteractionPreference(request, admin.token, false)
    await postRootComment(readerSession.page, optedOutComment)
    const optedOutID = await waitForPendingComment(request, admin.token, optedOutComment)
    createdCommentIDs.push(optedOutID)
    await approveComments(request, admin.token, [optedOutID])

    await authorSession.page.goto('/notifications', { waitUntil: 'domcontentloaded' })
    await expect(authorSession.page.getByText(optedOutComment)).toHaveCount(0)

    await setInteractionPreference(request, admin.token, true)
    await postRootComment(readerSession.page, restoredComment)
    const restoredID = await waitForPendingComment(request, admin.token, restoredComment)
    createdCommentIDs.push(restoredID)
    await approveComments(request, admin.token, [restoredID])
    await authorSession.page.reload({ waitUntil: 'domcontentloaded' })
    await expect(authorSession.page.getByText(restoredComment).first()).toBeVisible()

    const unread = await getAPIData(request, '/api/v1/auth/me/notifications/unread-count', admin.token)
    expect(Number(unread.count || 0)).toBe(0)
  })
})

async function loginByAPI(request: APIRequestContext, email: string, password: string): Promise<LoginSession> {
  const response = await request.post('/api/v1/auth/login', { form: { username: email, password } })
  const payload = await response.json()
  expect(response.status(), `login ${email}: ${JSON.stringify(payload)}`).toBe(200)
  expect(Boolean(payload.flag || payload.code === 'OK'), `login ${email}: ${JSON.stringify(payload)}`).toBe(true)
  return { token: String(payload.data?.token || ''), userInfo: payload.data || {} }
}

async function createBrowserSession(browser: Browser, session: LoginSession): Promise<BrowserSession> {
  const context = await browser.newContext({ baseURL: blogBaseURL, locale: 'zh-CN' })
  const page = await context.newPage()
  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await page.evaluate(({ token, userInfo }) => {
    sessionStorage.setItem('token', token)
    sessionStorage.setItem('userStore', JSON.stringify({
      userVisible: false,
      userInfo,
      token,
      accessArticles: [],
      tab: 0,
      page: 1
    }))
  }, session)
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(page.locator('[data-dia="notifications"]')).toBeVisible()
  return { context, page }
}

async function getAPIData(request: APIRequestContext, path: string, token = ''): Promise<any> {
  const response = await request.get(path, token ? { headers: authorization(token) } : undefined)
  const body = await response.text()
  expect(response.status(), `${path}: ${body}`).toBe(200)
  const payload = JSON.parse(body)
  expect(Boolean(payload.flag || payload.code === 'OK'), `${path}: ${body}`).toBe(true)
  return payload.data
}

async function putAPI(request: APIRequestContext, path: string, token: string, data: unknown): Promise<any> {
  const response = await request.put(path, { headers: authorization(token), data })
  return assertMutation(response, `PUT ${path}`)
}

async function deleteAPI(request: APIRequestContext, path: string, token: string, data?: unknown): Promise<any> {
  const response = await request.delete(path, { headers: authorization(token), data })
  return assertMutation(response, `DELETE ${path}`)
}

async function assertMutation(response: APIResponse, label: string): Promise<any> {
  const body = await response.text()
  expect(response.status(), `${label}: ${body}`).toBe(200)
  const payload = JSON.parse(body)
  expect(Boolean(payload.flag || payload.code === 'OK'), `${label}: ${body}`).toBe(true)
  return payload.data
}

function authorization(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` }
}

async function findFixtureArticleID(request: APIRequestContext): Promise<number> {
  const data = await getAPIData(request, '/api/v1/public/authors/e2e-admin/articles?current=1&size=100')
  const item = itemsOf(data).find((entry: any) => entry.articleTitle === 'Integration notification fixture')
  expect(item, 'seeded integration article must exist').toBeTruthy()
  return Number(item.id)
}

async function findRootCommentID(request: APIRequestContext, articleID: number): Promise<number> {
  const data = await getAPIData(request, `/api/v1/public/comments?type=1&topicId=${articleID}&current=1&size=100`)
  const item = itemsOf(data).find((entry: any) => entry.commentContent === 'Integration author root comment')
  expect(item, 'seeded approved author comment must exist').toBeTruthy()
  return Number(item.id)
}

async function postRootComment(page: Page, text: string): Promise<void> {
  const textarea = page.locator('#comment-content')
  await expect(textarea).toBeVisible()
  await textarea.fill(text)
  await page.getByRole('button', { name: 'Add Comment', exact: true }).click()
  await expect(page.getByText('评论成功,正在审核中').last()).toBeVisible()
}

async function waitForPendingComment(request: APIRequestContext, token: string, text: string): Promise<number> {
  let found = 0
  await expect.poll(async () => {
    const data = await getAPIData(request, `/api/v1/admin/comments?current=1&size=100&keywords=${encodeURIComponent(text)}`, token)
    const item = itemsOf(data).find((entry: any) => entry.commentContent === text && Number(entry.isReview) === 0)
    found = Number(item?.id || 0)
    return found
  }, { timeout: 15_000 }).toBeGreaterThan(0)
  return found
}

async function approveComments(request: APIRequestContext, token: string, ids: number[]): Promise<void> {
  await putAPI(request, '/api/v1/admin/comments/review', token, { ids, isReview: 1 })
}

async function findCommentIDsByRun(request: APIRequestContext, token: string, runID: string): Promise<number[]> {
  const data = await getAPIData(request, `/api/v1/admin/comments?current=1&size=100&keywords=${encodeURIComponent(runID)}`, token)
  return itemsOf(data).map((entry: any) => Number(entry.id)).filter((id: number) => id > 0)
}

async function setInteractionPreference(request: APIRequestContext, token: string, enabled: boolean): Promise<void> {
  const payload = await putAPI(request, '/api/v1/auth/me/notification-preferences', token, { notifyInteraction: enabled ? 1 : 0 })
  expect(Number(payload.notifyInteraction)).toBe(enabled ? 1 : 0)
}

async function waitForMailpitMessage(request: APIRequestContext, text: string): Promise<void> {
  await expect.poll(async () => {
    const response = await request.get(`${mailpitBaseURL}/api/v1/search?query=${encodeURIComponent(text)}`)
    if (!response.ok()) return false
    const body = await response.text()
    return body.includes(text)
  }, { timeout: 20_000, intervals: [500, 1_000, 2_000] }).toBe(true)
}

function itemsOf(data: any): any[] {
  if (Array.isArray(data?.items)) return data.items
  if (Array.isArray(data?.records)) return data.records
  return []
}
