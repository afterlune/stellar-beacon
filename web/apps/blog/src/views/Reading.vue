<template>
  <div class="reading-page">
    <PageHeader :title="t('reading.pageTitle')" :current="t('menu.reading')" />

    <section class="reading-page__hero">
      <p class="reading-page__eyebrow">{{ t('reading.eyebrow') }}</p>
      <h2 class="reading-page__hero-title">{{ t('reading.title') }}</h2>
      <p>{{ t('reading.description') }}</p>
    </section>

    <p v-if="clearedNotice" class="reading-page__status is-accent" role="status" data-testid="reading-cleared">
      {{ t('reading.cleared') }}
    </p>
    <p v-if="seriesError" class="reading-page__status is-error" role="alert">{{ t('reading.loadFailed') }}</p>

    <section v-if="seriesCards.length" class="reading-page__panel" data-testid="reading-continue">
      <header class="reading-page__panel-head">
        <div>
          <h2>{{ t('reading.continueTitle') }}</h2>
          <p>{{ t('reading.continueHint') }}</p>
        </div>
      </header>
      <ul class="reading-page__series">
        <li v-for="card in seriesCards" :key="card.id">
          <div class="reading-page__series-card">
            <strong>{{ card.name || t('reading.unknownSeries') }}</strong>
            <span v-if="card.resolved" class="reading-page__series-progress">
              {{ t('reading.progress', { completed: card.completed, total: card.total }) }}
            </span>
            <router-link
              v-if="card.continueId"
              class="reading-page__series-action"
              :to="`/articles/${card.continueId}`">
              <span>{{ t('reading.continueAction') }}</span>
              <em v-if="card.continueTitle">{{ card.continueTitle }}</em>
              <span aria-hidden="true">→</span>
            </router-link>
          </div>
        </li>
      </ul>
    </section>

    <section v-if="recentEntries.length" class="reading-page__panel" data-testid="reading-recent">
      <header class="reading-page__panel-head">
        <div>
          <h2>{{ t('reading.recentTitle') }}</h2>
          <p>{{ t('reading.recentHint') }}</p>
        </div>
        <div class="reading-page__clear">
          <button v-if="!confirmingClear" type="button" data-testid="reading-clear" @click="confirmingClear = true">
            {{ t('reading.clear') }}
          </button>
          <template v-else>
            <span class="reading-page__clear-hint">{{ t('reading.clearConfirm') }}</span>
            <button type="button" data-testid="reading-clear-confirm" @click="confirmClear">
              {{ t('reading.confirmAction') }}
            </button>
            <button type="button" @click="confirmingClear = false">{{ t('reading.cancelAction') }}</button>
          </template>
        </div>
      </header>
      <ul class="reading-page__list">
        <li v-for="entry in recentEntries" :key="entry.articleId">
          <router-link :to="`/articles/${entry.articleId}`">
            <span class="reading-page__title">{{ entry.articleTitle || `#${entry.articleId}` }}</span>
            <span class="reading-page__meta">
              <span v-if="entry.progressPercent >= 5">
                {{ t('reading.progressPercent', { percent: Math.round(entry.progressPercent) }) }}
              </span>
              <span v-if="formatVisitedAt(entry.visitedAt)">{{ formatVisitedAt(entry.visitedAt) }}</span>
            </span>
          </router-link>
        </li>
      </ul>
    </section>

    <section v-if="!recentEntries.length && !seriesCards.length" class="reading-page__empty" data-testid="reading-empty">
      <h2>{{ t('reading.empty') }}</h2>
      <p>{{ t('reading.emptyHint') }}</p>
      <div class="reading-page__empty-links">
        <router-link to="/">{{ t('menu.home') }}</router-link>
        <router-link to="/search">{{ t('discovery.search') }}</router-link>
      </div>
    </section>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { PageHeader } from '@/components/PageHeader'
import api from '@/api/api'
import { useDiscoveryStore } from '@/stores/discovery'
import { useReaderStore, type ReadingHistoryEntry } from '@/stores/reader'

interface ReadingSeriesCard {
  id: number
  name: string
  resolved: boolean
  total: number
  completed: number
  continueId: number | null
  continueTitle: string
}

const RECENT_LIMIT = 20
const SERIES_LIMIT = 3

