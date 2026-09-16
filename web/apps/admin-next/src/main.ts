import { createApp, watch, type App as VueApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue, { Message } from '@arco-design/web-vue'
import { addI18nMessages, useLocale } from '@arco-design/web-vue/es/locale'
import arcoEnUS from '@arco-design/web-vue/es/locale/lang/en-us'
import '@arco-design/web-vue/dist/arco.css'

import App from '@/App.vue'
import router, { resetMenuRoutes } from '@/router'
import '@/styles.css'
import { AUTH_EXPIRED_EVENT } from '@/api/http'
import { locale, t } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { shouldNotifySessionExpired } from '@/utils/session-notice'

// Arco 自带组件（日期选择、分页、表格空态、确认弹窗…）的文案要跟随本站语言。
// zh-CN 是它的内置默认值，英文需要显式注册。
addI18nMessages({ 'en-US': arcoEnUS })

// i18n 模块内部已经在初始化时写过一次 <html lang>（见 src/i18n/index.ts 的 watch），
// 这里只需要把语言同步给 Arco。
watch(locale, (value) => useLocale(value), { immediate: true })

const app: VueApp = createApp(App)

// 模板里直接写 t('域.键')，不必在每个 SFC 里 import。
app.config.globalProperties.t = t

app.use(createPinia())
app.use(router)

// Register the full Arco component set: the admin console uses almost every
// primitive (table, form, grid, modal, pagination, descriptions, …) and the
// per-component list was drifting out of sync with the templates. The library
// is already a single vendor chunk and the shell loads it on first paint, so a
// complete registration costs nothing extra while removing a whole class of
// "component not resolved" runtime warnings.
app.use(ArcoVue)

app.mount('#app')

window.addEventListener(AUTH_EXPIRED_EVENT, () => {
  const auth = useAuthStore()
  const wasAuthenticated = auth.isAuthenticated
  auth.clear()
  useMenuStore().reset()
  resetMenuRoutes()

  // 并发请求会同时触发 401：同一轮失效只提示一次。
  if (wasAuthenticated && shouldNotifySessionExpired()) {
    Message.warning(t('shell.sessionExpired'))
  }

  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})
