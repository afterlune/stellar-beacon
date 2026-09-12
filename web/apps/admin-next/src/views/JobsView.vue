<template>
  <section class="admin-page">
    <AdminPageHeader title="定时任务" description="让重复的后台工作安静、可靠地运行。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索任务名"
          allow-clear
          @search="reload" />
        <a-button type="primary" @click="openEditor()">
          <template #icon><IconPlus /></template>
          新增
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-radio-group v-model="statusFilter" type="button" size="small" @change="reload">
            <a-radio value="all">全部</a-radio>
            <a-radio value="1">正常</a-radio>
            <a-radio value="0">已暂停</a-radio>
          </a-radio-group>
          <a-tag v-if="keywords.trim()" color="arcoblue">关键词：{{ keywords.trim() }}</a-tag>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">清空搜索</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 个任务</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-alert v-if="runHint" type="info" closable @close="runHint = ''">{{ runHint }}</a-alert>

      <div class="admin-table-shell">
        <a-table
          :data="jobs"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #jobName="{ record }">
            <div class="job-name-cell">
              <span class="admin-title-cell">{{ record.jobName || '未命名任务' }}</span>
              <small v-if="record.remark" :title="String(record.remark)">{{ record.remark }}</small>
            </div>
          </template>
          <template #group="{ record }">
            <a-tag :color="groupColor(record.jobGroup)">{{ record.jobGroup || '默认' }}</a-tag>
          </template>
          <template #target="{ record }">
            <span class="admin-mono-cell ellipsis" :title="String(record.invokeTarget || '')">{{ record.invokeTarget || '—' }}</span>
          </template>
          <template #cron="{ record }">
            <span class="admin-mono-cell" :title="String(record.cronExpression || '')">{{ record.cronExpression || '—' }}</span>
          </template>
          <template #status="{ record }">
            <a-tooltip :content="Number(record.status) === 1 ? '点击暂停该任务' : '点击启用该任务'">
              <a-switch
                :model-value="Number(record.status) === 1"
                :checked-value="true"
                :unchecked-value="false"
                :loading="pendingStatusId === Number(record.id)"
                :disabled="pendingStatusId !== 0 && pendingStatusId !== Number(record.id)"
                @change="changeStatus(record, $event)">
                <template #checked>正常</template>
                <template #unchecked>暂停</template>
              </a-switch>
            </a-tooltip>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm v-if="record.canRunOnce" content="只处理一个已入队任务，确认立即执行吗？" @ok="runOnce(record)">
                <a-button type="text" size="small" :loading="runningId === Number(record.id)">执行一次</a-button>
              </a-popconfirm>
              <a-tooltip v-else :content="String(record.runOnceReason || '该任务目标暂不支持手动执行')">
                <a-button type="text" size="small" disabled>执行一次</a-button>
              </a-tooltip>
              <a-popconfirm
                :content="`确定删除任务「${record.jobName}」吗？删除后调度配置不可恢复。`"
                @ok="deleteJob(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconCalendarClock"
              :title="hasFilters ? '没有匹配的任务' : '还没有定时任务'"
              :description="hasFilters ? '换个关键词或重置筛选条件再试一次。' : '创建任务后，后台会按 Cron 表达式自动执行它。'">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">重置筛选</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">新增任务</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? '编辑任务' : '新增任务'"
      :ok-loading="saving"
      :ok-button-props="{ disabled: editorLoading }"
      :mask-closable="false"
      width="720px"
      @ok="saveEditor">
      <a-spin v-if="editorLoading" class="job-editor-loading" tip="正在读取任务详情…" />
      <a-form v-else :model="editor" layout="vertical">
        <div class="admin-form-grid">
          <a-form-item field="jobName" label="任务名称" required>
            <a-input v-model="editor.jobName" maxlength="64" show-word-limit placeholder="例如：清理过期文章" />
          </a-form-item>
          <a-form-item field="jobGroup" label="任务分组" required>
            <a-input v-model="editor.jobGroup" maxlength="64" show-word-limit placeholder="例如：默认" />
          </a-form-item>
        </div>
        <a-form-item field="invokeTarget" label="调用目标" required>
          <a-input v-model="editor.invokeTarget" maxlength="500" placeholder="例如：article.cleanup" />
          <template #help>
            这里只保存目标标识；“执行一次”仅允许后端固定白名单中、且已启用 Worker 的目标。
          </template>
        </a-form-item>
        <a-form-item field="cronExpression" label="Cron 表达式" required>
          <a-input v-model="editor.cronExpression" maxlength="255" placeholder="例如：0 0 3 * * ?" />
          <template #help>使用 Quartz 六段式 Cron，例如每天凌晨 3 点执行：0 0 3 * * ?</template>
        </a-form-item>
        <div class="admin-form-grid">
          <a-form-item label="错误策略">
            <a-select v-model="editor.misfirePolicy">
              <a-option :value="0">默认策略</a-option>
              <a-option :value="1">立即执行</a-option>
              <a-option :value="2">执行一次</a-option>
              <a-option :value="3">放弃执行</a-option>
            </a-select>
            <template #help>错过触发时间后如何处理本次调度。</template>
          </a-form-item>
          <a-form-item label="并发执行">
            <a-radio-group v-model="editor.concurrent">
              <a-radio :value="0">允许</a-radio>
              <a-radio :value="1">禁止</a-radio>
            </a-radio-group>
            <template #help>禁止并发可避免同一任务被重复执行。</template>
          </a-form-item>
        </div>
        <a-form-item label="状态">
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">正常</a-radio>
            <a-radio :value="0">暂停</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea v-model="editor.remark" maxlength="500" show-word-limit :auto-size="{ minRows: 2, maxRows: 4 }" placeholder="记录这个任务的用途，方便日后维护" />
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
  runAdminJob,
  saveAdminJob,
  updateAdminJobStatus
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatDateTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminJob } from '@benetnasch/api-contract'

