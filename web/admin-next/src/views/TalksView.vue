<template>
  <section class="admin-page">
    <AdminPageHeader title="说说管理" description="记录那些不必写成文章的片刻。">
      <template #actions>
        <a-space>
          <a-select v-model="status" allow-clear placeholder="发布状态" style="width: 130px" @change="reload">
            <a-option :value="1">公开</a-option>
            <a-option :value="2">私密</a-option>
          </a-select>
          <a-button type="primary" @click="router.push('/talks')">
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
          <template #status="{ record }"><a-tag class="admin-status-tag" :color="Number(record.status) === 1 ? 'green' : 'orange'">{{ Number(record.status) === 1 ? '公开' : '私密' }}</a-tag></template>
          <template #top="{ record }"><a-tag class="admin-status-tag" :color="Number(record.isTop) === 1 ? 'arcoblue' : 'gray'">{{ Number(record.isTop) === 1 ? '是' : '否' }}</a-tag></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="router.push(`/talks/${record.id}`)">编辑</a-button>
              <a-popconfirm content="确定删除这条说说吗？" @ok="deleteTalk(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无说说" /></div></template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, deleteAdminTalks, listAdminTalks } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { tablePagination } from '@/utils/pagination'
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
