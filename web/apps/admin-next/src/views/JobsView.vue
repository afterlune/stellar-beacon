<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('logs.jobs.title')" :description="t('logs.jobs.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('logs.common.searchJobName')"
          allow-clear
          @search="reload" />
        <a-button type="primary" @click="openEditor()">
          <template #icon><IconPlus /></template>
          {{ t('common.create') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-radio-group v-model="statusFilter" type="button" size="small" @change="reload">
            <a-radio value="all">{{ t('common.all') }}</a-radio>
            <a-radio value="1">{{ t('logs.jobs.statusNormal') }}</a-radio>
            <a-radio value="0">{{ t('logs.jobs.statusPaused') }}</a-radio>
          </a-radio-group>
          <a-select
            v-model="groupFilter"
            :placeholder="t('logs.common.allGroups')"
            allow-clear
            style="width: 150px"
            @change="reload">
            <a-option v-for="group in groupOptions" :key="group" :value="group">{{ group }}</a-option>
          </a-select>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">{{ t('logs.jobs.resetFilters') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('logs.jobs.total', { total }) }}</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('logs.jobs.loadFailed')" @retry="load" />
      <a-alert v-if="runHint" type="info" closable @close="runHint = ''">{{ runHint }}</a-alert>

      <AdminBatchBar :count="selectedIds.length" :hint="t('logs.jobs.pageCount', { count: jobs.length })" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">{{ t('logs.common.batchDelete') }}</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="jobs"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #jobName="{ record }">
            <div class="job-name-cell">
              <span class="admin-title-cell">{{ record.jobName || t('logs.jobs.untitled') }}</span>
              <small v-if="record.remark" :title="String(record.remark)">{{ record.remark }}</small>
            </div>
          </template>
          <template #group="{ record }">
            <a-tag :color="groupColor(record.jobGroup)">{{ record.jobGroup || t('logs.jobs.defaultGroup') }}</a-tag>
          </template>
          <template #target="{ record }">
            <span class="admin-mono-cell ellipsis" :title="String(record.invokeTarget || '')">{{ record.invokeTarget || '—' }}</span>
          </template>
          <template #cron="{ record }">
            <span class="admin-mono-cell" :title="String(record.cronExpression || '')">{{ record.cronExpression || '—' }}</span>
          </template>
          <template #status="{ record }">
            <a-tooltip :content="Number(record.status) === 1 ? t('logs.jobs.pauseHint') : t('logs.jobs.resumeHint')">
              <a-switch
                :model-value="Number(record.status) === 1"
                :checked-value="true"
                :unchecked-value="false"
                :loading="isPending(record.id)"
                :disabled="pending.some((item) => item !== Number(record.id))"
                @change="changeStatus(record, $event)">
                <template #checked>{{ t('logs.jobs.statusNormal') }}</template>
                <template #unchecked>{{ t('logs.jobs.pause') }}</template>
              </a-switch>
            </a-tooltip>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">{{ t('common.edit') }}</a-button>
              <a-popconfirm v-if="record.canRunOnce" :content="t('logs.jobs.runOnceConfirm')" @ok="runOnce(record)">
                <a-button type="text" size="small" :loading="runningId === Number(record.id)">{{ t('logs.jobs.runOnce') }}</a-button>
              </a-popconfirm>
              <a-tooltip v-else :content="String(record.runOnceReason || t('logs.jobs.runOnceUnsupported'))">
                <a-button type="text" size="small" disabled>{{ t('logs.jobs.runOnce') }}</a-button>
              </a-tooltip>
              <a-popconfirm
                :content="t('logs.jobs.deleteConfirm', { name: record.jobName })"
                @ok="deleteJob(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconCalendarClock"
              :title="hasFilters ? t('logs.jobs.emptyFiltered') : t('logs.jobs.empty')"
              :description="hasFilters ? t('logs.jobs.emptyFilteredHint') : t('logs.jobs.emptyHint')">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">{{ t('logs.jobs.resetFilters') }}</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">{{ t('logs.jobs.create') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? t('logs.jobs.edit') : t('logs.jobs.create')"
      :ok-loading="saving"
      :ok-button-props="{ disabled: editorLoading }"
      :mask-closable="false"
      width="720px"
      @ok="saveEditor">
      <a-spin v-if="editorLoading" class="job-editor-loading" :tip="t('logs.jobs.loadingDetail')" />
      <a-form v-else :model="editor" layout="vertical">
        <div class="admin-form-grid">
          <a-form-item field="jobName" :label="t('logs.common.colJobName')" required>
            <a-input v-model="editor.jobName" maxlength="64" show-word-limit :placeholder="t('logs.jobs.namePlaceholder')" />
          </a-form-item>
          <a-form-item field="jobGroup" :label="t('logs.common.colJobGroup')" required>
            <a-input v-model="editor.jobGroup" maxlength="64" show-word-limit :placeholder="t('logs.jobs.groupPlaceholder')" />
            <!-- 建议分组只是把已有分组填进输入框，不新增输入控件（分组字段的
                 输入序号被 E2E 契约锁定，且 fill() 需要可写的 text input）。 -->
            <template v-if="editorGroupSuggestions.length" #help>
              <span class="job-group-hint">
                {{ t('logs.jobs.existingGroups') }}
                <a-button
                  v-for="group in editorGroupSuggestions"
                  :key="group"
                  type="text"
                  size="mini"
                  @click="editor.jobGroup = group">{{ group }}</a-button>
              </span>
            </template>
          </a-form-item>
        </div>
        <a-form-item field="invokeTarget" :label="t('logs.common.colInvokeTarget')" required>
          <a-input v-model="editor.invokeTarget" maxlength="500" :placeholder="t('logs.jobs.targetPlaceholder')" />
          <template #help>
            {{ t('logs.jobs.targetHelp') }}
          </template>
        </a-form-item>
        <a-form-item field="cronExpression" :label="t('logs.jobs.cronLabel')" required>
          <a-input v-model="editor.cronExpression" maxlength="255" :placeholder="t('logs.jobs.cronPlaceholder')" />
          <template #help>{{ t('logs.jobs.cronHelp') }}</template>
        </a-form-item>
        <div class="admin-form-grid">
          <a-form-item :label="t('logs.jobs.misfireLabel')">
            <a-select v-model="editor.misfirePolicy">
              <a-option :value="0">{{ t('logs.jobs.misfireDefault') }}</a-option>
              <a-option :value="1">{{ t('logs.jobs.misfireFireNow') }}</a-option>
              <a-option :value="2">{{ t('logs.jobs.misfireOnce') }}</a-option>
              <a-option :value="3">{{ t('logs.jobs.misfireAbandon') }}</a-option>
            </a-select>
            <template #help>{{ t('logs.jobs.misfireHelp') }}</template>
          </a-form-item>
          <a-form-item :label="t('logs.jobs.concurrentLabel')">
            <a-radio-group v-model="editor.concurrent">
              <a-radio :value="0">{{ t('logs.jobs.concurrentAllow') }}</a-radio>
              <a-radio :value="1">{{ t('logs.jobs.concurrentDeny') }}</a-radio>
            </a-radio-group>
            <template #help>{{ t('logs.jobs.concurrentHelp') }}</template>
          </a-form-item>
        </div>
        <a-form-item :label="t('common.status')">
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">{{ t('logs.jobs.statusNormal') }}</a-radio>
            <a-radio :value="0">{{ t('logs.jobs.pause') }}</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item :label="t('common.remark')">
          <a-textarea v-model="editor.remark" maxlength="500" show-word-limit :auto-size="{ minRows: 2, maxRows: 4 }" :placeholder="t('logs.jobs.remarkPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconCalendarClock, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteAdminJobs,
  getAdminJob,
  listAdminJobs,
  listAdminJobGroups,
  runAdminJob,
  saveAdminJob,
  updateAdminJobStatus
} from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { usePendingIds } from '@/composables/usePendingIds'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { formatDateTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminJob } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'jobs'

// 列标题要跟着语言切换，因此放在 computed 里求值。
const columns = computed(() => [
  { title: t('logs.common.colJobName'), dataIndex: 'jobName', slotName: 'jobName', minWidth: 200 },
  { title: t('logs.jobs.colGroup'), dataIndex: 'jobGroup', slotName: 'group', width: 118 },
  { title: t('logs.common.colInvokeTarget'), dataIndex: 'invokeTarget', slotName: 'target', ellipsis: true, tooltip: true, minWidth: 180 },
  { title: t('logs.jobs.colCron'), dataIndex: 'cronExpression', slotName: 'cron', width: 156 },
  { title: t('common.status'), dataIndex: 'status', slotName: 'status', width: 118 },
  { title: t('logs.jobs.colCreateTime'), dataIndex: 'createTime', slotName: 'time', width: 168 },
  { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 224 }
])

const keywords = ref('')
const statusFilter = ref<'all' | '0' | '1'>('all')
const groupFilter = ref<string | undefined>(undefined)
const groupOptions = ref<string[]>([])
const selectedKeys = ref<number[]>([])
const saving = ref(false)
const editorLoading = ref(false)
const runningId = ref(0)
const runHint = ref('')
const editorVisible = ref(false)
const batchDeleting = ref(false)
const editor = reactive<AdminJob>({
  id: 0,
  jobName: '',
  jobGroup: '',
  invokeTarget: '',
  cronExpression: '',
  misfirePolicy: 0,
  concurrent: 0,
  status: 1,
  remark: ''
})

const { pending, isPending, withPending } = usePendingIds()

const {
  items: rows,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminJob>(
  ({ current: page, pageSize: size, signal }) => listAdminJobs({
    current: page,
    size,
    jobName: keywords.value.trim(),
    jobGroup: groupFilter.value || ''
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('logs.jobs.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'status', ref: statusFilter },
  { key: 'jobGroup', ref: groupFilter },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

onMounted(() => void loadGroupOptions())

/**
 * 任务列表接口只按「名称 / 分组 / 状态」过滤，其中 `status=0` 在后端等于「不过滤」，
 * 因此状态页签仍在本地生效（否则无法只看已暂停任务），分组与名称走服务端。
 */
const jobs = computed(() => (statusFilter.value === 'all'
  ? rows.value
  : rows.value.filter((job) => Number(job.status) === Number(statusFilter.value))))

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() => Boolean(keywords.value.trim()) || statusFilter.value !== 'all' || Boolean(groupFilter.value))
const selectedIds = computed(() => selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))
const editorGroupSuggestions = computed(() => {
  const current = editor.jobGroup.trim()
  return groupOptions.value.filter((group) => group !== current).slice(0, 8)
})

/** 分组接口失败时静默降级：只是没有下拉选项，不影响列表本身。 */
async function loadGroupOptions(): Promise<void> {
  try {
    groupOptions.value = await listAdminJobGroups()
  } catch {
    groupOptions.value = []
  }
}

function resetFilters(): void {
  keywords.value = ''
  statusFilter.value = 'all'
  groupFilter.value = undefined
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

async function batchDelete(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0) return
  batchDeleting.value = true
  try {
    await deleteAdminJobs(ids)
    Message.success(t('logs.jobs.batchDeleted', { count: ids.length }))
    clearSelection()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.common.batchDeleteFailed')))
  } finally {
    batchDeleting.value = false
  }
}

async function openEditor(job?: AdminJob): Promise<void> {
  resetEditor(job)
  editorVisible.value = true
  if (!job?.id) return
  // Re-read the row so the form always edits the latest server state.
  editorLoading.value = true
  try {
    resetEditor(await getAdminJob(Number(job.id)))
  } catch (error) {
    editorVisible.value = false
    Message.error(apiErrorMessage(error, t('logs.jobs.detailLoadFailed')))
  } finally {
    editorLoading.value = false
  }
}

function resetEditor(job?: AdminJob): void {
  editor.id = Number(job?.id || 0)
  editor.jobName = String(job?.jobName || '')
  editor.jobGroup = String(job?.jobGroup || '')
  editor.invokeTarget = String(job?.invokeTarget || '')
  editor.cronExpression = String(job?.cronExpression || '')
  editor.misfirePolicy = Number(job?.misfirePolicy ?? 0)
  editor.concurrent = Number(job?.concurrent ?? 0)
  editor.status = Number(job?.status ?? 1)
  editor.remark = String(job?.remark || '')
}

async function saveEditor(): Promise<void> {
  if (editorLoading.value) return
  if (!editor.jobName.trim() || !editor.jobGroup.trim() || !editor.invokeTarget.trim() || !editor.cronExpression.trim()) {
    Message.error(t('logs.jobs.requiredFields'))
    return
  }
  saving.value = true
  try {
    const editing = Boolean(editor.id)
    await saveAdminJob({
      id: editor.id || undefined,
      jobName: editor.jobName.trim(),
      jobGroup: editor.jobGroup.trim(),
      invokeTarget: editor.invokeTarget.trim(),
      cronExpression: editor.cronExpression.trim(),
      misfirePolicy: Number(editor.misfirePolicy),
      concurrent: Number(editor.concurrent),
      status: Number(editor.status),
      remark: editor.remark?.trim() || ''
    })
    editorVisible.value = false
    Message.success(editing ? t('logs.jobs.updated') : t('logs.jobs.created'))
    // 新建的分组要立刻出现在筛选下拉里。
    void loadGroupOptions()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.jobs.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function changeStatus(job: AdminJob, value: boolean | string | number): Promise<void> {
  const jobId = Number(job.id)
  if (!Number.isInteger(jobId) || jobId <= 0) return
  const enabled = value === true || value === 1 || value === 'true'
  const previous = Number(job.status) === 1 ? 1 : 0
  const next = enabled ? 1 : 0
  job.status = next
  await withPending(jobId, async () => {
    try {
      await updateAdminJobStatus(jobId, next)
      Message.success(next === 1 ? t('logs.jobs.enabled') : t('logs.jobs.paused'))
    } catch (error) {
      job.status = previous
      Message.error(apiErrorMessage(error, t('logs.jobs.statusFailed')))
    }
  })
}

async function runOnce(job: AdminJob): Promise<void> {
  const jobId = Number(job.id)
  if (!job.canRunOnce || !Number.isFinite(jobId) || jobId <= 0) return
  runningId.value = jobId
  try {
    const outcome = await runAdminJob(jobId, String(job.jobGroup || ''))
    if (outcome.processed) Message.success(t('logs.jobs.runOnceDone'))
    else runHint.value = t('logs.jobs.runOnceEmpty')
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.jobs.runFailed')))
  } finally {
    runningId.value = 0
  }
}

async function deleteJob(id: unknown): Promise<void> {
  const jobId = Number(id)
  if (!Number.isInteger(jobId) || jobId <= 0) return
  try {
    await deleteAdminJobs([jobId])
    if (rows.value.length === 1 && current.value > 1) current.value -= 1
    Message.success(t('logs.jobs.deleted'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('logs.jobs.deleteFailed')))
  }
}

function groupColor(group: unknown): string {
  const value = String(group || '')
  if (!value) return 'gray'
  const palette = ['arcoblue', 'green', 'orange', 'purple', 'cyan', 'magenta']
  let hash = 0
  for (let index = 0; index < value.length; index += 1) hash = (hash * 31 + value.charCodeAt(index)) % 997
  return palette[hash % palette.length]
}
</script>

<style scoped>
.job-name-cell {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.job-name-cell small {
  overflow: hidden;
  color: var(--admin-subtle);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ellipsis {
  display: block;
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.job-editor-loading {
  display: flex;
  justify-content: center;
  padding: 56px 0;
}

.job-group-hint {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  color: var(--admin-muted);
}
</style>
