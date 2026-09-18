<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('contentPerformance.title')" :description="t('contentPerformance.description')">
      <template #actions>
        <a-radio-group v-model="range" type="button" :disabled="loading" @change="reload">
          <a-radio value="7d">{{ t('contentPerformance.range.days7') }}</a-radio>
          <a-radio value="30d">{{ t('contentPerformance.range.days30') }}</a-radio>
          <a-radio value="90d">{{ t('contentPerformance.range.days90') }}</a-radio>
          <a-radio value="12m">{{ t('contentPerformance.range.months12') }}</a-radio>
        </a-radio-group>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <div class="content-performance-privacy">{{ t('contentPerformance.privacy') }}</div>
    <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('contentPerformance.loadFailed')" @retry="load" />

    <div class="admin-stat-grid">
      <AdminStatCard
        v-for="stat in stats"
        :key="stat.label"
        :label="stat.label"
        :value="stat.value"
        :caption="stat.caption"
        :icon="stat.icon"
        :tone="stat.tone" />
    </div>

    <a-card class="admin-panel" :bordered="false" :title="t('contentPerformance.trend.title')">
      <template #extra>
        <span class="admin-muted-cell">{{ rangeLabel }} · {{ analytics.unit === 'month' ? t('contentPerformance.unit.month') : t('contentPerformance.unit.day') }}</span>
      </template>
      <div v-if="loading && !analytics.trend.length" class="admin-skeleton content-performance-chart-skeleton" />
      <AdminEChart v-else-if="analytics.trend.length" :option="trendOption" height="330px" />
      <AdminEmptyState
        v-else
        :title="t('contentPerformance.trend.emptyTitle')"
        :description="t('contentPerformance.trend.emptyDescription')" />
    </a-card>

    <a-card class="admin-panel content-performance-ranking" :bordered="false" :title="t('contentPerformance.ranking.title')">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-select v-model="sort" style="width: 180px" @change="reload">
            <a-option value="views">{{ t('contentPerformance.table.views') }}</a-option>
            <a-option value="uniqueReaders">{{ t('contentPerformance.table.uniqueReaders') }}</a-option>
            <a-option value="avgActiveMs">{{ t('contentPerformance.table.avgActiveTime') }}</a-option>
            <a-option value="completionRate">{{ t('contentPerformance.table.completionRate') }}</a-option>
          </a-select>
        </div>
        <span class="admin-toolbar-caption">{{ t('contentPerformance.ranking.total', { total }) }}</span>
      </div>

      <div class="admin-table-shell">
        <a-table
          :data="articles"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="articleId"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #article="{ record }">
            <div class="content-performance-article">
              <img v-if="isHttpUrl(record.articleCover)" :src="String(record.articleCover)" alt="" />
              <span v-else class="content-performance-article__fallback"><IconBook /></span>
              <div>
                <strong :title="record.articleTitle">{{ record.articleTitle || t('contentPerformance.untitled') }}</strong>
                <small>{{ record.categoryName || t('contentPerformance.uncategorized') }}</small>
              </div>
            </div>
          </template>
          <template #views="{ record }">{{ formatNumber(record.views) }}</template>
          <template #uniqueReaders="{ record }">{{ formatNumber(record.uniqueReaders) }}</template>
          <template #avgActiveMs="{ record }">{{ formatDuration(record.avgActiveMs) }}</template>
          <template #completionRate="{ record }">{{ formatPercent(record.completionRate) }}</template>
          <template #effectiveSessions="{ record }">{{ formatNumber(record.effectiveSessions) }}</template>
          <template #actions="{ record }">
            <a-button type="text" size="small" @click="openDetail(record.articleId)">{{ t('contentPerformance.table.detail') }}</a-button>
          </template>
          <template #empty>{{ t('contentPerformance.empty') }}</template>
        </a-table>
      </div>
    </a-card>

    <a-drawer v-model:visible="detailVisible" width="720px" :footer="false" :title="t('contentPerformance.detail.title')">
      <AdminErrorState v-if="detailError" :error="detailError" :title="t('contentPerformance.detail.loadFailed')" @retry="reloadDetail" />
      <template v-else-if="detail">
        <div class="content-performance-detail-head">
          <img v-if="isHttpUrl(detail.articleCover)" :src="detail.articleCover" alt="" />
          <div>
            <h3>{{ detail.articleTitle || t('contentPerformance.untitled') }}</h3>
            <span>{{ detail.categoryName || t('contentPerformance.uncategorized') }} · {{ formatDateTime(detail.createTime) }}</span>
          </div>
        </div>
        <div class="content-performance-detail-stats">
          <div><span>{{ t('contentPerformance.table.views') }}</span><strong>{{ formatNumber(detail.overview.views) }}</strong></div>
          <div><span>{{ t('contentPerformance.table.uniqueReaders') }}</span><strong>{{ formatNumber(detail.overview.uniqueReaders) }}</strong></div>
          <div><span>{{ t('contentPerformance.table.avgActiveTime') }}</span><strong>{{ formatDuration(detail.overview.avgActiveMs) }}</strong></div>
          <div><span>{{ t('contentPerformance.table.completionRate') }}</span><strong>{{ formatPercent(detail.overview.completionRate) }}</strong></div>
        </div>
        <AdminEChart v-if="detail.trend.length" :option="detailTrendOption" height="300px" />
        <AdminEmptyState
          v-else
          :title="t('contentPerformance.trend.emptyTitle')"
          :description="t('contentPerformance.trend.emptyDescription')" />
      </template>
      <div v-else class="admin-skeleton content-performance-chart-skeleton" />
    </a-drawer>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { IconBook, IconCheckCircle, IconClockCircle, IconEye, IconRefresh, IconUserGroup } from '@arco-design/web-vue/es/icon'
