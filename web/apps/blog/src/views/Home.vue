<template>
  <div class="plaza-page">
    <section class="plaza-hero">
      <div class="plaza-hero__copy">
        <p class="plaza-hero__eyebrow">STELLAR BEACON <span>/</span> PUBLIC PLAZA</p>
        <h1>每个账号，<br /><em>都是一座独立电台。</em></h1>
        <p class="plaza-hero__intro">
          公开文章、随想与系列在这里汇流。发现不同作者的长期思考，也建立自己的私有创作空间。
        </p>
        <div class="plaza-hero__actions">
          <button type="button" @click="scrollToFeed">浏览公共内容 <span aria-hidden="true">↓</span></button>
          <router-link to="/topics">话题广场</router-link>
          <router-link to="/authors">作者榜</router-link>
          <router-link to="/studio">进入我的创作台</router-link>
        </div>
      </div>
      <div class="plaza-hero__signal" aria-hidden="true">
        <span v-for="n in 9" :key="n" :style="{ '--i': n }" />
        <strong>{{ total || 'LIVE' }}</strong>
        <small>PUBLIC SIGNALS</small>
      </div>
    </section>

    <RecommendationPanel v-if="Boolean(userInfo)" class="plaza-recommendations" compact />

    <section v-if="authors.length" class="plaza-authors" aria-labelledby="plaza-authors-title">
      <header>
        <div>
          <p>AUTHORS ONLINE</p>
          <h2 id="plaza-authors-title">来自不同频道</h2>
        </div>
        <div class="plaza-authors__links">
          <span>{{ authors.length }} 位作者正在公开写作</span>
          <router-link class="plaza-authors__more" to="/authors">查看作者榜 →</router-link>
        </div>
      </header>
      <div class="plaza-authors__rail">
        <router-link v-for="author in authors" :key="author.id" :to="`/u/${author.handle}`" class="plaza-author">
          <img :src="author.avatar || defaultAvatar" :alt="author.nickname || author.handle" />
          <span>
            <strong>{{ author.nickname || author.handle }}</strong>
            <small>@{{ author.handle }}</small>
          </span>
          <em>{{ author.articleCount }} 文章 · {{ author.talkCount }} 随想</em>
        </router-link>
      </div>
    </section>

    <section id="public-feed" class="plaza-feed" data-testid="public-feed" aria-labelledby="plaza-feed-title">
      <header class="plaza-feed__head">
        <div>
          <p>COMMUNITY CHANNEL</p>
          <h2 id="plaza-feed-title">公共信息流</h2>
        </div>
        <div class="plaza-feed__controls">
          <div class="plaza-feed__filters">
            <button type="button" :class="{ active: activeType === 'article' }" @click="activeType = 'article'">文章</button>
            <button type="button" :class="{ active: activeType === 'talk' }" @click="activeType = 'talk'">随想</button>
          </div>
          <div v-if="activeType === 'article'" class="plaza-feed__filters" aria-label="信息流排序">
            <button type="button" :class="{ active: activeSort === 'latest' }" @click="activeSort = 'latest'">最新</button>
            <button type="button" :class="{ active: activeSort === 'hot' }" @click="activeSort = 'hot'">热门</button>
            <button type="button" :class="{ active: activeSort === 'featured' }" @click="activeSort = 'featured'">精选</button>
          </div>
        </div>
      </header>

      <p v-if="loading && !records.length" class="plaza-feed__state">正在接收公开信号…</p>
      <p v-else-if="error" class="plaza-feed__state is-error">{{ error }}</p>
      <div v-else-if="records.length" class="plaza-feed__grid">
        <template v-for="item in records" :key="`${activeType}-${item.id}`">
          <ArticleFeedCard v-if="activeType === 'article'" :data="item" />
          <article v-else class="plaza-card">
          <div class="plaza-card__body">
            <div class="plaza-card__author">
              <img :src="authorOf(item).avatar || defaultAvatar" :alt="authorOf(item).nickname" />
              <router-link :to="`/u/${authorOf(item).handle}`">
                {{ authorOf(item).nickname || authorOf(item).handle }}
              </router-link>
              <time>{{ formatDate(item.createTime) }}</time>
            </div>
            <router-link class="plaza-card__title" :to="`/talks/${item.id}`">{{ excerpt(item.content, 120) }}</router-link>
            <div class="plaza-card__meta">
              <span>{{ item.commentCount || 0 }} 条回应</span>
              <span v-if="item.isTop === 1">置顶</span>
            </div>
          </div>
          </article>
        </template>
      </div>
      <p v-else class="plaza-feed__state">公共空间还没有内容。登录后发布第一篇公开文章吧。</p>

      <button v-if="records.length < total" type="button" class="plaza-feed__more" :disabled="loading" @click="loadMore">
        {{ loading ? '加载中…' : '加载更多' }}
      </button>
    </section>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, ref, watch } from 'vue'
