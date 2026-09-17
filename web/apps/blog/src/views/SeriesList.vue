<template>
  <div class="series-page">
    <PageHeader :title="t('series.pageTitle')" />
    <p v-if="loading" class="series-hint">{{ t('reactions.loading') }}</p>
    <p v-else-if="series.length === 0" class="series-hint">{{ t('series.empty') }}</p>
    <div v-else class="series-grid">
      <router-link v-for="item in series" :key="item.id" class="series-card" :to="'/series/' + item.id">
        <strong>{{ item.seriesName }}</strong>
        <span class="series-card__count">{{ t('series.articleCount', { count: item.articleCount }) }}</span>
        <p v-if="item.seriesDesc">{{ item.seriesDesc }}</p>
      </router-link>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { PageHeader } from '@/components/PageHeader'
import api from '@/api/api'

export default defineComponent({
  name: 'SeriesList',
  components: { PageHeader },
  setup() {
    const { t } = useI18n()
    const loading = ref(true)
    const series = ref<any[]>([])

    onMounted(() => {
      api
        .getSeriesList()
        .then(({ data }: any) => {
          series.value = Array.isArray(data?.data) ? data.data : []
        })
        .catch(() => {
          series.value = []
        })
        .finally(() => {
          loading.value = false
        })
    })

    return { t, loading, series }
  }
})
</script>

<style lang="scss" scoped>
.series-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 2rem 1rem 4rem;
}
.series-hint { opacity: .7; }
.series-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 1rem; }
.series-card {
  display: flex;
  flex-direction: column;
  gap: .35rem;
  padding: 1rem 1.15rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 24%, transparent);
  border-radius: 12px;
  color: inherit;
  text-decoration: none;
  transition: border-color .2s ease, transform .2s ease;
}
.series-card:hover { border-color: var(--color-ob); transform: translateY(-2px); }
.series-card__count { font-size: .8rem; opacity: .65; }
.series-card p { margin: .25rem 0 0; font-size: .85rem; opacity: .75; }
</style>
