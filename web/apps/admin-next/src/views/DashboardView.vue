<template>
  <section class="admin-page dashboard-page">
    <AdminPageHeader :title="t('dashboard.title')" :description="t('dashboard.description')">
      <template #actions>
        <a-radio-group v-model="range" type="button" :disabled="loading" @change="load">
          <a-radio value="7d">{{ t('dashboard.range.days7') }}</a-radio>
          <a-radio value="30d">{{ t('dashboard.range.days30') }}</a-radio>
          <a-radio value="12m">{{ t('dashboard.range.months12') }}</a-radio>
        </a-radio-group>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <AdminErrorState
      v-if="errorMessage"
      :error="errorMessage"
      :title="t('dashboard.loadFailed')"
      @retry="load" />

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

    <div class="dashboard-grid dashboard-grid-wide">
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.trend')">
        <template #extra>
          <span class="admin-muted-cell">{{ rangeLabel }} · {{ analytics.unit === 'month' ? t('dashboard.unit.month') : t('dashboard.unit.day') }}</span>
        </template>
        <div v-if="loading && !analytics.trend.length" class="admin-skeleton dashboard-chart-skeleton" />
        <AdminEChart v-else-if="analytics.trend.length" :option="trendOption" height="330px" />
        <AdminEmptyState
          v-else
          :icon="IconBarChart"
          :title="t('dashboard.empty.trend.title')"
          :description="t('dashboard.empty.trend.description')" />
      </a-card>

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.hotArticles')">
        <template #extra><span class="admin-muted-cell">{{ t('dashboard.panels.viewRank') }}</span></template>
        <div v-if="analytics.articleRank.length" class="admin-rank-list dashboard-rank-scroll">
          <div v-for="(article, index) in analytics.articleRank.slice(0, 10)" :key="article.id" class="admin-rank-item">
            <span class="admin-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="admin-rank-title" :title="article.title">{{ article.title || t('dashboard.untitledArticle') }}</span>
            <span class="admin-rank-metric">{{ formatNumber(article.views) }}</span>
          </div>
        </div>
        <AdminEmptyState
          v-else
          :icon="IconBook"
          :title="t('dashboard.empty.rank.title')"
          :description="t('dashboard.empty.rank.description')" />
      </a-card>
    </div>

    <div class="dashboard-grid dashboard-grid-three">
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.regions')">
        <template #extra>
          <a-radio-group v-model="areaType" type="button" size="mini" :disabled="loading" @change="load">
            <a-radio value="users">{{ t('dashboard.area.users') }}</a-radio>
            <a-radio value="visitors">{{ t('dashboard.area.visitors') }}</a-radio>
          </a-radio-group>
        </template>
        <AdminEChart v-if="regionMapData.length" :option="regionOption" height="300px" />
        <div v-else class="admin-chart-empty">
          <AdminEmptyState
            :icon="IconLocation"
            :title="t('dashboard.empty.regions.title')"
            :description="t('dashboard.empty.regions.description')" />
        </div>
        <div v-if="analytics.regions.length" class="dashboard-region-list">
          <div v-for="region in analytics.regions.slice(0, 8)" :key="`${region.label}-${region.code}`" class="dashboard-region-item">
            <span :title="region.name">{{ region.label || region.name }}</span>
            <strong>{{ formatNumber(region.value) }}</strong>
          </div>
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.categories')">
        <AdminEChart v-if="analytics.categories.length" :option="categoryOption" height="300px" />
        <div v-else class="admin-chart-empty">
          <AdminEmptyState
            :icon="IconFolder"
            :title="t('dashboard.empty.categories.title')"
            :description="t('dashboard.empty.categories.description')" />
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.tags')">
        <AdminEChart v-if="analytics.tags.length" :option="tagOption" height="300px" />
        <div v-else class="admin-chart-empty">
          <AdminEmptyState
            :icon="IconTags"
            :title="t('dashboard.empty.tags.title')"
            :description="t('dashboard.empty.tags.description')" />
        </div>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import * as echarts from 'echarts'
