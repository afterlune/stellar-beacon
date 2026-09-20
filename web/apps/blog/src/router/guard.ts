import router from '@/router'
import { useAppStore } from '@/stores/app'

router.beforeEach(async (to, from, next) => {
  const appStore = useAppStore()
  appStore.startLoading()
  if (to.meta.requiresAuth && !sessionStorage.getItem('token')) {
    next({ path: '/', query: { login: '1', redirect: to.fullPath } })
    return
  }
  next()
})

router.afterEach(() => {
  const appStore = useAppStore()
  appStore.endLoading()
  document.getElementById('App-Container')?.focus()
})
