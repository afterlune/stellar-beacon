<template>
  <section class="admin-page">
    <AdminPageHeader title="说说管理" description="发布和管理说说。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="在本页搜索说说内容"
          allow-clear />
        <a-button v-if="keywords.trim()" @click="keywords = ''">清空搜索</a-button>
        <a-button type="primary" @click="router.push('/talks')">
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
            <a-radio value="1">公开</a-radio>
            <a-radio value="2">私密</a-radio>
          </a-radio-group>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">重置筛选</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 条说说 · 本页 {{ visibleTalks.length }} 条</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="说说列表加载失败" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="`本页 ${visibleTalks.length} 条`" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">批量删除</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="visibleTalks"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #content="{ record }">
            <span class="talk-content" :title="plainText(record.content)">{{ plainText(record.content) || '—' }}</span>
          </template>
          <template #images="{ record }">
            <div v-if="talkImages(record).length" class="talk-images">
              <AdminImagePreview :src="talkImages(record)[0]" alt="说说图片" :width="76" :height="54" />
              <a-tooltip v-if="talkImages(record).length > 1" :content="`共 ${talkImages(record).length} 张图片`">
                <span class="talk-image-count">+{{ talkImages(record).length - 1 }}</span>
              </a-tooltip>
            </div>
            <span v-else class="admin-muted-cell">—</span>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.status) === 1 ? 'public' : 'private'" />
          </template>
          <template #top="{ record }">
            <a-tooltip :content="Number(record.isTop) === 1 ? '点击取消置顶' : '点击置顶这条说说'">
              <a-switch
                :model-value="Number(record.isTop) === 1"
                :loading="pendingTopId === Number(record.id)"
                @change="(value) => toggleTop(record, value)" />
            </a-tooltip>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="router.push(`/talks/${record.id}`)">编辑</a-button>
              <a-popconfirm content="确定删除这条说说吗？删除后无法恢复。" @ok="deleteTalk(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconMessage"
              :title="hasFilters ? '没有匹配的说说' : '还没有说说'"
              :description="hasFilters ? '换个关键词或重置筛选条件再试一次。' : '发布说说后，会显示在这里。'">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">重置筛选</a-button>
              <a-button v-else type="primary" size="small" @click="router.push('/talks')">发布说说</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconMessage, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, deleteAdminTalks, getAdminTalk, listAdminPage, saveAdminTalk } from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { formatDateTime, isHttpUrl, plainText } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminTalk } from '@stellar-beacon/api-contract'

// `commentCount` is intentionally absent: the admin talk DTO does not expose it
// yet, so the column would always render an empty cell.
const columns = [
  { title: 'ID', dataIndex: 'id', width: 78, slotName: 'id' },
  { title: '内容', dataIndex: 'content', minWidth: 260, slotName: 'content' },
  { title: '图片', dataIndex: 'images', width: 112, slotName: 'images' },
  { title: '作者', dataIndex: 'nickname', width: 130, ellipsis: true, tooltip: true },
  { title: '置顶', dataIndex: 'isTop', width: 86, slotName: 'top' },
  { title: '状态', dataIndex: 'status', width: 96, slotName: 'status' },
  { title: '创建时间', dataIndex: 'createTime', width: 184, slotName: 'time' },
  { title: '操作', dataIndex: 'actions', width: 150, slotName: 'actions' }
]

const VIEW_KEY = 'talks'

const router = useRouter()
const keywords = ref('')
const statusFilter = ref<'all' | '1' | '2'>('all')
const pendingTopId = ref(0)
const selectedKeys = ref<number[]>([])
const batchDeleting = ref(false)

const {
  items: talks,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminTalk>(
  ({ current: page, pageSize: size, signal }) => listAdminPage<AdminTalk>('admin/talks', { current: page, size }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: '说说列表加载失败' }
)

useStoredPageSize(VIEW_KEY, pageSize)
// 说说的关键词与状态都是页内过滤，同步进地址栏只是为了刷新/分享后仍保留条件。
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'status', ref: statusFilter },
  { key: 'page', ref: current }
], { onRestore: () => void load() })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const selectedIds = computed(() =>
  [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
)
const hasFilters = computed(() => Boolean(keywords.value.trim()) || statusFilter.value !== 'all')
/**
 * The talk list endpoint filters by status only, so the keyword box narrows the
 * current page client-side and the caption always reports both numbers.
 */
const visibleTalks = computed(() => {
  const query = keywords.value.trim().toLowerCase()
  const scoped = statusFilter.value === 'all'
    ? talks.value
    : talks.value.filter((talk) => Number(talk.status) === Number(statusFilter.value))
  if (!query) return scoped
  return scoped.filter((talk) => plainText(talk.content).toLowerCase().includes(query))
})

function resetFilters(): void {
  keywords.value = ''
  statusFilter.value = 'all'
  void reload()
}

function changePage(page: number): void {
  clearSelection()
  gotoPage(page)
}

function changePageSize(size: number): void {
  clearSelection()
  applyPageSize(size)
}

function clearSelection(): void {
  selectedKeys.value = []
}

function batchDelete(): void {
  const ids = selectedIds.value
  if (ids.length === 0) return
  Modal.confirm({
    title: '批量删除',
    content: `删除选中的 ${ids.length} 条说说后无法恢复，确定继续吗？`,
    okText: '批量删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      batchDeleting.value = true
      try {
        await deleteAdminTalks(ids)
        Message.success(`已删除 ${ids.length} 条说说`)
        clearSelection()
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, '批量删除失败'))
      } finally {
        batchDeleting.value = false
      }
    }
  })
}

/**
 * The talk API has no dedicated "pin" endpoint, so the toggle re-reads the row
 * and writes it back through the regular save endpoint with `isTop` flipped.
 */
async function toggleTop(talk: AdminTalk, value: boolean | string | number): Promise<void> {
  const id = Number(talk.id)
  if (!Number.isInteger(id) || id <= 0) return
  pendingTopId.value = id
  const next = Boolean(value) ? 1 : 0
  try {
    const full = await getAdminTalk(id)
    await saveAdminTalk({
      id,
      content: String(full.content || ''),
      images: String(full.images || JSON.stringify(full.imgs || [])),
      isTop: next,
      status: Number(full.status ?? 1)
    })
    talk.isTop = next
    Message.success(next === 1 ? '说说已置顶' : '已取消置顶')
  } catch (error) {
    Message.error(apiErrorMessage(error, '置顶状态更新失败'))
  } finally {
    pendingTopId.value = 0
  }
}

async function deleteTalk(id: unknown): Promise<void> {
  const talkId = Number(id)
  if (!Number.isInteger(talkId) || talkId <= 0) return
  try {
    await deleteAdminTalks([talkId])
    if (talks.value.length === 1 && current.value > 1) current.value -= 1
    Message.success('说说已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '说说删除失败'))
  }
}

function talkImages(talk: AdminTalk): string[] {
  if (Array.isArray(talk.imgs)) return talk.imgs.filter(isHttpUrl)
  if (typeof talk.images !== 'string' || !talk.images.trim()) return []
  try {
    const parsed: unknown = JSON.parse(talk.images)
    if (Array.isArray(parsed)) return parsed.map(String).filter(isHttpUrl)
  } catch {
    // Legacy comma-separated image values.
  }
  return talk.images.split(',').map((value) => value.trim()).filter(isHttpUrl)
}
</script>

<style scoped>
.talk-content {
  display: block;
  max-width: 380px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.talk-images {
  display: flex;
  align-items: center;
  gap: 6px;
}

.talk-image-count {
  color: var(--admin-muted);
  font-size: 12px;
  font-weight: 650;
}
</style>
