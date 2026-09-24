<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('growth.title')" :description="t('growth.description')">
      <template #actions>
        <a-button :loading="loading" @click="loadAll">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <div class="growth-overview-grid">
      <a-card class="admin-panel growth-stat-panel" :bordered="false">
        <span class="growth-stat-label">{{ t('growth.active') }}</span>
        <strong class="growth-stat-value">{{ formatNumber(health.subscribers.active) }}</strong>
        <span class="growth-stat-caption">{{ t('growth.confirmationRate') }} {{ formatPercent(health.subscribers.confirmationRate) }}</span>
      </a-card>
      <a-card class="admin-panel growth-stat-panel" :bordered="false">
        <span class="growth-stat-label">{{ t('growth.pending') }}</span>
        <strong class="growth-stat-value">{{ formatNumber(health.subscribers.pending) }}</strong>
        <span class="growth-stat-caption">{{ t('growth.totalSubscribers') }} {{ formatNumber(health.subscribers.total) }}</span>
      </a-card>
      <a-card class="admin-panel growth-stat-panel" :bordered="false">
        <span class="growth-stat-label">{{ t('growth.queued') }}</span>
        <strong class="growth-stat-value">{{ formatNumber(health.deliveries.queued + health.deliveries.sending) }}</strong>
        <span class="growth-stat-caption">{{ t('growth.deliverySuccessRate') }} {{ formatPercent(health.deliveries.successRate) }}</span>
      </a-card>
      <a-card class="admin-panel growth-stat-panel" :bordered="false">
        <span class="growth-stat-label">{{ t('growth.failed') }}</span>
        <strong class="growth-stat-value" :class="{ 'growth-danger': health.deliveries.failed > 0 }">{{ formatNumber(health.deliveries.failed) }}</strong>
        <span class="growth-stat-caption">{{ t('growth.manualRetryHint') }}</span>
      </a-card>
    </div>

    <a-card class="admin-panel growth-health-panel" :bordered="false">
      <div class="growth-health-row">
        <div>
          <strong>{{ t('growth.smtpHealth') }}</strong>
          <span class="growth-health-message">{{ health.smtp.message }}</span>
        </div>
        <a-space>
          <a-tag :color="health.smtp.reachable ? 'green' : 'red'">
            {{ health.smtp.reachable ? t('growth.smtpReachable') : t('growth.smtpUnavailable') }}
          </a-tag>
          <span v-if="health.smtp.checkedAt" class="admin-muted-cell">{{ formatDateTime(health.smtp.checkedAt) }}</span>
          <a-button size="small" :loading="healthLoading" @click="loadHealth">{{ t('growth.checkNow') }}</a-button>
        </a-space>
      </div>
    </a-card>

    <a-card class="admin-panel" :bordered="false">
      <a-tabs v-model:active-key="activeTab" @change="loadAll">
        <a-tab-pane key="subscribers" :title="t('growth.subscribers')">
          <div class="admin-table-toolbar">
            <div class="admin-table-toolbar-main">
              <a-input-search v-model="keyword" :placeholder="t('growth.keyword')" allow-clear @search="reloadSubscribers" />
              <a-select v-model="status" :placeholder="t('growth.allStatus')" allow-clear style="width: 150px" @change="reloadSubscribers">
                <a-option value="pending">{{ t('growth.pending') }}</a-option>
                <a-option value="active">{{ t('growth.active') }}</a-option>
                <a-option value="unsubscribed">{{ t('growth.unsubscribed') }}</a-option>
              </a-select>
            </div>
            <span class="admin-toolbar-caption">{{ t('pagination.total', { total: subscriberTotal }) }}</span>
          </div>
          <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('growth.loadFailed')" @retry="loadSubscribers" />
          <a-table
            :data="subscribers"
            :columns="subscriberColumns"
            :loading="loading"
            :pagination="subscriberPagination"
            row-key="id"
            @page-change="changeSubscriberPage"
            @page-size-change="changeSubscriberPageSize">
            <template #status="{ record }">
              <a-tag :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
            </template>
            <template #createdAt="{ record }">{{ formatDateTime(record.createdAt) }}</template>
            <template #actions="{ record }">
              <a-space>
                <a-button v-if="record.status === 'pending'" type="text" size="small" @click="resend(record.id)">{{ t('growth.resend') }}</a-button>
                <a-button type="text" size="small" @click="toggleStatus(record)">{{ record.status === 'unsubscribed' ? t('growth.enable') : t('growth.disable') }}</a-button>
              </a-space>
            </template>
            <template #empty>
              <AdminEmptyState :icon="IconEmail" :title="t('growth.emptyTitle')" :description="t('growth.emptyHint')" />
            </template>
          </a-table>
        </a-tab-pane>

        <a-tab-pane key="deliveries" :title="t('growth.deliveries')">
          <div class="admin-table-toolbar">
            <div class="admin-table-toolbar-main">
              <a-select v-model="deliveryStatus" :placeholder="t('growth.allStatus')" allow-clear style="width: 150px" @change="reloadDeliveries">
                <a-option value="queued">{{ t('growth.queued') }}</a-option>
                <a-option value="sending">{{ t('growth.sending') }}</a-option>
                <a-option value="sent">{{ t('growth.sent') }}</a-option>
                <a-option value="failed">{{ t('growth.failed') }}</a-option>
              </a-select>
              <a-button type="outline" :disabled="health.deliveries.failed === 0" @click="retryFailed">{{ t('growth.retryFailed') }}</a-button>
            </div>
            <span class="admin-toolbar-caption">{{ t('pagination.total', { total: deliveryTotal }) }}</span>
          </div>
          <a-table :data="deliveries" :columns="deliveryColumns" :loading="loading" :pagination="deliveryPagination" row-key="id" @page-change="changeDeliveryPage" @page-size-change="changeDeliveryPageSize">
            <template #status="{ record }"><a-tag :color="deliveryColor(record.status)">{{ deliveryLabel(record.status) }}</a-tag></template>
            <template #attempts="{ record }">{{ record.attempts }}</template>
            <template #createdAt="{ record }">{{ formatDateTime(record.createdAt) }}</template>
            <template #actions="{ record }"><a-button v-if="record.status === 'failed'" type="text" size="small" @click="retry(record.id)">{{ t('growth.retry') }}</a-button></template>
            <template #empty><AdminEmptyState :icon="IconEmail" :title="t('growth.emptyDeliveriesTitle')" :description="t('growth.emptyDeliveriesHint')" /></template>
          </a-table>
        </a-tab-pane>
          <a-tab-pane key="analytics" :title="t('growth.analytics')">
            <div class="growth-analytics-toolbar">
              <a-radio-group v-model="growthRange" type="button" :disabled="loading" @change="loadGrowth">
                <a-radio value="7d">{{ t('dashboard.range.days7') }}</a-radio>
                <a-radio value="30d">{{ t('dashboard.range.days30') }}</a-radio>
                <a-radio value="12m">{{ t('dashboard.range.months12') }}</a-radio>
              </a-radio-group>
            </div>
            <div class="growth-activation-panel">
              <header>
                <strong>{{ t('growth.activation.title') }}</strong>
                <span>{{ t('growth.activation.description') }}</span>
              </header>
              <div class="growth-activation-grid">
                <article v-for="stage in activationStages" :key="stage.key">
                  <span>{{ stage.label }}</span>
                  <strong>{{ formatNumber(stage.value) }}</strong>
                  <small>{{ formatPercent(stage.rate) }}</small>
                </article>
              </div>
            </div>
            <AdminEChart v-if="growthDashboard.trend.length" :option="growthTrendOption" height="320px" />
            <AdminEmptyState v-else :icon="IconBarChart" :title="t('growth.emptyAnalyticsTitle')" :description="t('growth.emptyAnalyticsHint')" />
            <a-table :data="growthItems" :columns="growthColumns" :loading="loading" row-key="eventName-day">
              <template #eventName="{ record }">{{ t(`growth.${record.eventName}`) }}</template>
              <template #count="{ record }">{{ formatNumber(record.count) }}</template>
            </a-table>
          </a-tab-pane>
      </a-tabs>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconBarChart, IconEmail, IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, getAdminDashboardAnalytics, getGrowthSummary, getNewsletterHealth, listNewsletterDeliveries, listNewsletterSubscribers, resendNewsletterConfirmation, retryFailedNewsletterDeliveries, retryNewsletterDelivery, updateNewsletterSubscriberStatus } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { t } from '@/i18n'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor, verticalFade } from '@/utils/chart-theme'
