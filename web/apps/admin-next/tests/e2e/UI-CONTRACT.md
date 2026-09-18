# admin-next Playwright UI Contract (restyle-safe)

Scope: the six spec/config files below, all under `web/apps/admin-next`.
Every assertion below is quoted from those files with line numbers. Findings marked **[verified]** were
checked against the live app (Arco Design Vue 2.58.0, Chromium) during this analysis; the app was not modified.

Sources:

| File | Lines |
| --- | --- |
| `tests/e2e/admin-shell.spec.ts` | 939 |
| `tests/e2e/integration-crud.spec.ts` | 255 |
| `tests/e2e/integration-full-crud.spec.ts` | 506 |
| `tests/e2e/integration-readonly.spec.ts` | 243 |
| `tests/e2e/integration.spec.ts` | 60 |
| `playwright.config.ts` | 34 |

Two suites are always-on (baseline, mocked API via `page.route('**/*')`, admin-shell.spec.ts) and four are
gated behind env vars (`E2E_REAL_INTEGRATION=1` plus `E2E_ADMIN_ALLOW_LOGIN`/`E2E_ADMIN_EMAIL`/`E2E_ADMIN_PASSWORD`
or `E2E_ADMIN_TOKEN`) and self-skip otherwise (integration-crud.spec.ts:9-12, integration-full-crud.spec.ts:11-14,
integration-readonly.spec.ts:41-44 and 130-133, integration.spec.ts:8-11). A redesign must satisfy all of them,
including the gated ones, because they encode the same DOM contract.

Legend: **[FRAGILE]** = depends on exact ordinal position or on being the only/first match.

---

## 0. Global invariants (apply to every view)

These are asserted repeatedly and are the backbone of the contract.

| Contract item | Exact selector / assertion | Where |
| --- | --- | --- |
| Shell root | `page.locator('.admin-shell')` visible | admin-shell.spec.ts:899, 905; integration-crud.spec.ts:64; integration-full-crud.spec.ts:66; integration.spec.ts:28; integration-readonly.spec.ts:217 |
| Content root | `page.locator('.admin-content')` visible | admin-shell.spec.ts:900, 906; integration.spec.ts:51, 54; integration-readonly.spec.ts:218 |
| Content root must have a rendered child | `.admin-content` then `content.locator(':scope > *').first()` visible | integration-readonly.spec.ts:220 |
| Content root must not show error/placeholder copy | `.admin-content` `not.toContainText('页面不存在')` | admin-shell.spec.ts:901, 907; integration-crud.spec.ts (implicit), integration.spec.ts:52, 55; integration-readonly.spec.ts:222 |
| | `.admin-content` `not.toContainText('无权访问')` | integration-readonly.spec.ts:223 |
| | `.admin-content` `not.toContainText('模块迁移中')` | integration-readonly.spec.ts:224 |
| `main` landmark must exist | `page.getByRole('main')` used as the scope for almost every page assertion | admin-shell.spec.ts:617, 622, 625, 628, 632, 639, 651, 652, 672, 700, 717, 724, 727, 734, 749, 752, 773, 779, 792, 805, 824-826; integration-full-crud.ts:275, 291, 307, 360, 384; integration.spec.ts:29 |
| Sidebar root | `page.locator('.admin-sider')` visible + used as click scope for menu items | admin-shell.spec.ts:618, 620, 623, 864 |
| Sidebar must be an Arco sider | `page.locator('.arco-layout-sider').first()` used as menu scope | integration-readonly.spec.ts:97, 105 |
| Menu item must be `.arco-menu-item` | `sidebar.locator('.arco-menu-item').filter({ hasText: name })` must have count 1 | integration-readonly.spec.ts:105-106 |
| Table root | `page.locator('.arco-table')` visible | admin-shell.spec.ts:746; integration-crud.spec.ts:69, 94; integration-full-crud.spec.ts:113, 141, 181, 209, 240, 262, 334, 350, 386, 389; integration-readonly.spec.ts:112 |
| Rows are accessible rows | `page.getByRole('row')` (+ `.filter({ hasText })`) | admin-shell.spec.ts:719, 754; integration-crud.spec.ts:156; integration-full-crud.spec.ts:337, 393, 480 |
| Cells are accessible cells | `row.getByRole('cell').nth(1)` | integration-full-crud.spec.ts:339 |
| Table body class (used only for `.count()`, not for interaction) | `.arco-table-tbody .arco-table-tr` | integration-full-crud.spec.ts:241, 263 |
| Modal root | `page.locator('.arco-modal:visible')` | see per-view tables |
| Popconfirm root | `page.locator('.arco-popconfirm:visible')` | admin-shell.spec.ts:675, 703, 775; integration-crud.spec.ts:163, 203, 224, 244; integration-full-crud.spec.ts:355, 443, 451, 467 |
| Search input | `.arco-input-search` wrapper containing `getByRole('textbox')` | integration-crud.spec.ts:168; integration-full-crud.spec.ts:427 |
| Search trigger icon | `.arco-input-search .arco-icon-hover:not(.arco-input-clear-btn)` — exactly 1 match **[verified: count = 1]** | integration-crud.spec.ts:178; integration-full-crud.spec.ts:433 |
| URL shape | `toHaveURL` exact path regexes per route, incl. `/\/articles$/`, `/\/articles\/42$/`, `/\/tags$/`, `/\/quartz\/log\/\d+$/`, `/\/albums\/5$/`, `/\/talks\/7$/`, `/\/photos\/delete$/`, `/\/online\/users$/`, `/\/website$/`, `/\/setting$/` | admin-shell.spec.ts:616, 621, 624, 627, 631, 638, 650, 654, 658, 664, 671, 689, 696, 699, 706, 716, 726, 730, 733, 736, 745, 748, 751, 772, 778, 791, 804, 823 |
| Route render check | `expect(page).toHaveURL((url) => url.pathname === expectedPath)` | integration-readonly.spec.ts:216 |
| HTTP 200 on navigation and refresh | `expect(response?.status()).toBe(200)` for every route and every `page.reload()` | admin-shell.spec.ts:898, 904, 914-915; integration-readonly.spec.ts:83, 173, 215; integration.spec.ts:50 |

**Why `<main class="admin-content">` matters** **[verified]**: the app renders `<main class="admin-content">`
inside `<section class="admin-shell">` (Arco `a-layout` renders `SECTION`), and the `.admin-sider` is a `DIV`.
`getByRole('main')` resolves to exactly 1 node only because the tag stays `<main>`. Replacing it with a `<div>`
(keeping the class) silently breaks ~30 assertions with "resolved to 0 elements".

**Why `getByRole('row')` matters** **[verified]**: Arco's `tr`/`td` carry no explicit `role` attribute, but they
are native `tr`/`td` inside a native `table`, so Playwright infers `row`, `cell`, `table`. A card/div/grid-based
table redesign breaks every `getByRole('row')` and `row.getByRole('cell')` assertion even if the visual result is
identical.

**Checkbox select pattern** **[FRAGILE]**: admin-shell.spec.ts:719 is
`page.getByRole('row', { name: /首页截图/ }).getByRole('checkbox').locator('..').click()` — it clicks the *parent*
of the checkbox, not the checkbox. This requires (a) the row to contain exactly one accessible checkbox, and
(b) that checkbox's parent element to be a clickable label/wrapper. Arco's row-selection cell is
`.arco-table-checkbox` wrapping an `<input type="checkbox">`; wrapping the checkbox in extra elements or adding a
second checkbox (e.g. a per-row toggle) changes `locator('..')` semantics.

---

## 1. Per-view locked selectors

### 1.0 `/login` — LoginView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| `data-testid` | `login-username` — must be an ancestor of an `<input>`, matched with `.locator('input')` | 612, 842, 858 |
| `data-testid` | `login-password` — same | 613, 843, 859 |
| `data-testid` | `login-submit` — must be a `button` (clicked directly, no descendant selector) | 614, 844, 860 |
| Ordinal | `getByTestId('login-username').locator('input')` must resolve to exactly **1** input **[verified: count = 1]** | 612 |
| Fill values | `admin@example.com` / `password` (mocked flow); real env values in gated suites | 612-613 |
| Post-login | `await expect(page).toHaveURL(/\/$/)` then `.admin-shell` visible, or `page.waitForURL((url) => url.pathname === '/')` | 616, 846, 861; integration-crud.spec.ts:63-64; integration-full-crud.spec.ts:65-66; integration.spec.ts:27-28 |
| Storage | `sessionStorage.setItem('token', …)` pre-seeded by `page.addInitScript` for the 404 and read-only suites | 832; integration-readonly.spec.ts:78-80, 140-142 |
| Storage | after login, `sessionStorage.getItem('token')` must be non-empty (integration-full-crud.spec.ts:32-33) and `sessionStorage.getItem('stellar-beacon.admin.user')` must hold JSON whose `nickname` equals the saved value (admin-shell.spec.ts:817-821) |

**[verified]** live DOM: `[data-testid="login-username"]` → `SPAN.arco-input-wrapper` containing
`<input class="arco-input …" type="text" id="admin-username">`; `[data-testid="login-password"]` → same with
`type="password"`; `[data-testid="login-submit"]` → `BUTTON.arco-btn…`. So the testid must stay on the **wrapper**
(`a-input` root), never moved onto the inner `<input>` (that would make `.locator('input')` resolve to 0) and never
moved above a component that renders no `input` descendant.

`sessionStorage` keys are hard asserts, not implementation details: `token` (integration-full-crud.spec.ts:32,
integration-readonly.spec.ts:79) and `stellar-beacon.admin.user` with a `nickname` field (admin-shell.spec.ts:818-820).

### 1.1 `/` — home inside the shell

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Role+name **[FRAGILE]** | `page.getByRole('main').getByRole('button', { name: '发布文章', exact: true })` — must resolve to exactly 1 button and be visible | 617, 862 |
| Text scope | `page.locator('.admin-sider').getByText('内容管理', { exact: true })` visible | 618, 864 |
| Text absence | `page.getByText('无可见菜单', { exact: true })` → `toHaveCount(0)` | 619 |
| Text absence | `page.getByText('隐藏页面', { exact: true })` → `toHaveCount(0)` | 865 |
| Text absence | `page.getByText('仅用于权限测试', { exact: true })` → `toHaveCount(0)` | 866 |
| Main landmark text (integration.spec.ts only) | `home` (`getByRole('main')`) `toContainText` of: `账号状态`, `已认证`, `权限来源`, `RBAC` | integration.spec.ts:30-33 |

