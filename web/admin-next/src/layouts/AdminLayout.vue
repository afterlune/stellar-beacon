<template>
  <a-layout class="admin-shell">
    <a-layout-sider v-model:collapsed="collapsed" class="admin-sider" collapsible breakpoint="xl" :width="232" :collapsed-width="72">
      <div class="admin-brand">
        <span class="admin-brand-mark" aria-hidden="true" />
        <span v-show="!collapsed">Benetnasch 编辑台</span>
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
              <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
              {{ menu.name }}
            </template>
            <a-menu-item v-for="child in visibleChildren(menu)" :key="menuItemPath(menu, child)">
              <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(child)" /></span>
              {{ child.name }}
            </a-menu-item>
          </a-sub-menu>
          <a-menu-item v-else :key="menuItemPath(menu)">
            <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
            {{ menu.name }}
          </a-menu-item>
        </template>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <header class="admin-topbar">
        <div class="admin-topbar-title">
          <a-button class="admin-menu-toggle" type="text" aria-label="切换菜单" @click="collapsed = !collapsed">
            <IconMenuUnfold v-if="collapsed" />
            <IconMenuFold v-else />
          </a-button>
          <div class="admin-topbar-context">
            <nav class="admin-breadcrumb" aria-label="面包屑">
              <span>Benetnasch</span>
              <IconRight class="admin-breadcrumb-sep" aria-hidden="true" />
              <span>管理台</span>
              <IconRight class="admin-breadcrumb-sep" aria-hidden="true" />
              <span class="admin-breadcrumb-current">{{ String(route.meta.title || '首页') }}</span>
            </nav>
            <h1>{{ String(route.meta.title || '首页') }}</h1>
          </div>
        </div>
        <div class="admin-topbar-actions">
          <a-button class="admin-theme-toggle" type="text" :aria-label="themeStore.theme === 'dark' ? '切换为浅色' : '切换为深色'" @click="themeStore.toggle()">
            <IconSun v-if="themeStore.theme === 'dark'" />
            <IconMoon v-else />
          </a-button>
          <a-dropdown trigger="click">
            <button class="admin-user" type="button">
              <a-avatar :size="32" :image-url="auth.user?.avatar">{{ avatarText }}</a-avatar>
              <span class="admin-user-name">{{ auth.user?.nickname || auth.user?.username || '管理员' }}</span>
              <IconDown class="admin-user-chevron" aria-hidden="true" />
            </button>
            <template #content>
              <a-doption @click="logout">退出登录</a-doption>
            </template>
          </a-dropdown>
        </div>
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
import { IconDown, IconMenuFold, IconMenuUnfold, IconMoon, IconRight, IconSun } from '@arco-design/web-vue/es/icon'

import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { useThemeStore } from '@/stores/theme'
import { resetMenuRoutes } from '@/router'
import { isMenuGroup, menuItemPath, visibleChildren, type NormalizedMenu } from '@/types'
import { menuIconFor } from '@/utils/menu-icon'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
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

function group(menu: NormalizedMenu): boolean {
  return isMenuGroup(menu)
}
</script>
