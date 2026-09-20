<template>
  <div class="following-page">
    <header class="following-head">
      <div>
        <p>FOLLOWING FEED</p>
        <h1>关注动态</h1>
        <span>只展示你关注作者在关注之后发布的新内容。</span>
      </div>
      <router-link to="/authors">发现作者 →</router-link>
    </header>
    <nav class="following-tabs" aria-label="关注管理">
      <button type="button" :class="{ active: tab === 'feed' }" @click="switchTab('feed')">动态</button>
      <button type="button" :class="{ active: tab === 'following' }" @click="switchTab('following')">关注中</button>
      <button type="button" :class="{ active: tab === 'followers' }" @click="switchTab('followers')">粉丝</button>
    </nav>
    <div v-if="tab === 'feed'" class="following-filters">
      <button v-for="item in feedTypes" :key="item.value" type="button" :class="{ active: feedType === item.value }" @click="changeFeedType(item.value)">{{ item.label }}</button>
    </div>
    <p v-if="loading && !records.length" class="following-state">正在加载…</p>
    <p v-else-if="error" class="following-state is-error">{{ error }}</p>
    <template v-else-if="tab === 'feed'">
      <div v-if="records.length" class="following-feed">
        <article v-for="item in records" :key="item.eventId">
          <router-link :to="contentPath(item)" class="following-feed__main">
            <span>{{ item.contentType === 'article' ? '文章' : '随想' }} · {{ formatDateTime(item.publishedAt) }}</span>
            <h2>{{ item.title || excerpt(item.excerpt, 60) }}</h2>
            <p>{{ excerpt(item.excerpt) }}</p>
          </router-link>
          <router-link :to="`/u/${item.author.handle}`" class="following-feed__author">
            <img :src="item.author.avatar || defaultAvatar" :alt="item.author.nickname || item.author.handle" />
            <span>{{ item.author.nickname || item.author.handle }}</span>
          </router-link>
        </article>
      </div>
      <p v-else class="following-state">关注后的新动态会出现在这里。</p>
    </template>
    <template v-else>
      <div v-if="records.length" class="following-users">
        <article v-for="item in records" :key="item.id">
          <router-link :to="`/u/${item.handle}`"><img :src="item.avatar || defaultAvatar" :alt="item.nickname || item.handle" /></router-link>
          <div><router-link :to="`/u/${item.handle}`"><strong>{{ item.nickname || item.handle }}</strong></router-link><span>@{{ item.handle }}</span><p>{{ item.intro || '暂无简介' }}</p></div>
          <FollowButton v-if="tab === 'following'" :author-id="item.id" :following="true" compact @changed="(value: boolean) => userChanged(item, value)" />
        </article>
      </div>
      <p v-else class="following-state">{{ tab === 'following' ? '还没有关注任何作者。' : '还没有粉丝。' }}</p>
    </template>
    <button v-if="records.length < total" type="button" class="following-more" :disabled="loading" @click="loadMore">{{ loading ? '加载中…' : '加载更多' }}</button>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'
import FollowButton from '@/components/FollowButton.vue'