Note: the '发布文章' button on `/` is asserted inside `main`, and the menu item '发布文章' lives in the sider, so the
two never collide — **but only while the home button stays inside the `main` landmark**. If a redesign moves the
quick-action button into the topbar or the sider, `getByRole('main').getByRole('button', …)` resolves to 0.

### 1.2 `/article-list` — ArticleListView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click | `.admin-sider` `getByText('文章列表', { exact: true })` | 620 |
| Main text | `getByRole('main').getByText('文章列表')` visible (substring match, not exact) | 622 |
| Table | `.arco-table` visible | integration-readonly.spec.ts:112 (via `table: true`) |
| Marker text | `main` contains `文章列表` | integration-readonly.spec.ts:17, 111, 221 |

### 1.3 `/articles` — ArticleEditorView (create)

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Content selector | `page.locator('.article-form')` visible | integration-full-crud.spec.ts:75, 76, 92; integration-readonly.spec.ts:16, 114 |
| Marker | `main` contains `发布文章` | admin-shell.spec.ts:625; integration-readonly.spec.ts:16 |
| Input ordinals **[FRAGILE]** | `.article-form input` `nth(0)` = title, `nth(1)` = category select input, `nth(2)` = tag select input | integration-full-crud.spec.ts:77-80 |
| Textarea | `.article-form textarea` — first/only textarea = body | integration-full-crud.spec.ts:81 |
| Select | `form.locator('.arco-select').first()` → click, then `.arco-select-option` filtered by `草稿`, then `getByText('草稿', { exact: true })` | integration-full-crud.spec.ts:82, 404-407 |
| Save button | `form.getByRole('button', { name: '保存', exact: true })` | integration-full-crud.spec.ts:83, 95 |
| Post-save | URL `/article-list$` | integration-full-crud.spec.ts:84, 96 |

Note the hidden `<input type="file" hidden>` (ArticleEditorView.vue:41) is inside the form but is irrelevant to the
`input` ordinals above only because it is *last*; inserting a file input or any input before title shifts
`nth(0..2)`.

### 1.4 `/articles/42` — ArticleEditorView (edit)

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Main text | `getByRole('main').getByText('修改文章')` visible | 628; integration-readonly.spec.ts:182 |
| Input ordinal **[FRAGILE]** | `page.locator('input').first()` — *page-wide*, unscoped — must be the title input and must have value `已存在文章` | 629 |
| Content selector | `.article-form` visible | integration-readonly.spec.ts:182, 195 |
| Title locked | `page.locator('.article-form input').nth(0)` is title (edited in full-CRUD) | integration-full-crud.spec.ts:93 |

**[FRAGILE]** 629 is the strictest ordinal in the suite: `page.locator('input').first()` is page-wide, so the
*first* `<input>` in the entire document must be the article title. Adding any input to the topbar (search,
command palette, theme switcher rendered as input), or rendering an input before the form, breaks it.

### 1.5 `/categories` and `/tags` — TaxonomyView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Main text | `getByRole('main').getByText('工程化')` visible (mock row) | 632, 639, 652 |
| Main text | `getByRole('main').getByText('标签管理')` visible (after reload) | 651 |
| Sider clicks | `page.getByText('分类管理')` / `page.getByText('标签管理')` (unscoped, substring) | 630, 637 |
| Table | `.arco-table` visible | integration-crud.spec.ts:69 |
| Create button | `page.getByRole('button', { name: '新增' })` — **not** `exact` in admin-shell | 633, 645; integration-crud.spec.ts:70 uses `exact: true` |
| Dialog | `page.locator('.arco-modal:visible')` | 634, 641, 646 |
| Dialog input ordinal **[FRAGILE]** | `dialog.locator('input')` — singular, so exactly **one** input in the dialog; filled directly | 635, 643, 647 |
| Dialog text | `tagEditDialog` `toContainText('编辑')` (modal title is `编辑` for edit, `新增` for create) | 642 |
| Dialog confirm | `dialog.getByRole('button', { name: '确定' })` | 636, 644, 648 |
| Row-scoped edit | `tableRow(page, created).getByRole('button', { name: '编辑', exact: true })` | integration-crud.spec.ts:77 |
| Row-scoped delete | `row.getByRole('button', { name: '删除', exact: true })` → `.arco-popconfirm:visible` `.last()` → `getByRole('button', { name: '确定', exact: true })` | integration-crud.spec.ts:162-164, 202-208 |
| Row lookup | `page.getByRole('row').filter({ hasText: text }).first()` | integration-crud.spec.ts:156 |
| Dict input in dialog | `modal.getByRole('textbox').first()` | integration-crud.spec.ts:72, 79 |

`dialog.locator('input')` (singular `fill`, no `.first()`) is a **strict-mode** contract: if a redesign adds any
second input to the taxonomy dialog (search box, icon picker, sort field, hidden file), Playwright throws
"strict mode violation: resolved to 2 elements" at 635/643/647.

### 1.6 `/comments` — CommentsView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click | `page.getByText('评论管理')` | 653 |
| Unscoped text | `page.getByText('不错')` (unscoped, substring) visible | 655 |
| Button | `page.getByRole('button', { name: '通过审核' })` visible **[FRAGILE: must be the only match]** | 656 |
| Table | `.arco-table` visible | integration-full-crud.spec.ts:262 |
| First row action | `row.getByRole('button').first()` — clicks the **first button of the row**, whatever it is; text is captured and later compared | integration-full-crud.spec.ts:266-270 |
| Row source | `page.locator('.arco-table-tbody .arco-table-tr')` `.first()` | integration-full-crud.spec.ts:263 |

**[FRAGILE]** integration-full-crud.spec.ts:266-270 asserts that clicking the row's first button toggles review
(`PUT /api/v1/admin/comments/review`) and that the button label returns to its original text after two clicks
(`toHaveText(originalAction)`). A redesign that puts any non-review control first in the actions cell (view, reply,
delete) changes which endpoint is called and breaks this step. Also, the label must alternate between `通过审核`
and `取消审核` (admin-shell.spec.ts:656 requires `通过审核` to be visible for the mocked `isReview: 0` row).

### 1.7 `/users` — UsersView (and `/online/users` — OnlineUsersView)

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider clicks | `page.getByText('用户管理')`, `page.getByText('在线用户')` | 657, 747 |
| Unscoped text | `page.getByText('测试用户')` (unscoped, substring) visible | 659 |
| Unscoped text | `page.getByText('编辑者')` visible (role tag) | 665 |
| Edit button **[FRAGILE]** | `page.getByRole('button', { name: '编辑' })` — page-wide, **no row scope**; must be the only match | 660 |
| Dialog text | `page.locator('.arco-modal:visible')` `toContainText('修改用户')` | 661 |
| Confirm **[FRAGILE]** | `page.getByRole('button', { name: '确定' })` — **page-wide and unscoped by any dialog** | 662 |
| Table | `.arco-table` visible | integration-full-crud.spec.ts:240 |
| Row source | `.arco-table-tbody .arco-table-tr`, `.first()` | integration-full-crud.spec.ts:241, 243 |
| Row-scoped edit | `row.getByRole('button', { name: '编辑', exact: true })` | integration-full-crud.spec.ts:244, 253 |
| Dialog input ordinal **[FRAGILE]** | `modal.locator('input').first()` = nickname; value is read then restored | integration-full-crud.spec.ts:246-249, 255 |
| Main landmark | `getByRole('main').getByText('在线用户')` visible | 749 |

Two page-level locators here are the most fragile in the baseline suite:

* 660 `page.getByRole('button', { name: '编辑' })` — non-exact, page-wide. Any additional button whose accessible
  name *contains* 编辑 (e.g. `编辑权限`, `编辑友链`, a toolbar `编辑`) resolves to multiple elements and the click
  throws. Today the users table has exactly one data row with one such button.
* 662 `page.getByRole('button', { name: '确定' })` — page-wide, not scoped to `.arco-modal`. It works only because
  exactly one visible dialog (and no other 确定 button) exists at that moment. A redesign that keeps a second 确定
  (e.g. a batch-confirm bar) breaks it.

### 1.8 `/roles` — RoleView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('角色管理')`, `/roles$` | 663-664 |
| Unscoped text | `page.getByText('编辑者')` visible | 665 |
| Create button | `page.getByRole('button', { name: '新增' })` | 666 |
| Dialog | `page.locator('.arco-modal:visible')` | 667 |
| Text input ordinal **[FRAGILE]** | `roleDialog.locator('input[type="text"]').first()` = role name | 668 |
| Confirm | `roleDialog.getByRole('button', { name: '确定' })` | 669 |
| Table | `.arco-table` visible | integration-full-crud.spec.ts:181 |
| Row-scoped edit-permission button | `rowWithText(page, name).getByRole('button', { name: '编辑权限', exact: true })` | integration-full-crud.spec.ts:189 |
| Dialog input ordinal | `modal.locator('input[type="text"]').first()` = role name (again, on the edit path) | integration-full-crud.spec.ts:184, 191 |
| Confirm | `modal.getByRole('button', { name: '确定', exact: true })` | integration-full-crud.spec.ts:185, 192 |
| Row-scoped delete | `row.getByRole('button', { name: '删除', exact: true })` | integration-full-crud.spec.ts:442 (`/删除\|移除/`), 198 → `deleteTableRow` |

`input[type="text"]` is literal: **[verified]** Arco inputs render `type="text"` explicitly, so the selector works,
but any custom control that omits the attribute (or uses `type="search"`/`type="email"`) drops out of the ordinal
list and shifts `first()`/`nth(k)` onto the wrong field.

