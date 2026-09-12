<template>
  <a-result class="admin-placeholder" status="404" title="页面不存在">
    <template #subtitle>
      <span>没有找到这个地址对应的页面。</span>
      <br />
      <code class="admin-error-path">{{ route.fullPath }}</code>
    </template>
    <template #extra>
      <a-space>
        <a-button type="primary" @click="router.push('/')">返回首页</a-button>
        <a-button @click="router.back()">返回上一页</a-button>
      </a-space>
    </template>
  </a-result>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

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
