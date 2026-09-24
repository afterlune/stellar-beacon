<template>
  <section class="admin-page monitor-page">
    <AdminPageHeader :title="t('dashboard.monitor.title')" :description="t('dashboard.monitor.description')">
      <template #actions>
        <button
          class="monitor-live-badge"
          type="button"
          :aria-pressed="autoRefresh"
          @click="toggleAutoRefresh">
          <span class="admin-live-dot" aria-hidden="true" />
          {{ autoRefresh ? t('dashboard.monitor.autoRefreshOn') : t('dashboard.monitor.autoRefreshOff') }}
        </button>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('dashboard.actions.refreshNow') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-alert v-if="errorMessage" type="warning" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

    <div class="admin-stat-grid">
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-blue" aria-hidden="true"><IconUserGroup /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">{{ t('dashboard.monitor.onlineNow') }}</span>
          <strong class="admin-stat-value">{{ formatNumber(onlineTotal) }}</strong>
          <span class="admin-stat-caption">{{ t('dashboard.monitor.onlineNowCaption') }}</span>
        </span>
      </a-card>
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-green" aria-hidden="true"><IconDashboard /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">{{ t('dashboard.stats.todayViews') }}</span>
          <strong class="admin-stat-value">{{ formatNumber(analytics.overview.todayViews) }}</strong>
          <span class="admin-stat-caption">{{ t('dashboard.monitor.todayViewsCaption') }}</span>
        </span>
      </a-card>
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-info" aria-hidden="true"><IconBarChart /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">{{ t('dashboard.stats.weekViews') }}</span>
          <strong class="admin-stat-value">{{ formatNumber(trendTotal) }}</strong>
          <span class="admin-stat-caption">{{ t('dashboard.stats.weekViewsCaption') }}</span>
        </span>
      </a-card>
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-danger" aria-hidden="true"><IconExclamationCircle /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">{{ t('dashboard.stats.exceptions') }}</span>
          <strong class="admin-stat-value">{{ formatNumber(exceptionTotal) }}</strong>
          <span class="admin-stat-caption">{{ t('dashboard.stats.exceptionsCaption') }}</span>
        </span>
      </a-card>
    </div>

    <div class="monitor-grid">
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.onlineUsers')">
        <template #extra>
          <a-button v-if="onlineUsersPath" type="text" size="small" @click="router.push(onlineUsersPath)">{{ t('dashboard.actions.viewAll') }}</a-button>
        </template>
        <div class="admin-table-shell">
          <a-table :data="onlineUsers" :columns="onlineColumns" :pagination="false" :loading="loading">
            <template #user="{ record }">
              <div class="online-user-cell">
                <a-avatar :size="28" :image-url="record.avatar">{{ initialOf(record.nickname || record.username) }}</a-avatar>
                <strong>{{ record.nickname || record.username || t('dashboard.untitledUser') }}</strong>
              </div>
            </template>
            <template #status><AdminStatusTag kind="enabled" :label="t('dashboard.monitor.online')" /></template>
            <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.lastLoginTime) }}</span></template>
            <template #empty>
              <AdminEmptyState
                :icon="IconUser"
                :title="t('dashboard.empty.online.title')"
                :description="t('dashboard.empty.online.description')" />
            </template>
          </a-table>
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.trend')">
        <template #extra><span class="admin-muted-cell">{{ t('dashboard.range.days7') }}</span></template>
        <AdminEChart :option="trendOption" height="300px" />
      </a-card>
    </div>

    <div class="monitor-grid monitor-grid-bottom">
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.recentOperations')">
        <template #extra>
          <a-button v-if="operationLogsPath" type="text" size="small" @click="router.push(operationLogsPath)">{{ t('dashboard.actions.logCenter') }}</a-button>
        </template>
        <div v-if="operationLogs.length" class="admin-feed-list">
          <div v-for="record in operationLogs.slice(0, 8)" :key="String(record.id)" class="admin-feed-item">
            <span class="admin-feed-dot admin-feed-dot-success" aria-hidden="true" />
            <div class="admin-feed-copy">
              <strong :title="operationTitle(record)">{{ operationTitle(record) }}</strong>
              <small>{{ formatDateTime(record.createTime) }} · {{ record.nickname || t('dashboard.role.admin') }}</small>
            </div>
          </div>
        </div>
        <AdminEmptyState
          v-else
          :icon="IconHistory"
          :title="t('dashboard.empty.operations.title')"
          :description="t('dashboard.empty.operations.description')" />
      </a-card>

      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.panels.exceptionAlerts')">
        <template #extra>
          <a-button v-if="exceptionLogsPath" type="text" status="danger" size="small" @click="router.push(exceptionLogsPath)">{{ t('dashboard.actions.viewExceptions') }}</a-button>
        </template>
        <div v-if="exceptionLogs.length" class="admin-feed-list">
          <div v-for="record in exceptionLogs.slice(0, 8)" :key="String(record.id)" class="admin-feed-item">
            <span class="admin-feed-dot admin-feed-dot-danger" aria-hidden="true" />
            <div class="admin-feed-copy">
              <strong :title="exceptionTitle(record)">{{ exceptionTitle(record) }}</strong>
              <small>{{ formatDateTime(record.createTime) }} · {{ record.optUri || t('dashboard.fallback.unknownEndpoint') }}</small>
            </div>
          </div>
        </div>
        <AdminEmptyState
          v-else
          :icon="IconEmpty"
          :title="t('dashboard.empty.exceptions.title')"
          :description="t('dashboard.empty.exceptions.description')" />
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconBarChart,
  IconDashboard,
  IconEmpty,
  IconExclamationCircle,
  IconHistory,
  IconRefresh,
  IconUser,
  IconUserGroup
} from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  getAdminDashboardAnalytics,
  listAdminExceptionLogs,
  listAdminOnlineUsers,
  listAdminOperationLogs
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { t } from '@/i18n'
import { useMenuStore } from '@/stores/menu'
import { formatDateTime, formatNumber, initialOf } from '@/utils/format'
import type { AdminDashboardAnalytics, AdminUser } from '@stellar-beacon/api-contract'

