<template>
  <section class="admin-page workplace-page">
    <div class="workplace-welcome">
      <div class="workplace-welcome-copy">
        <div class="admin-page-eyebrow">STELLAR BEACON / WORKPLACE</div>
        <h2>{{ t('dashboard.workplace.greetingLine', { greeting, name: auth.user?.nickname || auth.user?.username || t('dashboard.role.admin') }) }}</h2>
        <p>{{ t('dashboard.workplace.subtitle') }}</p>
        <a-space wrap class="workplace-welcome-actions">
          <a-button type="primary" @click="router.push('/articles')">
            <template #icon><IconPlus /></template>
            {{ t('dashboard.actions.publishArticle') }}
          </a-button>
          <a-button @click="router.push('/talks')">{{ t('dashboard.actions.publishTalk') }}</a-button>
          <a-button @click="router.push('/dashboard')">{{ t('dashboard.actions.viewDashboard') }}</a-button>
        </a-space>
      </div>
      <div class="workplace-welcome-aside">
        <span class="workplace-welcome-label">{{ t('dashboard.workplace.updatedAt') }}</span>
        <strong>{{ updatedAt || t('common.loading') }}</strong>
        <span class="admin-muted-cell">{{ t('dashboard.workplace.weekViews', { total: formatNumber(trendTotal) }) }}</span>
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
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.trend')">
        <template #extra>
          <a-space :size="8">
            <span class="admin-muted-cell">{{ t('dashboard.range.days7') }}</span>
            <a-button type="text" size="small" @click="router.push('/dashboard')">{{ t('dashboard.workplace.detailAnalysis') }}</a-button>
          </a-space>
        </template>
        <AdminEChart v-if="analytics.trend.length" :option="trendOption" height="320px" />
        <AdminEmptyState
          v-else
          :icon="IconBarChart"
          :title="t('dashboard.empty.workplaceTrend.title')"
          :description="t('dashboard.empty.workplaceTrend.description')" />
      </a-card>

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.hotArticles')">
        <template #extra><span class="admin-muted-cell">{{ t('dashboard.panels.viewRank') }}</span></template>
        <div v-if="analytics.articleRank.length" class="admin-rank-list">
          <div v-for="(article, index) in analytics.articleRank.slice(0, 6)" :key="article.id" class="admin-rank-item">
            <span class="admin-rank-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="admin-rank-title" :title="article.title">{{ article.title || t('dashboard.untitledArticle') }}</span>
            <span class="admin-rank-metric">{{ formatNumber(article.views) }}</span>
          </div>
        </div>
        <AdminEmptyState
          v-else
          :icon="IconBook"
          :title="t('dashboard.empty.workplaceRank.title')"
          :description="t('dashboard.empty.workplaceRank.description')" />
      </a-card>
    </div>

    <div class="workplace-grid-bottom">
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.shortcuts')">
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

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.todayStatus')">
        <div class="workplace-status-list">
          <div><span>{{ t('dashboard.workplace.accountStatus') }}</span><AdminStatusTag kind="enabled" :label="t('dashboard.workplace.verified')" /></div>
          <div><span>{{ t('dashboard.workplace.permissionSource') }}</span><a-tag color="arcoblue">RBAC</a-tag></div>
          <div><span>{{ t('dashboard.stats.todayViews') }}</span><strong class="admin-num-cell">{{ formatNumber(analytics.overview.todayViews) }}</strong></div>
          <div><span>{{ t('dashboard.stats.pendingComments') }}</span><strong class="admin-num-cell">{{ formatNumber(analytics.overview.messageCount) }}</strong></div>
          <div><span>{{ t('dashboard.workplace.dataUpdatedAt') }}</span><span class="admin-muted-cell">{{ updatedAt || t('common.loading') }}</span></div>
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
import { t } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor, verticalFade, withAlpha } from '@/utils/chart-theme'
import { formatDateTime, formatNumber } from '@/utils/format'
import type { AdminDashboardAnalytics } from '@stellar-beacon/api-contract'

const router = useRouter()
const auth = useAuthStore()
const themeStore = useThemeStore()

const loading = ref(false)
const errorMessage = ref('')
const updatedAt = ref('')
const analytics = reactive<AdminDashboardAnalytics>({
  range: '7d',
  unit: 'day',
  overview: { totalViews: 0, todayViews: 0, monthViews: 0, userCount: 0, articleCount: 0, messageCount: 0 },
  trend: [],
  regions: [],
  categories: [],
  tags: [],
  articleRank: [],
  growth: {
    subscribers: { total: 0, active: 0, pending: 0, unsubscribed: 0, confirmationRate: 0 },
    deliveries: { queued: 0, sending: 0, sent: 0, failed: 0, successRate: 0 },
    trend: []
  },
  generatedAt: ''
})

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return t('dashboard.workplace.greeting.night')
  if (hour < 12) return t('dashboard.workplace.greeting.morning')
  if (hour < 14) return t('dashboard.workplace.greeting.noon')
  if (hour < 18) return t('dashboard.workplace.greeting.afternoon')
  return t('dashboard.workplace.greeting.evening')
})

const stats = computed(() => [
  { label: t('dashboard.stats.totalViews'), value: formatNumber(analytics.overview.totalViews), caption: t('dashboard.stats.totalViewsCaption'), icon: IconDashboard, tone: 'blue' as const },
  { label: t('dashboard.stats.todayViews'), value: formatNumber(analytics.overview.todayViews), caption: t('dashboard.stats.todayViewsCaption'), icon: IconBarChart, tone: 'green' as const },
  { label: t('dashboard.stats.articles'), value: formatNumber(analytics.overview.articleCount), caption: t('dashboard.stats.articlesCaptionPublished'), icon: IconBook, tone: 'warm' as const },
  { label: t('dashboard.stats.users'), value: formatNumber(analytics.overview.userCount), caption: t('dashboard.stats.usersCaption'), icon: IconUserGroup, tone: 'violet' as const }
])

const trendTotal = computed(() => analytics.trend.reduce((total, item) => total + Number(item.views || 0), 0))

const shortcuts = computed(() => [
  { path: '/article-list', label: t('dashboard.workplace.shortcut.articles'), caption: t('dashboard.workplace.shortcut.articlesCaption'), icon: IconBook },
  { path: '/talk-list', label: t('dashboard.workplace.shortcut.talks'), caption: t('dashboard.workplace.shortcut.talksCaption'), icon: IconMessage },
  { path: '/albums', label: t('dashboard.workplace.shortcut.albums'), caption: t('dashboard.workplace.shortcut.albumsCaption'), icon: IconImage },
  { path: '/setting', label: t('dashboard.workplace.shortcut.profile'), caption: t('dashboard.workplace.shortcut.profileCaption'), icon: IconSettings }
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
      name: t('dashboard.chart.views'),
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
    updatedAt.value = value.generatedAt ? formatDateTime(value.generatedAt) : t('dashboard.workplace.justNow')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('dashboard.workplace.loadFailed'))
    updatedAt.value = t('dashboard.workplace.unavailable')
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
