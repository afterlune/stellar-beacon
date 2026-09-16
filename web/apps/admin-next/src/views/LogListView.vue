<template>
  <section class="admin-page">
    <AdminPageHeader :title="config.title" :description="config.description">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="config.placeholder"
          allow-clear
          @search="reload" />
        <a-popconfirm
          v-if="mode === 'job'"
          :content="t('logs.jobLogs.cleanConfirm')"
          @ok="clean">
          <a-button status="danger" :loading="cleaning">{{ t('logs.jobLogs.clean') }}</a-button>
        </a-popconfirm>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-tag v-if="jobScopeLabel" color="arcoblue">{{ jobScopeLabel }}</a-tag>
          <a-select
            v-if="mode === 'job'"
            v-model="groupFilter"
            :placeholder="t('logs.common.allGroups')"
            allow-clear
            style="width: 150px"
            @change="reload">
            <a-option v-for="group in groupOptions" :key="group" :value="group">{{ group }}</a-option>
          </a-select>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">{{ t('logs.common.clearSearch') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('pagination.total', { total }) }}</span>
          <a-dropdown trigger="click" position="br">
            <a-button size="small">
              <template #icon><IconSettings /></template>
              {{ t('logs.common.columnSettings') }}
            </a-button>
            <template #content>
              <div class="admin-column-settings">
                <div class="admin-column-settings-head">
                  <span>{{ t('logs.common.visibleColumns') }}</span>
                  <a-button type="text" size="mini" @click="columnPrefs.reset">{{ t('logs.common.showAll') }}</a-button>
                </div>
                <a-checkbox
                  v-for="column in config.columns"
                  :key="column.dataIndex"
                  :model-value="columnPrefs.isVisible(column.dataIndex)"
                  :disabled="column.dataIndex === 'actions'"
                  @change="(value) => columnPrefs.toggle(column.dataIndex, Boolean(value))">
                  {{ column.title }}
                </a-checkbox>
              </div>
            </template>
          </a-dropdown>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('logs.common.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('logs.common.pageCount', { count: records.length })" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">{{ t('logs.common.batchDelete') }}</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="records"
          :columns="tableColumns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #method="{ record }">
            <a-tag :color="methodColor(record.requestMethod || record.optMethod)">
              {{ formatCell(record.requestMethod || record.optMethod) }}
            </a-tag>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.status) === 1 ? 'failed' : 'success'" />
          </template>
          <template #content="{ record }">
            <span
              class="ellipsis admin-log-message"
              :title="String(record.exceptionInfo || record.jobMessage || '')">
              {{ formatCell(record.exceptionInfo || record.jobMessage) }}
            </span>
          </template>
          <template #time="{ record }">
            <span class="admin-cell-nowrap">{{ formatDateTime(record.createTime || record.startTime) }}</span>
          </template>
          <template #formatted="{ record, column }">{{ formatCell(record[column.dataIndex]) }}</template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="showDetail(record)">{{ t('common.detail') }}</a-button>
              <a-popconfirm
                :content="t('logs.common.deleteConfirm')"
                @ok="deleteLog(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconHistory"
              :title="keywords.trim() ? t('logs.common.emptyFiltered') : t('logs.common.empty')"
              :description="keywords.trim() ? t('logs.common.emptyFilteredHint') : config.emptyHint">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('logs.common.clearSearch') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="detailVisible" :title="t('logs.common.detailTitle')" width="760px" :footer="false">
      <div v-if="detail" class="log-detail">
        <div class="log-detail-head">
          <span class="log-detail-id">#{{ detail.id }}</span>
          <a-button size="mini" @click="copyDetail">
            <template #icon><IconCopy /></template>
            {{ t('logs.common.copyRaw') }}
          </a-button>
        </div>
        <a-descriptions :column="2" size="small" bordered>
          <a-descriptions-item v-for="field in detailFields" :key="field.label" :label="field.label" :span="field.span">
            <span :class="{ 'admin-mono-cell': field.mono }">{{ field.value }}</span>
          </a-descriptions-item>
        </a-descriptions>

        <template v-for="section in detailSections" :key="section.label">
          <h4 class="log-detail-section">{{ section.label }}</h4>
          <pre class="detail">{{ section.value }}</pre>
        </template>
      </div>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconCopy, IconHistory, IconRefresh, IconSettings } from '@arco-design/web-vue/es/icon'
