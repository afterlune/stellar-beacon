import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'
const allowLogin = process.env.E2E_ADMIN_ALLOW_LOGIN === '1'

test.describe.configure({ mode: 'serial' })

test.describe('admin-next full isolated CRUD integration', () => {
  test('covers every write-capable migrated module with reversible actions @integration @crud @full', async ({ page }) => {
    test.setTimeout(360_000)
    test.skip(
      !realIntegration || !allowLogin || !process.env.E2E_ADMIN_EMAIL || !process.env.E2E_ADMIN_PASSWORD,
      'set E2E_REAL_INTEGRATION=1, E2E_ADMIN_ALLOW_LOGIN=1 and isolated admin credentials to run full CRUD integration'
    )

    page.setDefaultTimeout(10_000)
    page.setDefaultNavigationTimeout(15_000)
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
    const token = await page.evaluate(() => sessionStorage.getItem('token') || '')
    expect(token).not.toBe('')

    const suffix = String(Date.now()).slice(-8)
    await traceStep('article', () => runArticleDraftCRUD(page, token, suffix))
    await traceStep('album', () => runAlbumCRUD(page, suffix))
    await traceStep('job', () => runJobCRUD(page, suffix))
    await traceStep('role', () => runRoleCRUD(page, suffix))
    await traceStep('menus', () => runPermissionCRUD(page, 'menus', suffix))
    await traceStep('resources', () => runPermissionCRUD(page, 'resources', suffix))
    await traceStep('user', () => runUserEditRoundTrip(page, suffix))
    await traceStep('comment', () => runCommentReviewRoundTrip(page))
    await traceStep('website', () => runWebsiteRoundTrip(page, suffix))
    await traceStep('about', () => runAboutRoundTrip(page, suffix))
    await traceStep('setting', () => runSettingRoundTrip(page, suffix))
    await traceStep('photos', () => runPhotoCRUD(page, suffix))
    await traceStep('logs', () => runLogReadOnlyRoundTrips(page))
    await traceStep('provider-probe-disabled', () => runProviderProbeDisabled(page, token))

    // Agent management menus are deliberately optional while AI flags are off.
    // When a deployment exposes them, exercise their reversible configuration paths too.
    if (await hasVisibleMenu(page, 'Agent 人设')) await traceStep('agent-profile', () => runAgentProfileRoundTrip(page, suffix))
    if (await hasVisibleMenu(page, '审核策略')) await traceStep('agent-review-policy', () => runAgentReviewPolicyRoundTrip(page))

    expect(apiFailures).toEqual([])
    expect(pageErrors).toEqual([])
  })
})

async function traceStep(name: string, action: () => Promise<void>): Promise<void> {
  console.log(`[full-crud] start ${name}`)
  await action()
  console.log(`[full-crud] passed ${name}`)
}

async function login(page: Page): Promise<void> {
  await page.goto('/login', { waitUntil: 'domcontentloaded' })
  await page.getByTestId('login-username').locator('input').fill(process.env.E2E_ADMIN_EMAIL || '')
  await page.getByTestId('login-password').locator('input').fill(process.env.E2E_ADMIN_PASSWORD || '')
  await page.getByTestId('login-submit').click()
  await page.waitForURL((url) => url.pathname === '/')
  await expect(page.locator('.admin-shell')).toBeVisible()
}