### 1.9 `/quartz` — JobsView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('定时任务')`, `/quartz$` | 670-671 |
| Heading role | `getByRole('main').getByRole('heading', { name: '定时任务' })` visible | 672 |
| Run-once button | `rowWithText(page, name).getByRole('button', { name: '执行一次', exact: true })` `toBeEnabled()` then clicked | 673-674 |
| Popconfirm | `page.locator('.arco-popconfirm:visible').getByRole('button', { name: '确定' })` | 675 |
| Create button | `page.getByRole('button', { name: '新增' })` | 677 |
| Dialog | `page.locator('.arco-modal:visible')` | 678 |
| Dialog fields | Label-scoped 任务名称 / 任务分组 / Cron 表达式 inputs; 调用目标 is an Arco select with a registered target such as `userArea.refresh` | 679-683 |
| Confirm | `jobDialog.getByRole('button', { name: '确定' })` | 684 |
| Edit button **[FRAGILE]** | `page.getByRole('button', { name: '编辑' })` — page-wide, must be the only match | 685 |
| Dialog text | `page.locator('.arco-modal:visible')` `toContainText('编辑任务')` | 686 |
| Table | `.arco-table` visible | integration-full-crud.spec.ts:141 |
| Row-scoped edit | `rowWithText(page, name).getByRole('button', { name: '编辑', exact: true })` | integration-full-crud.spec.ts:160 |
| Dialog text | `modal` `toContainText('编辑任务')` | integration-full-crud.spec.ts:155 |
| Manual run | Row-scoped 执行一次 waits for `PUT /api/v1/admin/jobs/run`, then verifies one `jobName`-filtered row from `/api/v1/admin/logs/jobs` | integration-full-crud.spec.ts:186-191 |
| Status toggle | `row.locator('.arco-switch')` — if count > 0, clicked twice, each time expecting `PUT /api/v1/admin/jobs/status` | integration-full-crud.spec.ts:194-198 |
| Row-scoped delete | `/删除|移除/` via `deleteTableRow` | integration-full-crud.spec.ts:200, 442 |

The manual-run lookup must stay row-scoped. Target selection is a closed allowlist supplied by
`GET /api/v1/admin/jobs/targets`; Cron expressions are standard five fields and no misfire selector is rendered.

`javascript:` The visible heading assertion (672) requires `AdminPageHeader` (or whatever renders the page title) to
keep emitting a real heading element — `getByRole('heading', { name: '定时任务' })` resolves to 0 if the title
becomes a `div`/`span`.

### 1.10 `/quartz/log/85` — JobLogsView (LogListView mode `job`)

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| URL | `/quartz/log/\d+$/` and `page.goto('/quartz/log/85')` after which `toHaveURL(/\/quartz\/log\/\d+$/)` | 698-699 |
| Main exact text | `getByRole('main').getByText('定时任务', { exact: true })` visible (mock `jobName` in the log row) | 700 |
| Query param | after load, the request to `/api/v1/admin/logs/jobs` must carry `jobId=85` (`jobLogQueryJobId`) | 429, 701 |
| Clean button | `page.getByRole('button', { name: '清空任务日志' })` | 702 |
| Popconfirm | `.arco-popconfirm:visible` `getByRole('button', { name: '确定' })` | 703 |
| Main contains | `main` `toContainText('任务日志')` | integration-full-crud.spec.ts:379, 385, 388 |
| Table | `main.locator('.arco-table')` visible | integration-full-crud.spec.ts:386, 389 |
| Detail flow | rows filtered by `getByRole('button', { name: '详情', exact: true })`; first row clicked; modal `toContainText('日志详情')`; `Escape`; modal `toBeHidden()` | integration-full-crud.spec.ts:393-401 |

### 1.11 `/operation/log`, `/exception/log` — LogListView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider clicks / URLs | `page.getByText('操作日志')` `/operation/log$`; `page.getByText('异常日志')` `/exception/log$` | 688-689, 695-696 |
| Unscoped text | `page.getByText('文章模块')` visible | 690 |
| Unscoped text | `page.getByText('模拟异常')` visible | 697 |
| Detail button | `page.getByRole('button', { name: '详情' })` — page-wide, must be the only match | 691 |
| Dialog text | `page.locator('.arco-modal:visible')` `toContainText('新增或修改')` | 692 |
| Close behavior | `page.keyboard.press('Escape')` then `page.locator('.arco-modal:visible')` `toHaveCount(0)` | 693-694 |
| Main contains | `main` `toContainText('操作日志')` / `'异常日志'` | integration-full-crud.spec.ts:377-378, 385, 388 |
| Detail modal | `modal` `toContainText('日志详情')`, `Escape`, `toBeHidden()` | integration-full-crud.spec.ts:399-401 |

Two behavioral locks: the detail modal must be closable with `Escape` (Arco default) and must be **removed or
hidden** afterwards, since `toHaveCount(0)` on `.arco-modal:visible` (694) and `toBeHidden()` on the `.last()` modal
(401) are both asserted. A redesign with `mask-closable={false}` + a persistent (non-visible) modal node is fine
because the selector filters `:visible`, but a design where Escape does not close the modal fails.

### 1.12 `/albums` — AlbumsView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('相册管理')`, `/albums$` | 705-706 |
| Unscoped text | `page.getByText('项目截图')` visible | 707 |
| Create button | `page.getByRole('button', { name: '新增' })` | 708 |
| Dialog | `page.locator('.arco-modal:visible')` | 709 |
| Ordinals **[FRAGILE]** | `albumTextInputs = albumDialog.locator('input[type="text"]')`; `nth(0)` = 相册名称, then `albumDialog.locator('textarea')` = 相册描述, then `nth(1)` = 封面 URL | 710-713 |
| Confirm | `albumDialog.getByRole('button', { name: '确定' })` | 714 |
| Recycle-bin button | `page.getByRole('button', { name: '回收站' })` | 715 |
| Table | `.arco-table` visible | integration-full-crud.spec.ts:113 |
| Row-scoped edit | `rowWithText(page, name).getByRole('button', { name: '编辑', exact: true })` | integration-full-crud.spec.ts:124 |
| Edit dialog ordinal | `modal.locator('input[type="text"]').nth(0)` = album name | integration-full-crud.spec.ts:126 |
| Row-scoped delete | `row.getByRole('button', { name: '删除', exact: true })` + popconfirm (endpoint `/api/v1/admin/albums/:id`) | integration-full-crud.spec.ts:133, 463-477 |

Why `input[type="text"]` here and bare `input` elsewhere: AlbumsView has a hidden `<input type="file">`
(AlbumsView.vue:57) inside the dialog, so the text inputs must be addressed by attribute. **A redesign that removes
the hidden file input is safe; one that adds a second text-ish input (e.g. a number field rendered as
`type="text"`) shifts `nth(1)` from cover URL to that field.**

### 1.13 `/albums/5` and `/albums/11` — PhotoView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| URL | `/albums/5$`, `page.goto('/albums/5')` | 725-726 |
| Main text | `getByRole('main').getByText('项目截图 · 照片')` visible (title = `${albumName} · 照片`) | 727 |
| Unscoped text | `page.getByText('首页截图')` visible | 728 |
| Marker | `main` contains `照片管理` (fallback title exists in source; the read-only route table uses `照片管理` for `/albums/5`) | integration-readonly.spec.ts:184 |
| Content selector | `.arco-table` visible on `/albums/5` | integration-readonly.spec.ts:184, 195 |
| Full-CRUD flow | `page.locator('.arco-table')` visible; `page.getByRole('row').filter({ has: page.getByRole('button', { name: '编辑', exact: true }) }).first()`; `row.getByRole('cell').nth(1)` = photo name | integration-full-crud.spec.ts:334-339 |
| Edit dialog | `modal.locator('input[type="text"]').first()` = photo name | integration-full-crud.spec.ts:345 |
| Remove button | `editedRow.getByRole('button', { name: '移除', exact: true })` + popconfirm → `PUT /api/v1/admin/photos/trash` | integration-full-crud.spec.ts:354-356 |
| Restore button | `deletedRow.getByRole('button', { name: '恢复', exact: true })` → `PUT /api/v1/admin/photos/trash` | integration-full-crud.spec.ts:363 |

> ⚠️ **Pre-existing contract conflict, worth knowing before you touch this view.** `PhotoView.vue` renders a
> masonry of `<article class="photo-tile">` elements and contains **no** `a-table` and **no** `row`/`cell` markup
> (verified: no `a-table` occurrence in the file). Yet the read-only suite requires `.arco-table` on `/albums/5`
> (integration-readonly.spec.ts:184, 195) and the full-CRUD suite drives `/albums/11` entirely through
> `.arco-table`, `getByRole('row')` and `row.getByRole('cell').nth(1)`
> (integration-full-crud.spec.ts:334-373). Those two gated suites cannot pass against the current PhotoView as
> written; if you are redesigning this view, a table-shaped contract for photo rows is what the tests encode.

### 1.14 `/photos/delete` — PhotoTrashView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Main text | `getByRole('main').getByText('照片回收站')` visible (also after reload) | 717, 724; integration-full-crud.spec.ts:360 |
| Unscoped text | `page.getByText('首页截图')` visible | 718 |
| Checkbox click **[FRAGILE]** | `page.getByRole('row', { name: /首页截图/ }).getByRole('checkbox').locator('..').click()` | 719 |
| Batch restore | `page.getByRole('button', { name: '批量恢复' })` `toBeEnabled()` then clicked → `PUT /api/v1/admin/photos/trash` with `isDelete: 0` | 720-722 (assert 519-520) |
| Table | `.arco-table` visible | integration-readonly.spec.ts:185 |

The `批量恢复` button must start **disabled** and become enabled only after row selection (`:disabled` bound to
`selectedIds.length === 0`, PhotoTrashView.vue:6) — `toBeEnabled()` at 720 is a real state assertion.

### 1.15 `/talk-list` — TalksView and `/talks`, `/talks/7` — TalkEditorView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('说说管理')`, `/talk-list$` | 729-730 |
| Unscoped text | `page.getByText('一次完整的前后端联调')` visible | 731 |
| Row edit **[FRAGILE]** | `page.getByRole('button', { name: '编辑' })` — page-wide, must be the only match; clicking it must *navigate* to `/talks/7` | 732-733 |
| Main text | `getByRole('main').getByText('编辑说说')` visible | 734; integration-readonly.spec.ts:186 |
| Form root | `page.locator('.talk-form')` visible | integration-crud.spec.ts:127, 135; integration-readonly.spec.ts:29, 114, 186, 195 |
| Textarea | `page.locator('.talk-form textarea')` (singleton) | integration-crud.spec.ts:128, 136 |
| Save button | `page.locator('.talk-form').getByRole('button', { name: '保存', exact: true })` | integration-crud.spec.ts:129, 137 |
| Post-save URL | `page.waitForURL((url) => url.pathname === '/talk-list')` | integration-crud.spec.ts:130, 138 |
| Edit navigation | `page.waitForURL((url) => /^\/talks\/\d+$/.test(url.pathname))` | integration-crud.spec.ts:134 |
| Row-scoped edit/delete | `tableRow(...).getByRole('button', { name: '编辑'|'删除', exact: true })` | integration-crud.spec.ts:133, 243 |
| Marker | `main` contains `发布说说` for `/talks` | integration-readonly.spec.ts:29 |

