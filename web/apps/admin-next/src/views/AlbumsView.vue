<template>
  <section class="admin-page">
    <AdminPageHeader title="相册管理" description="用一个舒服的空间整理博客里的影像。">
      <template #actions>
        <a-space>
          <a-input-search v-model="keywords" class="admin-filter-input" placeholder="搜索相册名" allow-clear @search="reload" />
          <a-button @click="router.push('/photos/delete')">回收站</a-button>
          <a-button type="primary" @click="openEditor()">
            <template #icon><IconPlus /></template>
            新增
          </a-button>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table
          :data="albums"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #cover="{ record }">
            <img v-if="isHttpUrl(record.albumCover)" class="album-cover" :src="record.albumCover" alt="相册封面" />
            <span v-else>—</span>
          </template>
          <template #status="{ record }"><a-tag class="admin-status-tag" :color="Number(record.status) === 1 ? 'green' : 'orange'">{{ Number(record.status) === 1 ? '公开' : '私密' }}</a-tag></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm content="确定删除该相册吗？" @ok="deleteAlbum(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无相册" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="editorVisible" :title="editor.id ? '编辑相册' : '新增相册'" :ok-loading="saving" width="620px" @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="albumName" label="相册名称" required>
          <a-input v-model="editor.albumName" maxlength="20" show-word-limit />
        </a-form-item>
        <a-form-item field="albumDesc" label="相册描述" required>
          <a-textarea v-model="editor.albumDesc" maxlength="50" show-word-limit />
        </a-form-item>
        <a-form-item field="albumCover" label="封面 URL" required>
          <a-space direction="vertical" fill>
            <a-input v-model="editor.albumCover" placeholder="也可以直接填写 HTTPS 图片地址" />
            <a-space>
              <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
              <a-button :loading="uploading" @click="coverInput?.click()">上传封面</a-button>
              <span class="field-hint">上传结果由后端对象存储接口返回</span>
            </a-space>
          </a-space>
        </a-form-item>
        <a-form-item field="status" label="发布状态">
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">公开</a-radio>
            <a-radio :value="2">私密</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import {
  apiErrorMessage,
  deleteAdminAlbum,
  listAdminAlbums,
  saveAdminAlbum,
  uploadAdminAlbumCover
} from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { tablePagination } from '@/utils/pagination'
import type { AdminAlbum } from '@benetnasch/api-contract'

const columns = [
  { title: '封面', dataIndex: 'albumCover', width: 90, slotName: 'cover' },
  { title: '相册名称', dataIndex: 'albumName' },
  { title: '描述', dataIndex: 'albumDesc', ellipsis: true, tooltip: true },
  { title: '照片数', dataIndex: 'photoCount', width: 100 },
  { title: '状态', dataIndex: 'status', width: 100, slotName: 'status' },
  { title: '操作', dataIndex: 'actions', width: 150, slotName: 'actions' }
]

const router = useRouter()
const albums = ref<AdminAlbum[]>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(8)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const uploading = ref(false)
const errorMessage = ref('')
const editorVisible = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const editor = reactive({ id: 0, albumName: '', albumDesc: '', albumCover: '', status: 1 })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminAlbums({ current: current.value, size: pageSize.value, keywords: keywords.value.trim() })
    albums.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '相册加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void load()
}

function openEditor(album?: AdminAlbum): void {
  editor.id = Number(album?.id || 0)
  editor.albumName = String(album?.albumName || '')
  editor.albumDesc = String(album?.albumDesc || '')
  editor.albumCover = String(album?.albumCover || '')
  editor.status = Number(album?.status || 1)
  editorVisible.value = true
}

async function saveEditor(): Promise<void> {
  if (!editor.albumName.trim() || !editor.albumDesc.trim() || !editor.albumCover.trim()) {
    Message.error('相册名称、描述和封面不能为空')
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
    Message.success('相册已保存')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '相册保存失败'))
  } finally {
    saving.value = false
  }
}

async function deleteAlbum(id: unknown): Promise<void> {
  const albumId = Number(id)
  if (!albumId) return
  try {
    await deleteAdminAlbum(albumId)
    Message.success('相册已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '相册删除失败'))
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
    Message.success('封面上传成功')
  } catch (error) {
    Message.error(apiErrorMessage(error, '封面上传失败'))
  } finally {
    uploading.value = false
  }
}

function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && /^https?:\/\//i.test(value)
}
</script>

<style scoped>
.album-cover {
  width: 56px;
  height: 40px;
  border-radius: 6px;
  object-fit: cover;
}

.field-hint {
  color: var(--color-text-3);
  font-size: 12px;
}
</style>