import { useRoute } from 'vue-router'

import {
  apiErrorMessage,
  cleanAdminJobLogs,
  deleteAdminLogs,
  listAdminJobLogGroups,
  listAdminPage
} from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useColumnPrefs, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { copyText } from '@/utils/clipboard'
import { formatCell, formatDateTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

type LogMode = 'operation' | 'exception' | 'job'

interface LogRecord {
  id: number
  [key: string]: unknown
}

interface DetailField {
  label: string
  value: string
  mono?: boolean
  span?: number
}

interface LogColumn {
  title: string
  dataIndex: string
  slotName?: string
  width?: number
  minWidth?: number
  ellipsis?: boolean
  tooltip?: boolean
}

interface LogConfig {
  title: string
  description: string
  placeholder: string
  endpoint: string
  emptyHint: string
  columns: LogColumn[]
}

const props = defineProps<{ mode: LogMode }>()
const route = useRoute()

const mode = computed(() => props.mode)

// 列定义与页头文案都要跟着语言切换，因此放在 computed 里求值。
const configs = computed<Record<LogMode, LogConfig>>(() => ({
  operation: {
    title: t('logs.operation.title'),
    description: t('logs.operation.description'),
    placeholder: t('logs.operation.searchPlaceholder'),
    endpoint: 'admin/logs/operations',
    emptyHint: t('logs.operation.emptyHint'),
    columns: [
      { title: t('logs.operation.colModule'), dataIndex: 'optModule', width: 140 },
      { title: t('common.type'), dataIndex: 'optType', width: 130 },
      { title: 'URI', dataIndex: 'optUri', minWidth: 200, ellipsis: true, tooltip: true },
      { title: t('logs.common.colMethod'), dataIndex: 'requestMethod', slotName: 'method', width: 100 },
      { title: t('logs.operation.colUser'), dataIndex: 'nickname', width: 168, ellipsis: true, tooltip: true },
      { title: t('common.time'), dataIndex: 'createTime', slotName: 'time', width: 184 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  },
  exception: {
    title: t('logs.exception.title'),
    description: t('logs.exception.description'),
    placeholder: t('logs.exception.searchPlaceholder'),
    endpoint: 'admin/logs/exceptions',
    emptyHint: t('logs.exception.emptyHint'),
    columns: [
      { title: 'URI', dataIndex: 'optUri', minWidth: 200, ellipsis: true, tooltip: true },
      { title: t('logs.common.colMethod'), dataIndex: 'requestMethod', slotName: 'method', width: 100 },
      { title: t('common.description'), dataIndex: 'optDesc', minWidth: 160, ellipsis: true, tooltip: true },
      { title: t('logs.exception.colException'), dataIndex: 'exceptionInfo', slotName: 'content', ellipsis: true, tooltip: true, minWidth: 200 },
      { title: t('common.time'), dataIndex: 'createTime', slotName: 'time', width: 184 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  },
  job: {
    title: t('logs.jobLogs.title'),
    description: t('logs.jobLogs.description'),
    placeholder: t('logs.common.searchJobName'),
    endpoint: 'admin/logs/jobs',
    emptyHint: t('logs.jobLogs.emptyHint'),
    columns: [
      { title: t('logs.jobLogs.colJob'), dataIndex: 'jobName', minWidth: 170 },
      { title: t('logs.jobLogs.colJobGroup'), dataIndex: 'jobGroup', width: 120 },
      { title: t('logs.common.colInvokeTarget'), dataIndex: 'invokeTarget', ellipsis: true, tooltip: true, minWidth: 180 },
      { title: t('common.status'), dataIndex: 'status', slotName: 'status', width: 96 },
      { title: t('common.time'), dataIndex: 'startTime', slotName: 'time', width: 184 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  }
}))

const config = computed(() => configs.value[props.mode])

const viewKey = `log-${props.mode}`
const columnPrefs = useColumnPrefs(viewKey, config.value.columns.map((column) => column.dataIndex))

// Give every column an explicit slot so no cell leaks a raw RFC3339 timestamp.
const tableColumns = computed(() => config.value.columns
  .filter((column) => columnPrefs.isVisible(column.dataIndex))
  .map((column) => ({
    ...column,
    slotName: 'slotName' in column ? column.slotName : 'formatted'
  })))

const keywords = ref('')
const groupFilter = ref<string | undefined>(undefined)
const groupOptions = ref<string[]>([])
const selectedKeys = ref<number[]>([])
const cleaning = ref(false)
const batchDeleting = ref(false)
const detailVisible = ref(false)
const detail = ref<LogRecord | null>(null)

const {
  items: records,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<LogRecord>(
  ({ current: page, pageSize: size, signal }) => {
    // 任务日志按任务名过滤，其余日志按关键词过滤。
    const filterKey = props.mode === 'job' ? 'jobName' : 'keywords'
    const params: Record<string, string | number> = {
      current: page,
      size,
      [filterKey]: keywords.value.trim()
    }
    if (props.mode === 'job') {
      const jobId = Number(route.params.quartzId)
      if (Number.isInteger(jobId) && jobId > 0) params.jobId = jobId
      if (groupFilter.value) params.jobGroup = groupFilter.value
    }
    return listAdminPage<LogRecord>(config.value.endpoint, params, { signal })
  },
  { pageSize: readStoredPageSize(viewKey), fallbackMessage: t('logs.common.loadFailed'), immediate: false }
)

useStoredPageSize(viewKey, pageSize)
useQueryFilters([
  { key: props.mode === 'job' ? 'jobName' : 'keywords', ref: keywords, debounce: true },
  { key: 'jobGroup', ref: groupFilter },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

onMounted(() => {
  void load()
  if (props.mode === 'job') void loadGroupOptions()
})

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const selectedIds = computed(() => selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))
const jobScopeLabel = computed(() => {
  if (props.mode !== 'job') return ''
  const jobId = Number(route.params.quartzId)
  return Number.isInteger(jobId) && jobId > 0 ? t('logs.jobLogs.scopeLabel', { id: jobId }) : ''
})

const detailFields = computed<DetailField[]>(() => {
  const record = detail.value
  if (!record) return []
  const hidden = new Set(['id', 'requestParam', 'responseData', 'exceptionInfo', 'jobMessage'])
  return Object.entries(record)
    .filter(([key, value]) => !hidden.has(key) && value !== null && value !== undefined && value !== '')
    .map(([key, value]) => ({
      label: fieldLabel(key),
      value: formatCell(value),
      mono: /uri|url|method|target/i.test(key),
      span: /desc|module|info/i.test(key) ? 2 : 1
    }))
})

const detailSections = computed(() => {
  const record = detail.value
  if (!record) return []
  return [
    { label: t('logs.common.sectionRequestParam'), raw: record.requestParam },
    { label: t('logs.common.sectionResponse'), raw: record.responseData },
    { label: t('logs.common.sectionException'), raw: record.exceptionInfo },
    { label: t('logs.common.sectionJobOutput'), raw: record.jobMessage }
  ]
    .filter((section) => section.raw !== null && section.raw !== undefined && section.raw !== '')
    .map((section) => ({ label: section.label, value: pretty(section.raw) }))
})

watch(() => route.params.quartzId, () => {
  if (props.mode === 'job') void reload()
})

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

function clearSelection(): void {
  selectedKeys.value = []
}

function changePage(page: number): void {
  clearSelection()
  gotoPage(page)
}

function changePageSize(size: number): void {
  clearSelection()
  applyPageSize(size)
}

/** 分组接口只用于下拉选项，失败时静默降级为「不提供分组筛选」。 */
async function loadGroupOptions(): Promise<void> {
  try {
    groupOptions.value = await listAdminJobLogGroups()
  } catch {
    groupOptions.value = []
  }
}

async function batchDelete(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0) return
  batchDeleting.value = true
  try {
    await deleteAdminLogs(props.mode, ids)
    Message.success(t('logs.common.batchDeleted', { count: ids.length }))
    clearSelection()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.common.batchDeleteFailed')))
  } finally {
    batchDeleting.value = false
  }
}

function showDetail(record: LogRecord): void {
  detail.value = record
  detailVisible.value = true
}

function copyDetail(): void {
  if (!detail.value) return
  void copyText(JSON.stringify(detail.value, null, 2), {
    success: t('logs.common.copyRawSuccess'),
    failure: t('common.copyFailed')
  })
}

async function deleteLog(id: unknown): Promise<void> {
  const logId = Number(id)
  if (!Number.isInteger(logId) || logId <= 0) return
  try {
    await deleteAdminLogs(props.mode, [logId])
    if (records.value.length === 1 && current.value > 1) current.value -= 1
    Message.success(t('logs.common.deleted'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.common.deleteFailed')))
  }
}

async function clean(): Promise<void> {
  cleaning.value = true
  try {
    await cleanAdminJobLogs()
    Message.success(t('logs.jobLogs.cleaned'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.jobLogs.cleanFailed')))
  } finally {
    cleaning.value = false
  }
}

// 详情字段名来自后端字段，这里只翻译已知字段；语言切换后重新计算。
const LABELS = computed<Record<string, string>>(() => ({
  optModule: t('logs.operation.fieldModule'),
  optType: t('logs.operation.fieldType'),
  optUri: t('logs.common.fieldRequestUri'),
  optDesc: t('logs.operation.fieldDesc'),
  requestMethod: t('logs.common.fieldRequestMethod'),
  optMethod: t('logs.common.fieldRequestMethod'),
  nickname: t('logs.operation.fieldUser'),
  ipAddress: t('logs.common.fieldIp'),
  ipSource: t('logs.common.fieldIpSource'),
  createTime: t('logs.common.fieldOccurredAt'),
  startTime: t('logs.common.fieldStartTime'),
  endTime: t('logs.common.fieldEndTime'),
  status: t('logs.common.fieldRunStatus'),
  jobName: t('logs.common.colJobName'),
  jobGroup: t('logs.common.colJobGroup'),
  invokeTarget: t('logs.common.colInvokeTarget'),
  elapsed: t('logs.common.fieldElapsed')
}))

function fieldLabel(key: string): string {
  return LABELS.value[key] || key
}

function pretty(value: unknown): string {
  const text = String(value ?? '')
  const trimmed = text.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2)
    } catch {
      // Not valid JSON: fall through and show the raw text.
    }
  }
  return text
}

function methodColor(method: unknown): string {
  switch (String(method || '').toUpperCase()) {
    case 'GET': return 'green'
    case 'POST': return 'arcoblue'
    case 'PUT': return 'orange'
    case 'DELETE': return 'red'
    case 'PATCH': return 'purple'
    default: return 'gray'
  }
}
</script>

<style scoped>
.ellipsis {
  display: block;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-log-message {
  font-family: ui-monospace, "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
}

.log-detail {
  display: grid;
  gap: 14px;
}

.log-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.log-detail-id {
  color: var(--admin-subtle);
  font-family: ui-monospace, "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
}

.log-detail-section {
  margin: 4px 0 0;
  color: var(--admin-ink-strong);
  font-size: 13px;
  font-weight: 680;
}

.detail {
  max-height: 40vh;
  overflow: auto;
  margin: 0;
}
</style>