### 1.16 `/menus` and `/resources` — PermissionTreeView (MenuView / ResourceView wrappers)

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('菜单管理')` `/menus$`; `page.getByText('资源管理')` `/resources$` | 735-736, 744-745 |
| Table text (exact) **[FRAGILE]** | `page.getByRole('table').getByText('文章列表', { exact: true })` visible | 737 |
| Table text (exact) | `page.locator('.arco-table').getByText('文章读取', { exact: true })` visible | 746 |
| Marker | `main` contains `菜单管理` on `/menus`, `接口资源管理` on `/resources` | integration-readonly.spec.ts:30-31 |
| Create dialog ordinals **[FRAGILE]** | `menuDialog.locator('input[type="text"]').nth(0)` = 菜单名称, `nth(1)` = 路径, `nth(2)` = 组件路径 | 740-742 |
| Full-CRUD create ordinals | same, from `modal.locator('input[type="text"]')` | integration-full-crud.spec.ts:212-216 |
| Resources create ordinals | `nth(0)` = name, `nth(1)` = URL (only two used) | integration-full-crud.spec.ts:217-219 |
| Row-scoped edit/delete | `rowWithText(page, name).getByRole('button', { name: '编辑'|'删除', exact: true })` | integration-full-crud.spec.ts:225-227, 450 |
| Delete endpoint | `DELETE` on a path **starting with** `${endpoint}/` | integration-full-crud.spec.ts:452-455 |

737 asserts `getByRole('table').getByText('文章列表', { exact: true })` — i.e. the table landmark must contain an
element whose exact text is `文章列表`, and it must be the only exact match inside the table. **[verified]** Arco's
`.arco-table` wrapper exposes the `table` role; the node with the exact text is the cell `<span>`.

### 1.17 `/links` — FriendLinksView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('友链管理')`, `/links$` (also after reload) | 750-751, 772 |
| Main text | `getByRole('main').getByText('友链管理')` visible | 752, 773 |
| Unscoped exact text | `page.getByText('项目友链', { exact: true })` visible | 753 |
| Row scoping | `page.getByRole('row', { name: /项目友链/ })` — the regex row-name form, used for both edit and delete | 754, 774 |
| Row edit | `friendLinkRow.getByRole('button', { name: '编辑' }).click()` | 755 |
| Edit dialog | `page.locator('.arco-modal:visible')`; `toContainText('编辑友链')` | 756-757 |
| Input ordinal **[FRAGILE]** | `friendLinkEditDialog.locator('input').nth(0)` must have value `项目友链` | 758 |
| Confirm | `friendLinkEditDialog.getByRole('button', { name: '确定' })` | 759 |
| Create button | `page.getByRole('button', { name: '新增', exact: true })` (note: `exact` here, unlike 633/645) | 762 |
| Create dialog ordinals **[FRAGILE]** | `input.nth(0)` = 友链名称, `nth(1)` = 头像地址, `nth(2)` = 链接地址; then `locator('textarea')` = 友链介绍 | 764-768 |
| Row delete | `friendLinkRow.getByRole('button', { name: '删除' })` + `.arco-popconfirm:visible` 确定 | 774-775 |
| Table | `.arco-table` visible | integration-crud.spec.ts:94 |
| Row-scoped edit | `tableRow(page, created).getByRole('button', { name: '编辑', exact: true })` | integration-crud.spec.ts:102 |
| Row-scoped delete | `/删除|移除/` exact + popconfirm | integration-crud.spec.ts:162, 223 |

The row-name form at 754/774 means the row's **accessible name must contain `项目友链`**: that depends on the row
containing real text content (Arco cells), and is another reason a non-`tr` table breaks.

### 1.18 `/about` — AboutView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('关于我')`, `/about$` | 777-778 |
| Main text | `aboutMain.getByText('关于我')` visible (also marker `关于我` in read-only) | 780; integration-readonly.spec.ts:33 |
| Textarea **[FRAGILE]** | `aboutMain.locator('textarea')` — must be a singleton scoped to `main`; `toHaveValue(defaultAboutContent)`; then filled; value re-asserted after reload | 781-783, 788 |
| Save button | `aboutMain.getByRole('button', { name: '保存', exact: true })` | 784 |
| Full-CRUD variant | `main.locator('textarea')` singleton; `main.getByRole('button', { name: '保存', exact: true })` → `PUT /api/v1/admin/about` | integration-full-crud.spec.ts:292-300 |

### 1.19 `/website` — WebsiteView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('网站管理')`, `/website$` | 790-791 |
| Main text | `websiteMain.getByText('网站配置')` visible | 793; integration-full-crud.spec.ts:276 |
| Input ordinal **[FRAGILE]** | `websiteMain.locator('input').nth(0)` = 网站名称, asserted `星际信标`, refilled, re-asserted after reload | 794, 796, 801 |
| Textarea singleton **[FRAGILE]** | `websiteMain.locator('textarea')` `toHaveValue('欢迎来到星际信标')` — exactly one textarea inside `main` | 795 |
| Save button | `websiteMain.getByRole('button', { name: '保存', exact: true })` → `PUT /api/v1/admin/site` | 797; integration-full-crud.spec.ts:280, 284 |
| Full-CRUD variant | `main.locator('input').first()` = name; `main.getByRole('button', { name: '保存', exact: true })` | integration-full-crud.spec.ts:275-284 |

`main.locator('input').nth(0)` is only the site name while the name field stays **first** among inputs in the form
grid and no input (search, switch rendered as input, file) precedes it. `textarea` singletons in `main` forbid a
second textarea anywhere on these pages.

### 1.20 `/setting` — SettingView

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Sider click / URL | `page.getByText('个人中心')`, `/setting$` (also after reload) | 803-804, 823 |
| Main text | `settingMain.getByText('个人中心')` visible | 807; integration-readonly.spec.ts:35 |
| Filtered inputs **[FRAGILE]** | `settingMain.locator('input:not([type="file"])')`; `nth(0)` = 昵称, `nth(1)` = 个人网站 | 806, 808, 810, 811, 813, 824, 826 |
| Textarea singleton **[FRAGILE]** | `settingMain.locator('textarea')` `toHaveValue('保持公开资料边界')`, filled, re-asserted after reload | 809, 812, 825 |
| Save button | `settingMain.getByRole('button', { name: '保存资料', exact: true })` → `PUT /api/v1/auth/me` | 814 |
| Storage round-trip | `sessionStorage['stellar-beacon.admin.user']` parsed, `nickname === '测试管理员 E2E'` | 817-821 |
| Full-CRUD variant | `main.locator('input')` `nth(0)`/`nth(1)` and `main.locator('textarea')` (all inside `main`) | integration-full-crud.spec.ts:308-325 |

Note the difference between the two suites: admin-shell excludes `type="file"` from the input list (the avatar input
is `hidden`), while integration-full-crud uses plain `main.locator('input').nth(0)`/`nth(1)`. Today the avatar file
input is `hidden` but still an `input` in DOM order **before** nickname? It is rendered in `.profile-avatar-wrap`
(SettingView.vue:7), i.e. before the form. integration-full-crud.spec.ts:308-315 therefore only works while that
hidden file input either does not come first in DOM order or is absent at that moment — **[FRAGILE]**, treat the
first two non-file inputs and the singleton textarea as the locked contract, and do not add inputs above them.

### 1.21 `/unknown-page` — NotFoundView, and the no-menu state

| Kind | Exact locked value | Lines |
| --- | --- | --- |
| Text | `page.getByText('页面不存在')` visible | 834 |
| Negative text | `.admin-content` must not contain `页面不存在` on any valid route | 901, 907, integration.spec.ts:52, 55, integration-readonly.spec.ts:222 |
| Negative text | `.admin-content` must not contain `无权访问` | integration-readonly.spec.ts:223 |
| Negative text | `.admin-content` must not contain `模块迁移中` | integration-readonly.spec.ts:224 |
| No-menu state | `page.getByText('当前账号没有可见菜单，请联系管理员分配权限。')` visible (menu list = `[]`) | 847 |
| Hidden menu absence | every menu item with `hidden: true` must have `toHaveCount(0)` in `.arco-layout-sider` | integration-readonly.spec.ts:98-100 |
| Visible menu presence | each expected path must be exactly one `.arco-menu-item` inside the sider | integration-readonly.spec.ts:105-106 |
| Menu click navigation | `Promise.all([page.waitForURL((url) => url.pathname === menu.path), item.click()])` | integration-readonly.spec.ts:107-110 |
| Route-status assertion | `response?.status()` must be 200 on direct navigation **and** reload for every route | 898, 904 |

The no-menu sentence must match **exactly** (847 is an exact-string `getByText`), and the 404 page must contain
`页面不存在` while valid pages must never contain it — including as a substring.

### 1.22 Routes that must render (matrix)

`/`, `/article-list`, `/articles`, `/articles/42`, `/categories`, `/tags`, `/comments`, `/users`, `/roles`,
`/operation/log`, `/exception/log`, `/quartz`, `/quartz/log/85`, `/albums`, `/albums/5`, `/photos/delete`,
`/talk-list`, `/talks/7`, `/menus`, `/resources`, `/links`, `/about`, `/website`, `/online/users`, `/setting`
(admin-shell.spec.ts:868-894), each asserted with `.admin-shell` + `.admin-content` visible, no `页面不存在`,
URL unchanged, and HTTP 200 before and after reload (899-907). The read-only suite re-walks 21 of these with an
additional content marker and optional `.arco-table` / content-selector check (integration-readonly.spec.ts:14-36,
102-120), plus `/articles/42`, `/quartz/log/85`, `/albums/5`, `/photos/delete`, `/talks/7` (181-187), each with
`content.locator(':scope > *').first()` visible (220).

---

## 2. Required visible text strings (deduplicated)

Grouped by view. Exact-case; `exact: true` marks strings asserted with Playwright's exact option (no substring
tolerance), otherwise the assertion is a substring/text-content match.

### Login
- `邮箱`, `密码` (labels; not asserted directly, but the testids must remain ancestors of their inputs)
- `管理员邮箱` / `密码` (aria-labels; not asserted — informational)

