<template>
  <section class="admin-page">
    <AdminPageHeader title="相册管理" description="用一个舒服的空间整理博客里的影像。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索相册名"
          allow-clear
          @search="reload" />
        <a-button @click="router.push('/photos/delete')">
          <template #icon><IconDelete /></template>
          回收站
        </a-button>
        <a-button type="primary" @click="openEditor()">
          <template #icon><IconPlus /></template>
          新增
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-radio-group v-model="statusFilter" type="button" size="small" @change="reload">
            <a-radio value="all">全部</a-radio>
            <a-radio value="1">公开</a-radio>
            <a-radio value="2">私密</a-radio>
          </a-radio-group>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">重置筛选</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ filteredAlbums.length }} 个相册 · {{ totalPhotos }} 张照片</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

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
              :alt="`${String(record.albumName || '相册')} 封面`"
              :width="92"
              :height="62" />
            <span v-else class="admin-cover-cell" aria-hidden="true"><IconImage /></span>
          </template>
          <template #albumName="{ record }">
            <button class="album-name-link" type="button" @click="openPhotos(record)">
              {{ record.albumName || '未命名相册' }}
            </button>
          </template>
          <template #albumDesc="{ record }">
            <span class="admin-muted-cell" :title="String(record.albumDesc || '')">{{ record.albumDesc || '暂无描述' }}</span>
          </template>
          <template #photoCount="{ record }">
            <a-tag :color="Number(record.photoCount) > 0 ? 'arcoblue' : 'gray'">{{ formatNumber(record.photoCount ?? 0) }} 张</a-tag>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.status) === 1 ? 'public' : 'private'" />
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openPhotos(record)">照片</a-button>
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm
                :content="`确定删除相册「${record.albumName}」吗？相册内的照片会一并移入回收站。`"
                @ok="deleteAlbum(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconImage"
              :title="hasFilters ? '没有匹配的相册' : '还没有相册'"
              :description="hasFilters ? '换个关键词或重置筛选条件再试一次。' : '创建相册后就可以批量上传照片了。'">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">重置筛选</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">新增相册</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? '编辑相册' : '新增相册'"
      :ok-loading="saving"
      :mask-closable="false"
      width="620px"
      @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="albumName" label="相册名称" required>
          <a-input v-model="editor.albumName" maxlength="20" show-word-limit placeholder="例如：项目截图" />
        </a-form-item>
        <a-form-item field="albumDesc" label="相册描述" required>
          <a-textarea v-model="editor.albumDesc" maxlength="50" show-word-limit :auto-size="{ minRows: 2, maxRows: 4 }" placeholder="一句话说明这个相册记录了什么" />
        </a-form-item>
        <a-form-item field="albumCover" label="封面 URL" required>
          <a-space direction="vertical" fill>
            <a-input v-model="editor.albumCover" placeholder="也可以直接填写 HTTPS 图片地址" />
            <a-space wrap>
              <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
              <a-button :loading="uploading" @click="coverInput?.click()">上传封面</a-button>
              <a-button :disabled="!editor.albumCover.trim()" @click="editor.albumCover = ''">清除</a-button>
            </a-space>
          </a-space>
        </a-form-item>
        <div v-if="isHttpUrl(editor.albumCover)" class="album-cover-preview">
          <AdminImagePreview :src="editor.albumCover" alt="封面预览" :width="150" :height="100" />
          <span class="admin-field-hint">封面会以 16:10 的比例显示在相册卡片上。</span>
        </div>
        <a-form-item field="status" label="发布状态">
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">公开</a-radio>
            <a-radio :value="2">私密</a-radio>
          </a-radio-group>
          <template #help>私密相册不会出现在博客前台的相册列表中。</template>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconImage, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import {
  apiErrorMessage,
  deleteAdminAlbum,
  listAdminAlbums,
  saveAdminAlbum,
  uploadAdminAlbumCover
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { formatNumber, isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminAlbum } from '@benetnasch/api-contract'

const columns = [
  { title: '封面', dataIndex: 'albumCover', width: 116, slotName: 'cover' },
  { title: '相册名称', dataIndex: 'albumName', slotName: 'albumName', minWidth: 170 },
  { title: '描述', dataIndex: 'albumDesc', slotName: 'albumDesc', ellipsis: true, tooltip: true },
  { title: '照片数', dataIndex: 'photoCount', width: 106, slotName: 'photoCount' },
  { title: '状态', dataIndex: 'status', width: 96, slotName: 'status' },
  { title: '操作', dataIndex: 'actions', width: 176, slotName: 'actions' }
]

const router = useRouter()
const albums = ref<AdminAlbum[]>([])
const keywords = ref('')
const statusFilter = ref<'all' | '1' | '2'>('all')
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
const hasFilters = computed(() => Boolean(keywords.value.trim()) || statusFilter.value !== 'all')
const filteredAlbums = computed(() => statusFilter.value === 'all'
  ? albums.value
  : albums.value.filter((album) => Number(album.status) === Number(statusFilter.value)))
const totalPhotos = computed(() => albums.value.reduce((sum, album) => sum + Number(album.photoCount || 0), 0))

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function resetFilters(): void {
  keywords.value = ''
  statusFilter.value = 'all'
  void reload()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminAlbums({
      current: current.value,
      size: pageSize.value,
      keywords: keywords.value.trim()
    })
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

function openPhotos(album: AdminAlbum): void {
  const id = Number(album.id)
  if (Number.isInteger(id) && id > 0) void router.push(`/albums/${id}`)
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
    Message.success(editor.id ? '相册已更新' : '相册已创建')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '相册保存失败'))
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
