<template>
  <div class="flex flex-col">
    <div class="main-grid">
      <div class="post-header">
        <span class="post-labels">
          <ob-skeleton v-if="loading" tag="b" height="20px" width="35px" />
          <b v-else-if="!loading && article.categoryName">
            <span>{{ article.categoryName }}</span>
          </b>
          <b v-else>{{ t('settings.default-category') }}</b>
          <ul>
            <ob-skeleton v-if="loading" :count="2" tag="li" height="16px" width="35px" class="mr-2" />
            <template v-else-if="!loading && normalizedTags.length > 0">
              <li v-for="tag in normalizedTags" :key="tag.id">
                <em class="opacity-50">#</em>
                {{ tag.tagName }}
              </li>
            </template>
            <template v-else>
              <li>
                <b class="opacity-50">#</b>
                {{ t('settings.default-tag') }}
              </li>
            </template>
          </ul>
        </span>
        <h1 v-if="article.articleTitle" class="post-title text-white">
          {{ article.articleTitle }}
        </h1>
        <ob-skeleton
          v-else
          class="post-title text-white uppercase"
          width="100%"
          height="clamp(1.2rem, calc(1rem + 3.5vw), 4rem)" />
        <div class="flex flex-row items-center justify-start mt-4 mb-4">
          <div class="post-footer" v-if="article.author">
            <img
              class="hover:opacity-50 cursor-pointer"
              v-lazy="article.author.avatar || avatarPlaceholder"
              alt="author avatar"
              @error="handleImageError"
              @click="handleAuthorClick(article.author.website)" />
            <span class="text-white opacity-80">
              <strong
                class="text-white pr-1.5 hover:opacity-50 cursor-pointer"
                @click="handleAuthorClick(article.author.website)">
                {{ article.author.nickname }}
              </strong>
              <span class="opacity-70">
                {{ t('settings.shared-on') }} {{ t(`settings.months[${new Date(article.createTime).getMonth()}]`) }}
                {{ new Date(article.createTime).getDate() }}, {{ new Date(article.createTime).getFullYear() }}
              </span>
            </span>
          </div>
          <div class="post-footer" v-else>
            <div class="flex flex-row items-center">
              <ob-skeleton class="mr-2" height="28px" width="28px" :circle="true" />
              <span class="text-ob-dim mt-1">
                <ob-skeleton height="20px" width="150px" />
              </span>
            </div>
          </div>
          <div class="post-stats" v-if="wordNum !== '' && readTime !== ''">
            <span>
              <svg-icon icon-class="text-outline" style="stroke: white" />
              <span class="pl-2 opacity-70">
                {{ wordNum }}
              </span>
            </span>
            <span>
              <svg-icon icon-class="clock-outline" style="stroke: white" />
              <span class="pl-2 opacity-70">
                {{ readTime }}
              </span>
            </span>
          </div>
          <div v-else class="post-stats">
            <span>
              <svg-icon icon-class="clock" />
              <span class="pl-2">
                <ob-skeleton width="40px" height="16px" />
              </span>
            </span>
            <span>
              <svg-icon icon-class="text" />
              <span class="pl-2">
                <ob-skeleton width="40px" height="16px" />
              </span>
            </span>
          </div>
        </div>
      </div>
    </div>
    <div class="main-grid">
      <div>
        <template v-if="article.articleContent">
          <div class="post-html" ref="articleRef" v-html="article.articleContent" />
        </template>
        <div v-else class="surface-panel px-14 py-16 rounded-2xl block min-h-screen">
          <ob-skeleton tag="div" :count="1" height="36px" width="150px" class="mb-6" />
          <br />
          <ob-skeleton tag="div" :count="35" height="16px" width="100px" class="mr-2" />
          <br />
          <br />
          <ob-skeleton tag="div" :count="25" height="16px" width="100px" class="mr-2" />
        </div>
        <div class="article-actions" v-if="article.articleTitle">
          <span class="article-actions__label">{{ t('newsletter.share') }}</span>
          <button type="button" @click="shareArticle">{{ navigatorShareAvailable ? t('newsletter.native') : t('newsletter.copy') }}</button>
          <button type="button" @click="copyArticleLink">{{ t('newsletter.copy') }}</button>
          <span v-if="shareMessage" class="article-actions__message">{{ shareMessage }}</span>
        </div>
        <div class="article-reactions" v-if="article.articleTitle">
          <button
            type="button"
            class="article-reaction"
            :class="{ 'article-reaction--active': reactions.like }"
            :aria-pressed="reactions.like"
            :disabled="reactionPending"
            data-testid="article-like"
            @click="toggleReaction('like')">
            <span>{{ reactions.like ? t('reactions.liked') : t('reactions.like') }}</span>
            <span class="article-reaction__count">{{ reactions.likeCount }}</span>
          </button>
          <button
            type="button"
            class="article-reaction"
            :class="{ 'article-reaction--active': reactions.favorite }"
            :aria-pressed="reactions.favorite"
            :disabled="reactionPending"
            data-testid="article-favorite"
            @click="toggleReaction('favorite')">
            <span>{{ reactions.favorite ? t('reactions.favorited') : t('reactions.favorite') }}</span>
            <span class="article-reaction__count">{{ reactions.favoriteCount }}</span>
          </button>
          <router-link v-if="userToken" class="article-reaction__link" to="/favorites">{{ t('reactions.favorites') }}</router-link>
          <router-link v-if="seriesInfo" class="article-reaction__link" :to="'/series/' + seriesInfo.id">
            {{ t('series.inSeries', { name: seriesInfo.seriesName }) }}
          </router-link>
          <span v-if="reactionMessage" class="article-actions__message">{{ reactionMessage }}</span>
        </div>
        <section
          v-if="seriesInfo && seriesIndex >= 0"
          ref="seriesContextRef"
          class="series-context"
          data-testid="series-context"
          data-continuation-event="series_impression"
          :aria-label="t('series.inSeries', { name: seriesInfo.seriesName })">
          <div class="series-context__head">
            <div>
              <span class="series-context__eyebrow">{{ t('series.progress', { current: seriesIndex + 1, total: seriesArticles.length }) }}</span>
              <strong>{{ seriesInfo.seriesName }}</strong>
            </div>
            <router-link :to="'/series/' + seriesInfo.id" @click="trackContinuationClick('series_click', 'series', seriesInfo.id, 'series_index', 0)">{{ t('series.backToSeries') }}</router-link>
          </div>
          <div class="series-context__nav">
            <router-link v-if="seriesPrevious" class="series-context__link" :to="'/articles/' + seriesPrevious.id" data-testid="series-previous" @click="trackContinuationClick('series_click', 'article', seriesPrevious.id, 'series_previous', 0)">
              <small>{{ t('series.previous') }}</small>
              <span>{{ seriesPrevious.articleTitle }}</span>
            </router-link>
            <router-link v-if="seriesNext" class="series-context__link series-context__link--next" :to="'/articles/' + seriesNext.id" data-testid="series-next" @click="trackContinuationClick('series_click', 'article', seriesNext.id, 'series_next', 0)">
              <small>{{ t('series.next') }}</small>
              <span>{{ seriesNext.articleTitle }}</span>
            </router-link>
          </div>
        </section>
        <section v-if="article.relatedArticles && article.relatedArticles.length" ref="relatedArticlesRef" class="related-articles" data-testid="related-articles" data-continuation-event="related_impression" :aria-labelledby="'related-title-' + articleId">
          <h2 :id="'related-title-' + articleId">{{ t('newsletter.related') }}</h2>
          <div class="related-articles__grid">
            <router-link v-for="(related, index) in article.relatedArticles" :key="related.id" :to="'/articles/' + related.id" class="related-article" @click="trackContinuationClick('related_click', 'article', related.id, 'related', Number(index) + 1)">
              <span>{{ related.categoryName || t('settings.default-category') }}</span>
              <strong>{{ related.articleTitle }}</strong>
            </router-link>
          </div>
        </section>
        <NewsletterSubscribe />
        <div class="flex flex-col lg:flex-row justify-start items-end my-8 my-gap">
          <div class="w-full h-full self-stretch mr-0 lg:mr-4" v-if="preArticleCard">
            <SubTitle title="settings.paginator.pre" icon="arrow-left-circle" />
            <ArticleCard class="pre-and-next-article" :data="preArticleCard" />
          </div>
          <div class="w-full h-full self-stretch mt-0" v-if="nextArticleCard">
            <SubTitle title="settings.paginator.next" :side="!isMobile ? 'right' : 'left'" icon="arrow-right-circle" />
            <ArticleCard class="pre-and-next-article" :data="nextArticleCard" />
          </div>
        </div>
        <Comment />
      </div>
      <div>
        <Sidebar>
          <Profile />
          <Sticky :stickyTop="32" endingElId="footer" dynamicElClass="#sticky-sidebar">
            <div id="sticky-sidebar">
              <transition name="fade-slide-y" mode="out-in">
                <div class="sidebar-box mb-4">
                  <SubTitle :title="'titles.toc'" icon="toc" compact />
                  <div id="toc1"></div>
                </div>
              </transition>
              <Navigator />
            </div>
          </Sticky>
        </Sidebar>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Sidebar, Profile, Navigator } from '@/components/Sidebar'
