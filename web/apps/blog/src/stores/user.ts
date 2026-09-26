import { defineStore } from 'pinia'

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
    clearSession() {
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
