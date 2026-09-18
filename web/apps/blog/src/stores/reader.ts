import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface ReaderTocItem {
  id: string
  text: string
  level: number
}

export interface ReaderSeriesLink {
  id: number
  articleTitle: string
}

export interface ReaderSeriesContext {
  id: number
  name: string
  articleIds: number[]
  currentArticleId: number
  currentIndex: number
  total: number
  previous: ReaderSeriesLink | null
  next: ReaderSeriesLink | null
}

export interface SeriesProgressEntry {
  lastArticleId: number | null
  completedArticleIds: number[]
  updatedAt: string
}

export interface SeriesProgressSummary {
  total: number
  completedCount: number
  percent: number
  lastArticleId: number | null
  continueArticleId: number | null
  completedArticleIds: number[]
}

type SeriesProgressMap = Record<string, SeriesProgressEntry>

export const SERIES_PROGRESS_STORAGE_KEY = 'stellar-beacon.reader.series-progress.v1'

function readStoredProgress(): SeriesProgressMap {
  if (typeof localStorage === 'undefined') return {}
  try {
    const raw = localStorage.getItem(SERIES_PROGRESS_STORAGE_KEY)
    if (!raw) return {}
    return normalizeProgress(JSON.parse(raw))
  } catch {
    return {}
  }
}

function normalizeProgress(value: unknown): SeriesProgressMap {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  const result: SeriesProgressMap = {}
  for (const [seriesId, rawEntry] of Object.entries(value as Record<string, unknown>)) {
    if (!/^\d+$/.test(seriesId) || !rawEntry || typeof rawEntry !== 'object' || Array.isArray(rawEntry)) continue
    const entry = rawEntry as Record<string, unknown>
    const lastArticleId = Number(entry.lastArticleId)
    const completedArticleIds = Array.isArray(entry.completedArticleIds)
      ? Array.from(new Set(entry.completedArticleIds.map(Number).filter((id) => Number.isInteger(id) && id > 0)))
      : []
    result[seriesId] = {
      lastArticleId: Number.isInteger(lastArticleId) && lastArticleId > 0 ? lastArticleId : null,
      completedArticleIds,
      updatedAt: typeof entry.updatedAt === 'string' ? entry.updatedAt : ''
    }
  }
  return result
}

export const useReaderStore = defineStore('readerStore', () => {
  const tocItems = ref<ReaderTocItem[]>([])
  const activeHeadingId = ref('')
  const seriesContext = ref<ReaderSeriesContext | null>(null)
  const seriesProgress = ref<SeriesProgressMap>(readStoredProgress())

  function persistProgress(): void {
    if (typeof localStorage === 'undefined') return
    try {
      localStorage.setItem(SERIES_PROGRESS_STORAGE_KEY, JSON.stringify(seriesProgress.value))
    } catch {
      // Private browsing or a full storage quota must not block reading.
    }
  }

  function setTocItems(items: ReaderTocItem[]): void {
    tocItems.value = items
    activeHeadingId.value = items[0]?.id || ''
  }

  function setActiveHeading(id: string): void {
    if (id) activeHeadingId.value = id
  }

  function setSeriesContext(context: ReaderSeriesContext | null): void {
    seriesContext.value = context
  }

  function clearArticleContext(): void {
    tocItems.value = []
    activeHeadingId.value = ''
    seriesContext.value = null
  }

  function updateEntry(seriesId: number, update: (current: SeriesProgressEntry) => SeriesProgressEntry): void {
    if (!Number.isInteger(seriesId) || seriesId <= 0) return
    const key = String(seriesId)
    const current = seriesProgress.value[key] || { lastArticleId: null, completedArticleIds: [], updatedAt: '' }
    seriesProgress.value = {
      ...seriesProgress.value,
      [key]: update(current)
    }
    persistProgress()
  }

  function markOpened(seriesId: number, articleId: number): void {
    if (!Number.isInteger(articleId) || articleId <= 0) return
    updateEntry(seriesId, (current) => ({
      ...current,
      lastArticleId: articleId,
      updatedAt: new Date().toISOString()
    }))
  }

  function markCompleted(seriesId: number, articleId: number): void {
    if (!Number.isInteger(articleId) || articleId <= 0) return
    updateEntry(seriesId, (current) => ({
      lastArticleId: articleId,
      completedArticleIds: Array.from(new Set([...current.completedArticleIds, articleId])),
      updatedAt: new Date().toISOString()
    }))
  }

  function seriesSummary(seriesId: number, articleIds: number[]): SeriesProgressSummary {
    const orderedIds = Array.from(new Set(articleIds.map(Number).filter((id) => Number.isInteger(id) && id > 0)))
    const entry = seriesProgress.value[String(seriesId)]
    const completedSet = new Set(entry?.completedArticleIds || [])
    const completedArticleIds = orderedIds.filter((id) => completedSet.has(id))
    const lastArticleId = entry?.lastArticleId && orderedIds.includes(entry.lastArticleId) ? entry.lastArticleId : null
    const firstUnread = orderedIds.find((id) => !completedSet.has(id)) || null
    const continueArticleId = lastArticleId && !completedSet.has(lastArticleId)
      ? lastArticleId
      : firstUnread || orderedIds[orderedIds.length - 1] || null
    const total = orderedIds.length
    const completedCount = completedArticleIds.length
    return {
      total,
      completedCount,
      percent: total ? Math.round((completedCount / total) * 100) : 0,
      lastArticleId,
      continueArticleId,
      completedArticleIds
    }
  }

  return {
    tocItems,
    activeHeadingId,
    seriesContext,
    seriesProgress,
    setTocItems,
    setActiveHeading,
    setSeriesContext,
    clearArticleContext,
    markOpened,
    markCompleted,
    seriesSummary
  }
})