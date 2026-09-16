import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import AdminLayout from '@/layouts/AdminLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { flattenMenuRoutes } from '@/router/menu'
import ForbiddenView from '@/views/ForbiddenView.vue'
import HomeView from '@/views/WorkplaceView.vue'
import LoginView from '@/views/LoginView.vue'
import NotFoundView from '@/views/NotFoundView.vue'

const routes: RouteRecordRaw[] = [
  // meta.title 保留中文原文：显示层统一由 i18n/menu.ts 的 routeTitle() 按路由名/路径
  // 解析成当前语言，所以这些字面量不会直接渲染出去。
  { path: '/login', name: 'login', component: LoginView, meta: { public: true, title: '管理员登录' } },
  {
    path: '/',
    name: 'admin-shell',
    component: AdminLayout,
    children: [
      { path: '', name: 'admin-home', component: HomeView, meta: { title: '首页' } }
    ]
  },
  { path: '/403', name: 'forbidden', component: ForbiddenView, meta: { title: '无权访问' } },
  { path: '/404', name: 'not-found-page', component: NotFoundView, meta: { title: '页面不存在' } },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView, meta: { title: '页面不存在' } }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

const dynamicRouteNames = new Set<string>()

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  auth.restore()

  if (to.name === 'login') {
    if (!auth.isAuthenticated) return true
    await ensureMenus(auth, useMenuStore())
    return typeof to.query.redirect === 'string' ? safeRedirect(to.query.redirect) : '/'
  }

  if (!auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  try {
    const menu = useMenuStore()
    await ensureMenus(auth, menu)
    if (to.name === 'not-found') {
      const resolved = router.resolve(to.fullPath)
      if (resolved.name && resolved.name !== 'not-found') return resolved.fullPath
    }
    if (typeof to.name === 'string' && to.name.startsWith('admin-menu-') && !hasMenuPath(menu, to.name, to.path)) {
      return { name: 'forbidden' }
    }
    return true
  } catch {
    auth.clear()
    resetMenuRoutes()
    useMenuStore().reset()
    return { name: 'login', query: { redirect: to.fullPath } }
  }
})

async function ensureMenus(auth: ReturnType<typeof useAuthStore>, menu: ReturnType<typeof useMenuStore>): Promise<void> {
  if (!auth.isAuthenticated) return
  const records = await menu.load()
  for (const record of flattenMenuRoutes(records)) {
    if (record.path === '/' || router.hasRoute(record.name)) continue
    router.addRoute('admin-shell', {
      path: record.path.replace(/^\//, ''),
      name: record.name,
      component: record.component,
      meta: record.meta as unknown as Record<string, unknown>
    })
    dynamicRouteNames.add(record.name)
  }
}

export function resetMenuRoutes(): void {
  for (const name of dynamicRouteNames) {
    if (router.hasRoute(name)) router.removeRoute(name)
  }
  dynamicRouteNames.clear()
}

function hasMenuPath(menu: ReturnType<typeof useMenuStore>, name: string, path: string): boolean {
  return flattenMenuRoutes(menu.menus).some((record) => record.name === name || record.path === path)
}

function safeRedirect(value: string): string {
  return value.startsWith('/') && !value.startsWith('//') ? value : '/'
}

export default router
