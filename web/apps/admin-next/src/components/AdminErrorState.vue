<template>
  <section class="admin-error-state" role="alert">
    <span class="admin-error-state-icon" aria-hidden="true"><IconExclamationCircle /></span>
    <div class="admin-error-state-copy">
      <strong>{{ title }}</strong>
      <p>{{ error }}</p>
    </div>
    <a-button size="small" :loading="retrying" @click="retry">
      <template #icon><IconRefresh /></template>
      重试
    </a-button>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { IconExclamationCircle, IconRefresh } from '@arco-design/web-vue/es/icon'

const props = withDefaults(defineProps<{
  error: string
  title?: string
}>(), {
  title: '数据加载失败'
})

const emit = defineEmits<{ retry: [] }>()

// 重试中的即时反馈：父级刷新完成后 error 会被清空或替换，这里以 error 变化收尾。
const retrying = ref(false)
watch(() => props.error, () => {
  retrying.value = false
})

function retry(): void {
  retrying.value = true
  emit('retry')
}
</script>
