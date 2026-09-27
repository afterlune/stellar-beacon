<template>
  <a-tag class="admin-status-tag" :color="meta.color">
    <span v-if="meta.dot" class="admin-status-dot" :style="{ background: 'currentColor' }" aria-hidden="true" />
    {{ meta.label }}
  </a-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import { t } from '@/i18n'

/** 语义状态：把散落在各视图里的数字魔法值收敛到一处。 */
export type StatusKind =
  | 'enabled' | 'disabled'
  | 'public' | 'private' | 'draft'
  | 'reviewed' | 'pending'
  | 'top' | 'normal'
  | 'success' | 'failed'
  | 'original' | 'repost' | 'translated'
  | 'anonymous' | 'authenticated'

const props = withDefaults(defineProps<{
  kind: StatusKind
  /** 覆盖默认文案 */
  label?: string
}>(), { label: '' })

/** 文案键指向 `common.status.*`：颜色在这里，措辞在词典里。 */
const registry: Record<StatusKind, { key: string; color: string; dot?: boolean }> = {
  enabled: { key: 'status.enabled', color: 'green', dot: true },
  disabled: { key: 'status.disabled', color: 'orange', dot: true },
  public: { key: 'status.published', color: 'green' },
  private: { key: 'status.private', color: 'orange' },
  draft: { key: 'status.draft', color: 'arcoblue' },
  reviewed: { key: 'status.approved', color: 'green' },
  pending: { key: 'status.pending', color: 'orange', dot: true },
  top: { key: 'status.pinned', color: 'arcoblue' },
  normal: { key: 'status.normal', color: 'gray' },
  success: { key: 'status.success', color: 'green' },
  failed: { key: 'status.failed', color: 'red' },
  original: { key: 'status.original', color: 'arcoblue' },
  repost: { key: 'status.reprint', color: 'purple' },
  translated: { key: 'status.translate', color: 'cyan' },
  anonymous: { key: 'status.anonymous', color: 'green' },
  authenticated: { key: 'status.loginRequired', color: 'gray' }
}

const meta = computed(() => {
  const entry = registry[props.kind]
  return {
    label: props.label || t(entry.key),
    color: entry.color,
    dot: entry.dot
  }
})
</script>

<style scoped>
.admin-status-dot {
  width: 6px;
  height: 6px;
  display: inline-block;
  margin-right: 5px;
  border-radius: 50%;
  vertical-align: middle;
}
</style>
