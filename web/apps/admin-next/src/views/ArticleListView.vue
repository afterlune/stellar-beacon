<template>
  <section class="admin-page">
    <AdminPageHeader title="文章列表" description="把文章、状态和发布节奏整理在一张清晰的桌面上。">
      <template #actions>
        <a-space>
          <a-button type="primary" @click="router.push('/articles')">
            <template #icon><IconPlus /></template>
            发布文章
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
            placeholder="搜索文章标题"
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
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #cover="{ record }">
            <span class="admin-cover-cell">
              <img v-if="isHttpUrl(record.articleCover)" :src="String(record.articleCover)" alt="文章封面" />
              <IconBook v-else aria-hidden="true" />
            </span>
          </template>
          <template #title="{ record }">
            <span class="admin-title-cell" :title="String(record.articleTitle || '')">{{ record.articleTitle || '未命名文章' }}</span>
          </template>
          <template #status="{ record }">
            <a-tag class="admin-status-tag" :color="statusMeta(record.status).color">{{ statusMeta(record.status).label }}</a-tag>
          </template>
          <template #type="{ record }">
            <span class="admin-muted-cell">{{ typeLabel(record.type) }}</span>
          </template>
          <template #flags="{ record }">
            <a-space v-if="Number(record.isTop) === 1 || Number(record.isFeatured) === 1" wrap>
              <a-tag v-if="Number(record.isTop) === 1" color="arcoblue">置顶</a-tag>
              <a-tag v-if="Number(record.isFeatured) === 1" color="green">精选</a-tag>
            </a-space>
            <span v-else class="admin-muted-cell">—</span>
          </template>
          <template #actions="{ record }">
            <a-button class="admin-action-button" type="text" size="small" @click="editArticle(record.id)">编辑</a-button>
          </template>
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
import { IconBook, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, listAdminPage } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatCell } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

interface TableColumn {
  title: string
  dataIndex: string
  slotName?: string
  width?: number
  ellipsis?: boolean
  tooltip?: boolean
}

const router = useRouter()
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const errorMessage = ref('')
const records = ref<Record<string, unknown>[]>([])
const total = ref(0)

const columns: TableColumn[] = [
  { title: '封面', dataIndex: 'articleCover', slotName: 'cover', width: 76 },
  { title: '标题', dataIndex: 'articleTitle', slotName: 'title', ellipsis: true, tooltip: true },
  { title: '分类', dataIndex: 'categoryName', width: 120 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '类型', dataIndex: 'type', slotName: 'type', width: 90 },
  { title: '标记', dataIndex: 'flags', slotName: 'flags', width: 120 },
  { title: '浏览量', dataIndex: 'viewsCount', width: 90 },
  { title: '创建时间', dataIndex: 'createTime', width: 175 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 76 }
]

const tableColumns = computed(() => columns.map((column) => ({ ...column, slotName: column.slotName || 'formatted' })))
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
    const page = await listAdminPage<Record<string, unknown>>('admin/articles', {
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

function editArticle(id: unknown): void {
  const articleId = Number(id)
  if (Number.isInteger(articleId) && articleId > 0) void router.push(`/articles/${articleId}`)
}

function statusMeta(value: unknown): { label: string; color: string } {
  const status = Number(value)
  if (status === 1) return { label: '公开', color: 'green' }
  if (status === 2) return { label: '私密', color: 'orange' }
  if (status === 3) return { label: '草稿', color: 'arcoblue' }
  return { label: '未知', color: 'gray' }
}

function typeLabel(value: unknown): string {
  const type = Number(value)
  if (type === 1) return '原创'
  if (type === 2) return '转载'
  if (type === 3) return '翻译'
  return '—'
}

function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && /^https?:\/\//i.test(value)
}
</script>

<style scoped>
.admin-toolbar-caption {
  color: var(--admin-muted);
  font-size: 12px;
}
</style>