const columns = [
  { title: '任务名称', dataIndex: 'jobName', slotName: 'jobName', minWidth: 200 },
  { title: '分组', dataIndex: 'jobGroup', slotName: 'group', width: 118 },
  { title: '调用目标', dataIndex: 'invokeTarget', slotName: 'target', ellipsis: true, tooltip: true, minWidth: 180 },
  { title: 'Cron', dataIndex: 'cronExpression', slotName: 'cron', width: 156 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 118 },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'time', width: 168 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 224 }
]

const jobs = ref<AdminJob[]>([])
const keywords = ref('')
const statusFilter = ref<'all' | '0' | '1'>('all')
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const editorLoading = ref(false)
const pendingStatusId = ref(0)
const runningId = ref(0)
const errorMessage = ref('')
const runHint = ref('')
const editorVisible = ref(false)
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

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() => Boolean(keywords.value.trim()) || statusFilter.value !== 'all')

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function resetFilters(): void {
  keywords.value = ''
  statusFilter.value = 'all'
  void reload()
}

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminJobs({
      current: current.value,
      size: pageSize.value,
      jobName: keywords.value.trim()
    })
    // The jobs endpoint filters by name only, so the status tab is applied here.
    jobs.value = statusFilter.value === 'all'
      ? page.items
      : page.items.filter((job) => Number(job.status) === Number(statusFilter.value))
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '任务列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void load()
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
    Message.error(apiErrorMessage(error, '任务详情加载失败'))
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
    Message.error('任务名称、分组、调用目标和 Cron 表达式不能为空')
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
    Message.success(editing ? '任务已更新' : '任务已创建')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务保存失败'))
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
  pendingStatusId.value = jobId
  try {
    await updateAdminJobStatus(jobId, next)
    Message.success(next === 1 ? '任务已启用' : '任务已暂停')
  } catch (error) {
    job.status = previous
    Message.error(apiErrorMessage(error, '任务状态更新失败'))
  } finally {
    pendingStatusId.value = 0
  }
}

async function runOnce(job: AdminJob): Promise<void> {
  const jobId = Number(job.id)
  if (!job.canRunOnce || !Number.isFinite(jobId) || jobId <= 0) return
  runningId.value = jobId
  try {
    const outcome = await runAdminJob(jobId, String(job.jobGroup || ''))
    if (outcome.processed) Message.success('任务已执行一次')
    else runHint.value = '没有可执行的队列任务：该目标当前没有待处理的数据。'
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务执行失败'))
  } finally {
    runningId.value = 0
  }
}

async function deleteJob(id: unknown): Promise<void> {
  const jobId = Number(id)
  if (!Number.isInteger(jobId) || jobId <= 0) return
  try {
    await deleteAdminJobs([jobId])
    if (jobs.value.length === 1 && current.value > 1) current.value -= 1
    Message.success('任务已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务删除失败'))
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
</style>
