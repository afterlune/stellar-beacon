<template>
  <a-result class="admin-placeholder" status="warning" :title="String(route.meta.title || '页面未接入')">
    <template #subtitle>
      <p>这个菜单指向的视图还没有在前端注册，因此暂时无法展示业务内容。</p>
      <a-descriptions :column="1" size="small" bordered class="placeholder-meta">
        <a-descriptions-item label="菜单路径">{{ route.path }}</a-descriptions-item>
        <a-descriptions-item label="后端组件">{{ String(route.meta.menuComponent || '未提供') }}</a-descriptions-item>
      </a-descriptions>
      <p class="placeholder-hint">
        请联系管理员在「菜单管理」中把组件路径改为已注册的视图，或在前端补充对应实现。
      </p>
    </template>
    <template #extra>
      <a-space>
        <a-button type="primary" @click="router.push('/')">返回首页</a-button>
        <a-button @click="copyPath">复制菜单路径</a-button>
      </a-space>
    </template>
  </a-result>
</template>

<script setup lang="ts">
import { Message } from '@arco-design/web-vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

async function copyPath(): Promise<void> {
  try {
    await navigator.clipboard.writeText(String(route.meta.menuComponent || route.path))
    Message.success('已复制组件路径')
  } catch {
    Message.warning('浏览器不允许自动复制，请手动选择文本')
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
