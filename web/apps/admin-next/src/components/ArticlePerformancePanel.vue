<template>
  <section id="article-performance" class="article-performance" data-testid="article-performance">
    <a-card :bordered="false" class="admin-panel article-performance__card">
      <header class="article-performance__header">
        <div>
          <h2>{{ t('articles.performance.title') }}</h2>
          <p>{{ t('articles.performance.description') }}</p>
        </div>
        <div class="article-performance__actions">
          <a-radio-group v-model="range" type="button" size="small" :disabled="loading">
            <a-radio value="7d">{{ t('contentPerformance.range.days7') }}</a-radio>
            <a-radio value="30d">{{ t('contentPerformance.range.days30') }}</a-radio>
            <a-radio value="90d">{{ t('contentPerformance.range.days90') }}</a-radio>
            <a-radio value="12m">{{ t('contentPerformance.range.months12') }}</a-radio>
          </a-radio-group>
          <a-button size="small" @click="toggleExpanded">
            {{ expanded ? t('articles.performance.collapse') : t('articles.performance.expand') }}
          </a-button>
        </div>
      </header>

      <div v-if="loading && !detail" class="admin-skeleton article-performance__skeleton" />
      <AdminErrorState
        v-else-if="errorMessage"
        :error="errorMessage"
        :title="t('articles.performance.loadFailed')"
        @retry="load" />

      <template v-else-if="detail">
        <div class="article-performance__stats">
          <div v-for="stat in stats" :key="stat.label" class="article-performance__stat">
            <span>{{ stat.label }}</span>
            <strong>{{ stat.value }}</strong>
            <small v-if="stat.caption">{{ stat.caption }}</small>
          </div>
        </div>

        <div v-if="diagnostics.length" class="article-performance__diagnostics">
          <div
            v-for="diagnostic in diagnostics"
            :key="diagnostic.key"
            class="article-performance__diagnostic"
            :class="`is-${diagnostic.tone}`">
            <strong>{{ diagnostic.title }}</strong>
            <span>{{ diagnostic.description }}</span>
          </div>
        </div>

        <div v-if="expanded" class="article-performance__details">
          <div class="article-performance__detail-grid">
            <section class="article-performance__section">
              <h3>{{ t('articles.performance.trend') }}</h3>
              <AdminEChart v-if="detail.trend.length" :option="trendOption" height="260px" />
              <AdminEmptyState
                v-else
                :title="t('contentPerformance.trend.emptyTitle')"
                :description="t('contentPerformance.trend.emptyDescription')" />
            </section>

            <section class="article-performance__section">
              <h3>{{ t('articles.performance.channels') }}</h3>
              <div class="article-performance__channels">
                <div v-for="channel in channels" :key="channel.key" class="article-performance__channel">
                  <div>
                    <span>{{ channel.label }}</span>
                    <strong>{{ channel.rate }}</strong>
                  </div>
                  <small>{{ channel.detail }}</small>
                </div>
              </div>
            </section>
          </div>

          <section class="article-performance__section article-performance__targets">
            <h3>{{ t('articles.performance.targetsTitle') }}</h3>
            <a-table
              v-if="displayTargets.length"
              :data="displayTargets"
              :columns="targetColumns"
              :pagination="false"
              row-key="rowKey"
              size="small">
              <template #target="{ record }">
                <div class="article-performance__target">
                  <a-tag size="small">
                    {{ record.targetType === 'series' ? t('contentPerformance.targets.series') : t('contentPerformance.targets.article') }}
                  </a-tag>
                  <a-link
                    v-if="record.targetType === 'article'"
                    :title="targetLabel(record)"
                    @click="openTarget(record.targetId)">
                    {{ targetLabel(record) }}
                  </a-link>
                  <span v-else :title="targetLabel(record)">{{ targetLabel(record) }}</span>
                  <a-tag
                    v-if="targetQuality(record)"
                    size="small"
                    :color="targetQuality(record)?.color">
                    {{ targetQuality(record)?.label }}
                  </a-tag>
                </div>
              </template>
              <template #placement="{ record }">
                <span class="admin-muted-cell">{{ placementLabel(record.placement) }}</span>
              </template>
              <template #clicks="{ record }">{{ formatNumber(record.clicks) }}</template>
              <template #clickRate="{ record }">{{ formatPercent(record.clickRate) }}</template>
            </a-table>
            <AdminEmptyState
              v-else
              :title="t('contentPerformance.targets.empty')"
              :description="t('articles.performance.targetsEmptyHint')" />
          </section>
        </div>
      </template>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type {
  ContentAnalyticsRange,
  ContentArticleAnalyticsDetail,
  ContentContinuationTarget
} from '@stellar-beacon/api-contract'

