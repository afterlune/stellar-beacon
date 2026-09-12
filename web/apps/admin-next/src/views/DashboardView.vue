<template>
  <section class="admin-page dashboard-page">
    <AdminPageHeader title="数据仪表盘" description="把访问趋势、内容分布和访客地域放在同一个观察面里。">
      <template #actions>
        <a-radio-group v-model="range" type="button" :disabled="loading" @change="load">
          <a-radio value="7d">近 7 天</a-radio>
          <a-radio value="30d">近 30 天</a-radio>
          <a-radio value="12m">近 12 月</a-radio>
        </a-radio-group>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          刷新
        </a-button>
      </template>
    </AdminPageHeader>

    <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

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
      <a-card class="admin-panel" :bordered="false" title="访问趋势">
        <template #extra>
          <span class="admin-muted-cell">{{ rangeLabel }} · {{ analytics.unit === 'month' ? '按月' : '按日' }}</span>
        </template>
        <div v-if="loading && !analytics.trend.length" class="admin-skeleton dashboard-chart-skeleton" />
        <AdminEChart v-else-if="analytics.trend.length" :option="trendOption" height="330px" />
        <AdminEmptyState
          v-else
          :icon="IconBarChart"
          title="该区间暂无访问数据"
          description="换一个时间区间，或等待博客产生新的访问。" />
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="热门文章">
        <template #extra><span class="admin-muted-cell">浏览量排行</span></template>
        <div v-if="analytics.articleRank.length" class="admin-rank-list dashboard-rank-scroll">
          <div v-for="(article, index) in analytics.articleRank.slice(0, 10)" :key="article.id" class="admin-rank-item">
            <span class="admin-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="admin-rank-title" :title="article.title">{{ article.title || '未命名文章' }}</span>
            <span class="admin-rank-metric">{{ formatNumber(article.views) }}</span>
          </div>
        </div>
        <AdminEmptyState
          v-else
          :icon="IconBook"
          title="暂无浏览排行"
          description="文章被阅读后，这里会显示浏览量最高的内容。" />
      </a-card>
    </div>

    <div class="dashboard-grid dashboard-grid-three">
      <a-card class="admin-panel" :bordered="false" title="访客地域">
        <template #extra><span class="admin-muted-cell">按大洲聚合</span></template>
        <AdminEChart v-if="regionMapData.length" :option="regionOption" height="300px" />
        <div v-else class="admin-chart-empty">
          <AdminEmptyState
            :icon="IconLocation"
            title="暂无地域数据"
            description="访客分布需要站点持续积累访问记录。" />
        </div>
        <div v-if="analytics.regions.length" class="dashboard-region-list">
          <div v-for="region in analytics.regions.slice(0, 8)" :key="`${region.label}-${region.code}`" class="dashboard-region-item">
            <span :title="region.name">{{ region.label || region.name }}</span>
            <strong>{{ formatNumber(region.value) }}</strong>
          </div>
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="分类分布">
        <AdminEChart v-if="analytics.categories.length" :option="categoryOption" height="300px" />
        <div v-else class="admin-chart-empty">
          <AdminEmptyState :icon="IconFolder" title="暂无分类数据" description="为文章设置分类后即可看到分布。" />
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="标签分布">
        <AdminEChart v-if="analytics.tags.length" :option="tagOption" height="300px" />
        <div v-else class="admin-chart-empty">
          <AdminEmptyState :icon="IconTags" title="暂无标签数据" description="为文章添加标签后即可看到分布。" />
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
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatCard from '@/components/AdminStatCard.vue'
import { formatDateTime, formatNumber } from '@/utils/format'
import worldMap from '@/assets/world.json'
import type { AdminDashboardAnalytics, DashboardRange } from '@benetnasch/api-contract'

echarts.registerMap('world', worldMap as never)

const range = ref<DashboardRange>('7d')
const loading = ref(false)
const errorMessage = ref('')
const analytics = reactive<AdminDashboardAnalytics>(emptyAnalytics())

// Guards against out-of-order responses when the range is switched quickly:
// only the newest request is allowed to write into `analytics`.
let requestSeq = 0

const rangeLabel = computed(() => (range.value === '30d' ? '近 30 天' : range.value === '12m' ? '近 12 月' : '近 7 天'))