### Shell / navigation (all views)
- `内容管理` (exact, in `.admin-sider`) — 618, 864
- `文章列表` (exact, sider click) — 620
- `发布文章` (exact, sider click) — 623
- `分类管理`, `标签管理`, `评论管理`, `用户管理`, `角色管理`, `操作日志`, `异常日志`, `定时任务`, `相册管理`, `说说管理`, `菜单管理`, `资源管理`, `在线用户`, `友链管理`, `关于我`, `网站管理`, `个人中心` (sider clicks / URLs) — 630, 637, 653, 657, 663, 670, 688, 695, 705, 729, 735, 744, 747, 750, 777, 790, 803
- Hidden menu labels that must be absent: `无可见菜单` (exact, count 0), `隐藏页面` (exact, count 0), `仅用于权限测试` (exact, count 0) — 619, 865, 866; plus all `hidden: true` labels from the API (read-only: 98-100)
- No-menu state (exact): `当前账号没有可见菜单，请联系管理员分配权限。` — 847

### `/` (home)
- `发布文章` — as an accessible button name inside `main` (exact) — 617, 862
- `账号状态`, `已认证`, `权限来源`, `RBAC` — integration.spec.ts:30-33

### `/articles` and `/articles/42`
- `发布文章` — 625
- `修改文章` — 628, integration-readonly.spec.ts:182
- `草稿` (select option, exact) — integration-full-crud.spec.ts:82
- `保存` (exact, button) — integration-full-crud.spec.ts:83, 95
- Mock-asserted values: `已存在文章` (title input value) — 629; `工程化` (category), `e2e` (tag) — integration-full-crud.spec.ts:79-80

### `/article-list`
- `文章列表` — 622, integration-readonly.spec.ts:17

### `/categories`
- `分类管理` (marker + sider) — 630, integration-readonly.spec.ts:18
- `工程化` (row) — 632
- `新增` (button) — 633
- `确定` (dialog button) — 636
- `编辑` (dialog title, via `toContainText`) — 642

### `/tags`
- `标签管理` — 651, integration-readonly.spec.ts:19
- `工程化` — 639, 652
- `编辑`, `新增`, `确定` — 640-648

### `/comments`
- `评论管理` — integration-readonly.spec.ts:20
- `不错` — 655
- `通过审核` (button, must be visible; source alternates with `取消审核`) — 656

### `/users`
- `用户管理` — integration-readonly.spec.ts:21
- `测试用户` — 659
- `编辑者` — 665
- `修改用户` (dialog, via `toContainText`) — 661
- `编辑`, `确定` — 660, 662

### `/roles`
- `角色管理` — integration-readonly.spec.ts:23
- `编辑者` — 665
- `新增`, `确定` — 666, 669
- `编辑权限` (exact, row button) — integration-full-crud.spec.ts:189

### `/quartz`
- `定时任务` (heading name + marker) — 672, integration-readonly.spec.ts:26
- `执行一次` (button; also an `a-option` label in the dialog) — 673-674
- `确定` (popconfirm + dialog) — 675, 684, 687
- `新增`, `编辑` — 677, 685
- `编辑任务` (dialog, via `toContainText`) — 686, integration-full-crud.spec.ts:155
- `新增任务` (create-dialog title, source)

### `/quartz/log/85`
- `定时任务` (exact, inside `main`) — 700
- `任务日志` (marker) — integration-full-crud.spec.ts:379
- `清空任务日志` (button) — 702
- `确定` (popconfirm) — 703
- `详情` (exact, row button) — integration-full-crud.spec.ts:394
- `日志详情` (modal, via `toContainText`) — integration-full-crud.spec.ts:399

### `/operation/log`
- `操作日志` — integration-full-crud.spec.ts:377, integration-readonly.spec.ts:24
- `文章模块` — 690
- `新增或修改` (modal, via `toContainText`) — 692
- `详情`, `日志详情` — 691, integration-full-crud.spec.ts:399

### `/exception/log`
- `异常日志` — integration-full-crud.spec.ts:378, integration-readonly.spec.ts:25
- `模拟异常` — 697

### `/albums`
- `相册管理` — 707, integration-readonly.spec.ts:27
- `项目截图` — 707
- `新增`, `确定`, `回收站` — 708, 714, 715
- `编辑相册` / `新增相册` (dialog titles, source; not asserted by name)

### `/albums/5` + `/albums/11`
- `项目截图 · 照片` (title, `albumName + ' · 照片'`) — 727
- `首页截图` — 728
- `照片管理` (fallback title; read-only marker for `/albums/5`) — integration-readonly.spec.ts:184
- `编辑`, `移除`, `确定`, `恢复` (buttons) — integration-full-crud.spec.ts:337, 354, 356, 363

### `/photos/delete`
- `照片回收站` — 717, 724, integration-full-crud.spec.ts:360
- `首页截图` — 718
- `批量恢复` (button) — 720-721

### `/talk-list`, `/talks`, `/talks/7`
- `说说管理` — 731, integration-readonly.spec.ts:28
- `一次完整的前后端联调` — 731
- `发布说说` (marker) — integration-readonly.spec.ts:29
- `编辑说说` — 734, integration-readonly.spec.ts:186
- `编辑`, `保存` (exact) — 732, integration-crud.spec.ts:129, 137

### `/menus`, `/resources`
- `菜单管理` — integration-readonly.spec.ts:30
- `接口资源管理` (note: the resources page title is `接口资源管理`, whereas the *menu* label is `资源管理`) — integration-readonly.spec.ts:31
- `文章列表` (exact, inside `getByRole('table')`) — 737
- `文章读取` (exact, inside `.arco-table`) — 746
- `新增`, `确定` — 738, 743

### `/links`
- `友链管理` — 752, 773, integration-readonly.spec.ts:32
- `项目友链` (exact) — 753
- `编辑友链` (dialog, via `toContainText`) — 757
- `新增` (exact), `编辑`, `删除`, `确定` — 762, 755, 774, 759/775

### `/about`
- `关于我` — 780, integration-readonly.spec.ts:33
- `保存` (exact) — 784, integration-full-crud.spec.ts:296, 300

### `/website`
- `网站配置` — 793, integration-full-crud.spec.ts:276
- `保存` (exact) — 797, integration-full-crud.spec.ts:280, 284

### `/setting`
- `个人中心` — 807, integration-readonly.spec.ts:35
- `保存资料` (exact) — 814

### 404 / forbidden / placeholder
- `页面不存在` (must be visible on unknown route; must never appear in `.admin-content` on valid routes) — 834, 901, 907
- `无权访问` (must never appear on valid routes) — integration-readonly.spec.ts:223
- `模块迁移中` (must never appear on valid routes) — integration-readonly.spec.ts:224

### Filter/search query values (asserted via network, not DOM text)
`keywords` param must equal the typed value on `/categories`, `/tags`, `/links`, `/albums`, `/roles`, `/menus`,
`/resources`, `/users`, `/articles`, `/talks` (integration-crud.spec.ts:169-174; integration-full-crud.spec.ts:428-431);
`jobName` param on `/quartz` (integration-full-crud.spec.ts:150, 159, 169); `jobId` param on `/quartz/log/:id`
(admin-shell.spec.ts:429, 701).

---

## 3. Interaction sequence dependencies (FRAGILE inventory)

Each entry: what the test does, why it depends on uniqueness/ordering, and exactly what breaks it.

**F1 — `getByRole('button', { name: '新增' })` unscoped (admin-shell.spec.ts:633, 645, 666, 677, 708, 738).**
Not `exact`, not row-scoped, page-wide. Works only while one visible button's accessible name contains `新增`.
Breaks if: a second `新增`-ish button appears (e.g. `新增分类` in an empty-state card, a dropdown item rendered as a
button, a floating action button), or the label becomes `新建`. Note the same tests use `exact: true` at
admin-shell.spec.ts:762 and in all CRUD suites (integration-crud.spec.ts:70, 95; integration-full-crud.spec.ts:114,
142, 182, 210) — a label change to `新增xxx` breaks the `exact` variants while still passing the loose ones.

**F2 — `getByRole('button', { name: '编辑' })` unscoped (admin-shell.spec.ts:640, 660, 685, 732).**
Four different pages rely on a page-wide, non-exact `编辑` click. On `/tags` (640) it opens the row edit dialog; on
`/users` (660) it opens `修改用户`; on `/quartz` (685) it opens `编辑任务`; on `/talk-list` (732) it **navigates** to
`/talks/7`. Breaks if: more than one button's name contains `编辑` (a toolbar `批量编辑`, a row `编辑权限` rendered
on the same page — note `/roles` already has `编辑权限`, which is why `/roles` never uses the unscoped form for
editing), or if the click target stops being an element with the `button` role (icon-only action with `aria-label`
changed, or a `<a>` link instead of a button). For 732 the navigation contract adds a second requirement: the
table's 编辑 action must route to `/talks/:id` (`integration-crud.spec.ts:134` asserts `/^\/talks\/\d+$/`).

**F3 — `getByRole('button', { name: '确定' })` page-wide (admin-shell.spec.ts:662).**
Not scoped to the modal. Works only because exactly one visible `确定` exists. Breaks if any second confirm control
is rendered simultaneously (batch toolbar, popconfirm still in the DOM and visible, a second dialog). Note that
elsewhere the same author scopes the button (`tagEditDialog.getByRole(...)` at 644, `jobDialog…` at 684), so only
662 has this exposure.

**F4 — Dialog `locator('input')` singular (admin-shell.spec.ts:635, 643, 647).**
`fill` without `.first()` → strict mode requires exactly one input inside the taxonomy dialog. Breaks if the dialog
gains any second input (hidden file input, search, number field). Same class of risk: `tagEditDialog.locator('input')`
at 643.

**F5 — Dialog text-input ordinals (the most fragile group).**

