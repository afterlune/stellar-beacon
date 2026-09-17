<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('articles.list.title')" :description="t('articles.list.description')">
      <template #actions>
        <input ref="importInput" type="file" accept=".md,.markdown,.txt" hidden @change="onImportFile" />
        <a-button :loading="importing" @click="pickImportFile">
          <template #icon><IconUpload /></template>
          {{ t('articles.list.import') }}
        </a-button>
        <a-button type="primary" @click="router.push('/articles')">
          <template #icon><IconPlus /></template>
          {{ t('articles.actions.publish') }}
        </a-button>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="keywords"
            class="admin-filter-input"
            :placeholder="t('articles.list.searchPlaceholder')"
            allow-clear
            @search="reload" />
          <a-select v-model="status" :placeholder="t('articles.list.allStatuses')" allow-clear style="width: 130px" @change="reload">
            <a-option :value="1">{{ t('status.published') }}</a-option>
            <a-option :value="2">{{ t('status.private') }}</a-option>
            <a-option :value="3">{{ t('status.draft') }}</a-option>
          </a-select>
          <a-select v-model="type" :placeholder="t('articles.list.allTypes')" allow-clear style="width: 130px" @change="reload">
            <a-option :value="1">{{ t('status.original') }}</a-option>
            <a-option :value="2">{{ t('status.reprint') }}</a-option>
            <a-option :value="3">{{ t('status.translate') }}</a-option>
          </a-select>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">{{ t('articles.list.resetFilters') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('articles.list.total', { total }) }}</span>
          <a-dropdown trigger="click" position="br">
            <a-button size="small">
              <template #icon><IconSettings /></template>
              {{ t('articles.list.columnSettings') }}
            </a-button>
            <template #content>
              <div class="admin-column-settings">
                <div class="admin-column-settings-head">
                  <span>{{ t('articles.list.visibleColumns') }}</span>
                  <a-button type="text" size="mini" @click="columnPrefs.reset">{{ t('articles.list.showAllColumns') }}</a-button>
                </div>
                <a-checkbox
                  v-for="column in columns"
                  :key="column.dataIndex"
                  :model-value="columnPrefs.isVisible(column.dataIndex)"
                  :disabled="column.dataIndex === 'actions'"
                  @change="(value) => columnPrefs.toggle(column.dataIndex, Boolean(value))">
                  {{ column.title }}
                </a-checkbox>
              </div>
            </template>
          </a-dropdown>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('articles.list.loadFailed')" @retry="load" />

      <AdminBatchBar
        :count="selectedKeys.length"
        :hint="t('articles.list.pageHint', { count: records.length })"
        @clear="clearSelection">
        <a-button size="small" @click="exportSelected">
          <template #icon><IconDownload /></template>
          {{ t('articles.actions.exportMarkdown') }}
        </a-button>
        <a-button size="small" @click="batchTrash">{{ t('articles.actions.trash') }}</a-button>
        <a-button size="small" status="danger" @click="batchDelete">{{ t('articles.actions.deleteForever') }}</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
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
              :alt="t('articles.list.coverAlt', { title: String(record.articleTitle || t('articles.list.articleFallback')) })"
              :width="88"
              :height="60" />
            <span v-else class="admin-cover-cell" aria-hidden="true"><IconBook /></span>
          </template>
          <template #title="{ record }">
            <span class="admin-title-cell" :title="String(record.articleTitle || '')">{{ record.articleTitle || t('articles.list.untitled') }}</span>
          </template>
          <template #category="{ record }">
            <a-tag v-if="record.categoryName" color="arcoblue">{{ record.categoryName }}</a-tag>
            <span v-else class="admin-muted-cell">{{ t('articles.list.uncategorized') }}</span>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="statusKind(record.status)" />
          </template>
          <template #type="{ record }">
            <span class="admin-muted-cell">{{ typeLabel(record.type) }}</span>
          </template>
          <template #flags="{ record }">
            <!-- 间距收到 4px：默认 8px 会让「置顶 + 精选」在 120px 列里折行，把整行撑高 -->
            <a-space v-if="Number(record.isTop) === 1 || Number(record.isFeatured) === 1" wrap :size="4">
              <a-tag v-if="Number(record.isTop) === 1" color="arcoblue">{{ t('status.pinned') }}</a-tag>
              <a-tag v-if="Number(record.isFeatured) === 1" color="green">{{ t('articles.featured') }}</a-tag>
            </a-space>
            <span v-else class="admin-muted-cell">—</span>
          </template>
          <template #views="{ record }">
            <span class="admin-num-cell">{{ formatNumber(record.viewsCount) }}</span>
          </template>
          <template #likes="{ record }">
            <span class="admin-num-cell" data-testid="article-like-count">{{ formatNumber(record.likeCount) }}</span>
          </template>
          <template #favorites="{ record }">
            <span class="admin-num-cell" data-testid="article-favorite-count">{{ formatNumber(record.favoriteCount) }}</span>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="editArticle(record.id)">{{ t('common.edit') }}</a-button>
              <a-dropdown trigger="click" position="br">
                <a-button type="text" size="small" :loading="isPending(record.id)">
                  {{ t('common.more') }}
                  <template #icon><IconDown /></template>
                </a-button>
                <template #content>
                  <a-doption @click="toggleFlag(record, 'isTop')">
                    {{ Number(record.isTop) === 1 ? t('articles.list.unsetTop') : t('articles.list.setTop') }}
                  </a-doption>
                  <a-doption @click="toggleFlag(record, 'isFeatured')">
                    {{ Number(record.isFeatured) === 1 ? t('articles.list.unsetFeatured') : t('articles.list.setFeatured') }}
                  </a-doption>
                  <a-doption @click="moveToTrash(record)">{{ t('articles.actions.trash') }}</a-doption>
                  <a-doption class="admin-danger-option" @click="removeArticle(record)">{{ t('articles.actions.deleteForever') }}</a-doption>
                </template>
              </a-dropdown>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconBook"
              :title="hasFilters ? t('articles.list.emptyFiltered') : t('articles.list.empty')"
              :description="hasFilters ? t('articles.list.emptyFilteredHint') : t('articles.list.emptyHint')">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">{{ t('articles.list.clearFilters') }}</a-button>
              <a-button v-else type="primary" size="small" @click="router.push('/articles')">{{ t('articles.actions.publish') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="exportVisible" :title="t('articles.list.exportTitle')" :footer="false" width="560px">
      <p class="admin-export-hint">{{ t('articles.list.exportHint') }}</p>
      <div class="admin-export-list">
        <div v-for="(url, index) in exportUrls" :key="url" class="admin-export-item">
          <span class="admin-export-index">{{ index + 1 }}</span>
          <a-link :href="url" target="_blank" rel="noopener" class="admin-export-link">{{ url }}</a-link>
          <a-button size="mini" @click="copyText(url, { success: t('articles.list.exportLinkCopied') })">{{ t('articles.actions.copyLink') }}</a-button>
        </div>
      </div>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconBook,
  IconDown,
  IconDownload,
  IconPlus,
  IconRefresh,
  IconSettings,
  IconUpload
} from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteAdminArticles,
  exportAdminArticles,
  importAdminArticles,
  listAdminPage,
  updateAdminArticleFeatured,
  updateAdminArticleTrash
} from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag, { type StatusKind } from '@/components/AdminStatusTag.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { usePendingIds } from '@/composables/usePendingIds'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useColumnPrefs, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { copyText } from '@/utils/clipboard'
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