async function runArticleDraftCRUD(page: Page, token: string, suffix: string): Promise<void> {
  const title = `e2e-article-${suffix}`
  const editedTitle = `e2e-article-edited-${suffix}`
  let articleID = 0
  try {
    await page.goto('/articles', { waitUntil: 'domcontentloaded' })
    await expect(page.locator('.article-form')).toBeVisible()
    const form = page.locator('.article-form')
    const inputs = form.locator('input')
    await inputs.nth(0).fill(title)
    await inputs.nth(1).fill('工程化')
    await inputs.nth(2).fill('e2e')
    await form.locator('textarea').fill(`隔离文章正文 ${suffix}`)
    await selectOption(page, form.locator('.arco-select').first(), '草稿')
    await expectMutation(page, '/api/admin/articles', 'POST', () => form.getByRole('button', { name: '保存', exact: true }).click())
    await expect(page).toHaveURL(/\/article-list$/)

    let searchResponse = await filterTable(page, '/api/admin/articles', title)
    articleID = firstRecordID(await searchResponse.json())
    expect(articleID, 'created draft article id').toBeGreaterThan(0)
    await expect(rowWithText(page, title)).toBeVisible()

    await page.goto(`/articles/${articleID}`, { waitUntil: 'domcontentloaded' })
    await expect(page.locator('.article-form')).toBeVisible()
    await page.locator('.article-form input').nth(0).fill(editedTitle)
    await page.locator('.article-form textarea').fill(`隔离文章正文已编辑 ${suffix}`)
    await expectMutation(page, '/api/admin/articles', 'POST', () => page.locator('.article-form').getByRole('button', { name: '保存', exact: true }).click())
    await expect(page).toHaveURL(/\/article-list$/)
    searchResponse = await filterTable(page, '/api/admin/articles', editedTitle)
    expect(firstRecordID(await searchResponse.json())).toBe(articleID)
    await expect(rowWithText(page, editedTitle)).toBeVisible()
    await page.reload({ waitUntil: 'domcontentloaded' })
    await filterTable(page, '/api/admin/articles', editedTitle)
    await expect(rowWithText(page, editedTitle)).toBeVisible()
  } finally {
    if (articleID > 0) await deleteAdminIDs(page.context().request, token, '/api/admin/articles/delete', [articleID])
  }
}

