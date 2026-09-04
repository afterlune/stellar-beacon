import { defineStore } from 'pinia'
import { i18n } from '@/locales/index'
import cookies from 'js-cookie'
import nProgress from 'nprogress'
import 'nprogress/nprogress.css'
import type { AgentFeatureFlags } from '@shared/api-contract'

export type { AgentFeatureFlags }

nProgress.configure({
  showSpinner: false,
  trickleSpeed: 100,
  parent: '#loading-bar-wrapper'
})

const hasProgressParent = () =>
  typeof document !== 'undefined' && document.querySelector('#loading-bar-wrapper') !== null

const setTheme = (theme: string) => {
  if (theme === 'theme-dark') {
    document.body.classList.remove('theme-light')
    document.body.classList.add('theme-dark')
  } else {
    document.body.classList.remove('theme-dark')
    document.body.classList.add('theme-light')
  }
}

export const useAppStore = defineStore('appStore', {
  state: () => {
    return {
      themeConfig: {
        theme: cookies.get('theme') ? String(cookies.get('theme')) : 'theme-dark',
        profile_shape: 'circle-avatar',
        feature: true,
        gradient: {
          color_1: '#24c6dc',
          color_2: '#5433ff',
          color_3: '#ff0099'
        },
        header_gradient_css: 'linear-gradient(130deg, #24c6dc, #5433ff 41.07%, #ff0099 76.05%)',
        background_gradient_style: {
          background: 'linear-gradient(130deg, #24c6dc, #5433ff 41.07%, #ff0099 76.05%)',
          '-webkit-background-clip': 'text',
          '-webkit-text-fill-color': 'transparent',
          '-webkit-box-decoration-break': 'clone',
          'box-decoration-break': 'clone'
        }
      },
      appLoading: false,
      websiteConfig: '' as any,
      viewCount: 0,
      articleCount: 0,
      talkCount: 0,
      categoryCount: 0,
      tagCount: 0,
      NPTimeout: -1,
      loadingTimeout: -1,
      aurora_bot_enable: true,
      agentFeaturesReady: false,
      agentFeatures: {
        publicChat: false,
        vitals: false,
        galaxy: false,
        dreams: false,
        capsules: false,
        radio: false,
        videos: false,
        ttsEnabled: false
      } as AgentFeatureFlags
    }
  },
  actions: {
    changeLocale(locale: string) {
      cookies.set('locale', locale, { expires: 7 })
      i18n.global.locale.value = locale
    },
    initializeTheme(mode: string) {
      setTheme(mode)
    },
    toggleTheme(isDark?: boolean) {
      this.themeConfig.theme =
        isDark === true || this.themeConfig.theme === 'theme-light' ? 'theme-dark' : 'theme-light'
      cookies.set('theme', this.themeConfig.theme, { expires: 7 })
      setTheme(this.themeConfig.theme)
    },
    startLoading() {
      if (this.appLoading === true) return
      if (this.NPTimeout !== -1) clearTimeout(this.NPTimeout)
      if (this.loadingTimeout !== -1) clearTimeout(this.loadingTimeout)
      // The router's first navigation can run before App.vue has mounted the
      // progress wrapper. Avoid asking NProgress to render into a null parent.
      if (hasProgressParent()) nProgress.start()
      this.appLoading = true
    },
    endLoading() {
      this.NPTimeout = <any>setTimeout(() => {
        if (hasProgressParent()) nProgress.done()
      }, 100)

      this.loadingTimeout = <any>setTimeout(() => {
        this.appLoading = false
      }, 300)
    },
    setAgentFeatures(flags: Partial<AgentFeatureFlags> | null | undefined) {
      this.agentFeaturesReady = true
      this.agentFeatures = {
        publicChat: flags?.publicChat === true,
        vitals: flags?.vitals === true,
        galaxy: flags?.galaxy === true,
        dreams: flags?.dreams === true,
        capsules: flags?.capsules === true,
        radio: flags?.radio === true,
        videos: flags?.videos === true,
        ttsEnabled: flags?.ttsEnabled === true
      }
    }
  }
})