import {
  IconBarChart,
  IconBook,
  IconDashboard,
  IconFolder,
  IconLocation,
  IconMessage,
  IconRefresh,
  IconTags,
  IconUserGroup
} from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, getAdminDashboardAnalytics } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatCard from '@/components/AdminStatCard.vue'
import { useLatestRequest } from '@/composables/useAsyncList'
import { t } from '@/i18n'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor, verticalFade, withAlpha } from '@/utils/chart-theme'
import { formatNumber } from '@/utils/format'
import worldMap from '@/assets/world.json'
import type { AdminDashboardAnalytics, DashboardRange } from '@stellar-beacon/api-contract'

echarts.registerMap('world', worldMap as never)

const range = ref<DashboardRange>('7d')
const areaType = ref<'users' | 'visitors'>('users')
const loading = ref(false)
const errorMessage = ref('')
const analytics = reactive<AdminDashboardAnalytics>(emptyAnalytics())
const themeStore = useThemeStore()

// 区间与地域维度可以连续切换，由共享的「最新请求胜出」保护丢弃过期响应。
const latest = useLatestRequest()

const rangeLabel = computed(() => (range.value === '30d'
  ? t('dashboard.range.days30')
  : range.value === '12m' ? t('dashboard.range.months12') : t('dashboard.range.days7')))

const stats = computed(() => [
  { label: t('dashboard.stats.totalViews'), value: formatNumber(analytics.overview.totalViews), caption: t('dashboard.stats.totalViewsCaption'), icon: IconDashboard, tone: 'blue' as const },
  { label: t('dashboard.stats.todayViews'), value: formatNumber(analytics.overview.todayViews), caption: t('dashboard.stats.todayViewsCaption'), icon: IconBarChart, tone: 'green' as const },
  { label: t('dashboard.stats.monthViews'), value: formatNumber(analytics.overview.monthViews), caption: t('dashboard.stats.monthViewsCaption'), icon: IconMessage, tone: 'warm' as const },
  { label: t('dashboard.stats.users'), value: formatNumber(analytics.overview.userCount), caption: t('dashboard.stats.usersCaption'), icon: IconUserGroup, tone: 'violet' as const },
  { label: t('dashboard.stats.articles'), value: formatNumber(analytics.overview.articleCount), caption: t('dashboard.stats.articlesCaption'), icon: IconBook, tone: 'info' as const },
  { label: t('dashboard.stats.comments'), value: formatNumber(analytics.overview.messageCount), caption: t('dashboard.stats.commentsCaption'), icon: IconMessage, tone: 'danger' as const }
])

// 坐标轴、网格线、提示框都交给 chart-theme 注册的主题，这里只描述数据本身。
const trendOption = computed(() => {
  const brand = chartSeriesColor(themeStore.theme, 0)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 4, right: 16, top: 24, bottom: 2, containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: analytics.trend.map((item) => (range.value === '12m' ? item.period : item.period.slice(5))),
      axisLabel: { hideOverlap: true }
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{
      name: t('dashboard.chart.views'),
      type: 'line',
      // 折线只在悬浮时露出圆点：默认满屏圆点会把趋势线切成一串珠子。
      showSymbol: false,
      symbolSize: 7,
      lineStyle: {
        width: 2.4,
        color: brand,
        shadowBlur: 14,
        shadowColor: withAlpha(brand, 0.32),
        shadowOffsetY: 7
      },
      itemStyle: { color: brand, borderWidth: 2 },
      emphasis: { focus: 'series', scale: 1.6 },
      areaStyle: { color: verticalFade(brand) },
      data: analytics.trend.map((item) => item.views)
    }]
  }
})

const regionOption = computed(() => {
  const brand = chartSeriesColor(themeStore.theme, 0)
  return {
    tooltip: {
      trigger: 'item',
      formatter: (params: { name?: string; value?: number }) => t('dashboard.chart.regionTooltip', {
        name: params.name || t('dashboard.chart.unknownRegion'),
        value: formatNumber(params.value || 0)
      })
    },
    visualMap: {
      min: 0,
      max: Math.max(1, ...regionMapData.value.map((item) => item.value)),
      left: 'center',
      bottom: 0,
      calculable: true,
      itemWidth: 12,
      itemHeight: 74,
      inRange: { color: [withAlpha(brand, 0.14), brand] }
    },
    series: [{
      name: t('dashboard.chart.regionSeries'),
      type: 'map',
      map: 'world',
      roam: true,
      emphasis: { label: { show: false }, itemStyle: { areaColor: withAlpha(brand, 0.42) } },
      data: regionMapData.value
    }]
  }
})

