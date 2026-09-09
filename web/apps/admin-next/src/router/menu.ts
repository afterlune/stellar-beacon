import { defineAsyncComponent, type Component } from 'vue'

import PlaceholderView from '@/views/PlaceholderView.vue'
import type { MenuRouteMeta, NormalizedMenu } from '@/types'
import { menuItemPath } from '@/types'

export interface ResolvedMenuRoute {
  path: string
  name: string
  component: Component
  meta: MenuRouteMeta
}

const componentRegistry: Record<string, Component> = {
  Layout: PlaceholderView,
  '/home/Home.vue': lazyView(() => import('@/views/HomeView.vue')),
  '/article/ArticleList.vue': lazyView(() => import('@/views/ArticleListView.vue')),
  '/article/Article.vue': lazyView(() => import('@/views/ArticleEditorView.vue')),
  '/category/Category.vue': lazyView(() => import('@/views/CategoryView.vue')),
  '/tag/Tag.vue': lazyView(() => import('@/views/TagView.vue')),
  '/comment/Comment.vue': lazyView(() => import('@/views/CommentsView.vue')),
  '/user/User.vue': lazyView(() => import('@/views/UsersView.vue')),
  '/user/Online.vue': lazyView(() => import('@/views/OnlineUsersView.vue')),
  '/role/Role.vue': lazyView(() => import('@/views/RoleView.vue')),
  '/log/OperationLog.vue': lazyView(() => import('@/views/OperationLogsView.vue')),
  '/log/ExceptionLog.vue': lazyView(() => import('@/views/ExceptionLogsView.vue')),
  '/log/QuartzLog.vue': lazyView(() => import('@/views/JobLogsView.vue')),
  '/quartz/Quartz.vue': lazyView(() => import('@/views/JobsView.vue')),
  '/album/Album.vue': lazyView(() => import('@/views/AlbumsView.vue')),
  '/talk/TalkList.vue': lazyView(() => import('@/views/TalksView.vue')),
  '/talk/Talk.vue': lazyView(() => import('@/views/TalkEditorView.vue')),
  '/album/Photo.vue': lazyView(() => import('@/views/PhotoView.vue')),
  '/album/Delete.vue': lazyView(() => import('@/views/PhotoTrashView.vue')),
  '/menu/Menu.vue': lazyView(() => import('@/views/MenuView.vue')),
  '/resource/Resource.vue': lazyView(() => import('@/views/ResourceView.vue')),
  '/friendLink/FriendLink.vue': lazyView(() => import('@/views/FriendLinksView.vue')),
  '/website/Website.vue': lazyView(() => import('@/views/WebsiteView.vue')),
  '/about/About.vue': lazyView(() => import('@/views/AboutView.vue')),
  '/setting/Setting.vue': lazyView(() => import('@/views/SettingView.vue'))
}

function lazyView(loader: () => Promise<{ default: Component }>): Component {
  return defineAsyncComponent(loader)
}

export function resolveMenuComponent(componentPath: string | undefined): Component {
  if (!componentPath) return PlaceholderView
  return componentRegistry[canonicalComponentPath(componentPath)] || PlaceholderView
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
