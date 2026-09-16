/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  const component: DefineComponent<object, object, unknown>
  export default component
}

declare module 'markdown-it-emoji' {
  import type MarkdownIt from 'markdown-it'

  const plugin: (md: MarkdownIt) => void
  export default plugin
}

declare module 'markdown-it-katex-external' {
  import type MarkdownIt from 'markdown-it'

  const plugin: (md: MarkdownIt) => void
  export default plugin
}
