<template>
  <a-result class="admin-placeholder" status="404" title="页面不存在">
    <template #subtitle>请从左侧菜单选择一个已授权页面。</template>
    <template #extra><a-button type="primary" @click="router.push('/')">返回首页</a-button></template>
  </a-result>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>
