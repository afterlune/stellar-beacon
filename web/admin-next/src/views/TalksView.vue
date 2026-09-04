<template>
  <section>
    <a-card title="说说管理">
      <template #extra>
        <a-space>
          <a-select v-model="status" allow-clear placeholder="发布状态" style="width: 130px" @change="reload">
            <a-option :value="1">公开</a-option>
            <a-option :value="2">私密</a-option>
          </a-select>
          <a-button type="primary" @click="router.push('/talks')">新增</a-button>
        </a-space>
      </template>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-table
        :data="talks"
        :columns="columns"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @page-change="changePage"
        @page-size-change="changePageSize">
        <template #content="{ record }">
          <span class="talk-content" :title="plainText(record.content)">{{ plainText(record.content) || '—' }}</span>
        </template>
        <template #status="{ record }">{{ Number(record.status) === 1 ? '公开' : '私密' }}</template>
        <template #top="{ record }">{{ Number(record.isTop) === 1 ? '是' : '否' }}</template>
        <template #actions="{ record }">
          <a-space>
            <a-button type="text" size="small" @click="router.push(`/talks/${record.id}`)">编辑</a-button>
            <a-popconfirm content="确定删除这条说说吗？" @ok="deleteTalk(record.id)">
              <a-button type="text" status="danger" size="small">删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
        <template #empty><a-empty description="暂无说说" /></template>
      </a-table>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRouter } from 'vue-router'

import { apiErrorMessage, deleteAdminTalks, listAdminTalks } from '@/api/http'
import type { AdminTalk } from '@shared/api-contract'

const router = useRouter()
const columns = [
  { title: '内容', dataIndex: 'content', ellipsis: true, tooltip: true, slotName: 'content' },
  { title: '作者', dataIndex: 'nickname', width: 130 },
  { title: '评论数', dataIndex: 'commentCount', width: 100 },
  { title: '置顶', dataIndex: 'isTop', width: 80, slotName: 'top' },
  { title: '状态', dataIndex: 'status', width: 90, slotName: 'status' },
  { title: '创建时间', dataIndex: 'createTime', width: 190 },
  { title: '操作', dataIndex: 'actions', width: 150, slotName: 'actions' }
]

const talks = ref<AdminTalk[]>([])
const status = ref<number | undefined>(undefined)
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const errorMessage = ref('')

const pagination = computed(() => ({
  current: current.value,
  pageSize: pageSize.value,
  total: total.value,
  showTotal: true,
  showJumper: true,
  showPageSize: true
}))

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminTalks({ current: current.value, size: pageSize.value, status: status.value ?? 0 })
    talks.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '说说列表加载失败')
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

async function deleteTalk(id: unknown): Promise<void> {
  const talkId = Number(id)
  if (!talkId) return
  try {
    await deleteAdminTalks([talkId])
    Message.success('说说已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '说说删除失败'))
  }
}

function plainText(value: unknown): string {
  return String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
}
</script>

<style scoped>
.talk-content {
  display: block;
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
