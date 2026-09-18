<template>
  <section class="admin-page" data-testid="content-performance-page">
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
        :tone="stat.tone"
        :data-testid="stat.testId" />
    </div>

    <a-card class="admin-panel" :bordered="false" :title="t('contentPerformance.trend.title')" data-testid="content-performance-trend">
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

    <a-card class="admin-panel content-performance-continuation" :bordered="false" :title="t('contentPerformance.continuation.title')" data-testid="content-performance-continuation">
      <div class="content-performance-continuation__summary">
        <div data-testid="content-performance-continuation-series">
          <span>{{ t('contentPerformance.continuation.series') }}</span>
          <strong>{{ formatPercent(analytics.overview.continuation.seriesClickRate) }}</strong>
          <small>{{ t('contentPerformance.continuation.detail', { clicks: formatNumber(analytics.overview.continuation.seriesClicks), impressions: formatNumber(analytics.overview.continuation.seriesImpressions) }) }}</small>
        </div>
        <div data-testid="content-performance-continuation-related">
          <span>{{ t('contentPerformance.continuation.related') }}</span>
          <strong>{{ formatPercent(analytics.overview.continuation.relatedClickRate) }}</strong>
          <small>{{ t('contentPerformance.continuation.detail', { clicks: formatNumber(analytics.overview.continuation.relatedClicks), impressions: formatNumber(analytics.overview.continuation.relatedImpressions) }) }}</small>
        </div>
      </div>
      <AdminEChart v-if="analytics.trend.length" :option="continuationTrendOption" height="280px" />
      <AdminEmptyState
        v-else
        :title="t('contentPerformance.continuation.emptyTitle')"
        :description="t('contentPerformance.continuation.emptyDescription')" />
    </a-card>

    <a-card class="admin-panel content-performance-targets" :bordered="false" :title="t('contentPerformance.targets.title')" data-testid="content-performance-targets">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-select v-model="targetSort" style="width: 180px" @change="reloadTargets">
            <a-option value="clicks">{{ t('contentPerformance.targets.clicks') }}</a-option>
            <a-option value="clickRate">{{ t('contentPerformance.targets.clickRate') }}</a-option>
          </a-select>
        </div>
        <span class="admin-toolbar-caption">{{ t('contentPerformance.targets.total', { total: targetTotal }) }}</span>
      </div>
      <div class="admin-table-shell">
        <a-table
          :data="targets"
          :columns="targetColumns"
          :loading="loading"
          :pagination="targetPagination"
          row-key="rowKey"
          @page-change="changeTargetPage"
          @page-size-change="changeTargetPageSize">
          <template #targetSource="{ record }">
            <span class="admin-title-cell" :title="record.sourceArticleTitle">{{ record.sourceArticleTitle || `#${record.sourceArticleId}` }}</span>
          </template>
          <template #targetTarget="{ record }">
            <div class="content-performance-target">
              <a-tag size="small">{{ record.targetType === 'series' ? t('contentPerformance.targets.series') : t('contentPerformance.targets.article') }}</a-tag>
              <span :title="targetLabel(record)">{{ targetLabel(record) }}</span>
            </div>
          </template>
          <template #targetPlacement="{ record }">
            <span class="admin-muted-cell">{{ placementLabel(record.placement) }}<template v-if="record.position"> · {{ record.position }}</template></span>
          </template>
          <template #targetClicks="{ record }">{{ formatNumber(record.clicks) }}</template>
          <template #targetClickRate="{ record }">
            <span :title="t('contentPerformance.targets.rateDetail', { clicks: formatNumber(record.clicks), impressions: formatNumber(record.moduleImpressions) })">
              {{ formatPercent(record.clickRate) }}
            </span>
          </template>
          <template #empty>{{ t('contentPerformance.targets.empty') }}</template>
        </a-table>
      </div>
    </a-card>
    <a-card class="admin-panel content-performance-ranking" :bordered="false" :title="t('contentPerformance.ranking.title')" data-testid="content-performance-ranking">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-select v-model="sort" style="width: 180px" @change="reload">
            <a-option value="views">{{ t('contentPerformance.table.views') }}</a-option>
            <a-option value="uniqueReaders">{{ t('contentPerformance.table.uniqueReaders') }}</a-option>
            <a-option value="avgActiveMs">{{ t('contentPerformance.table.avgActiveTime') }}</a-option>
            <a-option value="completionRate">{{ t('contentPerformance.table.completionRate') }}</a-option>
            <a-option value="continuationRate">{{ t('contentPerformance.table.continuationRate') }}</a-option>
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
          <template #continuationRate="{ record }">{{ formatPercent(record.continuation.continuationRate) }}</template>
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
        <div data-testid="content-performance-detail">
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
            <div>
              <span>{{ t('contentPerformance.continuation.series') }}</span>
              <strong>{{ formatPercent(detail.overview.continuation.seriesClickRate) }}</strong>
              <small>{{ t('contentPerformance.continuation.detail', { clicks: formatNumber(detail.overview.continuation.seriesClicks), impressions: formatNumber(detail.overview.continuation.seriesImpressions) }) }}</small>
            </div>
            <div>
              <span>{{ t('contentPerformance.continuation.related') }}</span>
              <strong>{{ formatPercent(detail.overview.continuation.relatedClickRate) }}</strong>
              <small>{{ t('contentPerformance.continuation.detail', { clicks: formatNumber(detail.overview.continuation.relatedClicks), impressions: formatNumber(detail.overview.continuation.relatedImpressions) }) }}</small>
            </div>
          </div>
          <AdminEChart v-if="detail.trend.length" :option="detailTrendOption" height="300px" />
          <AdminEChart v-if="detail.trend.length" :option="detailContinuationTrendOption" height="260px" />
              <div class="content-performance-detail-targets" data-testid="content-performance-detail-targets">
            <h4>{{ t('contentPerformance.targets.detailTitle') }}</h4>
            <a-table :data="detailTargets" :columns="detailTargetColumns" :pagination="false" row-key="rowKey">
              <template #targetTarget="{ record }">
                <div class="content-performance-target">
                  <a-tag size="small">{{ record.targetType === 'series' ? t('contentPerformance.targets.series') : t('contentPerformance.targets.article') }}</a-tag>
                  <span :title="targetLabel(record)">{{ targetLabel(record) }}</span>
                </div>
              </template>
              <template #targetPlacement="{ record }">
                <span class="admin-muted-cell">{{ placementLabel(record.placement) }}<template v-if="record.position"> · {{ record.position }}</template></span>
              </template>
              <template #targetClicks="{ record }">{{ formatNumber(record.clicks) }}</template>
              <template #targetClickRate="{ record }">{{ formatPercent(record.clickRate) }}</template>
              <template #empty>{{ t('contentPerformance.targets.empty') }}</template>
            </a-table>
          </div>
          <AdminEmptyState
            v-if="!detail.trend.length"
            :title="t('contentPerformance.trend.emptyTitle')"
            :description="t('contentPerformance.trend.emptyDescription')" />
        </div>
      </template>
      <div v-else class="admin-skeleton content-performance-chart-skeleton" />
    </a-drawer>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { IconBook, IconCheckCircle, IconClockCircle, IconEye, IconRefresh, IconUserGroup } from '@arco-design/web-vue/es/icon'
