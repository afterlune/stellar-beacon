<template>
  <div v-if="count > 0" class="admin-batch-bar">
    <span class="admin-batch-bar-count">
      已选 <strong>{{ count }}</strong> 项
      <span v-if="hint" class="admin-batch-bar-hint">{{ hint }}</span>
    </span>
    <div class="admin-batch-bar-actions"><slot /></div>
    <a-button type="text" size="small" class="admin-batch-bar-clear" @click="emit('clear')">取消选择</a-button>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  count: number
  /** 选中范围的补充说明，例如「本页共 N 条」。 */
  hint?: string
}>(), {
  hint: ''
})

const emit = defineEmits<{ clear: [] }>()
</script>
