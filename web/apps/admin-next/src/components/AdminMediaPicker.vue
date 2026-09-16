<template>
  <a-modal
    v-model:visible="visible"
    :title="title"
    :footer="false"
    :mask-closable="false"
    width="900px"
    @cancel="close">
    <div class="media-picker-toolbar">
      <a-input-search
        v-model="prefix"
        class="media-picker-search"
        :placeholder="t('mediaPicker.filterHint')"
        allow-clear
        @search="reload" />
      <a-space>
        <input ref="fileInput" type="file" accept="image/*" multiple hidden @change="upload" />
        <a-button size="small" :loading="uploading" @click="fileInput?.click()">{{ t('common.upload') }}</a-button>
        <a-button size="small" :loading="loading" @click="load">{{ t('common.refresh') }}</a-button>
      </a-space>
    </div>
    <p class="media-picker-hint">{{ t('mediaPicker.hint') }}</p>

    <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

    <div v-if="loading" class="media-picker-grid">
      <div v-for="index in 8" :key="index" class="admin-skeleton media-picker-skeleton" />
    </div>

    <div v-else-if="assets.length" class="media-picker-grid">
      <div
        v-for="asset in assets"
        :key="asset.key"
        class="media-picker-item"
        role="button"
        tabindex="0"
        :title="asset.name"
        @click="select(asset)"
        @keydown.enter="select(asset)"
        @keydown.space.prevent="select(asset)">
        <AdminImagePreview :src="asset.url" :alt="asset.name" :width="150" :height="104" />
        <span class="media-picker-item-name">{{ asset.name }}</span>
        <small>{{ formatFileSize(asset.size) }}</small>
      </div>
    </div>

    <AdminEmptyState
      v-else
      :icon="IconImage"
      :title="t('mediaPicker.empty')"
      :description="t('mediaPicker.emptyHint')">
      <a-button type="primary" size="small" @click="fileInput?.click()">{{ t('common.upload') }}</a-button>
    </AdminEmptyState>

    <a-pagination
      v-if="total > pageSize"
      class="media-picker-pagination"
      :total="total"
      :current="current"
      :page-size="pageSize"
      show-total
      @change="changePage" />
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconImage } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, listAdminMedia, uploadAdminMedia } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import { t } from '@/i18n'
import { formatFileSize } from '@/utils/format'
import type { AdminMediaAsset } from '@stellar-beacon/api-contract'

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024

const props = withDefaults(defineProps<{ modelValue: boolean; title?: string }>(), { title: '' })
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; select: [asset: AdminMediaAsset] }>()

/** 标题默认走词条：语言切换后跟着变，父级显式传入时以父级为准。 */
const title = computed(() => props.title || t('mediaPicker.title'))

const pageSize = 24

const visible = ref(props.modelValue)
const assets = ref<AdminMediaAsset[]>([])
const prefix = ref('')
const current = ref(1)
const total = ref(0)
const loading = ref(false)
const uploading = ref(false)
const errorMessage = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

watch(() => props.modelValue, (value) => {
  visible.value = value
  if (value) void reload()
})

watch(visible, (value) => emit('update:modelValue', value))

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminMedia({
      current: current.value,
      size: pageSize,
      prefix: prefix.value.trim()
    })
    assets.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('mediaPicker.loadFailed'))
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function close(): void {
  visible.value = false
}

function select(asset: AdminMediaAsset): void {
  emit('select', asset)
  close()
}

async function upload(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return
  const rejected = files.filter((file) => !file.type.startsWith('image/') || file.size > MAX_UPLOAD_BYTES)
  const accepted = files.filter((file) => !rejected.includes(file))
  if (rejected.length) Message.warning(t('mediaPicker.skipped', { count: rejected.length }))
  if (!accepted.length) return
  uploading.value = true
  let uploaded: AdminMediaAsset | null = null
  let uploadedCount = 0
  const failures: string[] = []
  try {
    for (const file of accepted) {
      try {
        uploaded = await uploadAdminMedia(file)
        uploadedCount += 1
      } catch (error) {
        failures.push(`${file.name}：${apiErrorMessage(error, t('upload.failed'))}`)
      }
    }
    // Jump back to page 1 so the newest upload is visible, then auto-select it.
    if (uploadedCount > 0) {
      current.value = 1
      await load()
      Message.success(uploadedCount === 1
        ? t('mediaPicker.uploaded')
        : t('mediaPicker.uploadedCount', { count: uploadedCount }))
      if (uploaded) select(uploaded)
    }
    if (failures.length) {
      Message.error(t('mediaPicker.uploadFailedCount', { count: failures.length, detail: failures[0] }))
    }
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.media-picker-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  margin-bottom: 6px;
}

.media-picker-search {
  width: min(320px, 100%);
}

.media-picker-hint {
  margin: 0 0 14px;
  color: var(--admin-muted);
  font-size: 12px;
}

.media-picker-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 14px;
  max-height: 56vh;
  overflow: auto;
  padding: 2px;
}

.media-picker-item {
  min-width: 0;
  display: grid;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-card);
  color: var(--admin-ink);
  background: var(--admin-surface);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--admin-duration-fast) var(--admin-ease),
    background-color var(--admin-duration-fast) var(--admin-ease);
}

.media-picker-item:hover {
  border-color: var(--admin-brand);
  background: var(--admin-brand-soft);
}

.media-picker-item :deep(.admin-image-preview) {
  width: 100% !important;
  height: 104px !important;
}

.media-picker-item-name {
  overflow: hidden;
  font-size: 12px;
  font-weight: 620;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.media-picker-item small {
  color: var(--admin-subtle);
  font-size: 11px;
}

.media-picker-skeleton {
  height: 156px;
}

.media-picker-pagination {
  margin-top: 14px;
  justify-content: flex-end;
}
</style>
