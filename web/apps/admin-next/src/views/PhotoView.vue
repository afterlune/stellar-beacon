<template>
  <section class="admin-page">
    <AdminPageHeader :title="pageTitle" :description="albumDescription">
      <template #actions>
        <input ref="photoInput" type="file" accept="image/*" multiple hidden @change="selectPhotos" />
        <a-button type="primary" :loading="uploading" :disabled="!albumId" @click="photoInput?.click()">
          <template #icon><IconUpload /></template>
          {{ t('media.photos.upload') }}
        </a-button>
        <a-button @click="router.push('/albums')">
          <template #icon><IconLeft /></template>
          {{ t('media.shared.backToAlbums') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <div v-if="album.albumName" class="album-summary">
      <div class="album-summary-item">
        <span>{{ t('media.photos.album') }}</span>
        <strong>{{ album.albumName }}</strong>
      </div>
      <div class="album-summary-item">
        <span>{{ t('media.photos.count') }}</span>
        <strong>{{ t('media.shared.photoCount', { count: formatNumber(total || photos.length) }) }}</strong>
      </div>
      <div class="album-summary-item">
        <span>{{ t('media.photos.visibility') }}</span>
        <AdminStatusTag :kind="Number(album.status) === 1 ? 'public' : 'private'" />
      </div>
    </div>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="keywords"
            class="admin-filter-input"
            :placeholder="t('media.photos.searchPlaceholder')"
            allow-clear />
          <span class="admin-toolbar-caption">{{ t('media.photos.pageCount', { count: filteredPhotos.length }) }}</span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button :loading="loading" size="small" @click="reloadAll">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('media.photos.loadFailed')" @retry="reloadAll" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('media.photos.pageCount', { count: filteredPhotos.length })" @clear="clearSelection">
        <a-button size="small" @click="openMove">{{ t('media.photos.move') }}</a-button>
        <a-button size="small" status="danger" @click="batchRemove">{{ t('media.photos.trash') }}</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="filteredPhotos"
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
              <a-button type="text" size="small" @click="openEditor(record)">{{ t('common.edit') }}</a-button>
              <a-popconfirm
                :content="t('media.photos.removeConfirm')"
                @ok="removePhoto(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.remove') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconImage"
              :title="keywords.trim() ? t('media.photos.emptyFiltered') : t('media.photos.empty')"
              :description="keywords.trim() ? t('media.shared.searchAgain') : t('media.photos.emptyHint')">
              <a-button v-if="keywords.trim()" size="small" @click="keywords = ''">{{ t('media.shared.clearSearch') }}</a-button>
              <a-button v-else type="primary" size="small" @click="photoInput?.click()">{{ t('media.photos.upload') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="t('media.photos.edit')"
      :ok-loading="saving"
      :mask-closable="false"
      width="520px"
      @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="photoName" :label="t('media.shared.photoName')" required>
          <a-input v-model="editor.photoName" maxlength="20" show-word-limit :placeholder="t('media.photos.namePlaceholder')" />
        </a-form-item>
        <a-form-item field="photoDesc" :label="t('media.photos.desc')">
          <a-input v-model="editor.photoDesc" maxlength="50" show-word-limit :placeholder="t('media.photos.descPlaceholder')" />
        </a-form-item>
      </a-form>
      <div v-if="editorPreview" class="photo-editor-preview">
        <AdminImagePreview :src="editorPreview" :alt="t('media.photos.preview')" :width="120" :height="84" />
        <span class="admin-field-hint">{{ t('media.photos.renameHint') }}</span>
      </div>
    </a-modal>

    <a-modal
      v-model:visible="moveVisible"
      :title="t('media.photos.move')"
      :ok-loading="moving"
      :ok-button-props="{ disabled: !targetAlbumId }"
      :mask-closable="false"
      width="480px"
      @ok="confirmMove">
      <a-form :model="{ targetAlbumId }" layout="vertical">
        <a-form-item :label="t('media.photos.targetAlbum')" required>
          <a-select v-model="targetAlbumId" :placeholder="t('media.photos.targetAlbumPlaceholder')" :loading="albumOptionsLoading">
            <a-option
              v-for="option in albumOptions"
              :key="option.id"
              :value="option.id"
              :disabled="Number(option.id) === albumId">
              {{ option.albumName }}
            </a-option>
          </a-select>
          <template #help>{{ t('media.photos.moveHint') }}</template>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconImage, IconLeft, IconRefresh, IconUpload } from '@arco-design/web-vue/es/icon'
import { useRoute, useRouter } from 'vue-router'

import {
  apiErrorMessage,
  getAdminAlbum,
  listAdminAlbumOptions,
  listAdminPhotos,
  moveAdminPhotosToAlbum,
  saveAdminPhotos,
  updateAdminPhoto,
  updateAdminPhotoDelete,
  uploadAdminPhoto,
  type AdminAlbumOption
} from '@/api/http'
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
import { formatNumber, isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminAlbum, AdminPhoto } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'photos'

const route = useRoute()
const router = useRouter()

const albumId = computed(() => Number(route.params.albumId || route.params.id || route.params.articleId || 0))
const pageTitle = computed(() => (album.value.albumName
  ? t('media.photos.titleOf', { name: album.value.albumName })
  : t('media.photos.title')))
const albumDescription = computed(() => (album.value.albumName
  ? t('media.photos.description')
  : t('media.photos.loading')))

// 列定义随语言切换重算，所以用 computed 而不是模块级常量。
const columns = computed(() => [
  { title: t('common.preview'), dataIndex: 'photoSrc', width: 108, slotName: 'source' },
  { title: t('media.shared.photoName'), dataIndex: 'photoName', slotName: 'photoName', minWidth: 180 },
  { title: t('common.description'), dataIndex: 'photoDesc', slotName: 'photoDesc', ellipsis: true, tooltip: true },
  { title: t('common.actions'), dataIndex: 'actions', width: 156, slotName: 'actions' }
])

const album = ref<AdminAlbum>({ id: 0, albumName: '' })
const keywords = ref('')
const uploading = ref(false)
const saving = ref(false)
const editorVisible = ref(false)
const albumError = ref('')
const photoInput = ref<HTMLInputElement | null>(null)
const editor = reactive({ id: 0, photoName: '', photoDesc: '', photoSrc: '' })

const selectedKeys = ref<number[]>([])
const moveVisible = ref(false)
const moving = ref(false)
const targetAlbumId = ref<number | undefined>(undefined)
const albumOptions = ref<AdminAlbumOption[]>([])
const albumOptionsLoading = ref(false)

const {
  items: photos,
  total,
  current,
  pageSize,
  loading,
  error: listError,
  load: loadPhotos,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminPhoto>(
  ({ current: page, pageSize: size, signal }) => listAdminPhotos({
    current: page,
    size,
    albumId: albumId.value,
    isDelete: 0
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY, 18), fallbackMessage: t('media.photos.listLoadFailed'), immediate: false }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => void loadPhotos() })

onMounted(() => void reloadAll())

const errorMessage = computed(() => listError.value || albumError.value)
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value, [18, 36, 72]))
const selectedIds = computed(() => selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))
const filteredPhotos = computed(() => {
  const query = keywords.value.trim().toLowerCase()
  if (!query) return photos.value
  return photos.value.filter((photo) =>
    `${photo.photoName || ''} ${photo.photoDesc || ''}`.toLowerCase().includes(query)
  )
})
const editorPreview = computed(() => (isHttpUrl(editor.photoSrc) ? editor.photoSrc : ''))

async function reloadAll(): Promise<void> {
  albumError.value = ''
  if (!albumId.value) {
    albumError.value = t('media.photos.albumMissing')
    return
  }
  await Promise.all([loadAlbum(), loadPhotos()])
}

async function loadAlbum(): Promise<void> {
  try {
    album.value = await getAdminAlbum(albumId.value)
  } catch (error) {
    albumError.value = apiErrorMessage(error, t('media.photos.albumLoadFailed'))
  }
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

function openEditor(photo: AdminPhoto): void {
  editor.id = Number(photo.id || 0)
  editor.photoName = String(photo.photoName || '')
  editor.photoDesc = String(photo.photoDesc || '')
  editor.photoSrc = String(photo.photoSrc || '')
  editorVisible.value = true
}

async function saveEditor(): Promise<void> {
  if (!editor.id || !editor.photoName.trim()) {
    Message.error(t('media.photos.nameRequired'))
    return
  }
  saving.value = true
  try {
    await updateAdminPhoto({
      id: editor.id,
      photoName: editor.photoName.trim(),
      photoDesc: editor.photoDesc.trim()
    })
    editorVisible.value = false
    Message.success(t('media.photos.saved'))
    await loadPhotos()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.photos.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function removePhoto(id: unknown): Promise<void> {
  const photoId = Number(id)
  if (!Number.isInteger(photoId) || photoId <= 0) return
  try {
    await updateAdminPhotoDelete([photoId])
    Message.success(t('media.photos.trashed'))
    await Promise.all([loadPhotos(), loadAlbum()])
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.photos.removeFailed')))
  }
}

async function batchRemove(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0) return
  try {
    await updateAdminPhotoDelete(ids)
    Message.success(t('media.photos.trashedCount', { count: ids.length }))
    clearSelection()
    await Promise.all([loadPhotos(), loadAlbum()])
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.photos.batchRemoveFailed')))
  }
}

async function openMove(): Promise<void> {
  if (selectedIds.value.length === 0) return
  targetAlbumId.value = undefined
  moveVisible.value = true
  albumOptionsLoading.value = true
  try {
    albumOptions.value = await listAdminAlbumOptions()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.photos.albumOptionsFailed')))
  } finally {
    albumOptionsLoading.value = false
  }
}

async function confirmMove(): Promise<void> {
  const ids = selectedIds.value
  if (ids.length === 0 || !targetAlbumId.value) return
  moving.value = true
  try {
    await moveAdminPhotosToAlbum(ids, Number(targetAlbumId.value))
    Message.success(t('media.photos.moved', { count: ids.length }))
    moveVisible.value = false
    clearSelection()
    // 照片离开了当前相册，页数与总数都要重新取。
    await Promise.all([loadPhotos(), loadAlbum()])
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.photos.moveFailed')))
  } finally {
    moving.value = false
  }
}

async function selectPhotos(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length || !albumId.value) return
  uploading.value = true
  try {
    const urls: string[] = []
    for (const file of files) urls.push(await uploadAdminPhoto(file))
    await saveAdminPhotos(albumId.value, urls)
    Message.success(t('media.photos.uploaded', { count: urls.length }))
    await Promise.all([loadPhotos(), loadAlbum()])
  } catch (error) {
    Message.error(apiErrorMessage(error, t('media.photos.uploadFailed')))
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.album-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 14px;
  margin-bottom: var(--admin-gap-lg);
  padding: 16px 20px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-card);
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-card);
}

.album-summary-item {
  display: grid;
  gap: 4px;
}

.album-summary-item > span {
  color: var(--admin-muted);
  font-size: 12px;
  font-weight: 600;
}

.album-summary-item > strong {
  color: var(--admin-ink-strong);
  font-size: 16px;
  font-weight: 700;
}

.photo-editor-preview {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 4px;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}
</style>
