import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './router/guard'
import '@/styles/index.scss'
import 'normalize.css/normalize.css'
import { createPinia } from 'pinia'
import { i18n } from './locales'
import VueClickAway from 'vue3-click-away'
import lazyPlugin from 'vue3-lazy'
import { registerSvgIcon } from '@/icons'
import { registerObSkeleton } from '@/components/LoadingSkeleton'
import 'prismjs/themes/prism.css'
import 'prismjs'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import infiniteScroll from 'vue3-infinite-scroll-better'
import v3ImgPreview from 'v3-img-preview'
import api from './api/api'
import { notify } from '@/services/notifications'
import defaultCover from '@/assets/default-cover.jpg'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)
export const app = createApp(App)
  .use(router)
  .use(pinia)
  .use(i18n)
  .use(VueClickAway)
  .use(infiniteScroll)
  .use(v3ImgPreview, {})
  .use(lazyPlugin, { loading: defaultCover, error: defaultCover })
app.config.globalProperties.$notify = notify
registerSvgIcon(app)
registerObSkeleton(app)

// Failed background requests must not surface as uncaught page errors: surface
// one throttled toast instead so rate limits and outages stay explainable.
let lastFailureNotice = 0
window.addEventListener('unhandledrejection', (event) => {
  const reason: any = event.reason
  if (!reason || (!reason.isAxiosError && !reason.response)) return
  event.preventDefault()
  const now = Date.now()
  if (now - lastFailureNotice < 4000) return
  lastFailureNotice = now
  notify.warning(i18n.global.t('settings.request-failed'))
})

app.mount('#app')
console.log('%c 网站作者:小老师爱了爱了', 'color:#bada55')
console.log('%c qq:1909925152', 'color:#bada55')
api.report().catch(() => {})
