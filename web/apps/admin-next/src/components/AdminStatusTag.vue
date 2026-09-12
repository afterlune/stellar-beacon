<template>
  <a-tag class="admin-status-tag" :color="meta.color">
    <span v-if="meta.dot" class="admin-status-dot" :style="{ background: 'currentColor' }" aria-hidden="true" />
    {{ meta.label }}
  </a-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

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

const registry: Record<StatusKind, { label: string; color: string; dot?: boolean }> = {
  enabled: { label: '启用', color: 'green', dot: true },
  disabled: { label: '禁用', color: 'orange', dot: true },
  public: { label: '公开', color: 'green' },
  private: { label: '私密', color: 'orange' },
  draft: { label: '草稿', color: 'arcoblue' },
  reviewed: { label: '已审核', color: 'green' },
  pending: { label: '待审核', color: 'orange', dot: true },
  top: { label: '置顶', color: 'arcoblue' },
  normal: { label: '普通', color: 'gray' },
  success: { label: '成功', color: 'green' },
  failed: { label: '失败', color: 'red' },
  original: { label: '原创', color: 'arcoblue' },
  repost: { label: '转载', color: 'purple' },
  translated: { label: '翻译', color: 'cyan' },
  anonymous: { label: '匿名', color: 'green' },
  authenticated: { label: '需登录', color: 'gray' }
}

const meta = computed(() => {
  const entry = registry[props.kind]
  return props.label ? { ...entry, label: props.label } : entry
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