type Tab = 'feed' | 'following' | 'followers'
type FeedType = 'all' | 'article' | 'talk'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Following',
  components: { FollowButton },
  setup() {
    const tab = ref<Tab>('feed')
    const feedType = ref<FeedType>('all')
    const records = ref<any[]>([])
    const loading = ref(false)
    const error = ref('')
    const page = ref(1)
    const total = ref(0)
    const pageSize = 12
    const feedTypes = [
      { value: 'all' as const, label: '全部' },
      { value: 'article' as const, label: '文章' },
      { value: 'talk' as const, label: '随想' }
    ]
    const load = async (reset = false) => {
      if (reset) {
        page.value = 1
        records.value = []
      }
      loading.value = true
      error.value = ''
      try {
        const params = { current: page.value, size: pageSize }
        const response = tab.value === 'feed'
          ? await api.getFollowingFeed({ ...params, type: feedType.value })
          : tab.value === 'following'
            ? await api.getMyFollowing(params)
            : await api.getMyFollowers(params)
        const data = response?.data?.data || {}
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
      } catch {
        error.value = '关注数据加载失败'
      } finally {
        loading.value = false
      }
    }
    const switchTab = (value: Tab) => {
      if (tab.value === value) return
      tab.value = value
      void load(true)
    }
    const changeFeedType = (value: FeedType) => {
      if (feedType.value === value) return
      feedType.value = value
      void load(true)
    }
    const loadMore = () => {
      page.value += 1
      void load(false)
    }
    const userChanged = (item: any, following: boolean) => {
      if (!following) records.value = records.value.filter((row) => row.id !== item.id)
      total.value = Math.max(0, total.value - 1)
    }
    const contentPath = (item: any) => item.contentType === 'article' ? `/articles/${item.contentId}` : `/talks/${item.contentId}`
    const excerpt = (value: string, limit = 180) => {
      const text = String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text.length > limit ? text.slice(0, limit) + '…' : text
    }
    const formatDateTime = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : ''
    onMounted(() => { void load(true) })
    return { tab, feedType, feedTypes, records, loading, error, total, defaultAvatar, switchTab, changeFeedType, loadMore, userChanged, contentPath, excerpt, formatDateTime }
  }
})
</script>

<style scoped>
.following-page { max-width: 980px; margin: 0 auto; padding: 32px 0 96px; }
.following-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: clamp(26px, 5vw, 46px); border: 1px solid var(--border-hairline); border-radius: 22px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .2), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.following-head p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.following-head h1 { margin: 0 0 10px; font-size: clamp(2rem, 4vw, 3.4rem); }
.following-head span { color: var(--text-ob-dim); }
.following-head a { color: var(--color-ob); text-decoration: none; }
.following-tabs, .following-filters { display: flex; gap: 8px; margin: 18px 0; }
.following-tabs button, .following-filters button { padding: 8px 16px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.following-tabs button.active, .following-filters button.active { border-color: var(--color-ob); color: var(--color-ob); }
.following-feed { display: grid; gap: 12px; }
.following-feed article { display: grid; grid-template-columns: minmax(0, 1fr) 180px; gap: 18px; align-items: center; padding: 20px; border: 1px solid var(--border-hairline); border-radius: 16px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.following-feed__main { color: inherit; text-decoration: none; }
.following-feed__main > span { color: var(--color-ob); font-size: 10px; }
.following-feed__main h2 { margin: 7px 0; }
.following-feed__main p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.65; }
.following-feed__author { display: flex; align-items: center; gap: 10px; color: inherit; font-size: 12px; text-decoration: none; }
.following-feed__author img { width: 38px; height: 38px; border-radius: 50%; object-fit: cover; }
.following-users { display: grid; gap: 10px; }
.following-users article { display: grid; grid-template-columns: 58px minmax(0, 1fr) auto; gap: 14px; align-items: center; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 15px; }
.following-users img { width: 58px; height: 58px; border-radius: 50%; object-fit: cover; }
.following-users strong { color: inherit; text-decoration: none; }
.following-users span, .following-users p { display: block; margin: 3px 0 0; color: var(--text-ob-dim); font-size: 11px; }
.following-state { padding: 58px 0; color: var(--text-ob-dim); text-align: center; }
.following-state.is-error { color: #df8177; }
.following-more { display: block; margin: 26px auto 0; padding: 9px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 700px) { .following-head { align-items: flex-start; flex-direction: column; } .following-feed article { grid-template-columns: 1fr; } .following-users article { grid-template-columns: 50px minmax(0, 1fr); } .following-users img { width: 50px; height: 50px; } .following-users > article > .follow-button { grid-column: 2; justify-self: start; } }
</style>