const REFRESH_INTERVAL_MS = 30_000

const router = useRouter()
const menuStore = useMenuStore()
const loading = ref(false)
const autoRefresh = ref(true)
const errorMessage = ref('')
const onlineUsers = ref<AdminUser[]>([])
const operationLogs = ref<Record<string, unknown>[]>([])
const exceptionLogs = ref<Record<string, unknown>[]>([])
const onlineTotal = ref(0)
const exceptionTotal = ref(0)
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
    activation: { started: 0, identityCompleted: 0, contentCompleted: 0, profileVisited: 0, completed: 0, completionRate: 0 },
    trend: []
  },
  generatedAt: ''
})

let timer: ReturnType<typeof setInterval> | undefined
let inFlight = false

// 列标题走 computed：模块级常量不会跟着语言切换重算。
const onlineColumns = computed(() => [
  { title: t('dashboard.area.users'), dataIndex: 'nickname', slotName: 'user', minWidth: 160 },
  { title: 'IP', dataIndex: 'ipAddress', width: 140 },
  { title: t('dashboard.table.browser'), dataIndex: 'browser', width: 130, ellipsis: true, tooltip: true },
  { title: t('common.status'), dataIndex: 'status', slotName: 'status', width: 84 },
  { title: t('dashboard.table.lastActive'), dataIndex: 'lastLoginTime', slotName: 'time', width: 168 }
])

