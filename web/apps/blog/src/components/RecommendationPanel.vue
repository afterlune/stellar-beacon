<template>
  <section class="recommendation-panel" :class="{ 'is-compact': compact }" data-testid="recommendation-panel">
    <header class="recommendation-head">
      <div>
        <p>{{ compact ? 'FOR YOU' : 'PERSONAL SIGNAL' }}</p>
        <h2>{{ compact ? '为你推荐' : '根据你的阅读与关注生成' }}</h2>
        <span v-if="personalized">推荐依据只使用你的关注、订阅、互动和本机阅读记录。</span>
        <span v-else>还没有足够的个人信号，先看看近期热门与最新文章。</span>
      </div>
      <router-link v-if="compact" to="/for-you">查看全部 →</router-link>
    </header>

    <p v-if="loading && !items.length" class="recommendation-state">正在整理推荐…</p>
    <p v-else-if="error" class="recommendation-state is-error">{{ error }}</p>
    <div v-else-if="items.length" class="recommendation-grid">
      <article v-for="item in items" :key="item.id" class="recommendation-card" data-testid="recommendation-card">
        <router-link :to="`/articles/${item.id}`" class="recommendation-card__cover">
          <img v-if="item.articleCover" :src="item.articleCover" :alt="item.articleTitle" loading="lazy" />
          <span v-else>{{ String(item.articleTitle || 'SIGNAL').slice(0, 1) }}</span>
        </router-link>
        <div class="recommendation-card__body">
          <div class="recommendation-card__author">
            <img :src="item.author?.avatar || defaultAvatar" :alt="item.author?.nickname || '作者头像'" />
            <router-link :to="`/u/${item.author?.handle}`">{{ item.author?.nickname || item.author?.handle }}</router-link>
            <time>{{ formatDate(item.createTime) }}</time>
          </div>
          <router-link :to="`/articles/${item.id}`" class="recommendation-card__title">{{ item.articleTitle }}</router-link>
          <p>{{ excerpt(item.articleContent) }}</p>
          <footer>
            <span class="recommendation-reason">{{ item.reason?.label || '为你推荐' }}</span>
            <details class="recommendation-menu" @toggle="onMenuToggle">
              <summary>不感兴趣</summary>
              <div>
                <button type="button" @click="applyFeedback(item, { targetType: 'article', articleId: item.id }, '已隐藏这篇文章')">隐藏这篇文章</button>
                <button type="button" @click="applyFeedback(item, { targetType: 'author', authorId: item.userId }, '已减少这位作者')">少看 {{ item.author?.nickname || item.author?.handle }}</button>
                <button v-for="topic in topicOptions(item)" :key="`${topic.topicType}:${topic.topicKey}`" type="button" @click="applyFeedback(item, { targetType: 'topic', topicType: topic.topicType, topicKey: topic.topicKey }, `已减少「${topic.label}」`)">少看「{{ topic.label }}」</button>
              </div>
            </details>
          </footer>
        </div>
      </article>
    </div>
    <p v-else class="recommendation-state">暂时没有可推荐的公开文章。</p>

    <button v-if="!compact && hasMore" type="button" class="recommendation-more" :disabled="loadingMore" @click="loadMore">
      {{ loadingMore ? '正在加载…' : '加载更多' }}
    </button>
  </section>
</template>

<script lang="ts">
import { defineComponent, getCurrentInstance, onMounted, ref } from 'vue'
import api from '@/api/api'
import { useReaderStore } from '@/stores/reader'
import type { RecommendationItem, RecommendationTargetType } from '@stellar-beacon/api-contract'

type TopicOption = { topicType: 'category' | 'tag'; topicKey: string; label: string }
type FeedbackPayload = { targetType: RecommendationTargetType; articleId?: number; authorId?: number; topicType?: 'category' | 'tag'; topicKey?: string }

