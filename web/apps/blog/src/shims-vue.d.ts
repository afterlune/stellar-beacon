/* eslint-disable */
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module 'prismjs/components/*'
declare module 'prismjs/plugins/*'

declare module '@vitejs/plugin-vue' {
  import type { PluginOption } from 'vite'
  const vue: () => PluginOption
  export default vue
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

declare module 'vue-avatar-cropper'
declare module 'js-cookie'