import api from '@/api/api'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import RecommendationPanel from '@/components/RecommendationPanel.vue'
import { ArticleFeedCard } from '@/components/ArticleCard'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="80" height="80"%3E%3Crect width="80" height="80" rx="40" fill="%23172554"/%3E%3Ccircle cx="40" cy="30" r="14" fill="%239bb8ff"/%3E%3Cpath d="M15 72c3-18 14-27 25-27s22 9 25 27" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Home',
  components: { RecommendationPanel, ArticleFeedCard },
  setup() {
    const appStore = useAppStore()
    const userStore = useUserStore()
    const userInfo = computed(() => userStore.userInfo)
    const records = ref<any[]>([])
    const authors = ref<any[]>([])
    const activeType = ref<'article' | 'talk'>('article')
    const activeSort = ref<'latest' | 'hot' | 'featured'>('latest')
    // Talks have no ranking signal yet, so they always read the recency feed.
    const requestSort = computed(() => activeType.value === 'talk' ? 'latest' : activeSort.value)
    const loading = ref(false)
    const error = ref('')
    const page = ref(1)
    const total = ref(0)
    const pageSize = 12

    const payload = (response: any) => response?.data?.data || {}

    const loadAuthors = async () => {
      try {
        const response = await api.getPlatformAuthors({ current: 1, size: 12 })
        authors.value = Array.isArray(payload(response).records) ? payload(response).records : []
      } catch {
        authors.value = []
      }
    }

    const loadFeed = async (reset = false) => {
      if (reset) {
        page.value = 1
        records.value = []
      }
      loading.value = true
      error.value = ''
      try {
        const response = await api.getPlatformFeed({
          type: activeType.value,
          sort: requestSort.value,
          current: page.value,
          size: pageSize
        })
        const data = payload(response)
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
      } catch {
        error.value = '公共内容暂时无法加载，请稍后重试。'
      } finally {
        loading.value = false
      }
    }

    const loadMore = () => {
      page.value += 1
      void loadFeed(false)
    }

    const authorOf = (item: any) => item.author || {
      handle: item.handle || '',
      nickname: item.nickName || item.nickname || item.handle || '匿名作者',
      avatar: item.avatar || ''
    }

    const excerpt = (value: string, limit = 92) => {
      const text = String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text.length > limit ? text.slice(0, limit) + '…' : text
    }

    const formatDate = (value: string) => {
      if (!value) return ''
      return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric' }).format(new Date(value))
    }

    const scrollToFeed = () => document.getElementById('public-feed')?.scrollIntoView({ behavior: 'smooth' })

    watch([activeType, activeSort], () => void loadFeed(true))
    onMounted(() => {
      void loadAuthors()
      void loadFeed(true)
    })

    return {
      appStore,
      records,
      authors,
      activeType,
      activeSort,
      loading,
      error,
      total,
      userInfo,
      defaultAvatar,
      loadMore,
      authorOf,
      excerpt,
      formatDate,
      scrollToFeed
    }
  }
})
</script>

