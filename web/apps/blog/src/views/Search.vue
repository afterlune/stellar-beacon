<template>
  <div class="search-page">
    <PageHeader :title="t('menu.search')" :current="t('menu.search')" />
    <section class="search-page__hero">
      <p class="search-page__eyebrow">{{ t('search.eyebrow') }}</p>
      <h1>{{ t('search.title') }}</h1>
      <p>{{ t('search.description') }}</p>
      <form class="search-page__form" role="search" @submit.prevent="submitSearch">
        <label class="sr-only" for="search-page-input">{{ t('search.placeholder') }}</label>
        <input
          id="search-page-input"
          v-model="input"
          type="search"
          autocomplete="off"
          :placeholder="t('search.placeholder')" />
        <button type="submit">{{ t('search.submit') }}</button>
      </form>
    </section>

    <section v-if="topicMatches.length" class="search-page__topics" data-testid="search-topics">
      <header>
        <h2>{{ query ? t('search.matchingTopics') : t('search.suggestedTopics') }}</h2>
        <span>{{ t('search.topicHint') }}</span>
      </header>
      <div class="search-page__topic-grid">
        <router-link
          v-for="topic in topicMatches"
          :key="`${topic.type}-${topic.id}`"
          class="search-page__topic"
          :to="topic.path">
          <small>{{ topicLabel(topic.type) }}</small>
          <strong>{{ topic.name }}</strong>
          <span v-if="topic.description">{{ topic.description }}</span>
          <em v-if="topic.count">{{ topic.count }}</em>
        </router-link>
      </div>
    </section>

    <section class="search-page__results" data-testid="search-results">
      <header class="search-page__results-head">
        <div>
          <h2>{{ t('search.articleResults') }}</h2>
          <p v-if="query">{{ t('search.resultCount', { total }) }}</p>
          <p v-else>{{ t('search.emptyPrompt') }}</p>
        </div>
      </header>

      <p v-if="loading" class="search-page__status">{{ t('reactions.loading') }}</p>
      <p v-else-if="errorMessage" class="search-page__status is-error" role="alert">{{ errorMessage }}</p>
      <p v-else-if="query && articles.length === 0" class="search-page__status">{{ t('search.empty') }}</p>
      <ul v-else-if="articles.length" class="search-page__list">
        <li v-for="result in articles" :key="result.id">
          <router-link :to="`/articles/${result.id}`">
            <span class="search-page__result-title" v-html="safeSearchHighlight(result.articleTitle)" />
            <span class="search-page__result-excerpt" v-html="safeSearchHighlight(result.articleContent)" />
            <span class="search-page__result-action">{{ t('search.openArticle') }} →</span>
          </router-link>
        </li>
      </ul>
      <Paginator
        v-if="query && total > pageSize"
        :page-size="pageSize"
        :page-total="total"
        :page="currentPage"
        @pageChange="changePage" />
    </section>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import type { ArticleSearchResult } from '@stellar-beacon/api-contract'

import { PageHeader } from '@/components/PageHeader'
import Paginator from '@/components/Paginator.vue'
import api from '@/api/api'
import { useDiscoveryStore, type DiscoveryTopic } from '@/stores/discovery'
import { normalizeSearchPage, safeSearchHighlight } from '@/utils/search'

const PAGE_SIZE = 20

export default defineComponent({
  name: 'Search',
  components: { PageHeader, Paginator },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const { t } = useI18n()
    const discoveryStore = useDiscoveryStore()
    const input = ref('')
    const query = ref('')
    const articles = ref<ArticleSearchResult[]>([])
    const total = ref(0)
    const loading = ref(false)
    const errorMessage = ref('')
    let requestVersion = 0

    const currentPage = computed(() => Math.max(1, Number(route.query.page || 1) || 1))
    const topicMatches = computed(() => {
      const groups = discoveryStore.topicGroups(query.value)
      return [...groups.series, ...groups.categories, ...groups.tags].slice(0, 8)
    })

    async function load(): Promise<void> {
      query.value = String(route.query.q || '').trim()
      input.value = query.value
      const version = ++requestVersion
      loading.value = true
      errorMessage.value = ''
      try {
        await discoveryStore.load()
        if (!query.value) {
          articles.value = []
          total.value = 0
          return
        }
        const { data } = await api.searchArticles({
          keywords: query.value,
          current: currentPage.value,
          size: PAGE_SIZE
        })
        if (version !== requestVersion) return
        const page = normalizeSearchPage(data?.data)
        articles.value = page.items
        total.value = page.total
      } catch {
        if (version !== requestVersion) return
        articles.value = []
        total.value = 0
        errorMessage.value = t('search.loadFailed')
      } finally {
        if (version === requestVersion) loading.value = false
      }
    }

    function submitSearch(): void {
      const value = input.value.trim()
      void router.push({
        path: '/search',
        query: value ? { q: value } : {}
      })
    }

    function changePage(page: number): void {
      void router.push({
        path: '/search',
        query: { q: query.value, page: page > 1 ? String(page) : undefined }
      })
    }

    function topicLabel(type: DiscoveryTopic['type']): string {
      if (type === 'series') return t('menu.series')
      if (type === 'category') return t('menu.categories')
      return t('menu.tags')
    }

    watch(() => [route.query.q, route.query.page], () => void load(), { immediate: true })

    return {
      t,
      input,
      query,
      articles,
      total,
      pageSize: PAGE_SIZE,
      currentPage,
      loading,
      errorMessage,
      topicMatches,
      submitSearch,
      changePage,
      topicLabel,
      safeSearchHighlight
    }
  }
})
</script>

