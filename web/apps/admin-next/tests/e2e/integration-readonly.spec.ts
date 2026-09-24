import { expect, test, type Page } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const adminToken = process.env.E2E_ADMIN_TOKEN || ''

interface MenuContract {
  path: string
  marker: string
  table?: boolean
  contentSelector?: string
  required?: boolean
}

const visibleMenuContracts: MenuContract[] = [
  { path: '/', marker: '发布文章' },
  { path: '/growth', marker: '订阅与增长', contentSelector: '.growth-overview-grid' },
  { path: '/articles', marker: '发布文章', contentSelector: '.article-form' },
  { path: '/article-list', marker: '文章列表', table: true },
  { path: '/content-moderation', marker: '内容审核', table: true },
  { path: '/categories', marker: '分类管理', table: true },
  { path: '/tags', marker: '标签管理', table: true },
  { path: '/comments', marker: '评论管理', table: true, contentSelector: '.comment-governance-queues' },
  { path: '/users', marker: '用户管理', table: true },
  { path: '/online/users', marker: '在线用户', table: true },
  { path: '/roles', marker: '角色管理', table: true },
  { path: '/operation/log', marker: '操作日志', table: true },
  { path: '/exception/log', marker: '异常日志', table: true },
  { path: '/quartz', marker: '定时任务', table: true },
  { path: '/albums', marker: '相册管理', table: true },
  { path: '/talk-list', marker: '说说管理', table: true },
  { path: '/talks', marker: '发布说说', contentSelector: '.talk-form' },
  { path: '/menus', marker: '菜单管理', table: true },
  { path: '/resources', marker: '接口资源管理', table: true },
  { path: '/links', marker: '友链管理', table: true },
  { path: '/about', marker: '关于我' },
  { path: '/website', marker: '网站配置' },
  { path: '/setting', marker: '个人中心' }
]

