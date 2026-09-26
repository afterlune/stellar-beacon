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
          <span class="admin-brand-name">{{ t('shell.brandName') }}</span>
          <span class="admin-brand-tagline">{{ t('shell.brandTagline') }}</span>
        </span>
      </div>

      <nav class="admin-nav" :aria-label="t('shell.mainNav')">
        <div v-if="menuStore.loading" class="admin-menu-state">{{ t('shell.menuLoading') }}</div>
        <div v-else-if="menuStore.error" class="admin-menu-state">
          <strong>{{ t('shell.menuFailed') }}</strong>
          <p>{{ t('shell.menuFailedHint') }}</p>
          <a-button size="small" long @click="reloadMenus">{{ t('shell.reload') }}</a-button>
        </div>
        <div v-else-if="menuStore.visibleMenus.length === 0" class="admin-menu-state">
          <strong>{{ t('shell.menuEmpty') }}</strong>
          <p>{{ t('shell.menuEmptyHint') }}</p>
        </div>
        <template v-else>
          <div class="admin-nav-label">{{ t('shell.workspace') }}</div>
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
                <span class="admin-menu-text">{{ menuLabel(menu.name, menuItemPath(menu, visibleChildren(menu)[0])) }}</span>
              </a-menu-item>
              <a-sub-menu v-else-if="group(menu)" :key="menu.path">
                <template #title>
                  <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
                  <span class="admin-menu-text">{{ menuLabel(menu.name, menu.path) }}</span>
                </template>
                <a-menu-item v-for="child in visibleChildren(menu)" :key="menuItemPath(menu, child)">
                  <template #icon>
                    <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(child)" /></span>
                  </template>
                  <span class="admin-menu-text">{{ menuLabel(child.name, menuItemPath(menu, child)) }}</span>
                </a-menu-item>
              </a-sub-menu>
              <a-menu-item v-else :key="menuItemPath(menu)">
                <template #icon>
                  <span class="admin-menu-icon" aria-hidden="true"><component :is="menuIconFor(menu)" /></span>
                </template>
                <span class="admin-menu-text">{{ menuLabel(menu.name, menuItemPath(menu)) }}</span>
              </a-menu-item>
            </template>
          </a-menu>
        </template>
      </nav>

      <div class="admin-nav-footer">
        <span class="admin-nav-user">
          <span class="admin-nav-avatar" aria-hidden="true">{{ avatarText }}</span>
          <span class="admin-nav-user-copy">
            <strong>{{ auth.user?.nickname || auth.user?.username || t('shell.admin') }}</strong>
            <span>{{ t('shell.rbacNote') }}</span>
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
            :aria-label="collapsed ? t('shell.expandSider') : t('shell.collapseSider')"
            :aria-expanded="!collapsed"
            @click="collapsed = !collapsed">
            <IconMenuUnfold v-if="collapsed" />
            <IconMenuFold v-else />
          </button>
          <div class="admin-topbar-context">
            <nav class="admin-breadcrumb" :aria-label="t('shell.breadcrumb')">
              <button class="admin-breadcrumb-link" type="button" @click="router.push('/')">{{ t('shell.console') }}</button>
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
          <button class="admin-search-trigger" type="button" @click="paletteVisible = true">
            <IconSearch aria-hidden="true" />
            <span>{{ t('shell.quickJump') }}</span>
            <kbd>Ctrl K</kbd>
          </button>
          <AdminShellControls />
          <a-dropdown trigger="click">
            <button class="admin-user" type="button" >
              <a-avatar :size="32" :image-url="auth.user?.avatar">{{ avatarText }}</a-avatar>
              <span class="admin-user-meta">
                <span class="admin-user-name">{{ auth.user?.nickname || auth.user?.username || t('shell.admin') }}</span>
                <span class="admin-user-role">{{ t('shell.adminRole') }}</span>
              </span>
              <IconDown class="admin-user-chevron" aria-hidden="true" /><span class="admin-sr-only">{{ t('shell.accountMenu') }}</span>
            </button>
            <template #content>
              <div class="admin-user-menu-header">
                <strong>{{ auth.user?.nickname || t('shell.admin') }}</strong>
                <span>{{ auth.user?.email || auth.user?.username || t('shell.noEmail') }}</span>
              </div>
              <a-doption @click="router.push('/setting')">
                <template #icon><IconUser /></template>
                {{ t('nav.setting') }}
              </a-doption>
              <a-doption @click="paletteVisible = true">
                <template #icon><IconSearch /></template>
                {{ t('shell.quickJump') }}
              </a-doption>
              <a-doption @click="logout">
                <template #icon><IconPoweroff /></template>
                {{ t('login.logout') }}
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  IconDown,
  IconMenuFold,
  IconMenuUnfold,
  IconPoweroff,
  IconRight,
  IconSearch,
  IconUser
} from '@arco-design/web-vue/es/icon'

