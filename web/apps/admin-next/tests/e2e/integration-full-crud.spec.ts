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
    // Seeded articles still point at the retired OSS bucket, so the media proxy
    // answers 502 for those covers.  That is an environment artifact, not a
    // regression; the read-only integration suite ignores it the same way.
    expect(apiFailures.filter((failure) => !failure.startsWith('502 GET /api/v1/public/media/proxy'))).toEqual([])
    expect(pageErrors.filter((error) => !error.includes('status of 502 (Bad Gateway)'))).toEqual([])
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

/**
 * The editor opens in rich-text mode, where the body lives in a contenteditable
 * surface rather than a textarea.  These CRUD round-trips only need a plain text
 * body, so switch to the Markdown/HTML source mode first.
 */
async function useSourceMode(form: ReturnType<Page['locator']>): Promise<void> {
  // Arco renders the button-style radio as a visually hidden input, so click the
  // visible label text instead of the input itself.
  await form.locator('.article-editor-head').getByText('源码', { exact: true }).click()
  await expect(form.locator('textarea')).toBeVisible()
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
    await useSourceMode(form)
    await form.locator('textarea').fill(`隔离文章正文 ${suffix}`)
    await selectOption(page, formSelectByLabel(page, '状态'), '草稿')
    await expectMutation(page, '/api/v1/admin/articles', 'POST', () => form.getByRole('button', { name: '保存', exact: true }).click())
    await expect(page).toHaveURL(/\/article-list$/)

    let searchResponse = await filterTable(page, '/api/v1/admin/articles', title)
    articleID = firstRecordID(await searchResponse.json())
    expect(articleID, 'created draft article id').toBeGreaterThan(0)
    await expect(rowWithText(page, title)).toBeVisible()

    await page.goto(`/articles/${articleID}`, { waitUntil: 'domcontentloaded' })
    await expect(page.locator('.article-form')).toBeVisible()
    await page.locator('.article-form input').nth(0).fill(editedTitle)
    await useSourceMode(page.locator('.article-form'))
    await page.locator('.article-form textarea').fill(`隔离文章正文已编辑 ${suffix}`)
    await expectMutation(page, '/api/v1/admin/articles', 'POST', () => page.locator('.article-form').getByRole('button', { name: '保存', exact: true }).click())
    await expect(page).toHaveURL(/\/article-list$/)
    searchResponse = await filterTable(page, '/api/v1/admin/articles', editedTitle)
    expect(firstRecordID(await searchResponse.json())).toBe(articleID)
    await expect(rowWithText(page, editedTitle)).toBeVisible()
    await page.reload({ waitUntil: 'domcontentloaded' })
    await filterTable(page, '/api/v1/admin/articles', editedTitle)
    await expect(rowWithText(page, editedTitle)).toBeVisible()
  } finally {
    if (articleID > 0) await deleteAdminIDs(page.context().request, token, '/api/v1/admin/articles/batch-delete', [articleID])
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
  await expectMutation(page, '/api/v1/admin/albums', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/albums', name)
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').nth(0).fill(editedName)
  await expectMutation(page, '/api/v1/admin/albums', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/albums', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/v1/admin/albums', editedName)
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
  await inputs.nth(2).fill('article.cleanup')
  await inputs.nth(3).fill('0 0 1 1 1 ?')
  await expectMutation(page, '/api/v1/admin/jobs', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/jobs', name, 'jobName')
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await expect(modal).toContainText('编辑任务')
  inputs = modal.locator('input[type="text"]')
  await inputs.nth(0).fill(editedName)
  await expectMutation(page, '/api/v1/admin/jobs', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/jobs', editedName, 'jobName')
  await expect(rowWithText(page, editedName)).toBeVisible()

  const row = rowWithText(page, editedName)
  const toggle = row.locator('.arco-switch')
  if (await toggle.count() > 0) {
    await expectMutation(page, '/api/v1/admin/jobs/status', 'PUT', () => toggle.click())
    await expectMutation(page, '/api/v1/admin/jobs/status', 'PUT', () => toggle.click())
  }
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/v1/admin/jobs', editedName, 'jobName')
  await expect(rowWithText(page, editedName)).toBeVisible()
  await deleteTableRow(page, '/api/v1/admin/jobs', editedName)
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
  await expectMutation(page, '/api/v1/admin/roles', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/roles', name)
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑权限', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(editedName)
  await expectMutation(page, '/api/v1/admin/roles', 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/roles', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await page.reload({ waitUntil: 'domcontentloaded' })
  await filterTable(page, '/api/v1/admin/roles', editedName)
  await expect(rowWithText(page, editedName)).toBeVisible()
  await deleteTableRow(page, '/api/v1/admin/roles', editedName)
  await expect(rowWithText(page, editedName)).toHaveCount(0)
}

async function runPermissionCRUD(page: Page, mode: 'menus' | 'resources', suffix: string): Promise<void> {
  const route = mode === 'menus' ? '/menus' : '/resources'
  const endpoint = mode === 'menus' ? '/api/v1/admin/menus' : '/api/v1/admin/permissions'
  const shortSuffix = suffix.slice(-4)
  const name = mode === 'menus' ? `e2e-m-${shortSuffix}` : `e2e-r-${shortSuffix}`
  const editedName = mode === 'menus' ? `e2e-m2-${shortSuffix}` : `e2e-r2-${shortSuffix}`
  await page.goto(route, { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.arco-table')).toBeVisible()
  await page.getByRole('button', { name: '新增', exact: true }).click()
  let modal = visibleModal(page)
  if (mode === 'menus') {
    await fillFieldByLabel(page, modal, '菜单名称', name)
    await fillFieldByLabel(page, modal, '路径', `/e2e-menu-${suffix}`)
    await fillFieldByLabel(page, modal, '组件路径', '/article/ArticleList.vue')
  } else {
    await fillFieldByLabel(page, modal, '资源名称', name)
    await fillFieldByLabel(page, modal, 'URL', `/admin/e2e-${suffix}`)
  }
  await expectMutation(page, endpoint, 'POST', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, endpoint, name)
  await expect(rowWithText(page, name)).toBeVisible()

  await rowWithText(page, name).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await fillFieldByLabel(page, modal, mode === 'menus' ? '菜单名称' : '资源名称', editedName)
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
  await expectMutation(page, '/api/v1/admin/users/roles', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await filterTable(page, '/api/v1/admin/users', changed)
  await expect(rowWithText(page, changed)).toBeVisible()
  await rowWithText(page, changed).getByRole('button', { name: '编辑', exact: true }).click()
  const restoreModal = visibleModal(page)
  await restoreModal.locator('input').first().fill(original)
  await expectMutation(page, '/api/v1/admin/users/roles', 'PUT', () => restoreModal.getByRole('button', { name: '确定', exact: true }).click())
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
  await expectMutation(page, '/api/v1/admin/comments/review', 'PUT', () => action.click())
  await expectMutation(page, '/api/v1/admin/comments/review', 'PUT', () => row.getByRole('button').first().click())
  await expect(row.getByRole('button').first()).toHaveText(originalAction)
}

async function runWebsiteRoundTrip(page: Page, suffix: string): Promise<void> {
  await page.goto('/website', { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  await expect(main.getByText('网站配置')).toBeVisible()
  const name = main.locator('input').first()
  const original = await name.inputValue()
  await name.fill(`${original || '星际信标'}-${suffix}`)
  await expectMutation(page, '/api/v1/admin/site', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(name).toHaveValue(`${original || '星际信标'}-${suffix}`)
  await name.fill(original)
  await expectMutation(page, '/api/v1/admin/site', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
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
  await expectMutation(page, '/api/v1/admin/about', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(textarea).toHaveValue(changed)
  await textarea.fill(original)
  await expectMutation(page, '/api/v1/admin/about', 'PUT', () => main.getByRole('button', { name: '保存', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(textarea).toHaveValue(original)
}

async function runSettingRoundTrip(page: Page, suffix: string): Promise<void> {
  await page.goto('/setting', { waitUntil: 'domcontentloaded' })
  const main = page.getByRole('main')
  // The avatar uploader contributes a hidden file input; only text controls count.
  const inputs = main.locator('input:not([type="file"])')
  const textarea = main.locator('textarea')
  const originalNickname = await inputs.nth(0).inputValue()
  const originalWebsite = await inputs.nth(1).inputValue()
  const originalIntro = await textarea.inputValue()
  // The nickname field is capped at 30 characters, so trim the base before
  // appending the run suffix instead of relying on the browser truncation.
  const changedNickname = `${(originalNickname || 'admin').slice(0, 30 - suffix.length - 1)}-${suffix}`
  await inputs.nth(0).fill(changedNickname)
  await textarea.fill(`${originalIntro}\n[e2e-${suffix}]`)
  await inputs.nth(1).fill(originalWebsite)
  await expectMutation(page, '/api/v1/auth/me', 'PUT', () => main.getByRole('button', { name: '保存资料', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(inputs.nth(0)).toHaveValue(changedNickname)
  await inputs.nth(0).fill(originalNickname)
  await textarea.fill(originalIntro)
  await inputs.nth(1).fill(originalWebsite)
  await expectMutation(page, '/api/v1/auth/me', 'PUT', () => main.getByRole('button', { name: '保存资料', exact: true }).click())
  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(inputs.nth(0)).toHaveValue(originalNickname)
  await expect(textarea).toHaveValue(originalIntro)
}

async function runPhotoCRUD(page: Page, suffix: string): Promise<void> {
  // The isolated seed contains album 11 with reversible photo fixtures.  Do
  // not upload a new object here: the photo delete API removes the database
  // row but deliberately does not own object-store garbage collection.
  const albumID = 11
  await page.goto(`/albums/${albumID}`, { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.photo-masonry')).toBeVisible()
  // Arco's table body classes differ between builds; use the accessible row
  // contract already used by the other CRUD steps.
  const row = page.getByRole('row').filter({ has: page.getByRole('button', { name: '编辑', exact: true }) }).first()
  await expect(row, 'isolated photo fixture').toBeVisible()
  const originalName = (await row.locator('.photo-tile-copy strong').innerText()).trim()
  expect(originalName).not.toBe('')
  const editedName = `e2e-p-${suffix.slice(-8)}`

  await row.getByRole('button', { name: '编辑', exact: true }).click()
  let modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(editedName)
  await expectMutation(page, '/api/v1/admin/photos', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
  await expect(rowWithText(page, editedName)).toBeVisible()

  await page.reload({ waitUntil: 'domcontentloaded' })
  await expect(page.locator('.photo-masonry')).toBeVisible()
  await expect(rowWithText(page, editedName)).toBeVisible()

  const editedRow = rowWithText(page, editedName)
  await editedRow.getByRole('button', { name: '移除', exact: true }).click()
  const removeConfirm = page.locator('.arco-popconfirm:visible').last()
  await expectMutation(page, '/api/v1/admin/photos/trash', 'PUT', () => removeConfirm.getByRole('button', { name: '确定', exact: true }).click())
  await expect(rowWithText(page, editedName)).toHaveCount(0)

  await page.goto('/photos/delete', { waitUntil: 'domcontentloaded' })
  await expect(page.getByRole('main').getByText('照片回收站')).toBeVisible()
  await expect(rowWithText(page, editedName)).toBeVisible()
  const deletedRow = rowWithText(page, editedName)
  await expectMutation(page, '/api/v1/admin/photos/trash', 'PUT', () => deletedRow.getByRole('button', { name: '恢复', exact: true }).click())

  await page.goto(`/albums/${albumID}`, { waitUntil: 'domcontentloaded' })
  await expect(rowWithText(page, editedName)).toBeVisible()
  await rowWithText(page, editedName).getByRole('button', { name: '编辑', exact: true }).click()
  modal = visibleModal(page)
  await modal.locator('input[type="text"]').first().fill(originalName)
  await expectMutation(page, '/api/v1/admin/photos', 'PUT', () => modal.getByRole('button', { name: '确定', exact: true }).click())
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

/** Fill a text control inside a dialog by its field label, not by position. */
async function fillFieldByLabel(page: Page, modal: ReturnType<Page['locator']>, label: string, value: string): Promise<void> {
  await modal.locator('.arco-form-item')
    .filter({ has: page.locator('.arco-form-item-label', { hasText: label }) })
    .first()
    .locator('input')
    .first()
    .fill(value)
}

/**
 * The article editor renders several selects (category, tags, status, type), so
 * positional lookups are unstable; resolve the control through its field label.
 */
function formSelectByLabel(page: Page, label: string): ReturnType<Page['locator']> {
  return page.locator('.article-form .arco-form-item')
    .filter({ has: page.locator('.arco-form-item-label', { hasText: label }) })
    .locator('.arco-select')
    .first()
}

async function selectOption(page: Page, select: ReturnType<Page['locator']>, label: string): Promise<void> {
  await select.click()
  // Every select keeps an (hidden) dropdown in the DOM, so the option has to be
  // resolved inside the dropdown that is actually open.
  await page.locator('.arco-select-option:visible').filter({ hasText: label }).first().click()
}

async function expectMutation(page: Page, path: string, method: string, action: () => Promise<void> | void): Promise<void> {
  const response = await waitForMutation(page, path, method, action)
  const payload = await response.json() as { flag?: boolean; code?: number | string }
  expect(payload.code === 'OK' || payload.flag === true, `${method} ${path} success`).toBe(true)
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
  const payload = await response.json() as { flag?: boolean; code?: number | string }
  expect(payload.code === 'OK' || payload.flag === true, `DELETE ${endpoint}/:id success`).toBe(true)
}

async function deleteAlbumRow(page: Page, text: string): Promise<void> {
  const row = rowWithText(page, text)
  await expect(row).toBeVisible()
  await row.getByRole('button', { name: '删除', exact: true }).click()
  const popconfirm = page.locator('.arco-popconfirm:visible').last()
  const responsePromise = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname.startsWith('/api/v1/admin/albums/') && response.request().method() === 'DELETE'
  })
  await popconfirm.getByRole('button', { name: '确定', exact: true }).click()
  const response = await responsePromise
  expect(response.status(), 'DELETE /api/v1/admin/albums/:id').toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number | string }
  expect(payload.code === 'OK' || payload.flag === true).toBe(true)
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
  const records = Array.isArray((data as { items?: unknown }).items)
    ? (data as { items: unknown }).items
    : (data as { records?: unknown }).records
  if (!Array.isArray(records) || !records[0] || typeof records[0] !== 'object') return 0
  return Number((records[0] as { id?: unknown }).id || 0)
}

async function deleteAdminIDs(request: APIRequestContext, token: string, path: string, ids: number[]): Promise<void> {
  const response = await request.delete(path, {
    headers: { Authorization: `Bearer ${token}` },
    data: ids
  })
  expect(response.status(), `DELETE ${path} cleanup`).toBe(200)
  const payload = await response.json() as { flag?: boolean; code?: number | string }
  expect(payload.code === 'OK' || payload.flag === true, `DELETE ${path} cleanup success`).toBe(true)
}
