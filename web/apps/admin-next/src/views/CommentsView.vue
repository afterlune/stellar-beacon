<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('comments.comments.title')" :description="t('comments.comments.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('comments.comments.searchPlaceholder')"
          allow-clear
          @search="reload" />
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-radio-group v-model="reviewFilter" type="button" size="small" @change="reload">
            <a-radio :value="0">{{ t('common.all') }}</a-radio>
            <a-radio :value="2">{{ t('status.pending') }}</a-radio>
            <a-radio :value="1">{{ t('comments.comments.filterApproved') }}</a-radio>
          </a-radio-group>
          <a-select v-model="collectionFilter" size="small" class="admin-filter-select" @change="reload">
            <a-option :value="0">{{ t('comments.comments.collectionAll') }}</a-option>
            <a-option v-for="item in collections" :key="item.id" :value="Number(item.id)">{{ item.title }}</a-option>
          </a-select>
          <span class="admin-toolbar-caption">{{ t('comments.comments.total', { total }) }}</span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button
            v-if="pendingCount > 0"
            size="small"
            type="primary"
            :loading="approvingAll"
            @click="approveAllPending">
            {{ t('comments.comments.approvePagePending', { count: pendingCount }) }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('comments.comments.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('comments.common.pageCount', { count: records.length })" @clear="clearSelection">
        <a-button size="small" type="primary" :loading="batchApproving" @click="batchReview">{{ t('comments.comments.batchApprove') }}</a-button>
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">{{ t('comments.common.batchDelete') }}</a-button>
        <a-button size="small" :loading="batchRestoring" @click="batchRestore">{{ t('comments.comments.batchRestore') }}</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="records"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #article="{ record }">
            <span v-if="record.articleTitle" :title="String(record.articleTitle)" class="comment-article">
              <small v-if="Number(record.type) === 6">书单 · </small>{{ record.articleTitle }}
            </span>
            <span v-else class="admin-muted-cell">{{ t('comments.comments.noArticle') }}</span>
          </template>
          <template #content="{ record }">
            <span class="comment-content" :title="plainText(record.commentContent)">{{ plainText(record.commentContent) || '—' }}</span>
          </template>
          <template #review="{ record }">
            <AdminStatusTag :kind="Number(record.isReview) === 1 ? 'reviewed' : 'pending'" />
          </template>
          <template #status="{ record }">
            <a-space size="mini">
              <AdminStatusTag v-if="Number(record.isDelete) === 1" kind="failed" :label="t('comments.comments.softDeleted')" />
              <AdminStatusTag v-if="Number(record.isTop) === 1" kind="top" />
              <span v-if="Number(record.reportCount) > 0" class="comment-report-count">{{ t('comments.comments.reports') }} {{ record.reportCount }}</span>
            </a-space>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space v-if="Number(record.isDelete) === 1" class="admin-action-space">
              <a-popconfirm :content="t('comments.comments.restoreConfirm')" @ok="restoreOne(record.id)">
                <a-button type="text" size="small" :loading="isPending(record.id)">{{ t('comments.comments.restore') }}</a-button>
              </a-popconfirm>
            </a-space>
            <a-space v-else class="admin-action-space">
              <a-button type="text" size="small" :loading="isPending(record.id)" @click="toggleReview(record)">
                {{ Number(record.isReview) === 1 ? t('comments.comments.revokeReview') : t('comments.comments.approve') }}
              </a-button>
              <a-popconfirm :content="t('comments.comments.deleteConfirm')" @ok="remove(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconMessage"
              :title="keywords.trim() ? t('comments.comments.emptySearch') : t('comments.comments.empty')"
              :description="keywords.trim() ? t('comments.common.noMatch') : t('comments.comments.emptyHint')">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('comments.common.clearSearch') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
    <CommentGovernanceQueues />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconMessage, IconRefresh } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteComment,
  deleteComments,
  getAdminCollections,
  listAdminPage,
  listCollectionComments,
  restoreComment,
  reviewComment,
  reviewComments
} from '@/api/http'
import CommentGovernanceQueues from '@/components/CommentGovernanceQueues.vue'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { usePendingIds } from '@/composables/usePendingIds'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import type { CollectionSummary } from '@stellar-beacon/api-contract'
import { t } from '@/i18n'
import { formatDateTime, plainText } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

interface CommentRow extends Record<string, unknown> {
  id: number
  type?: number
  isReview?: number
  commentContent?: string
  createTime?: string
}

const VIEW_KEY = 'comments'

// 列定义必须在 computed 里生成：它只在 setup 时求值一次，语言切换后不会再更新。
const columns = computed(() => [
  { title: 'ID', dataIndex: 'id', width: 78, slotName: 'id' },
  { title: t('comments.comments.user'), dataIndex: 'nickname', width: 140, ellipsis: true, tooltip: true },
  { title: t('comments.comments.article'), dataIndex: 'articleTitle', width: 200, slotName: 'article' },
  { title: t('comments.common.content'), dataIndex: 'commentContent', minWidth: 260, slotName: 'content' },
  { title: t('comments.comments.review'), dataIndex: 'isReview', width: 104, slotName: 'review' },
  { title: t('comments.comments.status'), dataIndex: 'isDelete', width: 150, slotName: 'status' },
  { title: t('comments.common.createdAt'), dataIndex: 'createTime', width: 184, slotName: 'time' },
  { title: t('common.actions'), dataIndex: 'actions', width: 168, slotName: 'actions' }
])

const keywords = ref('')
const reviewFilter = ref(0)
const collectionFilter = ref(0)
const collections = ref<CollectionSummary[]>([])
const selectedKeys = ref<number[]>([])
const approvingAll = ref(false)
const batchApproving = ref(false)
const batchDeleting = ref(false)
const batchRestoring = ref(false)

const { isPending, withPending } = usePendingIds()

const {
  items: records,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<CommentRow>(
  ({ current: page, pageSize: size, signal }) => {
    const params = {
      current: page,
      size,
      keywords: keywords.value.trim(),
      isReview: reviewFilter.value
    }
    return Number(collectionFilter.value) > 0
      ? listCollectionComments(Number(collectionFilter.value), { ...params, type: 6 }, { signal })
      : listAdminPage<CommentRow>('admin/comments', params, { signal })
  },
  // fallbackMessage 只在 setup 时取一次值（useAsyncList 的参数是普通字符串），
  // 因此这里保留当前语言的快照；错误块的标题会跟着语言切换重新渲染。
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('comments.comments.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'isReview', ref: reviewFilter },
  { key: 'collectionId', ref: collectionFilter },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const pendingCount = computed(() => records.value.filter((row) => Number(row.isReview) !== 1).length)
const selectedIds = computed(() => selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

function clearSelection(): void {
  selectedKeys.value = []
}

function changePage(page: number): void {
  clearSelection()
  gotoPage(page)
}

function changePageSize(size: number): void {
  clearSelection()
  applyPageSize(size)
}

async function toggleReview(row: CommentRow): Promise<void> {
  const next = Number(row.isReview) === 1 ? 0 : 1
  await withPending(row.id, async () => {
    try {
      await reviewComment(Number(row.id), next)
      row.isReview = next
      Message.success(next === 1 ? t('comments.comments.approved') : t('comments.comments.reviewRevoked'))
      await load()
    } catch (error) {
      Message.error(apiErrorMessage(error, t('comments.comments.reviewFailed')))
    }
  })
}

async function batchReview(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0) return
  batchApproving.value = true
  try {
    await reviewComments(ids, 1)
    Message.success(t('comments.comments.approvedCount', { count: ids.length }))
    clearSelection()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.comments.batchApproveFailed')))
  } finally {
    batchApproving.value = false
  }
}

async function batchDelete(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0) return
  batchDeleting.value = true
  try {
    await deleteComments(ids)
    Message.success(t('comments.comments.deletedCount', { count: ids.length }))
    clearSelection()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.common.batchDeleteFailed')))
  } finally {
    batchDeleting.value = false
  }
}

async function approveAllPending(): Promise<void> {
  const pending = records.value.filter((row) => Number(row.isReview) !== 1)
  if (pending.length === 0) return
  approvingAll.value = true
  try {
    await reviewComments(pending.map((row) => Number(row.id)), 1)
    Message.success(t('comments.comments.approvedPageCount', { count: pending.length }))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.comments.batchApproveFailed')))
    await load()
  } finally {
    approvingAll.value = false
  }
}

async function loadCollections(): Promise<void> {
  try {
    const page = await getAdminCollections({ current: 1, size: 100 })
    collections.value = page.items ?? []
  } catch {
    collections.value = []
  }
}

async function restoreOne(id: unknown): Promise<void> {
  const commentId = Number(id)
  if (!Number.isInteger(commentId) || commentId <= 0) return
  await withPending(commentId, async () => {
    try {
      await restoreComment(commentId)
      Message.success(t('comments.comments.restored'))
      await load()
    } catch (error) {
      Message.error(apiErrorMessage(error, t('comments.comments.restoreFailed')))
    }
  })
}

async function batchRestore(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0) return
  batchRestoring.value = true
  try {
    let restored = 0
    for (const id of ids) {
      try {
        await restoreComment(id)
        restored += 1
      } catch {
        // 单条失败不阻塞其余恢复，最终以成功数量提示。
      }
    }
    Message.success(t('comments.comments.restoredCount', { count: restored }))
    clearSelection()
    await load()
  } finally {
    batchRestoring.value = false
  }
}

onMounted(() => void loadCollections())
async function remove(id: unknown): Promise<void> {
  const commentId = Number(id)
  if (!Number.isInteger(commentId) || commentId <= 0) return
  try {
    await deleteComment(commentId)
    if (records.value.length === 1 && current.value > 1) current.value -= 1
    await load()
    Message.success(t('comments.comments.deleted'))
  } catch (error) {
    Message.error(apiErrorMessage(error, t('common.deleteFailed')))
  }
}
</script>

<style scoped>
.comment-content {
  display: inline-block;
  max-width: 340px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}

.comment-article {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}

.admin-filter-select {
  width: 200px;
}

.comment-report-count {
  color: var(--color-warning, #d2761b);
  font-size: 12px;
  white-space: nowrap;
}
</style>
