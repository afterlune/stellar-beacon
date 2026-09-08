<template>
  <section class="admin-page">
    <AdminPageHeader title="定时任务" description="让重复的后台工作安静、可靠地运行。">
      <template #actions>
        <a-space>
          <a-input-search v-model="keywords" class="admin-filter-input" placeholder="搜索任务名" allow-clear @search="reload" />
          <a-button type="primary" @click="openEditor()">
            <template #icon><IconPlus /></template>
            新增
          </a-button>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table
          :data="jobs"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #group="{ record }"><a-tag>{{ record.jobGroup }}</a-tag></template>
          <template #target="{ record }"><span class="ellipsis" :title="record.invokeTarget">{{ record.invokeTarget }}</span></template>
          <template #status="{ record }">
            <a-switch
              :model-value="Number(record.status) === 1"
              :checked-value="true"
              :unchecked-value="false"
              @change="changeStatus(record, $event)">
              <template #checked>正常</template>
              <template #unchecked>暂停</template>
            </a-switch>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm v-if="record.canRunOnce" content="只处理一个已入队任务，确认执行吗？" @ok="runOnce(record)">
                <a-button type="text" size="small">执行一次</a-button>
              </a-popconfirm>
              <a-tooltip v-else :content="record.runOnceReason || '该任务目标暂不支持手动执行'">
                <a-button type="text" size="small" disabled>执行一次</a-button>
              </a-tooltip>
              <a-popconfirm content="确定删除该任务吗？" @ok="deleteJob(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无任务" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="editorVisible" :title="editor.id ? '编辑任务' : '新增任务'" :ok-loading="saving" width="700px" @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-grid :cols="2" :col-gap="16">
          <a-grid-item>
            <a-form-item field="jobName" label="任务名称" required>
              <a-input v-model="editor.jobName" maxlength="64" show-word-limit />
            </a-form-item>
          </a-grid-item>
          <a-grid-item>
            <a-form-item field="jobGroup" label="任务分组" required>
              <a-input v-model="editor.jobGroup" maxlength="64" show-word-limit />
            </a-form-item>
          </a-grid-item>
          <a-grid-item :span="2">
            <a-form-item field="invokeTarget" label="调用目标" required>
              <a-input v-model="editor.invokeTarget" maxlength="500" show-word-limit />
              <template #help>这里只保存目标标识；"执行一次"仅允许后端固定白名单中、且已启用 Worker 的目标。</template>
            </a-form-item>
          </a-grid-item>
          <a-grid-item :span="2">
            <a-form-item field="cronExpression" label="Cron 表达式" required>
              <a-input v-model="editor.cronExpression" maxlength="255" show-word-limit />
            </a-form-item>
          </a-grid-item>
          <a-grid-item>
            <a-form-item label="错误策略">
              <a-select v-model="editor.misfirePolicy">
                <a-option :value="0">默认策略</a-option>
                <a-option :value="1">立即执行</a-option>
                <a-option :value="2">执行一次</a-option>
                <a-option :value="3">放弃执行</a-option>
              </a-select>
            </a-form-item>
          </a-grid-item>
          <a-grid-item>
            <a-form-item label="并发执行">
              <a-radio-group v-model="editor.concurrent">
                <a-radio :value="0">允许</a-radio>
                <a-radio :value="1">禁止</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-grid-item>
          <a-grid-item>
            <a-form-item label="状态">
              <a-radio-group v-model="editor.status">
                <a-radio :value="1">正常</a-radio>
                <a-radio :value="0">暂停</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-grid-item>
          <a-grid-item :span="2">
            <a-form-item label="备注">
              <a-textarea v-model="editor.remark" maxlength="500" show-word-limit />
            </a-form-item>
          </a-grid-item>
        </a-grid>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteAdminJobs,
  getAdminJob,
  listAdminJobs,
  runAdminJob,
  saveAdminJob,
  updateAdminJobStatus
} from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { tablePagination } from '@/utils/pagination'
import type { AdminJob } from '@shared/api-contract'

const columns = [
  { title: '任务名称', dataIndex: 'jobName' },
  { title: '任务分组', dataIndex: 'jobGroup', slotName: 'group', width: 120 },
  { title: '调用目标', dataIndex: 'invokeTarget', slotName: 'target', ellipsis: true, tooltip: true },
  { title: 'Cron', dataIndex: 'cronExpression', ellipsis: true, tooltip: true, width: 170 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 110 },
  { title: '创建时间', dataIndex: 'createTime', width: 190 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 230 }
]

const jobs = ref<AdminJob[]>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const editorLoading = ref(false)
const errorMessage = ref('')
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

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminJobs({ current: current.value, size: pageSize.value, jobName: keywords.value.trim() })
    jobs.value = page.records
    total.value = page.count
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
  editorVisible.value = true
  editorLoading.value = Boolean(job?.id)
  resetEditor(job)
  if (!job?.id) {
    editorLoading.value = false
    return
  }
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
    Message.success('任务已保存')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务保存失败'))
  } finally {
    saving.value = false
  }
}

async function changeStatus(job: AdminJob, enabled: boolean): Promise<void> {
  const previous = Number(job.status) === 1 ? 1 : 0
  const next = enabled ? 1 : 0
  job.status = next
  try {
    await updateAdminJobStatus(Number(job.id), next)
    Message.success(next === 1 ? '任务已启用' : '任务已暂停')
  } catch (error) {
    job.status = previous
    Message.error(apiErrorMessage(error, '任务状态更新失败'))
  }
}

async function runOnce(job: AdminJob): Promise<void> {
  const jobId = Number(job.id)
  if (!job.canRunOnce || !Number.isFinite(jobId) || jobId <= 0) return
  try {
    const outcome = await runAdminJob(jobId, String(job.jobGroup || ''))
    Message.success(outcome.processed ? '任务已执行一次' : '没有可执行的队列任务')
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务执行失败'))
  }
}

async function deleteJob(id: unknown): Promise<void> {
  const jobId = Number(id)
  if (!Number.isFinite(jobId) || jobId <= 0) return
  try {
    await deleteAdminJobs([jobId])
    Message.success('任务已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务删除失败'))
  }
}
</script>

<style scoped>
.ellipsis {
  display: block;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
