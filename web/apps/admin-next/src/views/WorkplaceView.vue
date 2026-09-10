<template>
  <section class="admin-page workplace-page">
    <div class="workplace-welcome">
      <div>
        <div class="admin-page-eyebrow">BENETNASCH / WORKPLACE</div>
        <h2>欢迎回来，{{ auth.user?.nickname || auth.user?.username || '管理员' }}</h2>
        <p>从内容发布、互动管理到站点运维，今天也把博客维护得井然有序。</p>
      </div>
      <a-space wrap>
        <a-button type="primary" @click="router.push('/articles')">发布文章</a-button>
        <a-button @click="router.push('/talks')">发布说说</a-button>
        <a-button @click="router.push('/dashboard')">查看仪表盘</a-button>
      </a-space>
    </div>

    <div class="workplace-stat-grid">
      <a-card v-for="stat in stats" :key="stat.label" class="admin-card workplace-stat-card" :bordered="false">
        <div class="workplace-stat-icon" :class="`workplace-stat-${stat.tone}`"><component :is="stat.icon" /></div>
        <div class="workplace-stat-copy">
          <span>{{ stat.label }}</span>
          <strong>{{ stat.value }}</strong>
          <small>{{ stat.caption }}</small>
        </div>
      </a-card>
    </div>

    <div class="workplace-main-grid">
      <a-card class="admin-panel" :bordered="false" title="访问趋势">
        <template #extra><a-button type="text" size="small" @click="router.push('/dashboard')">详细分析</a-button></template>
        <AdminEChart :option="trendOption" height="320px" />
      </a-card>
      <a-card class="admin-panel workplace-rank-card" :bordered="false" title="热门文章">
        <template #extra><span class="admin-muted-cell">实时排行</span></template>
        <div v-if="analytics.articleRank.length" class="workplace-rank-list">
          <div v-for="(article, index) in analytics.articleRank.slice(0, 6)" :key="article.id" class="workplace-rank-item">
            <span class="workplace-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="workplace-rank-title">{{ article.title || '未命名文章' }}</span>
            <strong>{{ article.views }}</strong>
          </div>
        </div>
        <a-empty v-else description="暂无文章浏览数据" />
      </a-card>
    </div>

    <div class="workplace-bottom-grid">
      <a-card class="admin-panel" :bordered="false" title="快速入口">
        <div class="workplace-shortcuts">
          <button v-for="item in shortcuts" :key="item.path" type="button" class="workplace-shortcut" @click="router.push(item.path)">
            <span class="workplace-shortcut-icon"><component :is="item.icon" /></span>
            <span><strong>{{ item.label }}</strong><small>{{ item.caption }}</small></span>
          </button>
        </div>
      </a-card>
      <a-card class="admin-panel" :bordered="false" title="今日状态">
        <div class="workplace-status-list">
          <div><span>账号状态</span><a-tag color="green">已认证</a-tag></div>
          <div><span>权限来源</span><a-tag color="arcoblue">RBAC</a-tag></div>
          <div><span>数据更新时间</span><span class="admin-muted-cell">{{ updatedAt }}</span></div>
        </div>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { IconBook, IconDashboard, IconImage, IconMessage, IconSettings, IconUserGroup } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'
import { getAdminDashboardAnalytics } from '@/api/http'
import AdminEChart from '@/components/AdminEChart.vue'
import { useAuthStore } from '@/stores/auth'
import type { AdminDashboardAnalytics } from '@benetnasch/api-contract'

const router = useRouter()
const auth = useAuthStore()
const analytics = reactive<AdminDashboardAnalytics>({
  range: '7d', unit: 'day', overview: { totalViews: 0, todayViews: 0, monthViews: 0, userCount: 0, articleCount: 0, messageCount: 0 },
  trend: [], regions: [], categories: [], tags: [], articleRank: [], generatedAt: ''
})
const updatedAt = ref('加载中')

const stats = computed(() => [
  { label: '累计访问', value: formatNumber(analytics.overview.totalViews), caption: '独立访客累计访问', icon: IconDashboard, tone: 'blue' },
  { label: '今日访问', value: formatNumber(analytics.overview.todayViews), caption: '今日独立访问', icon: IconDashboard, tone: 'green' },
  { label: '文章数量', value: formatNumber(analytics.overview.articleCount), caption: '已发布与草稿内容', icon: IconBook, tone: 'warm' },
  { label: '注册用户', value: formatNumber(analytics.overview.userCount), caption: '站点用户总数', icon: IconUserGroup, tone: 'purple' }
])