const categoryOption = computed(() => distributionOption(analytics.categories))
const tagOption = computed(() => distributionOption(analytics.tags))

const regionMapData = computed(() => {
  const values = new Map<string, number>()
  for (const region of analytics.regions) {
    const continent = continentFor(region.code, region.name)
    if (continent) values.set(continent, (values.get(continent) || 0) + Number(region.value || 0))
  }
  return Array.from(values, ([name, value]) => ({ name, value }))
})

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    // 只允许最新一次请求写入：切换区间/地域维度很快，旧响应绝不能覆盖新结果。
    const value = await latest.run((signal) => getAdminDashboardAnalytics(range.value, areaType.value, { signal }))
    if (!value) return
    Object.assign(analytics, value)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('dashboard.loadFailed'))
  } finally {
    loading.value = false
  }
}

function distributionOption(items: AdminDashboardAnalytics['categories']): Record<string, unknown> {
  const total = items.reduce((sum, item) => sum + Number(item.value || 0), 0)
  const shares = new Map(items.map((item) => [
    item.name,
    total > 0 ? Math.round((Number(item.value || 0) / total) * 100) : 0
  ]))
  return {
    tooltip: { trigger: 'item', formatter: t('dashboard.chart.shareTooltip') },
    // 环心留白里放总计：环形图最缺的就是“整体量级”这个参照。
    title: {
      text: formatNumber(total),
      subtext: t('dashboard.chart.total'),
      left: 'center',
      top: '35%',
      textAlign: 'center',
      textStyle: { fontSize: 21, fontWeight: 700 },
      subtextStyle: { fontSize: 11 }
    },
    // 占比放进图例而不是环形外侧：窄卡片里外侧引线标签会被裁成“工程化 3…”，
    // 图例本身就在卡片内，位置稳定且不遮挡环体。
    legend: {
      type: 'scroll',
      bottom: 0,
      itemGap: 12,
      formatter: (name: string) => (shares.has(name) ? `${name} ${shares.get(name)}%` : name)
    },
    series: [{
      type: 'pie',
      radius: ['52%', '72%'],
      center: ['50%', '44%'],
      label: { show: false },
      labelLine: { show: false },
      avoidLabelOverlap: false,
      emphasis: { scale: true, scaleSize: 6 },
      data: items.map((item) => ({ name: item.name, value: item.value }))
    }]
  }
}

function continentFor(code: string, name: string): string {
  const value = `${code} ${name}`.toLowerCase()
  if (/(cn|jp|kr|asia|中国|日本|韩国)/.test(value)) return 'Asia'
  if (/(us|ca|north america|美国|加拿大)/.test(value)) return 'North America'
  if (/(br|ar|south america|巴西|阿根廷)/.test(value)) return 'South America'
  if (/(gb|de|fr|it|eu|europe|英国|德国|法国)/.test(value)) return 'Europe'
  if (/(au|australia|澳大利亚)/.test(value)) return 'Australia'
  if (/(africa|非洲)/.test(value)) return 'Africa'
  return ''
}

function emptyAnalytics(): AdminDashboardAnalytics {
  return {
    range: '7d',
    unit: 'day',
    overview: { totalViews: 0, todayViews: 0, monthViews: 0, userCount: 0, articleCount: 0, messageCount: 0 },
    trend: [],
    regions: [],
    categories: [],
    tags: [],
    articleRank: [],
    generatedAt: ''
  }
}
</script>

<style scoped>
.dashboard-chart-skeleton {
  height: 330px;
}

.dashboard-rank-scroll {
  max-height: 330px;
  overflow-y: auto;
}
</style>
