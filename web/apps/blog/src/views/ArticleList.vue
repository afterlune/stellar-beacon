<template>
  <div class="flex flex-col">
    <PageHeader :title="pageTitle" />
    <div class="surface-panel px-14 py-16 rounded-2xl block">
      <p v-if="loading" class="taxonomy-status">{{ t('reactions.loading') }}</p>
      <p v-else-if="articles.length === 0" class="taxonomy-status">{{ t('taxonomy.emptyArticles') }}</p>
      <ul v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-10">
        <li v-for="article in articles" :key="article.id">
          <ArticleCard class="tag-article" :data="article" />
        </li>
      </ul>
      <Paginator
        v-if="pagination.total > pagination.size"
        :page-size="pagination.size"
        :page-total="pagination.total"
        :page="pagination.current"
        @pageChange="pageChangeHandler" />
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import MarkdownIt from 'markdown-it'

import { PageHeader } from '@/components/PageHeader'
import { ArticleCard } from '@/components/ArticleCard'
import Paginator from '@/components/Paginator.vue'
import api from '@/api/api'
import { useDiscoveryStore } from '@/stores/discovery'

function positiveQueryNumber(value: unknown): number {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1
}

export default defineComponent({
  name: 'ArticleList',
  components: { PageHeader, ArticleCard, Paginator },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const { t } = useI18n()
    const discoveryStore = useDiscoveryStore()
    const md = new MarkdownIt({ html: true })
    const articles = ref<any[]>([])
    const loading = ref(false)
    const pageTitle = ref('')
    const pagination = reactive({
      size: 12,
      total: 0,
      current: 1
    })

    const isCategory = computed(() => route.name === 'CategoryArticles')
    const routeId = computed(() => Number(isCategory.value ? route.params.categoryId : route.params.tagId))

    function queryName(): string {
      const value = isCategory.value ? route.query.name : route.query.tagName
      return String(value || '').trim()
    }

    async function resolveTitle(): Promise<void> {
      const fromQuery = queryName()
      if (fromQuery) {
        pageTitle.value = fromQuery
        return
      }
      await discoveryStore.load()
      if (isCategory.value) {
        const item = discoveryStore.categories.find((category) => Number(category.id) === routeId.value)
        pageTitle.value = item?.categoryName || t('menu.categories')
      } else {
        const item = discoveryStore.tags.find((tag) => Number(tag.id) === routeId.value)
        pageTitle.value = item?.tagName || t('menu.tags')
      }
    }

    async function fetchArticles(): Promise<void> {
      loading.value = true
      articles.value = []
      try {
        const params = {
          current: pagination.current,
          size: pagination.size
        }
        const response = isCategory.value
          ? await api.getArticlesByCategoryId({ ...params, categoryId: routeId.value })
          : await api.getArticlesByTagId({ ...params, tagId: routeId.value })
        const payload = response.data?.data || {}
        const records = Array.isArray(payload.items) ? payload.items : Array.isArray(payload.records) ? payload.records : []
        records.forEach((item: any) => {
          item.articleContent = md
            .render(item.articleContent)
            .replace(/<\/?[^>]*>/g, '')
            .replace(/[|]*\n/, '')
            .replace(/&npsp;/gi, '')
        })
        articles.value = records
        pagination.total = Number(payload.total ?? payload.count ?? records.length)
      } catch {
        articles.value = []
        pagination.total = 0
      } finally {
        loading.value = false
      }
    }

    async function loadRoute(): Promise<void> {
      pagination.current = positiveQueryNumber(route.query.page)
      await resolveTitle()
      await fetchArticles()
    }

    function pageChangeHandler(page: number): void {
      void router.push({
        query: {
          ...route.query,
          page: page > 1 ? String(page) : undefined
        }
      })
    }

    watch(
      () => [route.name, route.params.categoryId, route.params.tagId, route.query.name, route.query.tagName, route.query.page],
      () => void loadRoute(),
      { immediate: true }
    )

    return {
      t,
      articles,
      loading,
      pageTitle,
      pagination,
      pageChangeHandler
    }
  }
})
</script>

<style lang="scss">
.taxonomy-status {
  margin: 0;
  padding: 4rem 0;
  color: var(--text-dim);
  text-align: center;
}
.tag-article {
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
</style>