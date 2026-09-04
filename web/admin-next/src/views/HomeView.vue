<template>
  <section>
    <a-card>
      <template #title>欢迎回来，{{ auth.user?.nickname || auth.user?.username || '管理员' }}</template>
      <p>这是 Vue 3 + Vite + Pinia + Arco Design 的新后台壳。业务模块会按菜单权限逐步迁移。</p>
    </a-card>
    <div class="admin-home-grid">
      <a-card class="admin-home-card">
        <a-statistic title="动态菜单" :value="menuStore.visibleMenus.length" />
        <p>菜单由后端 RBAC 返回，页面路由只接受白名单组件映射。</p>
      </a-card>
      <a-card class="admin-home-card">
        <div class="admin-home-metric-title">会话状态</div>
        <div class="admin-home-metric-value">已认证</div>
        <p>Token 与现有后台共用 sessionStorage 的 token 键，便于灰度回滚。</p>
      </a-card>
      <a-card class="admin-home-card">
        <div class="admin-home-metric-title">迁移策略</div>
        <div class="admin-home-metric-value">渐进式</div>
        <p>旧后台保持可构建，新后台按模块切换，不修改生产 Caddy 根目录。</p>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'

const auth = useAuthStore()
const menuStore = useMenuStore()
</script>
