import { onBeforeUnmount, onMounted, watch, type WatchSource } from 'vue'

export interface SeoMetaInput {
  title: string
  description?: string
  canonical?: string
  image?: string
  type?: string
  jsonLd?: Record<string, unknown>
}

const MANAGED_ATTR = 'data-stellar-seo'

export function useSeoMeta(source?: WatchSource<SeoMetaInput | null | undefined>): { setSeo: (value: SeoMetaInput) => void } {
  let activeNodes: HTMLElement[] = []

  const setSeo = (value: SeoMetaInput): void => {
    document.querySelectorAll(`[${MANAGED_ATTR}]`).forEach((node) => node.remove())
    activeNodes = []
    document.title = value.title
    setMeta('description', value.description || '', 'name', activeNodes)
    setMeta('og:title', value.title, 'property', activeNodes)
    setMeta('og:description', value.description || '', 'property', activeNodes)
    setMeta('og:type', value.type || 'website', 'property', activeNodes)
    setMeta('og:url', value.canonical || window.location.href, 'property', activeNodes)
    setMeta('twitter:card', value.image ? 'summary_large_image' : 'summary', 'name', activeNodes)
    setMeta('twitter:title', value.title, 'name', activeNodes)
    setMeta('twitter:description', value.description || '', 'name', activeNodes)
    if (value.image) {
      setMeta('og:image', value.image, 'property', activeNodes)
      setMeta('twitter:image', value.image, 'name', activeNodes)
    }
    const canonical = document.createElement('link')
    canonical.rel = 'canonical'
    canonical.href = value.canonical || window.location.href
    canonical.setAttribute(MANAGED_ATTR, 'true')
    document.head.appendChild(canonical)
    activeNodes.push(canonical)
    if (value.jsonLd) {
      const script = document.createElement('script')
      script.type = 'application/ld+json'
      script.textContent = JSON.stringify(value.jsonLd)
      script.setAttribute(MANAGED_ATTR, 'true')
      document.head.appendChild(script)
      activeNodes.push(script)
    }
  }

  onMounted(() => {
    if (source) watch(source, (value) => value && setSeo(value), { immediate: true })
  })
  onBeforeUnmount(() => activeNodes.forEach((node) => node.remove()))

  return { setSeo }
}

function setMeta(name: string, content: string, attribute: 'name' | 'property', nodes: HTMLElement[]): void {
  const meta = document.createElement('meta')
  meta.setAttribute(attribute, name)
  meta.content = content
  meta.setAttribute(MANAGED_ATTR, 'true')
  document.head.appendChild(meta)
  nodes.push(meta)
}
