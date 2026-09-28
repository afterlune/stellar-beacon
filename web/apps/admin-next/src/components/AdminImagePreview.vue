<template>
  <button
    class="admin-image-preview"
    :class="{ 'admin-image-preview-natural': fit === 'natural', 'admin-image-preview-contain': fit === 'contain' }"
    type="button"
    :style="style"
    :aria-label="t('image.previewOf', { alt: altLabel })"
    :disabled="!activeSrc"
    @click="open">
    <img v-if="activeSrc && !broken" :src="activeSrc" :alt="altLabel" loading="lazy" decoding="async" @error="handleError" />
    <span v-if="!activeSrc || broken" class="admin-image-preview-fallback">
      {{ broken ? t('image.unavailable') : t('image.none') }}
    </span>
  </button>

  <a-modal v-model:visible="visible" :title="altLabel" :footer="false" width="min(92vw, 1000px)">
    <div class="admin-image-preview-large">
      <img v-if="activeSrc && !broken" :src="activeSrc" :alt="altLabel" @error="handleError" />
      <span v-else class="admin-image-preview-fallback">{{ t('image.unavailable') }}</span>
    </div>
    <div v-if="activeSrc" class="admin-image-preview-meta">
      <span class="admin-muted-cell">{{ displaySrc }}</span>
      <a-button size="mini" @click="copySrc">{{ t('image.copyUrl') }}</a-button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { API_BASE_URL } from '@stellar-beacon/api-client'

import { t } from '@/i18n'

const props = withDefaults(defineProps<{
  src?: string
  alt?: string
  width?: number | string
  height?: number | string
  fit?: 'cover' | 'contain' | 'natural'
}>(), {
  src: '',
  alt: '',
  width: 96,
  height: 64,
  fit: 'cover'
})

/** 未传 alt 时用词条兜底，保证无障碍标签也跟着语言走。 */
const altLabel = computed(() => props.alt || t('image.alt'))

const visible = ref(false)
const broken = ref(false)
const fallbackSrc = ref('')

/** Accept both `120` and `'100%'` so callers never produce `100%px`. */
function toLength(value: number | string): string {
  return typeof value === 'number' ? `${value}px` : value
}

const style = computed(() => ({ width: toLength(props.width), height: toLength(props.height) }))

/** Some legacy object-storage URLs are missing a path separator. */
const normalizedSrc = computed(() => props.src.replace(
  /(aliyuncs\.com)(?=(?:talks|photos|articles|avatar|config)\/)/gi,
  '$1/'
))

/**
 * Resolve the renderable URL: OSS hosts are routed through the backend media
 * proxy (which hides credentials and normalises CORS), everything else is used
 * as-is so relative `/api` paths keep working.
 */
const displaySrc = computed(() => {
  const value = normalizedSrc.value
  if (!value) return ''
  try {
    const parsed = new URL(value, window.location.origin)
    const hostname = parsed.hostname.toLowerCase()
    if (hostname.endsWith('.aliyuncs.com')) {
      return `${API_BASE_URL}/public/media/proxy?url=${encodeURIComponent(value)}`
    }
  } catch {
    return value
  }
  return value
})

const activeSrc = computed(() => fallbackSrc.value || displaySrc.value)

watch(() => props.src, () => {
  broken.value = false
  fallbackSrc.value = ''
})

/** On proxy failure fall back to the raw URL once before giving up. */
function handleError(): void {
  if (normalizedSrc.value && activeSrc.value !== normalizedSrc.value) {
    fallbackSrc.value = normalizedSrc.value
    return
  }
  broken.value = true
}

function open(): void {
  if (!activeSrc.value || broken.value) return
  visible.value = true
}

async function copySrc(): Promise<void> {
  try {
    await navigator.clipboard.writeText(displaySrc.value)
    Message.success(t('image.copied'))
  } catch {
    Message.warning(t('common.copyFailed'))
  }
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

.admin-image-preview:disabled {
  cursor: default;
}

.admin-image-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 180ms var(--admin-ease);
}

.admin-image-preview-contain img {
  object-fit: contain;
}

.admin-image-preview-natural {
  height: auto !important;
  display: block;
}

.admin-image-preview-natural img {
  height: auto;
  object-fit: contain;
  display: block;
}

.admin-image-preview:not(:disabled):hover img {
  transform: scale(1.05);
}

.admin-image-preview-fallback {
  padding: 6px;
  color: var(--admin-muted);
  font-size: 11px;
}

.admin-image-preview-large {
  display: grid;
  min-height: 220px;
  place-items: center;
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}

.admin-image-preview-large img {
  max-width: 100%;
  max-height: 70vh;
  object-fit: contain;
}

.admin-image-preview-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-top: 12px;
}

.admin-image-preview-meta span {
  overflow: hidden;
  font-family: ui-monospace, "SFMono-Regular", Consolas, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
