<template>
  <a-result class="admin-placeholder" status="404" :title="t('page.notFound')">
    <template #subtitle>
      <span>{{ t('error.notFoundHint') }}</span>
      <br />
      <code class="admin-error-path">{{ route.fullPath }}</code>
    </template>
    <template #extra>
      <a-space>
        <a-button type="primary" @click="router.push('/')">{{ t('error.backHome') }}</a-button>
        <a-button @click="router.back()">{{ t('error.backPrev') }}</a-button>
      </a-space>
    </template>
  </a-result>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

import { t } from '@/i18n'

const route = useRoute()
const router = useRouter()
</script>

<style scoped>
.admin-error-path {
  display: inline-block;
  max-width: 440px;
  overflow: hidden;
  padding: 2px 8px;
  border-radius: 5px;
  background: var(--admin-surface-soft);
  font-size: 12px;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}
</style>