import { formatDateTime, formatNumber } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { DashboardGrowth, DashboardRange, GrowthSummaryItem, NewsletterDelivery, NewsletterHealth, NewsletterSubscriber } from '@stellar-beacon/api-contract'

const activeTab = ref('subscribers')
const loading = ref(false)
const errorMessage = ref('')
const keyword = ref('')
const status = ref<string | undefined>(undefined)
const deliveryStatus = ref<string | undefined>(undefined)
const growthRange = ref<DashboardRange>('30d')
const current = ref(1)
const pageSize = ref(20)
const subscriberTotal = ref(0)
const subscribers = ref<NewsletterSubscriber[]>([])
const growthItems = ref<GrowthSummaryItem[]>([])
const growthDashboard = ref<DashboardGrowth>(emptyGrowth())
const health = ref<NewsletterHealth>(emptyHealth())
const healthLoading = ref(false)
const deliveries = ref<NewsletterDelivery[]>([])
const deliveryTotal = ref(0)
const themeStore = useThemeStore()

const subscriberColumns = computed(() => [
  { title: t('growth.email'), dataIndex: 'email', slotName: 'email', minWidth: 260 },
  { title: t('growth.status'), dataIndex: 'status', slotName: 'status', width: 130 },
  { title: t('growth.createdAt'), dataIndex: 'createdAt', slotName: 'createdAt', width: 190 },
  { title: t('growth.actions'), dataIndex: 'actions', slotName: 'actions', width: 250 }
])
const growthColumns = computed(() => [
  { title: t('growth.event'), dataIndex: 'eventName', slotName: 'eventName', minWidth: 220 },
  { title: t('growth.day'), dataIndex: 'day', width: 180 },
  { title: t('growth.count'), dataIndex: 'count', slotName: 'count', width: 140 }
])
const deliveryColumns = computed(() => [
  { title: t('growth.email'), dataIndex: 'subscriberEmail', minWidth: 240 },
  { title: t('growth.article'), dataIndex: 'articleTitle', minWidth: 260, ellipsis: true, tooltip: true },
  { title: t('growth.status'), dataIndex: 'status', slotName: 'status', width: 120 },
  { title: t('growth.attempts'), dataIndex: 'attempts', slotName: 'attempts', width: 100 },
  { title: t('growth.createdAt'), dataIndex: 'createdAt', slotName: 'createdAt', width: 190 },
  { title: t('growth.actions'), dataIndex: 'actions', slotName: 'actions', width: 90 }
])
const subscriberPagination = computed(() => tablePagination(current.value, pageSize.value, subscriberTotal.value))
const deliveryPagination = computed(() => tablePagination(current.value, pageSize.value, deliveryTotal.value))

