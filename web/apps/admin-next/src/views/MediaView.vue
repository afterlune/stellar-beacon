<template>
  <section class="admin-page media-page">
    <AdminPageHeader :title="t('media.library.title')" :description="t('media.library.description')">
      <template #actions>
        <input ref="fileInput" type="file" accept="image/*" multiple hidden @change="uploadFiles" />
        <a-button type="primary" :loading="uploading" @click="fileInput?.click()">
          <template #icon><IconUpload /></template>
          {{ uploading ? uploadLabel : t('media.library.upload') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="prefix"
            class="admin-filter-input"
            :placeholder="t('mediaPicker.filterHint')"
            allow-clear
            @search="reload" />
          <a-button v-if="prefix.trim()" type="text" size="small" @click="clearPrefix">{{ t('media.library.clearFilter') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('media.library.total', { total }) }}</span>
          <a-button v-if="assets.length" size="small" @click="toggleSelectAll">
            {{ selectedKeys.length === assets.length ? t('media.library.deselectAll') : t('media.library.selectAllPage') }}
          </a-button>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('mediaPicker.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedKeys.length" @clear="selectedKeys = []">
        <a-popconfirm
          :content="t('media.library.deleteSelectedConfirm', { count: selectedKeys.length })"
          :disabled="selectedKeys.length === 0"
          @ok="removeSelected">
          <a-button status="danger" :disabled="selectedKeys.length === 0">
            <template #icon><IconDelete /></template>
            {{ t('media.library.deleteSelected') }}
          </a-button>
        </a-popconfirm>
      </AdminBatchBar>

      <div v-if="loading" class="media-skeleton-grid">
        <div v-for="index in 8" :key="index" class="admin-skeleton media-skeleton-card" />
      </div>

      <div v-else-if="assets.length" class="media-grid">
        <article
          v-for="asset in assets"
          :key="asset.key"
          class="media-card"
          :class="{ 'media-card-selected': selectedKeys.includes(asset.key) }">
          <label class="media-card-check">
            <input v-model="selectedKeys" type="checkbox" :value="asset.key" :aria-label="t('media.library.selectAsset', { name: asset.name })" />
          </label>
          <AdminImagePreview :src="asset.url" :alt="asset.name" :width="220" :height="148" />
          <div class="media-card-copy">
            <strong :title="asset.name">{{ asset.name }}</strong>
            <small>{{ formatFileSize(asset.size) }} · {{ formatDate(asset.lastModified) }}</small>
          </div>
          <div class="media-card-actions">
            <a-button size="small" @click="copyUrl(asset.url)">{{ t('image.copyUrl') }}</a-button>
            <a-popconfirm
              v-if="asset.deletable"
              :content="t('media.library.deleteConfirm')"
              @ok="remove(asset.key)">
              <a-button size="small" status="danger">{{ t('common.delete') }}</a-button>
            </a-popconfirm>
            <a-tooltip v-else :content="t('media.library.undeletable')">
              <a-button size="small" disabled>{{ t('common.delete') }}</a-button>
            </a-tooltip>
          </div>
        </article>
      </div>

      <AdminEmptyState
        v-else
        :icon="IconImage"
        :title="t('media.library.empty')"
        :description="prefix.trim() ? t('media.library.emptyFiltered') : t('media.library.emptyHint')">
        <a-button v-if="prefix.trim()" size="small" @click="clearPrefix">{{ t('media.library.clearFilter') }}</a-button>
        <a-button v-else type="primary" size="small" @click="fileInput?.click()">{{ t('media.library.upload') }}</a-button>
      </AdminEmptyState>

      <a-pagination
        v-if="total > pageSize"
        class="media-pagination"
        :total="total"
        :current="current"
        :page-size="pageSize"
        show-total
        show-jumper
        @change="changePage" />
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconImage, IconRefresh, IconUpload } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteAdminMedia, listAdminPage, uploadAdminMedia } from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { formatDate, formatFileSize } from '@/utils/format'
import type { AdminMediaAsset } from '@stellar-beacon/api-contract'

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024

const VIEW_KEY = 'media'

const prefix = ref('')
const selectedKeys = ref<string[]>([])
const uploading = ref(false)
const uploadProgress = ref({ done: 0, total: 0 })
const fileInput = ref<HTMLInputElement | null>(null)

const {
  items: assets,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage
} = useAsyncList<AdminMediaAsset>(
  ({ current: page, pageSize: size, signal }) =>
    listAdminPage<AdminMediaAsset>('admin/media', {
      current: page,
      size,
      prefix: prefix.value.trim()
    }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY, 24), fallbackMessage: t('mediaPicker.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'prefix', ref: prefix, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const uploadLabel = computed(() => {
  const { done, total: count } = uploadProgress.value
  return count > 0
    ? t('media.library.uploadingProgress', { done, total: count })
    : t('media.library.uploading')
})

// 分页或筛选后丢弃已不在列表中的选中项，避免删除到看不见的图片。
watch(assets, (list) => {
  const available = new Set(list.map((asset) => asset.key))
  selectedKeys.value = selectedKeys.value.filter((key) => available.has(key))
})

function clearPrefix(): void {
  prefix.value = ''
  void reload()
}

function changePage(page: number): void {
  gotoPage(page)
}

function toggleSelectAll(): void {
  selectedKeys.value = selectedKeys.value.length === assets.value.length
    ? []
    : assets.value.map((asset) => asset.key)
}

async function uploadFiles(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return

  const rejected = files.filter((file) => !file.type.startsWith('image/') || file.size > MAX_UPLOAD_BYTES)
  const accepted = files.filter((file) => !rejected.includes(file))
  if (rejected.length) {
    Message.warning(t('media.library.skipped', { count: rejected.length }))
  }
  if (!accepted.length) return

  uploading.value = true
  uploadProgress.value = { done: 0, total: accepted.length }
  const failures: string[] = []
  try {
    for (const file of accepted) {
      try {
        await uploadAdminMedia(file)
      } catch (error) {
        failures.push(`${file.name}：${apiErrorMessage(error, t('upload.failed'))}`)
      }
      uploadProgress.value = { done: uploadProgress.value.done + 1, total: accepted.length }
    }
    const succeeded = accepted.length - failures.length
    if (succeeded > 0) Message.success(t('media.library.uploaded', { count: succeeded }))
    if (failures.length) Message.error(t('media.library.uploadFailedCount', { count: failures.length, detail: failures[0] }))
    if (succeeded > 0) await reload()
  } finally {
    uploading.value = false
    uploadProgress.value = { done: 0, total: 0 }
  }
}

async function remove(key: string): Promise<void> {
  await removeKeys([key])
}

async function removeSelected(): Promise<void> {
  await removeKeys([...selectedKeys.value])
}

async function removeKeys(keys: string[]): Promise<void> {
  if (!keys.length) return
  try {
    await deleteAdminMedia(keys)
    Message.success(keys.length > 1
      ? t('media.library.deleted', { count: keys.length })
      : t('media.library.deletedOne'))
    selectedKeys.value = []
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.library.deleteFailed')))
  }
}

async function copyUrl(url: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(url)
    Message.success(t('image.copied'))
  } catch {
    // Clipboard access is blocked outside secure contexts; fall back to a manual hint.
    Message.warning(t('media.library.copyManual'))
  }
}
</script>

