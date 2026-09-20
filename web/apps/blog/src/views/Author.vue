<template>
  <div class="author-page">
    <p v-if="loadingAuthor" class="author-state">正在连接作者频道…</p>
    <p v-else-if="error" class="author-state is-error">{{ error }}</p>
    <template v-else-if="author">
      <section class="author-hero">
        <img :src="author.avatar || defaultAvatar" :alt="author.nickname || author.handle" />
        <div class="author-hero__copy">
          <p>PUBLIC AUTHOR</p>
          <h1>{{ author.nickname || author.handle }}</h1>
          <span class="author-hero__handle">@{{ author.handle }}</span>
          <p class="author-hero__intro">{{ author.intro || '这位作者还没有写下简介。' }}</p>
          <a v-if="author.website" :href="author.website" target="_blank" rel="noopener noreferrer">{{ author.website }}</a>
        </div>
        <dl class="author-hero__stats">
          <div><dt>{{ author.articleCount }}</dt><dd>公开文章</dd></div>
          <div><dt>{{ author.talkCount }}</dt><dd>公开随想</dd></div>
          <div><dt>{{ author.seriesCount }}</dt><dd>公开系列</dd></div>
        </dl>
      </section>

      <section class="author-content">
        <nav class="author-tabs">
          <button type="button" :class="{ active: tab === 'articles' }" @click="switchTab('articles')">文章</button>
          <button type="button" :class="{ active: tab === 'talks' }" @click="switchTab('talks')">随想</button>
          <button type="button" :class="{ active: tab === 'series' }" @click="switchTab('series')">系列</button>
        </nav>

        <p v-if="loading" class="author-state">加载中…</p>
        <div v-else-if="tab === 'articles'" class="author-list">
          <router-link v-for="item in records" :key="item.id" :to="`/articles/${item.id}`" class="author-list__item">
            <div>
              <span>{{ item.categoryName || '未分类' }} · {{ formatDate(item.createTime) }}</span>
              <h2>{{ item.articleTitle }}</h2>
              <p>{{ excerpt(item.articleContent) }}</p>
            </div>
            <img v-if="item.articleCover" :src="item.articleCover" :alt="item.articleTitle" loading="lazy" />
          </router-link>
        </div>
        <div v-else-if="tab === 'talks'" class="author-talks">
          <article v-for="item in records" :key="item.id">
            <router-link :to="`/talks/${item.id}`">{{ item.content }}</router-link>
            <img v-for="image in item.imgs || []" :key="image" :src="image" alt="" loading="lazy" />
            <small>{{ formatDate(item.createTime) }} · {{ item.commentCount || 0 }} 条回应</small>
          </article>
        </div>
        <div v-else class="author-series">
          <router-link v-for="item in records" :key="item.id" :to="`/series/${item.id}`" class="author-series__card">
            <img v-if="item.cover" :src="item.cover" :alt="item.seriesName" />
            <span>
              <small>{{ item.articleCount }} 篇文章</small>
              <h2>{{ item.seriesName }}</h2>
              <p>{{ item.seriesDesc || '暂无系列说明' }}</p>
            </span>
          </router-link>
        </div>
        <p v-if="!loading && !records.length" class="author-state">这里还没有公开内容。</p>
        <button v-if="records.length < total" type="button" class="author-more" :disabled="loading" @click="loadMore">加载更多</button>
      </section>
    </template>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/api'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="120" height="120"%3E%3Crect width="120" height="120" rx="60" fill="%23172554"/%3E%3Ccircle cx="60" cy="44" r="21" fill="%239bb8ff"/%3E%3Cpath d="M20 108c4-27 21-41 40-41s36 14 40 41" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Author',
  setup() {
    const route = useRoute()
    const author = ref<any>(null)
    const records = ref<any[]>([])
    const tab = ref<'articles' | 'talks' | 'series'>('articles')
    const loading = ref(false)
    const loadingAuthor = ref(true)
    const error = ref('')
    const page = ref(1)
    const total = ref(0)
    const pageSize = 12
    const handle = String(route.params.handle || '')

    const responseData = (response: any) => response?.data?.data || {}

    const loadAuthor = async () => {
      loadingAuthor.value = true
      error.value = ''
      try {
        author.value = responseData(await api.getAuthorByHandle(handle))
        if (!author.value?.id) throw new Error('author missing')
      } catch {
        error.value = '没有找到这个作者频道。'
      } finally {
        loadingAuthor.value = false
      }
    }

    const loadContent = async (reset = false) => {
      if (!author.value) return
      if (reset) {
        page.value = 1
        records.value = []
      }
      loading.value = true
      try {
        const params = { current: page.value, size: pageSize }
        const request = tab.value === 'articles'
          ? api.getAuthorArticles(handle, params)
          : tab.value === 'talks'
            ? api.getAuthorTalks(handle, params)
            : api.getAuthorSeries(handle, params)
        const data = responseData(await request)
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
      } catch {
        records.value = []
      } finally {
        loading.value = false
      }
    }

    const switchTab = (next: 'articles' | 'talks' | 'series') => {
      tab.value = next
      void loadContent(true)
    }
    const loadMore = () => {
      page.value += 1
      void loadContent(false)
    }
    const excerpt = (value: string, limit = 150) => {
      const text = String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text.length > limit ? text.slice(0, limit) + '…' : text
    }
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''

    onMounted(async () => {
      await loadAuthor()
      await loadContent(true)
    })

    return { author, records, tab, loading, loadingAuthor, error, total, defaultAvatar, switchTab, loadMore, excerpt, formatDate }
  }
})
</script>