const activationStages = computed(() => {
  const activation = growthDashboard.value.activation || {
    started: 0, identityCompleted: 0, contentCompleted: 0, profileVisited: 0, completed: 0, completionRate: 0
  }
  const rate = (value: number) => activation.started > 0 ? value / activation.started * 100 : 0
  return [
    { key: 'started', label: t('growth.activation.started'), value: activation.started, rate: activation.started > 0 ? 100 : 0 },
    { key: 'identity', label: t('growth.activation.identity'), value: activation.identityCompleted, rate: rate(activation.identityCompleted) },
    { key: 'content', label: t('growth.activation.content'), value: activation.contentCompleted, rate: rate(activation.contentCompleted) },
    { key: 'profile', label: t('growth.activation.profile'), value: activation.profileVisited, rate: rate(activation.profileVisited) },
    { key: 'completed', label: t('growth.activation.completed'), value: activation.completed, rate: activation.completionRate }
  ]
})

const growthTrendOption = computed(() => {
  const colors = [0, 1, 2, 3].map((index) => chartSeriesColor(themeStore.theme, index))
  const labels = growthDashboard.value.trend.map((item) => growthRange.value === '12m' ? item.period : item.period.slice(5))
  return {
    tooltip: { trigger: 'axis' },
    legend: { top: 0, right: 0, itemGap: 14 },
    grid: { left: 4, right: 16, top: 34, bottom: 2, containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: labels, axisLabel: { hideOverlap: true } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      growthLine(t('growth.subscribe_confirm'), colors[0], growthDashboard.value.trend.map((item) => item.subscribeConfirms)),
      growthLine(t('growth.share_click'), colors[1], growthDashboard.value.trend.map((item) => item.shareClicks)),
      growthLine(t('growth.deliverySent'), colors[2], growthDashboard.value.trend.map((item) => item.deliverySent)),
      growthLine(t('growth.deliveryFailed'), colors[3], growthDashboard.value.trend.map((item) => item.deliveryFailed))
    ]
  }
})

