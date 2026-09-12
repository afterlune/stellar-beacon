<template>
  <section class="admin-page">
    <AdminPageHeader title="照片回收站" description="恢复误删的照片，或永久清理不再需要的内容。">
      <template #actions>
        <a-button :disabled="selectedIds.length === 0" :loading="restoring" @click="restoreSelected">
          <template #icon><IconUndo /></template>
          批量恢复
        </a-button>
        <a-popconfirm
          content="永久删除选中的照片？此操作不可撤销。"
          :disabled="selectedIds.length === 0"
          @ok="deleteSelected">
          <a-button status="danger" :disabled="selectedIds.length === 0" :loading="deleting">
            <template #icon><IconDelete /></template>
            批量删除
          </a-button>
        </a-popconfirm>
        <a-button @click="router.push('/albums')">
          <template #icon><IconLeft /></template>
          返回相册
        </a-button>
      </template>
    </AdminPageHeader>

    <a-alert v-if="selectedIds.length > 0" type="info" class="recycle-selection">
      已选中 {{ selectedIds.length }} 张照片：可以批量恢复到原相册，或永久删除。
      <a-button type="text" size="mini" @click="selectedKeys = []">取消选择</a-button>
    </a-alert>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search
            v-model="keywords"
            class="admin-filter-input"
            placeholder="在本页搜索照片名称或描述"
            allow-clear />
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">清空搜索</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 张已删除照片</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

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
              <a-button type="text" size="small" @click="restore([Number(record.id)])">恢复</a-button>
              <a-popconfirm content="永久删除这张照片？此操作不可撤销。" @ok="deletePermanently([Number(record.id)])">
                <a-button type="text" status="danger" size="small">永久删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconUndo"
              :title="keywords.trim() ? '没有匹配的照片' : '回收站是空的'"
              :description="keywords.trim() ? '换个关键词再试一次。' : '从相册里移除的照片会先放到这里，确认无误后再永久删除。'">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
              <a-button v-else size="small" @click="router.push('/albums')">返回相册</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconImage, IconLeft, IconRefresh, IconUndo } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, deleteAdminPhotos, listAdminPage, updateAdminPhotoDelete } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminPhoto } from '@benetnasch/api-contract'

const columns = [
  { title: '预览', dataIndex: 'photoSrc', width: 108, slotName: 'source' },
  { title: '照片名称', dataIndex: 'photoName', slotName: 'photoName', minWidth: 180 },
  { title: '描述', dataIndex: 'photoDesc', slotName: 'photoDesc', ellipsis: true, tooltip: true },
  { title: '操作', dataIndex: 'actions', width: 176, slotName: 'actions' }
]

const router = useRouter()
const photos = ref<AdminPhoto[]>([])
const selectedKeys = ref<Array<string | number>>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(18)
const total = ref(0)
const loading = ref(false)
const restoring = ref(false)
const deleting = ref(false)
const errorMessage = ref('')

const selectedIds = computed(() =>
  [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
)
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value, [18, 36, 72]))
/** The photos endpoint has no keyword filter, so search stays within the page. */
const filteredPhotos = computed(() => {
  const query = keywords.value.trim().toLowerCase()
  if (!query) return photos.value
  return photos.value.filter((photo) =>
    `${photo.photoName || ''} ${photo.photoDesc || ''}`.toLowerCase().includes(query)
  )
})

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function clearKeywords(): void {
  keywords.value = ''
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<AdminPhoto>('admin/photos', {
      current: current.value,
      size: pageSize.value,
      isDelete: 1
    })
    photos.value = page.items
    total.value = page.total
    // Drop selections that are no longer rendered to keep the toolbar honest.
    const available = new Set(photos.value.map((photo) => Number(photo.id)))
    selectedKeys.value = selectedKeys.value.filter((key) => available.has(Number(key)))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '回收站加载失败')
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

async function restore(ids: number[]): Promise<void> {
  const validIds = normalizeIds(ids)
  if (!validIds.length) return
  restoring.value = true
  try {
    await updateAdminPhotoDelete(validIds, 0)
    selectedKeys.value = []
    Message.success(`已恢复 ${validIds.length} 张照片`)
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片恢复失败'))
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
    Message.success(`已永久删除 ${validIds.length} 张照片`)
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片删除失败'))
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
