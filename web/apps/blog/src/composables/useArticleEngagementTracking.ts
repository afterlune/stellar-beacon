import { nextTick, onMounted, onUnmounted, watch, type Ref } from 'vue'
import { API_BASE_URL } from '@stellar-beacon/api-client'

interface ArticleEngagementReader {
  markCompleted(seriesId: number, articleId: number): unknown
  recordProgress(articleId: number, maxScrollPercent: number): unknown
}

interface ArticleEngagementTrackingOptions {
  articleId: () => string | number
  article: () => unknown
  seriesInfo: Ref<Record<string, any> | null>
  seriesArticles: Ref<any[]>
  seriesContextRef: Ref<HTMLElement | null>
  relatedArticlesRef: Ref<HTMLElement | null>
  reader: ArticleEngagementReader
}

function createReadingSessionId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `read-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`
}

export function useArticleEngagementTracking(options: ArticleEngagementTrackingOptions) {
  let readingSessionId = createReadingSessionId()
  let readingActiveMs = 0
  let readingStartedAt: number | null = null
  let readingMaxScrollPercent = 0
  let readingSessionSent = false
  let continuationObserver: IntersectionObserver | null = null
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
    const seriesId = Number(options.seriesInfo.value?.id || 0)
    const articleId = Number(options.articleId())
    if (!seriesId || !articleId || currentReadingActiveMs() < 3000) return
    options.reader.markCompleted(seriesId, articleId)
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

  const clearContinuationTimers = () => {
    for (const timer of continuationTimers.values()) window.clearTimeout(timer)
    continuationTimers.clear()
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
    const articleId = options.articleId()
    if (readingSessionSent || !articleId) return
    pauseReadingSession()
    const activeMs = Math.min(7200000, Math.max(0, Math.round(readingActiveMs)))
    options.reader.recordProgress(Number(articleId), readingMaxScrollPercent)
    if (activeMs < 3000) return
    readingSessionSent = true
    const payload = JSON.stringify({
      sessionId: readingSessionId,
      activeMs,
      maxScrollPercent: Math.round(readingMaxScrollPercent * 100) / 100
    })
    const url = `${API_BASE_URL}/public/articles/${encodeURIComponent(String(articleId))}/read-sessions`
    if (typeof navigator.sendBeacon === 'function' && navigator.sendBeacon(url, new Blob([payload], { type: 'application/json' }))) return
    void fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: payload,
      keepalive: true
    }).catch(() => undefined)
  }

  const sendContinuationEvent = (
    eventType: string,
    targetType?: string,
    targetId?: number,
    placement?: string,
    position?: number
  ) => {
    const articleId = Number(options.articleId())
    if (!Number.isFinite(articleId) || articleId <= 0 || typeof navigator === 'undefined') return
    const payload = JSON.stringify({ eventType, targetType, targetId, placement, position })
    const url = `${API_BASE_URL}/public/articles/${encodeURIComponent(String(articleId))}/continuation-events`
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

  const continuationImpressionKey = (element: Element) => `${options.articleId()}:${element.getAttribute('data-continuation-event') || ''}`

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
    if (!element || !options.articleId()) return
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
      observeContinuationSection(options.seriesContextRef.value)
      observeContinuationSection(options.relatedArticlesRef.value)
    })
  }

  const resetContinuationTracking = () => {
    clearContinuationTimers()
    continuationImpressions.clear()
    continuationObserver?.disconnect()
  }

  const cleanup = () => {
    flushReadingSession()
    resetContinuationTracking()
    document.removeEventListener('visibilitychange', handleReadingVisibility)
    window.removeEventListener('scroll', updateReadingScrollDepth)
    window.removeEventListener('pagehide', flushReadingSession)
    clearReadingCompletionTimer()
  }

  watch([options.article, options.seriesInfo, options.seriesArticles], scheduleContinuationTracking, { flush: 'post' })

  onMounted(() => {
    if (typeof IntersectionObserver !== 'undefined') {
      continuationObserver = new IntersectionObserver(handleContinuationIntersection, { threshold: 0.5 })
    }
    document.addEventListener('visibilitychange', handleReadingVisibility)
    window.addEventListener('scroll', updateReadingScrollDepth, { passive: true })
    window.addEventListener('pagehide', flushReadingSession)
  })

  onUnmounted(cleanup)

  return {
    flushReadingSession,
    resetReadingSession,
    resetContinuationTracking,
    scheduleSeriesCompletionCheck,
    startReadingSession,
    trackContinuationClick
  }
}
