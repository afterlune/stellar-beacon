/**
 * 模板里通过 `app.config.globalProperties` 注入的翻译函数
 * （见 `src/i18n/index.ts`），在这里补充类型，SFC 模板中的 `t(...)` 才有提示。
 *
 * 注意：这个文件必须是模块（无顶层 import/export 的 `.d.ts` 里写
 * `declare module 'vue'` 会**覆盖**整个 vue 模块，而不是做接口合并）。
 */
import 'vue'

declare module 'vue' {
  interface ComponentCustomProperties {
    t: (key: string, params?: Record<string, string | number>) => string
  }
}
