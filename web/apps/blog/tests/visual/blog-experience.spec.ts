import { expect, test, type Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

const adminEmail = process.env.E2E_ADMIN_EMAIL || ''
const adminPassword = process.env.E2E_ADMIN_PASSWORD || ''
// The authenticated reader routes are only audited when the run can obtain a
// real session; a fake token would only screenshot an error state. Every other
// route keeps the historical fake token so the audit baseline is unchanged.
const authenticatedRoutes = adminEmail && adminPassword
  ? [
      { name: 'for-you', path: '/for-you', authenticated: true },
      { name: 'following', path: '/following', authenticated: true },
      { name: 'studio-collections', path: '/studio/collections', authenticated: true },
      { name: 'notifications', path: '/notifications', authenticated: true }
    ]
  : []

const routes = [
  { name: 'home', path: '/' },
  { name: 'author', path: '/u/admin' },
  { name: 'article', path: '/articles/158' },
  { name: 'archives', path: '/archives' },
  { name: 'tags', path: '/tags' },
  { name: 'topics', path: '/topics' },
  { name: 'collections', path: '/collections' },
  { name: 'categories', path: '/categories' },
  { name: 'series', path: '/series' },
  { name: 'talks', path: '/talks' },
  { name: 'photos', path: '/photos/11' },
  { name: 'search', path: '/search?q=java' },
  { name: 'reading', path: '/studio/library/reading' },
  { name: 'friends', path: '/friends' },
  { name: 'about', path: '/about' },
  { name: 'not-found', path: '/nope' }
]

const themes = ['theme-dark', 'theme-light']
let session: { token: string; userInfo: Record<string, unknown> } | null = null
const outputRoot = 'test-results/visual/screens'

async function browse(page: Page, route: string, theme: string, authenticated = false): Promise<void> {
  await page.context().addInitScript(({ themeName, session: current }) => {
    document.cookie = 'locale=cn; path=/'
    document.cookie = `theme=${themeName}; path=/`
    sessionStorage.setItem('token', current?.token || 'visual-reading-token')
    if (current) {
      sessionStorage.setItem('userStore', JSON.stringify({
        userVisible: false, userInfo: current.userInfo, token: current.token, accessArticles: [], tab: 0, page: 1
      }))
    }
  }, { themeName: theme, session: authenticated ? session : null })
  await page.goto(route, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
}

async function auditPage(page: Page, mobile: boolean, light: boolean) {
  const cdp = await page.context().newCDPSession(page)
  const ax = await cdp.send('Accessibility.getFullAXTree').catch(() => ({ nodes: [] as any[] }))
  const nodes: any[] = (ax as any).nodes || []
  const unnamedInteractive = nodes
    .filter((node) => ['link', 'button', 'textbox', 'combobox'].includes(node.role?.value))
    .filter((node) => !String(node.name?.value || '').trim())
    .map((node) => String(node.role?.value))

  const dom = await page.evaluate((measureContrast: boolean) => {
    const root = document.documentElement
    const overflow = Array.from(document.querySelectorAll('body *'))
      .filter((node) => node instanceof HTMLElement)
      .map((node) => ({ node: node as HTMLElement, rect: (node as HTMLElement).getBoundingClientRect() }))
      .filter((item) => item.rect.width > 0 && item.rect.height > 0 && (item.rect.right > window.innerWidth + 2 || item.rect.left < -2))
      .slice(0, 5)
      .map((item) => `${item.node.tagName.toLowerCase()}.${String(item.node.className || '').slice(0, 40)}`)
    const smallTargets = Array.from(document.querySelectorAll('a, button, [role="button"]'))
      .filter((node) => node instanceof HTMLElement)
      .map((node) => ({ node: node as HTMLElement, rect: (node as HTMLElement).getBoundingClientRect() }))
      .filter((item) => item.rect.width > 0 && item.rect.height > 0 && item.rect.height < 24)
      .slice(0, 5)
      .map((item) => `${(item.node.textContent || '').trim().slice(0, 20)} h=${Math.round(item.rect.height)}`)
    const contrast: string[] = []
    if (measureContrast) {
      const parse = (value: string) => {
        const match = String(value).match(/rgba?\(([^)]+)\)/)
        if (!match) return null
        const parts = match[1].split(',').map((item) => Number(item.trim()))
        return parts.length < 3 ? null : { r: parts[0], g: parts[1], b: parts[2], a: parts.length > 3 ? parts[3] : 1 }
      }
      const luminance = (color: { r: number; g: number; b: number }) => {
        const channel = (value: number) => {
          const v = value / 255
          return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
        }
        return 0.2126 * channel(color.r) + 0.7152 * channel(color.g) + 0.0722 * channel(color.b)
      }
      const background = (element: Element) => {
        let node: Element | null = element
        while (node) {
          const style = getComputedStyle(node)
          if (style.backgroundImage !== 'none') return null
          const color = parse(style.backgroundColor)
          if (color && color.a > 0.5) return color
          node = node.parentElement
        }
        return { r: 255, g: 255, b: 255, a: 1 }
      }
      for (const node of document.querySelectorAll('p, span, a, li, h1, h2, h3, em, strong, label, button')) {
        if (!(node instanceof HTMLElement) || node.children.length > 0) continue
        const text = (node.textContent || '').trim()
        if (text.length < 2) continue
        const rect = node.getBoundingClientRect()
        if (rect.width === 0 || rect.height === 0) continue
        const style = getComputedStyle(node)
        const fg = parse(style.color)
        const bg = background(node)
        if (!fg || !bg) continue
        const ratio = (Math.max(luminance(fg), luminance(bg)) + 0.05) / (Math.min(luminance(fg), luminance(bg)) + 0.05)
        const size = Number(style.fontSize.replace('px', ''))
        const threshold = size >= 24 ? 3 : 4.5
        if (ratio < threshold) contrast.push(`${text.slice(0, 18)} ${Math.round(ratio * 100) / 100}`)
        if (contrast.length >= 6) break
      }
    }
    return {
      horizontalOverflow: root.scrollWidth > window.innerWidth + 1,
      h1Count: document.querySelectorAll('h1').length,
      imagesWithoutAlt: Array.from(document.images).filter((image) => !image.hasAttribute('alt')).length,
      overflow,
      smallTargets,
      contrast
    }
  }, light)

  return { ...dom, unnamedInteractive, mobile }
}

test.describe('blog visual gate', () => {
  test.beforeAll(async ({ request }) => {
    const probe = await request.get('/api/v1/public/').catch(() => null)
    test.skip(!probe || !probe.ok(), 'visual gate needs a reachable blog environment')
    if (!adminEmail || !adminPassword) return
    const login = await request.post('/api/v1/auth/login', { form: { username: adminEmail, password: adminPassword } }).catch(() => null)
    if (!login || !login.ok()) return
    const payload = await login.json().catch(() => null)
    const token = String(payload?.data?.token || '')
    if (token) session = { token, userInfo: payload.data }
  })

  for (const theme of themes) {
    test(`renders every route in ${theme}`, async ({ page }, testInfo) => {
      const mobile = testInfo.project.name === 'mobile'
      const outputDir = path.join(outputRoot, theme, testInfo.project.name)
      fs.mkdirSync(outputDir, { recursive: true })
      const failures: string[] = []

      for (const route of [...routes, ...authenticatedRoutes]) {
        await browse(page, route.path, theme, 'authenticated' in route && Boolean(route.authenticated))
        const audit = await auditPage(page, mobile, theme === 'theme-light')
        await page.screenshot({ path: path.join(outputDir, `${route.name}.png`) })
        await page.screenshot({ path: path.join(outputDir, `${route.name}-full.png`), fullPage: true })

        if (audit.horizontalOverflow) failures.push(`${route.name}: horizontal overflow`)
        if (audit.h1Count > 1) failures.push(`${route.name}: ${audit.h1Count} h1 elements`)
        if (audit.imagesWithoutAlt > 0) failures.push(`${route.name}: ${audit.imagesWithoutAlt} image(s) without alt`)
        if (audit.unnamedInteractive.length) failures.push(`${route.name}: unnamed ${audit.unnamedInteractive.join('/')}`)
        if (theme === 'theme-light' && audit.contrast.length) failures.push(`${route.name}: contrast ${audit.contrast.join(' | ')}`)
        if (mobile && audit.smallTargets.length) failures.push(`${route.name}: small targets ${audit.smallTargets.join(' | ')}`)
      }

      expect(failures, failures.join('\n')).toEqual([])
    })
  }

  test('home stays within the runtime budget under slow 4G', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'performance budget measured on desktop only')
    // The budget covers what this repository ships. The KaTeX CDN is a
    // third-party dependency whose latency the page cannot control, and the
    // shell tests already block it for the same reason.
    await page.route('https://cdnjs.cloudflare.com/**', (route) => route.abort())
    await page.context().addInitScript(() => {
      document.cookie = 'locale=cn; path=/'
      document.cookie = 'theme=theme-dark; path=/'
      const vitals = { lcp: 0, cls: 0 }
      ;(window as any).__vitals = vitals
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) vitals.lcp = Math.max(vitals.lcp, entry.startTime)
      }).observe({ type: 'largest-contentful-paint', buffered: true })
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (!(entry as any).hadRecentInput) vitals.cls += (entry as any).value
        }
      }).observe({ type: 'layout-shift', buffered: true })
    })
    const cdp = await page.context().newCDPSession(page)
    await cdp.send('Network.enable')
    await cdp.send('Network.emulateNetworkConditions', {
      offline: false,
      latency: 300,
      downloadThroughput: Math.round((1.6 * 1024 * 1024) / 8),
      uploadThroughput: Math.round((750 * 1024) / 8)
    })
    // The budget measures first paint, so it must not wait for the load event:
    // that event also waits for the third-party KaTeX CDN, whose latency is not
    // something the page can control and would otherwise dominate the result.
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)
    const vitals = await page.evaluate(() => (window as any).__vitals)
    expect(vitals.lcp, 'LCP under slow 4G').toBeLessThan(4000)
    expect(vitals.cls, 'CLS under slow 4G').toBeLessThan(0.1)
  })
})
