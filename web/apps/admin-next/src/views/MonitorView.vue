<template>
  <section class="admin-page monitor-page">
    <AdminPageHeader title="实时监控" description="查看在线会话、操作记录和异常日志。">
      <template #actions>
        <button
          class="monitor-live-badge"
          type="button"
          :aria-pressed="autoRefresh"
          @click="toggleAutoRefresh">
          <span class="admin-live-dot" aria-hidden="true" />
          {{ autoRefresh ? '自动刷新 30 秒' : '自动刷新已暂停' }}
        </button>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          立即刷新
        </a-button>
      </template>
    </AdminPageHeader>

    <a-alert v-if="errorMessage" type="warning" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

    <div class="admin-stat-grid">
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-blue" aria-hidden="true"><IconUserGroup /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">当前在线用户</span>
          <strong class="admin-stat-value">{{ formatNumber(onlineTotal) }}</strong>
          <span class="admin-stat-caption">活跃登录会话</span>
        </span>
      </a-card>
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-green" aria-hidden="true"><IconDashboard /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">今日访问</span>
          <strong class="admin-stat-value">{{ formatNumber(analytics.overview.todayViews) }}</strong>
          <span class="admin-stat-caption">按日实时统计</span>
        </span>
      </a-card>
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-info" aria-hidden="true"><IconBarChart /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">近 7 日访问</span>
          <strong class="admin-stat-value">{{ formatNumber(trendTotal) }}</strong>
          <span class="admin-stat-caption">访问趋势合计</span>
        </span>
      </a-card>
      <a-card class="admin-card admin-stat-card" :bordered="false">
        <span class="admin-stat-icon admin-tone-danger" aria-hidden="true"><IconExclamationCircle /></span>
        <span class="admin-stat-copy">
          <span class="admin-stat-label">累计异常</span>
          <strong class="admin-stat-value">{{ formatNumber(exceptionTotal) }}</strong>
          <span class="admin-stat-caption">可进入异常日志定位</span>
        </span>
      </a-card>
    </div>

    <div class="monitor-grid">
      <a-card class="admin-panel" :bordered="false" title="在线用户">
        <template #extra>
          <a-button v-if="onlineUsersPath" type="text" size="small" @click="router.push(onlineUsersPath)">查看全部</a-button>
        </template>
        <div class="admin-table-shell">
          <a-table :data="onlineUsers" :columns="onlineColumns" :pagination="false" :loading="loading">
            <template #user="{ record }">
              <div class="online-user-cell">
                <a-avatar :size="28" :image-url="record.avatar">{{ initialOf(record.nickname || record.username) }}</a-avatar>
                <strong>{{ record.nickname || record.username || '未命名用户' }}</strong>
              </div>
            </template>
            <template #status><AdminStatusTag kind="enabled" label="在线" /></template>
            <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.lastLoginTime) }}</span></template>
            <template #empty>
              <AdminEmptyState :icon="IconUser" title="当前没有在线用户" description="当有用户登录并保持会话时，会出现在这里。" />
            </template>
          </a-table>
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="访问趋势">
        <template #extra><span class="admin-muted-cell">近 7 天</span></template>
        <AdminEChart :option="trendOption" height="300px" />
      </a-card>
    </div>

    <div class="monitor-grid monitor-grid-bottom">
      <a-card class="admin-panel" :bordered="false" title="最近操作">
        <template #extra>
          <a-button v-if="operationLogsPath" type="text" size="small" @click="router.push(operationLogsPath)">日志中心</a-button>
        </template>
        <div v-if="operationLogs.length" class="admin-feed-list">
          <div v-for="record in operationLogs.slice(0, 8)" :key="String(record.id)" class="admin-feed-item">
            <span class="admin-feed-dot admin-feed-dot-success" aria-hidden="true" />
            <div class="admin-feed-copy">
              <strong :title="operationTitle(record)">{{ operationTitle(record) }}</strong>
              <small>{{ formatDateTime(record.createTime) }} · {{ record.nickname || '管理员' }}</small>
            </div>
          </div>
        </div>
        <AdminEmptyState v-else :icon="IconHistory" title="暂无操作记录" description="后台的写操作会记录在这里。" />
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="异常提醒">
        <template #extra>
          <a-button v-if="exceptionLogsPath" type="text" status="danger" size="small" @click="router.push(exceptionLogsPath)">查看异常</a-button>
        </template>
        <div v-if="exceptionLogs.length" class="admin-feed-list">
          <div v-for="record in exceptionLogs.slice(0, 8)" :key="String(record.id)" class="admin-feed-item">
            <span class="admin-feed-dot admin-feed-dot-danger" aria-hidden="true" />
            <div class="admin-feed-copy">
              <strong :title="exceptionTitle(record)">{{ exceptionTitle(record) }}</strong>
              <small>{{ formatDateTime(record.createTime) }} · {{ record.optUri || '未知接口' }}</small>
            </div>
          </div>
        </div>
        <AdminEmptyState v-else :icon="IconEmpty" title="暂无异常记录" description="未捕获的后端异常会记录在这里。" />
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
  generatedAt: ''
})

let timer: ReturnType<typeof setInterval> | undefined
let inFlight = false

const onlineColumns = [
  { title: '用户', dataIndex: 'nickname', slotName: 'user', minWidth: 160 },
  { title: 'IP', dataIndex: 'ipAddress', width: 140 },
  { title: '浏览器', dataIndex: 'browser', width: 130, ellipsis: true, tooltip: true },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 84 },
  { title: '最近活跃', dataIndex: 'lastLoginTime', slotName: 'time', width: 168 }
]

const trendTotal = computed(() => analytics.trend.reduce((total, item) => total + Number(item.views || 0), 0))
const trendOption = computed(() => ({
  color: ['#2f9e77'],
  tooltip: { trigger: 'axis' },
  grid: { left: 14, right: 16, top: 22, bottom: 18, containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: analytics.trend.map((item) => item.period.slice(5)) },
  yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: 'rgba(127,127,127,.14)' } } },
  series: [{
    name: '访问量',
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
    errorMessage.value = apiErrorMessage(error, '监控数据加载失败')
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
  return String(record.optDesc || record.optModule || record.optUri || '后台操作')
}

function exceptionTitle(record: Record<string, unknown>): string {
  return String(record.exceptionInfo || record.optDesc || record.optUri || '请求异常')
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