import {
  computed,
  defineComponent,
  nextTick,
  onUnmounted,
  onMounted,
  reactive,
  ref,
  toRefs,
  provide,
  getCurrentInstance,
  watch
} from 'vue'
import { useRoute, useRouter, onBeforeRouteUpdate } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Comment } from '@/components/Comment'
import { SubTitle } from '@/components/Title'
import { ArticleCard } from '@/components/ArticleCard'
import '@/styles/prism-aurora-future.css'
import { useCommonStore } from '@/stores/common'
import { useCommentStore } from '@/stores/comment'
import { useUserStore } from '@/stores/user'
import Sticky from '@/components/Sticky.vue'
import Prism from '@/utils/prism'
import MarkdownIt from 'markdown-it'
import tocbot from 'tocbot'
import emitter from '@/utils/mitt'
import { v3ImgPreviewFn } from 'v3-img-preview'
import api from '@/api/api'
import markdownToHtml from '@/utils/markdown'
import avatarPlaceholder from '@/assets/avatar-placeholder.svg'
import { pageCount, pageRecords } from '@/utils/page'
import NewsletterSubscribe from '@/components/NewsletterSubscribe.vue'
import { useSeoMeta } from '@/composables/useSeoMeta'
import { useReaderStore } from '@/stores/reader'
import { scrollToArticleHeading } from '@/utils/article-reader'
import { applyArticleImageFallback } from '@/utils/article-image'
import { normalizeArticleTags } from '@/utils/article-tags'
import { API_BASE_URL } from '@stellar-beacon/api-client'