| Test line | Dialog | Ordinal meaning that must hold |
| --- | --- | --- |
| 679-683 | `/quartz` create | label-scoped 任务名称, 任务分组 and Cron; target select option `userArea.refresh` |
| 710-713 | `/albums` create | nth 0=相册名称, then the **only** textarea=相册描述, then nth 1=封面 URL |
| 740-742 | `/menus` create | nth 0=菜单名称, 1=路径, 2=组件路径 |
| 668 | `/roles` create | `input[type="text"]` first=角色名 |
| integration-full-crud.spec.ts:167-176 | `/quartz` create | same label-scoped fields and target select |
| integration-full-crud.spec.ts:116-119 | `/albums` create | same 3 ordinals as 710-713 |
| integration-full-crud.spec.ts:212-219 | `/menus`/`/resources` create | menus: 0=name,1=path,2=component; resources: 0=name,1=url |
| integration-full-crud.spec.ts:126, 184, 191, 227, 345, 369 | edit dialogs | `input[type="text"]`.first() = the entity name |
| 758, 764-767 | `/links` edit/create | `input` nth 0=name, 1=avatar, 2=address |
| integration-full-crud.spec.ts:246, 255 | `/users` edit | `input`.first() = nickname |
| 806-813 | `/setting` | `input:not([type="file"])` nth 0=nickname, nth 1=website |
| 794, 796, 801; integration-full-crud.spec.ts:277 | `/website` | `input` first = site name |
| 629 | `/articles/42` | **page-wide** `input` first = article title |
| integration-full-crud.spec.ts:77-80 | `/articles` | `.article-form input` nth 0=title, 1=category, 2=tag |

Breaks if any input is added/removed/reordered before or between these fields, if a field's element stops emitting
`type="text"`, or if a select/number/switch control renders an extra `input` in between. Reordering fields, moving
the cover/file input earlier, or splitting the form into tabs (which may unmount/remount fields) all move the
ordinals.

**F6 — Textarea singletons.**
`locator('textarea')` with no index is used at admin-shell.spec.ts:712, 768, 781, 795, 809, 825;
integration-crud.spec.ts:122; integration-full-crud.spec.ts:81, 94, 118, 292, 309, 320, 325; and
`.talk-form textarea` at integration-crud.spec.ts:128, 136. Each must resolve to exactly one textarea **within its
scope** (`main`, the dialog, or the form). Adding a second textarea (hint, notes, JSON editor, markdown preview
wrapping a textarea) breaks all of them with a strict-mode violation.

**F7 — `getByRole('button', { name: '详情' })` (admin-shell.spec.ts:691) and `执行一次` (673-674) unscoped.**
Both are page-wide, non-exact. `详情` must be the only match on `/operation/log`; `执行一次` must be the only match
on `/quartz` **and must be enabled** (`toBeEnabled()` at 673) while the mocked `canRunOnce: true` row exists. Breaks
if the disabled variant is the only one rendered, if a second enabled/disabled 执行一次 appears, or if the button
becomes a link with a different role.

**F8 — `getByRole('button', { name: '通过审核' })` (656) and `getByRole('button', { name: '回收站' })` (715),
`批量恢复` (720-721), `清空任务日志` (702), `保存资料` (814), `保存` (784, 797 + all CRUD save sites).**
All page-wide and (except where noted) non-exact; each must be the single match by accessible name. The `保存` family
is guarded by `exact: true` in the form views (784, 797, integration-full-crud.spec.ts:83, 95, 129, 137, 280, 284,
296, 300, 316, 322) which means the label cannot become `保存并发布` / `保存草稿` without breaking those, even though
a substring match would still find it.

**F9 — Row-scoped action buttons (`getByRole('button', { name: '编辑'|'删除'|'编辑权限'|'移除'|'恢复', exact: true })`
inside `getByRole('row')`).**
Used at admin-shell.spec.ts:755, 774; integration-crud.spec.ts:77, 102, 133, 162, 202, 223, 243;
integration-full-crud.spec.ts:124, 153, 189, 225, 244, 253, 337, 343, 354, 363, 367, 442, 450, 466.
These are the *safest* action contracts (row-scoped + exact), and the redesign should preserve that shape: the row
must remain an accessible `row` whose text contains the entity name, and each action must remain an exact-named
button inside it. `deleteTableRow` additionally allows `/删除|移除/` (integration-full-crud.spec.ts:442), so both
labels are accepted there — but integration-crud's variant requires exactly `删除` (162).

**F10 — "first button in the row" is a review toggle (integration-full-crud.spec.ts:266-270).**
The comment step clicks `row.getByRole('button').first()`, records its text, then expects a second click to restore
that text, with `PUT /api/v1/admin/comments/review` between. Breaks if any other control precedes the review toggle
in the actions cell, or if the toggle is not a `button` (e.g. an `a-switch`, which has role `switch`).

**F11 — Checkbox parent click (admin-shell.spec.ts:719).**
`getByRole('row', { name: /首页截图/ }).getByRole('checkbox').locator('..').click()` requires the checkbox to be the
only checkbox in the row and its parent to be the clickable selection wrapper. Breaks if the checkbox is wrapped in
an extra element, if the row renders a second checkbox (per-row toggle), or if select-on-row-click removes the
checkbox input.

**F12 — Modal uniqueness / `:visible` filtering.**
`page.locator('.arco-modal:visible')` is used bare (no `.first()`) at 634, 641, 646, 661, 667, 678, 686, 687, 692,
709, 739, 756, 763 while `visibleModal()` uses `.last()` (integration-crud.spec.ts:148-150;
integration-full-crud.spec.ts:483-485). Because Arco keeps hidden modals in the DOM (v-show), the `:visible` filter is
load-bearing: removing it (or replacing the class name `arco-modal`) breaks every dialog assertion with strict-mode
violations. Breaks also if two dialogs are open at once (nested confirm inside a modal) — bare `.locator('.arco-modal:visible')`
then resolves to 2 elements.

**F13 — `.arco-popconfirm:visible` must be the popconfirm that owns the `确定` (675, 703, 775, and all CRUD delete
paths).** Arco popconfirms are rendered into the DOM and hidden when unused; the `:visible` filter plus
`getByRole('button', { name: '确定' })` inside it is the contract. Breaks if the confirm action moves to a modal, if
the confirm button label changes (`确定` vs `确认`), or if the confirm control is not a button role.

**F14 — `.admin-sider` vs `.arco-layout-sider` dual contract.**
admin-shell clicks menu items via `.admin-sider` (618, 620, 623) while integration-readonly scopes menus via
`.arco-layout-sider` and requires each entry to be an `.arco-menu-item` with count exactly 1 (97, 105-106). A
redesign that swaps the sidebar for a custom nav (no Arco sider/menu classes) satisfies the baseline suite but breaks
the read-only suite; keeping `class="admin-sider"` on an `a-layout-sider` satisfies both.

**F15 — Menu labels must be clickable *text* nodes.**
`page.getByText('分类管理')` etc. (630, 637, 653, …) and `sidebar.locator('.arco-menu-item').filter({ hasText })`
require the menu label text to be present in the DOM inside the clickable item. If the redesign moves labels into
`aria-label` only, or collapses long labels behind tooltips that render text elsewhere, both the click and the
count-1 assertions break. `exact: true` applies at 618, 620, 623, 864 (等) so decorative suffixes (e.g. a badge
rendered inside the same text node) break exact matching.

**F16 — `.arco-table` / accessible-row coupling (see §0).** Any table redesign must keep native table semantics
*and* the `.arco-table` class root, because both are asserted (e.g. 746 and integration-full-crud.spec.ts:113).
`getByRole('table').getByText('文章列表', { exact: true })` (737) additionally requires that the exact text node be
inside the table landmark.

**F17 — Search interactions require the Arco InputSearch structure.**
`filterTable` fills `.arco-input-search`'s textbox and then clicks
`.arco-input-search .arco-icon-hover:not(.arco-input-clear-btn)` (integration-crud.spec.ts:168-179;
integration-full-crud.spec.ts:427-434). The response promise is registered before the fill and awaits a GET whose
`keywords`/`jobName` param equals the typed value — so the search must be submit-triggered (icon click) and must
send the typed value as that exact query parameter. Breaks if: search becomes debounce-only (no icon click target,
so the click resolves to 0 elements or the request fires before the promise is registered), if the clear button
becomes the only `.arco-icon-hover`, or if the param is renamed.

**F18 — Nav-then-assert sequencing on refresh.**
Several assertions run immediately after `page.reload()` (649-652, 723-724, 771-773, 787-788, 800-801, 822-826,
integration-crud.spec.ts:85-87, integration-full-crud.spec.ts:100-102, 130-132, 168-170, 195-197, 231-233, 257,
281-286, 297-302, 317-325, 349-351, 372-373, 387-389). They assert the *same* DOM contract after a cold load, which
means any state that only exists in memory (unsaved form state rendered only client-side, modal kept open across
navigation) is not covered — but any text/ordinal that changes on a fresh load (e.g. columns reordered after data
arrives, virtualization moving inputs) breaks the post-reload assertions.

---

## 4. Safety rules for a redesign

### Must NOT change (hard contract)

1. `<main class="admin-content">` — keep a real `main` element with that class, wrapping the routed view, and keep at
   least one visible direct child (`:scope > *`, integration-readonly.spec.ts:220). Do not add `role="main"`
   duplicates.
2. `.admin-shell` on the shell root (an `a-layout`/`section`) and `.admin-sider` on the sidebar — both are asserted
   by class.
3. Keep the sidebar an Arco sider with Arco menu items: `.arco-layout-sider`, `.arco-menu-item`, label text present
   as text inside the item.
4. Arco `a-modal` for every dialog (root class `.arco-modal`), `a-popconfirm` for every confirm
   (`.arco-popconfirm`), `a-table` for every list (`.arco-table` + native table semantics).
5. Everything Arco-based that the tests touch at the DOM level: `a-input` must emit a real `<input>` (with
   `type="text"` for ordinary text fields), `a-textarea` a real `<textarea>`, `a-input-search` the
   `.arco-input-wrapper.arco-input-search` structure with one non-clear `.arco-icon-hover`, `a-switch` the
   `.arco-switch` class.
6. The three login testids, kept on the component wrappers (not on the inner `<input>`), and `sessionStorage` keys
   `token` + `stellar-beacon.admin.user` (with `nickname`).
7. Field order and count inside every dialog and form (see F5/F6). Do not add, remove, or reorder inputs/textareas
   before the fields the tests fill, and do not introduce a second textarea into a scope that has one.
8. Exact button labels: `新增`, `编辑`, `删除`, `移除`, `恢复`, `确定`, `保存`, `保存资料`, `通过审核`,
   `执行一次`, `详情`, `批量恢复`, `回收站`, `清空任务日志`, `编辑权限`, `发布文章`, `发布说说`, `返回列表`
   (where asserted). `确定` must stay the confirm label (not `确认`/`好的`).
