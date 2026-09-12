<template>
  <section class="admin-page workplace-page">
    <div class="workplace-welcome">
      <div class="workplace-welcome-copy">
        <div class="admin-page-eyebrow">BENETNASCH / WORKPLACE</div>
        <h2>{{ greeting }}，{{ auth.user?.nickname || auth.user?.username || '管理员' }}</h2>
        <p>从内容发布、互动管理到站点运维，今天也把博客维护得井然有序。</p>
        <a-space wrap class="workplace-welcome-actions">
          <a-button type="primary" @click="router.push('/articles')">
            <template #icon><IconPlus /></template>
            发布文章
          </a-button>
          <a-button @click="router.push('/talks')">发布说说</a-button>
          <a-button @click="router.push('/dashboard')">查看仪表盘</a-button>
        </a-space>
      </div>
      <div class="workplace-welcome-aside">
        <span class="workplace-welcome-label">数据更新于</span>
        <strong>{{ updatedAt }}</strong>
        <span class="admin-muted-cell">近 7 天访问 {{ formatNumber(trendTotal) }} 次</span>
      </div>
    </div>

    <a-alert v-if="errorMessage" type="warning" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

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

    <div class="workplace-grid-main">
      <a-card class="admin-panel" :bordered="false" title="访问趋势">
        <template #extra>
          <a-space :size="8">
            <span class="admin-muted-cell">近 7 天</span>
            <a-button type="text" size="small" @click="router.push('/dashboard')">详细分析</a-button>
          </a-space>
        </template>
        <AdminEChart v-if="analytics.trend.length" :option="trendOption" height="320px" />
        <AdminEmptyState
          v-else
          :icon="IconBarChart"
          title="暂无访问数据"
          description="博客产生访问后，这里会显示逐日趋势。" />
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="热门文章">
        <template #extra><span class="admin-muted-cell">浏览量排行</span></template>
        <div v-if="analytics.articleRank.length" class="admin-rank-list">
          <div v-for="(article, index) in analytics.articleRank.slice(0, 6)" :key="article.id" class="admin-rank-item">
            <span class="admin-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="admin-rank-title" :title="article.title">{{ article.title || '未命名文章' }}</span>
            <span class="admin-rank-metric">{{ formatNumber(article.views) }}</span>
          </div>
        </div>
        <AdminEmptyState
          v-else
          :icon="IconBook"
          title="暂无浏览数据"
          description="文章被阅读后，排行会出现在这里。" />
      </a-card>
    </div>

    <div class="workplace-grid-bottom">
      <a-card class="admin-panel" :bordered="false" title="快速入口">
        <div class="workplace-shortcuts">
          <button v-for="item in shortcuts" :key="item.path" type="button" class="workplace-shortcut" @click="router.push(item.path)">
            <span class="workplace-shortcut-icon" aria-hidden="true"><component :is="item.icon" /></span>
            <span class="workplace-shortcut-copy">
              <strong>{{ item.label }}</strong>
              <small>{{ item.caption }}</small>
            </span>
          </button>
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="今日状态">
        <div class="workplace-status-list">
          <div><span>账号状态</span><AdminStatusTag kind="enabled" label="已认证" /></div>
          <div><span>权限来源</span><a-tag color="arcoblue">RBAC</a-tag></div>
          <div><span>今日访问</span><strong class="admin-num-cell">{{ formatNumber(analytics.overview.todayViews) }}</strong></div>
          <div><span>待审核评论</span><strong class="admin-num-cell">{{ formatNumber(analytics.overview.messageCount) }}</strong></div>
          <div><span>数据更新时间</span><span class="admin-muted-cell">{{ updatedAt }}</span></div>
        </div>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  IconBarChart,
  IconBook,
  IconDashboard,
  IconImage,
  IconMessage,
  IconPlus,
  IconSettings,
  IconUserGroup
} from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, getAdminDashboardAnalytics } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminStatCard from '@/components/AdminStatCard.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor, verticalFade, withAlpha } from '@/utils/chart-theme'
import { formatDateTime, formatNumber } from '@/utils/format'
import type { AdminDashboardAnalytics } from '@benetnasch/api-contract'

const router = useRouter()
const auth = useAuthStore()
const themeStore = useThemeStore()

const loading = ref(false)
const errorMessage = ref('')
const updatedAt = ref('加载中…')
const analytics = reactive<AdminDashboardAnalytics>({
  range: '7d',
  unit: 'day',
  overview: { totalViews: 0, todayViews: 0, monthViews: 0, userCount: 0, articleCount: 0, messageCount: 0 },
  trend: [],
  regions: [],
  categories: [],
  tags: [],
  articleRank: [],
  generatedAt: ''
})

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了'
  if (hour < 12) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

const stats = computed(() => [
  { label: '累计访问', value: formatNumber(analytics.overview.totalViews), caption: '站点历史访问量', icon: IconDashboard, tone: 'blue' as const },
  { label: '今日访问', value: formatNumber(analytics.overview.todayViews), caption: '今日独立访客访问', icon: IconBarChart, tone: 'green' as const },
  { label: '文章数量', value: formatNumber(analytics.overview.articleCount), caption: '已发布与草稿内容', icon: IconBook, tone: 'warm' as const },
  { label: '注册用户', value: formatNumber(analytics.overview.userCount), caption: '站点用户总数', icon: IconUserGroup, tone: 'violet' as const }
])

const trendTotal = computed(() => analytics.trend.reduce((total, item) => total + Number(item.views || 0), 0))

const shortcuts = computed(() => [
  { path: '/article-list', label: '文章管理', caption: '整理发布内容', icon: IconBook },
  { path: '/talk-list', label: '说说管理', caption: '维护动态和图片', icon: IconMessage },
  { path: '/albums', label: '相册管理', caption: '管理照片资源', icon: IconImage },
  { path: '/setting', label: '个人中心', caption: '更新个人资料', icon: IconSettings }
])

const trendOption = computed(() => {
  const brand = chartSeriesColor(themeStore.theme, 0)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 4, right: 16, top: 24, bottom: 2, containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: analytics.trend.map((item) => item.period.slice(5)),
      axisLabel: { hideOverlap: true }
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{
      name: '访问量',
      type: 'line',
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

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const value = await getAdminDashboardAnalytics('7d')
    Object.assign(analytics, value)
    updatedAt.value = value.generatedAt ? formatDateTime(value.generatedAt) : '刚刚'
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '工作台数据加载失败，可以先使用快速入口继续工作')
    updatedAt.value = '暂不可用'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.workplace-welcome-copy {
  min-width: 0;
}

.workplace-welcome-actions {
  margin-top: 18px;
}

.workplace-welcome-aside {
  display: grid;
  flex: 0 0 auto;
  gap: 4px;
  align-content: center;
  padding-left: 24px;
  border-left: 1px solid rgb(79 107 216 / 16%);
  text-align: right;
}

.workplace-welcome-label {
  color: var(--admin-muted);
  font-size: 12px;
  font-weight: 600;
}

.workplace-welcome-aside strong {
  color: var(--admin-ink-strong);
  font-size: 18px;
  font-weight: 700;
}

@media (max-width: 760px) {
  .workplace-welcome-aside {
    padding-top: 16px;
    padding-left: 0;
    border-top: 1px solid var(--admin-border);
    border-left: 0;
    text-align: left;
  }
}
</style>
