<template>
  <a-result class="admin-placeholder" status="403" :title="t('page.forbidden')">
    <template #subtitle>
      <span>{{ t('error.forbiddenHint') }}</span>
      <br />
      <span class="admin-muted-cell">{{ t('error.forbiddenAction') }}</span>
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
import { useRouter } from 'vue-router'

import { t } from '@/i18n'

const router = useRouter()
</script>
