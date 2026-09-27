<template>
  <div v-if="count > 0" class="admin-batch-bar">
    <span class="admin-batch-bar-count">
      {{ t('batch.selectedPrefix') }} <strong>{{ count }}</strong> {{ t('batch.selectedSuffix') }}
      <span v-if="hint" class="admin-batch-bar-hint">{{ hint }}</span>
    </span>
    <div class="admin-batch-bar-actions"><slot /></div>
    <a-button type="text" size="small" class="admin-batch-bar-clear" @click="emit('clear')">{{ t('common.deselect') }}</a-button>
  </div>
</template>

<script setup lang="ts">
import { t } from '@/i18n'

withDefaults(defineProps<{
  count: number
  /** 选中范围的补充说明，例如「本页共 N 条」。 */
  hint?: string
}>(), {
  hint: ''
})

const emit = defineEmits<{ clear: [] }>()
</script>
