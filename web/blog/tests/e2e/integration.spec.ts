import { expect, test } from '@playwright/test'

const realIntegration = process.env.E2E_REAL_INTEGRATION === '1'

test.describe('blog isolated integration', () => {
  test.skip(!realIntegration, 'set E2E_REAL_INTEGRATION=1 to run against an existing isolated Caddy stack')

  test('serves public routes and APIs through Caddy @integration', async ({ page, request }) => {
    const pageErrors: string[] = []
    const agentFeaturePath = '/api/agent/features'
    page.on('pageerror', (error) => pageErrors.push(error.message))
    page.on('console', (message) => {
      if (message.type() === 'error') {
        const location = message.location()
        if (message.text().includes('Failed to load resource') && new URL(location.url).pathname === agentFeaturePath) {
          return
        }
        pageErrors.push(`console: ${message.text()} (${location.url}:${location.lineNumber})`)
      }
    })
    page.on('response', (response) => {
      if (response.status() >= 400) {
        if (response.status() === 404 && new URL(response.url()).pathname === agentFeaturePath) return
        pageErrors.push(`http: ${response.status()} ${response.url()}`)
      }
    })

    for (const endpoint of ['/api/', '/api/articles/topAndFeatured', '/api/articles/all?current=1&size=10', '/api/categories/all', '/api/tags/all']) {
      const response = await request.get(endpoint)
      expect(response.status(), endpoint).toBe(200)
      const payload = await response.json()
      expect(payload.flag, endpoint).toBe(true)
    }

    const agentFeaturesResponse = await request.get(agentFeaturePath)
    expect([200, 404], agentFeaturePath).toContain(agentFeaturesResponse.status())
    if (agentFeaturesResponse.status() === 200) {
      const payload = await agentFeaturesResponse.json()
      expect(payload.flag, agentFeaturePath).toBe(true)
      expect(payload.data, agentFeaturePath).toEqual(
        expect.objectContaining({
          publicChat: expect.any(Boolean),
          vitals: expect.any(Boolean),
          galaxy: expect.any(Boolean),
          dreams: expect.any(Boolean),
          capsules: expect.any(Boolean),
          radio: expect.any(Boolean),
          videos: expect.any(Boolean),
          ttsEnabled: expect.any(Boolean)
        })
      )
    }

    const response = await page.goto('/')
    expect(response?.status()).toBe(200)
    await expect(page.locator('#app')).toBeVisible()
    await expect(page.locator('#App-Container')).toBeVisible()

    for (const route of ['/archives', '/tags', '/about']) {
      const routeResponse = await page.goto(route)
      expect(routeResponse?.status(), route).toBe(200)
      await expect(page.locator('#App-Container')).toBeVisible()
      expect(new URL(page.url()).pathname).toBe(route)
    }

    expect(pageErrors).toEqual([])
  })
})