test.describe('admin-next real read-only integration', () => {
  test('uses only visible backend menu permissions for navigation @integration @readonly', async ({ page }) => {
    test.setTimeout(120_000)
    test.skip(
      !realIntegration || !adminToken,
      'set E2E_REAL_INTEGRATION=1 and E2E_ADMIN_TOKEN to run against an existing backend/Caddy entrypoint'
    )

    const pageErrors: string[] = []
    const apiFailures: string[] = []
    const unsafeRequests: string[] = []
    const failedRequests: string[] = []
    page.on('pageerror', (error) => pageErrors.push(error.message))
    page.on('console', (message) => {
      if (message.type() === 'error' && !message.text().includes('status of 502 (Bad Gateway)')) pageErrors.push(message.text())
    })
    page.on('request', (request) => {
      const requestURL = new URL(request.url())
      if (!requestURL.pathname.startsWith('/api')) return
      if (!['GET', 'HEAD', 'OPTIONS'].includes(request.method())) {
        unsafeRequests.push(`${request.method()} ${requestURL.pathname}`)
      }
    })
    page.on('requestfailed', (request) => {
      const requestURL = new URL(request.url())
      if (requestURL.pathname.startsWith('/api')) {
        failedRequests.push(`${request.method()} ${requestURL.pathname}: ${request.failure()?.errorText || 'request failed'}`)
      }
    })
    page.on('response', (response) => {
      const responseURL = new URL(response.url())
      if (responseURL.pathname.startsWith('/api') && response.status() >= 400) {
        apiFailures.push(`${response.status()} ${response.request().method()} ${responseURL.pathname}`)
      }
    })

    const menuResponsePromise = page.waitForResponse((response) => {
      const responseURL = new URL(response.url())
      return responseURL.pathname === '/api/v1/admin/me/menu' && response.request().method() === 'GET'
    })
    await page.addInitScript((token) => {
      window.sessionStorage.setItem('token', token)
    }, adminToken)

    const response = await page.goto('/', { waitUntil: 'domcontentloaded' })
    expect(response?.status()).toBe(200)
    const menuResponse = await menuResponsePromise
    expect(menuResponse.status()).toBe(200)
    const menuPayload = await menuResponse.json() as { flag?: boolean; code?: number | string; data?: unknown }
    expect(menuPayload.code === 'OK' || menuPayload.code === 'SUCCESS' || menuPayload.code === 20000 || menuPayload.flag === true).toBe(true)
    expect(Array.isArray(menuPayload.data)).toBe(true)

    const menuItems = flattenMenuItems(menuPayload.data)
    const visibleItems = menuItems.filter((item) => !item.hidden)
    const parentPaths = [...new Set(visibleItems
      .filter((item) => item.parentPath && item.path !== item.parentPath)
      .map((item) => item.parentPath)
      .filter((path): path is string => Boolean(path)))]
    const hiddenLabels = menuItems
      .filter((item) => item.hidden)
      .map((item) => item.name)
      .filter((name) => name.length > 0)
    const sidebar = page.locator('.arco-layout-sider').first()
    for (const label of hiddenLabels) {
      await expect(sidebar.getByText(label, { exact: true }), `hidden menu ${label}`).toHaveCount(0)
    }

    for (const menu of visibleMenuContracts) {
      const menuItem = visibleItems.find((item) => item.path === menu.path)
      if (!menuItem) throw new Error(`required backend menu path is missing: ${menu.path}`)
      const displayName = menuDisplayNames[menu.path] || menuItem.name
      let menuScope = sidebar
      if (menuItem.parentPath) {
        const parent = visibleItems.find((item) => item.path === menuItem.parentPath)
        if (!parent) throw new Error(`parent menu is missing for ${menu.path}: ${menuItem.parentPath}`)
        const parentIndex = parentPaths.indexOf(parent.path)
        if (parentIndex < 0) throw new Error(`parent menu order is missing for ${menu.path}: ${parent.path}`)
        menuScope = sidebar.locator('.arco-menu-inline').nth(parentIndex)
        const parentItem = menuScope.locator('.arco-menu-inline-header')
        await parentItem.scrollIntoViewIfNeeded()
        const childItem = menuScope.locator('.arco-menu-item').filter({ hasText: displayName })
        if (await childItem.count() === 0 || !(await childItem.first().isVisible())) {
          await parentItem.locator('.arco-menu-icon-suffix').click()
        }
      }
      const item = menuScope.locator('.arco-menu-item').filter({ hasText: displayName })
      await item.scrollIntoViewIfNeeded()
      await expect(item, `visible menu ${menu.path} (${menuItem.name})`).toHaveCount(1)
      await Promise.all([
        page.waitForURL((url) => url.pathname === menu.path),
        item.click()
      ])
      await assertRenderedRoute(page, `menu ${menu.path}`, menu.marker, undefined, menu.path)
      if (menu.table) await expect(page.locator('.arco-table'), `table ${menu.path}`).toBeVisible()
      if (menu.contentSelector) {
        await expect(page.locator(menu.contentSelector), `content ${menu.path}`).toBeVisible()
      }
      if (menu.path === '/growth') {
        await page.getByText('增长分析', { exact: true }).click()
        await expect(page.locator('.growth-activation-panel'), 'creator activation funnel').toContainText('创作者激活')
      }
      await page.waitForLoadState('networkidle')
      const refreshed = await page.reload({ waitUntil: 'domcontentloaded' })
      await assertRenderedRoute(page, `${menu.path} refresh`, menu.marker, refreshed?.status(), menu.path)
      await page.waitForLoadState('networkidle')
    }

    const ignorableProxyFailure = (failure: string) => failure.includes('/api/v1/public/media/proxy')
    expect(unsafeRequests).toEqual([])
    expect(apiFailures).toEqual([])
    // Seeded covers still point at the retired OSS bucket, so the media proxy
    // aborts those image requests. That is an environment artifact, not a
    // frontend regression.
    expect(failedRequests.filter((failure) => !ignorableProxyFailure(failure))).toEqual([])
    expect(pageErrors).toEqual([])
  })

  test('opens and refreshes migrated modules without API writes @integration @readonly', async ({ page }) => {
    test.setTimeout(120_000)
    test.skip(
      !realIntegration || !adminToken,
      'set E2E_REAL_INTEGRATION=1 and E2E_ADMIN_TOKEN to run against an existing backend/Caddy entrypoint'
    )

    const pageErrors: string[] = []
    const apiFailures: string[] = []
    const unsafeRequests: string[] = []
    const failedRequests: string[] = []

    await page.addInitScript((token) => {
      window.sessionStorage.setItem('token', token)
    }, adminToken)

    page.on('pageerror', (error) => pageErrors.push(error.message))
    page.on('console', (message) => {
      if (message.type() === 'error' && !message.text().includes('status of 502 (Bad Gateway)')) pageErrors.push(message.text())
    })
    page.on('request', (request) => {
      const requestURL = new URL(request.url())
      if (!requestURL.pathname.startsWith('/api')) return
      if (!['GET', 'HEAD', 'OPTIONS'].includes(request.method())) {
        unsafeRequests.push(`${request.method()} ${requestURL.pathname}`)
      }
    })
    page.on('requestfailed', (request) => {
      const requestURL = new URL(request.url())
      if (requestURL.pathname.startsWith('/api')) {
        failedRequests.push(`${request.method()} ${requestURL.pathname}: ${request.failure()?.errorText || 'request failed'}`)
      }
    })
    page.on('response', (response) => {
      const responseURL = new URL(response.url())
      if (responseURL.pathname.startsWith('/api') && response.status() >= 400) {
        apiFailures.push(`${response.status()} ${response.request().method()} ${responseURL.pathname}`)
      }
    })

    const menuResponsePromise = page.waitForResponse((response) => {
      const responseURL = new URL(response.url())
      return responseURL.pathname === '/api/v1/admin/me/menu' && response.request().method() === 'GET'
    })
    const homeResponse = await page.goto('/', { waitUntil: 'domcontentloaded' })
    expect(homeResponse?.status()).toBe(200)
    const menuResponse = await menuResponsePromise
    expect(menuResponse.status()).toBe(200)
    const menuPayload = await menuResponse.json() as { flag?: boolean; code?: number | string; data?: unknown }
    expect(menuPayload.code === 'OK' || menuPayload.code === 'SUCCESS' || menuPayload.code === 20000 || menuPayload.flag === true).toBe(true)
    const menuItems = flattenMenuItems(menuPayload.data)

    const routes = [
      { path: `/articles/${await firstAdminRecordID(page, adminToken, '/api/v1/admin/articles')}`, menuPaths: ['/articles/*'], marker: '修改文章', contentSelector: '.article-form' },
      { path: '/quartz/log/85', menuPaths: ['/quartz/log/:quartzId'], marker: '任务日志' },
      { path: `/albums/${await firstAdminRecordID(page, adminToken, '/api/v1/admin/albums')}`, menuPaths: ['/albums/*', '/albums/:albumId'], marker: '上传照片', contentSelector: '.photo-masonry' },
      { path: '/photos/delete', menuPaths: ['/photos/delete'], marker: '照片回收站' },
      { path: `/talks/${await firstAdminRecordID(page, adminToken, '/api/v1/admin/talks')}`, menuPaths: ['/talks/*', '/talks/:talkId'], marker: '编辑说说', contentSelector: '.talk-form' }
    ]

    for (const route of routes) {
      const available = route.menuPaths.some((path) => menuItems.some((item) => item.path === path))
      if (!available) throw new Error(`required backend menu path is missing for ${route.path}: ${route.menuPaths.join(', ')}`)
      const response = await page.goto(route.path, { waitUntil: 'domcontentloaded' })
      await assertRenderedRoute(page, route.path, route.marker, response?.status())
      if (route.contentSelector) {
        await expect(page.locator(route.contentSelector), `content ${route.path}`).toBeVisible()
      }
      await page.waitForLoadState('networkidle')

      const refreshed = await page.reload({ waitUntil: 'domcontentloaded' })
      await assertRenderedRoute(page, `${route.path} refresh`, route.marker, refreshed?.status(), route.path)
      if (route.contentSelector) {
        await expect(page.locator(route.contentSelector), `content ${route.path} refresh`).toBeVisible()
      }
      await page.waitForLoadState('networkidle')
    }

    expect(unsafeRequests).toEqual([])
    expect(apiFailures.filter((failure) => !failure.startsWith('502 GET /api/v1/public/media/proxy'))).toEqual([])
    expect(failedRequests.filter((failure) => !failure.includes('/api/v1/public/media/proxy'))).toEqual([])
    expect(pageErrors.filter((error) => !error.includes('status of 502 (Bad Gateway)'))).toEqual([])
  })
})