function normalizeArticleHeadings(html: string): string {
  return String(html || '')
    .replace(/<h1(\s[^>]*)?>/gi, '<h2$1>')
    .replace(/<\/h1>/gi, '</h2>')
}
function createReadingSessionId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `read-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`
}

export default defineComponent({
  name: 'Article',
  components: { Sidebar, Comment, SubTitle, ArticleCard, Profile, Sticky, Navigator, NewsletterSubscribe },
  setup() {
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const commonStore = useCommonStore()
    const commentStore = useCommentStore()
    const route = useRoute()
    const router = useRouter()
    const { t } = useI18n()
    const { setSeo } = useSeoMeta()
    const readerStore = useReaderStore()
    const loading = ref(true)
    const articleRef = ref()
    const md = new MarkdownIt({ html: true })
    const reactiveData = reactive({
      articleId: '' as any,
      article: '' as any,
      wordNum: '' as any,
      readTime: '' as any,
      comments: [] as any,
      images: [] as any,
      preArticleCard: '' as any,
      nextArticleCard: '' as any,
      haveMore: false as any,
      isReload: false as any
    })
    const shareMessage = ref('')
    const userStore = useUserStore()
    const reactionPending = ref(false)
    const reactionMessage = ref('')
    const reactions = reactive({
      like: false,
      favorite: false,
      likeCount: 0,
      favoriteCount: 0
    })
    const userToken = computed(() => Boolean(userStore.token))
    const seriesInfo = ref<any>(null)
    const seriesArticles = ref<any[]>([])
    const seriesContextRef = ref<HTMLElement | null>(null)
    const relatedArticlesRef = ref<HTMLElement | null>(null)
    const seriesIndex = computed(() => seriesArticles.value.findIndex((item) => Number(item?.id) === Number(reactiveData.articleId)))
    const seriesPrevious = computed(() => (seriesIndex.value > 0 ? seriesArticles.value[seriesIndex.value - 1] : null))
    const seriesNext = computed(() => (
      seriesIndex.value >= 0 && seriesIndex.value < seriesArticles.value.length - 1
        ? seriesArticles.value[seriesIndex.value + 1]
        : null
    ))
    const navigatorShareAvailable = computed(() => typeof navigator !== 'undefined' && typeof navigator.share === 'function')
    const pageInfo = reactive({
      current: 1,
      size: 7
    })
    let readingSessionId = createReadingSessionId()
    let readingActiveMs = 0
    let readingStartedAt: number | null = null
    let readingMaxScrollPercent = 0
    let readingSessionSent = false
    let continuationObserver: IntersectionObserver | null = null
    let tocObserver: IntersectionObserver | null = null
    let readingCompletionTimer: number | null = null
    let seriesCompletionMarked = false
    const continuationTimers = new Map<Element, number>()
    const continuationImpressions = new Set<string>()

    const clearReadingCompletionTimer = () => {
      if (readingCompletionTimer === null) return
      window.clearTimeout(readingCompletionTimer)
      readingCompletionTimer = null
    }
    const currentReadingActiveMs = () => (
      readingActiveMs + (readingStartedAt === null ? 0 : performance.now() - readingStartedAt)
    )
    const maybeMarkSeriesCompleted = () => {
      if (seriesCompletionMarked || readingMaxScrollPercent < 90) return
      const seriesId = Number(seriesInfo.value?.id || 0)
      const articleId = Number(reactiveData.articleId)
      if (!seriesId || !articleId || currentReadingActiveMs() < 3000) return
      readerStore.markCompleted(seriesId, articleId)
      seriesCompletionMarked = true
      clearReadingCompletionTimer()
    }
    const scheduleSeriesCompletionCheck = () => {
      clearReadingCompletionTimer()
      if (seriesCompletionMarked) return
      const remaining = Math.max(0, 3000 - currentReadingActiveMs())
      if (remaining === 0) {
        maybeMarkSeriesCompleted()
        return
      }
      readingCompletionTimer = window.setTimeout(maybeMarkSeriesCompleted, remaining)
    }

    const resetReadingSession = () => {
      clearReadingCompletionTimer()
      seriesCompletionMarked = false
      readingSessionId = createReadingSessionId()
      readingActiveMs = 0
      readingStartedAt = null
      readingMaxScrollPercent = 0
      readingSessionSent = false
    }
    const pauseReadingSession = () => {
      clearReadingCompletionTimer()
      if (readingStartedAt === null) return
      readingActiveMs += performance.now() - readingStartedAt
      readingStartedAt = null
    }
    const startReadingSession = () => {
      if (readingSessionSent || readingStartedAt !== null) return
      readingStartedAt = performance.now()
      scheduleSeriesCompletionCheck()
      updateReadingScrollDepth()
    }
    const updateReadingScrollDepth = () => {
      const root = document.documentElement
      const scrollable = Math.max(1, root.scrollHeight - window.innerHeight)
      const depth = Math.min(100, Math.max(0, ((window.scrollY + window.innerHeight) / (scrollable + window.innerHeight)) * 100))
      readingMaxScrollPercent = Math.max(readingMaxScrollPercent, depth)
      maybeMarkSeriesCompleted()
      scheduleSeriesCompletionCheck()
    }
    const handleReadingVisibility = () => {
      if (document.hidden) {
        clearContinuationTimers()
        continuationObserver?.disconnect()
        pauseReadingSession()
        return
      }
      startReadingSession()
      scheduleContinuationTracking()
    }
    const flushReadingSession = () => {
      maybeMarkSeriesCompleted()
      if (readingSessionSent || !reactiveData.articleId) return
      pauseReadingSession()
      const activeMs = Math.min(7200000, Math.max(0, Math.round(readingActiveMs)))
      readerStore.recordProgress(Number(reactiveData.articleId), readingMaxScrollPercent)
      if (activeMs < 3000) return
      readingSessionSent = true
      const payload = JSON.stringify({
        sessionId: readingSessionId,
        activeMs,
        maxScrollPercent: Math.round(readingMaxScrollPercent * 100) / 100
      })
      const url = `${API_BASE_URL}/public/articles/${encodeURIComponent(reactiveData.articleId)}/read-sessions`
      if (typeof navigator.sendBeacon === 'function' && navigator.sendBeacon(url, new Blob([payload], { type: 'application/json' }))) return
      void fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload,
        keepalive: true
      }).catch(() => undefined)
    }
    const continuationEventUrl = (articleId: number) => `${API_BASE_URL}/public/articles/${encodeURIComponent(String(articleId))}/continuation-events`

    const sendContinuationEvent = (
      eventType: string,
      targetType?: string,
      targetId?: number,
      placement?: string,
      position?: number
    ) => {
      const articleId = Number(reactiveData.articleId)
      if (!Number.isFinite(articleId) || articleId <= 0 || typeof navigator === 'undefined') return
      const payload = JSON.stringify({ eventType, targetType, targetId, placement, position })
      const url = continuationEventUrl(articleId)
      if (typeof navigator.sendBeacon === 'function' && navigator.sendBeacon(url, new Blob([payload], { type: 'application/json' }))) return
      void fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload,
        keepalive: true
      }).catch(() => undefined)
    }

    const trackContinuationClick = (eventType: string, targetType: string, targetId: number, placement: string, position: number) => {
      sendContinuationEvent(eventType, targetType, targetId, placement, position)
    }

    const clearContinuationTimers = () => {
      for (const timer of continuationTimers.values()) window.clearTimeout(timer)
      continuationTimers.clear()
    }

    const continuationImpressionKey = (element: Element) => `${reactiveData.articleId}:${element.getAttribute('data-continuation-event') || ''}`

    const handleContinuationIntersection = (entries: IntersectionObserverEntry[]) => {
      for (const entry of entries) {
        const eventType = entry.target.getAttribute('data-continuation-event')
        if (!eventType) continue
        const key = continuationImpressionKey(entry.target)
        if (continuationImpressions.has(key)) continue
        if (entry.isIntersecting && !document.hidden) {
          if (continuationTimers.has(entry.target)) continue
          const timer = window.setTimeout(() => {
            continuationTimers.delete(entry.target)
            if (!continuationImpressions.has(key)) {
              continuationImpressions.add(key)
              sendContinuationEvent(eventType)
            }
            continuationObserver?.unobserve(entry.target)
          }, 1000)
          continuationTimers.set(entry.target, timer)
          continue
        }
        const timer = continuationTimers.get(entry.target)
        if (timer !== undefined) {
          window.clearTimeout(timer)
          continuationTimers.delete(entry.target)
        }
      }
    }

    const observeContinuationSection = (element: HTMLElement | null) => {
      if (!element || !reactiveData.articleId) return
      const eventType = element.getAttribute('data-continuation-event')
      if (!eventType) return
      const key = continuationImpressionKey(element)
      if (continuationImpressions.has(key)) return
      if (!continuationObserver) {
        continuationImpressions.add(key)
        sendContinuationEvent(eventType)
        return
      }
      continuationObserver.observe(element)
    }

    const scheduleContinuationTracking = () => {
      void nextTick(() => {
        observeContinuationSection(seriesContextRef.value)
        observeContinuationSection(relatedArticlesRef.value)
      })
    }

    const resetContinuationTracking = () => {
      clearContinuationTimers()
      continuationImpressions.clear()
      continuationObserver?.disconnect()
    }

    watch([() => reactiveData.article, seriesInfo, seriesArticles], scheduleContinuationTracking, { flush: 'post' })
    commentStore.type = 1
    onMounted(() => {
      if (typeof IntersectionObserver !== 'undefined') {
        continuationObserver = new IntersectionObserver(handleContinuationIntersection, { threshold: 0.5 })
      }
      document.addEventListener('visibilitychange', handleReadingVisibility)
      window.addEventListener('scroll', updateReadingScrollDepth, { passive: true })
      window.addEventListener('pagehide', flushReadingSession)
      reactiveData.articleId = route.params.articleId
      toPageTop()
      fetchArticle()
      fetchComments()
    })
    onUnmounted(() => {
      flushReadingSession()
      resetContinuationTracking()
      document.removeEventListener('visibilitychange', handleReadingVisibility)
      window.removeEventListener('scroll', updateReadingScrollDepth)
      window.removeEventListener('pagehide', flushReadingSession)
      clearReadingCompletionTimer()
      tocObserver?.disconnect()
      tocObserver = null
      readerStore.clearArticleContext()
      commonStore.resetHeaderImage()
      reactiveData.article = ''
      tocbot.destroy()
    })
    onBeforeRouteUpdate((to) => {
      flushReadingSession()
      resetReadingSession()
      resetContinuationTracking()
      tocObserver?.disconnect()
      tocObserver = null
      readerStore.clearArticleContext()
      reactiveData.article = ''
      reactiveData.readTime = ''
      reactiveData.wordNum = ''
      reactiveData.comments = []
      reactiveData.images = []
      reactiveData.preArticleCard = ''
      reactiveData.nextArticleCard = ''
      shareMessage.value = ''
      reactionMessage.value = ''
      seriesInfo.value = null
      seriesArticles.value = []
      reactions.like = false
      reactions.favorite = false
      reactiveData.articleId = to.params.articleId
      pageInfo.current = 1
      reactiveData.isReload = true
      toPageTop()
      fetchArticle()
      fetchComments()
    })
    provide(
      'comments',
      computed(() => reactiveData.comments)
    )
    provide(
      'haveMore',
      computed(() => reactiveData.haveMore)
    )
    emitter.on('articleFetchComment', () => {
      pageInfo.current = 1
      reactiveData.isReload = true
      fetchComments()
    })
    emitter.on('articleFetchReplies', (index) => {
      fetchReplies(index)
    })

    emitter.on('articleLoadMore', () => {
      fetchComments()
    })
    const handlePreview = (index: any) => {
      v3ImgPreviewFn({ images: reactiveData.images, index: reactiveData.images.indexOf(index) })
    }
    const initTocbot = () => {
      if (!articleRef.value) return
      tocbot.destroy()
      tocObserver?.disconnect()
      tocObserver = null

      const headings = Array.from(articleRef.value.querySelectorAll('h1, h2, h3')) as HTMLElement[]
      const usedIds = new Set<string>()
      const tocItems = headings.map((heading, index) => {
        const baseId = heading.id || `article-heading-${index + 1}`
        let id = baseId
        let suffix = 2
        while (usedIds.has(id)) {
          id = `${baseId}-${suffix}`
          suffix++
        }
        usedIds.add(id)
        heading.id = id
        return {
          id,
          text: heading.textContent?.trim() || t('titles.toc'),
          level: Number(heading.tagName.slice(1))
        }
      })
      readerStore.setTocItems(tocItems)

      if (typeof IntersectionObserver !== 'undefined' && headings.length) {
        tocObserver = new IntersectionObserver((entries) => {
          const visible = entries
            .filter((entry) => entry.isIntersecting)
            .sort((left, right) => left.boundingClientRect.top - right.boundingClientRect.top)
          if (visible[0]) readerStore.setActiveHeading(visible[0].target.id)
        }, { rootMargin: '-96px 0px -68% 0px', threshold: 0 })
        headings.forEach((heading) => tocObserver?.observe(heading))
      }

      tocbot.init({
        tocSelector: '#toc1',
        contentSelector: '.post-html',
        headingSelector: 'h1, h2, h3',
        onClick: (event: MouseEvent) => {
          event.preventDefault()
          const link = event.currentTarget as HTMLAnchorElement | null
          const id = decodeURIComponent(link?.getAttribute('href')?.slice(1) || '')
          if (id && scrollToArticleHeading(id)) readerStore.setActiveHeading(id)
        }
      })

      const imgs = articleRef.value.getElementsByTagName('img')
      for (let i = 0; i < imgs.length; i++) {
        reactiveData.images.push(imgs[i].src)
        imgs[i].addEventListener('error', () => applyArticleImageFallback(imgs[i]))
        imgs[i].addEventListener('click', (event: Event) => {
          const target = event.target as HTMLImageElement
          handlePreview(target.currentSrc)
        })
      }
    }
    const fetchArticle = () => {
      loading.value = true
      api.getArticeById(reactiveData.articleId).then(({ data }) => {
        if (data.code === 52003) {
          proxy.$notify({
            title: '错误',
            message: '文章密码认证未通过',
            type: 'error'
          })
          router.push({ path: '/出错啦' })
          return
        }
        if (data.data === null) {
          router.push({ path: '/出错啦' })
          return
        }
        commonStore.setHeaderImage(data.data.articleCover)
        new Promise((resolve) => {
          data.data.articleContent = normalizeArticleHeadings(data.data.articleContentHtml || markdownToHtml(data.data.articleContent))
          resolve(data.data)
        }).then((article: any) => {
          reactiveData.article = article
          readerStore.recordVisit({
            articleId: Number(article.id),
            articleTitle: String(article.articleTitle || ''),
            seriesId: Number(article.seriesId) > 0 ? Number(article.seriesId) : null
          })
          syncReactionTotals(article)
          fetchSeriesInfo(article)
          fetchReactionStates()
          setSeo({
            title: `${article.articleTitle} · Stellar Beacon`,
            description: deleteHTMLTag(article.articleContent).slice(0, 180),
            canonical: window.location.href,
            image: article.articleCover,
            type: 'article',
            jsonLd: {
              '@context': 'https://schema.org',
              '@type': 'Article',
              headline: article.articleTitle,
              datePublished: article.createTime,
              dateModified: article.updateTime,
              mainEntityOfPage: window.location.href
            }
          })
          reactiveData.wordNum = Math.round(deleteHTMLTag(article.articleContent).length / 100) / 10 + 'k'
          reactiveData.readTime = Math.round(deleteHTMLTag(article.articleContent).length / 400) + 'mins'
          loading.value = false
          startReadingSession()
          nextTick(() => {
            Prism.highlightAll()
            initTocbot()
          })
        })
        new Promise((resolve) => {
          data.data.preArticleCard.articleContent = md
            .render(data.data.preArticleCard.articleContent)
            .replace(/<\/?[^>]*>/g, '')
            .replace(/[|]*\n/, '')
            .replace(/&npsp;/gi, '')
          resolve(data.data.preArticleCard)
        }).then((preArticleCard: any) => {
          reactiveData.preArticleCard = preArticleCard
        })
        new Promise((resolve) => {
          data.data.nextArticleCard.articleContent = md
            .render(data.data.nextArticleCard.articleContent)
            .replace(/<\/?[^>]*>/g, '')
            .replace(/[|]*\n/, '')
            .replace(/&npsp;/gi, '')
          resolve(data.data.nextArticleCard)
        }).then((nextArticleCard) => {
          reactiveData.nextArticleCard = nextArticleCard
        })
      })
    }
    const focusComment = (commentID: number) => {
      if (commentID <= 0) return
      nextTick(() => {
        const element = document.getElementById(`comment-${commentID}`)
        if (!element) return
        element.scrollIntoView({ behavior: 'smooth', block: 'center' })
        element.classList.remove('comment-focus')
        void element.offsetWidth
        element.classList.add('comment-focus')
        window.setTimeout(() => element.classList.remove('comment-focus'), 2500)
      })
    }
    const fetchComments = () => {
      const commentID = Number(route.query.comment || 0)
      const params: any = {
        type: 1,
        topicId: reactiveData.articleId,
        current: pageInfo.current,
        size: pageInfo.size
      }
      if (commentID > 0) params.focusCommentId = commentID
      api.getComments(params).then(({ data }) => {
        const wasReload = reactiveData.isReload
        const records = pageRecords(data)
        if (wasReload) {
          reactiveData.comments = records
          reactiveData.isReload = false
        } else {
          reactiveData.comments.push(...records)
        }
        if (commentID > 0 && wasReload) {
          pageInfo.current = Number(data?.data?.page || pageInfo.current)
        }
        if (pageCount(data) <= reactiveData.comments.length) {
          reactiveData.haveMore = false
        } else {
          reactiveData.haveMore = true
        }
        pageInfo.current++
        focusComment(commentID)
      })
    }
    const fetchReplies = (index: any) => {
      api.getRepliesByCommentId(reactiveData.comments[index].id).then(({ data }) => {
        reactiveData.comments[index].replyDTOs = data.data
      })
    }
    const handleAuthorClick = (link: string) => {
      if (link === '') link = window.location.href
      window.location.href = link
    }
    const handleImageError = (event: Event) => {
      const image = event.target as HTMLImageElement
      if (image.dataset.fallbackApplied === 'true') return
      image.dataset.fallbackApplied = 'true'
      image.src = avatarPlaceholder
    }
    const toPageTop = () => {
      window.scrollTo({
        top: 0
      })
    }
    const copyArticleLink = async () => {
      try {
        await navigator.clipboard.writeText(window.location.href)
        shareMessage.value = t('newsletter.copied')
      } catch {
        shareMessage.value = window.location.href
      }
      void api.trackGrowthEvent({ eventName: 'share_click', articleId: Number(reactiveData.articleId), path: window.location.pathname })
    }
    const shareArticle = async () => {
      if (!navigator.share) {
        await copyArticleLink()
        return
      }
      try {
        await navigator.share({ title: reactiveData.article.articleTitle, url: window.location.href })
        shareMessage.value = t('newsletter.native')
        void api.trackGrowthEvent({ eventName: 'share_click', articleId: Number(reactiveData.articleId), path: window.location.pathname })
      } catch {
        // A cancelled native share is not a failed page action.
      }
    }
    const deleteHTMLTag = (content: any) => {
      return content
        .replace(/<\/?[^>]*>/g, '')
        .replace(/[|]*\n/, '')
        .replace(/&npsp;/gi, '')
    }
    // The article payload carries shared totals; the per-account state comes
    // from the authenticated endpoint so the shared article cache stays valid.
    const syncReactionTotals = (article: any) => {
      reactions.likeCount = Number(article?.likeCount || 0)
      reactions.favoriteCount = Number(article?.favoriteCount || 0)
    }
    // The article payload only carries the series id; the badge needs its name.
    const fetchSeriesInfo = (article: any) => {
      seriesInfo.value = null
      seriesArticles.value = []
      readerStore.setSeriesContext(null)
      const seriesId = Number(article?.seriesId || 0)
      if (!seriesId) return
      api
        .getSeriesDetail(seriesId)
        .then(({ data }: any) => {
          const payload = data?.data || {}
          const series = payload.series || null
          const articles: any[] = Array.isArray(payload.articles) ? payload.articles : []
          const articleId = Number(reactiveData.articleId)
          const currentIndex = articles.findIndex((item) => Number(item?.id) === articleId)
          const previous = currentIndex > 0 ? articles[currentIndex - 1] : null
          const next = currentIndex >= 0 && currentIndex < articles.length - 1 ? articles[currentIndex + 1] : null
          seriesInfo.value = series
          seriesArticles.value = articles
          readerStore.setSeriesContext({
            id: seriesId,
            name: String(series?.seriesName || ''),
            articleIds: articles.map((item) => Number(item?.id)).filter((id) => Number.isInteger(id) && id > 0),
            currentArticleId: articleId,
            currentIndex,
            total: articles.length,
            previous: previous ? { id: Number(previous.id), articleTitle: String(previous.articleTitle || '') } : null,
            next: next ? { id: Number(next.id), articleTitle: String(next.articleTitle || '') } : null
          })
          if (articleId > 0) {
            readerStore.markOpened(seriesId, articleId)
            scheduleSeriesCompletionCheck()
          }
        })
        .catch(() => {
          seriesInfo.value = null
          seriesArticles.value = []
          readerStore.setSeriesContext(null)
        })
    }
    const fetchReactionStates = () => {
      reactions.like = false
      reactions.favorite = false
      if (!userStore.token || !reactiveData.articleId) return
      const articleId = Number(reactiveData.articleId)
      if (!Number.isFinite(articleId) || articleId <= 0) return
      api
        .getArticleReactionStates([articleId])
        .then(({ data }: any) => {
          const entry = (data?.data || [])[0]
          if (!entry) return
          reactions.like = Boolean(entry.like)
          reactions.favorite = Boolean(entry.favorite)
        })
        .catch(() => {
          reactions.like = false
          reactions.favorite = false
        })
    }
    const toggleReaction = (kind: 'like' | 'favorite') => {
      if (!userStore.token) {
        userStore.userVisible = true
        reactionMessage.value = t('reactions.loginRequired')
        return
      }
      const articleId = Number(reactiveData.articleId)
      if (!Number.isFinite(articleId) || articleId <= 0 || reactionPending.value) return
      reactionPending.value = true
      reactionMessage.value = ''
      api
        .toggleArticleReaction({ articleId, reaction: kind, active: !reactions[kind] })
        .then(({ data }: any) => {
          if (data?.code && data.code !== 'OK') {
            reactionMessage.value = t('reactions.failed')
            return
          }
          const payload = data?.data || {}
          reactions[kind] = Boolean(payload.active)
          reactions.likeCount = Number(payload.likeCount || 0)
          reactions.favoriteCount = Number(payload.favoriteCount || 0)
          reactionMessage.value = payload.active ? t('reactions.saved') : t('reactions.removed')
        })
        .catch(() => {
          reactionMessage.value = t('reactions.failed')
        })
        .finally(() => {
          reactionPending.value = false
        })
    }
    return {
      articleRef,
      ...toRefs(reactiveData),
      normalizedTags: computed(() => normalizeArticleTags(reactiveData.article?.tags)),
      isMobile: computed(() => commonStore.isMobile),
      handleAuthorClick,
      handleImageError,
      avatarPlaceholder,
      loading,
      t,
      shareMessage,
      navigatorShareAvailable,
      copyArticleLink,
      shareArticle,
      reactions,
      reactionPending,
      reactionMessage,
      userToken,
      seriesInfo,
      seriesArticles,
      seriesIndex,
      seriesPrevious,
      seriesNext,
      seriesContextRef,
      relatedArticlesRef,
      trackContinuationClick,
      toggleReaction
    }
  }
})
</script>
<style lang="scss">
.post-html {
  word-wrap: break-word;
  word-break: break-all;
}
#toc1 > ol {
  list-style: none;
  counter-reset: li;
  padding-left: 1.5rem;

  > li {
    @apply font-medium pb-1;
    &.is-active-li > .node-name--H1 {
      @apply text-ob;
    }
    &.is-active-li > .node-name--H2 {
      @apply text-ob;
    }
    &.is-active-li > .node-name--H3 {
      @apply text-ob;
    }
  }

  ol li {
    @apply font-medium mt-1.5 mb-1.5;
    padding-left: 1.5rem;
    &.is-active-li > .node-name--H2 {
      @apply text-ob;
    }
    &.is-active-li > .node-name--H3 {
      @apply text-ob;
    }
    ol li {
      @apply font-medium mt-1.5 mb-1.5;
      padding-left: 1.5rem;
      &.is-active-li .node-name--H3 {
        @apply text-ob;
      }
    }
  }

  ol,
  ol ol {
    position: relative;
  }

  > li::before,
  ol > li::before,
  ol ol > li::before,
  ol ol ol > li::before,
  ol ol ol ol > li::before {
    content: '•';
    color: var(--text-accent);
    display: inline-block;
    width: 1em;
    margin-left: -1.15em;
    padding: 0;
    font-weight: medium;
    text-shadow: 0 0 0.5em var(--accent-2);
  }

  > li::before {
    @apply text-xl;
  }

  > li > ol::before,
  > li > ol > li > ol::before {
    content: '';
    border-left: 1px solid var(--text-accent);
    position: absolute;
    opacity: 0.35;
    left: -1em;
    top: 0;
    bottom: 0;
  }

  > li > ol::before {
    left: -1.25em;
    border-left: 2px solid var(--text-accent);
  }
}
.pre-and-next-article {
  .article-content {
    p {
      overflow: hidden;
      text-overflow: ellipsis;
      display: -webkit-box;
      -webkit-line-clamp: 5;
      -webkit-box-orient: vertical;
    }
    .article-footer {
      margin-top: 13px;
    }
  }
}
.article-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: .65rem;
  margin: 1.5rem 0 2rem;
  padding-top: 1rem;
  border-top: 1px solid color-mix(in srgb, var(--text-ob-dim) 22%, transparent);
}
.article-actions__label { margin-right: .35rem; font-size: .8rem; opacity: .58; }
.article-actions button { padding: .5rem .85rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.article-actions button:hover { border-color: var(--color-ob); color: var(--color-ob); }
.article-actions__message { color: var(--color-ob); font-size: .82rem; }
.article-reactions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: .6rem;
  margin: -1rem 0 2rem;
}
.article-reaction {
  display: inline-flex;
  align-items: center;
  gap: .45rem;
  padding: .45rem .9rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  transition: border-color .2s ease, color .2s ease;
}
.article-reaction:hover:not(:disabled) { border-color: var(--color-ob); color: var(--color-ob); }
.article-reaction:disabled { opacity: .6; cursor: progress; }
.article-reaction--active { border-color: var(--color-ob); color: var(--color-ob); }
.article-reaction__count { font-variant-numeric: tabular-nums; opacity: .7; }
.article-reaction__link { font-size: .82rem; opacity: .7; text-decoration: underline; }
.series-context {
  margin: 2.5rem 0;
  padding: 1rem 1.1rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 24%, transparent);
  border-radius: .9rem;
  background: color-mix(in srgb, var(--surface-solid) 72%, transparent);
}
.series-context__head { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: .85rem; }
.series-context__head strong { display: block; margin-top: .2rem; }
.series-context__head > a { color: var(--color-ob); font-size: .82rem; white-space: nowrap; }
.series-context__eyebrow { color: var(--text-ob-dim); font-size: .72rem; letter-spacing: .04em; text-transform: uppercase; }
.series-context__nav { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: .75rem; }
.series-context__link { display: flex; min-height: 74px; flex-direction: column; justify-content: space-between; gap: .35rem; padding: .75rem .85rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 20%, transparent); border-radius: .7rem; color: inherit; text-decoration: none; }
.series-context__link:hover { border-color: var(--color-ob); }
.series-context__link small { color: var(--text-ob-dim); font-size: .7rem; }
.series-context__link span { line-height: 1.4; }
.series-context__link--next { text-align: right; }
@media (max-width: 640px) {
  .series-context__head { align-items: flex-start; }
  .series-context__link--next { text-align: left; }
}
.related-articles { margin: 2.5rem 0; }
.related-articles h2 { margin: 0 0 1rem; font-family: var(--font-display); font-size: 1.45rem; }
.related-articles__grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .85rem; }
.related-article { display: flex; min-height: 108px; flex-direction: column; justify-content: space-between; padding: 1rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 22%, transparent); border-radius: .9rem; color: inherit; text-decoration: none; transition: transform .2s ease, border-color .2s ease; }
.related-article:hover { transform: translateY(-3px); border-color: var(--color-ob); }
.related-article span { font-size: .72rem; color: var(--color-ob); text-transform: uppercase; }
.related-article strong { line-height: 1.45; }
@media (max-width: 800px) { .related-articles__grid { grid-template-columns: 1fr; } }
</style>
<style lang="scss" scoped>
.my-gap {
  gap: 1rem;
}
</style>
