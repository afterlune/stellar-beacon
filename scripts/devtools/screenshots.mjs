#!/usr/bin/env node
/**
 * Screenshot tour for the admin console.
 *
 * Signs in against the running API, then walks a list of routes (or every route
 * the signed-in account can see) and writes a screenshot per route plus a JSON
 * report with console errors and failed requests.
 *
 * Usage:
 *   node scripts/devtools/shots.mjs --base http://127.0.0.1:8082 --out .screenshots/tour
 *   node scripts/devtools/shots.mjs --routes /,/articles --theme light
 *   node scripts/devtools/shots.mjs --cdp http://127.0.0.1:9222   # drive an existing browser
 *
 * Options:
 *   --base <url>      app origin (default http://127.0.0.1:8082)
 *   --out <dir>       screenshot output directory (default .screenshots/tour)
 *   --routes <list>   comma separated routes; default = every menu route
 *   --theme <name>    dark | light (default dark)
 *   --width/--height  viewport size (default 1600x1000)
 *   --cdp <url>       attach to an existing Chromium/Edge instead of launching
 *   --full            also capture a full-page screenshot per route
 *   --user/--pass     credentials (default integration admin)
 *   --settle <ms>     extra settle time before capture (default 900)
 */
import { mkdirSync, writeFileSync, existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

// The browser dependency lives in the web workspace, so resolve it explicitly
// instead of relying on this script's own (dependency-free) directory.
async function loadPlaywright() {
  const candidates = [
    '../../web/apps/admin-next/node_modules/@playwright/test/index.mjs',
    '../../web/node_modules/@playwright/test/index.mjs',
    '../../web/apps/admin-next/node_modules/playwright-core/index.mjs'
  ]
  for (const candidate of candidates) {
    const path = resolve(import.meta.dirname, candidate)
    if (existsSync(path)) return import(pathToFileURL(path).href)
  }
  return import('@playwright/test')
}
const { chromium } = await loadPlaywright()

const argv = process.argv.slice(2)
function arg(name, fallback) {
  const i = argv.indexOf(`--${name}`)
  return i >= 0 ? argv[i + 1] : fallback
}
const hasFlag = (name) => argv.includes(`--${name}`)

const base = arg('base', 'http://127.0.0.1:8082')
const out = resolve(arg('out', '.screenshots/tour'))
const theme = arg('theme', 'dark')
const width = Number(arg('width', 1600))
const height = Number(arg('height', 1000))
const settle = Number(arg('settle', 900))
const cdp = arg('cdp', '')
const full = hasFlag('full')
const username = arg('user', process.env.E2E_ADMIN_EMAIL || 'e2e-admin@example.test')
const password = arg('pass', process.env.E2E_ADMIN_PASSWORD || 'IntegrationAdmin123!')
const evalExpr = arg('eval', '')
const clipSelector = arg('clip', '')
const clickSelector = arg('click', '')
const preEvalExpr = arg('pre-eval', '')

mkdirSync(out, { recursive: true })

function slug(route) {
  return route.replace(/^\//, '').replace(/[^\w-]+/g, '_') || 'root'
}

async function signIn(context) {
  const form = new URLSearchParams({ username, password })
  const response = await context.request.post(`${base}/api/v1/auth/login`, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    data: form.toString()
  })
  const body = await response.json()
  if (body.code !== 'OK' || !body.data?.token) {
    throw new Error(`login failed: ${response.status()} ${JSON.stringify(body).slice(0, 200)}`)
  }
  return body.data
}

async function menuRoutes(context, token) {
  const response = await context.request.get(`${base}/api/v1/admin/me/menu`, {
    headers: { Authorization: `Bearer ${token}` }
  })
  const body = await response.json()
  const routes = []
  const walk = (nodes, parentVisible = true) => {
    for (const node of nodes || []) {
      const visible = parentVisible && node.isHidden !== 1 && node.isHidden !== true
      const isPattern = node.path && (node.path.includes(':') || node.path.includes('*'))
      if (visible && node.path && node.path !== '/' && !isPattern && !routes.includes(node.path)) routes.push(node.path)
      if (node.children?.length) walk(node.children, visible)
    }
  }
  walk(body.data || [])
  return routes
}

const report = { base, theme, routes: [], console: [], network: [], startedAt: new Date().toISOString() }

const browser = cdp
  ? await chromium.connectOverCDP(cdp)
  : await chromium.launch({ headless: true, args: ['--force-color-profile=srgb', '--font-render-hinting=none'] })

const context = cdp ? (browser.contexts()[0] ?? (await browser.newContext())) : await browser.newContext({
  viewport: { width, height },
  deviceScaleFactor: Number(arg('scale', 1)),
  colorScheme: theme === 'dark' ? 'dark' : 'light'
})

const user = await signIn(context)
await context.addInitScript(
  ([token, userJson, themeName, collapsed]) => {
    sessionStorage.setItem('token', token)
    sessionStorage.setItem('stellar-beacon.admin.user', userJson)
    localStorage.setItem('stellar-beacon.admin.theme', themeName)
    localStorage.setItem('stellar-beacon.admin.sider-collapsed', collapsed ? '1' : '0')
  },
  [user.token, JSON.stringify(user), theme, hasFlag('collapsed')]
)

const page = cdp ? (context.pages()[0] ?? (await context.newPage())) : await context.newPage()
if (cdp) await page.setViewportSize({ width, height })

page.on('console', (message) => {
  if (message.type() === 'error' || message.type() === 'warning') {
    report.console.push({ type: message.type(), text: message.text(), url: page.url() })
  }
})
page.on('pageerror', (error) => report.console.push({ type: 'pageerror', text: String(error), url: page.url() }))
page.on('response', (response) => {
  if (response.status() >= 400) report.network.push({ status: response.status(), url: response.url() })
})

const routeList = arg('routes', '')
  ? arg('routes', '').split(',').map((r) => r.trim()).filter(Boolean)
  : ['/', ...(await menuRoutes(context, user.token))]

for (const route of routeList) {
  const url = `${base}${route}`
  const entry = { route, url, screenshots: [], errors: [] }
  try {
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await page.waitForSelector('.admin-content, .admin-shell', { timeout: 15000 }).catch(() => {})
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {})
    await page.waitForTimeout(settle)
    if (clickSelector) {
      await page.locator(clickSelector).first().click({ timeout: 8000 }).catch((error) => {
        entry.errors.push(`click ${clickSelector}: ${String(error.message || error)}`)
      })
      await page.waitForTimeout(400)
    }
    if (preEvalExpr) {
      await page.evaluate(preEvalExpr).catch((error) => {
        entry.errors.push(`pre-eval: ${String(error.message || error)}`)
      })
      await page.waitForTimeout(Number(arg('pre-eval-wait', 600)))
    }
    const suffix = clipSelector ? '.clip' : ''
    const viewPath = `${out}/${slug(route)}${suffix}.png`
    if (clipSelector) {
      const target = page.locator(clipSelector).first()
      await target.screenshot({ path: viewPath })
    } else {
      await page.screenshot({ path: viewPath })
    }
    entry.screenshots.push(viewPath)
    if (full) {
      const fullPath = `${out}/${slug(route)}.full.png`
      await page.screenshot({ path: fullPath, fullPage: true })
      entry.screenshots.push(fullPath)
    }
    entry.title = await page.title()
    entry.heading = await page.locator('.admin-page-header h2, h2, h1').first().textContent().catch(() => null)
    if (evalExpr) {
      entry.eval = await page.evaluate(evalExpr).catch((error) => ({ __error: String(error.message || error) }))
    }
  } catch (error) {
    entry.errors.push(String(error.message || error))
  }
  report.routes.push(entry)
}

report.finishedAt = new Date().toISOString()
writeFileSync(`${out}/report.json`, JSON.stringify(report, null, 2))
console.log(JSON.stringify({
  out,
  routes: report.routes.map((r) => ({ route: r.route, heading: r.heading, errors: r.errors })),
  console: report.console,
  network: report.network
}, null, 2))

await browser.close()
