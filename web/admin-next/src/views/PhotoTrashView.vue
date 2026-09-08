<template>
  <section class="admin-page">
    <AdminPageHeader title="照片回收站" description="恢复误删的照片，或清理不再需要的内容。">
      <template #actions>
        <a-space>
          <a-button :disabled="selectedIds.length === 0" @click="restoreSelected">批量恢复</a-button>
          <a-popconfirm content="永久删除选中的照片？此操作不可撤销。" @ok="deleteSelected">
            <a-button status="danger" :disabled="selectedIds.length === 0">批量删除</a-button>
          </a-popconfirm>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
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
            <img v-if="isHttpUrl(record.photoSrc)" class="photo-thumb" :src="record.photoSrc" alt="照片" />
            <span v-else>—</span>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="restore([Number(record.id)])">恢复</a-button>
              <a-popconfirm content="永久删除这张照片？此操作不可撤销。" @ok="deletePermanently([Number(record.id)])">
                <a-button type="text" status="danger" size="small">永久删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="回收站暂无照片" /></div></template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, deleteAdminPhotos, listAdminPhotos, updateAdminPhotoDelete } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { tablePagination } from '@/utils/pagination'
import type { AdminPhoto } from '@shared/api-contract'

const columns = [
  { title: '预览', dataIndex: 'photoSrc', width: 90, slotName: 'source' },
  { title: '名称', dataIndex: 'photoName' },
  { title: '描述', dataIndex: 'photoDesc', ellipsis: true, tooltip: true },
  { title: '操作', dataIndex: 'actions', width: 170, slotName: 'actions' }
]
const photos = ref<AdminPhoto[]>([])
const selectedKeys = ref<Array<string | number>>([])
const current = ref(1)
const pageSize = ref(18)
const total = ref(0)
const loading = ref(false)
const errorMessage = ref('')

const selectedIds = computed(() => selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPhotos({ current: current.value, size: pageSize.value, isDelete: 1 })
    photos.value = page.records
    total.value = page.count
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
  try {
    await updateAdminPhotoDelete(validIds, 0)
    selectedKeys.value = []
    Message.success('照片已恢复')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片恢复失败'))
  }
}

async function deletePermanently(ids: number[]): Promise<void> {
  const validIds = normalizeIds(ids)
  if (!validIds.length) return
  try {
    await deleteAdminPhotos(validIds)
    selectedKeys.value = []
    Message.success('照片已永久删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '照片删除失败'))
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
