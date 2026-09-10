<template>
  <section class="admin-page dashboard-page">
    <AdminPageHeader title="数据仪表盘" description="把访问趋势、内容分布和访客地域放在同一个观察面里。">
      <template #actions>
        <a-space>
          <a-radio-group v-model="range" type="button" @change="load">
            <a-radio value="7d">近 7 天</a-radio>
            <a-radio value="30d">近 30 天</a-radio>
            <a-radio value="12m">近 12 月</a-radio>
          </a-radio-group>
          <a-button :loading="loading" @click="load">刷新</a-button>
        </a-space>
      </template>
    </AdminPageHeader>

    <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

    <div class="dashboard-stat-grid">
      <a-card v-for="stat in stats" :key="stat.label" class="admin-card dashboard-stat-card" :bordered="false">
        <div class="dashboard-stat-icon" :class="`dashboard-stat-${stat.tone}`"><component :is="stat.icon" /></div>
        <div>
          <div class="dashboard-stat-label">{{ stat.label }}</div>
          <strong class="dashboard-stat-value">{{ formatNumber(stat.value) }}</strong>
          <div class="dashboard-stat-caption">{{ stat.caption }}</div>
        </div>
      </a-card>
    </div>

    <div class="dashboard-grid dashboard-grid-wide">
      <a-card class="admin-panel" :bordered="false" title="访问趋势">
        <template #extra><span class="admin-muted-cell">{{ rangeLabel }} · {{ analytics.unit === 'month' ? '按月' : '按日' }}</span></template>
        <AdminEChart :option="trendOption" height="330px" />
      </a-card>
      <a-card class="admin-panel" :bordered="false" title="热门文章">
        <template #extra><span class="admin-muted-cell">浏览量排行</span></template>
        <div v-if="analytics.articleRank.length" class="dashboard-rank-list">
          <div v-for="(article, index) in analytics.articleRank.slice(0, 8)" :key="article.id" class="dashboard-rank-item">
            <span class="dashboard-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="dashboard-rank-title">{{ article.title || '未命名文章' }}</span>
            <strong>{{ formatNumber(article.views) }}</strong>
          </div>
        </div>
        <a-empty v-else description="暂无浏览排行" />
      </a-card>
    </div>

    <div class="dashboard-grid dashboard-grid-three">
      <a-card class="admin-panel" :bordered="false" title="访客地域">
        <template #extra><span class="admin-muted-cell">世界视图</span></template>
        <AdminEChart :option="regionOption" height="300px" />
        <div class="dashboard-region-list">
          <div v-for="region in analytics.regions.slice(0, 8)" :key="`${region.label}-${region.code}`" class="dashboard-region-item">
            <span>{{ region.label || region.name }}</span>
            <strong>{{ formatNumber(region.value) }}</strong>
          </div>
          <span v-if="!analytics.regions.length" class="admin-muted-cell">暂时没有地域数据</span>
        </div>
      </a-card>
      <a-card class="admin-panel" :bordered="false" title="分类分布">
        <AdminEChart :option="categoryOption" height="300px" />
        <a-empty v-if="!analytics.categories.length" description="暂无分类数据" />
      </a-card>
      <a-card class="admin-panel" :bordered="false" title="标签分布">
        <AdminEChart :option="tagOption" height="300px" />
        <a-empty v-if="!analytics.tags.length" description="暂无标签数据" />
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import * as echarts from 'echarts'
import { IconBook, IconDashboard, IconMessage, IconUserGroup } from '@arco-design/web-vue/es/icon'

import { getAdminDashboardAnalytics } from '@/api/http'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import worldMap from '@/assets/world.json'
import type { AdminDashboardAnalytics, DashboardRange } from '@benetnasch/api-contract'

echarts.registerMap('world', worldMap as never)

const range = ref<DashboardRange>('7d')
const loading = ref(false)
const errorMessage = ref('')
const analytics = reactive<AdminDashboardAnalytics>(emptyAnalytics())

const rangeLabel = computed(() => range.value === '30d' ? '近 30 天' : range.value === '12m' ? '近 12 月' : '近 7 天')
const stats = computed(() => [
  { label: '累计访问', value: analytics.overview.totalViews, caption: '站点历史访问量', icon: IconDashboard, tone: 'blue' },
  { label: '今日访问', value: analytics.overview.todayViews, caption: '今日独立访客访问', icon: IconDashboard, tone: 'green' },
  { label: '本月访问', value: analytics.overview.monthViews, caption: '本月累计访问量', icon: IconMessage, tone: 'warm' },
  { label: '注册用户', value: analytics.overview.userCount, caption: '站点用户总数', icon: IconUserGroup, tone: 'purple' },
  { label: '文章数量', value: analytics.overview.articleCount, caption: '文章与草稿内容', icon: IconBook, tone: 'orange' },
  { label: '评论数量', value: analytics.overview.messageCount, caption: '已审核评论与留言', icon: IconMessage, tone: 'teal' }
])

