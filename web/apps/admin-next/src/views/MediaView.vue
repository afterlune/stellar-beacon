<template>
  <section class="admin-page media-page">
    <AdminPageHeader title="图片资源" description="集中管理文章、说说和相册上传的图片，支持预览、复制和安全删除。">
      <template #actions>
        <a-space>
          <input ref="fileInput" type="file" accept="image/*" multiple hidden @change="uploadFiles" />
          <a-button type="primary" :loading="uploading" @click="fileInput?.click()">上传图片</a-button>
          <a-button status="danger" :disabled="!selectedKeys.length" @click="removeSelected">删除选中</a-button>
        </a-space>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-input-search v-model="prefix" class="admin-filter-input" placeholder="按目录筛选，例如 media/" allow-clear @search="reload" />
          <span class="admin-toolbar-caption">共 {{ total }} 张图片</span>
        </div>
        <a-button :loading="loading" @click="load">刷新</a-button>
      </div>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div v-if="loading" class="admin-page-loading">图片资源加载中…</div>
      <div v-else-if="assets.length" class="media-grid">
        <article v-for="asset in assets" :key="asset.key" class="media-card" :class="{ 'media-card-selected': selectedKeys.includes(asset.key) }">
          <label class="media-card-check"><input v-model="selectedKeys" type="checkbox" :value="asset.key" /></label>
          <AdminImagePreview :src="asset.url" :alt="asset.name" :width="220" :height="148" />
          <div class="media-card-copy"><strong :title="asset.name">{{ asset.name }}</strong><small>{{ formatSize(asset.size) }} · {{ formatDate(asset.lastModified) }}</small></div>
          <div class="media-card-actions"><a-button size="small" @click="copyUrl(asset.url)">复制地址</a-button><a-button v-if="asset.deletable" size="small" status="danger" @click="remove(asset.key)">删除</a-button></div>
        </article>
      </div>
      <a-empty v-else description="还没有图片资源，先上传一张吧" />
      <a-pagination v-if="total > pageSize" class="media-pagination" :total="total" :current="current" :page-size="pageSize" show-total show-jumper @change="changePage" />
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, deleteAdminMedia, listAdminMedia, uploadAdminMedia } from '@/api/http'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import type { AdminMediaAsset } from '@benetnasch/api-contract'

const assets = ref<AdminMediaAsset[]>([])
const prefix = ref('')
const current = ref(1)
const pageSize = 24
const total = ref(0)
const selectedKeys = ref<string[]>([])
const loading = ref(false)
const uploading = ref(false)
const errorMessage = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

onMounted(() => void load())

async function reload(): Promise<void> { current.value = 1; await load() }
async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try { const page = await listAdminMedia({ current: current.value, size: pageSize, prefix: prefix.value.trim() }); assets.value = page.items; total.value = page.total; selectedKeys.value = [] }
  catch (error) { errorMessage.value = apiErrorMessage(error, '图片资源加载失败') }
  finally { loading.value = false }
}
function changePage(page: number): void { current.value = page; void load() }

async function uploadFiles(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return
  uploading.value = true
  try { for (const file of files) await uploadAdminMedia(file); Message.success(`已上传 ${files.length} 张图片`); await reload() }
  catch (error) { Message.error(apiErrorMessage(error, '图片上传失败')) }
  finally { uploading.value = false }
}

async function remove(key: string): Promise<void> { await removeKeys([key]) }
async function removeSelected(): Promise<void> { await removeKeys(selectedKeys.value) }
async function removeKeys(keys: string[]): Promise<void> {
  if (!keys.length) return
  try { await deleteAdminMedia(keys); Message.success('图片已删除'); await load() }
  catch (error) { Message.error(apiErrorMessage(error, '图片删除失败')) }
}

async function copyUrl(url: string): Promise<void> {
  try { await navigator.clipboard.writeText(url); Message.success('图片地址已复制') }
  catch { Message.warning('浏览器不允许自动复制，请手动复制图片地址') }
}
function formatSize(value?: number): string { const size = Number(value || 0); return size < 1024 * 1024 ? `${Math.max(1, Math.round(size / 1024))} KB` : `${(size / 1024 / 1024).toFixed(1)} MB` }
function formatDate(value?: string): string { if (!value) return '—'; const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString('zh-CN') }
</script>

<style scoped>
.admin-toolbar-caption { color: var(--admin-muted); font-size: 12px; }.media-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; }.media-card { position: relative; min-width: 0; padding: 10px; border: 1px solid var(--admin-border); border-radius: 14px; background: var(--admin-surface); }.media-card-selected { border-color: var(--admin-brand); box-shadow: 0 0 0 3px var(--admin-brand-soft); }.media-card-check { position: absolute; z-index: 1; top: 16px; left: 16px; width: 20px; height: 20px; display: grid; place-items: center; border-radius: 6px; background: rgb(255 255 255 / 86%); }.media-card-copy { display: grid; gap: 4px; min-width: 0; margin: 10px 2px; }.media-card-copy strong,.media-card-copy small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.media-card-copy strong { color: var(--admin-ink); font-size: 13px; }.media-card-copy small { color: var(--admin-muted); font-size: 11px; }.media-card-actions { display: flex; gap: 6px; }.media-pagination { margin-top: 22px; }.media-card :deep(.admin-image-preview) { width: 100% !important; height: 150px !important; }
@media (max-width: 1180px) { .media-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 760px) { .media-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 480px) { .media-grid { grid-template-columns: 1fr; } }
</style>