const stats = computed(() => [
  { label: '累计访问', value: formatNumber(analytics.overview.totalViews), caption: '站点历史访问量', icon: IconDashboard, tone: 'blue' as const },
  { label: '今日访问', value: formatNumber(analytics.overview.todayViews), caption: '今日独立访客访问', icon: IconBarChart, tone: 'green' as const },
  { label: '本月访问', value: formatNumber(analytics.overview.monthViews), caption: '本月累计访问量', icon: IconMessage, tone: 'warm' as const },
  { label: '注册用户', value: formatNumber(analytics.overview.userCount), caption: '站点用户总数', icon: IconUserGroup, tone: 'violet' as const },
  { label: '文章数量', value: formatNumber(analytics.overview.articleCount), caption: '文章与草稿内容', icon: IconBook, tone: 'info' as const },
  { label: '评论数量', value: formatNumber(analytics.overview.messageCount), caption: '已审核评论与留言', icon: IconMessage, tone: 'danger' as const }
])

const trendOption = computed(() => ({
  color: ['#4f6bd8'],
  tooltip: { trigger: 'axis' },
  grid: { left: 18, right: 18, top: 28, bottom: 20, containLabel: true },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: analytics.trend.map((item) => (range.value === '12m' ? item.period : item.period.slice(5))),
    axisLine: { lineStyle: { color: 'rgba(127,127,127,.25)' } },
    axisLabel: { color: 'rgba(127,127,127,1)' }
  },
  yAxis: {
    type: 'value',
    minInterval: 1,
    splitLine: { lineStyle: { color: 'rgba(127,127,127,.14)' } },
    axisLabel: { color: 'rgba(127,127,127,1)' }
  },
  series: [{
    name: '访问量',
    type: 'line',
    smooth: true,
    symbol: 'circle',
    symbolSize: 7,
    areaStyle: { color: 'rgba(79,107,216,.12)' },
    data: analytics.trend.map((item) => item.views)
  }]
}))

const regionOption = computed(() => ({
  tooltip: {
    trigger: 'item',
    formatter: (params: { name?: string; value?: number }) => `${params.name || '未知地域'}：${formatNumber(params.value || 0)}`
  },
  visualMap: {
    min: 0,
    max: Math.max(1, ...regionMapData.value.map((item) => item.value)),
    left: 'center',
    bottom: 0,
    calculable: true,
    textStyle: { color: 'rgba(127,127,127,1)' },
    inRange: { color: ['#e8edff', '#4f6bd8'] }
  },
  series: [{
    name: '访问地域',
    type: 'map',
    map: 'world',
    roam: true,
    itemStyle: { areaColor: 'rgba(127,127,127,.14)', borderColor: 'rgba(127,127,127,.28)' },
    emphasis: { label: { show: false } },
    data: regionMapData.value
  }]
}))

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
  const seq = ++requestSeq
  loading.value = true
  errorMessage.value = ''
  try {
    const value = await getAdminDashboardAnalytics(range.value, 'users')
    if (seq !== requestSeq) return
    Object.assign(analytics, value)
  } catch (error) {
    if (seq !== requestSeq) return
    errorMessage.value = apiErrorMessage(error, '仪表盘数据加载失败')
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

function distributionOption(items: AdminDashboardAnalytics['categories']): Record<string, unknown> {
  const total = items.reduce((sum, item) => sum + Number(item.value || 0), 0)
  const topNames = new Set(
    [...items]
      .sort((a, b) => Number(b.value || 0) - Number(a.value || 0))
      .slice(0, 5)
      .map((item) => item.name)
  )
  return {
    tooltip: { trigger: 'item', formatter: '{b}：{c}（{d}%）' },
    legend: { type: 'scroll', bottom: 0, textStyle: { color: 'rgba(127,127,127,1)' } },
    series: [{
      type: 'pie',
      radius: ['38%', '68%'],
      center: ['50%', '43%'],
      avoidLabelOverlap: true,
      // 只给占比最高的 5 个扇区画标签：长尾碎片（各占 6% 甚至 0%）
      // 全画出来会挤满圆环边缘，反而看不清主要构成，其余看图例与悬浮提示。
      minShowLabelAngle: 12,
      label: { formatter: '{b} {d}%', color: 'rgba(127,127,127,1)' },
      labelLine: { length: 6, length2: 8, smooth: true },
      itemStyle: { borderWidth: 2, borderColor: 'rgba(127,127,127,.08)' },
      data: items.map((item) => {
        const percent = total > 0 ? (Number(item.value || 0) / total) * 100 : 0
        const show = topNames.has(item.name) && percent >= 8
        return { name: item.name, value: item.value, label: { show }, labelLine: { show } }
      })
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