const trendOption = computed(() => ({
  color: ['#5871d8'],
  tooltip: { trigger: 'axis' },
  grid: { left: 18, right: 18, top: 28, bottom: 20, containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: analytics.trend.map(item => range.value === '12m' ? item.period : item.period.slice(5)) },
  yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: '#edf0f5' } } },
  series: [{ name: '访问量', type: 'line', smooth: true, symbol: 'circle', symbolSize: 7, areaStyle: { color: 'rgba(88,113,216,.12)' }, data: analytics.trend.map(item => item.views) }]
}))

const regionOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: (params: { name?: string; value?: number }) => `${params.name || '未知地域'}：${formatNumber(params.value || 0)}` },
  visualMap: { min: 0, max: Math.max(1, ...regionMapData.value.map(item => item.value)), left: 'center', bottom: 0, calculable: true, inRange: { color: ['#e8edff', '#5871d8'] } },
  series: [{ name: '访问地域', type: 'map', map: 'world', roam: true, emphasis: { label: { show: false } }, data: regionMapData.value }]
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
  loading.value = true
  errorMessage.value = ''
  try {
    Object.assign(analytics, await getAdminDashboardAnalytics(range.value, 'users'))
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '仪表盘数据加载失败'
  } finally {
    loading.value = false
  }
}

function distributionOption(items: AdminDashboardAnalytics['categories']): Record<string, unknown> {
  return {
    tooltip: { trigger: 'item' },
    legend: { type: 'scroll', bottom: 0 },
    series: [{ type: 'pie', radius: ['38%', '68%'], center: ['50%', '43%'], avoidLabelOverlap: true, label: { formatter: '{b}\n{d}%' }, data: items.map(item => ({ name: item.name, value: item.value })) }]
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

function formatNumber(value: number): string { return new Intl.NumberFormat('zh-CN').format(Number(value) || 0) }

function emptyAnalytics(): AdminDashboardAnalytics {
  return { range: '7d', unit: 'day', overview: { totalViews: 0, todayViews: 0, monthViews: 0, userCount: 0, articleCount: 0, messageCount: 0 }, trend: [], regions: [], categories: [], tags: [], articleRank: [], generatedAt: '' }
}
</script>

<style scoped>
.dashboard-stat-grid { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 14px; margin-bottom: 18px; }
.dashboard-stat-card { display: flex; align-items: center; gap: 12px; min-width: 0; }
.dashboard-stat-icon { width: 40px; height: 40px; display: grid; flex: 0 0 auto; place-items: center; border-radius: 13px; font-size: 19px; }
.dashboard-stat-blue { color: #5871d8; background: #eef1ff; }.dashboard-stat-green { color: #4d9d83; background: #eaf6f1; }.dashboard-stat-warm { color: #d78260; background: #fff0ea; }.dashboard-stat-purple { color: #8e76c6; background: #f2edff; }.dashboard-stat-orange { color: #c98c43; background: #fff6e7; }.dashboard-stat-teal { color: #3d9ca0; background: #e8f8f6; }
.dashboard-stat-label,.dashboard-stat-caption { color: var(--admin-muted); font-size: 12px; }.dashboard-stat-value { display: block; margin: 3px 0; font-size: 22px; line-height: 1.1; }.dashboard-stat-caption { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dashboard-grid { display: grid; gap: 18px; margin-bottom: 18px; }.dashboard-grid-wide { grid-template-columns: minmax(0, 1.7fr) minmax(300px, 1fr); }.dashboard-grid-three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.dashboard-rank-list { display: grid; gap: 4px; }.dashboard-rank-item { display: flex; align-items: center; gap: 10px; min-height: 40px; border-bottom: 1px solid var(--admin-border); }.dashboard-rank-index { width: 26px; color: var(--admin-brand); font-weight: 750; }.dashboard-rank-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.dashboard-rank-item strong { color: var(--admin-muted); font-size: 12px; }
.dashboard-region-list { display: grid; gap: 7px; max-height: 132px; overflow: auto; }.dashboard-region-item { display: flex; align-items: center; justify-content: space-between; gap: 14px; color: var(--admin-muted); font-size: 12px; }.dashboard-region-item strong { color: var(--admin-ink); }
@media (max-width: 1240px) { .dashboard-stat-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }.dashboard-grid-three { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 900px) { .dashboard-grid-wide,.dashboard-grid-three { grid-template-columns: 1fr; } }
@media (max-width: 520px) { .dashboard-stat-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.dashboard-stat-card { padding: 16px !important; }.dashboard-stat-value { font-size: 19px; } }
</style>
