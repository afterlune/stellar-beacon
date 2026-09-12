import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue, { Message } from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'

import App from '@/App.vue'
import router, { resetMenuRoutes } from '@/router'
import '@/styles.css'
import { AUTH_EXPIRED_EVENT } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { shouldNotifySessionExpired } from '@/utils/session-notice'

const app = createApp(App)

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
    Message.warning('登录状态已过期，请重新登录')
  }

  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})
