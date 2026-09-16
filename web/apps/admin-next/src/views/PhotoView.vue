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
          <span class="admin-toolbar-caption">{{ t('media.photos.pageCount', { count: photos.length }) }}</span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button :loading="loading" size="small" @click="reloadAll">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('media.photos.loadFailed')" @retry="reloadAll" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('media.photos.pageCount', { count: photos.length })" @clear="clearSelection">
        <a-button size="small" @click="openMove">{{ t('media.photos.move') }}</a-button>
        <a-button size="small" status="danger" @click="batchRemove">{{ t('media.photos.trash') }}</a-button>
      </AdminBatchBar>

      <div v-if="loading" class="photo-wall-skeleton" aria-hidden="true">
        <div v-for="index in 8" :key="index" class="admin-skeleton photo-wall-skeleton-tile" />
      </div>
      <div v-else-if="photos.length" class="photo-masonry">
        <article
          v-for="photo in photos"
          :key="photo.id"
          role="row"
          class="photo-tile"
          :class="{ 'photo-tile-selected': selectedIds.includes(Number(photo.id)) }">
          <label class="photo-tile-select">
            <input
              v-model="selectedKeys"
              type="checkbox"
              :value="photo.id"
              :aria-label="t('media.photos.select', { name: String(photo.photoName || t('media.shared.unnamedPhoto')) })" />
          </label>
          <AdminImagePreview
            v-if="isHttpUrl(photo.photoSrc)"
            :src="String(photo.photoSrc)"
            :alt="String(photo.photoName || t('image.alt'))"
            width="100%"
            height="auto"
            fit="natural" />
          <div v-else class="photo-tile-fallback"><IconImage /></div>
          <div class="photo-tile-copy">
            <strong :title="String(photo.photoName || '')">{{ photo.photoName || t('media.shared.unnamedPhoto') }}</strong>
            <span :title="String(photo.photoDesc || '')">{{ photo.photoDesc || t('media.shared.noDescription') }}</span>
          </div>
          <div class="photo-tile-actions">
            <a-button type="text" size="small" @click="openEditor(photo)">{{ t('common.edit') }}</a-button>
            <a-popconfirm :content="t('media.photos.removeConfirm')" @ok="removePhoto(photo.id)">
              <a-button type="text" status="danger" size="small">{{ t('common.remove') }}</a-button>
            </a-popconfirm>
          </div>
        </article>
      </div>
      <AdminEmptyState
        v-else
        :icon="IconImage"
        :title="keywords.trim() ? t('media.photos.emptyFiltered') : t('media.photos.empty')"
        :description="keywords.trim() ? t('media.shared.searchAgain') : t('media.photos.emptyHint')">
        <a-button v-if="keywords.trim()" size="small" @click="keywords = ''; reloadPhotos()">{{ t('media.shared.clearSearch') }}</a-button>
        <a-button v-else type="primary" size="small" @click="photoInput?.click()">{{ t('media.photos.upload') }}</a-button>
      </AdminEmptyState>
      <a-pagination
        v-if="total > pageSize"
        class="photo-pagination"
        :total="total"
        :current="current"
        :page-size="pageSize"
        show-total
        show-jumper
        @change="changePage"
        @page-size-change="changePageSize" />
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
import type { AdminAlbum, AdminPhoto } from '@stellar-beacon/api-contract'

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024
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
  reload: reloadPhotos,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminPhoto>(
  ({ current: page, pageSize: size, signal }) => listAdminPhotos({
    current: page,
    size,
    albumId: albumId.value,
    isDelete: 0,
    keywords: keywords.value.trim()
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY, 18), fallbackMessage: t('media.photos.listLoadFailed'), immediate: false }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => { clearSelection(); void loadPhotos() }, onSearch: () => { clearSelection(); void reloadPhotos() } })

onMounted(() => void reloadAll())

const errorMessage = computed(() => listError.value || albumError.value)
const selectedIds = computed(() => selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))
const editorPreview = computed(() => (isHttpUrl(editor.photoSrc) ? editor.photoSrc : ''))

watch(photos, (list) => {
  const available = new Set(list.map((photo) => Number(photo.id)))
  selectedKeys.value = selectedKeys.value.filter((key) => available.has(Number(key)))
})

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
  const rejected = files.filter((file) => !file.type.startsWith('image/') || file.size > MAX_UPLOAD_BYTES)
  const accepted = files.filter((file) => !rejected.includes(file))
  if (rejected.length) Message.warning(t('media.photos.skipped', { count: rejected.length }))
  if (!accepted.length) return
  uploading.value = true
  const failures: string[] = []
  try {
    const urls: string[] = []
    for (const file of accepted) {
      try {
        urls.push(await uploadAdminPhoto(file))
      } catch (error) {
        failures.push(`${file.name}：${apiErrorMessage(error, t('media.photos.uploadFailed'))}`)
      }
    }
    if (urls.length) {
      await saveAdminPhotos(albumId.value, urls)
      Message.success(t('media.photos.uploaded', { count: urls.length }))
      await Promise.all([loadPhotos(), loadAlbum()])
    }
    if (failures.length) {
      Message.error(t('media.photos.uploadFailedCount', { count: failures.length, detail: failures[0] }))
    }
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

.photo-wall-skeleton,
.photo-masonry {
  column-count: 4;
  column-gap: var(--admin-gap);
}

.photo-wall-skeleton-tile {
  min-height: 220px;
  margin-bottom: var(--admin-gap);
  break-inside: avoid;
  border-radius: var(--admin-radius-card);
}

.photo-tile {
  position: relative;
  overflow: hidden;
}

.photo-tile :deep(.admin-image-preview-natural) {
  max-width: 100%;
  border-radius: var(--admin-radius-control);
}

.photo-tile-select {
  position: absolute;
  z-index: 2;
  top: 16px;
  left: 16px;
  display: grid;
  width: 26px;
  height: 26px;
  place-items: center;
  border: 1px solid rgb(255 255 255 / 72%);
  border-radius: 8px;
  background: rgb(15 23 42 / 68%);
  box-shadow: 0 4px 12px rgb(15 23 42 / 16%);
}

.photo-tile-select input {
  width: 15px;
  height: 15px;
  accent-color: var(--admin-brand);
}

.photo-tile-selected {
  border-color: var(--admin-brand);
  box-shadow: 0 0 0 2px var(--admin-brand-soft);
}

.photo-tile-fallback {
  display: grid;
  min-height: 150px;
  place-items: center;
  color: var(--admin-muted);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}

@media (max-width: 1280px) {
  .photo-wall-skeleton,
  .photo-masonry {
    column-count: 3;
  }
}

@media (max-width: 860px) {
  .photo-wall-skeleton,
  .photo-masonry {
    column-count: 2;
  }
}

@media (max-width: 560px) {
  .photo-wall-skeleton,
  .photo-masonry {
    column-count: 1;
  }
}
</style>
