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

    <a-card class="admin-panel monitor-health-panel" :bordered="false" :title="t('dashboard.monitor.health.title')">
      <template #extra>
        <div class="health-panel-actions">
          <a-tag v-if="systemHealth" :color="statusColor(systemHealth.status)">{{ statusLabel(systemHealth.status) }}</a-tag>
          <a-select v-model="trendRange" size="small" class="health-range-select" @change="loadSystemHealth">
            <a-option value="1h">{{ t('dashboard.monitor.health.range1h') }}</a-option>
            <a-option value="24h">{{ t('dashboard.monitor.health.range24h') }}</a-option>
            <a-option value="7d">{{ t('dashboard.monitor.health.range7d') }}</a-option>
          </a-select>
        </div>
      </template>

      <a-alert v-if="healthError" type="warning" class="health-inline-alert">{{ healthError }}</a-alert>
      <a-alert v-else-if="systemHealth && !systemHealth.historyAvailable" type="warning" class="health-inline-alert">
        {{ historyMessage(systemHealth.historyMessage) }}
      </a-alert>

      <div v-if="onlineInstances.length" class="health-instance-list">
        <article v-for="instance in onlineInstances" :key="instance.instanceId" class="health-instance-card">
          <header class="health-instance-header">
            <div>
              <strong>{{ instance.instanceId }}</strong>
              <small>{{ t('dashboard.monitor.health.lastCheck') }} · {{ formatDateTime(instance.capturedAt) }}</small>
            </div>
            <a-tag :color="statusColor(instance.status)">{{ statusLabel(instance.status) }}</a-tag>
          </header>

          <div class="health-component-grid">
            <div v-for="component in instance.components" :key="component.name" class="health-component">
              <div class="health-component-heading">
                <strong>{{ componentLabel(component.name) }}</strong>
                <a-tag size="small" :color="statusColor(component.status)">{{ statusLabel(component.status) }}</a-tag>
              </div>
              <small>{{ componentMessage(component.message) }} · {{ formatLatency(component.latencyMs) }} · {{ formatDateTime(component.checkedAt) }}</small>
            </div>
          </div>

          <div class="health-metric-grid">
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.uptime') }}</small><strong>{{ formatUptime(instance.resources.uptimeSeconds) }}</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.cpu') }}</small><strong>{{ instance.resources.cpuPercent.toFixed(1) }}%</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.heap') }}</small><strong>{{ formatBytes(instance.resources.heapAllocBytes) }}</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.goroutines') }}</small><strong>{{ formatNumber(instance.resources.goroutines) }}</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.requests') }}</small><strong>{{ formatNumber(instance.http.requests) }}</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.errors') }}</small><strong>{{ formatNumber(instance.http.status4xx + instance.http.status5xx) }}</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.p95') }}</small><strong>{{ formatLatency(instance.http.p95LatencyMs) }}</strong></div>
            <div class="health-metric"><small>{{ t('dashboard.monitor.health.gc') }}</small><strong>{{ formatNumber(instance.resources.gcTotal) }}</strong></div>
          </div>

          <div class="health-worker-list">
            <div v-for="worker in instance.workers" :key="worker.name" class="health-worker">
              <span>{{ workerLabel(worker.name) }}</span>
              <a-tag size="small" :color="statusColor(worker.status)">{{ statusLabel(worker.status) }}</a-tag>
              <small v-if="worker.queued || worker.failed || worker.running">
                {{ t('dashboard.monitor.health.queue') }} {{ formatNumber(worker.queued) }} · {{ t('dashboard.monitor.health.running') }} {{ formatNumber(worker.running) }} · {{ t('dashboard.monitor.health.failed') }} {{ formatNumber(worker.failed) }}
              </small>
            </div>
          </div>
        </article>
      </div>
      <a-empty v-else-if="!healthLoading" :description="t('dashboard.monitor.health.noOnlineInstances')" />
      <a-skeleton v-else :animation="true" :loading="true" />
    </a-card>

    <div class="health-status-grid">
      <a-card class="admin-panel status-timeline-panel" :bordered="false" :title="t('dashboard.monitor.health.historyTitle')">
        <template #extra>
          <a-select v-model="historyRange" size="small" class="health-range-select" @change="loadSystemHealth">
            <a-option value="24h">{{ t('dashboard.monitor.health.range24h') }}</a-option>
            <a-option value="7d">{{ t('dashboard.monitor.health.range7d') }}</a-option>
            <a-option value="30d">{{ t('dashboard.monitor.health.range30d') }}</a-option>
            <a-option value="90d">{{ t('dashboard.monitor.health.range90d') }}</a-option>
          </a-select>
        </template>
        <a-alert v-if="timelineError" type="warning" class="health-inline-alert">{{ timelineError }}</a-alert>
        <a-alert v-else-if="statusTimeline?.message" type="warning" class="health-inline-alert">
          {{ historyMessage(statusTimeline.message) }}
        </a-alert>
        <div v-if="statusTimeline?.available && timelineRows.length" class="status-timeline">
          <div class="timeline-axis">
            <span>{{ formatDateTime(statusTimeline.from) }}</span>
            <span>{{ timelineMidpoint }}</span>
            <span>{{ formatDateTime(statusTimeline.to) }}</span>
          </div>
          <div v-for="row in timelineRows" :key="`${row.replicaId}:${row.scope}`" class="timeline-row">
            <div class="timeline-row-label">
              <div class="timeline-row-heading">
                <strong>{{ historyScopeLabel(row.scope) }}</strong>
                <span
                  class="timeline-health-share"
                  tabindex="0"
                  role="img"
                  :title="timelineShareTooltip(row)"
                  :aria-label="timelineShareTooltip(row)">
                  {{ statusLabel('healthy') }} {{ row.shares.healthy.toFixed(1) }}%
                </span>
              </div>
              <small>{{ row.replicaId }}</small>
            </div>
            <div class="timeline-track">
              <button
                v-for="segment in row.segments"
                :key="`${segment.period.startedAt}:${segment.period.status}`"
                class="timeline-period"
                :class="`timeline-period-${segment.displayStatus}`"
                :style="{ left: `${segment.left}%`, width: `${segment.width}%` }"
                :title="periodTooltip(segment.period, segment.displayStatus)"
                :aria-label="periodTooltip(segment.period, segment.displayStatus)"
                :disabled="!segment.period.incidentId"
                @click="segment.period.incidentId && openIncident(segment.period.incidentId)" />
            </div>
          </div>
          <div class="timeline-axis timeline-axis-bottom">
            <span>{{ t('dashboard.monitor.health.timelineFrom') }}</span>
            <span>{{ t('dashboard.monitor.health.timelineNow') }}</span>
          </div>
        </div>
        <a-empty v-else-if="!healthLoading" :description="t('dashboard.monitor.health.noStatusHistory')" />
        <a-skeleton v-else :animation="true" :loading="true" />
        <div class="timeline-legend" aria-label="Status legend">
          <span><i class="timeline-legend-healthy" />{{ statusLabel('healthy') }}</span>
          <span><i class="timeline-legend-degraded" />{{ statusLabel('degraded') }}</span>
          <span><i class="timeline-legend-unhealthy" />{{ statusLabel('unhealthy') }}</span>
          <span><i class="timeline-legend-gap" />{{ statusLabel('monitoring_gap') }}</span>
        </div>
      </a-card>

      <a-card class="admin-panel incident-list-panel" :bordered="false" :title="t('dashboard.monitor.health.incidentsTitle')">
        <template #extra>
          <a-tag :color="openIncidentCount ? 'red' : 'green'">
            {{ t('dashboard.monitor.health.openIncidentCount', { count: openIncidentCount }) }}
          </a-tag>
        </template>
        <div v-if="visibleIncidents.length" class="incident-list">
          <button v-for="incident in visibleIncidents" :key="incident.id" class="incident-list-item" type="button" @click="openIncident(incident.id)">
            <i class="incident-severity" :class="`incident-severity-${incident.severity}`" aria-hidden="true" />
            <span class="incident-list-copy">
              <strong>{{ historyScopeLabel(incident.scope) }}</strong>
              <small>{{ formatDateTime(incident.startedAt) }} · {{ incident.instanceId }}</small>
            </span>
            <a-tag size="small" :color="incident.state === 'open' ? 'red' : 'green'">
              {{ incident.state === 'open' ? t('dashboard.monitor.health.incidentOpen') : t('dashboard.monitor.health.incidentResolved') }}
            </a-tag>
          </button>
        </div>
        <a-empty v-else-if="!healthLoading" :description="t('dashboard.monitor.health.noIncidents')" />
        <a-skeleton v-else :animation="true" :loading="true" />
      </a-card>
    </div>

    <a-drawer
      v-model:visible="incidentVisible"
      width="min(720px, 94vw)"
      :footer="false"
      :title="selectedIncident ? historyScopeLabel(selectedIncident.scope) : t('dashboard.monitor.health.incidentDetails')">
      <a-skeleton v-if="incidentLoading" :animation="true" :loading="true" />
      <a-alert v-else-if="incidentError" type="warning">{{ incidentError }}</a-alert>
      <template v-else-if="selectedIncident">
        <div class="incident-detail-heading">
          <a-tag :color="selectedIncident.severity === 'outage' ? 'red' : 'orange'">
            {{ selectedIncident.severity === 'outage' ? t('dashboard.monitor.health.outage') : t('dashboard.monitor.health.warning') }}
          </a-tag>
          <a-tag :color="selectedIncident.state === 'open' ? 'red' : 'green'">
            {{ selectedIncident.state === 'open' ? t('dashboard.monitor.health.incidentOpen') : t('dashboard.monitor.health.incidentResolved') }}
          </a-tag>
        </div>
        <dl class="incident-detail-meta">
          <div><dt>{{ t('dashboard.monitor.health.startedAt') }}</dt><dd>{{ formatDateTime(selectedIncident.startedAt) }}</dd></div>
          <div><dt>{{ t('dashboard.monitor.health.lastSeenAt') }}</dt><dd>{{ formatDateTime(selectedIncident.lastSeenAt) }}</dd></div>
          <div><dt>{{ t('dashboard.monitor.health.instance') }}</dt><dd>{{ selectedIncident.instanceId }}</dd></div>
        </dl>
        <p v-if="selectedIncident.message" class="incident-detail-message">{{ historyMessage(selectedIncident.message) }}</p>
        <h3 class="incident-updates-title">{{ t('dashboard.monitor.health.updatesTitle') }}</h3>
        <div v-if="selectedIncident.updates.length" class="incident-updates">
          <article v-for="update in selectedIncident.updates" :key="update.id" class="incident-update">
            <header><strong>{{ update.authorName }}</strong><time>{{ formatDateTime(update.createdAt) }}</time></header>
            <p>{{ update.content }}</p>
          </article>
        </div>
        <a-empty v-else :description="t('dashboard.monitor.health.noUpdates')" />
        <form class="incident-update-form" @submit.prevent="submitIncidentUpdate">
          <a-alert v-if="incidentError" type="warning" class="health-inline-alert">{{ incidentError }}</a-alert>
          <a-textarea
            v-model="incidentNote"
            :max-length="4000"
            show-word-limit
            :auto-size="{ minRows: 3, maxRows: 7 }"
            :placeholder="t('dashboard.monitor.health.updatePlaceholder')" />
          <a-button type="primary" :loading="incidentSaving" :disabled="!incidentNote.trim()" html-type="submit">
            {{ t('dashboard.monitor.health.addUpdate') }}
          </a-button>
        </form>
      </template>
    </a-drawer>

    <a-alert
      v-if="systemTrends?.message && systemHealth?.historyAvailable"
      type="warning"
      class="health-inline-alert">
      {{ historyMessage(systemTrends.message) }}
    </a-alert>

    <div class="monitor-grid health-trend-grid">
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.monitor.health.httpTrend')">
        <AdminEChart v-if="systemTrends?.available" :option="httpHealthTrendOption" height="280px" />
        <a-empty v-else :description="t('dashboard.monitor.health.noHistory')" />
      </a-card>
      <a-card class="admin-panel" :bordered="false" :title="t('dashboard.monitor.health.resourceTrend')">
        <AdminEChart v-if="systemTrends?.available" :option="resourceHealthTrendOption" height="280px" />
        <a-empty v-else :description="t('dashboard.monitor.health.noHistory')" />
      </a-card>
    </div>

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
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
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
  addSystemMonitorIncidentUpdate,
  getAdminDashboardAnalytics,
  getSystemHealth,
  getSystemHealthTimeline,
  getSystemHealthTrends,
  getSystemMonitorIncident,
  listAdminExceptionLogs,
  listAdminOnlineUsers,
  listAdminOperationLogs,
  listSystemMonitorIncidents
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { locale, t, te } from '@/i18n'
import { useMenuStore } from '@/stores/menu'
import { formatDateTime, formatNumber, initialOf } from '@/utils/format'
import type {
  AdminDashboardAnalytics,
  AdminUser,
  SystemHealthSnapshot,
  SystemMonitorHistoryRange,
  SystemMonitorIncident,
  SystemMonitorRange,
  SystemMonitorStatusPeriod,
  SystemMonitorTimeline,
  SystemMonitorTrends
} from '@stellar-beacon/api-contract'

