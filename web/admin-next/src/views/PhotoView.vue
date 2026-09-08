<template>
  <section class="admin-page">
    <AdminPageHeader :title="pageTitle" description="上传、预览并维护相册里的照片。">
      <template #actions>
        <a-space>
          <input ref="photoInput" type="file" accept="image/*" multiple hidden @change="selectPhotos" />
          <a-button type="primary" :loading="uploading" @click="photoInput?.click()">
            <template #icon><IconUpload /></template>
            上传照片
          </a-button>
          <a-button @click="router.push('/albums')">返回相册</a-button>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-descriptions v-if="album.albumName" :column="3" bordered style="margin-bottom: 16px">
        <a-descriptions-item label="相册">{{ album.albumName }}</a-descriptions-item>
        <a-descriptions-item label="照片数">{{ album.photoCount ?? 0 }}</a-descriptions-item>
        <a-descriptions-item label="状态">{{ Number(album.status) === 1 ? '公开' : '私密' }}</a-descriptions-item>
      </a-descriptions>
      <div class="admin-table-shell">
        <a-table
          :data="photos"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #source="{ record }">
            <img v-if="isHttpUrl(record.photoSrc)" class="photo-thumb" :src="record.photoSrc" alt="照片" />
            <span v-else>—</span>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm content="确定移除这张照片吗？" @ok="removePhoto(record.id)">
                <a-button type="text" status="danger" size="small">移除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无照片" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="editorVisible" title="编辑照片" :ok-loading="saving" @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="photoName" label="照片名称" required>
          <a-input v-model="editor.photoName" maxlength="20" />
        </a-form-item>
        <a-form-item field="photoDesc" label="照片描述">
          <a-input v-model="editor.photoDesc" maxlength="50" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconUpload } from '@arco-design/web-vue/es/icon'
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
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { tablePagination } from '@/utils/pagination'
import type { AdminAlbum, AdminPhoto } from '@shared/api-contract'

const route = useRoute()
const router = useRouter()
const albumId = computed(() => Number(route.params.albumId || route.params.id || route.params.articleId || 0))
const pageTitle = computed(() => album.value.albumName ? `${album.value.albumName} · 照片` : '照片管理')
const album = ref<AdminAlbum>({ id: 0, albumName: '' })
const photos = ref<AdminPhoto[]>([])
const columns = [
  { title: '预览', dataIndex: 'photoSrc', width: 90, slotName: 'source' },
  { title: '名称', dataIndex: 'photoName' },
  { title: '描述', dataIndex: 'photoDesc', ellipsis: true, tooltip: true },
  { title: '操作', dataIndex: 'actions', width: 140, slotName: 'actions' }
]
const current = ref(1)
const pageSize = ref(18)
const total = ref(0)
const loading = ref(false)
const uploading = ref(false)
const saving = ref(false)
const editorVisible = ref(false)
const errorMessage = ref('')
const photoInput = ref<HTMLInputElement | null>(null)
const editor = reactive({ id: 0, photoName: '', photoDesc: '' })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

onMounted(() => void loadAll())

async function loadAll(): Promise<void> {
  if (!albumId.value) {
    errorMessage.value = '相册不存在'
    return
  }
  await Promise.all([loadAlbum(), loadPhotos()])
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
    const page = await listAdminPhotos({ current: current.value, size: pageSize.value, albumId: albumId.value, isDelete: 0 })
    photos.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '照片列表加载失败')
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
  editorVisible.value = true
}

async function saveEditor(): Promise<void> {
  if (!editor.id || !editor.photoName.trim()) {
    Message.error('照片名称不能为空')
    return
  }
  saving.value = true
  try {
    await updateAdminPhoto({ id: editor.id, photoName: editor.photoName.trim(), photoDesc: editor.photoDesc.trim() })
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
  if (!photoId) return
  try {
    await updateAdminPhotoDelete([photoId])
    Message.success('照片已移除')
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

function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && /^https?:\/\//i.test(value)
}
</script>

<style scoped>
.photo-thumb {
  width: 56px;
  height: 40px;
  border-radius: 6px;
  object-fit: cover;
}
</style>
