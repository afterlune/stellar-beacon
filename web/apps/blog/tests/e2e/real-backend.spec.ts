import { expect, test, type APIRequestContext, type APIResponse, type Browser, type BrowserContext, type Page } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const blogBaseURL = process.env.BLOG_BASE_URL || ''
const adminEmail = process.env.E2E_ADMIN_EMAIL || ''
const adminPassword = process.env.E2E_ADMIN_PASSWORD || ''
const userEmail = process.env.E2E_USER_EMAIL || ''
const userPassword = process.env.E2E_USER_PASSWORD || ''
const mailpitBaseURL = 'http://127.0.0.1:18025'
let e2eHeaders: Record<string, string> = {}

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
  let createdCollectionID = 0
  let createdSubscriptionCollectionID = 0
  let createdInteractionCollectionID = 0
  let createdGovernanceCollectionID = 0
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
    if (createdCollectionID > 0) {
      await deleteAPI(request, `/api/v1/studio/collections/${createdCollectionID}`, user.token).catch(() => undefined)
    }
    if (createdSubscriptionCollectionID > 0) {
      await deleteAPI(request, `/api/v1/studio/collections/${createdSubscriptionCollectionID}`, admin.token).catch(() => undefined)
    }
    if (createdInteractionCollectionID > 0) {
      await deleteAPI(request, `/api/v1/studio/collections/${createdInteractionCollectionID}`, admin.token).catch(() => undefined)
    }
    if (createdGovernanceCollectionID > 0) {
      await deleteAPI(request, `/api/v1/studio/collections/${createdGovernanceCollectionID}`, admin.token).catch(() => undefined)
    }

  })

  test.beforeEach(async ({}, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'real backend acceptance runs once on desktop Chrome')
    e2eHeaders = { 'X-Real-IP': `198.51.100.${10 + Math.floor(Math.random() * 200)}` }
  })

  test('public routes load from the real API', async ({ page }) => {
    const routes = [
      '/',
      '/search?q=integration',
      `/articles/${fixtureArticleID}`,
      '/u/e2e-admin',
      '/authors',
      '/topics',
      '/collections',
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

  test('persists studio activation state per account', async ({ request }) => {
    const first = await putAPI(request, '/api/v1/studio/activation', admin.token, {
      started: true, collapsed: true, identityComplete: false, contentComplete: false, profileVisited: false, completed: false
    })
    expect(first.startedAt).toBeTruthy()
    expect(first.collapsed).toBe(true)

    const dashboard = await getAPIData(request, '/api/v1/studio/dashboard', admin.token)
    expect(dashboard.activation).toMatchObject({ startedAt: first.startedAt, collapsed: true })

    const completed = await putAPI(request, '/api/v1/studio/activation', admin.token, {
      started: false, collapsed: false, identityComplete: true, contentComplete: true, profileVisited: true, completed: true
    })
    expect(completed.startedAt).toBe(first.startedAt)
    expect(completed.identityCompletedAt).toBeTruthy()
    expect(completed.completedAt).toBeTruthy()
    expect(completed.collapsed).toBe(false)
  })

  test('creates, discovers and moderates a public reading list', async ({ request, browser }) => {
    const title = `integration reading list ${runID}`
    const created = await postAPI(request, '/api/v1/studio/collections', user.token, {
      title,
      description: 'integration collection description',
      visibility: 'public'
    })
    createdCollectionID = Number(created.id)
    expect(createdCollectionID).toBeGreaterThan(0)
    expect(String(created.slug)).toBeTruthy()

    await putAPI(request, `/api/v1/studio/collections/${createdCollectionID}/items/${fixtureArticleID}`, user.token, {
      note: 'integration recommendation note'
    })
    await putAPI(request, `/api/v1/studio/collections/${createdCollectionID}/order`, user.token, {
      articleIds: [fixtureArticleID]
    })

    const publicDetail = await getAPIData(request, `/api/v1/public/collections/${created.slug}`)
    expect(publicDetail.collection.title).toBe(title)
    expect(itemsOf(publicDetail)).toHaveLength(1)
    expect(publicDetail.items[0].note).toBe('integration recommendation note')

    const discovered = await getAPIData(request, '/api/v1/public/collections?sort=latest&current=1&size=100')
    expect(itemsOf(discovered).some((item: any) => item.slug === created.slug)).toBe(true)

    const session = await createBrowserSession(browser, user)
    try {
      await session.page.goto(`/collections/${created.slug}`, { waitUntil: 'domcontentloaded' })
      await expect(session.page.getByRole('heading', { name: title })).toBeVisible()
      await expect(session.page.getByText('integration recommendation note')).toBeVisible()
      await session.page.goto('/collections', { waitUntil: 'domcontentloaded' })
      await expect(session.page.getByText(title).first()).toBeVisible()
    } finally {
      await session.context.close()
    }

    await putAPI(request, `/api/v1/studio/collections/${createdCollectionID}`, user.token, {
      title,
      description: 'integration collection description',
      visibility: 'unlisted'
    })
    const unlistedPage = await getAPIData(request, '/api/v1/public/collections?sort=latest&current=1&size=100')
    expect(itemsOf(unlistedPage).some((item: any) => item.slug === created.slug)).toBe(false)
    await getAPIData(request, `/api/v1/public/collections/${created.slug}`)

    await putAPI(request, `/api/v1/studio/collections/${createdCollectionID}`, user.token, {
      title,
      description: 'integration collection description',
      visibility: 'public'
    })
    await putAPI(request, `/api/v1/admin/content/collection/${createdCollectionID}/moderation`, admin.token, {
      contentType: 'collection',
      id: createdCollectionID,
      hidden: true,
      reason: 'integration moderation exercise'
    })
    const hiddenResponse = await request.get(`/api/v1/public/collections/${created.slug}`)
    const hiddenPayload = await hiddenResponse.json()
    expect(hiddenPayload.code).not.toBe('OK')
    await putAPI(request, `/api/v1/admin/content/collection/${createdCollectionID}/moderation`, admin.token, {
      contentType: 'collection',
      id: createdCollectionID,
      hidden: false,
      reason: ''
    })
  })

  test('subscribes to collection updates and receives in-app notifications', async ({ request, browser }) => {
    const title = `integration subscribed list ${runID}`
    const readerFixtureID = await findReaderFixtureArticleID(request)
    const created = await postAPI(request, '/api/v1/studio/collections', admin.token, {
      title,
      description: 'integration subscribed collection',
      visibility: 'public'
    })
    createdSubscriptionCollectionID = Number(created.id)
    expect(createdSubscriptionCollectionID).toBeGreaterThan(0)

    await putAPI(request, `/api/v1/studio/collections/${createdSubscriptionCollectionID}/items/${fixtureArticleID}`, admin.token, {
      note: 'existing item before subscription'
    })
    await putAPI(request, `/api/v1/auth/me/collection-subscriptions/${createdSubscriptionCollectionID}`, user.token, {})
    const status = await getAPIData(request, `/api/v1/auth/me/collection-subscriptions/${createdSubscriptionCollectionID}`, user.token)
    expect(status.subscribed).toBe(true)
    expect(status.muted).toBe(false)

    await putAPI(request, `/api/v1/studio/collections/${createdSubscriptionCollectionID}/items/${readerFixtureID}`, admin.token, {
      note: 'new item after subscription'
    })
    const feed = await getAPIData(request, '/api/v1/auth/me/collection-feed?current=1&size=20', user.token)
    const update = itemsOf(feed).find((item: any) => Number(item.collectionId) === createdSubscriptionCollectionID)
    expect(update?.articleId).toBe(readerFixtureID)
    expect(update?.slug).toBe(created.slug)

    const notificationPage = await getAPIData(request, '/api/v1/auth/me/notifications?group=collection&current=1&size=20', user.token)
    const notification = itemsOf(notificationPage).find((item: any) => Number(item.contentId) === createdSubscriptionCollectionID)
    expect(notification?.type).toBe('collection_update')
    expect(notification?.articleId).toBe(readerFixtureID)
    expect(notification?.slug).toBe(created.slug)

    const session = await createBrowserSession(browser, user)
    try {
      await session.page.goto('/following', { waitUntil: 'domcontentloaded' })
      await session.page.getByRole('button', { name: '书单', exact: true }).click()
      await expect(session.page.getByText(title).first()).toBeVisible()
      await expect(session.page.getByText('新增：', { exact: false }).first()).toBeVisible()

      await session.page.goto('/notifications', { waitUntil: 'domcontentloaded' })
      await session.page.getByRole('button', { name: '书单更新', exact: true }).click()
      const headline = session.page.getByText(`书单「${title}」新增了`, { exact: false }).first()
      await expect(headline).toBeVisible()
      await headline.click()
      await session.page.waitForURL(new RegExp(`/collections/${created.slug}\\?article=${readerFixtureID}`))
      await expect(session.page.locator(`[data-article-id="${readerFixtureID}"]`)).toHaveClass(/is-highlighted/)
    } finally {
      await session.context.close()
    }

    await deleteAPI(request, `/api/v1/auth/me/collection-subscriptions/${createdSubscriptionCollectionID}`, user.token)
    const unsubscribed = await getAPIData(request, `/api/v1/auth/me/collection-subscriptions/${createdSubscriptionCollectionID}`, user.token)
    expect(unsubscribed.subscribed).toBe(false)
  })

  test('likes and reviews comments on a public reading list', async ({ request, browser }) => {
    await putAPI(request, '/api/v1/auth/me/notification-preferences', admin.token, { notifyInteraction: 1 })
    const title = `integration interaction list ${runID}`
    const commentText = `integration collection comment ${runID}`
    const created = await postAPI(request, '/api/v1/studio/collections', admin.token, {
      title,
      description: 'integration interaction collection',
      visibility: 'public'
    })
    createdInteractionCollectionID = Number(created.id)
    expect(createdInteractionCollectionID).toBeGreaterThan(0)
    await putAPI(request, `/api/v1/studio/collections/${createdInteractionCollectionID}/items/${fixtureArticleID}`, admin.token, {
      note: 'integration interaction item'
    })

    const liked = await putAPI(request, '/api/v1/auth/me/collection-reactions', user.token, {
      collectionId: createdInteractionCollectionID,
      reaction: 'like',
      active: true
    })
    expect(liked.active).toBe(true)
    expect(liked.likeCount).toBe(1)
    const favorited = await putAPI(request, '/api/v1/auth/me/collection-reactions', user.token, {
      collectionId: createdInteractionCollectionID,
      reaction: 'favorite',
      active: true
    })
    expect(favorited.favoriteCount).toBe(1)
    expect(favorited.likeCount).toBe(1)
    const state = await getAPIData(request, `/api/v1/auth/me/collection-reactions/state?collectionId=${createdInteractionCollectionID}`, user.token)
    expect(state.like).toBe(true)
    expect(state.favorite).toBe(true)
    const savedCollections = await getAPIData(request, '/api/v1/auth/me/collection-reactions?reaction=favorite&current=1&size=20', user.token)
    expect(itemsOf(savedCollections).some((item: any) => Number(item.id) === createdInteractionCollectionID)).toBe(true)

    await postAPI(request, '/api/v1/public/comments', user.token, {
      type: 6,
      topicId: String(createdInteractionCollectionID),
      commentContent: commentText
    })
    const commentID = await waitForPendingComment(request, admin.token, commentText)
    createdCommentIDs.push(commentID)

    const pendingDetail = await getAPIData(request, `/api/v1/public/collections/${created.slug}`)
    expect(pendingDetail.collection.likeCount).toBe(1)
    expect(pendingDetail.collection.favoriteCount).toBe(1)
    expect(pendingDetail.collection.commentCount, 'pending comments must not count publicly').toBe(0)

    const reactionPage = await getAPIData(request, '/api/v1/auth/me/notifications?group=reaction&current=1&size=50', admin.token)
    const reactionNotification = itemsOf(reactionPage).find((item: any) => item.type === 'like' && item.contentType === 'collection' && Number(item.contentId) === createdInteractionCollectionID)
    expect(reactionNotification?.type).toBe('like')
    expect(reactionNotification?.slug).toBe(created.slug)
    const favoriteNotification = itemsOf(reactionPage).find((item: any) => item.type === 'favorite' && item.contentType === 'collection' && Number(item.contentId) === createdInteractionCollectionID)
    expect(favoriteNotification?.slug).toBe(created.slug)

    await approveComments(request, admin.token, [commentID])
    const approvedDetail = await getAPIData(request, `/api/v1/public/collections/${created.slug}`)
    expect(approvedDetail.collection.commentCount).toBe(1)
    const commentPage = await getAPIData(request, '/api/v1/auth/me/notifications?group=comment&current=1&size=50', admin.token)
    const commentNotification = itemsOf(commentPage).find((item: any) => item.contentType === 'collection' && Number(item.commentId) === commentID)
    expect(commentNotification?.type).toBe('comment')
    expect(commentNotification?.slug).toBe(created.slug)

    const commentLike = await putAPI(request, '/api/v1/auth/me/comment-reactions', user.token, { commentId: commentID, active: true })
    expect(commentLike.active).toBe(true)
    expect(commentLike.likeCount).toBe(1)

    const session = await createBrowserSession(browser, user)
    try {
      await session.page.goto('/studio/library/favorites?tab=collections', { waitUntil: 'domcontentloaded' })
      await expect(session.page.getByText(title).first()).toBeVisible()
      await session.page.goto(`/collections/${created.slug}`, { waitUntil: 'domcontentloaded' })
      await expect(session.page.getByText(commentText)).toBeVisible()
      const commentLikeButton = session.page.locator(`#comment-${commentID} [data-testid="comment-like"]`)
      await expect(commentLikeButton).toHaveAttribute('aria-pressed', 'true')
      await commentLikeButton.click()
      await expect(commentLikeButton).toHaveAttribute('aria-pressed', 'false')
      const likeButton = session.page.getByRole('button', { name: /已点赞/ })
      await expect(likeButton).toBeVisible()
      await likeButton.click()
      await expect(session.page.getByRole('button', { name: /点赞 · 0/ })).toBeVisible()
      const unliked = await getAPIData(request, `/api/v1/auth/me/collection-reactions/state?collectionId=${createdInteractionCollectionID}`, user.token)
      expect(unliked.like).toBe(false)
      const favoriteButton = session.page.getByRole('button', { name: /已收藏/ })
      await expect(favoriteButton).toBeVisible()
      await favoriteButton.click()
      await expect(session.page.getByRole('button', { name: /收藏 · 0/ })).toBeVisible()
    } finally {
      await session.context.close()
    }
  })

  test('lets a reading-list owner pin and soft-delete comments', async ({ request, browser }) => {
    const title = `integration moderation list ${runID}`
    const keepText = `integration keep comment ${runID}`
    const pinnedText = `integration pinned comment ${runID}`
    const deletedText = `integration deleted comment ${runID}`
    const replyText = `integration moderation reply ${runID}`
    const created = await postAPI(request, '/api/v1/studio/collections', admin.token, { title, description: 'integration moderation collection', visibility: 'public' })
    const collectionID = Number(created.id)
    expect(collectionID).toBeGreaterThan(0)
    await putAPI(request, `/api/v1/studio/collections/${collectionID}/items/${fixtureArticleID}`, admin.token, { note: 'integration moderation item' })

    const keepCommentID = await seedApprovedCollectionComment(request, admin, collectionID, keepText)
    const pinnedCommentID = await seedApprovedCollectionComment(request, admin, collectionID, pinnedText)
    const deletedCommentID = await seedApprovedCollectionComment(request, admin, collectionID, deletedText)
    const replyID = await seedApprovedCollectionReply(request, admin, collectionID, pinnedCommentID, replyText)

    await expect(deleteAPI(request, `/api/v1/studio/collections/${collectionID}/comments/${replyID}`, user.token)).rejects.toThrow()

    const session = await createBrowserSession(browser, admin)
    try {
      await session.page.goto(`/collections/${created.slug}`, { waitUntil: 'domcontentloaded' })
      const pinnedItem = session.page.locator(`#comment-${pinnedCommentID}`)
      await expect(pinnedItem.getByText(pinnedText)).toBeVisible()
      const replyItem = session.page.locator(`#comment-${replyID}`)
      await expect(replyItem.getByText(replyText)).toBeVisible()

      await pinnedItem.getByTestId('comment-pin-action').click()
      await expect(pinnedItem.getByTestId('comment-pinned')).toBeVisible()
      await expect(replyItem.getByTestId('comment-reply-delete-action')).toBeVisible()

      const ordered = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=10`, admin.token)
      expect(Number(itemsOf(ordered)[0]?.id)).toBe(pinnedCommentID)
      expect(Number(itemsOf(ordered)[0]?.isTop)).toBe(1)

      await expect(replyItem.getByTestId('comment-reply-delete-action')).toBeVisible()
      await replyItem.getByTestId('comment-reply-delete-action').click()
      await session.page.getByRole('button', { name: '删除', exact: true }).last().click()
      await expect(replyItem.getByText('已被书单作者删除')).toBeVisible()
      const replyNotifications = await getAPIData(request, '/api/v1/auth/me/notifications?group=comment&current=1&size=50', admin.token)
      expect(itemsOf(replyNotifications).some((item: any) => Number(item.commentId) === replyID)).toBe(false)

      await session.page.locator(`#comment-${deletedCommentID}`).getByTestId('comment-delete-action').click()
      await session.page.getByRole('button', { name: '删除', exact: true }).last().click()
      await expect(session.page.locator(`#comment-${deletedCommentID}`).getByText('已被书单作者删除')).toBeVisible()
      const moderated = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=10`)
      expect(itemsOf(moderated).some((item: any) => Number(item.id) === deletedCommentID)).toBe(false)
      expect(Number(itemsOf(moderated)[0]?.id)).toBe(pinnedCommentID)
      expect(Number(itemsOf(moderated)[0]?.isTop)).toBe(1)
      const detail = await getAPIData(request, `/api/v1/public/collections/${created.slug}`)
      expect(Number(detail.collection.commentCount)).toBe(2)
      expect(Number(itemsOf(ordered).length)).toBe(3)
      expect(itemsOf(ordered).some((item: any) => Number(item.id) === keepCommentID)).toBe(true)
    } finally {
      await session.context.close()
    }
  })

  test('batch moderates reading-list comments from the studio governance panel', async ({ request, browser }) => {
    const title = `integration batch moderation list ${runID}`
    const firstText = `integration batch first ${runID}`
    const secondText = `integration batch second ${runID}`
    const created = await postAPI(request, '/api/v1/studio/collections', admin.token, { title, description: 'integration batch moderation collection', visibility: 'public' })
    const collectionID = Number(created.id)
    expect(collectionID).toBeGreaterThan(0)
    await putAPI(request, `/api/v1/studio/collections/${collectionID}/items/${fixtureArticleID}`, admin.token, { note: 'integration batch item' })

    const firstID = await seedApprovedCollectionComment(request, admin, collectionID, firstText)
    const secondID = await seedApprovedCollectionComment(request, admin, collectionID, secondText)

    const session = await createBrowserSession(browser, admin)
    try {
      await session.page.goto(`/studio/collections/${collectionID}/edit`, { waitUntil: 'domcontentloaded' })
      const list = session.page.locator('.governance-list > article')
      await expect(list).toHaveCount(2)

      await list.filter({ hasText: secondText }).getByRole('checkbox').check()
      await list.filter({ hasText: firstText }).getByRole('checkbox').check()
      await session.page.getByRole('button', { name: '批量置顶', exact: true }).click()
      await expect(session.page.getByTestId('governance-message')).toContainText('成功 2 条')

      const pinned = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=10`, admin.token)
      expect(Number(itemsOf(pinned)[0]?.id)).toBe(secondID)
      expect(Number(itemsOf(pinned)[0]?.isTop)).toBe(1)
      expect(itemsOf(pinned).filter((item: any) => Number(item.isTop) === 1).length).toBe(1)

      await list.filter({ hasText: firstText }).getByRole('checkbox').check()
      await session.page.getByRole('button', { name: '批量删除', exact: true }).click()
      await expect(session.page.getByTestId('governance-message')).toContainText('成功 1 条')

      const remaining = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=10`)
      expect(itemsOf(remaining).some((item: any) => Number(item.id) === firstID)).toBe(false)

      await list.filter({ hasText: firstText }).getByRole('checkbox').check()
      await session.page.getByRole('button', { name: '恢复选中', exact: true }).click()
      await expect(session.page.getByTestId('governance-message')).toContainText('成功 1 条')
    } finally {
      await session.context.close()
    }
  })

  test('closes the comment report and appeal loop across reader, owner and admin views', async ({ request, browser }) => {
    const title = `integration governance loop ${runID}`
    const commentText = `integration reported comment ${runID}`
    const appealReason = `integration appeal reason ${runID}`
    const isolatedHeaders = { 'X-Real-IP': '198.51.100.201' }
    const created = await postAPI(request, '/api/v1/studio/collections', admin.token, {
      title,
      description: 'integration report and appeal collection',
      visibility: 'public'
    }, isolatedHeaders)
    const collectionID = Number(created.id)
    createdGovernanceCollectionID = collectionID
    expect(collectionID).toBeGreaterThan(0)
    await putAPI(request, `/api/v1/studio/collections/${collectionID}/items/${fixtureArticleID}`, admin.token, { note: 'integration governance item' }, isolatedHeaders)

    await postAPI(request, '/api/v1/public/comments', user.token, { type: 6, topicId: String(collectionID), commentContent: commentText }, isolatedHeaders)
    const commentID = await waitForPendingComment(request, admin.token, commentText)
    await approveComments(request, admin.token, [commentID], isolatedHeaders)
    createdCommentIDs.push(commentID)

    await postAPI(request, '/api/v1/auth/me/comment-reports', admin.token, { commentId: commentID, reason: 'spam', detail: `integration report ${runID}` }, isolatedHeaders)

    const ownerSession = await createBrowserSession(browser, admin, isolatedHeaders)
    const readerSessionForGovernance = await createBrowserSession(browser, user, isolatedHeaders)
    try {
      await ownerSession.page.goto(`/studio/collections/${collectionID}/edit`, { waitUntil: 'domcontentloaded' })
      const reportQueue = ownerSession.page.getByTestId('comment-report-queue')
      await expect(reportQueue.getByText(commentText)).toBeVisible()
      await reportQueue.getByRole('button', { name: '隐藏', exact: true }).click()
      await expect(ownerSession.page.getByTestId('governance-queue-message')).toContainText('举报已处理')

      const publicComments = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=20`)
      expect(itemsOf(publicComments).some((item: any) => Number(item.id) === commentID)).toBe(false)
      const authorComments = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=20`, user.token)
      const hiddenComment = itemsOf(authorComments).find((item: any) => Number(item.id) === commentID)
      expect(Number(hiddenComment?.isDelete)).toBe(1)

      await readerSessionForGovernance.page.goto(`/collections/${created.slug}`, { waitUntil: 'domcontentloaded' })
      const hiddenItem = readerSessionForGovernance.page.locator(`#comment-${commentID}`)
      await expect(hiddenItem.getByText('已被书单作者删除')).toBeVisible()
      await hiddenItem.getByTestId('comment-appeal-action').click()
      await hiddenItem.getByTestId('comment-appeal-reason').fill(appealReason)
      await hiddenItem.getByTestId('comment-appeal-submit').click()
      await expect(hiddenItem.getByText('申诉待书单作者处理')).toBeVisible()

      await ownerSession.page.goto(`/studio/collections/${collectionID}/edit`, { waitUntil: 'domcontentloaded' })
      const appealQueue = ownerSession.page.getByTestId('comment-appeal-queue')
      await expect(appealQueue.getByText(appealReason)).toBeVisible()
      await appealQueue.getByRole('button', { name: '驳回', exact: true }).click()
      await expect(ownerSession.page.getByTestId('governance-queue-message')).toContainText('申诉已驳回')

      await readerSessionForGovernance.page.reload({ waitUntil: 'domcontentloaded' })
      const rejectedItem = readerSessionForGovernance.page.locator(`#comment-${commentID}`)
      await expect(rejectedItem.getByText('书单作者已驳回，可升级给管理员')).toBeVisible()
      await rejectedItem.getByTestId('comment-appeal-escalate').click()
      await expect(rejectedItem.getByText('申诉待管理员终审')).toBeVisible()

      const adminAppeals = await getAPIData(request, '/api/v1/admin/comment-appeals?current=1&size=50', admin.token)
      const appeal = itemsOf(adminAppeals).find((item: any) => Number(item.commentId) === commentID)
      expect(Number(appeal?.id)).toBeGreaterThan(0)
      await putAPI(request, `/api/v1/admin/comment-appeals/${Number(appeal.id)}`, admin.token, { decision: 'restore', reason: 'integration appeal upheld' }, isolatedHeaders)

      await readerSessionForGovernance.page.reload({ waitUntil: 'domcontentloaded' })
      const restoredItem = readerSessionForGovernance.page.locator(`#comment-${commentID}`)
      await expect(restoredItem.getByText(commentText)).toBeVisible()
      await expect(restoredItem.getByText('已被书单作者删除')).toHaveCount(0)
      await expect(restoredItem.getByTestId('comment-like')).toBeVisible()

      const restoredComments = await getAPIData(request, `/api/v1/public/comments?type=6&topicId=${collectionID}&current=1&size=20`)
      const restoredComment = itemsOf(restoredComments).find((item: any) => Number(item.id) === commentID)
      expect(Number(restoredComment?.isDelete || 0)).toBe(0)
    } finally {
      await ownerSession.context.close()
      await readerSessionForGovernance.context.close()
    }
  })

  test('personalizes recommendations and persists reader feedback', async ({ request, browser }) => {
    const topicKey = 'integration topic'
    const readerFixtureID = await findReaderFixtureArticleID(request)
    const existing = await getAPIData(request, '/api/v1/auth/me/recommendation-feedback?current=1&size=100', user.token)
    for (const item of itemsOf(existing)) {
      await deleteAPI(request, `/api/v1/auth/me/recommendation-feedback/${Number(item.id)}`, user.token).catch(() => undefined)
    }
    await deleteAPI(request, `/api/v1/auth/me/following/${fixtureAuthorID}`, user.token).catch(() => undefined)
    await putAPI(request, '/api/v1/auth/me/reactions', user.token, { articleId: fixtureArticleID, reaction: 'favorite', active: false }).catch(() => undefined)
    await putAPI(request, '/api/v1/auth/me/reactions', user.token, { articleId: fixtureArticleID, reaction: 'like', active: false }).catch(() => undefined)
    await putAPI(request, `/api/v1/auth/me/topic-subscriptions/${encodeURIComponent('tag')}/${encodeURIComponent(topicKey)}`, user.token, {}).catch(() => undefined)

    try {
      const personalized = await postAPI(request, '/api/v1/auth/me/recommendations/query', user.token, {
        size: 12,
        seedArticleIds: [readerFixtureID]
      })
      expect(personalized.personalized, 'a topic subscription and local seed must personalize the feed').toBe(true)
      const recommendations = itemsOf(personalized)
      expect(recommendations.length, 'recommendations must contain public articles').toBeGreaterThan(0)
      expect(recommendations.some((item: any) => Number(item.id) === fixtureArticleID)).toBe(true)
      expect(recommendations.every((item: any) => item.author?.handle !== 'e2e-user')).toBe(true)
      expect(recommendations.every((item: any) => Boolean(item.reason?.label))).toBe(true)

      const target = recommendations.find((item: any) => Number(item.id) === fixtureArticleID)
      const articleFeedback = await putAPI(request, '/api/v1/auth/me/recommendation-feedback', user.token, {
        targetType: 'article', articleId: fixtureArticleID
      })
      expect(Number(articleFeedback.id)).toBeGreaterThan(0)
      const hidden = await postAPI(request, '/api/v1/auth/me/recommendations/query', user.token, { size: 12 })
      expect(itemsOf(hidden).some((item: any) => Number(item.id) === fixtureArticleID)).toBe(false)
      await deleteAPI(request, `/api/v1/auth/me/recommendation-feedback/${Number(articleFeedback.id)}`, user.token)

      const authorFeedback = await putAPI(request, '/api/v1/auth/me/recommendation-feedback', user.token, {
        targetType: 'author', authorId: Number(target.userId)
      })
      const topicFeedback = await putAPI(request, '/api/v1/auth/me/recommendation-feedback', user.token, {
        targetType: 'topic', topicType: 'tag', topicKey
      })
      const stored = await getAPIData(request, '/api/v1/auth/me/recommendation-feedback?current=1&size=100', user.token)
      expect(itemsOf(stored).some((item: any) => Number(item.id) === Number(authorFeedback.id))).toBe(true)
      expect(itemsOf(stored).some((item: any) => Number(item.id) === Number(topicFeedback.id))).toBe(true)
      await deleteAPI(request, `/api/v1/auth/me/recommendation-feedback/${Number(authorFeedback.id)}`, user.token)
      await deleteAPI(request, `/api/v1/auth/me/recommendation-feedback/${Number(topicFeedback.id)}`, user.token)

      readerSession = await createBrowserSession(browser, user)
      await readerSession.page.goto('/for-you', { waitUntil: 'domcontentloaded' })
      await expect(readerSession.page.getByTestId('recommendation-panel')).toBeVisible()
      await expect(readerSession.page.getByTestId('recommendation-card').first()).toBeVisible()
    } finally {
      await deleteAPI(request, `/api/v1/auth/me/topic-subscriptions/${encodeURIComponent('tag')}/${encodeURIComponent(topicKey)}`, user.token).catch(() => undefined)
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

  test('ranks real reader signals and keeps private work out of discovery', async ({ request }) => {
    const publicTitle = `integration discovery public ${runID}`
    const privateTitle = `integration private ${runID}`
    const publicArticle = await postAPI(request, '/api/v1/studio/articles', admin.token, {
      articleTitle: publicTitle,
      articleContent: 'public discovery fixture',
      visibility: 'public',
      type: 1
    })
    const publicArticleID = Number(publicArticle?.id || 0)
    expect(publicArticleID, 'public article must be created').toBeGreaterThan(0)
    const created = await postAPI(request, '/api/v1/studio/articles', admin.token, {
      articleTitle: privateTitle,
      articleContent: 'private discovery fixture',
      visibility: 'private',
      type: 1
    })
    const privateArticleID = Number(created?.id || 0)
    expect(privateArticleID, 'private article must be created').toBeGreaterThan(0)

    try {
      const authorLatest = await getAPIData(request, '/api/v1/public/authors/e2e-admin/articles?sort=latest&current=1&size=36')
      const authorLatestRecords = itemsOf(authorLatest)
      expect(
        authorLatestRecords.some((item: any) => Number(item.id) === publicArticleID),
        'a freshly published article must reach its public author feed'
      ).toBe(true)
      expect(authorLatestRecords.some((item: any) => Number(item.id) === privateArticleID)).toBe(false)
      const hot = await getAPIData(request, '/api/v1/public/feed?type=article&sort=hot&current=1&size=12')
      const hotRecords = itemsOf(hot)
      expect(hotRecords.length, 'hot feed must not be empty').toBeGreaterThan(0)
      expect(hotRecords.every((item: any) => Number(item.status) === 1)).toBe(true)
      expect(hotRecords.every((item: any) => typeof item.likeCount === 'number')).toBe(true)
      expect(
        hotRecords.some((item: any) => Number(item.id) === fixtureArticleID),
        'the fixture article carries seeded reads, a like, a favourite and approved comments'
      ).toBe(true)

      const latest = await getAPIData(request, '/api/v1/public/feed?type=article&sort=latest&current=1&size=12')
      expect(itemsOf(latest).length).toBeGreaterThan(0)

      const featured = await getAPIData(request, '/api/v1/public/feed?type=article&sort=featured&current=1&size=12')
      expect(Array.isArray(itemsOf(featured))).toBe(true)

      const authors = await getAPIData(request, '/api/v1/public/authors?sort=followers&current=1&size=12')
      expect(itemsOf(authors).some((item: any) => item.handle === 'e2e-admin')).toBe(true)

      const activeAuthors = await getAPIData(request, '/api/v1/public/authors?sort=active&current=1&size=12')
      const activeEntry = itemsOf(activeAuthors).find((item: any) => item.handle === 'e2e-admin')
      expect(activeEntry?.lastPublishedAt, 'active ranking must expose the last public publish').toBeTruthy()

      const topics = await getAPIData(request, '/api/v1/public/topics?size=12')
      const topicGroups = [
        ...(Array.isArray(topics?.categories) ? topics.categories : []),
        ...(Array.isArray(topics?.tags) ? topics.tags : []),
        ...(Array.isArray(topics?.series) ? topics.series : [])
      ]
      expect(topicGroups.length, 'topic plaza must expose at least one public topic').toBeGreaterThan(0)
      expect(topicGroups.every((item: any) => Number(item.articleCount) > 0)).toBe(true)

      const authorHot = await getAPIData(request, '/api/v1/public/authors/e2e-admin/articles?sort=hot&current=1&size=24')
      expect(itemsOf(authorHot).some((item: any) => Number(item.id) === fixtureArticleID)).toBe(true)

      // The private article is the isolation probe: it must never surface on a
      // public discovery surface, in any sort order.
      const publicSurfaces = [
        ...hotRecords,
        ...itemsOf(latest),
        ...itemsOf(featured),
        ...itemsOf(await getAPIData(request, '/api/v1/public/feed?type=article&sort=hot&current=1&size=36')),
        ...itemsOf(await getAPIData(request, '/api/v1/public/authors/e2e-admin/articles?sort=latest&current=1&size=36')),
        ...itemsOf(authorHot)
      ]
      expect(publicSurfaces.some((item: any) => Number(item.id) === privateArticleID)).toBe(false)
      expect(publicSurfaces.some((item: any) => String(item.articleTitle || '').includes(privateTitle))).toBe(false)

      const search = await getAPIData(request, `/api/v1/public/articles/search?keywords=${encodeURIComponent(privateTitle)}`)
      expect(JSON.stringify(search || {})).not.toContain(privateTitle)

      // Direct access is the last line of defence: even knowing the id, an
      // anonymous reader must not receive the private body.
      const privateResponse = await request.get(`/api/v1/public/articles/${privateArticleID}`)
      expect(await privateResponse.text()).not.toContain('private discovery fixture')

      const publicResponse = await request.get(`/api/v1/public/articles/${publicArticleID}`)
      expect(await publicResponse.text()).toContain('public discovery fixture')
    } finally {
      await deleteAPI(request, '/api/v1/studio/articles', admin.token, [publicArticleID, privateArticleID]).catch(() => undefined)
    }
  })

  test('subscribes to a cross-author topic and receives the feed and notification', async ({ request, browser }) => {
    const topicName = 'Integration Topic'
    const topicKey = 'integration topic'
    readerSession = await createBrowserSession(browser, user)
    authorSession = await createBrowserSession(browser, admin)

    // Start from a clean slate: an earlier run must never leave the reader
    // already subscribed, which would hide the subscribe control.
    for (const topicType of ['category', 'tag']) {
      await deleteAPI(request, `/api/v1/auth/me/topic-subscriptions/${topicType}/${encodeURIComponent(topicKey)}`, user.token).catch(() => undefined)
    }

    // Subscribe from the public plaza so the reader-facing control is exercised.
    // The category and tag groups share the display name, so the tag group is
    // targeted explicitly.
    await readerSession.page.goto('/topics', { waitUntil: 'domcontentloaded' })
    const tagGroup = readerSession.page.locator('.topics-group').filter({ has: readerSession.page.locator('h2', { hasText: '标签' }) })
    const topicCard = tagGroup.locator('.topic-card', { hasText: topicName }).first()
    await expect(topicCard).toBeVisible()
    await topicCard.getByRole('button', { name: '订阅', exact: true }).click()
    await expect(topicCard.getByRole('button', { name: '已订阅', exact: true })).toBeVisible()

    const subscriptions = await getAPIData(request, '/api/v1/auth/me/topic-subscriptions?current=1&size=50', user.token)
    const subscription = itemsOf(subscriptions).find((item: any) => item.topicKey === topicKey)
    expect(subscription, 'subscription must be stored').toBeTruthy()
    expect(Number(subscription.articleCount), 'seeded topic spans both authors').toBeGreaterThanOrEqual(2)

    // Publish a matching article as the other author.
    const tags = await getAPIData(request, '/api/v1/studio/tags', admin.token)
    const tagItems = Array.isArray(tags) ? tags : Array.isArray(tags?.records) ? tags.records : []
    const tag = tagItems.find((item: any) => String(item.tagName || '').trim().toLowerCase() === topicKey)
    expect(tag, 'seeded topic tag must belong to the author').toBeTruthy()

    const articleTitle = `integration topic article ${runID}`
    const created = await postAPI(request, '/api/v1/studio/articles', admin.token, {
      articleTitle,
      articleContent: 'topic subscription fixture',
      visibility: 'public',
      type: 1,
      tagIds: [Number(tag.id)]
    })
    const articleID = Number(created?.id || 0)
    expect(articleID, 'topic article must be created').toBeGreaterThan(0)

    try {
      const feed = await getAPIData(request, '/api/v1/auth/me/topic-feed?current=1&size=20', user.token)
      expect(itemsOf(feed).some((item: any) => Number(item.contentId) === articleID)).toBe(true)

      const notifications = await getAPIData(request, '/api/v1/auth/me/notifications?group=topic&current=1&size=20', user.token)
      expect(itemsOf(notifications).some((item: any) => Number(item.contentId) === articleID)).toBe(true)
      expect(Number(notifications.unreadCount)).toBeGreaterThan(0)

      // Muting the subscription hides the notification but keeps the feed.
      await putAPI(request, `/api/v1/auth/me/topic-subscriptions/${encodeURIComponent('tag')}/${encodeURIComponent(topicKey)}/mute`, user.token, { muted: 1 })
      const muted = await getAPIData(request, '/api/v1/auth/me/notifications?group=topic&current=1&size=20', user.token)
      expect(itemsOf(muted).some((item: any) => Number(item.contentId) === articleID)).toBe(false)
      const mutedFeed = await getAPIData(request, '/api/v1/auth/me/topic-feed?current=1&size=20', user.token)
      expect(itemsOf(mutedFeed).some((item: any) => Number(item.contentId) === articleID)).toBe(true)

      // The authenticated following page renders the same content.
      await readerSession.page.goto('/following', { waitUntil: 'domcontentloaded' })
      await readerSession.page.getByRole('button', { name: '话题', exact: true }).click()
      await expect(readerSession.page.getByText(articleTitle).first()).toBeVisible()
      await expect(readerSession.page.getByRole('button', { name: '恢复通知', exact: true })).toBeVisible()
    } finally {
      await deleteAPI(request, '/api/v1/studio/articles', admin.token, [articleID]).catch(() => undefined)
      await deleteAPI(request, `/api/v1/auth/me/topic-subscriptions/${encodeURIComponent('tag')}/${encodeURIComponent(topicKey)}`, user.token).catch(() => undefined)
    }
  })

  async function seedApprovedCollectionComment(request: APIRequestContext, session: LoginSession, collectionID: number, text: string): Promise<number> {
    await postAPI(request, '/api/v1/public/comments', session.token, { type: 6, topicId: String(collectionID), commentContent: text })
    const commentID = await waitForPendingComment(request, session.token, text)
    await approveComments(request, session.token, [commentID])
    createdCommentIDs.push(commentID)
    return commentID
  }

  async function seedApprovedCollectionReply(request: APIRequestContext, session: LoginSession, collectionID: number, parentCommentID: number, text: string): Promise<number> {
    await postAPI(request, '/api/v1/public/comments', session.token, { type: 6, topicId: String(collectionID), parentId: parentCommentID, replyUserId: Number(user.userInfo.userInfoId), commentContent: text })
    const replyID = await waitForPendingComment(request, session.token, text)
    await approveComments(request, session.token, [replyID])
    createdCommentIDs.push(replyID)
    return replyID
  }

})

async function loginByAPI(request: APIRequestContext, email: string, password: string): Promise<LoginSession> {
  const response = await request.post('/api/v1/auth/login', { form: { username: email, password } })
  const payload = await response.json()
  expect(response.status(), `login ${email}: ${JSON.stringify(payload)}`).toBe(200)
  expect(Boolean(payload.flag || payload.code === 'OK'), `login ${email}: ${JSON.stringify(payload)}`).toBe(true)
  return { token: String(payload.data?.token || ''), userInfo: payload.data || {} }
}

async function createBrowserSession(browser: Browser, session: LoginSession, extraHTTPHeaders: Record<string, string> = {}): Promise<BrowserSession> {
  const context = await browser.newContext({ baseURL: blogBaseURL, locale: 'zh-CN', extraHTTPHeaders: { ...e2eHeaders, ...extraHTTPHeaders } })
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

async function getAPIData(request: APIRequestContext, path: string, token = '', extraHeaders: Record<string, string> = {}): Promise<any> {
  const response = await request.get(path, { headers: { ...e2eHeaders, ...(token ? authorization(token) : {}), ...extraHeaders } })
  const body = await response.text()
  expect(response.status(), `${path}: ${body}`).toBe(200)
  const payload = JSON.parse(body)
  expect(Boolean(payload.flag || payload.code === 'OK'), `${path}: ${body}`).toBe(true)
  return payload.data
}

async function postAPI(request: APIRequestContext, path: string, token: string, data: unknown, extraHeaders: Record<string, string> = {}): Promise<any> {
  const response = await request.post(path, { headers: { ...e2eHeaders, ...authorization(token), ...extraHeaders }, data })
  return assertMutation(response, `POST ${path}`)
}

async function putAPI(request: APIRequestContext, path: string, token: string, data: unknown, extraHeaders: Record<string, string> = {}): Promise<any> {
  const response = await request.put(path, { headers: { ...e2eHeaders, ...authorization(token), ...extraHeaders }, data })
  return assertMutation(response, `PUT ${path}`)
}

async function deleteAPI(request: APIRequestContext, path: string, token: string, data?: unknown, extraHeaders: Record<string, string> = {}): Promise<any> {
  const response = await request.delete(path, { headers: { ...e2eHeaders, ...authorization(token), ...extraHeaders }, data })
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

async function findReaderFixtureArticleID(request: APIRequestContext): Promise<number> {
  const data = await getAPIData(request, '/api/v1/public/authors/e2e-user/articles?current=1&size=100')
  const item = itemsOf(data).find((entry: any) => entry.articleTitle === 'Integration reader topic fixture')
  expect(item, 'seeded reader topic article must exist').toBeTruthy()
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

async function approveComments(request: APIRequestContext, token: string, ids: number[], extraHeaders: Record<string, string> = {}): Promise<void> {
  await putAPI(request, '/api/v1/admin/comments/review', token, { ids, isReview: 1 }, extraHeaders)
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
