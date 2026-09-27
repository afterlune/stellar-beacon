import MarkdownIt from 'markdown-it'
import markdownEmoji from 'markdown-it-emoji'
import markdownKatexExternal from 'markdown-it-katex-external'
import katex from 'katex'
import 'katex/dist/katex.min.css'

// Keep the existing markdown-it plugin, but provide its KaTeX renderer from the
// local bundle instead of an external global script.
;(window as Window & { katex?: typeof katex }).katex = katex

export default function markdownToHtml(content: string) {
  const md = new MarkdownIt({
    html: true
  })
  // markdown-it@15 removed utils.assign, while the pinned emoji plugin still
  // uses it during initialization. Keep the existing plugin behavior without
  // downgrading the Markdown parser.
  const markdownUtils = md.utils as { assign?: typeof Object.assign }
  if (typeof markdownUtils.assign !== 'function') {
    markdownUtils.assign = Object.assign
  }
  md.use(markdownKatexExternal).use(markdownEmoji)
  return md.render(content)
}

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