import AdminCommandPalette from '@/components/AdminCommandPalette.vue'
import AdminShellControls from '@/components/AdminShellControls.vue'
import BrandMark from '@/components/BrandMark.vue'
import { t } from '@/i18n'
import { menuLabel, routeTitle } from '@/i18n/menu'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { resetMenuRoutes } from '@/router'
import { isMenuGroup, menuItemPath, visibleChildren, type NormalizedMenu } from '@/types'
import { menuIconFor } from '@/utils/menu-icon'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const menuStore = useMenuStore()

const STORAGE_KEY = 'stellar-beacon.admin.sider-collapsed'
const LEGACY_STORAGE_KEY = 'benetnasch.admin.sider-collapsed'
const collapsed = ref(readCollapsed())
const paletteVisible = ref(false)
let tableSelectionObserver: MutationObserver | undefined

const avatarText = computed(() => (
  auth.user?.nickname || auth.user?.username || t('shell.adminFallback')
).slice(0, 1).toUpperCase())
const pageTitle = computed(() => routeTitle(route.name, route.meta.title ? String(route.meta.title) : undefined, route.path))

/** 依据当前路由在菜单树中回溯，生成真实可点击的面包屑。 */
const breadcrumbs = computed<Array<{ title: string; path: string }>>(() => {
  for (const menu of menuStore.visibleMenus) {
    const children = visibleChildren(menu)
    for (const child of children) {
      if (menuItemPath(menu, child) === route.path) {
        const rootPath = menuItemPath(menu)
        return rootPath === route.path ? [] : [{ title: menuLabel(menu.name, rootPath), path: rootPath }]
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

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  void nextTick(labelTableSelections)
  tableSelectionObserver = new MutationObserver(labelTableSelections)
  tableSelectionObserver.observe(document.body, { childList: true, subtree: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  tableSelectionObserver?.disconnect()
})

watch(() => route.fullPath, () => void nextTick(labelTableSelections))

/**
 * Arco 表格的批量选择框没有可访问名称；在全局补齐，避免每个列表重复实现。
 */
function labelTableSelections(): void {
  document.querySelectorAll<HTMLInputElement>('.arco-table input.arco-checkbox-target').forEach((input, index) => {
    const isHeader = Boolean(input.closest('th'))
    const name = isHeader ? 'table-select-all' : 'table-select-row'
    if (!input.name) input.name = name
    if (!input.id) input.id = `${name}-${index + 1}`
    if (!input.hasAttribute('aria-label')) {
      input.setAttribute('aria-label', isHeader ? t('shell.selectAllRows') : t('shell.selectRow'))
    }
  })
  document.querySelectorAll<HTMLInputElement>('.arco-pagination .arco-select-view-input').forEach((input, index) => {
    if (!input.name) input.name = 'table-page-size'
    if (!input.id) input.id = `table-page-size-${index + 1}`
    if (!input.hasAttribute('aria-label')) input.setAttribute('aria-label', t('shell.pageSize'))
  })
  document.querySelectorAll<HTMLInputElement>('.arco-pagination input.arco-input').forEach((input, index) => {
    if (!input.name) input.name = 'table-page-jump'
    if (!input.id) input.id = `table-page-jump-${index + 1}`
    if (!input.hasAttribute('aria-label')) input.setAttribute('aria-label', t('shell.pageJump'))
  })
  document.querySelectorAll<HTMLElement>('.arco-pagination-list > span.arco-pagination-item').forEach((item) => {
    const listItem = document.createElement('li')
    listItem.style.display = 'contents'
    item.parentNode?.insertBefore(listItem, item)
    listItem.appendChild(item)
  })
}

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
