import type { AdminUser, UserMenu } from '@stellar-beacon/api-contract'

export type { AdminUser, UserMenu }

export interface NormalizedMenu extends UserMenu {
  path: string
  children: NormalizedMenu[]
}

export interface MenuRouteMeta {
  title: string
  icon?: string
  hidden?: boolean
  menuComponent?: string
}

export function normalizeRoutePath(path: string | undefined, fallback = '/'): string {
  const value = (path || '').trim()
  if (!value || value === '/') return fallback
  if (value.endsWith('/*')) {
    const parent = value.slice(0, -2)
    return normalizeRoutePath(`${parent}/:articleId`, fallback)
  }
  const normalized = `/${value.replace(/^\/+/, '').replace(/\/+$/, '')}`
  return /^\/[A-Za-z0-9_:/?.=-]*$/.test(normalized) ? normalized : fallback
}

export function normalizeMenus(menus: UserMenu[] | undefined): NormalizedMenu[] {
  if (!Array.isArray(menus)) return []
  return menus.map((menu) => ({
    ...menu,
    // 后端没给名字时留空：显示层用 menuLabel() 兜底成当前语言的「未命名菜单」，
    // 这里不能写死中文，否则语言切换后不会变。
    name: menu.name || '',
    path: normalizeRoutePath(menu.path),
    children: normalizeMenus(menu.children)
  }))
}

export function menuItemPath(parent: NormalizedMenu, child?: NormalizedMenu): string {
  return normalizeRoutePath(child?.path, parent.path)
}

export function visibleChildren(menu: NormalizedMenu): NormalizedMenu[] {
  return menu.children.filter((child) => !child.hidden)
}

export function isMenuGroup(menu: NormalizedMenu): boolean {
  return menu.children.length > 0 && visibleChildren(menu).length > 0
}
