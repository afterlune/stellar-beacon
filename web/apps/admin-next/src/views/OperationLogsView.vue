<template>
  <a-tabs default-active-key="operations" lazy-load>
    <a-tab-pane key="operations" :title="t('logs.operation.title')">
      <LogListView mode="operation" />
    </a-tab-pane>
    <a-tab-pane key="content-audit" :title="t('logs.contentAudit.title')">
      <ContentAuditView />
    </a-tab-pane>
  </a-tabs>
</template>

<script setup lang="ts">
import ContentAuditView from '@/views/ContentAuditView.vue'
import LogListView from '@/views/LogListView.vue'
import { t } from '@/i18n'
</script>