import type { AdminContentAnalytics, ContentAnalyticsOverview, ContentAnalyticsRange, ContentAnalyticsTrend, ContentArticleAnalyticsDetail, ContentArticlePerformance } from '@stellar-beacon/api-contract'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatCard from '@/components/AdminStatCard.vue'
import { apiErrorMessage, getAdminContentAnalytics, getAdminContentArticleAnalytics, listAdminContentArticles } from '@/api/http'
import { t } from '@/i18n'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor, verticalFade } from '@/utils/chart-theme'
import { formatDateTime, formatNumber, isHttpUrl } from '@/utils/format'

function emptyOverview(): ContentAnalyticsOverview {
  return { views: 0, uniqueReaders: 0, effectiveSessions: 0, avgActiveMs: 0, completionRate: 0 }
}

function emptyAnalytics(): AdminContentAnalytics {
  return { range: '7d', unit: 'day', overview: emptyOverview(), trend: [], generatedAt: '' }
}

export default defineComponent({
  name: 'ContentPerformanceView',
  components: { AdminEmptyState, AdminErrorState, AdminEChart, AdminPageHeader, AdminStatCard, IconBook, IconRefresh },
  setup() {
    const themeStore = useThemeStore()
    const range = ref<ContentAnalyticsRange>('7d')
    const sort = ref('views')
    const analytics = reactive<AdminContentAnalytics>(emptyAnalytics())
    const articles = ref<ContentArticlePerformance[]>([])
    const total = ref(0)
    const current = ref(1)
    const pageSize = ref(20)
    const loading = ref(false)
    const errorMessage = ref('')
    const detailVisible = ref(false)
    const detailLoading = ref(false)
    const detailError = ref('')
    const detail = ref<ContentArticleAnalyticsDetail | null>(null)
    const detailArticleID = ref(0)

    const stats = computed(() => [
      { label: t('contentPerformance.stats.views'), value: formatNumber(analytics.overview.views), caption: t('contentPerformance.stats.viewsCaption'), icon: IconEye, tone: 'blue' as const },
      { label: t('contentPerformance.stats.uniqueReaders'), value: formatNumber(analytics.overview.uniqueReaders), caption: t('contentPerformance.stats.uniqueReadersCaption'), icon: IconUserGroup, tone: 'violet' as const },
      { label: t('contentPerformance.stats.avgActiveTime'), value: formatDuration(analytics.overview.avgActiveMs), caption: t('contentPerformance.stats.avgActiveTimeCaption'), icon: IconClockCircle, tone: 'warm' as const },
      { label: t('contentPerformance.stats.completionRate'), value: formatPercent(analytics.overview.completionRate), caption: t('contentPerformance.stats.completionRateCaption'), icon: IconCheckCircle, tone: 'green' as const }
    ])

    const columns = computed(() => [
      { title: t('contentPerformance.table.article'), dataIndex: 'articleTitle', slotName: 'article', width: 320 },
      { title: t('contentPerformance.table.views'), dataIndex: 'views', slotName: 'views', width: 110 },
      { title: t('contentPerformance.table.uniqueReaders'), dataIndex: 'uniqueReaders', slotName: 'uniqueReaders', width: 130 },
      { title: t('contentPerformance.table.avgActiveTime'), dataIndex: 'avgActiveMs', slotName: 'avgActiveMs', width: 150 },
      { title: t('contentPerformance.table.completionRate'), dataIndex: 'completionRate', slotName: 'completionRate', width: 120 },
      { title: t('contentPerformance.table.effectiveSessions'), dataIndex: 'effectiveSessions', slotName: 'effectiveSessions', width: 120 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 120 }
    ])

    const pagination = computed(() => ({
      current: current.value,
      pageSize: pageSize.value,
      total: total.value,
      showTotal: true,
      showPageSize: true
    }))

    const rangeLabel = computed(() => t(`contentPerformance.range.${range.value === '7d' ? 'days7' : range.value === '30d' ? 'days30' : range.value === '90d' ? 'days90' : 'months12'}`))
    const trendOption = computed(() => trendChart(analytics.trend, analytics.unit))
    const detailTrendOption = computed(() => trendChart(detail.value?.trend || [], analytics.unit))

    const load = async () => {
      loading.value = true
      errorMessage.value = ''
      try {
        const [summary, page] = await Promise.all([
          getAdminContentAnalytics(range.value),
          listAdminContentArticles(range.value, sort.value, current.value, pageSize.value)
        ])
        Object.assign(analytics, summary)
        articles.value = page.items
        total.value = page.total
      } catch (error) {
        errorMessage.value = apiErrorMessage(error, t('contentPerformance.loadFailed'))
      } finally {
        loading.value = false
      }
    }

    const reload = () => {
      current.value = 1
      void load()
    }

    const changePage = (page: number) => {
      current.value = page
      void load()
    }

    const changePageSize = (size: number) => {
      pageSize.value = size
      current.value = 1
      void load()
    }

    const openDetail = async (articleID: number) => {
      detailArticleID.value = articleID
      detailVisible.value = true
      detail.value = null
      detailError.value = ''
      await loadDetail(articleID)
    }

    const reloadDetail = () => {
      if (detailArticleID.value) void loadDetail(detailArticleID.value)
    }

    const loadDetail = async (articleID: number) => {
      detailLoading.value = true
      try {
        detail.value = await getAdminContentArticleAnalytics(articleID, range.value)
      } catch (error) {
        detailError.value = apiErrorMessage(error, t('contentPerformance.detail.loadFailed'))
      } finally {
        detailLoading.value = false
      }
    }

    function trendChart(rows: ContentAnalyticsTrend[], unit: 'day' | 'month'): Record<string, unknown> {
      const first = chartSeriesColor(themeStore.theme, 0)
      const second = chartSeriesColor(themeStore.theme, 1)
      const third = chartSeriesColor(themeStore.theme, 2)
      return {
        tooltip: { trigger: 'axis' },
        legend: { top: 0, right: 0 },
        grid: { left: 4, right: 20, top: 36, bottom: 2, containLabel: true },
        xAxis: { type: 'category', boundaryGap: false, data: rows.map((item) => unit === 'month' ? item.period : item.period.slice(5)) },
        yAxis: [
          { type: 'value', minInterval: 1 },
          { type: 'value', min: 0, max: 100, axisLabel: { formatter: '{value}%' } }
        ],
        series: [
          { name: t('contentPerformance.chart.views'), type: 'line', smooth: true, showSymbol: false, lineStyle: { color: first, width: 2.4 }, itemStyle: { color: first }, areaStyle: { color: verticalFade(first, 0.2, 0.01) }, data: rows.map((item) => item.views) },
          { name: t('contentPerformance.chart.uniqueReaders'), type: 'line', smooth: true, showSymbol: false, lineStyle: { color: second, width: 2.2 }, itemStyle: { color: second }, data: rows.map((item) => item.uniqueReaders) },
          { name: t('contentPerformance.chart.completionRate'), type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, lineStyle: { color: third, width: 2 }, itemStyle: { color: third }, data: rows.map((item) => item.completionRate) }
        ]
      }
    }

    onMounted(() => void load())

    return {
      t,
      range,
      sort,
      analytics,
      articles,
      total,
      loading,
      errorMessage,
      detailVisible,
      detailLoading,
      detailError,
      detail,
      stats,
      columns,
      pagination,
      rangeLabel,
      trendOption,
      detailTrendOption,
      load,
      reload,
      changePage,
      changePageSize,
      openDetail,
      reloadDetail,
      formatNumber,
      formatPercent,
      formatDuration,
      formatDateTime,
      isHttpUrl
    }
  }
})

