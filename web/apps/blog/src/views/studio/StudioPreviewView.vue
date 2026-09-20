<template>
  <div class="studio-preview-page">
    <header class="studio-preview-head">
      <button type="button" @click="backToList">← 返回{{ kindLabel }}列表</button>
      <div>
        <p>PREVIEW / {{ kindLabel }}</p>
        <h1>发布形态预览</h1>
        <span>预览不会增加阅读量，也不会触发评论或订阅分发。</span>
      </div>
      <div class="studio-preview-head__actions">
        <router-link :to="editPath">编辑</router-link>
        <a v-if="publicLink" :href="publicLink" target="_blank" rel="noopener noreferrer">打开公开页</a>
      </div>
    </header>

    <p v-if="loading" class="studio-preview-state">正在生成预览…</p>
    <p v-else-if="error" class="studio-preview-state is-error">{{ error }}</p>

    <template v-else>
      <aside class="studio-preview-notice" :class="{ 'is-hidden': moderationStatus === 'hidden' }">
        <strong>{{ noticeTitle }}</strong>
        <span>{{ noticeText }}</span>
      </aside>

      <main v-if="kind === 'article'" class="studio-preview-article">
        <img v-if="article.articleCover" :src="article.articleCover" :alt="article.articleTitle" class="studio-preview-cover" />
        <p class="studio-preview-kicker">{{ article.categoryName || '未分类' }} · {{ typeLabel(article.type) }}</p>
        <h1>{{ article.articleTitle }}</h1>
        <div class="studio-preview-byline">
          <img :src="author.avatar || defaultAvatar" :alt="author.nickname || '作者'" />
          <span><strong>{{ author.nickname || author.handle || '作者' }}</strong><small>@{{ author.handle || 'author' }}</small></span>
        </div>
        <article class="studio-preview-content" v-html="articleHtml"></article>
      </main>

      <main v-else-if="kind === 'talk'" class="studio-preview-talk">
        <div class="studio-preview-byline">
          <img :src="author.avatar || defaultAvatar" :alt="author.nickname || '作者'" />
          <span><strong>{{ author.nickname || author.handle || '作者' }}</strong><small>@{{ author.handle || 'author' }}</small></span>
        </div>
        <p>{{ talk.content }}</p>
        <div v-if="talkImages.length" class="studio-preview-talk__images" :class="{ 'is-single': talkImages.length === 1 }">
          <img v-for="image in talkImages" :key="image" :src="image" alt="" />
        </div>
      </main>

      <main v-else class="studio-preview-series">
        <header>
          <img v-if="series.cover" :src="series.cover" :alt="series.seriesName" />
          <div><p>SERIES / {{ seriesArticles.length }} ARTICLES</p><h1>{{ series.seriesName }}</h1><span>{{ series.seriesDesc || '暂无系列简介' }}</span></div>
        </header>
        <ol>
          <li v-for="(item, index) in seriesArticles" :key="item.id">
            <router-link :to="`/studio/articles/${item.id}/preview`">
              <span>{{ index + 1 }}</span>
              <strong>{{ item.articleTitle }}</strong>
              <em :class="`status-${item.status}`">{{ statusLabel(item.status) }}</em>
            </router-link>
          </li>
        </ol>
      </main>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import api from '@/api/api'
import { useUserStore } from '@/stores/user'
import markdownToHtml, { sanitizePreviewHtml } from '@/utils/markdown'
import { parseTalkImages } from '@/utils/studioUpload'

type Kind = 'article' | 'talk' | 'series'

const props = defineProps<{ kind: Kind }>()
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const loading = ref(true)
const error = ref('')
const article = ref<any>({})
const talk = ref<any>({})
const series = ref<any>({})
const seriesArticles = ref<any[]>([])
const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'

