<template>
  <div class="series-detail">
    <PageHeader :title="series?.seriesName || t('series.pageTitle')" />
    <p v-if="series?.seriesDesc" class="series-detail__desc">{{ series.seriesDesc }}</p>
    <p v-if="loading" class="series-hint">{{ t('reactions.loading') }}</p>
    <p v-else-if="articles.length === 0" class="series-hint">{{ t('series.emptyArticles') }}</p>
    <template v-else>
      <section class="series-overview" data-testid="series-progress">
        <div class="series-overview__head">
          <div>
            <span>{{ t('series.readProgress', { completed: progress.completedCount, total: progress.total }) }}</span>
            <strong>{{ progress.percent }}%</strong>
          </div>
          <router-link
            v-if="continueArticle"
            class="series-overview__continue"
            :to="`/articles/${continueArticle.id}`"
            data-testid="series-continue">
            <small>{{ progress.completedCount || progress.lastArticleId ? t('series.continueReading') : t('series.startReading') }}</small>
            <span>{{ continueArticle.articleTitle }}</span>
          </router-link>
        </div>
        <div
          class="series-overview__bar"
          role="progressbar"
          :aria-label="t('series.readProgress', { completed: progress.completedCount, total: progress.total })"
          :aria-valuenow="progress.percent"
          aria-valuemin="0"
          aria-valuemax="100">
          <i :style="{ width: `${progress.percent}%` }" />
        </div>
      </section>

      <ol class="series-detail__list">
        <li
          v-for="(article, index) in articles"
          :key="article.id"
          :class="{
            'is-completed': completedArticleIds.has(Number(article.id)),
            'is-current': progress.lastArticleId === Number(article.id)
          }">
          <router-link
            :to="'/articles/' + article.id"
            :aria-current="progress.lastArticleId === Number(article.id) ? 'step' : undefined">
            <span class="series-detail__index">{{ index + 1 }}</span>
            <span class="series-detail__title">{{ article.articleTitle }}</span>
            <span class="series-detail__badges">
              <em v-if="completedArticleIds.has(Number(article.id))">{{ t('series.read') }}</em>
              <em v-if="progress.lastArticleId === Number(article.id)" class="is-recent">{{ t('series.recent') }}</em>
            </span>
          </router-link>
        </li>
      </ol>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { PageHeader } from '@/components/PageHeader'
import api from '@/api/api'
import { useReaderStore } from '@/stores/reader'

export default defineComponent({
  name: 'SeriesDetail',
  components: { PageHeader },
  setup() {
    const { t } = useI18n()
    const route = useRoute()
    const readerStore = useReaderStore()
    const loading = ref(true)
    const series = ref<any>(null)
    const articles = ref<any[]>([])
    const seriesId = computed(() => Number(route.params.seriesId || 0))
    const articleIds = computed(() => articles.value.map((article) => Number(article.id)).filter((id) => Number.isInteger(id) && id > 0))
    const progress = computed(() => readerStore.seriesSummary(seriesId.value, articleIds.value))
    const completedArticleIds = computed(() => new Set(progress.value.completedArticleIds))
    const continueArticle = computed(() => (
      articles.value.find((article) => Number(article.id) === progress.value.continueArticleId) || null
    ))

    const load = () => {
      loading.value = true
      api
        .getSeriesDetail(seriesId.value)
        .then(({ data }: any) => {
          series.value = data?.data?.series || null
          articles.value = Array.isArray(data?.data?.articles) ? data.data.articles : []
        })
        .catch(() => {
          series.value = null
          articles.value = []
        })
        .finally(() => {
          loading.value = false
        })
    }

    onMounted(load)
    watch(() => route.params.seriesId, load)

    return { t, loading, series, articles, progress, completedArticleIds, continueArticle }
  }
})
</script>

<style lang="scss" scoped>
.series-detail {
  max-width: 820px;
  margin: 0 auto;
  padding: 2rem 1rem 4rem;
}
.series-hint { opacity: .7; }
.series-detail__desc { opacity: .78; margin: 0 0 1.5rem; }
.series-overview {
  margin-bottom: 1.25rem;
  padding: 1rem 1.1rem;
  border: 1px solid var(--border-hairline);
  border-radius: 14px;
  background: color-mix(in srgb, var(--surface-solid) 78%, transparent);
}
.series-overview__head { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
.series-overview__head > div { display: grid; gap: .3rem; }
.series-overview__head span,
.series-overview__continue small { color: var(--text-ob-dim); font-size: .75rem; }
.series-overview__head strong { font-size: 1.6rem; line-height: 1; }
.series-overview__continue {
  display: grid;
  gap: .25rem;
  min-width: 0;
  max-width: 58%;
  padding: .7rem .85rem;
  border: 1px solid color-mix(in srgb, var(--color-ob) 38%, transparent);
  border-radius: 10px;
  color: inherit;
  text-decoration: none;
}
.series-overview__continue span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 600; }
.series-overview__bar {
  height: 6px;
  margin-top: 1rem;
  overflow: hidden;
  border-radius: 999px;
  background: color-mix(in srgb, var(--text-ob-dim) 18%, transparent);
}
.series-overview__bar i { display: block; height: 100%; border-radius: inherit; background: var(--brand-gradient); transition: width .25s ease; }
.series-detail__list { display: flex; flex-direction: column; gap: .6rem; padding: 0; margin: 0; list-style: none; counter-reset: none; }
.series-detail__list a {
  display: flex;
  align-items: center;
  gap: .75rem;
  padding: .7rem .9rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 22%, transparent);
  border-radius: 10px;
  color: inherit;
  text-decoration: none;
}
.series-detail__list a:hover { border-color: var(--color-ob); }
.series-detail__list li.is-current a { border-color: color-mix(in srgb, var(--color-ob) 48%, transparent); background: color-mix(in srgb, var(--color-ob) 7%, transparent); }
.series-detail__index { font-variant-numeric: tabular-nums; opacity: .6; min-width: 1.5rem; }
.series-detail__title { flex: 1; min-width: 0; font-weight: 600; }
.series-detail__badges { display: flex; flex: none; align-items: center; gap: .35rem; }
.series-detail__badges em {
  padding: .2rem .45rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-ob) 12%, transparent);
  color: var(--color-ob);
  font-size: .68rem;
  font-style: normal;
}
.series-detail__badges em.is-recent { background: color-mix(in srgb, var(--text-ob-dim) 14%, transparent); color: var(--text-normal); }
@media (max-width: 640px) {
  .series-overview__head { flex-direction: column; }
  .series-overview__continue { max-width: 100%; width: 100%; }
  .series-detail__badges { flex-direction: column; align-items: flex-end; }
}
</style>