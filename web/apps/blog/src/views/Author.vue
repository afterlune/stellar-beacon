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
          <div class="author-hero__actions">
            <template v-if="isSelf">
              <router-link to="/studio/profile" class="author-hero__action">编辑公开资料</router-link>
              <router-link to="/studio/dashboard" class="author-hero__action">进入我的空间</router-link>
            </template>
            <FollowButton v-else :author-id="author.id" :following="Boolean(author.isFollowing)" @changed="followChanged" />
            <button type="button" class="author-hero__action author-hero__share" data-testid="author-share" @click="shareProfile">分享主页</button>
          </div>
        </div>
        <dl class="author-hero__stats">
          <div><dt>{{ author.followerCount || 0 }}</dt><dd>关注者</dd></div>
          <div><dt>{{ author.articleCount }}</dt><dd>公开文章</dd></div>
          <div><dt>{{ author.talkCount }}</dt><dd>公开随想</dd></div>
          <div><dt>{{ author.seriesCount }}</dt><dd>公开系列</dd></div>
          <div><dt>{{ author.collectionCount || 0 }}</dt><dd>公开书单</dd></div>
        </dl>
      </section>

      <section v-if="hasCuratedContent" class="author-curated" aria-labelledby="author-curated-title">
        <header>
          <div>
            <p>CURATED SIGNALS</p>
            <h2 id="author-curated-title">公开内容精选</h2>
          </div>
          <span>根据近期热度与更新时间自动整理</span>
        </header>
        <div class="author-curated__grid">
          <section v-if="highlights.length" class="author-curated__panel author-curated__panel--wide">
            <header><div><small>REPRESENTATIVE</small><h3>代表作</h3></div><button type="button" @click="switchTab('articles')">查看全部</button></header>
            <ol class="author-curated__articles">
              <li v-for="(item, index) in highlights" :key="item.id">
                <span>{{ String(index + 1).padStart(2, '0') }}</span>
                <router-link :to="`/articles/${item.id}`">
                  <strong>{{ item.articleTitle }}</strong>
                  <small>{{ item.categoryName || '未分类' }} · {{ formatDate(item.createTime) }}</small>
                </router-link>
              </li>
            </ol>
          </section>

          <section v-if="highlightSeries.length" class="author-curated__panel">
            <header><div><small>SERIES</small><h3>主题系列</h3></div><button type="button" @click="switchTab('series')">查看全部</button></header>
            <router-link v-for="item in highlightSeries" :key="item.id" :to="`/series/${item.id}`" class="author-curated__row">
              <span><strong>{{ item.seriesName }}</strong><small>{{ item.articleCount }} 篇文章</small></span>
              <em>{{ item.seriesDesc || '持续更新的主题合集' }}</em>
            </router-link>
          </section>

          <section v-if="highlightCollections.length" class="author-curated__panel">
            <header><div><small>READING LISTS</small><h3>公开书单</h3></div><button type="button" @click="switchTab('collections')">查看全部</button></header>
            <router-link v-for="item in highlightCollections" :key="item.slug" :to="`/collections/${item.slug}`" class="author-curated__row">
              <span><strong>{{ item.title }}</strong><small>{{ item.articleCount }} 篇文章</small></span>
              <em>{{ item.description || '按主题组织的阅读路径' }}</em>
            </router-link>
          </section>

          <section v-if="highlightTalks.length" class="author-curated__panel author-curated__panel--wide">
            <header><div><small>TALKS</small><h3>最近随想</h3></div><button type="button" @click="switchTab('talks')">查看全部</button></header>
            <div class="author-curated__talks">
              <router-link v-for="item in highlightTalks" :key="item.id" :to="`/talks/${item.id}`">
                <p>{{ excerpt(item.content, 90) }}</p>
                <small>{{ formatDate(item.createTime) }} · {{ item.commentCount || 0 }} 条回应</small>
              </router-link>
            </div>
          </section>
        </div>
      </section>

      <section class="author-content">
        <nav class="author-tabs">
          <button type="button" :class="{ active: tab === 'articles' }" @click="switchTab('articles')">文章</button>
          <button type="button" :class="{ active: tab === 'talks' }" @click="switchTab('talks')">随想</button>
          <button type="button" :class="{ active: tab === 'series' }" @click="switchTab('series')">系列</button>
          <button type="button" :class="{ active: tab === 'collections' }" @click="switchTab('collections')">书单</button>
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
        <div v-else-if="tab === 'series'" class="author-series">
          <router-link v-for="item in records" :key="item.id" :to="`/series/${item.id}`" class="author-series__card">
            <img v-if="item.cover" :src="item.cover" :alt="item.seriesName" />
            <span>
              <small>{{ item.articleCount }} 篇文章</small>
              <h2>{{ item.seriesName }}</h2>
              <p>{{ item.seriesDesc || '暂无系列说明' }}</p>
            </span>
          </router-link>
        </div>
        <div v-else class="author-series author-collections">
          <router-link v-for="item in records" :key="item.slug" :to="`/collections/${item.slug}`" class="author-series__card">
            <img v-if="item.cover" :src="item.cover" :alt="item.title" />
            <span v-else class="author-collection__cover">{{ String(item.title || 'LIST').slice(0, 1) }}</span>
            <span>
              <small>{{ item.articleCount }} 篇文章</small>
              <h2>{{ item.title }}</h2>
              <p>{{ item.description || '暂无书单简介' }}</p>
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
import { computed, defineComponent, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/api/api'
import FollowButton from '@/components/FollowButton.vue'
import { useSeoMeta } from '@/composables/useSeoMeta'
import { useUserStore } from '@/stores/user'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="120" height="120"%3E%3Crect width="120" height="120" rx="60" fill="%23172554"/%3E%3Ccircle cx="60" cy="44" r="21" fill="%239bb8ff"/%3E%3Cpath d="M20 108c4-27 21-41 40-41s36 14 40 41" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Author',
  components: { FollowButton },
  setup() {
    const route = useRoute()
    const userStore = useUserStore()
    const author = ref<any>(null)
    const records = ref<any[]>([])
    const tab = ref<'articles' | 'talks' | 'series' | 'collections'>('articles')
    const highlights = ref<any[]>([])
    const highlightSeries = ref<any[]>([])
    const highlightCollections = ref<any[]>([])
    const highlightTalks = ref<any[]>([])
    const loading = ref(false)
    const loadingAuthor = ref(true)
    const error = ref('')
    const page = ref(1)
    const total = ref(0)
    const pageSize = 12
    const handle = String(route.params.handle || '')

    const responseData = (response: any) => response?.data?.data || {}
    const isSelf = computed(() => {
      const currentID = Number(userStore.userInfo?.userInfoId || userStore.userInfo?.id || 0)
      return currentID > 0 && currentID === Number(author.value?.id || 0)
    })
    const followChanged = (following: boolean) => {
      if (!author.value) return
      author.value.isFollowing = following
      author.value.followerCount = Math.max(0, Number(author.value.followerCount || 0) + (following ? 1 : -1))
    }

    const hasCuratedContent = computed(() =>
      highlights.value.length > 0 ||
      highlightSeries.value.length > 0 ||
      highlightCollections.value.length > 0 ||
      highlightTalks.value.length > 0
    )

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

    const recordsFrom = (response: any) => {
      const data = responseData(response)
      const records = Array.isArray(data.records) ? data.records : Array.isArray(data.items) ? data.items : []
      return records.slice(0, 3)
    }
    const settledRecords = (result: PromiseSettledResult<any>) =>
      result.status === 'fulfilled' ? recordsFrom(result.value) : []

    // The overview is a best-effort glance: a missing module is hidden and must
    // never block the author's own tabs.
    const loadHighlights = async () => {
      const [hot, series, collections, talks] = await Promise.allSettled([
        api.getAuthorArticles(handle, { sort: 'hot', current: 1, size: 3 }),
        api.getAuthorSeries(handle, { current: 1, size: 3 }),
        api.getAuthorCollections(handle, { current: 1, size: 3 }),
        api.getAuthorTalks(handle, { current: 1, size: 3 })
      ])
      highlights.value = settledRecords(hot)
      highlightSeries.value = settledRecords(series)
      highlightCollections.value = settledRecords(collections)
      highlightTalks.value = settledRecords(talks)
      if (!highlights.value.length) {
        try {
          highlights.value = recordsFrom(await api.getAuthorArticles(handle, { sort: 'latest', current: 1, size: 3 }))
        } catch {
          highlights.value = []
        }
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
            : tab.value === 'series'
              ? api.getAuthorSeries(handle, params)
              : api.getAuthorCollections(handle, params)
        const data = responseData(await request)
        const next = Array.isArray(data.items) ? data.items : Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.total ?? data.count ?? 0)
      } catch {
        records.value = []
      } finally {
        loading.value = false
      }
    }

    const switchTab = (next: 'articles' | 'talks' | 'series' | 'collections') => {
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

    const profileURL = computed(() => {
      const currentHandle = String(author.value?.handle || handle).trim()
      return currentHandle ? new URL(`/u/${encodeURIComponent(currentHandle)}`, window.location.origin).toString() : ''
    })
    const authorSeoDescription = computed(() => {
      const current = author.value
      if (!current) return ''
      const intro = String(current.intro || '').trim()
      const fallback = `收录 ${Number(current.articleCount || 0)} 篇文章、${Number(current.talkCount || 0)} 条随想和 ${Number(current.seriesCount || 0)} 个系列。`
      const description = intro || fallback
      return description.length > 180 ? description.slice(0, 180) + '…' : description
    })
    useSeoMeta(computed(() => {
      const current = author.value
      if (!current || !profileURL.value) return null
      const name = String(current.nickname || current.handle || handle).trim()
      const handleName = String(current.handle || handle).trim()
      const image = String(current.avatar || '').trim()
      const person: Record<string, unknown> = {
        '@type': 'Person',
        name,
        alternateName: `@${handleName}`,
        description: authorSeoDescription.value,
        url: profileURL.value
      }
      if (image) person.image = image
      return {
        title: `${name} (@${handleName}) · Stellar Beacon`,
        description: authorSeoDescription.value,
        canonical: profileURL.value,
        image: image || undefined,
        type: 'profile',
        jsonLd: {
          '@context': 'https://schema.org',
          '@type': 'ProfilePage',
          url: profileURL.value,
          mainEntity: person
        }
      }
    }))

    const shareProfile = async () => {
      const current = author.value
      const url = profileURL.value
      if (!current || !url) return
      const name = String(current.nickname || current.handle || handle).trim()
      const payload = {
        title: `${name} · Stellar Beacon`,
        text: String(current.intro || '').trim() || `发现 ${name} 的公开主页`,
        url
      }
      if (typeof navigator.share === 'function') {
        try {
          await navigator.share(payload)
          return
        } catch (reason: any) {
          if (reason?.name === 'AbortError') return
        }
      }
      try {
        if (navigator.clipboard?.writeText) {
          await navigator.clipboard.writeText(url)
        } else {
          const input = document.createElement('input')
          input.value = url
          document.body.appendChild(input)
          input.select()
          document.execCommand('copy')
          input.remove()
        }
        ElMessage.success('公开主页链接已复制')
      } catch {
        ElMessage.error('分享失败，请手动复制浏览器地址')
      }
    }
    onMounted(async () => {
      await loadAuthor()
      await Promise.allSettled([loadHighlights(), loadContent(true)])
    })

    return {
      author, records, highlights, highlightSeries, highlightCollections, highlightTalks,
      hasCuratedContent, tab, loading, loadingAuthor, error, total,
      defaultAvatar, switchTab, loadMore, excerpt, formatDate, isSelf, followChanged, shareProfile
    }
  }
})
</script>

