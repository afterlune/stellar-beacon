<template>
  <div class="collection-detail-page">
    <p v-if="loading" class="collection-state">正在打开这本书单…</p>
    <p v-else-if="error" class="collection-state is-error">{{ error }}</p>
    <template v-else-if="detail">
      <header class="collection-hero">
        <div>
          <p>PUBLIC READING LIST</p>
          <h1>{{ detail.collection.title }}</h1>
          <span>{{ detail.collection.description || '这位读者还没有写书单简介。' }}</span>
          <div class="collection-hero__actions">
            <router-link v-if="detail.collection.owner" :to="`/u/${detail.collection.owner.handle}`">
              <img :src="detail.collection.owner.avatar || defaultAvatar" :alt="detail.collection.owner.nickname" />
              {{ detail.collection.owner.nickname || detail.collection.owner.handle }}
            </router-link>
            <button type="button" @click="share">{{ copied ? '链接已复制' : '分享书单' }}</button>
          </div>
        </div>
        <aside><strong>{{ detail.items.length }}</strong><small>篇文章</small><em>{{ detail.collection.visibility === 'unlisted' ? '链接可见' : '公开书单' }}</em></aside>
      </header>
      <ol class="collection-items">
        <li v-for="(item, index) in detail.items" :key="item.articleId">
          <span class="collection-items__index">{{ String(Number(index) + 1).padStart(2, '0') }}</span>
          <router-link v-if="item.article" :to="`/articles/${item.article.id}`" class="collection-items__main">
            <img v-if="item.article.articleCover" :src="item.article.articleCover" :alt="item.article.articleTitle" loading="lazy" />
            <div>
              <small>{{ item.article.categoryName || '未分类' }} · {{ formatDate(item.article.createTime) }}</small>
              <h2>{{ item.article.articleTitle }}</h2>
              <p>{{ item.note || excerpt(item.article.articleContent) }}</p>
            </div>
          </router-link>
        </li>
      </ol>
    </template>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/api'

export default defineComponent({
  name: 'CollectionDetail',
  setup() {
    const route = useRoute()
    const detail = ref<any>(null)
    const loading = ref(true)
    const error = ref('')
    const copied = ref(false)
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="72" height="72"%3E%3Crect width="72" height="72" rx="36" fill="%23172554"/%3E%3Ccircle cx="36" cy="27" r="13" fill="%239bb8ff"/%3E%3Cpath d="M12 67c4-17 12-25 24-25s20 8 24 25" fill="%239bb8ff"/%3E%3C/svg%3E'
    const load = async () => {
      loading.value = true; error.value = ''
      try {
        const response = await api.getPublicCollection(String(route.params.slug || ''))
        detail.value = response?.data?.data || null
        if (!detail.value?.collection) throw new Error('missing collection')
      } catch { error.value = '没有找到这个公开书单。'; detail.value = null } finally { loading.value = false }
    }
    const share = async () => {
      const url = window.location.href
      try {
        if (navigator.share) await navigator.share({ title: detail.value?.collection?.title, url })
        else { await navigator.clipboard.writeText(url); copied.value = true; window.setTimeout(() => { copied.value = false }, 1800) }
      } catch { /* user cancelled */ }
    }
    const excerpt = (value: string) => String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 160)
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''
    watch(() => route.params.slug, () => void load())
    onMounted(() => void load())
    return { detail, loading, error, copied, defaultAvatar, share, excerpt, formatDate }
  }
})
</script>

<style scoped>
.collection-detail-page { max-width: 1020px; margin: 0 auto; padding: 28px 0 96px; }
.collection-hero { display: grid; grid-template-columns: minmax(0, 1fr) 180px; gap: 24px; padding: clamp(28px, 5vw, 52px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 90% 0, rgba(96, 120, 220, .2), transparent 36%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.collection-hero p { margin: 0 0 10px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.collection-hero h1 { margin: 0 0 12px; font-size: clamp(2rem, 5vw, 3.8rem); }
.collection-hero > div > span { color: var(--text-ob-dim); font-size: 13px; line-height: 1.8; }
.collection-hero__actions { display: flex; align-items: center; gap: 12px; margin-top: 24px; }
.collection-hero__actions a, .collection-hero__actions button { display: inline-flex; align-items: center; min-height: 36px; padding: 7px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; font-size: 12px; text-decoration: none; cursor: pointer; }
.collection-hero__actions img { width: 25px; height: 25px; margin-right: 7px; border-radius: 50%; object-fit: cover; }
.collection-hero aside { display: grid; place-content: center; border: 1px solid var(--border-hairline); border-radius: 18px; text-align: center; }
.collection-hero aside strong { font-size: 2.4rem; }
.collection-hero aside small, .collection-hero aside em { color: var(--text-ob-dim); font-size: 11px; font-style: normal; }
.collection-hero aside em { margin-top: 8px; color: var(--color-ob); }
.collection-items { display: grid; gap: 12px; margin: 22px 0 0; padding: 0; list-style: none; }
.collection-items li { display: grid; grid-template-columns: 42px minmax(0, 1fr); gap: 12px; align-items: start; padding: 15px; border: 1px solid var(--border-hairline); border-radius: 17px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.collection-items__index { padding-top: 7px; color: var(--color-ob); font-family: ui-monospace, monospace; font-size: 11px; }
.collection-items__main { display: grid; grid-template-columns: minmax(0, 1fr) 180px; gap: 18px; color: inherit; text-decoration: none; }
.collection-items__main img { grid-column: 2; grid-row: 1; width: 100%; aspect-ratio: 16 / 9; border-radius: 12px; object-fit: cover; }
.collection-items__main small { color: var(--text-ob-dim); font-size: 10px; }
.collection-items__main h2 { margin: 7px 0; font-size: 1.1rem; }
.collection-items__main p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.collection-state { padding: 70px 0; color: var(--text-ob-dim); text-align: center; }
.collection-state.is-error { color: #df8177; }
@media (max-width: 680px) { .collection-hero { grid-template-columns: 1fr; } .collection-hero aside { min-height: 120px; } .collection-items__main { grid-template-columns: 1fr; } .collection-items__main img { grid-column: 1; grid-row: auto; } }
</style>