const VIEW_KEY = 'article-list'

const router = useRouter()
const keywords = ref('')
const status = ref<number | undefined>(undefined)
const type = ref<number | undefined>(undefined)
const selectedKeys = ref<number[]>([])
const importInput = ref<HTMLInputElement | null>(null)
const importing = ref(false)
const exportVisible = ref(false)
const exportUrls = ref<string[]>([])

const { isPending, withPending } = usePendingIds()

// 列宽预算：这里是按"列内容实际需要多宽"倒推出来的，不是随手填的数。
// - 封面 116 = 88px 缩略图 + 左右内边距（原来 108 会把缩略图裁掉一截）
// - 操作 148 = 「编辑」+「更多⌄」两个按钮 + 间距（原来收得太窄时「更多」被裁没）
// - 标记 128 = 置顶 + 精选 两个标签并排（96px 内容 + 间距），否则会折行把整行撑高
// - 标题给死 220：这一列原来不给宽度，1280 宽的窗口里被固定列宽挤成 0px，
//   表格里最重要的信息直接消失，是这次改版修掉的真实缺陷。
// 标题文案走 t()，所以做成 computed：语言切换后表头与「列设置」里的复选框一起更新。
const columns = computed<TableColumn[]>(() => [
  { title: t('articles.list.columnCover'), dataIndex: 'articleCover', slotName: 'cover', width: 116 },
  { title: t('common.title'), dataIndex: 'articleTitle', slotName: 'title', width: 220, ellipsis: true, tooltip: true },
  { title: t('articles.category'), dataIndex: 'categoryName', slotName: 'category', width: 96 },
  { title: t('common.status'), dataIndex: 'status', slotName: 'status', width: 84 },
  { title: t('common.type'), dataIndex: 'type', slotName: 'type', width: 62 },
  { title: t('articles.list.columnFlags'), dataIndex: 'flags', slotName: 'flags', width: 128 },
  { title: t('articles.list.columnViews'), dataIndex: 'viewsCount', slotName: 'views', width: 74 },
  { title: t('articles.list.columnLikes'), dataIndex: 'likeCount', slotName: 'likes', width: 74 },
  { title: t('articles.list.columnFavorites'), dataIndex: 'favoriteCount', slotName: 'favorites', width: 78 },
  { title: t('articles.list.columnCreatedAt'), dataIndex: 'createTime', slotName: 'time', width: 164 },
  { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 148 }
])

