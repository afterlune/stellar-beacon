import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import api from '@/api/api'

export interface DiscoveryCategory {
  id: number
  categoryName: string
  articleCount: number
}

export interface DiscoveryTag {
  id: number
  tagName: string
  count: number
}

export interface DiscoverySeries {
  id: number
  seriesName: string
  seriesDesc: string
  cover: string
  articleCount: number
  updateTime?: string
}

export interface DiscoveryTopic {
  type: 'category' | 'tag' | 'series'
  id: number
  name: string
  description: string
  count: number
  path: string
}

export interface DiscoveryTopicGroups {
  categories: DiscoveryTopic[]
  tags: DiscoveryTopic[]
  series: DiscoveryTopic[]
}

function payloadList<T>(value: unknown): T[] {
  return Array.isArray(value) ? value as T[] : []
}

function includesKeyword(value: string, keyword: string): boolean {
  return value.toLocaleLowerCase().includes(keyword.toLocaleLowerCase())
}

export const useDiscoveryStore = defineStore('discoveryStore', () => {
  const categories = ref<DiscoveryCategory[]>([])
  const tags = ref<DiscoveryTag[]>([])
  const series = ref<DiscoverySeries[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  let loadPromise: Promise<void> | null = null

  // Empty taxonomy entries are noise in discovery surfaces, so they are kept
  // out of the ranked lists while the raw lists stay available elsewhere.
  const visibleCategories = computed(() => [...categories.value]
    .filter((item) => Number(item.articleCount || 0) > 0)
    .sort((left, right) => Number(right.articleCount || 0) - Number(left.articleCount || 0)))
  const visibleTags = computed(() => [...tags.value]
    .filter((item) => Number(item.count || 0) > 0)
    .sort((left, right) => Number(right.count || 0) - Number(left.count || 0)))
  const topCategories = computed(() => visibleCategories.value.slice(0, 5))
  const topTags = computed(() => visibleTags.value.slice(0, 8))
  const featuredSeries = computed(() => [...series.value]
    .sort((left, right) => (
      Number(right.articleCount || 0) - Number(left.articleCount || 0)
      || String(right.updateTime || '').localeCompare(String(left.updateTime || ''))
    ))
    .slice(0, 3))

  function topicGroups(value: unknown): DiscoveryTopicGroups {
    const keyword = String(value ?? '').trim()
    const hasKeyword = keyword.length > 0
    const toCategory = (item: DiscoveryCategory): DiscoveryTopic => ({
      type: 'category',
      id: Number(item.id),
      name: String(item.categoryName || ''),
      description: '',
      count: Number(item.articleCount || 0),
      path: `/categories/${item.id}?name=${encodeURIComponent(item.categoryName || '')}`
    })
    const toTag = (item: DiscoveryTag): DiscoveryTopic => ({
      type: 'tag',
      id: Number(item.id),
      name: String(item.tagName || ''),
      description: '',
      count: Number(item.count || 0),
      path: `/tags/${item.id}?tagName=${encodeURIComponent(item.tagName || '')}`
    })
    const toSeries = (item: DiscoverySeries): DiscoveryTopic => ({
      type: 'series',
      id: Number(item.id),
      name: String(item.seriesName || ''),
      description: String(item.seriesDesc || ''),
      count: Number(item.articleCount || 0),
      path: `/series/${item.id}`
    })
    const matching = (items: DiscoveryTopic[]) => items
      .filter((item) => !hasKeyword || includesKeyword(item.name, keyword) || includesKeyword(item.description, keyword))
      .slice(0, 5)
    return {
      categories: matching(visibleCategories.value.map(toCategory)),
      tags: matching(visibleTags.value.map(toTag)),
      series: matching(series.value.map(toSeries))
    }
  }

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) return
    if (loadPromise) return loadPromise
    loading.value = true
    loadPromise = Promise.allSettled([
      api.getAllCategories(),
      api.getAllTags(),
      api.getSeriesList()
    ]).then(([categoryResponse, tagResponse, seriesResponse]) => {
      if (categoryResponse.status === 'fulfilled') {
        categories.value = payloadList<DiscoveryCategory>(categoryResponse.value?.data?.data)
      }
      if (tagResponse.status === 'fulfilled') {
        tags.value = payloadList<DiscoveryTag>(tagResponse.value?.data?.data)
      }
      if (seriesResponse.status === 'fulfilled') {
        series.value = payloadList<DiscoverySeries>(seriesResponse.value?.data?.data)
      }
      loaded.value = true
    }).finally(() => {
      loading.value = false
      loadPromise = null
    })
    return loadPromise
  }

  return {
    categories,
    tags,
    series,
    loading,
    loaded,
    visibleCategories,
    visibleTags,
    topCategories,
    topTags,
    featuredSeries,
    topicGroups,
    load
  }
})