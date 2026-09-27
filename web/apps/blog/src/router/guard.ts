import router from '@/router'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'

let keyboardNavigation = false

document.addEventListener('keydown', () => {
  keyboardNavigation = true
})

document.addEventListener('pointerdown', () => {
  keyboardNavigation = false
  document.getElementById('main-content')?.classList.remove('keyboard-route-focus')
})

router.beforeEach((to) => {
  const appStore = useAppStore()
  const userStore = useUserStore()
  appStore.startLoading()
  const token = sessionStorage.getItem('token')
  if (!token) userStore.clearSession()
  if (to.meta.requiresAuth && !token) {
    return { path: '/', query: { login: '1', redirect: to.fullPath } }
  }
})

router.afterEach(() => {
  const appStore = useAppStore()
  appStore.endLoading()
  const mainContent = document.getElementById('main-content')
  if (!mainContent) return
  mainContent.classList.toggle('keyboard-route-focus', keyboardNavigation)
  mainContent.focus()
})
