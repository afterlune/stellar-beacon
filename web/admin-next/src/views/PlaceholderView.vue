<template>
  <a-result class="admin-placeholder" status="info" :title="String(route.meta.title || '模块迁移中')">
    <template #subtitle>
      <span>此菜单已完成权限与路由接入，业务页面正在迁移。</span>
      <br />
      <code>{{ String(route.meta.menuComponent || 'unknown') }}</code>
    </template>
    <template #extra>
      <a-button type="primary" @click="router.push('/')">返回首页</a-button>
    </template>
  </a-result>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
</script>
