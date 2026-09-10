<template>
  <a-modal v-model:visible="visible" :title="title" :footer="false" width="860px" @cancel="close">
    <div class="media-picker-toolbar">
      <span class="admin-muted-cell">选择一张图片作为封面或内容资源</span>
      <span>
        <input ref="fileInput" type="file" accept="image/*" hidden @change="upload" />
        <a-button size="small" :loading="uploading" @click="fileInput?.click()">上传图片</a-button>
      </span>
    </div>
    <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
    <div v-if="loading" class="admin-page-loading">资源加载中…</div>
    <div v-else-if="assets.length" class="media-picker-grid">
      <button v-for="asset in assets" :key="asset.key" type="button" class="media-picker-item" @click="select(asset)">
        <AdminImagePreview :src="asset.url" :alt="asset.name" :width="150" :height="104" />
        <span>{{ asset.name }}</span>
      </button>
    </div>
    <a-empty v-else description="还没有可选择的图片" />
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { apiErrorMessage, listAdminMedia, uploadAdminMedia } from '@/api/http'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import type { AdminMediaAsset } from '@benetnasch/api-contract'

const props = withDefaults(defineProps<{ modelValue: boolean; title?: string }>(), { title: '图片资源' })
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; select: [asset: AdminMediaAsset] }>()
const visible = ref(props.modelValue)
const assets = ref<AdminMediaAsset[]>([])
const loading = ref(false)
const uploading = ref(false)
const errorMessage = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

watch(() => props.modelValue, value => { visible.value = value; if (value) void load() })
watch(visible, value => emit('update:modelValue', value))

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try { assets.value = (await listAdminMedia({ current: 1, size: 60 })).items }
  catch (error) { errorMessage.value = apiErrorMessage(error, '图片资源加载失败') }
  finally { loading.value = false }
}

function close(): void { visible.value = false }
function select(asset: AdminMediaAsset): void { emit('select', asset); close() }

async function upload(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  try { await uploadAdminMedia(file); await load() }
  catch (error) { errorMessage.value = apiErrorMessage(error, '图片上传失败') }
  finally { uploading.value = false }
}
</script>

<style scoped>
.media-picker-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 16px; }.media-picker-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; max-height: 58vh; overflow: auto; padding: 2px; }.media-picker-item { display: grid; gap: 8px; min-width: 0; padding: 8px; border: 1px solid var(--admin-border); border-radius: 12px; color: var(--admin-ink); background: var(--admin-surface); text-align: left; cursor: pointer; }.media-picker-item:hover { border-color: var(--admin-brand); background: var(--admin-brand-soft); }.media-picker-item > span { overflow: hidden; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
@media (max-width: 680px) { .media-picker-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
