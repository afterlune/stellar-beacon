<template>
  <div class="notifications-page">
    <header class="notifications-head">
      <div><p>SIGNAL INBOX</p><h1>发布提醒</h1><span>你关注的作者发布新文章或随想时会出现在这里。</span></div>
      <router-link to="/following">关注动态 →</router-link>
    </header>
    <p v-if="loading && !records.length" class="notifications-state">正在加载提醒…</p>
    <p v-else-if="error" class="notifications-state is-error">{{ error }}</p>
    <div v-else-if="records.length" class="notifications-list">
      <router-link v-for="item in records" :key="item.eventId" :to="contentPath(item)" class="notification-item" :class="{ 'is-unread': !item.read }">
        <span class="notification-item__type">{{ item.contentType === 'article' ? '新文章' : '新随想' }}</span>
        <div>
          <h2>{{ item.title || excerpt(item.excerpt, 60) }}</h2>
          <p>{{ excerpt(item.excerpt) }}</p>
          <small>{{ item.author.nickname || item.author.handle }} · {{ formatDateTime(item.publishedAt) }}</small>
        </div>
        <img :src="item.author.avatar || defaultAvatar" :alt="item.author.nickname || item.author.handle" />
      </router-link>
    </div>
    <p v-else class="notifications-state">暂时没有发布提醒。</p>
    <button v-if="records.length < total" type="button" class="notifications-more" :disabled="loading" @click="loadMore">{{ loading ? '加载中…' : '加载更多' }}</button>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'
import { useSocialStore } from '@/stores/social'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Notifications',
  setup() {
    const socialStore = useSocialStore()
    const records = ref<any[]>([])
    const loading = ref(false)
    const error = ref('')
    const page = ref(1)
    const pageSize = 20
    const total = ref(0)
    const load = async (reset = false) => {
      if (reset) {
        page.value = 1
        records.value = []
      }
      loading.value = true
      error.value = ''
      try {
        const response = await api.getFollowNotifications({ current: page.value, size: pageSize })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '提醒加载失败')
        const data = response.data.data || {}
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
        socialStore.setUnreadCount(Number(data.unreadCount || 0))
        if (records.value.some((item) => !item.read)) {
          try {
            await socialStore.markRead()
            records.value = records.value.map((item) => ({ ...item, read: true }))
          } catch {
            // Keep the list visible when the read cursor update is unavailable.
          }
        }
      } catch (reason: any) {
        error.value = reason?.response?.data?.message || reason?.message || '提醒加载失败'
      } finally {
        loading.value = false
      }
    }
    const loadMore = () => {
      page.value += 1
      void load(false)
    }
    const contentPath = (item: any) => item.contentType === 'article' ? `/articles/${item.contentId}` : `/talks/${item.contentId}`
    const excerpt = (value: string, limit = 180) => {
      const text = String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text.length > limit ? text.slice(0, limit) + '…' : text
    }
    const formatDateTime = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : ''
    onMounted(() => { void load(true) })
    return { records, loading, error, total, defaultAvatar, loadMore, contentPath, excerpt, formatDateTime }
  }
})
</script>

<style scoped>
.notifications-page { max-width: 900px; margin: 0 auto; padding: 32px 0 96px; }
.notifications-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: clamp(26px, 5vw, 46px); border: 1px solid var(--border-hairline); border-radius: 22px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .2), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.notifications-head p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.notifications-head h1 { margin: 0 0 10px; font-size: clamp(2rem, 4vw, 3.4rem); }
.notifications-head span { color: var(--text-ob-dim); }
.notifications-head a { color: var(--color-ob); text-decoration: none; }
.notifications-list { display: grid; gap: 10px; margin-top: 18px; }
.notification-item { display: grid; grid-template-columns: 68px minmax(0, 1fr) 44px; gap: 14px; align-items: center; padding: 17px 18px; border: 1px solid var(--border-hairline); border-radius: 15px; color: inherit; text-decoration: none; }
.notification-item.is-unread { border-color: color-mix(in srgb, var(--color-ob) 50%, transparent); background: color-mix(in srgb, var(--color-ob) 5%, transparent); }
.notification-item__type { color: var(--color-ob); font-size: 11px; font-weight: 700; }
.notification-item h2 { margin: 0 0 6px; }
.notification-item p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.6; }
.notification-item small { display: block; margin-top: 8px; color: var(--text-ob-dim); font-size: 10px; }
.notification-item img { width: 44px; height: 44px; border-radius: 50%; object-fit: cover; }
.notifications-state { padding: 58px 0; color: var(--text-ob-dim); text-align: center; }
.notifications-state.is-error { color: #df8177; }
.notifications-more { display: block; margin: 26px auto 0; padding: 9px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 640px) { .notifications-head { align-items: flex-start; flex-direction: column; } .notification-item { grid-template-columns: 56px minmax(0, 1fr); } .notification-item img { display: none; } }
</style>