const shortcuts = [
  { path: '/article-list', label: '文章管理', caption: '整理发布内容', icon: IconBook },
  { path: '/talk-list', label: '说说管理', caption: '维护动态和图片', icon: IconMessage },
  { path: '/albums', label: '相册管理', caption: '管理照片资源', icon: IconImage },
  { path: '/setting', label: '个人中心', caption: '更新个人资料', icon: IconSettings }
]

const trendOption = computed(() => ({
  color: ['#5871d8'],
  tooltip: { trigger: 'axis' },
  grid: { left: 18, right: 18, top: 24, bottom: 20, containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: analytics.trend.map(item => item.period.slice(5)) },
  yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: '#edf0f5' } } },
  series: [{ name: '访问量', type: 'line', smooth: true, symbol: 'circle', symbolSize: 7, areaStyle: { color: 'rgba(88,113,216,.12)' }, data: analytics.trend.map(item => item.views) }]
}))

onMounted(() => void load())

async function load(): Promise<void> {
  try {
    const value = await getAdminDashboardAnalytics('7d')
    Object.assign(analytics, value)
    updatedAt.value = formatDate(value.generatedAt)
  } catch {
    updatedAt.value = '暂不可用'
  }
}

function formatNumber(value: number): string { return new Intl.NumberFormat('zh-CN').format(Number(value) || 0) }
function formatDate(value: string): string { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '—' }
</script>

<style scoped>
.workplace-welcome { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 24px; padding: 30px 34px; border: 1px solid #dfe5f7; border-radius: 20px; background: linear-gradient(125deg, #f0f3ff, #f8fbff 58%, #eef8f5); box-shadow: var(--admin-shadow-card); }
.workplace-welcome h2 { margin: 0; font-size: clamp(24px, 3vw, 34px); letter-spacing: -.03em; }
.workplace-welcome p { margin: 10px 0 0; color: var(--admin-muted); }
.workplace-stat-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; margin-bottom: 18px; }
.workplace-stat-card { display: flex; align-items: center; gap: 14px; }
.workplace-stat-icon { width: 42px; height: 42px; display: grid; flex: 0 0 auto; place-items: center; border-radius: 14px; font-size: 21px; }
.workplace-stat-blue { color: #5871d8; background: #eef1ff; }.workplace-stat-green { color: #4d9d83; background: #eaf6f1; }.workplace-stat-warm { color: #d78260; background: #fff0ea; }.workplace-stat-purple { color: #8e76c6; background: #f2edff; }
.workplace-stat-copy { display: grid; gap: 2px; min-width: 0; }.workplace-stat-copy span,.workplace-stat-copy small { color: var(--admin-muted); font-size: 12px; }.workplace-stat-copy strong { font-size: 24px; line-height: 1.25; }.workplace-stat-copy small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.workplace-main-grid { display: grid; grid-template-columns: minmax(0, 1.65fr) minmax(300px, 1fr); gap: 18px; margin-bottom: 18px; }
.workplace-bottom-grid { display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(280px, 1fr); gap: 18px; }
.workplace-rank-list { display: grid; gap: 6px; }.workplace-rank-item { display: flex; align-items: center; gap: 10px; min-height: 39px; border-bottom: 1px solid #f0f2f6; }.workplace-rank-index { width: 26px; color: var(--admin-brand); font-size: 12px; font-weight: 750; }.workplace-rank-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.workplace-rank-item strong { color: var(--admin-muted); font-size: 12px; }
.workplace-shortcuts { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }.workplace-shortcut { display: flex; align-items: center; gap: 10px; padding: 14px; border: 1px solid var(--admin-border); border-radius: 12px; background: #fbfcff; text-align: left; cursor: pointer; }.workplace-shortcut:hover { border-color: #cbd5f3; background: var(--admin-brand-soft); }.workplace-shortcut-icon { width: 30px; height: 30px; display: grid; place-items: center; color: var(--admin-brand); border-radius: 9px; background: var(--admin-brand-soft); }.workplace-shortcut span:last-child { min-width: 0; }.workplace-shortcut strong,.workplace-shortcut small { display: block; }.workplace-shortcut strong { font-size: 13px; }.workplace-shortcut small { margin-top: 3px; color: var(--admin-muted); font-size: 11px; }
.workplace-status-list { display: grid; gap: 15px; }.workplace-status-list > div { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
@media (max-width: 980px) { .workplace-stat-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.workplace-main-grid,.workplace-bottom-grid { grid-template-columns: 1fr; } }
@media (max-width: 640px) { .workplace-welcome { align-items: flex-start; flex-direction: column; padding: 24px; }.workplace-shortcuts { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
