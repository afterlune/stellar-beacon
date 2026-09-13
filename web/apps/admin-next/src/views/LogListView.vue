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
          content="确定清空全部任务日志吗？此操作不可撤销。"
          @ok="clean">
          <a-button status="danger" :loading="cleaning">清空任务日志</a-button>
        </a-popconfirm>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          刷新
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
            placeholder="全部分组"
            allow-clear
            style="width: 150px"
            @change="reload">
            <a-option v-for="group in groupOptions" :key="group" :value="group">{{ group }}</a-option>
          </a-select>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">清空搜索</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 条记录</span>
          <a-dropdown trigger="click" position="br">
            <a-button size="small">
              <template #icon><IconSettings /></template>
              列设置
            </a-button>
            <template #content>
              <div class="admin-column-settings">
                <div class="admin-column-settings-head">
                  <span>显示列</span>
                  <a-button type="text" size="mini" @click="columnPrefs.reset">全部显示</a-button>
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

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="日志加载失败" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="`本页 ${records.length} 条`" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">批量删除</a-button>
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
              <a-button type="text" size="small" @click="showDetail(record)">详情</a-button>
              <a-popconfirm
                content="确定删除这条日志吗？删除后无法恢复。"
                @ok="deleteLog(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconHistory"
              :title="keywords.trim() ? '没有匹配的日志' : '暂无日志'"
              :description="keywords.trim() ? '换个关键词再试一次。' : config.emptyHint">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="detailVisible" title="日志详情" width="760px" :footer="false">
      <div v-if="detail" class="log-detail">
        <div class="log-detail-head">
          <span class="log-detail-id">#{{ detail.id }}</span>
          <a-button size="mini" @click="copyDetail">
            <template #icon><IconCopy /></template>
            复制原始数据
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

const props = defineProps<{ mode: LogMode }>()
const route = useRoute()

const mode = computed(() => props.mode)

const configs = {
  operation: {
    title: '操作日志',
    description: '查看后台操作记录。',
    placeholder: '搜索模块或描述',
    endpoint: 'admin/logs/operations',
    emptyHint: '后台的写操作会自动记录在这里。',
    columns: [
      { title: '模块', dataIndex: 'optModule', width: 140 },
      { title: '类型', dataIndex: 'optType', width: 130 },
      { title: 'URI', dataIndex: 'optUri', minWidth: 200, ellipsis: true, tooltip: true },
      { title: '方法', dataIndex: 'requestMethod', slotName: 'method', width: 100 },
      { title: '用户', dataIndex: 'nickname', width: 168, ellipsis: true, tooltip: true },
      { title: '时间', dataIndex: 'createTime', slotName: 'time', width: 184 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  },
  exception: {
    title: '异常日志',
    description: '查看后台异常记录和请求信息。',
    placeholder: '搜索请求 URI 或描述',
    endpoint: 'admin/logs/exceptions',
    emptyHint: '接口抛出未捕获异常时会记录在这里。',
    columns: [
      { title: 'URI', dataIndex: 'optUri', minWidth: 200, ellipsis: true, tooltip: true },
      { title: '方法', dataIndex: 'requestMethod', slotName: 'method', width: 100 },
      { title: '描述', dataIndex: 'optDesc', minWidth: 160, ellipsis: true, tooltip: true },
      { title: '异常', dataIndex: 'exceptionInfo', slotName: 'content', ellipsis: true, tooltip: true, minWidth: 200 },
      { title: '时间', dataIndex: 'createTime', slotName: 'time', width: 184 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  },
  job: {
    title: '任务日志',
    description: '回看后台任务的执行结果与耗时。',
    placeholder: '搜索任务名',
    endpoint: 'admin/logs/jobs',
    emptyHint: '定时任务每次执行都会生成一条日志。',
    columns: [
      { title: '任务', dataIndex: 'jobName', minWidth: 170 },
      { title: '任务组', dataIndex: 'jobGroup', width: 120 },
      { title: '调用目标', dataIndex: 'invokeTarget', ellipsis: true, tooltip: true, minWidth: 180 },
      { title: '状态', dataIndex: 'status', slotName: 'status', width: 96 },
      { title: '时间', dataIndex: 'startTime', slotName: 'time', width: 184 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  }
} as const

const config = computed(() => configs[props.mode])

const viewKey = `log-${props.mode}`
const columnPrefs = useColumnPrefs(viewKey, configs[props.mode].columns.map((column) => column.dataIndex))

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
    return listAdminPage<LogRecord>(configs[props.mode].endpoint, params, { signal })
  },
  { pageSize: readStoredPageSize(viewKey), fallbackMessage: '日志加载失败', immediate: false }
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
  return Number.isInteger(jobId) && jobId > 0 ? `任务 #${jobId}` : ''
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
    { label: '请求参数', raw: record.requestParam },
    { label: '响应数据', raw: record.responseData },
    { label: '异常信息', raw: record.exceptionInfo },
    { label: '任务输出', raw: record.jobMessage }
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
    Message.success(`已删除 ${ids.length} 条日志`)
    clearSelection()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '批量删除失败'))
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
    success: '日志原始数据已复制',
    failure: '浏览器不允许自动复制，请手动选择文本'
  })
}

async function deleteLog(id: unknown): Promise<void> {
  const logId = Number(id)
  if (!Number.isInteger(logId) || logId <= 0) return
  try {
    await deleteAdminLogs(props.mode, [logId])
    if (records.value.length === 1 && current.value > 1) current.value -= 1
    Message.success('日志已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '日志删除失败'))
  }
}

async function clean(): Promise<void> {
  cleaning.value = true
  try {
    await cleanAdminJobLogs()
    Message.success('任务日志已清空')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务日志清理失败'))
  } finally {
    cleaning.value = false
  }
}

const LABELS: Record<string, string> = {
  optModule: '操作模块',
  optType: '操作类型',
  optUri: '请求地址',
  optDesc: '操作描述',
  requestMethod: '请求方法',
  optMethod: '请求方法',
  nickname: '操作用户',
  ipAddress: 'IP 地址',
  ipSource: 'IP 归属',
  createTime: '发生时间',
  startTime: '开始时间',
  endTime: '结束时间',
  status: '执行状态',
  jobName: '任务名称',
  jobGroup: '任务分组',
  invokeTarget: '调用目标',
  elapsed: '耗时'
}

function fieldLabel(key: string): string {
  return LABELS[key] || key
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