9. Dialog titles containing the asserted substrings: `编辑` (taxonomy), `修改用户`, `编辑任务`, `编辑友链`,
   `日志详情`, plus page markers `网站配置`, `照片回收站`, `编辑说说`, `修改文章`, `发布文章`.
10. Page markers that must remain visible inside `main`: `分类管理`, `标签管理`, `评论管理`, `用户管理`,
    `角色管理`, `操作日志`, `异常日志`, `定时任务`, `相册管理`, `说说管理`, `菜单管理`, `接口资源管理`,
    `在线用户`, `友链管理`, `关于我`, `网站配置`, `个人中心`, `照片回收站`, `任务日志`, `文章列表`,
    `发布说说`, `编辑说说`, `修改文章`, `发布文章`, `照片管理`.
11. The strings `页面不存在`, `无权访问`, `模块迁移中` must appear only on their own pages, never inside
    `.admin-content` on a valid route.
12. The exact empty-state sentence `当前账号没有可见菜单，请联系管理员分配权限。`
13. The route table and URL shapes (§1.22) plus HTTP 200 on load and reload.
14. Visible menu labels and hidden-menu suppression (hidden items must not render in the sider).
15. Escape closes the log-detail modal and it becomes hidden afterwards.
16. Arco class names on the views' roots and forms that the tests select: `.article-form`, `.talk-form`,
    `.arco-table`, `.arco-modal`, `.arco-popconfirm`, `.arco-sider` (see 2/4).
17. The `keywords` / `jobName` / `jobId` query parameter names and the submit-triggered search flow.

### Safe to change (no test reads these)

