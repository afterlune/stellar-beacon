import type { RouteRecordRaw } from 'vue-router'

import PlaceholderView from '@/views/PlaceholderView.vue'
import type { MenuRouteMeta, NormalizedMenu } from '@/types'
import { menuItemPath } from '@/types'

export interface ResolvedMenuRoute {
  path: string
  name: string
  component: MenuViewComponent
  meta: MenuRouteMeta
}

/** 视图组件：普通组件，或 Vue Router 认可的 `() => import(...)` 懒加载函数。 */
export type MenuViewComponent = NonNullable<RouteRecordRaw['component']>

/**
 * 视图注册表保存的是 **动态 import 函数**（而不是 defineAsyncComponent 包出来的
 * 组件对象）：Vue Router 需要 `() => import(...)` 才能正确切分懒加载边界，
 * 直接给异步组件包装器会在控制台产生 "is defined using defineAsyncComponent()" 警告。
 */
const VIEWS = {
  workplace: () => import('@/views/WorkplaceView.vue'),
  dashboard: () => import('@/views/DashboardView.vue'),
  growth: () => import('@/views/GrowthView.vue'),
  monitor: () => import('@/views/MonitorView.vue'),
  media: () => import('@/views/MediaView.vue'),
  articleList: () => import('@/views/ArticleListView.vue'),
  article: () => import('@/views/ArticleEditorView.vue'),
  category: () => import('@/views/CategoryView.vue'),
  tag: () => import('@/views/TagView.vue'),
  comment: () => import('@/views/CommentsView.vue'),
  user: () => import('@/views/UsersView.vue'),
  onlineUser: () => import('@/views/OnlineUsersView.vue'),
  role: () => import('@/views/RoleView.vue'),
  operationLog: () => import('@/views/OperationLogsView.vue'),
  exceptionLog: () => import('@/views/ExceptionLogsView.vue'),
  quartzLog: () => import('@/views/JobLogsView.vue'),
  quartz: () => import('@/views/JobsView.vue'),
  album: () => import('@/views/AlbumsView.vue'),
  talkList: () => import('@/views/TalksView.vue'),
  talk: () => import('@/views/TalkEditorView.vue'),
  photo: () => import('@/views/PhotoView.vue'),
  photoTrash: () => import('@/views/PhotoTrashView.vue'),
  menu: () => import('@/views/MenuView.vue'),
  resource: () => import('@/views/ResourceView.vue'),
  friendLink: () => import('@/views/FriendLinksView.vue'),
  series: () => import('@/views/SeriesView.vue'),
  website: () => import('@/views/WebsiteView.vue'),
  about: () => import('@/views/AboutView.vue'),
  setting: () => import('@/views/SettingView.vue')
}

const componentRegistry: Record<string, MenuViewComponent> = {
  Layout: PlaceholderView,
  '/home/Home.vue': VIEWS.workplace,
  '/dashboard/Workplace.vue': VIEWS.workplace,
  '/dashboard/Dashboard.vue': VIEWS.dashboard,
  '/growth/Newsletter.vue': VIEWS.growth,
  '/dashboard/Monitor.vue': VIEWS.monitor,
  '/media/Media.vue': VIEWS.media,
  '/article/ArticleList.vue': VIEWS.articleList,
  '/article/Article.vue': VIEWS.article,
  '/category/Category.vue': VIEWS.category,
  '/tag/Tag.vue': VIEWS.tag,
  '/comment/Comment.vue': VIEWS.comment,
  '/user/User.vue': VIEWS.user,
  '/user/Online.vue': VIEWS.onlineUser,
  '/role/Role.vue': VIEWS.role,
  '/log/OperationLog.vue': VIEWS.operationLog,
  '/log/ExceptionLog.vue': VIEWS.exceptionLog,
  '/log/QuartzLog.vue': VIEWS.quartzLog,
  '/quartz/Quartz.vue': VIEWS.quartz,
  '/album/Album.vue': VIEWS.album,
  '/talk/TalkList.vue': VIEWS.talkList,
  '/talk/Talk.vue': VIEWS.talk,
  '/album/Photo.vue': VIEWS.photo,
  '/album/Delete.vue': VIEWS.photoTrash,
  '/menu/Menu.vue': VIEWS.menu,
  '/resource/Resource.vue': VIEWS.resource,
  '/friendLink/FriendLink.vue': VIEWS.friendLink,
  '/website/Website.vue': VIEWS.website,
  '/about/About.vue': VIEWS.about,
  '/setting/Setting.vue': VIEWS.setting
}

export function resolveMenuComponent(componentPath: string | undefined): MenuViewComponent {
  if (!componentPath) return PlaceholderView
  return componentRegistry[canonicalComponentPath(componentPath)] || PlaceholderView
}

/**
 * Whether a backend menu component path maps to a real view.
 *
 * The menu editor uses this to warn before saving a path that would silently
 * fall back to the "under migration" placeholder page.
 */
export function isRegisteredComponent(componentPath: string | undefined): boolean {
  const value = (componentPath || '').trim()
  if (!value) return false
  if (value === 'Layout') return true
  return Boolean(componentRegistry[canonicalComponentPath(value)])
}

/** Registered component paths, for editor autocomplete and validation hints. */
export function registeredComponentPaths(): string[] {
  return Object.keys(componentRegistry).filter((path) => path !== 'Layout').sort()
}

function canonicalComponentPath(componentPath: string): string {
  const value = componentPath.trim()
  if (value === 'Layout') return value
  const withoutViewsPrefix = value.replace(/^\/?views\//, '')
  const withLeadingSlash = withoutViewsPrefix.startsWith('/') ? withoutViewsPrefix : `/${withoutViewsPrefix}`
  return withLeadingSlash.endsWith('.vue') ? withLeadingSlash : `${withLeadingSlash}.vue`
}

export function flattenMenuRoutes(menus: NormalizedMenu[]): ResolvedMenuRoute[] {
  const routes: ResolvedMenuRoute[] = []
  const seen = new Set<string>()

  for (const menu of menus) {
    const children = menu.children.length > 0 ? menu.children : [menu]
    for (const child of children) {
      const path = menuItemPath(menu, menu.children.length > 0 ? child : undefined)
      if (seen.has(path)) continue
      seen.add(path)
      routes.push({
        path,
        name: routeName(path),
        component: resolveMenuComponent(child.component || menu.component),
        meta: {
          // 后端菜单名原样存进 meta（中文），显示时经 i18n/menu.ts 的 menuLabel() 反查词条，
          // 这样语言切换能实时生效。
          title: child.name || menu.name,
          icon: child.icon || menu.icon,
          hidden: Boolean(child.hidden || menu.hidden),
          menuComponent: child.component || menu.component
        }
      })
    }
  }
  return routes
}

export function routeName(path: string): string {
  const value = path.replace(/[^A-Za-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
  return `admin-menu-${value || 'home'}`
}
