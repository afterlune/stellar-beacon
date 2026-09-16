<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('media.trash.title')" :description="t('media.trash.description')">
      <template #actions>
        <a-button :disabled="selectedIds.length === 0" :loading="restoring" @click="restoreSelected">
          <template #icon><IconUndo /></template>
          {{ t('media.trash.restoreSelected') }}
        </a-button>
        <a-popconfirm
          :content="t('media.trash.deleteSelectedConfirm')"
          :disabled="selectedIds.length === 0"
          @ok="deleteSelected">
          <a-button status="danger" :disabled="selectedIds.length === 0" :loading="deleting">
            <template #icon><IconDelete /></template>
            {{ t('media.trash.deleteSelected') }}
          </a-button>
        </a-popconfirm>
        <a-button @click="router.push('/albums')">
          <template #icon><IconLeft /></template>
          {{ t('media.shared.backToAlbums') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-alert v-if="selectedIds.length > 0" type="info" class="recycle-selection">
      {{ t('media.trash.selectionHint', { count: selectedIds.length }) }}
      <a-button type="text" size="mini" @click="selectedKeys = []">{{ t('common.deselect') }}</a-button>
    </a-alert>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="keywords"
            class="admin-filter-input"
            :placeholder="t('media.trash.searchPlaceholder')"
            allow-clear />
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">{{ t('media.shared.clearSearch') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('media.trash.total', { total }) }}</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('media.trash.loadFailed')" @retry="load" />

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="photos"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #source="{ record }">
            <AdminImagePreview
              v-if="isHttpUrl(record.photoSrc)"
              :src="String(record.photoSrc)"
              :alt="String(record.photoName || t('image.alt'))"
              :width="84"
              :height="58" />
            <span v-else class="admin-cover-cell" aria-hidden="true"><IconImage /></span>
          </template>
          <template #photoName="{ record }">
            <span class="admin-title-cell">{{ record.photoName || t('media.shared.unnamedPhoto') }}</span>
          </template>
          <template #photoDesc="{ record }">
            <span class="admin-muted-cell" :title="String(record.photoDesc || '')">{{ record.photoDesc || t('media.shared.noDescription') }}</span>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="restore([Number(record.id)])">{{ t('media.trash.restore') }}</a-button>
              <a-popconfirm :content="t('media.trash.deleteConfirm')" @ok="deletePermanently([Number(record.id)])">
                <a-button type="text" status="danger" size="small">{{ t('media.trash.deletePermanently') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconUndo"
              :title="keywords.trim() ? t('media.trash.emptyFiltered') : t('media.trash.empty')"
              :description="keywords.trim() ? t('media.shared.searchAgain') : t('media.trash.emptyHint')">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('media.shared.clearSearch') }}</a-button>
              <a-button v-else size="small" @click="router.push('/albums')">{{ t('media.shared.backToAlbums') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconImage, IconLeft, IconRefresh, IconUndo } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, deleteAdminPhotos, listAdminPage, updateAdminPhotoDelete } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminPhoto } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'photo-trash'

// 列定义随语言切换重算，所以用 computed 而不是模块级常量。
const columns = computed(() => [
  { title: t('common.preview'), dataIndex: 'photoSrc', width: 108, slotName: 'source' },
  { title: t('media.shared.photoName'), dataIndex: 'photoName', slotName: 'photoName', minWidth: 180 },
  { title: t('common.description'), dataIndex: 'photoDesc', slotName: 'photoDesc', ellipsis: true, tooltip: true },
  { title: t('common.actions'), dataIndex: 'actions', width: 176, slotName: 'actions' }
])

const router = useRouter()
const selectedKeys = ref<Array<string | number>>([])
const keywords = ref('')
const restoring = ref(false)
const deleting = ref(false)

const {
  items: photos,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminPhoto>(
  ({ current: page, pageSize: size, signal }) =>
    listAdminPage<AdminPhoto>('admin/photos', { current: page, size, isDelete: 1, keywords: keywords.value.trim() }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY, 18), fallbackMessage: t('media.trash.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const selectedIds = computed(() =>
  [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
)
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value, [18, 36, 72]))

// 分页后丢弃已不在当前页的选中项，保持工具栏的计数与列表一致。
watch(photos, (list) => {
  const available = new Set(list.map((photo) => Number(photo.id)))
  selectedKeys.value = selectedKeys.value.filter((key) => available.has(Number(key)))
})

function changePage(page: number): void {
  gotoPage(page)
}

function changePageSize(size: number): void {
  applyPageSize(size)
}

function clearKeywords(): void {
  keywords.value = ''
}

async function restore(ids: number[]): Promise<void> {
  const validIds = normalizeIds(ids)
  if (!validIds.length) return
  restoring.value = true
  try {
    await updateAdminPhotoDelete(validIds, 0)
    selectedKeys.value = []
    Message.success(t('media.trash.restored', { count: validIds.length }))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.trash.restoreFailed')))
  } finally {
    restoring.value = false
  }
}

async function deletePermanently(ids: number[]): Promise<void> {
  const validIds = normalizeIds(ids)
  if (!validIds.length) return
  deleting.value = true
  try {
    await deleteAdminPhotos(validIds)
    selectedKeys.value = []
    Message.success(t('media.trash.deleted', { count: validIds.length }))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.trash.deleteFailed')))
  } finally {
    deleting.value = false
  }
}

function restoreSelected(): Promise<void> {
  return restore(selectedIds.value)
}

function deleteSelected(): Promise<void> {
  return deletePermanently(selectedIds.value)
}

function normalizeIds(ids: number[]): number[] {
  return [...new Set(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
}
</script>

<style scoped>
.recycle-selection {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