onMounted(() => { void loadAll() })

async function loadAll(): Promise<void> {
  await loadHealth()
  if (activeTab.value === 'deliveries') {
    await loadDeliveries()
    return
  }
  if (activeTab.value === 'analytics') {
    await loadGrowth()
    return
  }
  await loadSubscribers()
}

async function loadDeliveries(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listNewsletterDeliveries({ current: current.value, size: pageSize.value, status: deliveryStatus.value || '' })
    deliveries.value = page.items
    deliveryTotal.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('growth.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadSubscribers(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listNewsletterSubscribers({ current: current.value, size: pageSize.value, keyword: keyword.value.trim(), status: status.value || '' })
    subscribers.value = page.items
    subscriberTotal.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('growth.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadGrowth(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const days = growthRange.value === '7d' ? 7 : 30
    const [items, dashboard] = await Promise.all([
      getGrowthSummary(days),
      getAdminDashboardAnalytics(growthRange.value, 'users')
    ])
    growthItems.value = items
    growthDashboard.value = dashboard.growth || emptyGrowth()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('growth.loadFailed'))
  } finally { loading.value = false }
}

function reloadSubscribers(): void { current.value = 1; void loadSubscribers() }
function reloadDeliveries(): void { current.value = 1; void loadDeliveries() }
function changeSubscriberPage(page: number): void { current.value = page; void loadSubscribers() }
function changeSubscriberPageSize(size: number): void { pageSize.value = size; current.value = 1; void loadSubscribers() }
function changeDeliveryPage(page: number): void { current.value = page; void loadDeliveries() }
function changeDeliveryPageSize(size: number): void { pageSize.value = size; current.value = 1; void loadDeliveries() }

async function resend(id: number): Promise<void> {
  try { await resendNewsletterConfirmation(id); Message.success(t('growth.resent')) } catch (error) { Message.error(apiErrorMessage(error, t('growth.loadFailed'))) }
}

async function toggleStatus(record: NewsletterSubscriber): Promise<void> {
  const next = record.status === 'unsubscribed' ? 'active' : 'unsubscribed'
  try { await updateNewsletterSubscriberStatus(record.id, next); Message.success(t('growth.saved')); await loadSubscribers() } catch (error) { Message.error(apiErrorMessage(error, t('growth.loadFailed'))) }
}

async function retry(id: number): Promise<void> {
  try { await retryNewsletterDelivery(id); Message.success(t('common.operationSuccess')); await Promise.all([loadDeliveries(), loadHealth()]) } catch (error) { Message.error(apiErrorMessage(error, t('growth.loadFailed'))) }
}