- Any CSS property that does not affect `:visible`/layout-collapse: colors, gradients, fonts, sizes, spacing,
  radius, shadows, borders, backgrounds, z-index (as long as modals/popconfirms remain visible when open and
  hidden when closed), animations — **except** that element must not become permanently hidden or zero-size
  (Playwright's `:visible` requires a non-empty bounding box), and an animation must not leave the modal
  `opacity: 0`/`visibility: hidden` when it should be visible.
- Adding wrapper `div`s/`span`s, layout containers, `a-space`/`a-grid` scaffolding, CSS classes, `data-*`
  attributes that are not `data-testid` values listed above.
- Adding new non-asserted text (descriptions, hints, captions, column titles, empty-state copy that is not one of the
  asserted strings), extra table columns (appended **after** the existing ones; inserting a column changes
  `row.getByRole('cell').nth(1)` at integration-full-crud.spec.ts:339, which is the photo-name column on `/albums/:id`).
- Adding tooltips, icons, badges, avatars, tags (as long as they do not add a button whose accessible name contains
  `新增`/`编辑`, do not add a second `确定`, and do not add text that breaks the exact-text matches).
- Adding rows to a table is safe **only** when the test uses row-scoped locators; the mocked baseline data has one
  row per table, and the unscoped `编辑`/`新增`/`详情` clicks require the page to still expose a single such button.
  Real-backend suites create data dynamically, so do not rely on "there is only one row".
- Dark mode, responsive breakpoints, density/theme switches: not asserted, but the switch must not emit console
  errors and must not steal layout space that would make `.admin-content` scroll-hide its first child.
- Loading skeletons/spinners: safe, provided the final state renders the locked markers, and provided the loading
  state does not itself emit an input/textarea/button that breaks an ordinal or the singleton rules.
- Replacing `a-input` with a differently-styled component **only** if it still renders a native `<input>` with the
  same ordinal position and `type="text"` (riskier than keeping Arco).
- Adding a `role="main"`/landmark elsewhere is not needed; do not add a second element with the `main` role, and do
  not add a second element whose accessible name matches one of the `getByRole('button', { name })` strings.

### High-risk areas if you restyle them

- The two "page-wide unscoped" button lookups (`编辑` at 660/685/732, `确定` at 662) — any extra action button in the
  toolbar or actions column can break them. Prefer keeping the toolbar action set unchanged.
- The form ordinals (F5) — highest probability of accidental breakage while "just moving fields around".
- `page.locator('input').first()` at 629 (page-wide first input).
- The singleton textarea assertions in `/about`, `/website`, `/setting`.
- The comments "first button in the row" toggle.
- The photo table contract (see the ⚠️ note in §1.13): the gated suites already expect an `.arco-table` with rows and
  cells on `/albums/:id`, which `PhotoView.vue` does not currently render. If you restyle this view, decide
  consciously whether to add that table shape (which would satisfy the tests) or treat the conflict as out of scope.

---

## 5. Console-error constraint

### The mechanism

`capturePageErrors` is defined at admin-shell.spec.ts:932-939:

```
932  function capturePageErrors(page: import('@playwright/test').Page): () => string[] {
933    const errors: string[] = []
934    page.on('pageerror', (error) => errors.push(error.stack || error.message))
935    page.on('console', (message) => {
936      if (message.type() === 'error') errors.push(message.text())
937    })
938    return () => errors
939  }
```

It collects (a) every uncaught page error (`pageerror`: uncaught exceptions and unhandled promise rejections) and
(b) every console message of type `error`. It is then asserted to be **exactly empty** with `toEqual([])`:

| Assertion site | Line | Scope |
| --- | --- | --- |
| baseline, full mocked flow | admin-shell.spec.ts:827 | after the entire 200-line flow, including all reloads |
| unknown route / 404 page | 835 | after `page.goto('/unknown-page')` |
| empty-menu login | 848 | after login with `data: []` menus |
| all-routes + refresh matrix | 910 | after visiting and reloading all 25 routes |
| integration-crud | integration-crud.spec.ts:52 | after the whole CRUD + cleanup flow |
| integration-full-crud | integration-full-crud.spec.ts:50 | after all 13 steps |
| integration-readonly (nav) | integration-readonly.spec.ts:125 | after walking all visible menus |
| integration-readonly (routes) | integration-readonly.spec.ts:210 | after the dynamic-route walk |
| integration.spec.ts | 58 | after login + 11 routes with reloads |

Three of these are **always-on** (827, 835, 848, 910) and cover every route, so in practice the console must be
error-free on `/login`, `/`, and all 25 routes, on initial load, after client-side navigation, and after a hard
reload. Note the exactness: a single `console.error` anywhere in that window fails the whole test, even if the UI is
visually correct.

### What triggers a failure

1. **Vue runtime warnings promoted to errors** — none, by default (`console.warn` is not collected), *unless* the app
   or a plugin overrides `console.warn`/`app.config.warnHandler` to call `console.error`. Watch for library code
   that does this.
2. **Uncaught exceptions in components** — anything that throws during render/setup, in a watcher, in an event
   handler, or inside a `Promise` that is not caught. `pageerror` catches these regardless of console output.
3. **Unhandled promise rejections** — e.g. a `void router.push(...)`, an `await` in an event handler without
   `try/catch`, or a failed `fetch`/axios call that is not caught. Several views catch API errors into
   `errorMessage` + `Message.error` (e.g. JobsView.vue:184-185), which does not `console.error`; any refactor that
   drops the catch produces both a rejection and possibly a message.
4. **Failed network responses the app logs** — the baseline mock fulfils *every* `/api` path (admin-shell.spec.ts:597-601
   catch-all), so a 404/500 from a new endpoint the redesign calls (a new image URL, a font, a source map, a
   `favicon.ico`, an analytics beacon) can surface as a console error via the framework/axios interceptor. In the
   gated suites, any `/api` response with status ≥ 400 **also** fails `apiFailures` independently
   (integration-crud.spec.ts:20-25, 51; integration-full-crud.spec.ts:24-29, 49; integration-readonly.spec.ts:67-72,
   123, 163-166, 208), and in the read-only suite any non-GET/HEAD/OPTIONS `/api` request fails `unsafeRequests`
   (integration-readonly.spec.ts:54-60, 122, 148-154, 207) as does any failed request (61-66, 124, 155-160, 209).
5. **ECharts** (`echarts` is a dependency; `AdminEChart.vue` wraps it) — ECharts logs `[ECharts] ...` warnings via
   `console.warn` (not collected) but *also* emits `console.error` for real failures: `Can't get DOM width or
   height`, initialization on a zero-size container, `Instance is disposed` after re-render, or a theme/renderer
   error. Any dashboard/monitor restyle that mounts a chart in a hidden or zero-height container, or disposes an
   instance while an update is in flight, can produce `console.error` and fail the suite. Charts are not otherwise
   asserted by these specs (only that `.admin-content` renders a child), so chart changes are behaviourally free but
   console-risky.
6. **Vue Router / async component load failures** — a failed dynamic import or a component throwing in `setup`
   surfaces as `pageerror`. Also, the router's `beforeEach` swallows menu-load failures into a redirect
   (router/index.ts:60-65), which is safe; replacing that with a rethrow would break login.
7. **Arco Design internals** — Arco occasionally emits `console.error` for misuse (invalid prop types, missing
   required props, deprecated props, `Modal` used without a title, `Table` with a bad `row-key`, duplicate keys in
   `v-for`). A restyle that changes props (e.g. dropping `row-key`, passing an invalid prop type, `v-for` without a
   unique key) can trip these. Duplicate `v-for` keys are the classic source.
8. **Image/media load failures** — broken `<img src>` falls back in `AdminImagePreview` (it renders a
   `图片不可用` fallback) but a browser-level 404 is not a console error by itself; however any app-level `onerror`
   handler that logs, or a request to a URL the mock does not cover that an interceptor logs, will fail.
9. **Third-party scripts, fonts, favicons** — a `console.error` emitted by anything loaded on the page counts; the
   mock only intercepts `/api`, everything else goes to the network (`route.continue()`, admin-shell.spec.ts:44-47).
10. **Source-map / Vite HMR noise** — dev-server-specific warnings are `warn`-level and not collected, but an HMR
    error (failed module update) appears as `pageerror`.
11. **The `/unknown-page` test (830-836)** — the only page loaded there is the 404 view; it must not log anything
    while rendering, and must not attempt any failing request. Same for the empty-menu test (838-849), where the
    empty-menu alert path (AdminLayout.vue:12-14) must render without logging.

### Practical implications for the redesign

- Do not introduce `console.error` logging anywhere in view code, including "temporary" debug logs.
- Keep every API call wrapped so failures are caught and surfaced in the UI, not thrown.
- Do not add new network dependencies (fonts, CDN scripts, external images) to admin routes: the baseline mock
  fulfils only `/api/**`, and any resulting console error fails 827/910.
- If you keep ECharts on `/` or `/online/users`, ensure charts mount with a measurable box and are disposed
  cleanly; a chart error is the most likely new `console.error` a restyle would introduce.
- Ensure `v-for` keys stay unique and all Arco component props remain type-valid after refactors.
- Remember `retries: process.env.CI ? 2 : 0` (playwright.config.ts:14): locally a console error fails immediately;
  in CI it is retried twice and then still fails, so do not rely on flakiness.

### Timing constraint (only the login page)

`login first screen stays within the browser baseline budget` (admin-shell.spec.ts:913-930) asserts, on `/login` only:
`domContentLoaded > 0` and `< 5000 ms`, `loadEvent > 0` and `< 5000 ms`, and if `firstContentfulPaint > 0` then
`< 5000 ms`. A heavier login page (large images, blocking fonts, synchronous imports) can breach this. No timing
budget applies to admin views.

---

## Appendix: environment facts verified during this analysis

- `playwright.config.ts`: `testDir: './tests/e2e'`, `timeout: 30_000`, `workers: 1`, `fullyParallel: true`,
  baseline `baseURL` `http://127.0.0.1:8082` (overridable via `E2E_BASE_URL`), local `webServer` runs
  `npm run serve -- --host 127.0.0.1 --port 8082` with `reuseExistingServer: true`, single `chromium` project
  (Desktop Chrome). Two baseline tests raise their own timeout to 120 s (609, 855); the gated CRUD suites use
  180 s / 360 s (integration-crud.spec.ts:8, integration-full-crud.spec.ts:10).
- Arco Design Vue resolved version: **2.58.0** (`package.json` declares `^2.57.0`).
- Live DOM checks (Chromium, dev server, mocked API), which the "**[verified]**" markers above refer to:
  - `.admin-content` is `MAIN`; `.admin-shell` is `SECTION`; `.arco-layout-sider` is `DIV`; `getByRole('main')` → 1.
  - Arco `tr`/`td` have no `role` attribute but `getByRole('row')` → 2 (header + one data row) and
    `getByRole('cell')` → 5; `getByRole('table')` → 1; `.arco-table-tr` also matches the header row (so
    `.arco-table-tbody .arco-table-tr` is the body-only selector the gated suites use).
  - `.arco-modal-footer` is inside `.arco-modal` (so `modal.getByRole('button', { name: '确定' })` is valid); Arco
    modal inputs carry `type="text"`.
  - `.arco-input-search` renders as `SPAN.arco-input-wrapper.arco-input-search` containing
    `input.arco-input[type=text]` and exactly one `.arco-icon-hover` (the search icon); the clear icon only appears
    with a value and uses `.arco-input-clear-btn`.
  - Login: `[data-testid="login-username"]`/`-password` are `SPAN.arco-input-wrapper`, each containing exactly one
    `<input>` (`type=text` id `admin-username`; `type=password` id `admin-password`);
    `[data-testid="login-submit"]` is a `BUTTON.arco-btn`.

No file in the repository was modified by this analysis: the temporary probe spec and probe directory were created
outside the delivered artifacts and then deleted. Note that `web/apps/admin-next/src/styles.css` was already modified
in the working tree (2188 insertions / 811 deletions) before this analysis began — that pre-existing change is not
mine and was left untouched; the suite contract above is unaffected by it because it only asserts the class names and
roles listed in §1 and §2.

---

## Addendum (console capability round): new surfaces and the rules they impose

Everything in §0–§2 above still holds and is still asserted. This addendum records the DOM surfaces added when the
console gained batch operations, URL-synced filters, live search and the editor leave-guard, plus the constraints any
future change to them must respect.

### New always-renderable surfaces

| Surface | Where | Notes |
| --- | --- | --- |
| `.admin-batch-bar` | `components/AdminBatchBar.vue` | Renders **only when `count > 0`**. Holds a `已选 N 项` caption, the action slot, and a `取消选择` text button. |
| `.admin-error-state` | `components/AdminErrorState.vue` | Replaces the per-view list-load `a-alert`. Carries `role="alert"` and one `重试` button; renders only when the list request failed. |
| `.admin-column-settings` | `ArticleListView`, `LogListView` | Content of a `列设置` dropdown holding `a-checkbox` per column. Only reachable on user click; never visible by default. |
| Leave guard | `components/AdminLeaveGuard.vue` + `composables/useUnsavedGuard.ts` | An `a-modal` in `ArticleEditorView`/`TalkEditorView`, visible only when the form differs from its post-load baseline. Buttons are `放弃修改` / `继续编辑` — deliberately **not** `确定`. |

### Rules for future edits (each one is enforced by an existing assertion)

1. **Batch labels must avoid `新增`, `编辑`, `确定`, `通过审核`, `取消审核`, `执行一次`.** `admin-shell.spec.ts` clicks
   page-wide, non-exact `getByRole('button', { name: '新增' | '编辑' })` on `/categories`, `/tags`, `/users`, `/quartz`,
   `/talk-list`, `/albums`, `/menus`, `/resources` and a page-wide `确定` on `/users` (662). Any additional matching
   button turns those clicks into strict-mode violations. `批量审核` / `批量删除` / `取消选择` / `导出 Markdown` /
   `移动到相册` were chosen to satisfy this.
2. **Batch bars only exist while something is selected.** This is what keeps rule 1 satisfiable for the many buttons
   whose names would otherwise collide, and it is why `CommentsView`'s `通过审核` and `PhotoTrashView`'s `批量恢复`
   assertions still resolve to exactly one element.
3. **`PhotoTrashView` keeps exactly one checkbox per row** and `批量恢复` must stay `disabled` until a row is selected
   (719-722). The same `v-model:selected-keys` + `onlyCurrent` pattern was applied to the other list views, so a row
   checkbox now exists on `/article-list`, `/comments`, `/talk-list`, `/quartz`, `/roles`, `/tags`, `/categories`,
   `/links`, `/albums/5` and all three log routes. No assertion counts checkboxes on those pages.
4. **URL sync never writes a default value.** `composables/useQueryFilters.ts` drops any filter equal to its initial
   value, so the default URL stays `/article-list`, `/categories`, `/tags`, `/albums`, `/users`, `/roles`,
   `/online/users`, `/links`, `/media`. The `toHaveURL(/…$/)` anchors in §0 depend on this — do not "normalise" the
   query by always writing the full filter set.
5. **`/quartz/log/:quartzId`'s `jobId` must keep flowing.** It is derived from the route param (not the query) inside
   the `useAsyncList` fetcher (`LogListView.vue`), and `admin-shell.spec.ts:701` asserts the request carries `jobId=85`.
   `useQueryFilters` preserves unrelated query keys but must not be used to move `jobId` into the query.
6. **List state belongs to `composables/useAsyncList.ts`.** It owns `items/total/current/pageSize/loading/error`, drops
   stale responses by sequence number and aborts superseded requests. Views must not reintroduce their own `load()`
   that mutates those refs directly, or cancellation and the empty-`error` contract break.
7. **The editors' dirty baseline is set after load and after save** (`markClean()`), which is why `/articles/42` and
   `/talks/7` can be opened and left in the read-only test paths without a confirmation dialog appearing.

### Facts corrected by this round

- The note in §1.13 claiming `PhotoView.vue` renders a `photo-tile` masonry with **no** `a-table` is stale:
  `/albums/:id` now renders a real `a-table` with `tr`/`td`, so the `.arco-table` / `getByRole('row')` /
  `row.getByRole('cell').nth(1)` expectations of the gated suites are satisfiable as written.
- `admin-shell.spec.ts` now contains **6** baseline tests (the 6th is
  `imports and exports articles, moves photos, and keeps filters in the URL`), so a run reports
  `6 passed / 5 skipped`. It is still 5 skipped: the `@integration` suites remain env-gated and are not run by CI.

### Reader-interaction counters (article list)

The article list gained two numeric columns backed by the reaction ledger
(`integration-full-crud.spec.ts`, step `reactions`):

- `[data-testid="article-like-count"]` — text of the like total for that row.
- `[data-testid="article-favorite-count"]` — text of the favourite total for that row.

Both live inside the row's own cells, so `row.getByTestId(...)` stays scoped to
one article. The counters are plain text (formatted through `formatNumber`), not
inputs, and the columns are registered in `ArticleListView.vue`'s column list —
removing them or moving the testids to a shared wrapper breaks the gated suite.

### Content performance page

`/content-performance` (`ContentPerformanceView.vue`) is a read-only analytics
surface under the article submenu. Its stable contract is intentionally small:

- The page root keeps the shared `.admin-page` container and the header exposes
  the four range controls (`7d`, `30d`, `90d`, `12m`) plus the common refresh
  button.
- The summary uses `AdminStatCard` for views, unique readers, average effective
  reading time and completion rate. The trend uses `AdminEChart`; the ranking
  uses the shared `.arco-table` and a row-scoped `查看详情` text button.
- The detail drawer loads `/admin/content/analytics/articles/{articleId}` and
  must not add a second page-wide `确定`/`编辑` action that can collide with the
  strict-mode locators documented in §0.
- Stable selectors are `[data-testid="content-performance-page"]`,
  `[data-testid="content-performance-views"]`,
  `[data-testid="content-performance-unique-readers"]`,
  `[data-testid="content-performance-avg-active-time"]`,
  `[data-testid="content-performance-completion-rate"]`,
  `[data-testid="content-performance-trend"]`,
  `[data-testid="content-performance-ranking"]` and
  `[data-testid="content-performance-detail"]`.
- The page never renders raw account, IP, User-Agent or referrer values; the
  reading-session endpoint accepts only `sessionId`, `activeMs` and
  `maxScrollPercent`.
- `npm run test:admin` is a blocking CI step for the `admin-next` frontend
  matrix entry; the blog matrix does not install Playwright browsers.

### Local verification commands

```bash
cd web
npx vue-tsc --noEmit -p apps/admin-next/tsconfig.json                       # 0 errors
npx vue-tsc --noEmit -p apps/admin-next/tsconfig.json \
  --noUnusedLocals --noUnusedParameters                                     # 0 errors
npm run build:admin                                                         # vite build
npm run test:admin                                                          # 8 passed / 5 skipped
```
