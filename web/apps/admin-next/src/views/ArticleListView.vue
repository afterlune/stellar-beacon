<template>
  <section class="admin-page">
    <AdminPageHeader title="文章列表" description="把文章、状态和发布节奏整理在一张清晰的桌面上。">
      <template #actions>
        <input ref="importInput" type="file" accept=".md,.markdown,.txt" hidden @change="onImportFile" />
        <a-button :loading="importing" @click="pickImportFile">
          <template #icon><IconUpload /></template>
          导入文章
        </a-button>
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
          <a-dropdown trigger="click" position="br">
            <a-button size="small">
              <template #icon><IconSettings /></template>
              列设置
            </a-button>
            <template #content>
              <div class="admin-column-settings">
                <div class="admin-column-settings-head">
                  <span>显示列</span>
                  <a-button type="text" size="mini" @click="columnPrefs.reset">全部显示</a-button>
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

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="文章列表加载失败" @retry="load" />

      <AdminBatchBar
        :count="selectedKeys.length"
        :hint="`本页 ${records.length} 篇`"
        @clear="clearSelection">
        <a-button size="small" @click="exportSelected">
          <template #icon><IconDownload /></template>
          导出 Markdown
        </a-button>
        <a-button size="small" @click="batchTrash">移入回收站</a-button>
        <a-button size="small" status="danger" @click="batchDelete">永久删除</a-button>
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
            <!-- 间距收到 4px：默认 8px 会让「置顶 + 精选」在 120px 列里折行，把整行撑高 -->
            <a-space v-if="Number(record.isTop) === 1 || Number(record.isFeatured) === 1" wrap :size="4">
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
                <a-button type="text" size="small" :loading="isPending(record.id)">
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

    <a-modal v-model:visible="exportVisible" title="导出结果" :footer="false" width="560px">
      <p class="admin-export-hint">后端已把选中的文章导出为 Markdown 文件，可以打开或复制链接。</p>
      <div class="admin-export-list">
        <div v-for="(url, index) in exportUrls" :key="url" class="admin-export-item">
          <span class="admin-export-index">{{ index + 1 }}</span>
          <a-link :href="url" target="_blank" rel="noopener" class="admin-export-link">{{ url }}</a-link>
          <a-button size="mini" @click="copyText(url, { success: '导出链接已复制' })">复制链接</a-button>
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
const columns: TableColumn[] = [
  { title: '封面', dataIndex: 'articleCover', slotName: 'cover', width: 116 },
  { title: '标题', dataIndex: 'articleTitle', slotName: 'title', width: 220, ellipsis: true, tooltip: true },
  { title: '分类', dataIndex: 'categoryName', slotName: 'category', width: 96 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 84 },
  { title: '类型', dataIndex: 'type', slotName: 'type', width: 62 },
  { title: '标记', dataIndex: 'flags', slotName: 'flags', width: 128 },
  { title: '浏览量', dataIndex: 'viewsCount', slotName: 'views', width: 74 },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'time', width: 164 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 148 }
]

const columnPrefs = useColumnPrefs(VIEW_KEY, columns.map((column) => column.dataIndex))

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
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: '文章列表加载失败' }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'status', ref: status },
  { key: 'type', ref: type },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

// 每一列都显式给一个 slot：默认的 `formatted` 槽位会绕过共享的时间/数字格式化。
const tableColumns = computed(() => columns
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
    title: '移入回收站',
    content: `确定把选中的 ${ids.length} 篇文章移入回收站吗？之后可以在回收站中恢复。`,
    okText: '移入回收站',
    cancelText: '取消',
    onOk: () => runBatch(async () => {
      await updateAdminArticleTrash(ids, 1)
      Message.success(`已移入回收站 ${ids.length} 篇`)
    }, '移入回收站失败')
  })
}

function batchDelete(): void {
  const ids = selectedIds()
  if (ids.length === 0) return
  Modal.confirm({
    title: '永久删除',
    content: `永久删除选中的 ${ids.length} 篇文章后无法恢复，确定继续吗？`,
    okText: '永久删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: () => runBatch(async () => {
      await deleteAdminArticles(ids)
      Message.success(`已删除 ${ids.length} 篇文章`)
    }, '删除失败')
  })
}

async function exportSelected(): Promise<void> {
  const ids = selectedIds()
  if (ids.length === 0) return
  try {
    const urls = await exportAdminArticles(ids)
    if (urls.length === 0) {
      Message.warning('没有可导出的文章')
      return
    }
    exportUrls.value = urls
    exportVisible.value = true
  } catch (error) {
    Message.error(apiErrorMessage(error, '导出失败'))
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
    Message.success(`已导入《${file.name}》，默认保存为草稿`)
    await reload()
  } catch (error) {
    Message.error(apiErrorMessage(error, '导入失败，请确认文件为 Markdown 或文本'))
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
      Message.success(next === 1 ? (field === 'isTop' ? '已设为置顶' : '已设为精选') : '已取消标记')
    } catch (error) {
      Message.error(apiErrorMessage(error, '标记更新失败'))
    }
  })
}

function moveToTrash(record: Record<string, unknown>): void {
  const id = Number(record.id)
  if (!Number.isInteger(id) || id <= 0) return
  Modal.confirm({
    title: '移入回收站',
    content: `确定把《${String(record.articleTitle || '未命名文章')}》移入回收站吗？之后可以在回收站中恢复。`,
    okText: '移入回收站',
    cancelText: '取消',
    onOk: () => withPending(id, async () => {
      try {
        await updateAdminArticleTrash([id], 1)
        Message.success('已移入回收站')
        if (records.value.length === 1 && current.value > 1) current.value -= 1
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, '移入回收站失败'))
      }
    })
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
    onOk: () => withPending(id, async () => {
      try {
        await deleteAdminArticles([id])
        Message.success('文章已删除')
        if (records.value.length === 1 && current.value > 1) current.value -= 1
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, '删除失败'))
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
