<template>
  <a-layout class="admin-shell">
    <a-layout-sider
      v-model:collapsed="collapsed"
      class="admin-sider"
      collapsible
      :width="236"
      :collapsed-width="72"
      :trigger="null">
      <div class="admin-brand">
        <BrandMark />
        <span class="admin-brand-text">
          <span class="admin-brand-name">星际信标</span>
          <span class="admin-brand-tagline">Editorial Admin</span>
        </span>
      </div>

      <nav class="admin-nav" aria-label="主导航">
        <div v-if="menuStore.loading" class="admin-menu-state">菜单加载中，请稍候…</div>
        <div v-else-if="menuStore.error" class="admin-menu-state">
          <strong>菜单加载失败</strong>
          <p>请检查网络或登录状态后重试。</p>
          <a-button size="small" long @click="reloadMenus">重新加载</a-button>
        </div>
        <div v-else-if="menuStore.visibleMenus.length === 0" class="admin-menu-state">
          <strong>暂无可用菜单</strong>
          <p>当前账号没有可见菜单，请联系管理员分配权限。</p>
        </div>
        <template v-else>
          <div class="admin-nav-label">工作区</div>
          <a-menu
            :selected-keys="[route.path]"
            :auto-open="true"
            :style="{ width: '100%' }"
            @menu-item-click="navigate">
            <template v-for="menu in menuStore.visibleMenus" :key="menu.path">
              <a-menu-item v-if="singleWrapper(menu)" :key="menuItemPath(menu, visibleChildren(menu)[0])">
                <template #icon>
                  <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
                </template>
                <span class="admin-menu-text">{{ menu.name }}</span>
              </a-menu-item>
              <a-sub-menu v-else-if="group(menu)" :key="menu.path">
                <template #title>
                  <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
                  <span class="admin-menu-text">{{ menu.name }}</span>
                </template>
                <a-menu-item v-for="child in visibleChildren(menu)" :key="menuItemPath(menu, child)">
                  <template #icon>
                    <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(child)" /></span>
                  </template>
                  <span class="admin-menu-text">{{ child.name }}</span>
                </a-menu-item>
              </a-sub-menu>
              <a-menu-item v-else :key="menuItemPath(menu)">
                <template #icon>
                  <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
                </template>
                <span class="admin-menu-text">{{ menu.name }}</span>
              </a-menu-item>
            </template>
          </a-menu>
        </template>
      </nav>

      <div class="admin-nav-footer">
        <span class="admin-nav-user">
          <span class="admin-nav-avatar" aria-hidden="true">{{ avatarText }}</span>
          <span class="admin-nav-user-copy">
            <strong>{{ auth.user?.nickname || auth.user?.username || '管理员' }}</strong>
            <span>后端 RBAC 决定最终可见范围</span>
          </span>
        </span>
      </div>
    </a-layout-sider>

    <a-layout>
      <header class="admin-topbar">
        <div class="admin-topbar-title">
          <button
            class="admin-icon-button"
            type="button"
            :aria-label="collapsed ? '展开侧边导航' : '收起侧边导航'"
            :aria-expanded="!collapsed"
            @click="collapsed = !collapsed">
            <IconMenuUnfold v-if="collapsed" />
            <IconMenuFold v-else />
          </button>
          <div class="admin-topbar-context">
            <nav class="admin-breadcrumb" aria-label="面包屑">
              <button class="admin-breadcrumb-link" type="button" @click="router.push('/')">管理台</button>
              <template v-for="crumb in breadcrumbs" :key="crumb.path">
                <IconRight class="admin-breadcrumb-sep" aria-hidden="true" />
                <button class="admin-breadcrumb-link" type="button" @click="router.push(crumb.path)">{{ crumb.title }}</button>
              </template>
              <IconRight class="admin-breadcrumb-sep" aria-hidden="true" />
              <span class="admin-breadcrumb-current" aria-current="page">{{ pageTitle }}</span>
            </nav>
            <h1>{{ pageTitle }}</h1>
          </div>
        </div>

        <div class="admin-topbar-actions">
          <button class="admin-search-trigger" type="button" aria-label="快速跳转" @click="paletteVisible = true">
            <IconSearch aria-hidden="true" />
            <span>快速跳转</span>
            <kbd>Ctrl K</kbd>
          </button>
          <a-tooltip :content="themeStore.theme === 'dark' ? '切换为浅色主题' : '切换为深色主题'" position="bottom">
            <button
              class="admin-icon-button"
              type="button"
              :aria-label="themeStore.theme === 'dark' ? '切换为浅色主题' : '切换为深色主题'"
              @click="themeStore.toggle()">
              <IconSun v-if="themeStore.theme === 'dark'" />
              <IconMoon v-else />
            </button>
          </a-tooltip>
          <a-dropdown trigger="click">
            <button class="admin-user" type="button" aria-label="账号菜单">
              <a-avatar :size="32" :image-url="auth.user?.avatar">{{ avatarText }}</a-avatar>
              <span class="admin-user-meta">
                <span class="admin-user-name">{{ auth.user?.nickname || auth.user?.username || '管理员' }}</span>
                <span class="admin-user-role">后台管理员</span>
              </span>
              <IconDown class="admin-user-chevron" aria-hidden="true" />
            </button>
            <template #content>
              <div class="admin-user-menu-header">
                <strong>{{ auth.user?.nickname || '管理员' }}</strong>
                <span>{{ auth.user?.email || auth.user?.username || '未绑定邮箱' }}</span>
              </div>
              <a-doption @click="router.push('/setting')">
                <template #icon><IconUser /></template>
                个人中心
              </a-doption>
              <a-doption @click="paletteVisible = true">
                <template #icon><IconSearch /></template>
                快速跳转
              </a-doption>
              <a-doption @click="logout">
                <template #icon><IconPoweroff /></template>
                退出登录
              </a-doption>
            </template>
          </a-dropdown>
        </div>
      </header>

      <main class="admin-content">
        <router-view />
      </main>
    </a-layout>

    <AdminCommandPalette v-model="paletteVisible" :menus="menuStore.visibleMenus" />
  </a-layout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  IconDown,
  IconMenuFold,
  IconMenuUnfold,
  IconMoon,
  IconPoweroff,
  IconRight,
  IconSearch,
  IconSun,
  IconUser
} from '@arco-design/web-vue/es/icon'

