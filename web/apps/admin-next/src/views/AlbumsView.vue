<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('media.albums.title')" :description="t('media.albums.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('media.albums.searchPlaceholder')"
          allow-clear
          @search="reload" />
        <a-button @click="router.push('/photos/delete')">
          <template #icon><IconDelete /></template>
          {{ t('media.albums.trash') }}
        </a-button>
        <a-button type="primary" @click="openEditor()">
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
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">{{ t('media.albums.resetFilters') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">{{ t('media.albums.total', { albums: filteredAlbums.length, photos: totalPhotos }) }}</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('media.albums.loadFailed')" @retry="load" />

      <div class="admin-table-shell">
        <a-table
          :data="filteredAlbums"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #cover="{ record }">
            <AdminImagePreview
              v-if="isHttpUrl(record.albumCover)"
              :src="String(record.albumCover)"
              :alt="t('media.albums.coverAlt', { name: String(record.albumName || t('media.albums.unnamed')) })"
              :width="92"
              :height="62" />
            <span v-else class="admin-cover-cell" aria-hidden="true"><IconImage /></span>
          </template>
          <template #albumName="{ record }">
            <button class="album-name-link" type="button" @click="openPhotos(record)">
              {{ record.albumName || t('media.albums.unnamed') }}
            </button>
          </template>
          <template #albumDesc="{ record }">
            <span class="admin-muted-cell" :title="String(record.albumDesc || '')">{{ record.albumDesc || t('media.shared.noDescription') }}</span>
          </template>
          <template #photoCount="{ record }">
            <a-tag :color="Number(record.photoCount) > 0 ? 'arcoblue' : 'gray'">{{ t('media.shared.photoCount', { count: formatNumber(record.photoCount ?? 0) }) }}</a-tag>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.status) === 1 ? 'public' : 'private'" />
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openPhotos(record)">{{ t('media.albums.photos') }}</a-button>
              <a-button type="text" size="small" @click="openEditor(record)">{{ t('common.edit') }}</a-button>
              <a-popconfirm
                :content="t('media.albums.deleteConfirm', { name: String(record.albumName) })"
                @ok="deleteAlbum(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconImage"
              :title="hasFilters ? t('media.albums.emptyFiltered') : t('media.albums.empty')"
              :description="hasFilters ? t('media.albums.emptyFilteredHint') : t('media.albums.emptyHint')">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">{{ t('media.albums.resetFilters') }}</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">{{ t('media.albums.create') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? t('media.albums.edit') : t('media.albums.create')"
      :ok-loading="saving"
      :mask-closable="false"
      width="620px"
      @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="albumName" :label="t('media.albums.name')" required>
          <a-input v-model="editor.albumName" maxlength="20" show-word-limit :placeholder="t('media.albums.namePlaceholder')" />
        </a-form-item>
        <a-form-item field="albumDesc" :label="t('media.albums.desc')" required>
          <a-textarea v-model="editor.albumDesc" maxlength="50" show-word-limit :auto-size="{ minRows: 2, maxRows: 4 }" :placeholder="t('media.albums.descPlaceholder')" />
        </a-form-item>
        <a-form-item field="albumCover" :label="t('media.albums.coverUrl')" required>
          <a-space direction="vertical" fill>
            <a-input v-model="editor.albumCover" :placeholder="t('media.albums.coverPlaceholder')" />
            <a-space wrap>
              <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
              <a-button :loading="uploading" @click="coverInput?.click()">{{ t('media.albums.uploadCover') }}</a-button>
              <a-button :disabled="!editor.albumCover.trim()" @click="editor.albumCover = ''">{{ t('common.clear') }}</a-button>
            </a-space>
          </a-space>
        </a-form-item>
        <div v-if="isHttpUrl(editor.albumCover)" class="album-cover-preview">
          <AdminImagePreview :src="editor.albumCover" :alt="t('media.albums.coverPreview')" :width="150" :height="100" />
          <span class="admin-field-hint">{{ t('media.albums.coverHint') }}</span>
        </div>
        <a-form-item field="status" :label="t('media.albums.publishStatus')">
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">{{ t('status.published') }}</a-radio>
            <a-radio :value="2">{{ t('status.private') }}</a-radio>
          </a-radio-group>
          <template #help>{{ t('media.albums.privateHint') }}</template>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconImage, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import {
  apiErrorMessage,
  deleteAdminAlbum,
  listAdminPage,
  saveAdminAlbum,
  uploadAdminAlbumCover
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { formatNumber, isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminAlbum } from '@stellar-beacon/api-contract'

// 列定义随语言切换重算，所以用 computed 而不是模块级常量。
const columns = computed(() => [
  { title: t('media.albums.cover'), dataIndex: 'albumCover', width: 116, slotName: 'cover' },
  { title: t('media.albums.name'), dataIndex: 'albumName', slotName: 'albumName', minWidth: 170 },
  { title: t('common.description'), dataIndex: 'albumDesc', slotName: 'albumDesc', ellipsis: true, tooltip: true },
  { title: t('media.albums.photoCount'), dataIndex: 'photoCount', width: 106, slotName: 'photoCount' },
  { title: t('common.status'), dataIndex: 'status', width: 96, slotName: 'status' },
  { title: t('common.actions'), dataIndex: 'actions', width: 176, slotName: 'actions' }
])

const VIEW_KEY = 'albums'

const router = useRouter()
const keywords = ref('')
const statusFilter = ref<'all' | '1' | '2'>('all')
const saving = ref(false)
const uploading = ref(false)
const editorVisible = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const editor = reactive({ id: 0, albumName: '', albumDesc: '', albumCover: '', status: 1 })

const {
  items: albums,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminAlbum>(
  ({ current: page, pageSize: size, signal }) => listAdminPage<AdminAlbum>('admin/albums', {
    current: page,
    size,
    keywords: keywords.value.trim()
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY, 8), fallbackMessage: t('media.albums.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'status', ref: statusFilter },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() => Boolean(keywords.value.trim()) || statusFilter.value !== 'all')
const filteredAlbums = computed(() => statusFilter.value === 'all'
  ? albums.value
  : albums.value.filter((album) => Number(album.status) === Number(statusFilter.value)))
const totalPhotos = computed(() => albums.value.reduce((sum, album) => sum + Number(album.photoCount || 0), 0))

function resetFilters(): void {
  keywords.value = ''
  statusFilter.value = 'all'
  void reload()
}

function changePage(page: number): void {
  gotoPage(page)
}

function changePageSize(size: number): void {
  applyPageSize(size)
}

function openEditor(album?: AdminAlbum): void {
  editor.id = Number(album?.id || 0)
  editor.albumName = String(album?.albumName || '')
  editor.albumDesc = String(album?.albumDesc || '')
  editor.albumCover = String(album?.albumCover || '')
  editor.status = Number(album?.status || 1)
  editorVisible.value = true
}

function openPhotos(album: AdminAlbum): void {
  const id = Number(album.id)
  if (Number.isInteger(id) && id > 0) void router.push(`/albums/${id}`)
}

async function saveEditor(): Promise<void> {
  if (!editor.albumName.trim() || !editor.albumDesc.trim() || !editor.albumCover.trim()) {
    Message.error(t('media.albums.requiredFields'))
    return
  }
  saving.value = true
  try {
    await saveAdminAlbum({
      id: editor.id || undefined,
      albumName: editor.albumName.trim(),
      albumDesc: editor.albumDesc.trim(),
      albumCover: editor.albumCover.trim(),
      status: editor.status
    })
    editorVisible.value = false
    Message.success(editor.id ? t('media.albums.updated') : t('media.albums.created'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.albums.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function deleteAlbum(id: unknown): Promise<void> {
  const albumId = Number(id)
  if (!Number.isInteger(albumId) || albumId <= 0) return
  try {
    await deleteAdminAlbum(albumId)
    if (albums.value.length === 1 && current.value > 1) current.value -= 1
    Message.success(t('media.albums.deleted'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.albums.deleteFailed')))
  }
}

async function selectCover(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  try {
    editor.albumCover = await uploadAdminAlbumCover(file)
    Message.success(t('media.albums.coverUploaded'))
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.albums.coverUploadFailed')))
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.album-name-link {
  padding: 0;
  border: 0;
  color: var(--admin-ink-strong);
  background: none;
  font: inherit;
  font-weight: 650;
  text-align: left;
  cursor: pointer;
  transition: color var(--admin-duration-fast) var(--admin-ease);
}

.album-name-link:hover {
  color: var(--admin-brand);
}

.album-cover-preview {
  display: flex;
  align-items: center;
  gap: 14px;
  margin: -6px 0 16px;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}
</style>
