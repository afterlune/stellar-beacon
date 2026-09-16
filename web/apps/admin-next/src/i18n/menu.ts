/**
 * 菜单名本地化。
 *
 * 菜单树来自后端（`/admin/me/menu`），名字是数据库里的中文，没法直接改数据。
 * 这里按「菜单路径 → nav.* 词条」做反查，查不到就原样返回后端名字：
 *   /content          → nav.content
 *   /photos/delete    → nav.photoTrash（先用完整路径，再退回最后一段）
 *
 * 函数体里读了 locale.value，所以写在 computed / 模板里会随语言切换自动重算。
 */
import { locale, t, te } from './index'

/** 路径段 → 词条名。仅用于「最后一段」与词条名对不上的少数情况。 */
const SEGMENT_ALIASES: Record<string, string> = {
  operation: 'operationLog',
  'operation-log': 'operationLog',
  exception: 'exceptionLog',
  'exception-log': 'exceptionLog',
  'quartz-log': 'quartzLog',
  online: 'onlineUsers',
  'online-users': 'onlineUsers',
  talks: 'talk',
  'talk-list': 'talkList',
  delete: 'photoTrash',
  category: 'categories',
  tag: 'tags'
}

const cache = new Map<string, string>()
let cachedLocale = locale.value

/**
 * 后端菜单名 → 词条名。
 *
 * 菜单名存在数据库里（中文），改不了数据，所以在显示层反查：
 * 路径对得上就用路径，否则用名字本身。名字表让「父级分组」这类
 * 没有稳定路径的节点也能翻译（例如 `内容管理` → nav.content）。
 */
const NAME_KEYS: Record<string, string> = {
  内容管理: 'content',
  文章管理: 'content',
  文章列表: 'articleList',
  发布文章: 'articles',
  新增文章: 'articles',
  分类管理: 'categories',
  标签管理: 'tags',
  评论管理: 'comments',
  消息管理: 'comments',
  用户管理: 'users',
  用户与权限: 'rbac',
  权限管理: 'rbac',
  角色管理: 'roles',
  菜单管理: 'menus',
  资源管理: 'resources',
  接口资源管理: 'resources',
  媒体库: 'media',
  媒体管理: 'media',
  相册管理: 'albums',
  照片管理: 'photos',
  照片回收站: 'photoTrash',
  回收站: 'photoTrash',
  日志与任务: 'ops',
  日志管理: 'ops',
  系统管理: 'ops',
  操作日志: 'operationLog',
  异常日志: 'exceptionLog',
  定时任务: 'quartz',
  任务日志: 'quartzLog',
  个人中心: 'setting',
  友链管理: 'links',
  关于我: 'about',
  网站管理: 'website',
  站点管理: 'website',
  在线用户: 'onlineUsers',
  说说管理: 'talkList',
  发布说说: 'talk',
  仪表盘: 'dashboard',
  工作台: 'workplace',
  监控: 'monitor',
  订阅与增长: 'growth'
}

/**
 * 把后端返回的菜单名翻成当前语言。
 * @param name 后端菜单名（中文）
 * @param path 菜单路径，用于反查词条
 */
export function menuLabel(name: string | undefined, path?: string): string {
  const raw = (name ?? '').trim()
  if (!raw) return t('nav.unknownMenu')

  if (locale.value !== cachedLocale) {
    cache.clear()
    cachedLocale = locale.value
  }

  const cacheKey = `${path ?? ''}\u0000${raw}`
  const hit = cache.get(cacheKey)
  if (hit !== undefined) return hit

  const resolved = resolveMenuLabel(raw, path)
  cache.set(cacheKey, resolved)
  return resolved
}

function resolveMenuLabel(raw: string, path?: string): string {
  for (const candidate of navKeysFor(raw, path)) {
    if (te(candidate)) return t(candidate)
  }
  return raw
}

function navKeysFor(raw: string, path?: string): string[] {
  const keys: string[] = []

  // 1) 后端中文名优先：它是最精确的信号，而末级路径段会被同级菜单共用
  //    （`/online/users` 与 `/users` 的末段都是 `users`，只按路径反查会把
  //    「在线用户」翻成「用户管理」）。
  const byName = NAME_KEYS[raw]
  if (byName) keys.push(`nav.${byName}`)

  // 2) 名字没登记时再退回路径。
  const segments = (path ?? '')
    .split('/')
    .map((segment) => segment.trim())
    .filter((segment) => segment.length > 0 && !segment.startsWith(':'))
  const last = segments[segments.length - 1]
  if (last) {
    keys.push(`nav.${camel(last)}`)
    const aliased = SEGMENT_ALIASES[last]
    if (aliased) keys.push(`nav.${aliased}`)
  }

  return keys
}

function camel(segment: string): string {
  return segment
    .split(/[^A-Za-z0-9]+/)
    .filter(Boolean)
    .map((part, index) => (index === 0 ? part.toLowerCase() : part[0].toUpperCase() + part.slice(1).toLowerCase()))
    .join('')
}

/**
 * 路由 meta.title 的本地化。
 *
 * 后端菜单派生出来的路由名是运行时生成的（`admin-menu-…`），无法在这里穷举，
 * 所以走 `menuLabel()` 按路径反查；只有登录、首页、403、404 这几个固定路由
 * 有稳定名字，用一张小映射表接住。
 */
const ROUTE_TITLE_KEYS: Record<string, string> = {
  login: 'page.login',
  'admin-home': 'page.home',
  forbidden: 'page.forbidden',
  'not-found': 'page.notFound',
  'not-found-page': 'page.notFound'
}

export function routeTitle(routeName: unknown, metaTitle: string | undefined, path: string): string {
  const key = typeof routeName === 'string' ? ROUTE_TITLE_KEYS[routeName] : undefined
  if (key) return t(key)
  return menuLabel(metaTitle, path)
}
