import MarkdownIt from 'markdown-it'
import markdownEmoji from 'markdown-it-emoji'
import markdownKatexExternal from 'markdown-it-katex-external'

const markdown = new MarkdownIt({ html: true, linkify: true, breaks: true })

// markdown-it@15 removed utils.assign while the pinned emoji plugin still
// expects it during initialization.
const markdownUtils = markdown.utils as { assign?: typeof Object.assign }
if (typeof markdownUtils.assign !== 'function') markdownUtils.assign = Object.assign

markdown.use(markdownKatexExternal).use(markdownEmoji)

export function markdownToHtml(content: string): string {
  return markdown.render(content || '')
}

/**
 * The source preview runs inside the authenticated admin page, but it still
 * must not execute pasted scripts or event handlers while the author edits.
 */
export function sanitizePreviewHtml(content: string): string {
  if (typeof DOMParser === 'undefined') return content
  const document = new DOMParser().parseFromString(content, 'text/html')
  document.querySelectorAll('script, iframe, object, embed, style, link').forEach((node) => node.remove())
  document.querySelectorAll<HTMLElement>('*').forEach((element) => {
    for (const attribute of [...element.attributes]) {
      const name = attribute.name.toLowerCase()
      const value = attribute.value.trim().toLowerCase()
      if (name.startsWith('on') || ((name === 'href' || name === 'src') && /^(javascript:|data:|vbscript:)/.test(value))) {
        element.removeAttribute(attribute.name)
      }
      if (name === 'style' && /url\s*\(\s*["']?\s*(?:javascript|data):/i.test(value)) {
        element.removeAttribute(attribute.name)
      }
    }
  })
  return document.body.innerHTML
}