export default defineComponent({
  name: 'RecommendationPanel',
  props: { compact: { type: Boolean, default: false } },
  setup(props) {
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const readerStore = useReaderStore()
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="72" height="72"%3E%3Crect width="72" height="72" rx="36" fill="%23172554"/%3E%3Ccircle cx="36" cy="27" r="13" fill="%239bb8ff"/%3E%3Cpath d="M12 67c4-17 12-25 24-25s20 8 24 25" fill="%239bb8ff"/%3E%3C/svg%3E'
    const items = ref<RecommendationItem[]>([])
    const cursor = ref('')
    const hasMore = ref(false)
    const personalized = ref(false)
    const loading = ref(false)
    const loadingMore = ref(false)
    const error = ref('')

    const size = () => props.compact ? 6 : 12
    const seeds = () => readerStore.historyEntries.slice(0, 20).map((item) => Number(item.articleId)).filter((id) => Number.isInteger(id) && id > 0)
    const load = async (reset = true) => {
      if (loading.value || loadingMore.value) return
      reset ? loading.value = true : loadingMore.value = true
      error.value = ''
      try {
        const response = await api.getRecommendations({ size: size(), cursor: reset ? undefined : cursor.value || undefined, seedArticleIds: seeds() })
        const data = response?.data?.data || {}
        const next = Array.isArray(data.items) ? data.items : []
        items.value = reset ? next : items.value.concat(next)
        cursor.value = String(data.nextCursor || '')
        hasMore.value = Boolean(data.hasMore)
        personalized.value = Boolean(data.personalized)
      } catch (reason: any) {
        error.value = reason?.response?.data?.message || reason?.message || '推荐加载失败'
      } finally {
        loading.value = false
        loadingMore.value = false
      }
    }
    const loadMore = () => { void load(false) }
    const applyFeedback = async (item: RecommendationItem, payload: FeedbackPayload, message: string) => {
      try {
        const response = await api.saveRecommendationFeedback(payload)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '偏好保存失败')
        if (payload.targetType === 'article') {
          items.value = items.value.filter((row) => row.id !== item.id)
        } else {
          await load(true)
        }
        proxy?.$notify?.({ title: '成功', message, type: 'success' })
      } catch (reason: any) {
        proxy?.$notify?.({ title: '错误', message: reason?.response?.data?.message || reason?.message || '偏好保存失败', type: 'error' })
      }
    }
    const onMenuToggle = (event: Event) => {
      const current = event.currentTarget as HTMLDetailsElement
      if (!current.open) return
      document.querySelectorAll<HTMLDetailsElement>('details.recommendation-menu[open]').forEach((element) => {
        if (element !== current) element.open = false
      })
    }
    const topicOptions = (item: RecommendationItem): TopicOption[] => {
      const result: TopicOption[] = []
      const add = (topicType: 'category' | 'tag', label: string) => {
        const normalized = String(label || '').trim().toLowerCase()
        if (!normalized || result.some((entry) => entry.topicKey === normalized && entry.topicType === topicType)) return
        result.push({ topicType, topicKey: normalized, label: String(label).trim() })
      }
      add('category', item.categoryName || '')
      for (const tag of Array.isArray(item.tags) ? item.tags : []) add('tag', tag)
      return result.slice(0, 4)
    }
    const excerpt = (value: string) => String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 150)
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric' }).format(new Date(value)) : ''

    onMounted(() => { void load(true) })
    return { items, hasMore, personalized, loading, loadingMore, error, defaultAvatar, loadMore, applyFeedback, onMenuToggle, topicOptions, excerpt, formatDate }
  }
})
</script>

<style scoped>
.recommendation-panel { padding: clamp(20px, 4vw, 34px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 92% 0, rgba(91, 120, 229, .14), transparent 33%), color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.recommendation-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-bottom: 22px; }
.recommendation-head p { margin: 0 0 7px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.recommendation-head h2 { margin: 0 0 7px; font-size: clamp(1.45rem, 3vw, 2.1rem); }
.recommendation-head span, .recommendation-head a { color: var(--text-ob-dim); font-size: 12px; text-decoration: none; }
.recommendation-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
.is-compact .recommendation-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.recommendation-card { display: flex; min-width: 0; flex-direction: column; overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 17px; background: color-mix(in srgb, var(--background-primary) 94%, transparent); }
.recommendation-card__cover { display: block; aspect-ratio: 16 / 9; overflow: hidden; background: linear-gradient(135deg, #152250, #3c2c70); color: rgba(218, 228, 255, .9); text-decoration: none; }
.recommendation-card__cover img { width: 100%; height: 100%; object-fit: cover; }
.recommendation-card__cover span { display: grid; place-items: center; height: 100%; font-size: 3rem; font-weight: 800; }
.recommendation-card__body { display: flex; flex: 1; flex-direction: column; padding: 15px; }
.recommendation-card__author { display: flex; align-items: center; gap: 8px; margin-bottom: 11px; color: inherit; font-size: 11px; text-decoration: none; }
.recommendation-card__author img { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
.recommendation-card__author a { display: inline-flex; align-items: center; min-height: 24px; color: inherit; text-decoration: none; }
.recommendation-card__author time { margin-left: auto; color: var(--text-ob-dim); font-size: 10px; }
.recommendation-card__title { display: block; min-height: 24px; color: inherit; font-size: 1rem; font-weight: 700; line-height: 1.45; text-decoration: none; }
.recommendation-card__body > p { flex: 1; margin: 8px 0 14px; color: var(--text-ob-dim); font-size: 12px; line-height: 1.65; }
.recommendation-card footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.recommendation-reason { overflow: hidden; color: var(--color-ob); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.recommendation-menu { position: relative; flex: none; }
.recommendation-menu summary { padding: 5px 9px; border: 1px solid var(--border-hairline); border-radius: 999px; color: var(--text-ob-dim); font-size: 10px; cursor: pointer; list-style: none; }
.recommendation-menu summary::-webkit-details-marker { display: none; }
.recommendation-menu > div { position: absolute; right: 0; bottom: calc(100% + 7px); z-index: 8; display: grid; width: 190px; padding: 7px; border: 1px solid var(--border-hairline); border-radius: 12px; background: var(--background-primary); box-shadow: 0 16px 40px rgba(2, 8, 24, .28); }
.recommendation-menu button { padding: 8px 9px; border: 0; border-radius: 8px; background: transparent; color: inherit; font: inherit; font-size: 11px; text-align: left; cursor: pointer; }
.recommendation-menu button:hover { background: color-mix(in srgb, var(--color-ob) 12%, transparent); }
.recommendation-state { padding: 48px 0; color: var(--text-ob-dim); text-align: center; }
.recommendation-state.is-error { color: #df8177; }
.recommendation-more { display: block; margin: 24px auto 0; padding: 9px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 900px) { .recommendation-grid, .is-compact .recommendation-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 620px) { .recommendation-head { align-items: flex-start; flex-direction: column; } .recommendation-grid, .is-compact .recommendation-grid { grid-template-columns: 1fr; } }
</style>
