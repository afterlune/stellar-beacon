<template>
  <a-result class="admin-placeholder" status="warning" :title="String(route.meta.title || t('dashboard.placeholder.title'))">
    <template #subtitle>
      <p>{{ t('dashboard.placeholder.description') }}</p>
      <a-descriptions :column="1" size="small" bordered class="placeholder-meta">
        <a-descriptions-item :label="t('dashboard.placeholder.menuPath')">{{ route.path }}</a-descriptions-item>
        <a-descriptions-item :label="t('dashboard.placeholder.backendComponent')">{{ String(route.meta.menuComponent || t('dashboard.placeholder.notProvided')) }}</a-descriptions-item>
      </a-descriptions>
      <p class="placeholder-hint">
        {{ t('dashboard.placeholder.hint') }}
      </p>
    </template>
    <template #extra>
      <a-space>
        <a-button type="primary" @click="router.push('/')">{{ t('dashboard.placeholder.backHome') }}</a-button>
        <a-button @click="copyPath">{{ t('dashboard.placeholder.copyPath') }}</a-button>
      </a-space>
    </template>
  </a-result>
</template>

<script setup lang="ts">
import { Message } from '@arco-design/web-vue'
import { useRoute, useRouter } from 'vue-router'

import { t } from '@/i18n'

const route = useRoute()
const router = useRouter()

async function copyPath(): Promise<void> {
  try {
    await navigator.clipboard.writeText(String(route.meta.menuComponent || route.path))
    Message.success(t('dashboard.placeholder.copied'))
  } catch {
    Message.warning(t('common.copyFailed'))
  }
}
</script>

<style scoped>
.placeholder-meta {
  max-width: 520px;
  margin: 4px auto 12px;
  text-align: left;
}

.placeholder-hint {
  max-width: 520px;
  margin: 0 auto;
  color: var(--admin-muted);
  font-size: 12px;
  line-height: 1.7;
}
</style>