<style lang="scss" scoped>
.author-page { max-width: 1120px; margin: 0 auto; padding: 28px 0 96px; }
.author-curated { margin-top: 26px; padding: clamp(22px, 4vw, 34px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 92% 8%, color-mix(in srgb, var(--color-ob) 10%, transparent), transparent 32%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.author-curated > header { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; margin-bottom: 20px; }
.author-curated > header p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.author-curated > header h2 { margin: 0; font-size: clamp(1.35rem, 3vw, 1.9rem); }
.author-curated > header span { color: var(--text-ob-dim); font-size: 11px; }
.author-curated__grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.author-curated__panel { padding: 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary) 76%, transparent); }
.author-curated__panel--wide { grid-column: 1 / -1; }
.author-curated__panel > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; margin-bottom: 12px; }
.author-curated__panel header small { display: block; margin-bottom: 4px; color: var(--color-ob); font-size: 9px; letter-spacing: .17em; }
.author-curated__panel h3 { margin: 0; font-size: 1.15rem; }
.author-curated__panel header button { min-height: 32px; padding: 0 4px; border: 0; background: transparent; color: var(--text-ob-dim); font-size: 11px; cursor: pointer; }
.author-curated__panel header button:hover { color: var(--color-ob); }
.author-curated__articles { margin: 0; padding: 0; list-style: none; }
.author-curated__articles li { display: grid; grid-template-columns: 42px minmax(0, 1fr); gap: 14px; align-items: center; padding: 13px 0; border-bottom: 1px solid var(--border-hairline); }
.author-curated__articles li:last-child { border-bottom: 0; padding-bottom: 0; }
.author-curated__articles > li > span { color: var(--color-ob); font-size: 11px; letter-spacing: .14em; }
.author-curated__articles a { display: grid; gap: 5px; min-width: 0; color: inherit; text-decoration: none; }
.author-curated__articles a:hover strong, .author-curated__row:hover strong { color: var(--color-ob); }
.author-curated__articles strong { overflow: hidden; font-size: 1rem; text-overflow: ellipsis; white-space: nowrap; }
.author-curated__articles small, .author-curated__row small, .author-curated__talks small { color: var(--text-ob-dim); font-size: 11px; }
.author-curated__row { display: grid; grid-template-columns: minmax(0, .9fr) minmax(120px, 1.1fr); gap: 18px; align-items: center; padding: 14px 0; border-bottom: 1px solid var(--border-hairline); color: inherit; text-decoration: none; }
.author-curated__row:last-child { border-bottom: 0; padding-bottom: 0; }
.author-curated__row > span { display: grid; gap: 5px; min-width: 0; }
.author-curated__row strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.author-curated__row em { overflow: hidden; color: var(--text-ob-dim); font-size: 11px; font-style: normal; text-overflow: ellipsis; white-space: nowrap; }
.author-curated__talks { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.author-curated__talks a { display: flex; min-height: 132px; flex-direction: column; justify-content: space-between; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 14px; color: inherit; text-decoration: none; background: color-mix(in srgb, var(--background-primary-alt) 70%, transparent); transition: border-color .2s ease, transform .2s ease; }
.author-curated__talks a:hover { transform: translateY(-2px); border-color: color-mix(in srgb, var(--color-ob) 45%, transparent); }
.author-curated__talks p { display: -webkit-box; overflow: hidden; margin: 0 0 16px; line-height: 1.65; -webkit-box-orient: vertical; -webkit-line-clamp: 4; }
.author-state { padding: 60px 0; color: var(--text-ob-dim); text-align: center; }
@media (max-width: 760px) { .author-curated__grid, .author-curated__talks { grid-template-columns: 1fr; } .author-curated__panel--wide { grid-column: auto; } }
.author-state.is-error { color: #e2776c; }
.author-hero { display: grid; grid-template-columns: 132px minmax(0, 1fr) auto; gap: 28px; align-items: center; padding: clamp(28px, 5vw, 54px); border: 1px solid var(--border-hairline); border-radius: 26px; background: radial-gradient(circle at 88% 12%, rgba(103, 72, 188, .2), transparent 34%), linear-gradient(135deg, color-mix(in srgb, var(--background-primary-alt) 94%, #3159c7 6%), var(--background-primary)); }
.author-hero > img { width: 132px; height: 132px; border: 1px solid color-mix(in srgb, var(--color-ob) 45%, transparent); border-radius: 50%; object-fit: cover; box-shadow: 0 18px 55px rgba(7, 13, 38, .34); }
.author-hero__copy > p:first-child { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.author-hero h1 { margin: 0; font-size: clamp(2rem, 4vw, 3.6rem); letter-spacing: -.05em; }
.author-hero__handle { display: inline-block; margin-top: 6px; color: var(--text-ob-dim); font-size: 13px; }
.author-hero__intro { max-width: 620px; margin: 18px 0 8px; color: var(--text-ob-dim); line-height: 1.75; }
.author-hero__actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 16px; }
.author-hero__copy a { display: inline-flex; align-items: center; min-height: 24px; color: var(--color-ob); font-size: 12px; text-decoration: none; }
.author-hero__action { min-height: 36px; padding: 0 16px; border: 1px solid color-mix(in srgb, var(--color-ob) 40%, transparent); border-radius: 999px; background: color-mix(in srgb, var(--color-ob) 16%, transparent); color: var(--color-ob); font: inherit; font-size: 12px; cursor: pointer; }
.author-hero__action + .author-hero__action, .author-hero__share { background: transparent; }
.author-hero__stats { display: grid; grid-template-columns: repeat(5, auto); gap: 20px; margin: 0; }
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
.author-collection__cover { display: grid; width: 110px; height: 100%; min-height: 110px; place-items: center; border: 1px solid var(--border-hairline); border-radius: 12px; background: color-mix(in srgb, var(--color-ob) 12%, transparent); color: var(--color-ob); font-size: 2rem; font-weight: 800; }
.author-series__card h2 { margin: 7px 0; }
.author-series__card p { line-height: 1.6; }
.author-more { display: block; margin: 28px auto 0; padding: 9px 22px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 800px) { .author-hero { grid-template-columns: 92px 1fr; } .author-hero > img { width: 92px; height: 92px; } .author-hero__stats { grid-column: 1 / -1; justify-content: start; } .author-list__item { grid-template-columns: 1fr; } .author-list__item img { height: 180px; } .author-series { grid-template-columns: 1fr; } }
@media (max-width: 560px) { .author-talks { columns: 1; } }
</style>