<style scoped>
.search-page {
  display: grid;
  gap: 24px;
}

.search-page__hero,
.search-page__topics,
.search-page__results {
  padding: clamp(20px, 4vw, 42px);
  border: 1px solid var(--border-hairline);
  border-radius: 18px;
  background: color-mix(in srgb, var(--surface-solid) 86%, transparent);
  box-shadow: inset 0 1px 0 var(--glass-edge), var(--shadow-card);
}

.search-page__eyebrow {
  margin: 0 0 8px;
  color: var(--color-ob);
  font-size: 0.75rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.search-page__hero h1 {
  margin: 0 0 8px;
  font-family: var(--font-display);
  font-size: clamp(2rem, 5vw, 3.4rem);
}

.search-page__hero > p:not(.search-page__eyebrow) {
  margin: 0;
  color: var(--text-dim);
}

.search-page__form {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  max-width: 760px;
  margin-top: 22px;
}

.search-page__form input {
  min-width: 0;
  padding: 13px 16px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  background: var(--surface-2);
  color: var(--text-normal);
  font: inherit;
}

.search-page__form button {
  padding: 0 22px;
  border: 0;
  border-radius: 10px;
  background: var(--brand-gradient);
  color: white;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.search-page__topics header,
.search-page__results-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.search-page__topics h2,
.search-page__results h2 {
  margin: 0;
  font-size: 1.15rem;
}

.search-page__topics header span,
.search-page__results-head p {
  color: var(--text-ob-dim);
  font-size: 0.8rem;
}

.search-page__topic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
}

.search-page__topic {
  display: grid;
  gap: 5px;
  min-width: 0;
  padding: 13px 14px;
  border: 1px solid var(--border-hairline);
  border-radius: 11px;
  color: inherit;
  text-decoration: none;
}

.search-page__topic:hover {
  border-color: var(--color-ob);
}

.search-page__topic small,
.search-page__topic span {
  color: var(--text-ob-dim);
  font-size: 0.75rem;
}

.search-page__topic strong,
.search-page__topic span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-page__list {
  display: grid;
  gap: 10px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.search-page__list a {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px 18px;
  padding: 16px 18px;
  border: 1px solid var(--border-hairline);
  border-radius: 12px;
  color: inherit;
  text-decoration: none;
}

.search-page__list a:hover {
  border-color: var(--color-ob);
}

.search-page__result-title {
  font-size: 1.06rem;
  font-weight: 700;
}

.search-page__result-excerpt {
  grid-column: 1;
  color: var(--text-dim);
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.search-page__result-action {
  grid-row: 1 / span 2;
  grid-column: 2;
  align-self: center;
  color: var(--color-ob);
  font-size: 0.78rem;
  white-space: nowrap;
}

.search-page__status {
  margin: 0;
  padding: 32px 0;
  color: var(--text-dim);
  text-align: center;
}

.search-page__status.is-error {
  color: var(--color-danger, #d9584a);
}

:deep(mark) {
  padding: 0 0.15em;
  border-radius: 3px;
  background: color-mix(in srgb, var(--color-ob) 22%, transparent);
  color: var(--text-bright);
}

@media (max-width: 640px) {
  .search-page__form {
    grid-template-columns: 1fr;
  }

  .search-page__form button {
    min-height: 44px;
  }

  .search-page__list a {
    grid-template-columns: 1fr;
  }

  .search-page__result-action {
    grid-row: auto;
    grid-column: 1;
  }
}
</style>