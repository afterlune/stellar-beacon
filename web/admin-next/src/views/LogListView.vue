<template>
  <section class="admin-page">
    <AdminPageHeader :title="config.title" :description="description">
      <template #actions>
        <a-space>
          <a-input-search v-model="keywords" class="admin-filter-input" :placeholder="config.placeholder" allow-clear @search="reload" />
          <a-popconfirm v-if="mode === 'job'" content="确定清空全部任务日志吗？" @ok="clean">
            <a-button status="danger">清空任务日志</a-button>
          </a-popconfirm>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table :data="records" :columns="config.columns" :loading="loading" :pagination="pagination" row-key="id" @page-change="changePage" @page-size-change="changePageSize">
          <template #method="{ record }">{{ formatCell(record.requestMethod || record.optMethod) }}</template>
          <template #status="{ record }"><a-tag class="admin-status-tag" :color="Number(record.status) === 1 ? 'red' : 'green'">{{ Number(record.status) === 1 ? '失败' : '成功' }}</a-tag></template>
          <template #content="{ record }"><span class="ellipsis" :title="String(record.exceptionInfo || record.jobMessage || '')">{{ formatCell(record.exceptionInfo || record.jobMessage) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="showDetail(record)">详情</a-button>
              <a-popconfirm content="确定删除这条日志吗？" @ok="deleteLog(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无日志" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="detailVisible" title="日志详情" width="760px" :footer="false">
      <pre class="detail">{{ detailText }}</pre>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRoute } from 'vue-router'

import { apiErrorMessage, cleanAdminJobLogs, deleteAdminLogs, listAdminPage } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatCell } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

type LogMode = 'operation' | 'exception' | 'job'
interface LogRecord {
  id: number
  [key: string]: unknown
}

const props = defineProps<{ mode: LogMode }>()
const route = useRoute()
const mode = computed(() => props.mode)
const configs = {
  operation: {
    title: '操作日志',
    description: '留下可追溯的后台操作记录。',
    placeholder: '搜索模块或描述',
    endpoint: 'admin/operation/logs',
    columns: [
      { title: '模块', dataIndex: 'optModule' },
      { title: '类型', dataIndex: 'optType' },
      { title: 'URI', dataIndex: 'optUri' },
      { title: '方法', dataIndex: 'requestMethod', slotName: 'method' },
      { title: '用户', dataIndex: 'nickname' },
      { title: '时间', dataIndex: 'createTime' },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  },
  exception: {
    title: '异常日志',
    description: '把异常留在记录里，让问题更容易被定位。',
    placeholder: '搜索请求 URI 或描述',
    endpoint: 'admin/exception/logs',
    columns: [
      { title: 'URI', dataIndex: 'optUri' },
      { title: '方法', dataIndex: 'requestMethod', slotName: 'method' },
      { title: '描述', dataIndex: 'optDesc' },
      { title: '异常', dataIndex: 'exceptionInfo', slotName: 'content', ellipsis: true, tooltip: true },
      { title: '时间', dataIndex: 'createTime' },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  },
  job: {
    title: '任务日志',
    description: '回看后台任务的执行结果与耗时。',
    placeholder: '搜索任务名',
    endpoint: 'admin/jobLogs',
    columns: [
      { title: '任务', dataIndex: 'jobName' },
      { title: '任务组', dataIndex: 'jobGroup' },
      { title: '调用目标', dataIndex: 'invokeTarget', ellipsis: true, tooltip: true },
      { title: '状态', dataIndex: 'status', slotName: 'status' },
      { title: '时间', dataIndex: 'startTime' },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  }
} as const

const config = computed(() => configs[props.mode])
const description = computed(() => config.value.description)
const records = ref<LogRecord[]>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const errorMessage = ref('')
const detailVisible = ref(false)
const detail = ref<LogRecord | null>(null)
const detailText = computed(() => detail.value ? JSON.stringify(detail.value, null, 2) : '')
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

onMounted(() => void load())
watch(() => route.params.quartzId, () => {
  if (props.mode === 'job') void reload()
})

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const filterKey = props.mode === 'job' ? 'jobName' : 'keywords'
    const params: Record<string, string | number> = {
      current: current.value,
      size: pageSize.value,
      [filterKey]: keywords.value.trim()
    }
    if (props.mode === 'job') {
      const jobId = Number(route.params.quartzId)
      if (Number.isInteger(jobId) && jobId > 0) params.jobId = jobId
    }
    const page = await listAdminPage<LogRecord>(config.value.endpoint, params)
    records.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '日志加载失败')
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

function showDetail(record: LogRecord): void {
  detail.value = record
  detailVisible.value = true
}

async function deleteLog(id: unknown): Promise<void> {
  const logId = Number(id)
  if (!logId) return
  try {
    await deleteAdminLogs(props.mode, [logId])
    Message.success('日志已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '日志删除失败'))
  }
}

async function clean(): Promise<void> {
  try {
    await cleanAdminJobLogs()
    Message.success('任务日志已清空')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '任务日志清理失败'))
  }
}


</script>

<style scoped>
.ellipsis { display: block; max-width: 320px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail { max-height: 60vh; overflow: auto; white-space: pre-wrap; word-break: break-word; }
</style>