<style lang="scss" scoped>
.plaza-page { max-width: 1180px; margin: 0 auto; padding: 22px 0 96px; }
.plaza-hero { position: relative; display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(260px, .55fr); gap: 32px; overflow: hidden; min-height: 390px; padding: clamp(34px, 6vw, 72px); border: 1px solid var(--border-hairline); border-radius: 28px; background: linear-gradient(135deg, color-mix(in srgb, var(--background-primary-alt) 92%, #4167d8 8%), color-mix(in srgb, var(--background-primary) 88%, #9d59e8 12%)); box-shadow: 0 28px 90px rgba(7, 13, 38, .28); }
.plaza-hero::after { content: ''; position: absolute; inset: 0; pointer-events: none; background: linear-gradient(110deg, transparent 40%, rgba(112, 143, 255, .12) 60%, transparent 78%); }
.plaza-hero__copy { position: relative; z-index: 1; align-self: center; }
.plaza-hero__eyebrow, .plaza-feed__head p, .plaza-authors header p { margin: 0 0 14px; color: var(--color-ob); font-size: 11px; letter-spacing: .2em; text-transform: uppercase; }
.plaza-hero__eyebrow span { opacity: .45; }
.plaza-hero h1 { margin: 0; font-size: clamp(2.5rem, 6vw, 5.3rem); line-height: .98; letter-spacing: -.055em; }
.plaza-hero h1 em { color: var(--color-ob); font-style: normal; }
.plaza-hero__intro { max-width: 640px; margin: 26px 0 0; color: var(--text-ob-dim); font-size: 16px; line-height: 1.85; }
.plaza-hero__actions { display: flex; align-items: center; gap: 12px; margin-top: 32px; flex-wrap: wrap; }
.plaza-hero__actions button, .plaza-hero__actions a { display: inline-flex; align-items: center; gap: 10px; padding: 11px 18px; border: 1px solid var(--border-hairline); border-radius: 999px; background: rgba(7, 14, 36, .32); color: inherit; font: inherit; font-size: 13px; text-decoration: none; cursor: pointer; }
.plaza-hero__actions button { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
.plaza-hero__signal { position: relative; z-index: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 230px; border: 1px solid color-mix(in srgb, var(--color-ob) 30%, transparent); border-radius: 50%; background: radial-gradient(circle, rgba(67, 102, 220, .18), transparent 66%); }
.plaza-hero__signal > span { position: absolute; width: calc(24px + var(--i) * 16px); height: calc(24px + var(--i) * 16px); border: 1px solid color-mix(in srgb, var(--color-ob) calc(34% - var(--i) * 2%), transparent); border-radius: 50%; animation: plazaPulse 3.2s ease-in-out infinite; animation-delay: calc(var(--i) * -160ms); }
.plaza-hero__signal strong { position: relative; font-size: 2.25rem; letter-spacing: -.05em; }
.plaza-hero__signal small { position: relative; margin-top: 4px; color: var(--text-ob-dim); font-size: 10px; letter-spacing: .18em; }
@keyframes plazaPulse { 50% { transform: scale(.92); opacity: .45; } }
.plaza-recommendations, .plaza-authors, .plaza-feed { margin-top: 38px; padding: clamp(22px, 4vw, 38px); border: 1px solid var(--border-hairline); border-radius: 24px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.plaza-authors header, .plaza-feed__head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-bottom: 24px; }
.plaza-authors h2, .plaza-feed h2 { margin: 0; font-size: clamp(1.5rem, 3vw, 2.15rem); }
.plaza-authors__links { display: flex; align-items: center; gap: 10px; color: var(--text-ob-dim); font-size: 12px; } .plaza-authors__more { display: inline-flex; align-items: center; min-height: 24px; padding: 0 4px; color: var(--color-ob); text-decoration: none; }
.plaza-authors__rail { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.plaza-author { display: grid; grid-template-columns: 48px 1fr; gap: 12px; align-items: center; padding: 14px; border: 1px solid var(--border-hairline); border-radius: 16px; color: inherit; text-decoration: none; transition: border-color .2s ease, transform .2s ease; }
.plaza-author:hover { transform: translateY(-2px); border-color: color-mix(in srgb, var(--color-ob) 55%, transparent); }
.plaza-author img, .plaza-card__author img { width: 48px; height: 48px; border-radius: 50%; object-fit: cover; background: var(--background-primary); }
.plaza-author span { min-width: 0; }
.plaza-author strong, .plaza-author small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.plaza-author small, .plaza-author em { color: var(--text-ob-dim); font-size: 11px; font-style: normal; }
.plaza-author em { grid-column: 2; margin-top: -8px; }
.plaza-feed__controls { display: flex; align-items: center; gap: 14px; flex-wrap: wrap; } .plaza-feed__filters { display: flex; align-items: center; gap: 8px; }
.plaza-feed__filters button { padding: 7px 14px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.plaza-feed__filters button.active { border-color: var(--color-ob); background: color-mix(in srgb, var(--color-ob) 14%, transparent); color: var(--color-ob); }
.plaza-feed__filters label { display: flex; align-items: center; gap: 6px; color: var(--text-ob-dim); font-size: 12px; }
.plaza-feed__grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px; }
.plaza-card { display: flex; min-width: 0; flex-direction: column; overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary) 94%, transparent); }
.plaza-card__cover { position: relative; display: block; aspect-ratio: 16 / 9; overflow: hidden; background: linear-gradient(135deg, #152250, #3c2c70); }
.plaza-card__cover img { width: 100%; height: 100%; object-fit: cover; transition: transform .45s ease; }
.plaza-card:hover .plaza-card__cover img { transform: scale(1.035); }
.plaza-card__cover em { position: absolute; top: 12px; left: 12px; padding: 4px 9px; border-radius: 999px; background: rgba(7, 14, 36, .72); color: #dbe6ff; font-size: 10px; font-style: normal; backdrop-filter: blur(8px); }
.plaza-card__placeholder { display: grid; place-items: center; height: 100%; color: rgba(218, 228, 255, .85); font-size: 3.5rem; font-weight: 800; }
.plaza-card__body { display: flex; flex: 1; flex-direction: column; padding: 16px; }
.plaza-card__author { display: flex; align-items: center; gap: 8px; margin-bottom: 15px; color: var(--text-ob-dim); font-size: 12px; }
.plaza-card__author img { width: 28px; height: 28px; }
.plaza-card__author a { display: inline-flex; align-items: center; min-height: 24px; color: inherit; text-decoration: none; }
.plaza-card__author a:hover { color: var(--color-ob); }
.plaza-card__author time { margin-left: auto; }
.plaza-card__title { color: inherit; font-size: 17px; font-weight: 700; line-height: 1.45; text-decoration: none; }
.plaza-card__title:hover { color: var(--color-ob); }
.plaza-card__body p { display: -webkit-box; margin: 12px 0; overflow: hidden; color: var(--text-ob-dim); font-size: 13px; line-height: 1.7; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.plaza-card__meta { display: flex; gap: 12px; margin-top: auto; padding-top: 14px; color: var(--text-ob-dim); font-size: 11px; }
.plaza-feed__state { margin: 32px 0; color: var(--text-ob-dim); text-align: center; }
.plaza-feed__state.is-error { color: #e2776c; }
.plaza-feed__more { display: block; margin: 28px auto 0; padding: 9px 22px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.plaza-feed__more:hover { border-color: var(--color-ob); color: var(--color-ob); }
@media (max-width: 900px) { .plaza-hero { grid-template-columns: 1fr; } .plaza-hero__signal { min-height: 210px; } .plaza-authors__rail { grid-template-columns: repeat(2, minmax(0, 1fr)); } .plaza-feed__grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 620px) { .plaza-page { padding-top: 4px; } .plaza-hero { border-radius: 20px; } .plaza-authors header, .plaza-feed__head { align-items: flex-start; flex-direction: column; } .plaza-authors__rail, .plaza-feed__grid { grid-template-columns: 1fr; } }
</style>
