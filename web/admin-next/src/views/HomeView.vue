<template>
  <section class="admin-home">
    <div class="admin-home-hero">
      <div class="admin-home-hero-copy">
        <div class="admin-home-eyebrow">Benetnasch / 编辑台</div>
        <h2>让每一次发布，都更从容。</h2>
        <p>
          欢迎回到编辑台，{{ auth.user?.nickname || auth.user?.username || '管理员' }}。从文章、评论到站点设置，把博客维护成一件轻松而有秩序的事。
        </p>
      </div>
      <a-button class="admin-home-hero-action" type="primary" size="large" @click="goToFirstEntry">
        开始整理内容
        <template #icon><IconArrowRight /></template>
      </a-button>
    </div>

    <div class="admin-home-section">
      <div class="admin-home-section-heading">
        <h3>今天的编辑台</h3>
        <p>菜单与权限由后端 RBAC 统一管理</p>
      </div>
      <div class="admin-home-grid">
        <a-card class="admin-card admin-home-card" :bordered="false">
          <div class="admin-home-metric-title">可用模块</div>
          <div class="admin-home-metric-value">{{ leafMenuCount }}</div>
          <p>当前账号可以访问的内容、媒体和运维入口。</p>
        </a-card>
        <a-card class="admin-card admin-home-card" :bordered="false">
          <div class="admin-home-metric-title">账号状态</div>
          <div class="admin-home-metric-value">已认证</div>
          <p>会话沿用现有 Token 机制，退出后会立即清理本地状态。</p>
        </a-card>
        <a-card class="admin-card admin-home-card" :bordered="false">
          <div class="admin-home-metric-title">权限来源</div>
          <div class="admin-home-metric-value">RBAC</div>
          <p>页面路由与菜单可见性均以服务端返回结果为准。</p>
        </a-card>
      </div>
    </div>

    <div class="admin-home-section">
      <div class="admin-home-section-heading">
        <h3>快速入口</h3>
        <p>只展示当前账号已授权的页面</p>
      </div>
      <div v-if="quickLinks.length" class="admin-home-quick-grid">
        <button v-for="link in quickLinks" :key="link.path" class="admin-quick-link" type="button" @click="go(link.path)">
          <span class="admin-quick-link-icon" aria-hidden="true"><component :is="menuIconFor(link)" /></span>
          <span class="admin-quick-link-copy">
            <span class="admin-quick-link-title">{{ link.name }}</span>
            <span class="admin-quick-link-caption">打开管理页面</span>
          </span>
          <span class="admin-quick-link-arrow" aria-hidden="true"><IconArrowRight /></span>
        </button>
      </div>
      <a-empty v-else description="暂无可用入口" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { IconArrowRight } from '@arco-design/web-vue/es/icon'

import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { visibleChildren, type NormalizedMenu } from '@/types'
import { menuIconFor } from '@/utils/menu-icon'

const auth = useAuthStore()
const menuStore = useMenuStore()
const router = useRouter()

const quickLinks = computed(() => flattenMenus(menuStore.visibleMenus).filter((menu) => menu.path !== '/').slice(0, 6))
const leafMenuCount = computed(() => flattenMenus(menuStore.visibleMenus).length)

function flattenMenus(menus: NormalizedMenu[]): NormalizedMenu[] {
  const result: NormalizedMenu[] = []
  for (const menu of menus) {
    const children = visibleChildren(menu)
    if (children.length) result.push(...flattenMenus(children))
    else result.push(menu)
  }
  return result
}

function go(path: string): void {
  void router.push(path)
}

function goToFirstEntry(): void {
  go(quickLinks.value[0]?.path || '/')
}
</script>
