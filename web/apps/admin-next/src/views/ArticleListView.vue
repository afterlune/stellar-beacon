<template>
  <section class="admin-page">
    <AdminPageHeader title="文章列表" description="把文章、状态和发布节奏整理在一张清晰的桌面上。">
      <template #actions>
        <a-button type="primary" @click="router.push('/articles')">
          <template #icon><IconPlus /></template>
          发布文章
        </a-button>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          刷新
        </a-button>
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
          <a-select v-model="status" placeholder="全部状态" allow-clear style="width: 130px" @change="reload">
            <a-option :value="1">公开</a-option>
            <a-option :value="2">私密</a-option>
            <a-option :value="3">草稿</a-option>
          </a-select>
          <a-select v-model="type" placeholder="全部类型" allow-clear style="width: 130px" @change="reload">
            <a-option :value="1">原创</a-option>
            <a-option :value="2">转载</a-option>
            <a-option :value="3">翻译</a-option>
          </a-select>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">重置筛选</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 篇文章</span>
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
            <AdminImagePreview
              v-if="isHttpUrl(record.articleCover)"
              :src="String(record.articleCover)"
              :alt="`${String(record.articleTitle || '文章')} 封面`"
              :width="88"
              :height="60" />
            <span v-else class="admin-cover-cell" aria-hidden="true"><IconBook /></span>
          </template>
          <template #title="{ record }">
            <span class="admin-title-cell" :title="String(record.articleTitle || '')">{{ record.articleTitle || '未命名文章' }}</span>
          </template>
          <template #category="{ record }">
            <a-tag v-if="record.categoryName" color="arcoblue">{{ record.categoryName }}</a-tag>
            <span v-else class="admin-muted-cell">未分类</span>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="statusKind(record.status)" />
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
          <template #views="{ record }">
            <span class="admin-num-cell">{{ formatNumber(record.viewsCount) }}</span>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="editArticle(record.id)">编辑</a-button>
              <a-dropdown trigger="click" position="br">
                <a-button type="text" size="small" :loading="busyId === Number(record.id)">
                  更多
                  <template #icon><IconDown /></template>
                </a-button>
                <template #content>
                  <a-doption @click="toggleFlag(record, 'isTop')">
                    {{ Number(record.isTop) === 1 ? '取消置顶' : '设为置顶' }}
                  </a-doption>
                  <a-doption @click="toggleFlag(record, 'isFeatured')">
                    {{ Number(record.isFeatured) === 1 ? '取消精选' : '设为精选' }}
                  </a-doption>
                  <a-doption @click="moveToTrash(record)">移入回收站</a-doption>
                  <a-doption class="admin-danger-option" @click="removeArticle(record)">永久删除</a-doption>
                </template>
              </a-dropdown>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconBook"
              :title="hasFilters ? '没有匹配的文章' : '还没有文章'"
              :description="hasFilters ? '换个关键词或清空筛选条件再试一次。' : '写下第一篇内容，它会立即出现在这里。'">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">清空筛选</a-button>
              <a-button v-else type="primary" size="small" @click="router.push('/articles')">发布文章</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Modal, Message } from '@arco-design/web-vue'
import { IconBook, IconDown, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteAdminArticles,
  listAdminPage,
  updateAdminArticleFeatured,
  updateAdminArticleTrash
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag, { type StatusKind } from '@/components/AdminStatusTag.vue'
import { formatDateTime, formatNumber, isHttpUrl } from '@/utils/format'
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
const status = ref<number | undefined>(undefined)
const type = ref<number | undefined>(undefined)
const current = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const errorMessage = ref('')
const records = ref<Record<string, unknown>[]>([])
const total = ref(0)
const busyId = ref(0)

const columns: TableColumn[] = [
  { title: '封面', dataIndex: 'articleCover', slotName: 'cover', width: 108 },
  { title: '标题', dataIndex: 'articleTitle', slotName: 'title', ellipsis: true, tooltip: true },
  { title: '分类', dataIndex: 'categoryName', slotName: 'category', width: 130 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 96 },
  { title: '类型', dataIndex: 'type', slotName: 'type', width: 86 },
  { title: '标记', dataIndex: 'flags', slotName: 'flags', width: 118 },
  { title: '浏览量', dataIndex: 'viewsCount', slotName: 'views', width: 96 },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'time', width: 168 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 152 }
]