function formatPercent(value: unknown): string {
  return `${Number(value || 0).toFixed(2)}%`
}

function formatDuration(value: unknown): string {
  const ms = Number(value || 0)
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  const seconds = Math.round(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remain = seconds % 60
  return remain ? `${minutes}m ${remain}s` : `${minutes}m`
}
</script>

<style scoped>
.content-performance-privacy {
  margin: -4px 0 16px;
  color: var(--admin-muted);
  font-size: 12px;
}

.content-performance-chart-skeleton {
  height: 300px;
}

.content-performance-ranking {
  margin-top: 16px;
}

.content-performance-article {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.content-performance-article img,
.content-performance-article__fallback,
.content-performance-detail-head img {
  width: 48px;
  height: 36px;
  border-radius: 6px;
  object-fit: cover;
  background: var(--admin-panel-soft);
  color: var(--admin-muted);
}

.content-performance-article__fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.content-performance-article div {
  min-width: 0;
}

.content-performance-article strong,
.content-performance-article small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.content-performance-article small {
  color: var(--admin-muted);
  font-size: 12px;
}

.content-performance-detail-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
}

.content-performance-detail-head h3 {
  margin: 0 0 4px;
}

.content-performance-detail-head span {
  color: var(--admin-muted);
  font-size: 12px;
}

.content-performance-detail-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 18px;
}

.content-performance-detail-stats div {
  padding: 12px;
  border-radius: 8px;
  background: var(--admin-panel-soft);
}

.content-performance-detail-stats span,
.content-performance-detail-stats strong {
  display: block;
}

.content-performance-detail-stats span {
  color: var(--admin-muted);
  font-size: 12px;
}

.content-performance-detail-stats strong {
  margin-top: 4px;
  font-size: 20px;
}

@media (max-width: 800px) {
  .content-performance-detail-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
