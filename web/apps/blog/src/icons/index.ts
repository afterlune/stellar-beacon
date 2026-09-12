import type { App } from 'vue'
import SvgIcon from '@/components/SvgIcon/index.vue'

export const registerSvgIcon = (app: App): void => {
  app.component('svg-icon', SvgIcon)
}
