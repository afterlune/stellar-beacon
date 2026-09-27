<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('comments.talks.title')" :description="t('comments.talks.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('comments.talks.searchPlaceholder')"
          allow-clear />
        <a-button v-if="keywords.trim()" @click="keywords = ''">{{ t('comments.common.clearSearch') }}</a-button>
        <a-button type="primary" @click="router.push('/talks')">
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
            <a-radio value="1">{{ t('status.published') }}</a-radio>
            <a-radio value="2">{{ t('status.private') }}</a-radio>
          </a-radio-group>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">{{ t('comments.talks.resetFilters') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('comments.talks.total', { total, count: visibleTalks.length }) }}</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('comments.talks.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('comments.common.pageCount', { count: visibleTalks.length })" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" :disabled="!allSelectedOwned" @click="batchDelete">{{ t('comments.common.batchDelete') }}</a-button>
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
              <AdminImagePreview :src="talkImages(record)[0]" :alt="t('comments.talks.imageAlt')" :width="76" :height="54" />
              <a-tooltip v-if="talkImages(record).length > 1" :content="t('comments.talks.imageCount', { count: talkImages(record).length })">
                <span class="talk-image-count">+{{ talkImages(record).length - 1 }}</span>
              </a-tooltip>
            </div>
            <span v-else class="admin-muted-cell">—</span>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.status) === 1 ? 'public' : 'private'" />
          </template>
          <template #top="{ record }">
            <a-tooltip v-if="isOwner(record)" :content="Number(record.isTop) === 1 ? t('comments.talks.unpinHint') : t('comments.talks.pinHint')">
              <a-switch
                :model-value="Number(record.isTop) === 1"
                :loading="pendingTopId === Number(record.id)"
                @change="(value) => toggleTop(record, value)" />
            </a-tooltip>
            <a-tag v-else :color="Number(record.isTop) === 1 ? 'arcoblue' : 'gray'">
              {{ Number(record.isTop) === 1 ? t('status.pinned') : '—' }}
            </a-tag>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button v-if="isOwner(record)" type="text" size="small" @click="router.push(`/talks/${record.id}`)">{{ t('common.edit') }}</a-button>
              <a-popconfirm v-if="isOwner(record)" :content="t('comments.talks.deleteConfirm')" @ok="deleteTalk(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconMessage"
              :title="hasFilters ? t('comments.talks.emptySearch') : t('comments.talks.empty')"
              :description="hasFilters ? t('comments.talks.emptySearchHint') : t('comments.talks.emptyHint')">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">{{ t('comments.talks.resetFilters') }}</a-button>
              <a-button v-else type="primary" size="small" @click="router.push('/talks')">{{ t('comments.talks.create') }}</a-button>
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
import { t } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime, isHttpUrl, plainText } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminTalk } from '@stellar-beacon/api-contract'

// `commentCount` is intentionally absent: the admin talk DTO does not expose it
// yet, so the column would always render an empty cell.
// 列定义必须在 computed 里生成：它只在 setup 时求值一次，语言切换后不会再更新。
const columns = computed(() => [
  { title: 'ID', dataIndex: 'id', width: 78, slotName: 'id' },
  { title: t('comments.common.content'), dataIndex: 'content', minWidth: 260, slotName: 'content' },
  { title: t('common.image'), dataIndex: 'images', width: 112, slotName: 'images' },
  { title: t('comments.talks.author'), dataIndex: 'nickname', width: 130, ellipsis: true, tooltip: true },
  { title: t('status.pinned'), dataIndex: 'isTop', width: 86, slotName: 'top' },
  { title: t('common.status'), dataIndex: 'status', width: 96, slotName: 'status' },
  { title: t('comments.common.createdAt'), dataIndex: 'createTime', width: 184, slotName: 'time' },
  { title: t('common.actions'), dataIndex: 'actions', width: 150, slotName: 'actions' }
])

const VIEW_KEY = 'talks'

const router = useRouter()
const auth = useAuthStore()
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
  // fallbackMessage 只在 setup 时取一次值（useAsyncList 的参数是普通字符串），
  // 因此这里保留当前语言的快照；错误块的标题会跟着语言切换重新渲染。
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('comments.talks.loadFailed') }
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
const allSelectedOwned = computed(() => selectedIds.value.length > 0 && selectedIds.value.every((id) => {
  const record = talks.value.find((item) => Number(item.id) === id)
  return isOwner(record)
}))

function isOwner(record?: AdminTalk): boolean {
  const currentUserId = Number(auth.user?.userInfoId || auth.user?.id || 0)
  return Boolean(record && currentUserId > 0 && Number(record.userId) === currentUserId)
}
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
  if (!allSelectedOwned.value) return
  if (ids.length === 0) return
  Modal.confirm({
    title: t('comments.common.batchDelete'),
    content: t('comments.talks.batchDeleteConfirm', { count: ids.length }),
    okText: t('comments.common.batchDelete'),
    cancelText: t('common.cancel'),
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      batchDeleting.value = true
      try {
        await deleteAdminTalks(ids)
        Message.success(t('comments.talks.deletedCount', { count: ids.length }))
        clearSelection()
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, t('comments.common.batchDeleteFailed')))
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
    Message.success(next === 1 ? t('comments.talks.pinned') : t('comments.talks.unpinned'))
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.talks.pinFailed')))
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
    Message.success(t('comments.talks.deleted'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.talks.deleteFailed')))
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
