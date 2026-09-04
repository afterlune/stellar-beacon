<template>
  <a-layout class="admin-shell">
    <a-layout-sider v-model:collapsed="collapsed" collapsible breakpoint="xl" :width="240">
      <div class="admin-brand">
        <span class="admin-brand-mark" aria-hidden="true" />
        <span v-show="!collapsed">Benetnasch Admin</span>
      </div>
      <div v-if="menuStore.loading" class="admin-menu-state">菜单加载中…</div>
      <a-alert v-else-if="menuStore.error" class="admin-menu-state" type="error" :show-icon="false">
        菜单加载失败，请刷新重试。
      </a-alert>
      <a-alert v-else-if="menuStore.visibleMenus.length === 0" class="admin-menu-state" type="warning" :show-icon="false">
        当前账号没有可见菜单，请联系管理员分配权限。
      </a-alert>
      <a-menu
        v-else
        :selected-keys="[route.path]"
        :auto-open="true"
        :style="{ width: '100%' }"
        @menu-item-click="navigate">
        <template v-for="menu in menuStore.visibleMenus" :key="menu.path">
          <a-sub-menu v-if="group(menu)" :key="menu.path">
            <template #title>
              <span class="admin-menu-icon" aria-hidden="true">{{ iconText(menu.icon) }}</span>
              {{ menu.name }}
            </template>
            <a-menu-item v-for="child in visibleChildren(menu)" :key="menuItemPath(menu, child)">
              <span class="admin-menu-icon" aria-hidden="true">{{ iconText(child.icon) }}</span>
              {{ child.name }}
            </a-menu-item>
          </a-sub-menu>
          <a-menu-item v-else :key="menuItemPath(menu)">
            <span class="admin-menu-icon" aria-hidden="true">{{ iconText(menu.icon) }}</span>
            {{ menu.name }}
          </a-menu-item>
        </template>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <header class="admin-topbar">
        <div class="admin-topbar-title">
          <a-button type="text" aria-label="切换菜单" @click="collapsed = !collapsed">
            {{ collapsed ? '☰' : '‹' }}
          </a-button>
          <h1>{{ String(route.meta.title || '管理后台') }}</h1>
        </div>
        <a-dropdown trigger="click">
          <button class="admin-user" type="button">
            <a-avatar :size="32" :image-url="auth.user?.avatar">{{ avatarText }}</a-avatar>
            <span>{{ auth.user?.nickname || auth.user?.username || '管理员' }}</span>
            <span aria-hidden="true">⌄</span>
          </button>
          <template #content>
            <a-doption @click="logout">退出登录</a-doption>
          </template>
        </a-dropdown>
      </header>
      <main class="admin-content">
        <router-view />
      </main>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { resetMenuRoutes } from '@/router'
import { isMenuGroup, menuItemPath, visibleChildren, type NormalizedMenu } from '@/types'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const menuStore = useMenuStore()
const collapsed = ref(false)

const avatarText = computed(() => (auth.user?.nickname || auth.user?.username || '管').slice(0, 1).toUpperCase())

function navigate(path: string): void {
  void router.push(path)
}

function logout(): void {
  void auth.logout().finally(() => {
    resetMenuRoutes()
    menuStore.reset()
    void router.replace({ name: 'login' })
  })
}

function iconText(icon: string | undefined): string {
  const value = (icon || '').replace(/^iconfont\s+/, '').replace(/^el-icon-[a-z-]+$/, '')
  return value ? value.slice(0, 1).toUpperCase() : '◈'
}

function group(menu: NormalizedMenu): boolean {
  return isMenuGroup(menu)
}
</script>
