import { expect, test, type Page } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const allowLogin = process.env.E2E_ADMIN_ALLOW_LOGIN === '1'

test.describe('admin-next isolated CRUD integration', () => {
  test('creates edits refreshes and deletes representative content @integration @crud', async ({ page }) => {
    test.setTimeout(180_000)
    test.skip(
      !realIntegration || !allowLogin || !process.env.E2E_ADMIN_EMAIL || !process.env.E2E_ADMIN_PASSWORD,
      'set E2E_REAL_INTEGRATION=1, E2E_ADMIN_ALLOW_LOGIN=1 and isolated admin credentials to run CRUD integration'
    )

    const pageErrors: string[] = []
    const apiFailures: string[] = []
    page.on('pageerror', (error) => pageErrors.push(error.message))
    page.on('console', (message) => {
      if (message.type() === 'error') pageErrors.push(message.text())
    })
    page.on('response', (response) => {
      const responseURL = new URL(response.url())
      if (responseURL.pathname.startsWith('/api') && response.status() >= 400) {
        apiFailures.push(`${response.status()} ${response.request().method()} ${responseURL.pathname}`)
      }
    })

    await login(page)

    const suffix = String(Date.now()).slice(-6)
    const categoryName = `e2e-cat-${suffix}`
    const categoryEdited = `e2e-c2-${suffix}`
    const tagName = `e2e-tag-${suffix}`
    const tagEdited = `e2e-t2-${suffix}`
    const linkName = `e2e-link-${suffix}`
    const linkEdited = `e2e-l2-${suffix}`
    const talkContent = `e2e talk ${suffix}`
    const talkEdited = `e2e talk edited ${suffix}`

    try {
      await runTaxonomyCRUD(page, '/categories', 'categories', categoryName, categoryEdited)
      await runTaxonomyCRUD(page, '/tags', 'tags', tagName, tagEdited)
      await runFriendLinkCRUD(page, linkName, linkEdited)
      await runTalkCRUD(page, talkContent, talkEdited)
    } finally {
      await cleanupTaxonomy(page, '/categories', 'categories', [categoryName, categoryEdited])
      await cleanupTaxonomy(page, '/tags', 'tags', [tagName, tagEdited])
      await cleanupFriendLink(page, [linkName, linkEdited])
      await cleanupTalk(page, [talkContent, talkEdited])
    }

    expect(apiFailures).toEqual([])
    expect(pageErrors).toEqual([])
  })
})

async function login(page: Page): Promise<void> {
  const email = process.env.E2E_ADMIN_EMAIL || ''
  const password = process.env.E2E_ADMIN_PASSWORD || ''
  await page.goto('/login', { waitUntil: 'domcontentloaded' })
  await page.getByTestId('login-username').locator('input').fill(email)
  await page.getByTestId('login-password').locator('input').fill(password)
  await page.getByTestId('login-submit').click()
  await page.waitForURL((url) => url.pathname === '/')
  await expect(page.locator('.admin-shell')).toBeVisible()
}