// Every column gets an explicit slot: the fallback `formatted` slot renders
// untouched data, which would bypass the shared time/number formatting.
const tableColumns = computed(() => columns.map((column) => ({ ...column, slotName: column.slotName || 'formatted' })))
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() => Boolean(keywords.value.trim()) || status.value !== undefined || type.value !== undefined)

onMounted(() => void reload())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function resetFilters(): void {
  keywords.value = ''
  status.value = undefined
  type.value = undefined
  void reload()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<Record<string, unknown>>('admin/articles', {
      current: current.value,
      size: pageSize.value,
      keywords: keywords.value.trim(),
      status: status.value ?? 0,
      type: type.value ?? 0
    })
    records.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '文章列表加载失败')
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

/** 行内“更多”菜单：置顶 / 精选 / 回收站 / 删除，全部复用已有后端接口。 */
async function toggleFlag(record: Record<string, unknown>, field: 'isTop' | 'isFeatured'): Promise<void> {
  const id = Number(record.id)
  if (!Number.isInteger(id) || id <= 0) return
  const next = Number(record[field]) === 1 ? 0 : 1
  busyId.value = id
  try {
    await updateAdminArticleFeatured({
      id,
      isTop: field === 'isTop' ? next : Number(record.isTop) === 1 ? 1 : 0,
      isFeatured: field === 'isFeatured' ? next : Number(record.isFeatured) === 1 ? 1 : 0
    })
    record[field] = next
    Message.success(next === 1 ? (field === 'isTop' ? '已设为置顶' : '已设为精选') : '已取消标记')
  } catch (error) {
    Message.error(apiErrorMessage(error, '标记更新失败'))
  } finally {
    busyId.value = 0
  }
}

function moveToTrash(record: Record<string, unknown>): void {
  const id = Number(record.id)
  if (!Number.isInteger(id) || id <= 0) return
  Modal.confirm({
    title: '移入回收站',
    content: `确定把《${String(record.articleTitle || '未命名文章')}》移入回收站吗？之后可以在回收站中恢复。`,
    okText: '移入回收站',
    cancelText: '取消',
    onOk: async () => {
      busyId.value = id
      try {
        await updateAdminArticleTrash([id], 1)
        Message.success('已移入回收站')
        if (records.value.length === 1 && current.value > 1) current.value -= 1
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, '移入回收站失败'))
      } finally {
        busyId.value = 0
      }
    }
  })
}

function removeArticle(record: Record<string, unknown>): void {
  const id = Number(record.id)
  if (!Number.isInteger(id) || id <= 0) return
  Modal.confirm({
    title: '永久删除',
    content: `永久删除《${String(record.articleTitle || '未命名文章')}》后无法恢复，确定继续吗？`,
    okText: '永久删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      busyId.value = id
      try {
        await deleteAdminArticles([id])
        Message.success('文章已删除')
        if (records.value.length === 1 && current.value > 1) current.value -= 1
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, '删除失败'))
      } finally {
        busyId.value = 0
      }
    }
  })
}

function statusKind(value: unknown): StatusKind {
  const raw = Number(value)
  if (raw === 1) return 'public'
  if (raw === 2) return 'private'
  if (raw === 3) return 'draft'
  return 'normal'
}

function typeLabel(value: unknown): string {
  const raw = Number(value)
  if (raw === 1) return '原创'
  if (raw === 2) return '转载'
  if (raw === 3) return '翻译'
  return '—'
}
</script>

<style scoped>
.admin-toolbar-caption {
  color: var(--admin-muted);
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
</style>

<style>
/* 下拉菜单里的危险操作：与普通项区分开 */
.admin-danger-option {
  color: var(--admin-danger) !important;
}
</style>
