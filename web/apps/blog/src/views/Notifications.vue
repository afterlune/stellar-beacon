<template>
  <div class="notifications-page">
    <header class="notifications-head">
      <div><p>SIGNAL INBOX</p><h1>互动通知</h1><span>发布、评论回复、文章互动和书单更新都会汇流到这里。</span></div>
      <router-link to="/following">关注动态 →</router-link>
    </header>
    <nav class="notification-filters" aria-label="通知筛选">
      <button v-for="item in groups" :key="item.value" type="button" :class="{ active: activeGroup === item.value }" @click="switchGroup(item.value)">{{ item.label }}</button>
    </nav>
    <p v-if="loading && !records.length" class="notifications-state">正在加载通知…</p>
    <p v-else-if="error" class="notifications-state is-error">{{ error }}</p>
    <div v-else-if="records.length" class="notifications-list">
      <router-link v-for="item in records" :key="item.key" :to="notificationPath(item)" class="notification-item" :class="{ 'is-unread': !item.read }">
        <span class="notification-item__type">{{ typeLabel(item.type) }}</span>
        <div>
          <h2>{{ headline(item) }}</h2>
          <p>{{ excerpt(item.excerpt) }}</p>
          <small>{{ item.actor.nickname || item.actor.handle }} · {{ formatDateTime(item.createdAt) }}</small>
        </div>
        <img :src="item.actor.avatar || defaultAvatar" :alt="item.actor.nickname || item.actor.handle" />
      </router-link>
    </div>
    <p v-else class="notifications-state">当前筛选下没有通知。</p>
    <button v-if="records.length < total" type="button" class="notifications-more" :disabled="loading" @click="loadMore">{{ loading ? '加载中…' : '加载更多' }}</button>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'
import { useSocialStore } from '@/stores/social'
import type { NotificationCursor, NotificationGroup, NotificationItem, NotificationType } from '@stellar-beacon/api-contract'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Notifications',
  setup() {
    const socialStore = useSocialStore()
    const groups: Array<{ value: NotificationGroup; label: string }> = [
      { value: 'all', label: '全部' },
      { value: 'publish', label: '发布' },
      { value: 'comment', label: '评论与回复' },
      { value: 'reaction', label: '赞与收藏' },
      { value: 'topic', label: '话题订阅' },
      { value: 'collection', label: '书单更新' }
    ]
    const activeGroup = ref<NotificationGroup>('all')
    const records = ref<NotificationItem[]>([])
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
        const response = await api.getFollowNotifications({ group: activeGroup.value, current: page.value, size: pageSize })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '通知加载失败')
        const data = response.data.data || {}
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
        socialStore.setUnreadCount(Number(data.totalUnreadCount ?? data.unreadCount ?? 0))
        if (reset && Number(data.unreadCount || 0) > 0 && data.readCursor) {
          try {
            await socialStore.markRead(data.readCursor as NotificationCursor)
            records.value = records.value.map((item) => ({ ...item, read: true }))
          } catch {
            // Keep the list visible when the read cursor update is unavailable.
          }
        }
      } catch (reason: any) {
        error.value = reason?.response?.data?.message || reason?.message || '通知加载失败'
      } finally {
        loading.value = false
      }
    }
    const switchGroup = (group: NotificationGroup) => {
      if (activeGroup.value === group) return
      activeGroup.value = group
      void load(true)
    }
    const loadMore = () => {
      if (loading.value) return
      page.value += 1
      void load(false)
    }
    const typeLabel = (type: NotificationType) => ({
      publish: '新发布',
      comment: '新评论',
      reply: '新回复',
      like: '点赞',
      favorite: '收藏',
      collection_update: '书单更新'
    })[type] || '互动'
    const headline = (item: NotificationItem) => {
      if (item.group === 'topic') return `你订阅的话题有新文章：${item.title || item.excerpt || '查看文章'}`
      if (item.group === 'collection') return `书单「${item.title || '未命名'}」新增了：${item.excerpt || '查看更新'}`
      if (item.type === 'reply') return `回复了你的评论：${item.excerpt || '查看回复'}`
      if (item.type === 'comment') return `评论了你的内容：${item.title || item.excerpt || '查看评论'}`
      if (item.type === 'like') return `赞了你的内容：${item.title || item.excerpt || '查看文章'}`
      if (item.type === 'favorite') return `收藏了你的内容：${item.title || item.excerpt || '查看文章'}`
      return item.title || item.excerpt || '发布了新内容'
    }
    const notificationPath = (item: NotificationItem) => {
      if (item.contentType === 'collection') return `/collections/${item.slug || item.contentId}${item.articleId ? `?article=${item.articleId}` : ''}`
      const base = item.contentType === 'article' ? `/articles/${item.contentId}` : `/talks/${item.contentId}`
      if ((item.type === 'comment' || item.type === 'reply') && item.commentId) {
        return `${base}?comment=${item.commentId}#comment-${item.commentId}`
      }
      return base
    }
    const excerpt = (value: string, limit = 180) => {
      const text = String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text.length > limit ? text.slice(0, limit) + '…' : text
    }
    const formatDateTime = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : ''
    onMounted(() => { void load(true) })
    return { groups, activeGroup, records, loading, error, total, defaultAvatar, switchGroup, loadMore, typeLabel, headline, notificationPath, excerpt, formatDateTime }
  }
})
</script>

<style scoped>
.notifications-page { max-width: 900px; margin: 0 auto; padding: 32px 0 96px; }
.notifications-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: clamp(26px, 5vw, 46px); border: 1px solid var(--border-hairline); border-radius: 22px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .2), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.notifications-head p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.notifications-head h1 { margin: 0 0 10px; font-size: clamp(2rem, 4vw, 3.4rem); }
.notifications-head span { color: var(--text-ob-dim); }
.notifications-head a { display: inline-flex; align-items: center; min-height: 24px; padding: 2px 0; color: var(--color-ob); text-decoration: none; }
.notification-filters { display: flex; flex-wrap: wrap; gap: 8px; margin: 18px 0; }
.notification-filters button { padding: 8px 15px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.notification-filters button.active { border-color: var(--color-ob); color: var(--color-ob); }
.notifications-list { display: grid; gap: 10px; }
.notification-item { position: relative; display: grid; grid-template-columns: 76px minmax(0, 1fr) 44px; gap: 14px; align-items: center; padding: 17px 18px; border: 1px solid var(--border-hairline); border-radius: 15px; color: inherit; text-decoration: none; }
.notification-item.is-unread { border-color: color-mix(in srgb, var(--color-ob) 50%, transparent); background: color-mix(in srgb, var(--color-ob) 5%, transparent); }
.notification-item.is-unread::before { content: ''; position: absolute; left: 7px; top: 50%; width: 4px; height: 4px; border-radius: 50%; background: var(--color-ob); transform: translateY(-50%); }
.notification-item__type { color: var(--color-ob); font-size: 11px; font-weight: 700; }
.notification-item h2 { margin: 0 0 6px; font-size: 1rem; }
.notification-item p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.6; }
.notification-item small { display: block; margin-top: 8px; color: var(--text-ob-dim); font-size: 10px; }
.notification-item img { width: 44px; height: 44px; border-radius: 50%; object-fit: cover; }
.notifications-state { padding: 58px 0; color: var(--text-ob-dim); text-align: center; }
.notifications-state.is-error { color: #df8177; }
.notifications-more { display: block; margin: 26px auto 0; padding: 9px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 640px) { .notifications-head { align-items: flex-start; flex-direction: column; } .notification-item { grid-template-columns: 64px minmax(0, 1fr); } .notification-item img { display: none; } }
</style>
