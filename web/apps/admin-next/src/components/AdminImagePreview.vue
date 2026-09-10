<template>
  <button class="admin-image-preview" type="button" :style="style" :aria-label="`预览${alt}`" @click="open">
    <img v-if="activeSrc" :src="activeSrc" :alt="alt" @error="handleError" />
    <span v-if="!activeSrc || broken" class="admin-image-preview-fallback">{{ broken ? '图片不可用' : '暂无图片' }}</span>
  </button>
  <a-modal v-model:visible="visible" :title="alt" :footer="false" width="min(90vw, 960px)">
    <div class="admin-image-preview-large">
      <img v-if="activeSrc && !broken" :src="activeSrc" :alt="alt" @error="handleError" />
      <span v-else>图片不可用</span>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  src?: string
  alt?: string
  width?: number | string
  height?: number | string
}>(), {
  src: '',
  alt: '图片',
  width: 96,
  height: 64
})

const visible = ref(false)
const broken = ref(false)
const fallbackSrc = ref('')
const style = computed(() => ({ width: `${props.width}px`, height: `${props.height}px` }))
const normalizedSrc = computed(() => props.src.replace(
  /(aliyuncs\.com)(?=(?:talks|photos|articles|avatar|config)\/)/gi,
  '$1/'
))
const displaySrc = computed(() => {
  if (!normalizedSrc.value) return ''
  try {
    const parsed = new URL(normalizedSrc.value, window.location.origin)
    const hostname = parsed.hostname.toLowerCase()
    if (hostname.endsWith('.aliyuncs.com') || hostname === 'i.example.invalid') {
      return `/api/v1/public/media/proxy?url=${encodeURIComponent(normalizedSrc.value)}`
    }
  } catch {
    return normalizedSrc.value
  }
  return normalizedSrc.value
})
const activeSrc = computed(() => fallbackSrc.value || displaySrc.value)

watch(() => props.src, () => {
  broken.value = false
  fallbackSrc.value = ''
})

function handleError(): void {
  if (normalizedSrc.value && activeSrc.value !== normalizedSrc.value) {
    fallbackSrc.value = normalizedSrc.value
    return
  }
  broken.value = true
}

function open(): void {
  if (props.src && !broken.value) visible.value = true
}
</script>

<style scoped>
.admin-image-preview {
  display: inline-grid;
  place-items: center;
  overflow: hidden;
  padding: 0;
  border: 1px solid var(--admin-border);
  border-radius: 10px;
  background: var(--admin-surface-soft);
  cursor: pointer;
}

.admin-image-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 180ms ease;
}

.admin-image-preview:hover img { transform: scale(1.05); }
.admin-image-preview-fallback { padding: 6px; color: var(--admin-muted); font-size: 11px; }
.admin-image-preview-large { display: grid; min-height: 220px; place-items: center; background: #f6f8fc; }
.admin-image-preview-large img { max-width: 100%; max-height: 70vh; object-fit: contain; }
.admin-image-preview-large span { color: var(--admin-muted); }
</style>
