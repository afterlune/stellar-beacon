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
  actions: {},
  persist: {
    storage: window.sessionStorage
  }
})
