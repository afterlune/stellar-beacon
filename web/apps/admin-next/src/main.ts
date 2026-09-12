import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'

import App from '@/App.vue'
import router, { resetMenuRoutes } from '@/router'
import '@/styles.css'
import { AUTH_EXPIRED_EVENT } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'

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
  auth.clear()
  useMenuStore().reset()
  resetMenuRoutes()
  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})