const contentID = computed(() => Number(route.params.id || 0))
const kindLabel = computed(() => props.kind === 'article' ? '文章' : props.kind === 'talk' ? '随想' : '系列')
const kindPath = computed(() => props.kind === 'article' ? 'articles' : props.kind === 'talk' ? 'talks' : 'series')
const author = computed(() => userStore.userInfo || {})
const editPath = computed(() => `/studio/${kindPath.value}/${contentID.value}/edit`)
const currentStatus = computed(() => Number(props.kind === 'article' ? article.value.status : props.kind === 'talk' ? talk.value.status : series.value.status) || 0)
const moderationStatus = computed(() => props.kind === 'article' ? article.value.moderationStatus : props.kind === 'talk' ? talk.value.moderationStatus : series.value.moderationStatus)
const moderationReason = computed(() => props.kind === 'article' ? article.value.moderationReason : props.kind === 'talk' ? talk.value.moderationReason : series.value.moderationReason)
const publicLink = computed(() => currentStatus.value === 1 && moderationStatus.value !== 'hidden' ? `/${kindPath.value}/${contentID.value}` : '')
const talkImages = computed(() => parseTalkImages(talk.value.images ?? talk.value.imgs))
const articleHtml = computed(() => {
  const html = String(article.value.articleContentHtml || '')
  return sanitizePreviewHtml(html || markdownToHtml(String(article.value.articleContent || '')))
})
const noticeTitle = computed(() => moderationStatus.value === 'hidden' ? '内容已被审核隐藏' : currentStatus.value === 1 ? '公开内容预览' : '未发布预览')
const noticeText = computed(() => {
  if (moderationStatus.value === 'hidden') return moderationReason.value || '该内容当前不会出现在公开空间。'
  if (currentStatus.value === 4 && article.value.scheduledAt) return `计划于 ${formatDateTime(article.value.scheduledAt)} 发布。`
  if (currentStatus.value === 1) return '当前内容已公开，预览仅用于检查最终呈现。'
  return '只有你登录后可以看到此预览，公开读者无法访问。'
})

function formatDateTime(value: string): string {
  return value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : ''
}

function statusLabel(value: number): string {
  return ({ 1: '公开', 2: '私有', 3: '草稿', 4: '定时' } as Record<number, string>)[value] || '未知'
}

function typeLabel(value: number): string {
  return ({ 1: '原创', 2: '转载', 3: '翻译' } as Record<number, string>)[value] || '原创'
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    if (contentID.value <= 0) throw new Error('内容 ID 不正确')
    if (props.kind === 'article') {
      const response = await api.getStudioArticle(contentID.value)
      if (!response?.data?.flag || !response.data.data) throw new Error(response?.data?.message || '文章不存在')
      article.value = response.data.data
    } else if (props.kind === 'talk') {
      const response = await api.getStudioTalk(contentID.value)
      if (!response?.data?.flag || !response.data.data) throw new Error(response?.data?.message || '随想不存在')
      talk.value = response.data.data
    } else {
      const [detailResponse, articlesResponse] = await Promise.all([
        api.getStudioSeriesItem(contentID.value),
        api.getStudioArticles({ current: 1, size: 100, seriesId: contentID.value })
      ])
      if (!detailResponse?.data?.flag || !detailResponse.data.data) throw new Error(detailResponse?.data?.message || '系列不存在')
      series.value = detailResponse.data.data
      const page = articlesResponse?.data?.data || {}
      seriesArticles.value = Array.isArray(page.records) ? page.records : Array.isArray(page.items) ? page.items : []
    }
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '预览加载失败'
  } finally {
    loading.value = false
  }
}

function backToList(): void {
  void router.push(`/studio/${kindPath.value}`)
}

watch([() => props.kind, contentID], () => { void load() }, { immediate: true })
</script>