/**
 * Detail routes must point at a record that exists in the target environment;
 * a hard-coded id is only valid for one seeded dataset.  Read the first id
 * through the admin API so this read-only suite never writes anything.
 */
async function firstAdminRecordID(page: Page, token: string, endpoint: string): Promise<number> {
  const response = await page.request.get(`${endpoint}?current=1&size=1`, {
    headers: { Authorization: `Bearer ${token}` }
  })
  expect(response.status(), `GET ${endpoint}`).toBe(200)
  const payload = await response.json() as { code?: string | number; data?: { items?: Array<{ id?: number }> } }
  expect(payload.code === 'OK' || payload.code === 20000, `GET ${endpoint} application success`).toBe(true)
  const id = Number(payload.data?.items?.[0]?.id || 0)
  expect(id, `GET ${endpoint} returned no record to open`).toBeGreaterThan(0)
  return id
}

async function assertRenderedRoute(page: Page, label: string, marker: string, status: number | undefined, expectedPath = label): Promise<void> {
  if (status !== undefined) expect(status, label).toBe(200)
  await expect(page, label).toHaveURL((url) => url.pathname === expectedPath)
  await expect(page.locator('.admin-shell'), label).toBeVisible()
  const content = page.locator('.admin-content')
  await expect(content, label).toBeVisible()
  await expect(content.locator(':scope > *').first(), label).toBeVisible()
  await expect(content, label).toContainText(marker)
  await expect(content, label).not.toContainText('页面不存在')
  await expect(content, label).not.toContainText('无权访问')
  await expect(content, label).not.toContainText('模块迁移中')
}

function flattenMenuItems(value: unknown, parentPath = ''): Array<{ name: string; path: string; hidden: boolean; parentPath?: string }> {
  if (!Array.isArray(value)) return []
  const result: Array<{ name: string; path: string; hidden: boolean; parentPath?: string }> = []
  for (const item of value) {
    if (!item || typeof item !== 'object') continue
    const record = item as Record<string, unknown>
    const rawPath = typeof record.path === 'string' ? record.path.trim() : ''
    const path = rawPath === '' || rawPath === '/' ? parentPath || '/' : rawPath.startsWith('/') ? rawPath : `/${rawPath}`
    result.push({
      name: typeof record.name === 'string' ? record.name : '',
      path,
      hidden: Boolean(record.hidden),
      parentPath: parentPath || undefined
    })
    result.push(...flattenMenuItems(record.children, path))
  }
  return result
}

const menuDisplayNames: Record<string, string> = {
  '/albums': '相册管理',
  '/talk-list': '说说管理',
  '/users': '用户管理',
  '/resources': '资源管理'
}
