<template>
  <div class="series-detail">
    <PageHeader :title="series?.seriesName || t('series.pageTitle')" />
    <p v-if="series?.seriesDesc" class="series-detail__desc">{{ series.seriesDesc }}</p>
    <p v-if="loading" class="series-hint">{{ t('reactions.loading') }}</p>
    <p v-else-if="articles.length === 0" class="series-hint">{{ t('series.emptyArticles') }}</p>
    <ol v-else class="series-detail__list">
      <li v-for="(article, index) in articles" :key="article.id">
        <router-link :to="'/articles/' + article.id">
          <span class="series-detail__index">{{ index + 1 }}</span>
          <span class="series-detail__title">{{ article.articleTitle }}</span>
        </router-link>
      </li>
    </ol>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { PageHeader } from '@/components/PageHeader'
import api from '@/api/api'

export default defineComponent({
  name: 'SeriesDetail',
  components: { PageHeader },
  setup() {
    const { t } = useI18n()
    const route = useRoute()
    const loading = ref(true)
    const series = ref<any>(null)
    const articles = ref<any[]>([])

    const load = () => {
      loading.value = true
      api
        .getSeriesDetail(route.params.seriesId)
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

    return { t, loading, series, articles }
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
.series-detail__list { display: flex; flex-direction: column; gap: .6rem; padding: 0; margin: 0; list-style: none; counter-reset: none; }
.series-detail__list a {
  display: flex;
  align-items: baseline;
  gap: .75rem;
  padding: .7rem .9rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 22%, transparent);
  border-radius: 10px;
  color: inherit;
  text-decoration: none;
}
.series-detail__list a:hover { border-color: var(--color-ob); }
.series-detail__index { font-variant-numeric: tabular-nums; opacity: .6; min-width: 1.5rem; }
.series-detail__title { font-weight: 600; }
</style>