const REFRESH_INTERVAL_MS = 30_000

const router = useRouter()
const menuStore = useMenuStore()
const loading = ref(false)
const healthLoading = ref(false)
const autoRefresh = ref(true)
const errorMessage = ref('')
const healthError = ref('')
const systemHealth = ref<SystemHealthSnapshot | null>(null)
const systemTrends = ref<SystemMonitorTrends | null>(null)
const statusTimeline = ref<SystemMonitorTimeline | null>(null)
const incidents = ref<SystemMonitorIncident[]>([])
const trendRange = ref<SystemMonitorRange>('1h')
const historyRange = ref<SystemMonitorHistoryRange>('30d')
const timelineError = ref('')
const selectedIncident = ref<SystemMonitorIncident | null>(null)
const incidentVisible = ref(false)
const incidentLoading = ref(false)
const incidentSaving = ref(false)
const incidentNote = ref('')
const incidentError = ref('')
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
let healthInFlight = false
let healthRefreshQueued = false

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

const healthPoints = computed(() => systemTrends.value?.points || [])
const onlineInstances = computed(() => (systemHealth.value?.instances || []).filter((instance) => instance.status !== 'stale'))
const onlineReplicaIds = computed(() => new Set(onlineInstances.value.map((instance) => instance.replicaId)))
const visibleIncidents = computed(() => incidents.value.filter((incident) => onlineReplicaIds.value.has(incident.replicaId)))
const openIncidentCount = computed(() => visibleIncidents.value.filter((incident) => incident.state === 'open').length)
watch([selectedIncident, onlineReplicaIds], ([incident, activeReplicas]) => {
  if (incident && !activeReplicas.has(incident.replicaId)) {
    incidentVisible.value = false
    selectedIncident.value = null
  }
})
const timelineMidpoint = computed(() => {
  if (!statusTimeline.value) return ''
  const from = new Date(statusTimeline.value.from).getTime()
  const to = new Date(statusTimeline.value.to).getTime()
  return formatDateTime(new Date(from + (to - from) / 2).toISOString())
})
const timelineRows = computed(() => {
  const timeline = statusTimeline.value
  if (!timeline) return []
  const from = new Date(timeline.from).getTime()
  const to = new Date(timeline.to).getTime()
  const span = Math.max(1, to - from)
  const activeReplicas = onlineReplicaIds.value
  const groups = new Map<string, SystemMonitorStatusPeriod[]>()
  for (const period of timeline.periods) {
    if (!activeReplicas.has(period.replicaId)) continue
    const key = `${period.replicaId}\u0000${period.scope}`
    const group = groups.get(key) || []
    group.push(period)
    groups.set(key, group)
  }
  return Array.from(groups.entries()).map(([key, periods]) => {
    const [replicaId, scope] = key.split('\u0000')
    const durations: Record<TimelineStatus, number> = {
      healthy: 0,
      degraded: 0,
      unhealthy: 0,
      monitoring_gap: 0
    }
    const clippedPeriods = periods.map((period) => ({
      period,
      start: Math.max(from, new Date(period.startedAt).getTime()),
      end: Math.min(to, period.endedAt ? new Date(period.endedAt).getTime() : to)
    })).filter((item) => item.end > item.start).sort((left, right) => left.start - right.start)
    let cursor = from
    const segments: TimelineSegment[] = []
    for (const item of clippedPeriods) {
      if (item.start > cursor) durations.monitoring_gap += item.start - cursor
      const start = Math.max(cursor, item.start)
      const end = Math.max(start, item.end)
      const displayStatus = timelineStatus(item.period.status)
      durations[displayStatus] += end - start
      if (end > start) {
        segments.push({
          period: item.period,
          displayStatus,
          left: ((start - from) / span) * 100,
          width: ((end - start) / span) * 100
        })
      }
      cursor = Math.max(cursor, end)
    }
    if (cursor < to) durations.monitoring_gap += to - cursor
    const shares = Object.fromEntries(
      Object.entries(durations).map(([status, duration]) => [status, (duration / span) * 100])
    ) as Record<TimelineStatus, number>
    return {
      replicaId,
      scope,
      shares,
      segments
    }
  }).sort((left, right) => left.scope.localeCompare(right.scope) || left.replicaId.localeCompare(right.replicaId))
})
const httpHealthTrendOption = computed(() => {
  const requestsLabel = t('dashboard.monitor.health.requests')
  const clientErrorsLabel = t('dashboard.monitor.health.clientErrors')
  const errorsLabel = t('dashboard.monitor.health.serverErrors')
  const p95Label = t('dashboard.monitor.health.p95')
  return {
    color: ['#2f9e77', '#b37a36', '#df7552', '#5878a6'],
    tooltip: { trigger: 'axis' },
    legend: { data: [requestsLabel, clientErrorsLabel, errorsLabel, p95Label] },
    grid: { left: 18, right: 36, top: 38, bottom: 22, containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: healthPoints.value.map((point) => trendLabel(point.period)) },
    yAxis: [
      { type: 'value', minInterval: 1, name: requestsLabel, splitLine: { lineStyle: { color: 'rgba(127,127,127,.14)' } } },
      { type: 'value', min: 0, name: 'ms', position: 'right', splitLine: { show: false } }
    ],
    series: [
      { name: requestsLabel, type: 'line', smooth: true, symbol: 'none', data: healthPoints.value.map((point) => point.requests) },
      { name: clientErrorsLabel, type: 'line', smooth: true, symbol: 'none', data: healthPoints.value.map((point) => point.clientErrors) },
      { name: errorsLabel, type: 'line', smooth: true, symbol: 'none', data: healthPoints.value.map((point) => point.serverErrors) },
      { name: p95Label, type: 'line', yAxisIndex: 1, smooth: true, symbol: 'none', data: healthPoints.value.map((point) => point.p95LatencyMs) }
    ]
  }
})
const resourceHealthTrendOption = computed(() => ({
  color: ['#5878a6', '#b37a36'],
  tooltip: { trigger: 'axis' },
  legend: { data: [t('dashboard.monitor.health.cpu'), t('dashboard.monitor.health.heap')] },
  grid: { left: 18, right: 42, top: 38, bottom: 22, containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: healthPoints.value.map((point) => trendLabel(point.period)) },
  yAxis: [
    { type: 'value', min: 0, max: 100, name: '%', splitLine: { lineStyle: { color: 'rgba(127,127,127,.14)' } } },
    { type: 'value', min: 0, name: 'MB', position: 'right', splitLine: { show: false } }
  ],
  series: [
    { name: t('dashboard.monitor.health.cpu'), type: 'line', smooth: true, symbol: 'none', data: healthPoints.value.map((point) => point.cpuPercent) },
    { name: t('dashboard.monitor.health.heap'), type: 'line', yAxisIndex: 1, smooth: true, symbol: 'none', data: healthPoints.value.map((point) => point.heapAllocBytes / 1024 / 1024) }
  ]
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
  void loadSystemHealth()
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

async function loadSystemHealth(): Promise<void> {
  if (healthInFlight) {
    healthRefreshQueued = true
    return
  }
  healthInFlight = true
  healthLoading.value = true
  try {
    do {
      healthRefreshQueued = false
      healthError.value = ''
      const selectedRange = trendRange.value
      const selectedHistoryRange = historyRange.value
      try {
        const [snapshot, trends] = await Promise.all([
          getSystemHealth(),
          getSystemHealthTrends(selectedRange)
        ])
        systemHealth.value = snapshot
        systemTrends.value = trends
      } catch (error) {
        healthError.value = apiErrorMessage(error, t('dashboard.monitor.health.loadFailed'))
      }
      try {
        const [timelineResult, incidentResult] = await Promise.allSettled([
          getSystemHealthTimeline(selectedHistoryRange),
          listSystemMonitorIncidents(selectedHistoryRange, 100)
        ])
        if (timelineResult.status === 'fulfilled') statusTimeline.value = timelineResult.value
        if (incidentResult.status === 'fulfilled') incidents.value = incidentResult.value
        const failure = timelineResult.status === 'rejected'
          ? timelineResult.reason
          : incidentResult.status === 'rejected' ? incidentResult.reason : null
        timelineError.value = failure
          ? apiErrorMessage(failure, t('dashboard.monitor.health.timelineLoadFailed'))
          : ''
      } catch (error) {
        timelineError.value = apiErrorMessage(error, t('dashboard.monitor.health.timelineLoadFailed'))
      }
    } while (healthRefreshQueued)
  } finally {
    healthLoading.value = false
    healthInFlight = false
  }
}

function statusColor(status: string): string {
  if (status === 'healthy') return 'green'
  if (status === 'degraded' || status === 'not_configured') return 'orange'
  if (status === 'stale') return 'gray'
  if (status === 'unhealthy') return 'red'
  return 'gray'
}

function statusLabel(status: string): string {
  return t(`dashboard.monitor.health.status.${status}`)
}

function historyScopeLabel(scope: string): string {
  if (scope === 'system') return t('dashboard.monitor.health.systemScope')
  if (scope.startsWith('worker:')) return workerLabel(scope.slice('worker:'.length))
  return componentLabel(scope)
}

type TimelineStatus = 'healthy' | 'degraded' | 'unhealthy' | 'monitoring_gap'

interface TimelineSegment {
  period: SystemMonitorStatusPeriod
  displayStatus: TimelineStatus
  left: number
  width: number
}

function timelineStatus(status: SystemMonitorStatusPeriod['status']): TimelineStatus {
  if (status === 'healthy' || status === 'degraded' || status === 'unhealthy') return status
  return 'monitoring_gap'
}

function timelineShareTooltip(row: { shares: Record<TimelineStatus, number> }): string {
  const timeline = statusTimeline.value
  const range = timeline
    ? `${formatDateTime(timeline.from)} – ${formatDateTime(timeline.to)}`
    : ''
  const details = (['healthy', 'degraded', 'unhealthy', 'monitoring_gap'] as TimelineStatus[])
    .map((status) => `${statusLabel(status)}: ${row.shares[status].toFixed(1)}%`)
  return [t('dashboard.monitor.health.windowShare', { range }), ...details].join('\n')
}

function periodTooltip(period: SystemMonitorStatusPeriod, displayStatus = timelineStatus(period.status)): string {
  const range = period.endedAt
    ? `${formatDateTime(period.startedAt)} – ${formatDateTime(period.endedAt)}`
    : formatDateTime(period.startedAt)
  const detail = period.message ? ` · ${historyMessage(period.message)}` : ''
  return `${historyScopeLabel(period.scope)} · ${statusLabel(displayStatus)} · ${range}${detail}`
}

async function openIncident(id: number): Promise<void> {
  incidentVisible.value = true
  incidentLoading.value = true
  incidentError.value = ''
  incidentNote.value = ''
  selectedIncident.value = null
  try {
    selectedIncident.value = await getSystemMonitorIncident(id)
  } catch (error) {
    incidentError.value = apiErrorMessage(error, t('dashboard.monitor.health.incidentLoadFailed'))
  } finally {
    incidentLoading.value = false
  }
}

async function submitIncidentUpdate(): Promise<void> {
  const incident = selectedIncident.value
  const content = incidentNote.value.trim()
  if (!incident || !content || incidentSaving.value) return
  incidentSaving.value = true
  incidentError.value = ''
  try {
    const update = await addSystemMonitorIncidentUpdate(incident.id, content)
    incident.updates.push(update)
    incidentNote.value = ''
    void loadSystemHealth()
  } catch (error) {
    incidentError.value = apiErrorMessage(error, t('dashboard.monitor.health.updateFailed'))
  } finally {
    incidentSaving.value = false
  }
}

function componentLabel(name: string): string {
  return t(`dashboard.monitor.health.component.${name}`)
}

function componentMessage(message: string): string {
  const key = `dashboard.monitor.health.message.${message}`
  return te(key) ? t(key) : message
}

function historyMessage(message?: string): string {
  if (!message) return t('dashboard.monitor.health.historyUnavailable')
  const key = `dashboard.monitor.health.message.${message}`
  return te(key) ? t(key) : message
}

function workerLabel(name: string): string {
  return t(`dashboard.monitor.health.worker.${name}`)
}

function formatLatency(value: number): string {
  return `${Number(value || 0).toFixed(1)} ms`
}

function formatBytes(value: number): string {
  if (value < 1024) return `${formatNumber(value)} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

function formatUptime(value: number): string {
  const totalSeconds = Math.max(0, Math.floor(value || 0))
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  if (locale.value === 'zh-CN') return days ? `${days}天 ${hours}小时` : `${hours}小时 ${minutes}分钟`
  return days ? `${days}d ${hours}h` : `${hours}h ${minutes}m`
}

function trendLabel(value: string): string {
  const date = new Date(value)
  return trendRange.value === '7d'
    ? date.toLocaleDateString(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit' })
    : date.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
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

<style scoped src="./MonitorView.css"></style>
