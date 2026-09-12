<template>
  <section class="admin-page">
    <AdminPageHeader :title="pageTitle" :description="albumDescription">
      <template #actions>
        <input ref="photoInput" type="file" accept="image/*" multiple hidden @change="selectPhotos" />
        <a-button type="primary" :loading="uploading" :disabled="!albumId" @click="photoInput?.click()">
          <template #icon><IconUpload /></template>
          上传照片
        </a-button>
        <a-button @click="router.push('/albums')">
          <template #icon><IconLeft /></template>
          返回相册
        </a-button>
      </template>
    </AdminPageHeader>

    <div v-if="album.albumName" class="album-summary">
      <div class="album-summary-item">
        <span>所属相册</span>
        <strong>{{ album.albumName }}</strong>
      </div>
      <div class="album-summary-item">
        <span>照片数量</span>
        <strong>{{ formatNumber(total || photos.length) }} 张</strong>
      </div>
      <div class="album-summary-item">
        <span>可见性</span>
        <AdminStatusTag :kind="Number(album.status) === 1 ? 'public' : 'private'" />
      </div>
    </div>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="keywords"
            class="admin-filter-input"
            placeholder="在本页搜索照片名称"
            allow-clear />
          <span class="admin-toolbar-caption">本页 {{ filteredPhotos.length }} 张</span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button :loading="loading" size="small" @click="reload">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

      <div class="admin-table-shell">
        <a-table
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
              :alt="String(record.photoName || '照片')"
              :width="84"
              :height="58" />
            <span v-else class="admin-cover-cell" aria-hidden="true"><IconImage /></span>
          </template>
          <template #photoName="{ record }">
            <span class="admin-title-cell">{{ record.photoName || '未命名照片' }}</span>
          </template>
          <template #photoDesc="{ record }">
            <span class="admin-muted-cell" :title="String(record.photoDesc || '')">{{ record.photoDesc || '暂无描述' }}</span>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm
                content="移除后照片会进入回收站，可以在回收站恢复。确认移除吗？"
                @ok="removePhoto(record.id)">
                <a-button type="text" status="danger" size="small">移除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconImage"
              :title="keywords.trim() ? '本页没有匹配的照片' : '相册里还没有照片'"
              :description="keywords.trim() ? '换个关键词再试一次。' : '上传照片后，它们会按上传顺序显示在这里。'">
              <a-button v-if="keywords.trim()" size="small" @click="keywords = ''">清空搜索</a-button>
              <a-button v-else type="primary" size="small" @click="photoInput?.click()">上传照片</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      title="编辑照片"
      :ok-loading="saving"
      :mask-closable="false"
      width="520px"
      @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="photoName" label="照片名称" required>
          <a-input v-model="editor.photoName" maxlength="20" show-word-limit placeholder="用于在相册中识别这张照片" />
        </a-form-item>
        <a-form-item field="photoDesc" label="照片描述">
          <a-input v-model="editor.photoDesc" maxlength="50" show-word-limit placeholder="可选，例如拍摄地点或时间" />
        </a-form-item>
      </a-form>
      <div v-if="editorPreview" class="photo-editor-preview">
        <AdminImagePreview :src="editorPreview" alt="照片预览" :width="120" :height="84" />
        <span class="admin-field-hint">照片文件本身不会因为重命名而改变。</span>
      </div>
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
  listAdminPhotos,
  saveAdminPhotos,
  updateAdminPhoto,
  updateAdminPhotoDelete,
  uploadAdminPhoto
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { formatNumber, isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminAlbum, AdminPhoto } from '@benetnasch/api-contract'

const route = useRoute()
const router = useRouter()

const albumId = computed(() => Number(route.params.albumId || route.params.id || route.params.articleId || 0))
const pageTitle = computed(() => (album.value.albumName ? `${album.value.albumName} · 照片` : '照片管理'))
const albumDescription = computed(() => (album.value.albumName
  ? '上传、重命名并整理这个相册里的照片，移除的照片会进入回收站。'
  : '相册信息加载中，请稍候。'))

const columns = [
  { title: '预览', dataIndex: 'photoSrc', width: 108, slotName: 'source' },
  { title: '照片名称', dataIndex: 'photoName', slotName: 'photoName', minWidth: 180 },
  { title: '描述', dataIndex: 'photoDesc', slotName: 'photoDesc', ellipsis: true, tooltip: true },
  { title: '操作', dataIndex: 'actions', width: 156, slotName: 'actions' }
]

const album = ref<AdminAlbum>({ id: 0, albumName: '' })
const photos = ref<AdminPhoto[]>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(18)
const total = ref(0)
const loading = ref(false)
const uploading = ref(false)
const saving = ref(false)
const editorVisible = ref(false)
const errorMessage = ref('')
const photoInput = ref<HTMLInputElement | null>(null)
const editor = reactive({ id: 0, photoName: '', photoDesc: '', photoSrc: '' })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value, [18, 36, 72]))
const filteredPhotos = computed(() => {
  const query = keywords.value.trim().toLowerCase()
  if (!query) return photos.value
  return photos.value.filter((photo) =>
    `${photo.photoName || ''} ${photo.photoDesc || ''}`.toLowerCase().includes(query)
  )
})
const editorPreview = computed(() => (isHttpUrl(editor.photoSrc) ? editor.photoSrc : ''))

onMounted(() => void loadAll())

async function loadAll(): Promise<void> {
  if (!albumId.value) {
    errorMessage.value = '相册不存在或链接已失效。'
    return
  }
  await Promise.all([loadAlbum(), loadPhotos()])
}

async function reload(): Promise<void> {
  await loadAll()
}

async function loadAlbum(): Promise<void> {
  try {
    album.value = await getAdminAlbum(albumId.value)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '相册信息加载失败')
  }
}

async function loadPhotos(): Promise<void> {
  loading.value = true
  try {
    const page = await listAdminPhotos({
      current: current.value,
      size: pageSize.value,
      albumId: albumId.value,
      isDelete: 0
    })
    photos.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '照片列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  current.value = page
  void loadPhotos()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void loadPhotos()
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
    Message.error('照片名称不能为空')
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
    Message.success('照片信息已保存')
    await loadPhotos()
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片信息保存失败'))
  } finally {
    saving.value = false
  }
}

async function removePhoto(id: unknown): Promise<void> {
  const photoId = Number(id)
  if (!Number.isInteger(photoId) || photoId <= 0) return
  try {
    await updateAdminPhotoDelete([photoId])
    Message.success('照片已移入回收站')
    await Promise.all([loadPhotos(), loadAlbum()])
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片移除失败'))
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
    Message.success(`已上传 ${urls.length} 张照片`)
    await Promise.all([loadPhotos(), loadAlbum()])
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片上传失败'))
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
