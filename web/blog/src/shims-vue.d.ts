/* eslint-disable */
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
declare module '*.svg' {
  const source: string
  export default source
}
declare module 'vue-avatar-cropper'
declare module 'js-cookie'
