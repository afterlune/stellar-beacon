import { defineStore } from 'pinia'
import { clearAuthSession, writeAuthSession, type AuthUserInfo } from '@/utils/authSession'

export const useUserStore = defineStore('userStore', {
  state: () => {
    return {
      userVisible: false,
      userInfo: '' as any,
      token: '' as any,
      accessArticles: [] as any,
      tab: 0 as any,
      page: 1 as any
    }
  },
  actions: {
    setAuthSession(session: { userInfo: unknown; token: string }, persist = true) {
      const userInfo = session.userInfo && typeof session.userInfo === 'object'
        ? Object.fromEntries(Object.entries(session.userInfo as Record<string, unknown>).filter(([key]) => key !== 'token'))
        : {}
      if (this.token !== session.token) this.token = session.token
      if (JSON.stringify(this.userInfo) !== JSON.stringify(userInfo)) this.userInfo = userInfo
      if (typeof window !== 'undefined') {
        try { window.sessionStorage.setItem('token', session.token) } catch { /* storage may be disabled */ }
      }
      if (persist) writeAuthSession(userInfo as AuthUserInfo, session.token)
    },
    updateUserInfo(patch: AuthUserInfo) {
      if (!this.userInfo || typeof this.userInfo !== 'object') return
      this.userInfo = { ...this.userInfo, ...patch }
      if (this.token) writeAuthSession(this.userInfo, this.token)
    },
    clearSession() {
      clearAuthSession()
      this.userVisible = false
      this.userInfo = ''
      this.token = ''
      this.accessArticles = []
      this.tab = 0
      this.page = 1
    }
  },
  persist: {
    storage: window.sessionStorage
  }
})