import AdminCommandPalette from '@/components/AdminCommandPalette.vue'
import BrandMark from '@/components/BrandMark.vue'
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

const STORAGE_KEY = 'stellar-beacon.admin.sider-collapsed'
const LEGACY_STORAGE_KEY = 'benetnasch.admin.sider-collapsed'
const collapsed = ref(readCollapsed())
const paletteVisible = ref(false)

const avatarText = computed(() => (auth.user?.nickname || auth.user?.username || '管').slice(0, 1).toUpperCase())
const pageTitle = computed(() => String(route.meta.title || '首页'))

/** 依据当前路由在菜单树中回溯，生成真实可点击的面包屑。 */
const breadcrumbs = computed<Array<{ title: string; path: string }>>(() => {
  for (const menu of menuStore.visibleMenus) {
    const children = visibleChildren(menu)
    for (const child of children) {
      if (menuItemPath(menu, child) === route.path) {
        const rootPath = menuItemPath(menu)
        return rootPath === route.path ? [] : [{ title: menu.name, path: rootPath }]
      }
    }
    if (children.length === 0 && menuItemPath(menu) === route.path) return []
  }
  return []
})

watch(collapsed, (value) => {
  try {
    localStorage.setItem(STORAGE_KEY, value ? '1' : '0')
  } catch {
    /* private mode: keep the in-memory preference only */
  }
})

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

function onKeydown(event: KeyboardEvent): void {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    paletteVisible.value = !paletteVisible.value
    return
  }
  if (event.key === 'Escape' && paletteVisible.value) paletteVisible.value = false
}

function navigate(path: string): void {
  void router.push(path)
}

function reloadMenus(): void {
  void menuStore.load(true).catch(() => undefined)
}

async function logout(): Promise<void> {
  await auth.logout()
  resetMenuRoutes()
  menuStore.reset()
  void router.replace({ name: 'login' })
}

function group(menu: NormalizedMenu): boolean {
  return isMenuGroup(menu)
}

function singleWrapper(menu: NormalizedMenu): boolean {
  const children = visibleChildren(menu)
  return children.length === 1 && (children[0].path === '/' || children[0].path === menu.path) && children[0].name === menu.name
}

function readCollapsed(): boolean {
  try {
    let value = localStorage.getItem(STORAGE_KEY)
    if (value === null) {
      value = localStorage.getItem(LEGACY_STORAGE_KEY)
      if (value !== null) localStorage.setItem(STORAGE_KEY, value)
    }
    return value === '1'
  } catch {
    return false
  }
}
</script>
