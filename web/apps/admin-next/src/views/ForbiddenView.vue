<template>
  <a-result class="admin-placeholder" status="403" title="无权访问">
    <template #subtitle>
      <span>当前账号没有访问此页面的权限。</span>
      <br />
      <span class="admin-muted-cell">如需访问，请让管理员在「角色管理」中为你的角色勾选对应菜单。</span>
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
import { useRouter } from 'vue-router'

const router = useRouter()
</script>