const columnPrefs = useColumnPrefs(VIEW_KEY, columns.value.map((column) => column.dataIndex))

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
} = useAsyncList<Record<string, unknown>>(
  ({ current: page, pageSize: size, signal }) => listAdminPage<Record<string, unknown>>('admin/articles', {
    current: page,
    size,
    keywords: keywords.value.trim(),
    status: status.value ?? 0,
    type: type.value ?? 0
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('articles.list.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'status', ref: status },
  { key: 'type', ref: type },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

// 每一列都显式给一个 slot：默认的 `formatted` 槽位会绕过共享的时间/数字格式化。
const tableColumns = computed(() => columns.value
  .filter((column) => columnPrefs.isVisible(column.dataIndex))
  .map((column) => ({ ...column, slotName: column.slotName || 'formatted' })))

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() => Boolean(keywords.value.trim()) || status.value !== undefined || type.value !== undefined)

function resetFilters(): void {
  keywords.value = ''
  status.value = undefined
  type.value = undefined
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

function selectedIds(): number[] {
  return selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0)
}

/** 批量动作的公共收尾：清空选中、重新加载、错误只弹一次。 */
async function runBatch(action: () => Promise<void>, failureMessage: string): Promise<void> {
  try {
    await action()
    clearSelection()
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, failureMessage))
  }
}

function batchTrash(): void {
  const ids = selectedIds()
  if (ids.length === 0) return
  Modal.confirm({
    title: t('articles.actions.trash'),
    content: t('articles.list.confirmBatchTrash', { count: ids.length }),
    okText: t('articles.actions.trash'),
    cancelText: t('common.cancel'),
    onOk: () => runBatch(async () => {
      await updateAdminArticleTrash(ids, 1)
      Message.success(t('articles.list.trashedCount', { count: ids.length }))
    }, t('articles.list.trashFailed'))
  })
}

function batchDelete(): void {
  const ids = selectedIds()
  if (ids.length === 0) return
  Modal.confirm({
    title: t('articles.actions.deleteForever'),
    content: t('articles.list.confirmBatchDelete', { count: ids.length }),
    okText: t('articles.actions.deleteForever'),
    cancelText: t('common.cancel'),
    okButtonProps: { status: 'danger' },
    onOk: () => runBatch(async () => {
      await deleteAdminArticles(ids)
      Message.success(t('articles.list.deletedCount', { count: ids.length }))
    }, t('common.deleteFailed'))
  })
}

async function exportSelected(): Promise<void> {
  const ids = selectedIds()
  if (ids.length === 0) return
  try {
    const urls = await exportAdminArticles(ids)
    if (urls.length === 0) {
      Message.warning(t('articles.list.exportEmpty'))
      return
    }
    exportUrls.value = urls
    exportVisible.value = true
  } catch (error) {
    Message.error(apiErrorMessage(error, t('articles.list.exportFailed')))
  }
}

function pickImportFile(): void {
  importInput.value?.click()
}

async function onImportFile(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  // 清空 value，保证连续导入同一个文件也能再次触发 change。
  input.value = ''
  if (!file) return
  importing.value = true
  try {
    await importAdminArticles(file)
    Message.success(t('articles.list.importSuccess', { name: file.name }))
    await reload()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('articles.list.importFailed')))
  } finally {
    importing.value = false
  }
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
  await withPending(id, async () => {
    try {
      await updateAdminArticleFeatured({
        id,
        isTop: field === 'isTop' ? next : Number(record.isTop) === 1 ? 1 : 0,
        isFeatured: field === 'isFeatured' ? next : Number(record.isFeatured) === 1 ? 1 : 0
      })
      record[field] = next
      Message.success(next === 1
        ? (field === 'isTop' ? t('articles.list.markedTop') : t('articles.list.markedFeatured'))
        : t('articles.list.markCleared'))
    } catch (error) {
      Message.error(apiErrorMessage(error, t('articles.list.markFailed')))
    }
  })
}