async function runTaxonomyCRUD(page: Page, route: string, endpoint: string, created: string, edited: string): Promise<void> {
  await page.goto(route, { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  const modal = visibleModal(page)
  await modal.getByRole('textbox').first().fill(created)
  await expectMutation(page, `/api/v1/admin/${endpoint}`, 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, `/api/v1/admin/${endpoint}`, created)
  await expect(tableRow(page, created)).toBeVisible()

  await tableRow(page, created).getByRole('button', { name: '编辑', exact: true }).click()
  const editModal = visibleModal(page)
  await editModal.getByRole('textbox').first().fill(edited)
  await expectMutation(page, `/api/v1/admin/${endpoint}`, 'POST', () => editModal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, `/api/v1/admin/${endpoint}`, edited)
  await expect(tableRow(page, edited)).toBeVisible()
  await expect(tableRow(page, created)).toHaveCount(0)

  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, `/api/v1/admin/${endpoint}`, edited)
  await expect(tableRow(page, edited)).toBeVisible()
  await deleteTableRow(page, `/api/v1/admin/${endpoint}`, edited)
  await expect(tableRow(page, edited)).toHaveCount(0)
}

async function runFriendLinkCRUD(page: Page, created: string, edited: string): Promise<void> {
  await page.goto('/links', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  let modal = visibleModal(page)
  await fillFriendLink(modal, created)
  await expectMutation(page, '/api/v1/admin/friend-links', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/friend-links', created)
  await expect(tableRow(page, created)).toBeVisible()

  await tableRow(page, created).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await fillFriendLink(modal, edited)
  await expectMutation(page, '/api/v1/admin/friend-links', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/friend-links', edited)
  await expect(tableRow(page, edited)).toBeVisible()
  await expect(tableRow(page, created)).toHaveCount(0)

  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/v1/admin/friend-links', edited)
  await expect(tableRow(page, edited)).toBeVisible()
  await deleteTableRow(page, '/api/v1/admin/friend-links', edited)
  await expect(tableRow(page, edited)).toHaveCount(0)
}

async function fillFriendLink(modal: ReturnType<Page['locator']>, name: string): Promise<void> {
  const inputs = modal.locator('input')
  await inputs.nth(0).fill(name)
  await inputs.nth(1).fill('https://example.com/e2e-avatar.png')
  await inputs.nth(2).fill(`https://example.com/e2e-${name}`)
  await modal.locator('textarea').fill(`integration link ${name}`)
}

async function runTalkCRUD(page: Page, created: string, edited: string): Promise<void> {
  await page.goto('/talks', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.talk-form')).toBeVisible()
  await page.locator('.talk-form textarea').fill(created)
  await expectMutation(page, '/api/v1/admin/talks', 'POST', () => page.locator('.talk-form').getByRole('button', { name: '保存', exact: true }).click())
  await page.waitForURL((url) => url.pathname === '/talk-list')
  await expect(tableRow(page, created)).toBeVisible()

  await tableRow(page, created).getByRole('button', { name: '编辑', exact: true }).click()
  await page.waitForURL((url) => /^\/talks\/\d+$/.test(url.pathname))
  await expect(page.locator('.talk-form')).toBeVisible()
  await page.locator('.talk-form textarea').fill(edited)
  await expectMutation(page, '/api/v1/admin/talks', 'POST', () => page.locator('.talk-form').getByRole('button', { name: '保存', exact: true }).click())
  await page.waitForURL((url) => url.pathname === '/talk-list')
  await expect(tableRow(page, edited)).toBeVisible()
  await expect(tableRow(page, created)).toHaveCount(0)

  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(tableRow(page, edited)).toBeVisible()
  await deleteTableRow(page, '/api/v1/admin/talks', edited)
  await expect(tableRow(page, edited)).toHaveCount(0)
}

function visibleModal(page: Page): ReturnType<Page['locator']> {
  return page.locator('.arco-modal:visible').last()
}

function tableRow(page: Page, text: string): ReturnType<Page['locator']> {
  // Arco renders the body rowgroup without a stable tbody class.  The
  // accessibility row contract is stable across the Caddy build and Vite
  // baseline, so keep the test independent of generated table classes.
  return page.getByRole('row').filter({ hasText: text }).first()
}

async function deleteTableRow(page: Page, endpoint: string, text: string): Promise<void> {
  const row = tableRow(page, text)
  await expect(row).toBeVisible()
  await row.getByRole('button', { name: '删除', exact: true }).click()
  const popconfirm = page.locator('.arco-popconfirm:visible').last()
  await expectMutation(page, endpoint, 'DELETE', () => popconfirm.getByRole('button', { name: '确定', exact: true }).click())
}

async function filterTable(page: Page, endpoint: string, value: string): Promise<void> {
  const search = page.locator('.arco-input-search').getByRole('textbox')
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === endpoint &&
      response.request().method() === 'GET' &&
      responseURL.searchParams.get('keywords') === value
  })
  await search.fill(value)
  // InputSearch renders a clear icon before the search icon when it has a
  // value; target the search icon's hover wrapper explicitly.
  await page.locator('.arco-input-search .arco-icon-hover:not(.arco-input-clear-btn)').click()
  const response = await responsePromise
  expect(response.status(), `GET ${endpoint} filtered`).toBe(200)
}

async function expectMutation(page: Page, pathname: string, method: string, action: () => Promise<void> | void): Promise<void> {
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === pathname && response.request().method() === method
  })
  await action()
  const response = await responsePromise
  expect(response.status(), `${method} ${pathname}`).toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number | string }
  expect(payload.code === 'OK' || payload.flag === true, `${method} ${pathname} application success`).toBe(true)
}

async function cleanupTaxonomy(page: Page, route: string, endpoint: string, names: string[]): Promise<void> {
  try {
    await page.goto(route, { waitUntil: 'domcontentloaded' })
    for (const name of names) {
      await filterTable(page, `/api/v1/admin/${endpoint}`, name)
      const row = tableRow(page, name)
      if (await row.count() === 0) continue
      await row.getByRole('button', { name: '删除', exact: true }).click()
      const popconfirm = page.locator('.arco-popconfirm:visible').last()
      const responsePromise = page.waitForResponse((response) => {
        const responseURL = new URL(response.url())
        return responseURL.pathname === `/api/v1/admin/${endpoint}` && response.request().method() === 'DELETE'
      })
      await popconfirm.getByRole('button', { name: '确定', exact: true }).click()
      await responsePromise
    }
  } catch {
    // Preserve the original assertion when cleanup cannot reach the isolated UI.
  }
}

async function cleanupFriendLink(page: Page, names: string[]): Promise<void> {
  try {
    await page.goto('/links', { waitUntil: 'domcontentloaded' })
    for (const name of names) {
      await filterTable(page, '/api/v1/admin/friend-links', name)
      const row = tableRow(page, name)
      if (await row.count() === 0) continue
      await row.getByRole('button', { name: '删除', exact: true }).click()
      const popconfirm = page.locator('.arco-popconfirm:visible').last()
      const responsePromise = page.waitForResponse((response) => {
        const responseURL = new URL(response.url())
        return responseURL.pathname === '/api/v1/admin/friend-links' && response.request().method() === 'DELETE'
      })
      await popconfirm.getByRole('button', { name: '确定', exact: true }).click()
      await responsePromise
    }
  } catch {
    // Preserve the original assertion when cleanup cannot reach the isolated UI.
  }
}

async function cleanupTalk(page: Page, contents: string[]): Promise<void> {
  try {
    await page.goto('/talk-list', { waitUntil: 'domcontentloaded' })
    for (const content of contents) {
      const row = tableRow(page, content)
      if (await row.count() === 0) continue
      await row.getByRole('button', { name: '删除', exact: true }).click()
      const popconfirm = page.locator('.arco-popconfirm:visible').last()
      const responsePromise = page.waitForResponse((response) => {
        const responseURL = new URL(response.url())
        return responseURL.pathname === '/api/v1/admin/talks' && response.request().method() === 'DELETE'
      })
      await popconfirm.getByRole('button', { name: '确定', exact: true }).click()
      await responsePromise
    }
  } catch {
    // Preserve the original assertion when cleanup cannot reach the isolated UI.
  }
}
