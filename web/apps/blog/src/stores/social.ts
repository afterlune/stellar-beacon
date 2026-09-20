import { defineStore } from 'pinia'
import api from '@/api/api'

export const useSocialStore = defineStore('socialStore', {
  state: () => ({
    unreadCount: 0,
    refreshingUnread: false
  }),
  actions: {
    async refreshUnread() {
      if (this.refreshingUnread) return
      this.refreshingUnread = true
      try {
        const response = await api.getFollowNotificationUnreadCount()
        if (response?.data?.flag) {
          this.unreadCount = Number(response.data.data?.count || 0)
        }
      } catch {
        // The header badge is best-effort; page-level requests surface errors.
      } finally {
        this.refreshingUnread = false
      }
    },
    setUnreadCount(value: number) {
      this.unreadCount = Math.max(0, Number(value || 0))
    },
    async markRead() {
      const response = await api.markFollowNotificationsRead()
      if (!response?.data?.flag) {
        throw new Error(response?.data?.message || '标记已读失败')
      }
      this.unreadCount = 0
    },
    reset() {
      this.unreadCount = 0
      this.refreshingUnread = false
    }
  }
})