const trendTotal = computed(() => analytics.trend.reduce((total, item) => total + Number(item.views || 0), 0))
const trendOption = computed(() => ({
  color: ['#2f9e77'],
  tooltip: { trigger: 'axis' },
  grid: { left: 14, right: 16, top: 22, bottom: 18, containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: analytics.trend.map((item) => item.period.slice(5)) },
  yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: 'rgba(127,127,127,.14)' } } },
  series: [{
    name: t('dashboard.chart.views'),
    type: 'line',
    smooth: true,
    symbol: 'circle',
    symbolSize: 6,
    areaStyle: { color: 'rgba(47,158,119,.13)' },
    data: analytics.trend.map((item) => item.views)
  }]
}))

/**
 * The monitor summary must never link to a hard-coded path: the backend menu
 * owns the route, so resolve it from the loaded menu tree and hide the link
 * when the current account cannot see that page.
 */
const onlineUsersPath = computed(() => findMenuPath(['online']))
const operationLogsPath = computed(() => findMenuPath(['operation']))
const exceptionLogsPath = computed(() => findMenuPath(['exception']))

onMounted(() => {
  void load()
  timer = setInterval(tick, REFRESH_INTERVAL_MS)
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  timer = undefined
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

function onVisibilityChange(): void {
  // Refresh immediately when the tab becomes active again; skip hidden tabs.
  if (!document.hidden) void load()
}

function tick(): void {
  if (!autoRefresh.value || document.hidden) return
  void load()
}

function toggleAutoRefresh(): void {
  autoRefresh.value = !autoRefresh.value
}

async function load(): Promise<void> {
  // Guard against overlapping polls: a slow refresh must not stack requests.
  if (inFlight) return
  inFlight = true
  loading.value = true
  errorMessage.value = ''
  try {
    const [online, operations, exceptions, dashboard] = await Promise.all([
      listAdminOnlineUsers({ current: 1, size: 8 }),
      listAdminOperationLogs({ current: 1, size: 8 }),
      listAdminExceptionLogs({ current: 1, size: 8 }),
      getAdminDashboardAnalytics('7d', 'visitors')
    ])
    onlineUsers.value = online.items
    onlineTotal.value = online.total
    operationLogs.value = operations.items
    exceptionLogs.value = exceptions.items
    exceptionTotal.value = exceptions.total
    Object.assign(analytics, dashboard)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('dashboard.monitor.loadFailed'))
  } finally {
    loading.value = false
    inFlight = false
  }
}

function findMenuPath(keywords: string[]): string {
  for (const menu of menuStore.visibleMenus) {
    const candidates: Array<{ name: string; path: string }> = [
      { name: menu.name, path: menu.path },
      ...(menu.children || []).map((child) => ({ name: child.name, path: child.path }))
    ]
    for (const candidate of candidates) {
      const haystack = `${candidate.name} ${candidate.path}`.toLowerCase()
      if (keywords.every((keyword) => haystack.includes(keyword.toLowerCase()))) return candidate.path
    }
  }
  return ''
}

function operationTitle(record: Record<string, unknown>): string {
  return String(record.optDesc || record.optModule || record.optUri || t('dashboard.fallback.operation'))
}

function exceptionTitle(record: Record<string, unknown>): string {
  return String(record.exceptionInfo || record.optDesc || record.optUri || t('dashboard.fallback.exception'))
}
</script>

<style scoped>
.monitor-live-badge {
  cursor: pointer;
  font-family: inherit;
  transition: border-color var(--admin-duration-fast) var(--admin-ease),
    background-color var(--admin-duration-fast) var(--admin-ease);
}

.monitor-live-badge:hover {
  border-color: var(--admin-sage);
}

.monitor-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(320px, 1fr);
  gap: var(--admin-gap-lg);
  margin-bottom: var(--admin-gap-lg);
}

.monitor-grid-bottom {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.online-user-cell {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.online-user-cell strong {
  overflow: hidden;
  color: var(--admin-ink-strong);
  font-size: 13px;
  font-weight: 620;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .monitor-grid,
  .monitor-grid-bottom {
    grid-template-columns: 1fr;
  }
}
</style>
