import SvgIcon from '@/components/SvgIcon/index.vue' // svg component
import { App } from 'vue'

// register globally
export const registerSvgIcon = (app: App): void => {
  app.component('svg-icon', SvgIcon)
  if (typeof document === 'undefined' || document.getElementById('benetnasch-svg-sprite')) return

  const icons = import.meta.glob('./svg/*.svg', {
    eager: true,
    query: '?raw',
    import: 'default'
  }) as Record<string, string>
  if (Object.keys(icons).length === 0) {
    if (typeof require !== 'undefined' && typeof require.context === 'function') {
      const context = require.context('./svg', false, /\.svg$/i)
      context.keys().forEach(context)
    }
    return
  }
  const sprite = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
  sprite.id = 'benetnasch-svg-sprite'
  sprite.setAttribute('aria-hidden', 'true')
  sprite.style.position = 'absolute'
  sprite.style.width = '0'
  sprite.style.height = '0'
  sprite.style.overflow = 'hidden'

  Object.entries(icons).forEach(([path, source]) => {
    const name = path.split('/').pop()?.replace(/\.svg$/i, '')
    if (!name) return
    const parsed = new DOMParser().parseFromString(source, 'image/svg+xml').documentElement
    if (parsed.nodeName.toLowerCase() !== 'svg') return
    const symbol = document.createElementNS('http://www.w3.org/2000/svg', 'symbol')
    symbol.id = `icon-${name}`
    Array.from(parsed.attributes).forEach((attribute) => {
      if (attribute.name !== 'xmlns') symbol.setAttribute(attribute.name, attribute.value)
    })
    Array.from(parsed.childNodes).forEach((child) => {
      symbol.appendChild(document.importNode(child, true))
    })
    sprite.appendChild(symbol)
  })
  document.body.prepend(sprite)
}