function moveToTrash(record: Record<string, unknown>): void {
  const id = Number(record.id)
  if (!Number.isInteger(id) || id <= 0) return
  Modal.confirm({
    title: t('articles.actions.trash'),
    content: t('articles.list.confirmTrash', { title: String(record.articleTitle || t('articles.list.untitled')) }),
    okText: t('articles.actions.trash'),
    cancelText: t('common.cancel'),
    onOk: () => withPending(id, async () => {
      try {
        await updateAdminArticleTrash([id], 1)
        Message.success(t('articles.list.trashed'))
        if (records.value.length === 1 && current.value > 1) current.value -= 1
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, t('articles.list.trashFailed')))
      }
    })
  })
}

function removeArticle(record: Record<string, unknown>): void {
  const id = Number(record.id)
  if (!Number.isInteger(id) || id <= 0) return
  Modal.confirm({
    title: t('articles.actions.deleteForever'),
    content: t('articles.list.confirmDelete', { title: String(record.articleTitle || t('articles.list.untitled')) }),
    okText: t('articles.actions.deleteForever'),
    cancelText: t('common.cancel'),
    okButtonProps: { status: 'danger' },
    onOk: () => withPending(id, async () => {
      try {
        await deleteAdminArticles([id])
        Message.success(t('articles.list.deleted'))
        if (records.value.length === 1 && current.value > 1) current.value -= 1
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, t('common.deleteFailed')))
      }
    })
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
  if (raw === 1) return t('status.original')
  if (raw === 2) return t('status.reprint')
  if (raw === 3) return t('status.translate')
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

.admin-export-hint {
  margin: 0 0 12px;
  color: var(--admin-muted);
  font-size: 13px;
}

.admin-export-list {
  display: grid;
  gap: 6px;
  max-height: 320px;
  overflow-y: auto;
}

.admin-export-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
}

.admin-export-index {
  flex: none;
  width: 20px;
  color: var(--admin-muted);
  font-size: 12px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.admin-export-link {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
</style>

<style>
/* 下拉菜单里的危险操作：与普通项区分开 */
.admin-danger-option {
  color: var(--admin-danger) !important;
}
</style>