import {
  apiErrorMessage,
  getAdminContentArticleAnalytics,
  listAdminContinuationTargets
} from '@/api/http'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import { t } from '@/i18n'
import { useThemeStore } from '@/stores/theme'
import { chartSeriesColor } from '@/utils/chart-theme'
import { formatNumber } from '@/utils/format'

const MIN_COMPLETION_SESSIONS = 30
const LOW_COMPLETION_RATE = 35
const MIN_CONTINUATION_IMPRESSIONS = 50
const LOW_CONTINUATION_RATE = 2
const MIN_CHANNEL_IMPRESSIONS = 30
const CHANNEL_RATE_GAP = 1.5
const MIN_TARGET_IMPRESSIONS = 30
const LOW_TARGET_CLICK_RATE = 1

type DiagnosticTone = 'info' | 'warning'
interface PerformanceDiagnostic {
  key: string
  title: string
  description: string
  tone: DiagnosticTone
}

const props = withDefaults(defineProps<{
  articleId: number
  articleStatus?: number
  autoExpand?: boolean
}>(), {
  articleStatus: 1,
  autoExpand: false
})

const router = useRouter()
const theme = useThemeStore()
const range = ref<ContentAnalyticsRange>('30d')
const expanded = ref(props.autoExpand)
const detail = ref<ContentArticleAnalyticsDetail | null>(null)
const targets = ref<ContentContinuationTarget[]>([])
const loading = ref(false)
const errorMessage = ref('')
let loadVersion = 0

const stats = computed(() => {
  if (!detail.value) return []
  const overview = detail.value.overview
  return [
    { label: t('contentPerformance.stats.views'), value: formatNumber(overview.views), caption: t('contentPerformance.stats.viewsCaption') },
    { label: t('contentPerformance.stats.uniqueReaders'), value: formatNumber(overview.uniqueReaders), caption: t('contentPerformance.stats.uniqueReadersCaption') },
    { label: t('contentPerformance.table.effectiveSessions'), value: formatNumber(overview.effectiveSessions), caption: t('articles.performance.effectiveSessionsCaption') },
    { label: t('contentPerformance.stats.avgActiveTime'), value: formatDuration(overview.avgActiveMs), caption: t('contentPerformance.stats.avgActiveTimeCaption') },
    { label: t('contentPerformance.stats.completionRate'), value: formatPercent(overview.completionRate), caption: t('contentPerformance.stats.completionRateCaption') },
    { label: t('contentPerformance.table.continuationRate'), value: formatPercent(overview.continuation.continuationRate), caption: t('articles.performance.continuationCaption') }
  ]
})

const diagnostics = computed<PerformanceDiagnostic[]>(() => {
  if (!detail.value) return []
  const overview = detail.value.overview
  const continuation = overview.continuation
  const totalImpressions = continuation.seriesImpressions + continuation.relatedImpressions

  if (Number(props.articleStatus) !== 1) {
    return [diagnostic('private', 'info')]
  }
  if (overview.views === 0 && overview.effectiveSessions === 0 && totalImpressions === 0) {
    return [diagnostic('noData', 'info')]
  }

  const result: PerformanceDiagnostic[] = []
  const noExposure = overview.effectiveSessions >= 20 && totalImpressions === 0
  if (noExposure) {
    result.push(diagnostic('noExposure', 'warning'))
  } else if (overview.effectiveSessions < 20 || totalImpressions < MIN_CONTINUATION_IMPRESSIONS) {
    result.push(diagnostic('sample', 'info'))
  }
  if (overview.effectiveSessions >= MIN_COMPLETION_SESSIONS && overview.completionRate < LOW_COMPLETION_RATE) {
    result.push(diagnostic('lowCompletion', 'warning'))
  }
  if (totalImpressions >= MIN_CONTINUATION_IMPRESSIONS && continuation.continuationRate < LOW_CONTINUATION_RATE) {
    result.push(diagnostic('lowContinuation', 'warning'))
  }

  if (continuation.seriesImpressions >= MIN_CHANNEL_IMPRESSIONS && continuation.relatedImpressions >= MIN_CHANNEL_IMPRESSIONS) {
    if (continuation.seriesClickRate + CHANNEL_RATE_GAP <= continuation.relatedClickRate) {
      result.push(diagnostic('seriesWeak', 'warning'))
    } else if (continuation.relatedClickRate + CHANNEL_RATE_GAP <= continuation.seriesClickRate) {
      result.push(diagnostic('relatedWeak', 'warning'))
    }
  }

  if (targets.value.some((target) => target.moduleImpressions >= MIN_TARGET_IMPRESSIONS && target.clickRate < LOW_TARGET_CLICK_RATE)) {
    result.push(diagnostic('targetLow', 'warning'))
  }
  return result
})

