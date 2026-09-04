import MarkdownIt from 'markdown-it'
// These plugins are CommonJS packages without bundled TypeScript declarations.
// Vite handles their runtime interop; the explicit ignores keep the existing
// strict frontend type configuration usable during the migration.
// @ts-expect-error no declaration file is published by this plugin
import markdownItKatex from 'markdown-it-katex-external'
// @ts-expect-error no declaration file is published by this plugin
import { full as markdownItEmoji } from 'markdown-it-emoji'

export default function markdownToHtml(content: any) {
  const md = new MarkdownIt({
    html: true
  })
    .use(markdownItKatex)
    .use(markdownItEmoji)
  return md.render(content)
}