async function retryFailed(): Promise<void> {
  try {
    const count = await retryFailedNewsletterDeliveries()
    Message.success(t('growth.retried', { count }))
    await Promise.all([loadDeliveries(), loadHealth()])
  } catch (error) {
    Message.error(apiErrorMessage(error, t('growth.loadFailed')))
  }
}

async function loadHealth(): Promise<void> {
  healthLoading.value = true
  try {
    health.value = await getNewsletterHealth()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('growth.loadFailed'))
  } finally {
    healthLoading.value = false
  }
}

function statusLabel(value: unknown): string {
  const key = String(value || '')
  return t(key === 'pending' ? 'growth.pending' : key === 'active' ? 'growth.active' : 'growth.unsubscribed')
}

function statusColor(value: unknown): string {
  return value === 'active' ? 'green' : value === 'unsubscribed' ? 'gray' : 'orange'
}

function deliveryColor(value: unknown): string {
  return value === 'sent' ? 'green' : value === 'failed' ? 'red' : 'arcoblue'
}

function deliveryLabel(value: unknown): string {
  const key = String(value || '')
  return t(key === 'queued' ? 'growth.queued' : key === 'sending' ? 'growth.sending' : key === 'sent' ? 'growth.sent' : 'growth.failed')
}

function formatPercent(value: number): string {
  return `${Number(value || 0).toFixed(1)}%`
}

function growthLine(name: string, color: string, data: number[]): Record<string, unknown> {
  return { name, type: 'line', showSymbol: false, smooth: true, lineStyle: { width: 2, color }, itemStyle: { color }, areaStyle: { color: verticalFade(color, 0.04) }, data }
}

function emptyGrowth(): DashboardGrowth {
  return {
    subscribers: { total: 0, active: 0, pending: 0, unsubscribed: 0, confirmationRate: 0 },
    deliveries: { queued: 0, sending: 0, sent: 0, failed: 0, successRate: 0 },
    activation: { started: 0, identityCompleted: 0, contentCompleted: 0, profileVisited: 0, completed: 0, completionRate: 0 },
    trend: []
  }
}

function emptyHealth(): NewsletterHealth {
  return {
    subscribers: emptyGrowth().subscribers,
    deliveries: emptyGrowth().deliveries,
    smtp: { configured: false, reachable: false, host: '', port: 0, tls: false, auth: false, message: '' }
  }
}
</script>

<style scoped>
.growth-overview-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.growth-stat-panel {
  min-height: 126px;
}

.growth-stat-label,
.growth-stat-caption {
  display: block;
}

.growth-stat-label {
  color: var(--admin-muted);
  font-size: 13px;
}

.growth-stat-value {
  display: block;
  margin: 8px 0 4px;
  color: var(--admin-ink-strong);
  font-size: 28px;
  line-height: 1;
}

.growth-stat-caption {
  color: var(--admin-subtle);
  font-size: 12px;
}

.growth-danger {
  color: var(--admin-danger) !important;
}

.growth-health-panel {
  margin-bottom: 16px;
}

.growth-health-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.growth-health-row strong,
.growth-health-message {
  display: block;
}

.growth-health-message {
  margin-top: 4px;
  color: var(--admin-muted);
  font-size: 12px;
}

.growth-analytics-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}

.growth-activation-panel {
  margin-bottom: 18px;
  padding: 18px;
  border: 1px solid var(--admin-border);
  border-radius: 12px;
  background: var(--admin-surface-soft);
}

.growth-activation-panel header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.growth-activation-panel header span {
  color: var(--admin-muted);
  font-size: 12px;
}

.growth-activation-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.growth-activation-grid article {
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: 10px;
  background: var(--admin-surface);
}

.growth-activation-grid span,
.growth-activation-grid strong,
.growth-activation-grid small {
  display: block;
}

.growth-activation-grid span,
.growth-activation-grid small {
  color: var(--admin-muted);
  font-size: 12px;
}

.growth-activation-grid strong {
  margin: 6px 0 3px;
  color: var(--admin-ink-strong);
  font-size: 22px;
}

@media (max-width: 1050px) {
  .growth-overview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .growth-activation-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