const channels = computed(() => {
  const continuation = detail.value?.overview.continuation
  if (!continuation) return []
  return [
    {
      key: 'series',
      label: t('contentPerformance.continuation.series'),
      rate: formatPercent(continuation.seriesClickRate),
      detail: t('contentPerformance.continuation.detail', {
        clicks: formatNumber(continuation.seriesClicks),
        impressions: formatNumber(continuation.seriesImpressions)
      })
    },
    {
      key: 'related',
      label: t('contentPerformance.continuation.related'),
      rate: formatPercent(continuation.relatedClickRate),
      detail: t('contentPerformance.continuation.detail', {
        clicks: formatNumber(continuation.relatedClicks),
        impressions: formatNumber(continuation.relatedImpressions)
      })
    }
  ]
})

const trendOption = computed(() => {
  const rows = detail.value?.trend || []
  const monthly = range.value === '12m'
  const viewsColor = chartSeriesColor(theme.theme, 0)
  const completionColor = chartSeriesColor(theme.theme, 1)
  const continuationColor = chartSeriesColor(theme.theme, 2)
  return {
    tooltip: { trigger: 'axis' },
    legend: { top: 0, right: 0 },
    grid: { left: 4, right: 20, top: 36, bottom: 2, containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: rows.map((item) => monthly ? item.period : item.period.slice(5))
    },
    yAxis: [
      { type: 'value', minInterval: 1 },
      { type: 'value', min: 0, max: 100, axisLabel: { formatter: '{value}%' } }
    ],
    series: [
      { name: t('contentPerformance.chart.views'), type: 'line', smooth: true, showSymbol: false, lineStyle: { color: viewsColor, width: 2.4 }, itemStyle: { color: viewsColor }, data: rows.map((item) => item.views) },
      { name: t('contentPerformance.chart.completionRate'), type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, lineStyle: { color: completionColor, width: 2.2 }, itemStyle: { color: completionColor }, data: rows.map((item) => item.completionRate) },
      { name: t('contentPerformance.table.continuationRate'), type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, lineStyle: { color: continuationColor, width: 2.2 }, itemStyle: { color: continuationColor }, data: rows.map((item) => item.continuation.continuationRate) }
    ]
  }
})

const displayTargets = computed(() => targets.value.slice(0, 5))
const targetColumns = computed(() => [
  { title: t('contentPerformance.targets.target'), dataIndex: 'targetTitle', slotName: 'target', width: 310 },
  { title: t('contentPerformance.targets.placement'), dataIndex: 'placement', slotName: 'placement', width: 140 },
  { title: t('contentPerformance.targets.clicks'), dataIndex: 'clicks', slotName: 'clicks', width: 90 },
  { title: t('contentPerformance.targets.clickRate'), dataIndex: 'clickRate', slotName: 'clickRate', width: 110 }
])

async function load(): Promise<void> {
  if (!Number.isInteger(props.articleId) || props.articleId <= 0) return
  const version = ++loadVersion
  loading.value = true
  errorMessage.value = ''
  try {
    const [articleDetail, targetPage] = await Promise.all([
      getAdminContentArticleAnalytics(props.articleId, range.value),
      listAdminContinuationTargets(range.value, 'clicks', 1, 20, props.articleId)
    ])
    if (version !== loadVersion) return
    detail.value = articleDetail
    targets.value = targetPage.items
  } catch (error) {
    if (version !== loadVersion) return
    errorMessage.value = apiErrorMessage(error, t('articles.performance.loadFailed'))
  } finally {
    if (version === loadVersion) loading.value = false
  }
}

function diagnostic(key: string, tone: DiagnosticTone): PerformanceDiagnostic {
  return {
    key,
    tone,
    title: t(`articles.performance.diagnostics.${key}.title`),
    description: t(`articles.performance.diagnostics.${key}.description`)
  }
}

function toggleExpanded(): void {
  expanded.value = !expanded.value
}

