export default function markdownToHtml(content: any) {
  const MarkdownIt = require('markdown-it')
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
  md.use(require('markdown-it-katex-external')).use(require('markdown-it-emoji'))
  return md.render(content)
}
