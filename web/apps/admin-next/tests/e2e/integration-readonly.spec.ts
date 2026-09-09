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
  { path: '/', marker: '让每一次发布，都更从容。' },
  { path: '/articles', marker: '发布文章', contentSelector: '.article-form' },
  { path: '/article-list', marker: '文章列表', table: true },
  { path: '/categories', marker: '分类管理', table: true },
  { path: '/tags', marker: '标签管理', table: true },
  { path: '/comments', marker: '评论管理', table: true },
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
      if (message.type() === 'error') pageErrors.push(message.text())
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
    const menuPayload = await menuResponse.json() as { flag?: boolean; code?: number; data?: unknown }
    expect(menuPayload.flag).toBe(true)
    expect(menuPayload.code).toBe(20000)
    expect(Array.isArray(menuPayload.data)).toBe(true)

    const menuItems = flattenMenuItems(menuPayload.data)
    const visibleItems = menuItems.filter((item) => !item.hidden)
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
      const item = sidebar.locator('.arco-menu-item').filter({ hasText: menuItem.name })
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
      await page.waitForLoadState('networkidle')
      const refreshed = await page.reload({ waitUntil: 'domcontentloaded' })
      await assertRenderedRoute(page, `${menu.path} refresh`, menu.marker, refreshed?.status(), menu.path)
      await page.waitForLoadState('networkidle')
    }

    expect(unsafeRequests).toEqual([])
    expect(apiFailures).toEqual([])
    expect(failedRequests).toEqual([])
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
      if (message.type() === 'error') pageErrors.push(message.text())
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
    const menuPayload = await menuResponse.json() as { flag?: boolean; code?: number; data?: unknown }
    expect(menuPayload.flag).toBe(true)
    expect(menuPayload.code).toBe(20000)
    const menuItems = flattenMenuItems(menuPayload.data)

    const routes = [
      { path: '/articles/42', menuPaths: ['/articles/*'], marker: '修改文章', contentSelector: '.article-form' },
      { path: '/quartz/log/85', menuPaths: ['/quartz/log/:quartzId'], marker: '任务日志' },
      { path: '/albums/5', menuPaths: ['/albums/*', '/albums/:albumId'], marker: '照片管理', contentSelector: '.arco-table' },
      { path: '/photos/delete', menuPaths: ['/photos/delete'], marker: '照片回收站' },
      { path: '/talks/7', menuPaths: ['/talks/*', '/talks/:talkId'], marker: '编辑说说', contentSelector: '.talk-form' }
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
    expect(apiFailures).toEqual([])
    expect(failedRequests).toEqual([])
    expect(pageErrors).toEqual([])
  })
})

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

function flattenMenuItems(value: unknown, parentPath = ''): Array<{ name: string; path: string; hidden: boolean }> {
  if (!Array.isArray(value)) return []
  const result: Array<{ name: string; path: string; hidden: boolean }> = []
  for (const item of value) {
    if (!item || typeof item !== 'object') continue
    const record = item as Record<string, unknown>
    const rawPath = typeof record.path === 'string' ? record.path.trim() : ''
    const path = rawPath === '' || rawPath === '/' ? parentPath || '/' : rawPath.startsWith('/') ? rawPath : `/${rawPath}`
    result.push({
      name: typeof record.name === 'string' ? record.name : '',
      path,
      hidden: Boolean(record.hidden)
    })
    result.push(...flattenMenuItems(record.children, path))
  }
  return result
}