function expandAndScroll(): void {
  expanded.value = true
  void nextTick(() => scrollToPanel())
}

function scrollToPanel(): void {
  document.getElementById('article-performance')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function openTarget(targetId: number): void {
  if (!Number.isInteger(targetId) || targetId <= 0) return
  void router.push({ path: `/articles/${targetId}`, query: { panel: 'performance' } })
}

function targetQuality(target: ContentContinuationTarget): { label: string; color: string } | null {
  if (target.moduleImpressions < MIN_TARGET_IMPRESSIONS) {
    return { label: t('articles.performance.targetSample'), color: 'gray' }
  }
  if (target.clickRate < LOW_TARGET_CLICK_RATE) {
    return { label: t('articles.performance.targetLowTag'), color: 'orange' }
  }
  return null
}

function targetLabel(target: ContentContinuationTarget): string {
  return target.targetTitle || `#${target.targetId}`
}

function placementLabel(value: string): string {
  const key = `contentPerformance.targets.placements.${value}`
  const label = t(key)
  return label === key ? value : label
}

function formatPercent(value: unknown): string {
  return `${Number(value || 0).toFixed(2)}%`
}

function formatDuration(value: unknown): string {
  const ms = Number(value || 0)
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  const seconds = Math.round(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remain = seconds % 60
  return remain ? `${minutes}m ${remain}s` : `${minutes}m`
}

watch(() => [props.articleId, range.value], () => void load(), { immediate: true })
watch(() => props.autoExpand, (value) => {
  if (!value) return
  expanded.value = true
  void nextTick(() => scrollToPanel())
})

onMounted(() => {
  if (props.autoExpand) void nextTick(() => scrollToPanel())
})

defineExpose({ expandAndScroll })
</script>

<style scoped>
.article-performance {
  margin-bottom: 16px;
}

.article-performance__card {
  overflow: hidden;
}

.article-performance__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.article-performance__header h2 {
  margin: 0 0 4px;
  font-size: 17px;
}

.article-performance__header p {
  margin: 0;
  color: var(--admin-muted);
  font-size: 12px;
}

.article-performance__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.article-performance__skeleton {
  height: 126px;
}

.article-performance__stats {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px;
}

.article-performance__stat {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: 10px;
  background: var(--admin-panel-soft);
}

.article-performance__stat span,
.article-performance__stat strong,
.article-performance__stat small {
  display: block;
}

.article-performance__stat span,
.article-performance__stat small {
  color: var(--admin-muted);
  font-size: 11px;
}

.article-performance__stat strong {
  margin: 5px 0 3px;
  font-size: 20px;
  font-variant-numeric: tabular-nums;
}

.article-performance__diagnostics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 12px;
}

.article-performance__diagnostic {
  display: grid;
  gap: 3px;
  padding: 10px 12px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  font-size: 12px;
}

.article-performance__diagnostic strong {
  font-size: 12px;
}

.article-performance__diagnostic span {
  color: var(--admin-muted);
}

.article-performance__diagnostic.is-warning {
  border-color: rgb(217 128 63 / 35%);
  background: rgb(217 128 63 / 8%);
}

.article-performance__diagnostic.is-info {
  background: var(--admin-panel-soft);
}

.article-performance__details {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--admin-border);
}

.article-performance__detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(260px, 1fr);
  gap: 16px;
}

.article-performance__section {
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--admin-border);
  border-radius: 10px;
  background: var(--admin-surface);
}

.article-performance__section h3 {
  margin: 0 0 12px;
  font-size: 13px;
}

.article-performance__channels {
  display: grid;
  gap: 10px;
}

.article-performance__channel {
  display: grid;
  gap: 6px;
  padding: 14px;
  border-radius: 8px;
  background: var(--admin-panel-soft);
}

.article-performance__channel > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.article-performance__channel span,
.article-performance__channel small {
  color: var(--admin-muted);
  font-size: 12px;
}

.article-performance__channel strong {
  font-size: 22px;
}

.article-performance__targets {
  margin-top: 16px;
}

.article-performance__target {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}

.article-performance__target .arco-link,
.article-performance__target span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .article-performance__stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .article-performance__detail-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .article-performance__header {
    flex-direction: column;
  }

  .article-performance__actions {
    justify-content: flex-start;
  }

  .article-performance__stats,
  .article-performance__diagnostics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 480px) {
  .article-performance__stats,
  .article-performance__diagnostics {
    grid-template-columns: 1fr;
  }
}
</style>