<template>
  <div class="collections-page">
    <header class="collections-hero">
      <p>CURATED SIGNALS</p>
      <h1>公开书单</h1>
      <span>读者整理的公开文章路径。按最新整理，或按文章热度、点赞和评论互动发现下一条内容线索。</span>
      <div class="collections-tabs">
        <button type="button" :class="{ active: sort === 'latest' }" @click="switchSort('latest')">最新整理</button>
        <button type="button" :class="{ active: sort === 'hot' }" @click="switchSort('hot')">近期热门</button>
      </div>
    </header>
    <p v-if="loading && !records.length" class="collections-state">正在汇合公开书单…</p>
    <p v-else-if="error" class="collections-state is-error">{{ error }}</p>
    <div v-else-if="records.length" class="collections-grid">
      <router-link v-for="item in records" :key="item.slug" :to="`/collections/${item.slug}`" class="collection-card">
        <div class="collection-card__cover">
          <img v-if="item.cover" :src="item.cover" :alt="item.title" loading="lazy" />
          <span v-else>{{ String(item.title || 'LIST').slice(0, 1) }}</span>
        </div>
        <div class="collection-card__body">
          <small>{{ item.articleCount }} 篇文章 · {{ item.likeCount || 0 }} 赞 · {{ item.commentCount || 0 }} 评论 · 更新于 {{ formatDate(item.updatedAt) }}</small>
          <h2>{{ item.title }}</h2>
          <p>{{ item.description || '这位读者还没有写书单简介。' }}</p>
          <div>
            <img :src="item.owner?.avatar || defaultAvatar" :alt="item.owner?.nickname || item.owner?.handle || '作者'" />
            <span>{{ item.owner?.nickname || item.owner?.handle }}</span>
            <em v-if="sort === 'hot'">{{ item.hotScore || 0 }} 热度</em>
          </div>
        </div>
      </router-link>
    </div>
    <p v-else class="collections-state">还没有公开书单。</p>
    <button v-if="records.length < total" type="button" class="collections-more" :disabled="loading" @click="loadMore">加载更多</button>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'

export default defineComponent({
  name: 'Collections',
  setup() {
    const records = ref<any[]>([])
    const sort = ref<'latest' | 'hot'>('latest')
    const page = ref(1)
    const total = ref(0)
    const loading = ref(false)
    const error = ref('')
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="72" height="72"%3E%3Crect width="72" height="72" rx="36" fill="%23172554"/%3E%3Ccircle cx="36" cy="27" r="13" fill="%239bb8ff"/%3E%3Cpath d="M12 67c4-17 12-25 24-25s20 8 24 25" fill="%239bb8ff"/%3E%3C/svg%3E'
    const load = async (reset = true) => {
      if (reset) { page.value = 1; records.value = [] }
      loading.value = true; error.value = ''
      try {
        const response = await api.getPublicCollections({ sort: sort.value, current: page.value, size: 12 })
        const data = response?.data?.data || {}
        const next = Array.isArray(data.items) ? data.items : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.total || 0)
      } catch { error.value = '公开书单加载失败' } finally { loading.value = false }
    }
    const switchSort = (value: 'latest' | 'hot') => { if (sort.value === value) return; sort.value = value; void load(true) }
    const loadMore = () => { page.value += 1; void load(false) }
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''
    onMounted(() => void load(true))
    return { records, sort, loading, error, total, defaultAvatar, switchSort, loadMore, formatDate }
  }
})
</script>

<style scoped>
.collections-page { max-width: 1120px; margin: 0 auto; padding: 28px 0 96px; }
.collections-hero { padding: clamp(28px, 5vw, 52px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 88% 0, rgba(96, 120, 220, .22), transparent 36%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.collections-hero > p { margin: 0 0 10px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.collections-hero h1 { margin: 0 0 12px; font-size: clamp(2.2rem, 5vw, 4rem); }
.collections-hero > span { color: var(--text-ob-dim); font-size: 13px; }
.collections-tabs { display: flex; gap: 8px; margin-top: 24px; }
.collections-tabs button { padding: 8px 14px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.collections-tabs button.active { border-color: var(--color-ob); color: var(--color-ob); }
.collections-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; margin-top: 24px; }
.collection-card { display: flex; flex-direction: column; overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 94%, transparent); color: inherit; text-decoration: none; }
.collection-card__cover { aspect-ratio: 16 / 8; background: linear-gradient(135deg, #152250, #3c2c70); }
.collection-card__cover img { width: 100%; height: 100%; object-fit: cover; }
.collection-card__cover span { display: grid; place-items: center; height: 100%; font-size: 3.5rem; font-weight: 900; color: rgba(218, 228, 255, .85); }
.collection-card__body { flex: 1; padding: 17px; }
.collection-card__body small, .collection-card__body p { color: var(--text-ob-dim); font-size: 11px; }
.collection-card__body h2 { margin: 9px 0; font-size: 1.15rem; }
.collection-card__body p { min-height: 36px; line-height: 1.65; }
.collection-card__body > div { display: flex; align-items: center; gap: 8px; margin-top: 17px; font-size: 11px; }
.collection-card__body img { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
.collection-card__body em { margin-left: auto; color: var(--color-ob); font-style: normal; }
.collections-state { padding: 64px 0; color: var(--text-ob-dim); text-align: center; }
.collections-state.is-error { color: #df8177; }
.collections-more { display: block; margin: 26px auto 0; padding: 9px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 820px) { .collections-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 560px) { .collections-grid { grid-template-columns: 1fr; } }
</style>
