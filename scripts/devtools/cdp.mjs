#!/usr/bin/env node
/**
 * Minimal Chrome DevTools Protocol session runner.
 *
 * Connects to an already-running browser that exposes a browser-level CDP
 * WebSocket (e.g. Edge/Chrome "Allow remote debugging" on 127.0.0.1:9222, or a
 * browser started with --remote-debugging-port). It executes a JSON *plan* of
 * steps in one connection so that console errors, failed requests and
 * screenshots all describe the same session.
 *
 * Usage:
 *   node scripts/devtools/cdp.mjs <plan.json> [--cdp ws://127.0.0.1:9222/devtools/browser]
 *
 * Plan: { "name": "...", "viewport": {"width":1600,"height":1000}, "steps": [ ... ] }
 *
 * Steps:
 *   {"type":"open","url":"..."}                  open (or reuse) a tab
 *   {"type":"goto","url":"..."}                  navigate the active tab
 *   {"type":"wait","ms":500}                     sleep
 *   {"type":"waitFor","selector":"...","timeout":8000}
 *   {"type":"click","selector":"...","nth":0}    real mouse click at element center
 *   {"type":"fill","selector":"...","text":"...","nth":0}
 *   {"type":"press","key":"Enter"}               key event on the focused element
 *   {"type":"eval","expr":"...","label":"..."}   evaluate JS, print JSON result
 *   {"type":"shot","path":"...png","full":false,"selector":null}
 *   {"type":"log"}                               print collected console/network problems
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'

const args = process.argv.slice(2)
const planPath = args.find((a) => !a.startsWith('--'))
if (!planPath) {
  console.error('usage: cdp.mjs <plan.json> [--cdp <ws-url>]')
  process.exit(2)
}
const cdpArgIndex = args.indexOf('--cdp')
const CDP_WS =
  (cdpArgIndex >= 0 ? args[cdpArgIndex + 1] : undefined) ||
  process.env.CDP_WS ||
  'ws://127.0.0.1:9222/devtools/browser'

const plan = JSON.parse(readFileSync(planPath, 'utf8'))
const viewport = plan.viewport || { width: 1600, height: 1000 }

let ws = null
let nextId = 1
const pending = new Map()
const consoleMessages = []
const networkProblems = []
let sessionId = null
let targetId = null

function send(method, params = {}, sid = null, timeoutMs = 20000) {
  return new Promise((res, reject) => {
    const id = nextId++
    const timer = setTimeout(() => {
      pending.delete(id)
      reject(new Error(`${method}: timed out after ${timeoutMs}ms`))
    }, timeoutMs)
    pending.set(id, {
      res: (value) => {
        clearTimeout(timer)
        res(value)
      },
      reject: (error) => {
        clearTimeout(timer)
        reject(error)
      },
      method
    })
    ws.send(JSON.stringify({ id, method, params, ...(sid ? { sessionId: sid } : {}) }))
  })
}

function describe(value) {
  if (value === null || value === undefined) return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function handleEvent(msg) {
  const { method, params } = msg
  if (method === 'Runtime.consoleAPICalled') {
    const text = (params.args || [])
      .map((a) => (a.value !== undefined ? a.value : a.description || a.type))
      .join(' ')
    consoleMessages.push({ kind: params.type, text })
  } else if (method === 'Runtime.exceptionThrown') {
    const d = params.exceptionDetails || {}
    consoleMessages.push({
      kind: 'pageerror',
      text: d.exception?.description || d.text || 'exception'
    })
  } else if (method === 'Log.entryAdded') {
    const e = params.entry || {}
    if (e.level === 'error' || e.level === 'warning') {
      consoleMessages.push({ kind: e.level, text: `${e.text}${e.url ? ` (${e.url})` : ''}` })
    }
  } else if (method === 'Network.responseReceived') {
    const r = params.response || {}
    if (r.status >= 400 && !String(r.url).startsWith('data:')) {
      networkProblems.push({ status: r.status, url: r.url, type: params.type })
    }
  } else if (method === 'Network.loadingFailed') {
    networkProblems.push({ status: 0, url: params.requestId, error: params.errorText })
  }
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

async function attach(target) {
  targetId = target.targetId
  const { sessionId: sid } = await send('Target.attachToTarget', { targetId, flatten: true })
  sessionId = sid
  await send('Page.enable', {}, sid)
  await send('Runtime.enable', {}, sid)
  await send('Log.enable', {}, sid)
  await send('Network.enable', {}, sid)
  await send('DOM.enable', {}, sid)
  await send('Emulation.setDeviceMetricsOverride', {
    width: viewport.width,
    height: viewport.height,
    deviceScaleFactor: 1,
    mobile: false
  }, sid)
}

async function ensureSession() {
  if (sessionId) return
  const { targetInfos } = await send('Target.getTargets')
  const pages = targetInfos.filter((t) => t.type === 'page')
  const reusable =
    pages.find((t) => /127\.0\.0\.1:8082|localhost:8082/.test(t.url)) ||
    pages.find((t) => t.url === 'about:blank') ||
    pages[0]
  if (reusable) {
    await attach(reusable)
    return
  }
  const { targetId: newId } = await send('Target.createTarget', { url: 'about:blank' })
  await attach({ targetId: newId })
}

async function evalIn(expr) {
  const result = await send('Runtime.evaluate', {
    expression: `(function(){ try { return (${expr}) } catch (e) { return { __error: String(e) } } })()`,
    awaitPromise: true,
    returnByValue: true
  }, sessionId)
  if (result.exceptionDetails) {
    throw new Error(result.exceptionDetails.exception?.description || 'evaluate failed')
  }
  return result.result?.value
}

async function nodeIdFor(selector, nth = 0) {
  const { root } = await send('DOM.getDocument', { depth: 1 }, sessionId)
  const { nodeIds } = await send('DOM.querySelectorAll', { nodeId: root.nodeId, selector }, sessionId)
  const id = nodeIds?.[nth]
  return id || 0
}

async function centerOf(selector, nth = 0) {
  const nodeId = await nodeIdFor(selector, nth)
  if (!nodeId) return null
  const { model } = await send('DOM.getBoxModel', { nodeId }, sessionId)
  const q = model.content
  return {
    x: (q[0] + q[2] + q[4] + q[6]) / 4,
    y: (q[1] + q[3] + q[5] + q[7]) / 4
  }
}

async function clickAt(x, y) {
  await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y, button: 'none', clickCount: 0 }, sessionId)
  await send('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 }, sessionId)
  await send('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 }, sessionId)
}

const report = { name: plan.name || planPath, steps: [], screenshots: [] }

async function connectWithRetry(attempts = 8, perAttemptMs = 12000) {
  let lastError = null
  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    try {
      await new Promise((res, reject) => {
        const socket = new WebSocket(CDP_WS)
        const timer = setTimeout(() => {
          try {
            socket.close()
          } catch {
            /* ignore */
          }
          reject(new Error(`timed out connecting to ${CDP_WS}`))
        }, perAttemptMs)
        socket.onopen = () => {
          clearTimeout(timer)
          ws = socket
          res()
        }
        socket.onerror = (e) => {
          clearTimeout(timer)
          reject(new Error(`cannot connect to ${CDP_WS}: ${e.message || 'error'}`))
        }
      })
      return
    } catch (error) {
      lastError = error
      if (attempt < attempts) await new Promise((r) => setTimeout(r, 1500))
    }
  }
  throw lastError || new Error(`cannot connect to ${CDP_WS}`)
}