<style lang="scss" scoped>
.author-page { max-width: 1120px; margin: 0 auto; padding: 28px 0 96px; }
.author-state { padding: 60px 0; color: var(--text-ob-dim); text-align: center; }
.author-state.is-error { color: #e2776c; }
.author-hero { display: grid; grid-template-columns: 132px minmax(0, 1fr) auto; gap: 28px; align-items: center; padding: clamp(28px, 5vw, 54px); border: 1px solid var(--border-hairline); border-radius: 26px; background: radial-gradient(circle at 88% 12%, rgba(103, 72, 188, .2), transparent 34%), linear-gradient(135deg, color-mix(in srgb, var(--background-primary-alt) 94%, #3159c7 6%), var(--background-primary)); }
.author-hero > img { width: 132px; height: 132px; border: 1px solid color-mix(in srgb, var(--color-ob) 45%, transparent); border-radius: 50%; object-fit: cover; box-shadow: 0 18px 55px rgba(7, 13, 38, .34); }
.author-hero__copy > p:first-child { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.author-hero h1 { margin: 0; font-size: clamp(2rem, 4vw, 3.6rem); letter-spacing: -.05em; }
.author-hero__handle { display: inline-block; margin-top: 6px; color: var(--text-ob-dim); font-size: 13px; }
.author-hero__intro { max-width: 620px; margin: 18px 0 8px; color: var(--text-ob-dim); line-height: 1.75; }
.author-hero__copy a { display: inline-flex; align-items: center; min-height: 24px; color: var(--color-ob); font-size: 12px; text-decoration: none; }
.author-hero__stats { display: grid; grid-template-columns: repeat(3, auto); gap: 20px; margin: 0; }
.author-hero__stats div { text-align: center; }
.author-hero__stats dt { font-size: 1.7rem; font-weight: 800; }
.author-hero__stats dd { margin: 4px 0 0; color: var(--text-ob-dim); font-size: 11px; }
.author-content { margin-top: 30px; }
.author-tabs { display: flex; gap: 8px; margin-bottom: 22px; }
.author-tabs button { padding: 8px 18px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.author-tabs button.active { border-color: var(--color-ob); background: color-mix(in srgb, var(--color-ob) 13%, transparent); color: var(--color-ob); }
.author-list { display: grid; gap: 14px; }
.author-list__item { display: grid; grid-template-columns: minmax(0, 1fr) 210px; gap: 24px; padding: 24px; border: 1px solid var(--border-hairline); border-radius: 18px; color: inherit; text-decoration: none; transition: border-color .2s ease, transform .2s ease; }
.author-list__item:hover { transform: translateY(-2px); border-color: color-mix(in srgb, var(--color-ob) 48%, transparent); }
.author-list__item span, .author-list__item p, .author-series small, .author-series p { color: var(--text-ob-dim); font-size: 12px; }
.author-list__item h2 { margin: 8px 0; font-size: 1.3rem; }
.author-list__item p { line-height: 1.7; }
.author-list__item img { width: 100%; height: 132px; border-radius: 12px; object-fit: cover; }
.author-talks { columns: 2; column-gap: 16px; }
.author-talks article { break-inside: avoid; margin-bottom: 16px; padding: 22px; border: 1px solid var(--border-hairline); border-radius: 16px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.author-talks a { color: inherit; line-height: 1.8; text-decoration: none; white-space: pre-wrap; }
.author-talks img { display: block; width: 100%; margin-top: 12px; border-radius: 10px; }
.author-talks small { display: block; margin-top: 14px; color: var(--text-ob-dim); }
.author-series { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.author-series__card { display: grid; grid-template-columns: 110px 1fr; gap: 18px; min-height: 150px; padding: 18px; border: 1px solid var(--border-hairline); border-radius: 18px; color: inherit; text-decoration: none; }
.author-series__card img { width: 110px; height: 100%; border-radius: 12px; object-fit: cover; }
.author-series__card h2 { margin: 7px 0; }
.author-series__card p { line-height: 1.6; }
.author-more { display: block; margin: 28px auto 0; padding: 9px 22px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 800px) { .author-hero { grid-template-columns: 92px 1fr; } .author-hero > img { width: 92px; height: 92px; } .author-hero__stats { grid-column: 1 / -1; justify-content: start; } .author-list__item { grid-template-columns: 1fr; } .author-list__item img { height: 180px; } .author-series { grid-template-columns: 1fr; } }
@media (max-width: 560px) { .author-talks { columns: 1; } }
</style>