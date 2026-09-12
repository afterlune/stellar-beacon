import MarkdownIt from 'markdown-it'
import markdownEmoji from 'markdown-it-emoji'
import markdownKatexExternal from 'markdown-it-katex-external'

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
