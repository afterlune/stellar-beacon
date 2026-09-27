import { defineStore } from 'pinia'

let menuCloseTimer: ReturnType<typeof setTimeout> | undefined
let menuOpenTimer: ReturnType<typeof setTimeout> | undefined

export const useNavigatorStore = defineStore('navigatorStore', {
  state: () => ({
    openMenu: false,
    menuClosing: false,
    scrollPosition: 0,
    openNavigator: false
  }),
  getters: {},
  actions: {
    toggleMobileMenu() {
      if (this.openMenu) {
        this.closeMobileMenu()
      } else {
        this.openMobileMenu()
      }
    },
    openMobileMenu() {
      const bodyEl = document.querySelector('body')
      const appEl = document.getElementById('app')
      const appWrapperEl = document.getElementById('App-Wrapper')
      const appMobileEl = document.getElementById('App-Mobile-Profile')
      if (!appEl || !appWrapperEl || !appMobileEl || !bodyEl) return

      if (menuCloseTimer) {
        clearTimeout(menuCloseTimer)
        menuCloseTimer = undefined
      }
      if (menuOpenTimer) clearTimeout(menuOpenTimer)
      this.menuClosing = false
      this.scrollPosition = window.pageYOffset
      bodyEl.style.overflow = 'hidden'
      bodyEl.style.position = 'fixed'
      bodyEl.style.top = `-${this.scrollPosition}px`
      bodyEl.style.width = '100%'

      appEl.style.overflow = 'hidden'
      appEl.style.maxHeight = '100vh'
      appWrapperEl.style.borderRadius = '16px'
      appWrapperEl.style.overflow = 'hidden'
      appWrapperEl.style.maxHeight = '100vh'
      appWrapperEl.style.minHeight = '100vh'
      appWrapperEl.style.transform = 'translate3d(302px, 0px, 0px) scale3d(0.86, 0.86, 1)'
      menuOpenTimer = setTimeout(() => {
        appMobileEl.style.opacity = '1'
        appMobileEl.style.transform = 'translateY(0)'
        menuOpenTimer = undefined
      }, 200)
      this.openMenu = true
    },
    closeMobileMenu() {
      if (!this.openMenu || this.menuClosing) return
      const bodyEl = document.querySelector('body')
      const appEl = document.getElementById('app')
      const appWrapperEl = document.getElementById('App-Wrapper')
      const appMobileEl = document.getElementById('App-Mobile-Profile')
      if (!appEl || !appWrapperEl || !appMobileEl || !bodyEl) return

      this.menuClosing = true
      if (menuOpenTimer) {
        clearTimeout(menuOpenTimer)
        menuOpenTimer = undefined
      }
      bodyEl.style.removeProperty('overflow')
      bodyEl.style.removeProperty('position')
      bodyEl.style.removeProperty('top')
      bodyEl.style.removeProperty('width')
      window.scrollTo(0, this.scrollPosition)

      appMobileEl.style.opacity = '0'
      appMobileEl.style.transform = 'translateY(-20%)'
      appWrapperEl.style.transform = 'translate3d(0px, 0px, 0px) scale3d(1, 1, 1)'
      appWrapperEl.style.borderRadius = '0'

      if (menuCloseTimer) clearTimeout(menuCloseTimer)
      menuCloseTimer = setTimeout(() => {
        appEl.style.overflow = 'auto'
        appEl.style.maxHeight = 'initial'
        appWrapperEl.style.overflow = 'auto'
        appWrapperEl.style.maxHeight = 'initial'
        appWrapperEl.style.minHeight = 'initial'
        appWrapperEl.style.transform = 'none'
        this.openMenu = false
        this.menuClosing = false
        menuCloseTimer = undefined
      }, 376)
    },
    toggleOpenNavigator() {
      this.openNavigator = !this.openNavigator
    },
    setOpenNavigator(status: boolean) {
      this.openNavigator = status
    }
  }
})
