<template>
  <a-result class="admin-placeholder" status="403" title="无权访问">
    <template #subtitle>当前账号没有访问此页面的权限。</template>
    <template #extra><a-button type="primary" @click="router.push('/')">返回首页</a-button></template>
  </a-result>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>
