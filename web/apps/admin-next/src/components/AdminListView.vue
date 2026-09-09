<template>
  <section class="admin-page">
    <AdminPageHeader :title="config.title" :description="config.description">
      <template #actions>
        <a-space>
          <a-button v-if="config.createPath" type="primary" @click="router.push(config.createPath)">
            <template #icon><IconPlus /></template>
            {{ config.createText || '新增' }}
          </a-button>
          <a-button :loading="loading" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </a-space>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="keywords"
            class="admin-filter-input"
            :placeholder="config.placeholder"
            allow-clear
            @search="reload" />
          <span class="admin-toolbar-caption">共 {{ total }} 条记录</span>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table
          :data="records"
          :columns="tableColumns"
          :loading="loading"
          :pagination="pagination"
          :row-key="config.rowKey"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #formatted="{ record, column }">
            {{ formatCell(record[column.dataIndex]) }}
          </template>
          <template #empty>
            <div class="admin-table-empty">
              <a-empty description="还没有内容" />
            </div>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, listAdminPage } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatCell } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

type ListMode = 'onlineUsers'

interface TableColumn {
  title: string
  dataIndex: string
  width?: number
}

interface ListConfig {
  title: string
  description: string
  placeholder: string
  endpoint: string
  rowKey: string
  columns: TableColumn[]
  createPath?: string
  createText?: string
}

const props = defineProps<{ mode: ListMode }>()
const router = useRouter()
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const errorMessage = ref('')
const records = ref<Record<string, unknown>[]>([])
const total = ref(0)

const configs: Record<ListMode, ListConfig> = {
  onlineUsers: {
    title: '在线用户',
    description: '了解当前仍在活动的登录会话。',
    placeholder: '搜索用户昵称',
    endpoint: 'admin/users/online',
    rowKey: 'userInfoId',
    columns: [
      { title: '用户 ID', dataIndex: 'userInfoId', width: 100 },
      { title: '昵称', dataIndex: 'nickname' },
      { title: '浏览器', dataIndex: 'browser' },
      { title: '操作系统', dataIndex: 'os' },
      { title: 'IP', dataIndex: 'ipAddress' },
      { title: '最后登录', dataIndex: 'lastLoginTime', width: 180 }
    ]
  }
}

const config = computed(() => configs[props.mode])
const tableColumns = computed(() => config.value.columns.map((column) => ({ ...column, slotName: 'formatted' })))
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

onMounted(() => void reload())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<Record<string, unknown>>(config.value.endpoint, {
      current: current.value,
      size: pageSize.value,
      keywords: keywords.value.trim()
    })
    records.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '列表加载失败')
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
</script>

<style scoped>
.admin-toolbar-caption {
  color: var(--admin-muted);
  font-size: 12px;
}
</style>