import type { AdminContentAnalytics, ContentAnalyticsOverview, ContentAnalyticsRange, ContentAnalyticsTrend, ContentArticleAnalyticsDetail, ContentArticlePerformance, ContentContinuationTarget, ContentContinuationMetrics } from '@stellar-beacon/api-contract'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatCard from '@/components/AdminStatCard.vue'
import { apiErrorMessage, getAdminContentAnalytics, getAdminContentArticleAnalytics, listAdminContentArticles, listAdminContinuationTargets } from '@/api/http'
import { t } from '@/i18n'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor, verticalFade } from '@/utils/chart-theme'
import { formatDateTime, formatNumber, isHttpUrl } from '@/utils/format'

function emptyContinuation(): ContentContinuationMetrics {
  return {
    seriesImpressions: 0,
    seriesClicks: 0,
    seriesClickRate: 0,
    relatedImpressions: 0,
    relatedClicks: 0,
    relatedClickRate: 0,
    continuationRate: 0
  }
}

function emptyOverview(): ContentAnalyticsOverview {
  return { views: 0, uniqueReaders: 0, effectiveSessions: 0, avgActiveMs: 0, completionRate: 0, continuation: emptyContinuation() }
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
    const targetSort = ref('clicks')
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
    const targets = ref<ContentContinuationTarget[]>([])
    const targetTotal = ref(0)
    const targetCurrent = ref(1)
    const targetPageSize = ref(10)
    const detailTargets = ref<ContentContinuationTarget[]>([])

    const stats = computed(() => [
      { testId: 'content-performance-views', label: t('contentPerformance.stats.views'), value: formatNumber(analytics.overview.views), caption: t('contentPerformance.stats.viewsCaption'), icon: IconEye, tone: 'blue' as const },
      { testId: 'content-performance-unique-readers', label: t('contentPerformance.stats.uniqueReaders'), value: formatNumber(analytics.overview.uniqueReaders), caption: t('contentPerformance.stats.uniqueReadersCaption'), icon: IconUserGroup, tone: 'violet' as const },
      { testId: 'content-performance-avg-active-time', label: t('contentPerformance.stats.avgActiveTime'), value: formatDuration(analytics.overview.avgActiveMs), caption: t('contentPerformance.stats.avgActiveTimeCaption'), icon: IconClockCircle, tone: 'warm' as const },
      { testId: 'content-performance-completion-rate', label: t('contentPerformance.stats.completionRate'), value: formatPercent(analytics.overview.completionRate), caption: t('contentPerformance.stats.completionRateCaption'), icon: IconCheckCircle, tone: 'green' as const }
    ])

    const columns = computed(() => [
      { title: t('contentPerformance.table.article'), dataIndex: 'articleTitle', slotName: 'article', width: 320 },
      { title: t('contentPerformance.table.views'), dataIndex: 'views', slotName: 'views', width: 110 },
      { title: t('contentPerformance.table.uniqueReaders'), dataIndex: 'uniqueReaders', slotName: 'uniqueReaders', width: 130 },
      { title: t('contentPerformance.table.avgActiveTime'), dataIndex: 'avgActiveMs', slotName: 'avgActiveMs', width: 150 },
      { title: t('contentPerformance.table.completionRate'), dataIndex: 'completionRate', slotName: 'completionRate', width: 120 },
      { title: t('contentPerformance.table.effectiveSessions'), dataIndex: 'effectiveSessions', slotName: 'effectiveSessions', width: 120 },
      { title: t('contentPerformance.table.continuationRate'), dataIndex: 'continuationRate', slotName: 'continuationRate', width: 130 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 120 }
    ])

    const pagination = computed(() => ({
      current: current.value,
      pageSize: pageSize.value,
      total: total.value,
      showTotal: true,
      showPageSize: true
    }))
    const targetPagination = computed(() => ({
      current: targetCurrent.value,
      pageSize: targetPageSize.value,
      total: targetTotal.value,
      showTotal: true,
      showPageSize: true
    }))

    const targetColumns = computed(() => [
      { title: t('contentPerformance.targets.source'), dataIndex: 'sourceArticleTitle', slotName: 'targetSource', width: 220 },
      { title: t('contentPerformance.targets.target'), dataIndex: 'targetTitle', slotName: 'targetTarget', width: 260 },
      { title: t('contentPerformance.targets.placement'), dataIndex: 'placement', slotName: 'targetPlacement', width: 150 },
      { title: t('contentPerformance.targets.clicks'), dataIndex: 'clicks', slotName: 'targetClicks', width: 90 },
      { title: t('contentPerformance.targets.clickRate'), dataIndex: 'clickRate', slotName: 'targetClickRate', width: 120 }
    ])
    const detailTargetColumns = computed(() => targetColumns.value.slice(1))
    const rangeLabel = computed(() => t(`contentPerformance.range.${range.value === '7d' ? 'days7' : range.value === '30d' ? 'days30' : range.value === '90d' ? 'days90' : 'months12'}`))
    const trendOption = computed(() => trendChart(analytics.trend, analytics.unit))
    const continuationTrendOption = computed(() => continuationTrendChart(analytics.trend, analytics.unit))
    const detailTrendOption = computed(() => trendChart(detail.value?.trend || [], analytics.unit))
    const detailContinuationTrendOption = computed(() => continuationTrendChart(detail.value?.trend || [], analytics.unit))

    const load = async () => {
      loading.value = true
      errorMessage.value = ''
      try {
        const [summary, page, targetPage] = await Promise.all([
          getAdminContentAnalytics(range.value),
          listAdminContentArticles(range.value, sort.value, current.value, pageSize.value),
          listAdminContinuationTargets(range.value, targetSort.value, targetCurrent.value, targetPageSize.value)
        ])
        Object.assign(analytics, summary)
        articles.value = page.items
        total.value = page.total
        targets.value = targetPage.items
        targetTotal.value = targetPage.total
      } catch (error) {
        errorMessage.value = apiErrorMessage(error, t('contentPerformance.loadFailed'))
      } finally {
        loading.value = false
      }
    }

    const reload = () => {
      current.value = 1
      targetCurrent.value = 1
      void load()
    }

    const reloadTargets = () => {
      targetCurrent.value = 1
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

    const changeTargetPage = (page: number) => {
      targetCurrent.value = page
      void load()
    }

    const changeTargetPageSize = (size: number) => {
      targetPageSize.value = size
      targetCurrent.value = 1
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
        const [articleDetail, targetPage] = await Promise.all([
          getAdminContentArticleAnalytics(articleID, range.value),
          listAdminContinuationTargets(range.value, 'clicks', 1, 20, articleID)
        ])
        detail.value = articleDetail
        detailTargets.value = targetPage.items
      } catch (error) {
        detailError.value = apiErrorMessage(error, t('contentPerformance.detail.loadFailed'))
      } finally {
        detailLoading.value = false
      }
    }

    function continuationTrendChart(rows: ContentAnalyticsTrend[], unit: 'day' | 'month'): Record<string, unknown> {
      const first = chartSeriesColor(themeStore.theme, 0)
      const second = chartSeriesColor(themeStore.theme, 1)
      return {
        tooltip: { trigger: 'axis' },
        legend: { top: 0, right: 0 },
        grid: { left: 4, right: 20, top: 36, bottom: 2, containLabel: true },
        xAxis: { type: 'category', boundaryGap: false, data: rows.map((item) => unit === 'month' ? item.period : item.period.slice(5)) },
        yAxis: { type: 'value', min: 0, max: 100, axisLabel: { formatter: '{value}%' } },
        series: [
          { name: t('contentPerformance.chart.seriesClickRate'), type: 'line', smooth: true, showSymbol: false, lineStyle: { color: first, width: 2.4 }, itemStyle: { color: first }, data: rows.map((item) => item.continuation.seriesClickRate) },
          { name: t('contentPerformance.chart.relatedClickRate'), type: 'line', smooth: true, showSymbol: false, lineStyle: { color: second, width: 2.2 }, itemStyle: { color: second }, data: rows.map((item) => item.continuation.relatedClickRate) }
        ]
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

    const placementLabel = (placement: string) => t(`contentPerformance.targets.placements.${placement}`)

    const targetLabel = (target: ContentContinuationTarget) => target.targetTitle || `#${target.targetId}`
    onMounted(() => void load())

    return {
      t,
      range,
      sort,
      analytics,
      articles,
      total,
      targets,
      targetTotal,
      targetSort,
      targetPagination,
      targetColumns,
      detailTargets,
      detailTargetColumns,
      placementLabel,
      targetLabel,
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
      continuationTrendOption,
      detailTrendOption,
      detailContinuationTrendOption,
      load,
      reload,
      changePage,
      changePageSize,
      changeTargetPage,
      changeTargetPageSize,
      reloadTargets,
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

.content-performance-continuation {
  margin-top: 16px;
}

.content-performance-continuation__summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 8px;
}

.content-performance-continuation__summary > div {
  display: grid;
  gap: 4px;
  padding: 14px 16px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-1);
}

.content-performance-continuation__summary span,
.content-performance-continuation__summary small {
  color: var(--color-text-3);
  font-size: 12px;
}

.content-performance-continuation__summary strong {
  font-size: 24px;
  line-height: 1.2;
}

.content-performance-targets {
  margin-top: 16px;
}

.content-performance-target {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.content-performance-target span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.content-performance-targets .admin-title-cell {
  display: inline-block;
  max-width: 100%;
}

.content-performance-detail-targets {
  margin-top: 22px;
}

.content-performance-detail-targets h4 {
  margin: 0 0 12px;
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

.content-performance-detail-stats small {
  color: var(--admin-muted);
  font-size: 11px;
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
  .content-performance-continuation__summary,
  .content-performance-detail-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 520px) {
  .content-performance-continuation__summary {
    grid-template-columns: 1fr;
  }
}
</style>