async function runAlbumCRUD(page: Page, suffix: string): Promise<void> {
  const shortSuffix = suffix.slice(-4)
  const name = `e2e-a-${shortSuffix}`
  const editedName = `e2e-a2-${shortSuffix}`
  await page.goto('/albums', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  let modal = visibleModal(page)
  const textInputs = modal.locator('input[type="text"]')
  await textInputs.nth(0).fill(name)
  await modal.locator('textarea').fill(`隔离相册 ${suffix}`)
  await textInputs.nth(1).fill(`https://example.com/e2e-album-${suffix}.png`)
  await expectMutation(page, '/api/admin/photos/albums', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/photos/albums', name)
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').nth(0).fill(editedName)
  await expectMutation(page, '/api/admin/photos/albums', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/photos/albums', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/admin/photos/albums', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await deleteAlbumRow(page, editedName)
  await expect(rowWithText(page, editedName)).toHaveCount(0)
}

async function runJobCRUD(page: Page, suffix: string): Promise<void> {
  const name = `e2e-job-${suffix}`
  const editedName = `e2e-job-edited-${suffix}`
  await page.goto('/quartz', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  let modal = visibleModal(page)
  let inputs = modal.locator('input[type="text"]')
  await inputs.nth(0).fill(name)
  await inputs.nth(1).fill('e2e')
  await inputs.nth(2).fill('agent.content_understanding')
  await inputs.nth(3).fill('0 0 1 1 1 ?')
  await expectMutation(page, '/api/admin/jobs', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/jobs', name, 'jobName')
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await expect(modal).toContainText('编辑任务')
  inputs = modal.locator('input[type="text"]')
  await inputs.nth(0).fill(editedName)
  await expectMutation(page, '/api/admin/jobs', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/jobs', editedName, 'jobName')
  await expect(rowWithText(page, editedName)).toBeVisible()

  const row = rowWithText(page, editedName)
  const toggle = row.locator('.arco-switch')
  if (await toggle.count() > 0) {
    await expectMutation(page, '/api/admin/jobs/status', 'PUT', () => toggle.click())
    await expectMutation(page, '/api/admin/jobs/status', 'PUT', () => toggle.click())
  }
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/admin/jobs', editedName, 'jobName')
  await expect(rowWithText(page, editedName)).toBeVisible()
  await deleteTableRow(page, '/api/admin/jobs', editedName)
  await expect(rowWithText(page, editedName)).toHaveCount(0)
}

async function runRoleCRUD(page: Page, suffix: string): Promise<void> {
  // t_role.role_name is VARCHAR(20); keep both values within the persisted limit.
  const shortSuffix = suffix.slice(-8)
  const name = `e2e-r-${shortSuffix}`
  const editedName = `e2e-r2-${shortSuffix}`
  await page.goto('/roles', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  let modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(name)
  await expectMutation(page, '/api/admin/role', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/roles', name)
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑权限', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(editedName)
  await expectMutation(page, '/api/admin/role', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/roles', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/admin/roles', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await deleteTableRow(page, '/api/admin/roles', editedName)
  await expect(rowWithText(page, editedName)).toHaveCount(0)
}

async function runPermissionCRUD(page: Page, mode: 'menus' | 'resources', suffix: string): Promise<void> {
  const route = mode === 'menus' ? '/menus' : '/resources'
  const endpoint = mode === 'menus' ? '/api/admin/menus' : '/api/admin/resources'
  const shortSuffix = suffix.slice(-4)
  const name = mode === 'menus' ? `e2e-m-${shortSuffix}` : `e2e-r-${shortSuffix}`
  const editedName = mode === 'menus' ? `e2e-m2-${shortSuffix}` : `e2e-r2-${shortSuffix}`
  await page.goto(route, { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  let modal = visibleModal(page)
  const inputs = modal.locator('input[type="text"]')
  if (mode === 'menus') {
    await inputs.nth(0).fill(name)
    await inputs.nth(1).fill(`/e2e-menu-${suffix}`)
    await inputs.nth(2).fill('/article/ArticleList.vue')
  } else {
    await inputs.nth(0).fill(name)
    await inputs.nth(1).fill(`/admin/e2e-${suffix}`)
  }
  await expectMutation(page, endpoint, 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, endpoint, name)
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(editedName)
  await expectMutation(page, endpoint, 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, endpoint, editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, endpoint, editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await deletePermissionRow(page, endpoint, editedName)
  await expect(rowWithText(page, editedName)).toHaveCount(0)
}

async function runUserEditRoundTrip(page: Page, suffix: string): Promise<void> {
  await page.goto('/users', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  const rows = page.locator('.arco-table-tbody .arco-table-tr')
  if (await rows.count() === 0) return
  const row = rows.first()
  await row.getByRole('button', { name: '编辑', exact: true }).click()
  const modal = visibleModal(page)
  const input = modal.locator('input').first()
  const original = await input.inputValue()
  const changed = `${original || 'admin'}-${suffix}`
  await input.fill(changed)
  await expectMutation(page, '/api/admin/users/role', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/admin/users', changed)
  await expect(rowWithText(page, changed)).toBeVisible()
  await rowWithText(page, changed).getByRole('button', { name: '编辑', exact: true }).click()
  const restoreModal = visibleModal(page)
  await restoreModal.locator('input').first().fill(original)
  await expectMutation(page, '/api/admin/users/role', 'PUT', () => restoreModal.getByRole('button', { name: '确定', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
}

async function runCommentReviewRoundTrip(page: Page): Promise<void> {
  await page.goto('/comments', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  const rows = page.locator('.arco-table-tbody .arco-table-tr')
  if (await rows.count() === 0) return
  const row = rows.first()
  const action = row.getByRole('button').first()
  const originalAction = await action.innerText()
  await expectMutation(page, '/api/admin/comments/review', 'PUT', () => action.click())
  await expectMutation(page, '/api/admin/comments/review', 'PUT', () => row.getByRole('button').first().click())
  await expect(row.getByRole('button').first()).toHaveText(originalAction)
}

async function runWebsiteRoundTrip(page: Page, suffix: string): Promise<void> {
  await page.goto('/website', { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  await expect(main.getByText('网站配置')).toBeVisible()
  const name = main.locator('input').first()
  const original = await name.inputValue()
  await name.fill(`${original || 'Benetnasch'}-${suffix}`)
  await expectMutation(page, '/api/admin/website/config', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(name).toHaveValue(`${original || 'Benetnasch'}-${suffix}`)
  await name.fill(original)
  await expectMutation(page, '/api/admin/website/config', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(name).toHaveValue(original)
}

async function runAboutRoundTrip(page: Page, suffix: string): Promise<void> {
  await page.goto('/about', { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  const textarea = main.locator('textarea')
  const original = await textarea.inputValue()
  const changed = `${original}\n\n[e2e-${suffix}]`
  await textarea.fill(changed)
  await expectMutation(page, '/api/admin/about', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(textarea).toHaveValue(changed)
  await textarea.fill(original)
  await expectMutation(page, '/api/admin/about', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(textarea).toHaveValue(original)
}

async function runSettingRoundTrip(page: Page, suffix: string): Promise<void> {
  await page.goto('/setting', { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  const inputs = main.locator('input')
  const textarea = main.locator('textarea')
  const originalNickname = await inputs.nth(0).inputValue()
  const originalWebsite = await inputs.nth(1).inputValue()
  const originalIntro = await textarea.inputValue()
  await inputs.nth(0).fill(`${originalNickname || 'admin'}-${suffix}`)
  await textarea.fill(`${originalIntro}\n[e2e-${suffix}]`)
  await inputs.nth(1).fill(originalWebsite)
  await expectMutation(page, '/api/users/info', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(main.locator('input').nth(0)).toHaveValue(`${originalNickname || 'admin'}-${suffix}`)
  await main.locator('input').nth(0).fill(originalNickname)
  await main.locator('textarea').fill(originalIntro)
  await main.locator('input').nth(1).fill(originalWebsite)
  await expectMutation(page, '/api/users/info', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(main.locator('input').nth(0)).toHaveValue(originalNickname)
  await expect(main.locator('textarea')).toHaveValue(originalIntro)
}

async function runPhotoCRUD(page: Page, suffix: string): Promise<void> {
  // The isolated seed contains album 11 with reversible photo fixtures.  Do
  // not upload a new object here: the photo delete API removes the database
  // row but deliberately does not own object-store garbage collection.
  const albumID = 11
  await page.goto(`/albums/${albumID}`, { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  // Arco's table body classes differ between builds; use the accessible row
  // contract already used by the other CRUD steps.
  const row = page.getByRole('row').filter({ has: page.getByRole('button', { name: '编辑', exact: true }) }).first()
  await expect(row, 'isolated photo fixture').toBeVisible()
  const originalName = (await row.getByRole('cell').nth(1).innerText()).trim()
  expect(originalName).not.toBe('')
  const editedName = `e2e-p-${suffix.slice(-8)}`

  await row.getByRole('button', { name: '编辑', exact: true }).click()
  let modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(editedName)
  await expectMutation(page, '/api/admin/photos', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await expect(rowWithText(page, editedName)).toBeVisible()

  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await expect(rowWithText(page, editedName)).toBeVisible()

  const editedRow = rowWithText(page, editedName)
  await editedRow.getByRole('button', { name: '移除', exact: true }).click()
  const removeConfirm = page.locator('.arco-popconfirm:visible').last()
  await expectMutation(page, '/api/admin/photos/delete', 'PUT', () => removeConfirm.getByRole('button', { name: '确定', exact: true }).click())
  await expect(rowWithText(page, editedName)).toHaveCount(0)

  await page.goto('/photos/delete', { waitUntil: 'domcontentloaded' })
  await expect(page.getByRole('main').getByText('照片回收站')).toBeVisible()
  await expect(rowWithText(page, editedName)).toBeVisible()
  const deletedRow = rowWithText(page, editedName)
  await expectMutation(page, '/api/admin/photos/delete', 'PUT', () => deletedRow.getByRole('button', { name: '恢复', exact: true }).click())

  await page.goto(`/albums/${albumID}`, { waitUntil: 'domcontentloaded' })
  await expect(rowWithText(page, editedName)).toBeVisible()
  await rowWithText(page, editedName).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(originalName)
  await expectMutation(page, '/api/admin/photos', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await expect(rowWithText(page, originalName)).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(rowWithText(page, originalName)).toBeVisible()
}

async function runLogReadOnlyRoundTrips(page: Page): Promise<void> {
  await inspectLogPage(page, '/operation/log', '操作日志')
  await inspectLogPage(page, '/exception/log', '异常日志')
  await inspectLogPage(page, '/quartz/log/85', '任务日志')
}

async function inspectLogPage(page: Page, path: string, marker: string): Promise<void> {
  await page.goto(path, { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  await expect(main).toContainText(marker)
  await expect(main.locator('.arco-table')).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(main).toContainText(marker)
  await expect(main.locator('.arco-table')).toBeVisible()

  // Logs are append-only audit data in this flow. Inspecting the first row
  // exercises the detail path without deleting seeded or unrelated records.
  const rows = page.getByRole('row').filter({
    has: page.getByRole('button', { name: '详情', exact: true })
  })
  if (await rows.count() === 0) return
  await rows.first().getByRole('button', { name: '详情', exact: true }).click()
  const modal = visibleModal(page)
  await expect(modal).toContainText('日志详情')
  await page.keyboard.press('Escape')
  await expect(modal).toBeHidden()
}

async function runProviderProbeDisabled(page: Page, token: string): Promise<void> {
  const response = await page.context().request.post('/api/admin/ai/providers/test', {
    headers: { Authorization: `Bearer ${token}` },
    data: { useCase: 'vision' }
  })
  expect(response.status(), 'POST /api/admin/ai/providers/test').toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number; message?: string }
  expect(payload.flag, 'provider probe is disabled by default').toBe(false)
  expect(payload.code, 'provider probe disabled code').toBe(51000)
  expect(payload.message, 'provider probe disabled message').not.toContain('api_key')
  expect(payload.message, 'provider probe disabled message').not.toContain('secret')
}

async function runAgentProfileRoundTrip(page: Page, suffix: string): Promise<void> {
  const profileResponse = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === '/api/admin/ai/profile' && response.request().method() === 'GET'
  })
  await page.goto('/ai-profile', { waitUntil: 'domcontentloaded' })
  const response = await profileResponse
  expect(response.status(), 'GET /api/admin/ai/profile').toBe(200)
  const main = page.getByRole('main')
  const inputs = main.locator('input')
  const textareas = main.locator('textarea')
  await expect(inputs.nth(0)).toHaveValue(/.+/)
  const originalName = await inputs.nth(0).inputValue()
  await inputs.nth(0).fill(`${originalName}-${suffix}`)
  await expectMutation(page, '/api/admin/ai/profile', 'PATCH', () => main.getByRole('button', { name: '保存人设', exact: true }).click())
  await reloadAgentProfile(page, `${originalName}-${suffix}`)
  await main.locator('input').nth(0).fill(originalName)
  await expectMutation(page, '/api/admin/ai/profile', 'PATCH', () => main.getByRole('button', { name: '保存人设', exact: true }).click())
  await reloadAgentProfile(page, originalName)
  expect(await textareas.count()).toBeGreaterThan(0)
}

async function reloadAgentProfile(page: Page, expectedName: string): Promise<void> {
  const profileResponse = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === '/api/admin/ai/profile' && response.request().method() === 'GET'
  })
  await page.reload({ waitUntil: 'domcontentloaded' })
  const response = await profileResponse
  expect(response.status(), 'GET /api/admin/ai/profile').toBe(200)
  const main = page.getByRole('main')
  await expect(main.locator('input').nth(0)).toHaveValue(expectedName)
  await expect(main.getByRole('button', { name: '保存人设', exact: true })).toBeEnabled()
}

async function runAgentReviewPolicyRoundTrip(page: Page): Promise<void> {
  await page.goto('/ai-review-policy', { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  const input = main.locator('input').nth(1)
  const original = await input.inputValue()
  const next = String(Number(original || '168') + 1)
  await input.fill(next)
  await expectMutation(page, '/api/admin/ai/review-policy', 'PATCH', () => main.getByRole('button', { name: '保存策略', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(main.locator('input').nth(1)).toHaveValue(next)
  await main.locator('input').nth(1).fill(original)
  await expectMutation(page, '/api/admin/ai/review-policy', 'PATCH', () => main.getByRole('button', { name: '保存策略', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(main.locator('input').nth(1)).toHaveValue(original)
}

async function hasVisibleMenu(page: Page, label: string): Promise<boolean> {
  const item = page.locator('.arco-menu').getByText(label, { exact: false }).first()
  return await item.count() > 0 && await item.isVisible()
}

async function selectOption(page: Page, select: ReturnType<Page['locator']>, label: string): Promise<void> {
  await select.click()
  await page.locator('.arco-select-option').filter({ hasText: label }).getByText(label, { exact: true }).click()
}

async function expectMutation(page: Page, path: string, method: string, action: () => Promise<void> | void): Promise<void> {
  const response = await waitForMutation(page, path, method, action)
  const payload = await response.json() as { flag?: boolean; code?: number }
  expect(payload.flag, `${method} ${path} flag`).toBe(true)
  expect(payload.code, `${method} ${path} code`).toBe(20000)
}

async function waitForMutation(page: Page, path: string, method: string, action: () => Promise<void> | void): Promise<Awaited<ReturnType<Page['waitForResponse']>>> {
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === path && response.request().method() === method
  })
  await action()
  const response = await responsePromise
  expect(response.status(), `${method} ${path}`).toBe(200)
  return response
}

async function filterTable(page: Page, endpoint: string, value: string, queryParameter = 'keywords'): Promise<Awaited<ReturnType<Page['waitForResponse']>>> {
  const search = page.locator('.arco-input-search').getByRole('textbox')
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === endpoint && response.request().method() === 'GET' && responseURL.searchParams.get(queryParameter) === value
  })
  await search.fill(value)
  await page.locator('.arco-input-search .arco-icon-hover:not(.arco-input-clear-btn)').click()
  const response = await responsePromise
  expect(response.status(), `GET ${endpoint} filtered`).toBe(200)
  return response
}

async function deleteTableRow(page: Page, endpoint: string, text: string): Promise<void> {
  const row = rowWithText(page, text)
  await expect(row).toBeVisible()
  await row.getByRole('button', { name: /删除|移除/, exact: true }).click()
  const popconfirm = page.locator('.arco-popconfirm:visible').last()
  await expectMutation(page, endpoint, 'DELETE', () => popconfirm.getByRole('button', { name: '确定', exact: true }).click())
}

async function deletePermissionRow(page: Page, endpoint: string, text: string): Promise<void> {
  const row = rowWithText(page, text)
  await expect(row).toBeVisible()
  await row.getByRole('button', { name: '删除', exact: true }).click()
  const popconfirm = page.locator('.arco-popconfirm:visible').last()
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname.startsWith(`${endpoint}/`) && response.request().method() === 'DELETE'
  })
  await popconfirm.getByRole('button', { name: '确定', exact: true }).click()
  const response = await responsePromise
  expect(response.status(), `DELETE ${endpoint}/:id`).toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number }
  expect(payload.flag, `DELETE ${endpoint}/:id flag`).toBe(true)
  expect(payload.code, `DELETE ${endpoint}/:id code`).toBe(20000)
}

async function deleteAlbumRow(page: Page, text: string): Promise<void> {
  const row = rowWithText(page, text)
  await expect(row).toBeVisible()
  await row.getByRole('button', { name: '删除', exact: true }).click()
  const popconfirm = page.locator('.arco-popconfirm:visible').last()
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname.startsWith('/api/admin/photos/albums/') && response.request().method() === 'DELETE'
  })
  await popconfirm.getByRole('button', { name: '确定', exact: true }).click()
  const response = await responsePromise
  expect(response.status(), 'DELETE /api/admin/photos/albums/:id').toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number }
  expect(payload.flag).toBe(true)
  expect(payload.code).toBe(20000)
}

function rowWithText(page: Page, text: string): ReturnType<Page['getByRole']> {
  return page.getByRole('row').filter({ hasText: text }).first()
}

function visibleModal(page: Page): ReturnType<Page['locator']> {
  return page.locator('.arco-modal:visible').last()
}

function firstRecordID(payload: unknown): number {
  if (!payload || typeof payload !== 'object') return 0
  const data = (payload as { data?: unknown }).data
  if (!data || typeof data !== 'object') return 0
  const records = (data as { records?: unknown }).records
  if (!Array.isArray(records) || !records[0] || typeof records[0] !== 'object') return 0
  return Number((records[0] as { id?: unknown }).id || 0)
}

async function deleteAdminIDs(request: APIRequestContext, token: string, path: string, ids: number[]): Promise<void> {
  const response = await request.delete(path, {
    headers: { Authorization: `Bearer ${token}` },
    data: ids
  })
  expect(response.status(), `DELETE ${path} cleanup`).toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number }
  expect(payload.flag, `DELETE ${path} cleanup flag`).toBe(true)
  expect(payload.code, `DELETE ${path} cleanup code`).toBe(20000)
}