async function run() {
  await connectWithRetry()
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.id) {
      const entry = pending.get(msg.id)
      if (!entry) return
      pending.delete(msg.id)
      if (msg.error) entry.reject(new Error(`${entry.method}: ${msg.error.message}`))
      else entry.res(msg.result)
      return
    }
    handleEvent(msg)
  }

  for (const [index, step] of (plan.steps || []).entries()) {
    const label = `#${index + 1} ${step.type}`
    try {
      switch (step.type) {
        case 'open': {
          const { targetInfos } = await send('Target.getTargets')
          const match = targetInfos.find((t) => t.type === 'page' && t.url.startsWith(step.url))
          if (match && step.reuse !== false) {
            if (sessionId) await send('Target.closeTarget', { targetId }).catch(() => {})
            await attach(match)
          } else {
            const { targetId: newId } = await send('Target.createTarget', { url: step.url })
            if (sessionId) await send('Target.closeTarget', { targetId }).catch(() => {})
            await attach({ targetId: newId })
          }
          report.steps.push({ label, ok: true, url: step.url })
          break
        }
        case 'goto': {
          await ensureSession()
          await send('Page.navigate', { url: step.url }, sessionId)
          report.steps.push({ label, ok: true, url: step.url })
          break
        }
        case 'wait': {
          await sleep(step.ms || 300)
          report.steps.push({ label, ok: true })
          break
        }
        case 'viewport': {
          await ensureSession()
          await send('Emulation.setDeviceMetricsOverride', {
            width: step.width || viewport.width,
            height: step.height || viewport.height,
            deviceScaleFactor: step.scale || 1,
            mobile: false
          }, sessionId)
          report.steps.push({ label, ok: true })
          break
        }
        case 'waitFor': {
          await ensureSession()
          const deadline = Date.now() + (step.timeout || 8000)
          let found = false
          while (Date.now() < deadline) {
            if (await nodeIdFor(step.selector, step.nth || 0)) {
              found = true
              break
            }
            await sleep(120)
          }
          report.steps.push({ label, ok: found, selector: step.selector })
          if (!found) throw new Error(`waitFor timeout: ${step.selector}`)
          break
        }
        case 'click': {
          await ensureSession()
          const point = await centerOf(step.selector, step.nth || 0)
          if (!point) throw new Error(`click target not found: ${step.selector}`)
          await clickAt(point.x, point.y)
          report.steps.push({ label, ok: true, selector: step.selector })
          break
        }
        case 'fill': {
          await ensureSession()
          const done = await evalIn(`(() => {
            const nodes = [...document.querySelectorAll(${JSON.stringify(step.selector)})]
            const el = nodes[${step.nth || 0}]
            if (!el) return false
            el.focus()
            const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement : HTMLInputElement
            const setter = Object.getOwnPropertyDescriptor(proto.prototype, 'value').set
            setter.call(el, ${JSON.stringify(step.text ?? '')})
            el.dispatchEvent(new Event('input', { bubbles: true }))
            el.dispatchEvent(new Event('change', { bubbles: true }))
            return true
          })()`)
          report.steps.push({ label, ok: Boolean(done), selector: step.selector })
          if (!done) throw new Error(`fill target not found: ${step.selector}`)
          break
        }
        case 'press': {
          await ensureSession()
          const keys = { Enter: { windowsVirtualKeyCode: 13, key: 'Enter', code: 'Enter', text: '\r' }, Escape: { windowsVirtualKeyCode: 27, key: 'Escape', code: 'Escape' }, Tab: { windowsVirtualKeyCode: 9, key: 'Tab', code: 'Tab' } }
          const info = keys[step.key] || { key: step.key }
          await send('Input.dispatchKeyEvent', { type: 'keyDown', ...info }, sessionId)
          await send('Input.dispatchKeyEvent', { type: 'keyUp', ...info }, sessionId)
          report.steps.push({ label, ok: true, key: step.key })
          break
        }
        case 'eval': {
          await ensureSession()
          const value = await evalIn(step.expr)
          report.steps.push({ label: step.label || label, ok: true, value })
          break
        }
        case 'shot': {
          await ensureSession()
          const outPath = resolve(step.path)
          mkdirSync(dirname(outPath), { recursive: true })
          const params = { format: 'png', captureBeyondViewport: Boolean(step.full) }
          if (step.selector) {
            const point = await centerOf(step.selector, step.nth || 0)
            if (!point) throw new Error(`shot target not found: ${step.selector}`)
            const clip = await evalIn(`(() => {
              const el = document.querySelector(${JSON.stringify(step.selector)})
              const r = el.getBoundingClientRect()
              return { x: r.x + window.scrollX, y: r.y + window.scrollY, width: r.width, height: r.height, scale: 1 }
            })()`)
            params.clip = clip
          }
          if (step.full) {
            const metrics = await send('Page.getLayoutMetrics', {}, sessionId)
            const size = metrics.cssContentSize || metrics.contentSize
            params.clip = { x: 0, y: 0, width: size.width, height: size.height, scale: 1 }
          }
          if (step.quality !== undefined) params.quality = step.quality
          const { data } = await send('Page.captureScreenshot', params, sessionId)
          writeFileSync(outPath, Buffer.from(data, 'base64'))
          report.steps.push({ label, ok: true, path: outPath })
          report.screenshots.push(outPath)
          break
        }
        case 'log': {
          report.steps.push({ label, ok: true })
          break
        }
        default:
          throw new Error(`unknown step type: ${step.type}`)
      }
    } catch (error) {
      report.steps.push({ label, ok: false, error: String(error.message || error) })
      if (step.optional) continue
      break
    }
  }

  report.console = consoleMessages
  report.network = networkProblems
  console.log(JSON.stringify(report, null, 2))
  try {
    if (sessionId) await send('Target.detachFromTarget', { sessionId }, null, 3000).catch(() => {})
    ws.close()
  } catch {
    /* ignore */
  }
  process.exit(0)
}

run().catch((error) => {
  console.error(JSON.stringify({ ...report, fatal: String(error.message || error) }, null, 2))
  process.exit(1)
})