<style scoped>
.media-skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--admin-gap);
}

.media-skeleton-card {
  height: 236px;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--admin-gap);
}

.media-card {
  min-width: 0;
  position: relative;
  padding: 10px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-card);
  background: var(--admin-surface);
  transition: border-color var(--admin-duration) var(--admin-ease),
    box-shadow var(--admin-duration) var(--admin-ease),
    transform var(--admin-duration) var(--admin-ease);
}

.media-card:hover {
  border-color: var(--admin-brand-soft-strong);
  box-shadow: 0 12px 28px -18px rgb(16 24 40 / 40%);
  transform: translateY(-2px);
}

.media-card-selected {
  border-color: var(--admin-brand);
  box-shadow: var(--admin-shadow-focus);
}

.media-card-check {
  position: absolute;
  z-index: 2;
  top: 16px;
  left: 16px;
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  border-radius: 7px;
  background: rgb(255 255 255 / 92%);
  box-shadow: var(--admin-shadow-xs);
  cursor: pointer;
}

.media-card .admin-image-preview {
  width: 100% !important;
  height: 150px !important;
}

.media-card-copy {
  min-width: 0;
  display: grid;
  gap: 3px;
  margin: 10px 2px;
}

.media-card-copy strong,
.media-card-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.media-card-copy strong {
  color: var(--admin-ink-strong);
  font-size: 13px;
  font-weight: 650;
}

.media-card-copy small {
  color: var(--admin-subtle);
  font-size: 11px;
}

.media-card-actions {
  display: flex;
  gap: 6px;
}

.media-pagination {
  margin-top: 22px;
  justify-content: flex-end;
}
</style>