<style lang="scss" scoped>
.studio-preview-page { min-width: 0; padding-bottom: 90px; }
.studio-preview-head { display: grid; grid-template-columns: auto 1fr auto; gap: 18px; align-items: center; padding: 18px 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-preview-head > button, .studio-preview-head__actions a { padding: 8px 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; font: inherit; font-size: 12px; text-decoration: none; cursor: pointer; }
.studio-preview-head p { margin: 0 0 4px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-preview-head h1 { margin: 0 0 4px; font-size: 1.5rem; }
.studio-preview-head span { color: var(--text-ob-dim); font-size: 11px; }
.studio-preview-head__actions { display: flex; gap: 8px; }
.studio-preview-state { padding: 60px 0; color: var(--text-ob-dim); text-align: center; }
.studio-preview-state.is-error { color: #df8177; }
.studio-preview-notice { display: flex; align-items: center; gap: 12px; margin: 16px 0; padding: 12px 16px; border: 1px solid color-mix(in srgb, var(--color-ob) 40%, transparent); border-radius: 12px; color: var(--color-ob); font-size: 12px; }
.studio-preview-notice.is-hidden { border-color: rgba(223, 129, 119, .5); color: #df8177; }
.studio-preview-notice span { color: var(--text-ob-dim); }
.studio-preview-article { max-width: 820px; margin: 32px auto 0; }
.studio-preview-cover { width: 100%; max-height: 430px; border-radius: 18px; object-fit: cover; }
.studio-preview-kicker { margin: 28px 0 8px; color: var(--color-ob); font-size: 11px; letter-spacing: .12em; }
.studio-preview-article > h1, .studio-preview-series h1 { margin: 0 0 18px; font-size: clamp(2rem, 5vw, 3.6rem); letter-spacing: -.05em; }
.studio-preview-byline { display: flex; align-items: center; gap: 10px; margin-bottom: 28px; }
.studio-preview-byline img { width: 42px; height: 42px; border-radius: 50%; object-fit: cover; }
.studio-preview-byline strong, .studio-preview-byline small { display: block; }
.studio-preview-byline small { color: var(--text-ob-dim); }
.studio-preview-content { line-height: 1.9; }
.studio-preview-content :deep(img) { max-width: 100%; height: auto; border-radius: 12px; }
.studio-preview-content :deep(h1), .studio-preview-content :deep(h2), .studio-preview-content :deep(h3) { margin-top: 1.8em; }
.studio-preview-content :deep(pre) { overflow-x: auto; padding: 16px; border-radius: 12px; background: var(--background-primary-alt); }
.studio-preview-talk { max-width: 720px; margin: 32px auto 0; padding: clamp(24px, 5vw, 48px); border: 1px solid var(--border-hairline); border-radius: 20px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.studio-preview-talk > p { margin: 0 0 22px; line-height: 1.9; white-space: pre-wrap; }
.studio-preview-talk__images { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
.studio-preview-talk__images.is-single { grid-template-columns: 1fr; }
.studio-preview-talk__images img { width: 100%; border-radius: 12px; object-fit: cover; }
.studio-preview-series { max-width: 820px; margin: 32px auto 0; }
.studio-preview-series > header { display: grid; grid-template-columns: 220px 1fr; gap: 24px; align-items: center; padding: 24px; border: 1px solid var(--border-hairline); border-radius: 20px; }
.studio-preview-series > header img { width: 100%; height: 150px; border-radius: 14px; object-fit: cover; }
.studio-preview-series header p { color: var(--color-ob); font-size: 10px; letter-spacing: .14em; }
.studio-preview-series header span { color: var(--text-ob-dim); line-height: 1.7; }
.studio-preview-series ol { display: grid; gap: 10px; margin: 20px 0 0; padding: 0; list-style: none; }
.studio-preview-series li a { display: grid; grid-template-columns: 38px 1fr auto; gap: 12px; align-items: center; padding: 15px 18px; border: 1px solid var(--border-hairline); border-radius: 13px; color: inherit; text-decoration: none; }
.studio-preview-series li a:hover { border-color: color-mix(in srgb, var(--color-ob) 50%, transparent); }
.studio-preview-series li span { color: var(--text-ob-dim); }
.studio-preview-series li em { padding: 3px 8px; border: 1px solid var(--border-hairline); border-radius: 999px; font-size: 10px; font-style: normal; }
.studio-preview-series li em.status-1 { color: #78d0bb; }
.studio-preview-series li em.status-4 { color: #bca0ef; }
@media (max-width: 720px) { .studio-preview-head { grid-template-columns: 1fr; } .studio-preview-head__actions { flex-wrap: wrap; } .studio-preview-series > header { grid-template-columns: 1fr; } .studio-preview-series > header img { height: 190px; } .studio-preview-talk__images { grid-template-columns: 1fr; } }
</style>