export default defineComponent({
  name: 'Reading',
  components: { PageHeader },
  setup() {
    const { t } = useI18n()
    const readerStore = useReaderStore()
    const discoveryStore = useDiscoveryStore()
    const seriesCards = ref<ReadingSeriesCard[]>([])
    const seriesError = ref(false)
    const confirmingClear = ref(false)
    const clearedNotice = ref(false)

    const recentEntries = computed<ReadingHistoryEntry[]>(() => readerStore.historyEntries.slice(0, RECENT_LIMIT))
    const progressionEntries = computed(() => Object.entries(readerStore.seriesProgress)
      .map(([key, entry]) => ({ id: Number(key), ...entry }))
      .filter((item) => Number.isInteger(item.id) && item.id > 0)
      .sort((left, right) => String(right.updatedAt || '').localeCompare(String(left.updatedAt || '')))
      .slice(0, SERIES_LIMIT))

    function formatVisitedAt(value: string): string {
      const date = new Date(value)
      if (Number.isNaN(date.getTime())) return ''
      const pad = (input: number) => String(input).padStart(2, '0')
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
    }

    async function loadSeries(): Promise<void> {
      const entries = progressionEntries.value
      if (!entries.length) {
        seriesCards.value = []
        seriesError.value = false
        return
      }
      seriesError.value = false
      await discoveryStore.load()
      const cards: ReadingSeriesCard[] = []
      for (const entry of entries) {
        const known = discoveryStore.series.find((item) => Number(item.id) === entry.id)
        const card: ReadingSeriesCard = {
          id: entry.id,
          name: String(known?.seriesName || ''),
          resolved: false,
          total: 0,
          completed: 0,
          continueId: Number(entry.lastArticleId) > 0 ? Number(entry.lastArticleId) : null,
          continueTitle: ''
        }
        try {
          const { data } = await api.getSeriesDetail(entry.id)
          const payload = data?.data || {}
          const articles = Array.isArray(payload.articles) ? payload.articles : []
          const articleIds = articles
            .map((item: any) => Number(item?.id))
            .filter((id: number) => Number.isInteger(id) && id > 0)
          if (articleIds.length) {
            const summary = readerStore.seriesSummary(entry.id, articleIds)
            card.resolved = true
            card.total = summary.total
            card.completed = summary.completedCount
            card.continueId = summary.continueArticleId || card.continueId
            card.continueTitle = String(
              articles.find((item: any) => Number(item?.id) === Number(card.continueId))?.articleTitle || ''
            )
            card.name = card.name || String(payload.series?.seriesName || '')
          }
        } catch {
          seriesError.value = true
        }
        cards.push(card)
      }
      seriesCards.value = cards
    }

    function confirmClear(): void {
      readerStore.clearHistory()
      confirmingClear.value = false
      clearedNotice.value = true
    }

    watch(() => readerStore.seriesProgress, () => void loadSeries(), { immediate: true, deep: true })

    return {
      t,
      seriesCards,
      seriesError,
      confirmingClear,
      clearedNotice,
      recentEntries,
      formatVisitedAt,
      confirmClear
    }
  }
})
</script>

<style scoped>
.reading-page {
  display: grid;
  gap: 24px;
}

.reading-page__hero,
.reading-page__panel,
.reading-page__empty {
  padding: clamp(20px, 4vw, 42px);
  border: 1px solid var(--border-hairline);
  border-radius: 18px;
  background: color-mix(in srgb, var(--surface-solid) 86%, transparent);
  box-shadow: inset 0 1px 0 var(--glass-edge), var(--shadow-card);
}

.reading-page__eyebrow {
  margin: 0 0 8px;
  color: var(--color-ob);
  font-size: 0.75rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.reading-page__hero h1,
.reading-page__hero-title {
  margin: 0 0 8px;
  font-family: var(--font-display);
  font-size: clamp(2rem, 5vw, 3.4rem);
}

.reading-page__hero > p:not(.reading-page__eyebrow) {
  margin: 0;
  color: var(--text-dim);
}

.reading-page__panel-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.reading-page__panel-head h2 {
  margin: 0;
  font-size: 1.15rem;
}

.reading-page__panel-head p {
  margin: 4px 0 0;
  color: var(--text-ob-dim);
  font-size: 0.8rem;
}

.reading-page__series {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.reading-page__series-card {
  display: grid;
  gap: 8px;
  min-height: 140px;
  padding: 17px;
  border: 1px solid var(--border-hairline);
  border-radius: 13px;
}

.reading-page__series-card strong {
  align-self: center;
  font-size: 1.1rem;
}

.reading-page__series-progress,
.reading-page__meta {
  color: var(--text-ob-dim);
  font-size: 0.78rem;
}

.reading-page__series-action {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
  color: var(--color-ob);
  font-size: 0.82rem;
  text-decoration: none;
}

.reading-page__series-action em {
  color: var(--text-dim);
  font-style: normal;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reading-page__list {
  display: grid;
  gap: 10px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.reading-page__list a {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px 18px;
  padding: 14px 18px;
  border: 1px solid var(--border-hairline);
  border-radius: 12px;
  color: inherit;
  text-decoration: none;
}

.reading-page__list a:hover {
  border-color: var(--color-ob);
}

.reading-page__title {
  font-weight: 700;
}

.reading-page__meta {
  display: flex;
  gap: 12px;
  justify-self: end;
  align-self: center;
  white-space: nowrap;
}

.reading-page__clear {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-size: 0.78rem;
}

.reading-page__clear-hint {
  color: var(--text-dim);
}

.reading-page__clear button {
  padding: 5px 10px;
  border: 1px solid var(--border-hairline);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.reading-page__clear button:hover {
  border-color: var(--color-ob);
  color: var(--color-ob);
}

.reading-page__status {
  margin: 0;
  padding: 14px 18px;
  border: 1px solid var(--border-hairline);
  border-radius: 12px;
  color: var(--text-dim);
  font-size: 0.85rem;
}

.reading-page__status.is-accent {
  color: var(--color-ob);
}

.reading-page__status.is-error {
  color: var(--color-danger, #d9584a);
}

.reading-page__empty {
  text-align: center;
}

.reading-page__empty h2 {
  margin: 0 0 6px;
  font-size: 1.2rem;
}

.reading-page__empty p {
  margin: 0 0 16px;
  color: var(--text-dim);
}

.reading-page__empty-links {
  display: flex;
  justify-content: center;
  gap: 10px;
  flex-wrap: wrap;
}

.reading-page__empty-links a {
  padding: 6px 12px;
  border: 1px solid var(--border-hairline);
  border-radius: 999px;
  color: inherit;
  text-decoration: none;
}

.reading-page__empty-links a:hover {
  border-color: var(--color-ob);
  color: var(--color-ob);
}

@media (max-width: 640px) {
  .reading-page__panel-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .reading-page__list a {
    grid-template-columns: 1fr;
  }

  .reading-page__meta {
    justify-self: start;
  }
}
</style>
