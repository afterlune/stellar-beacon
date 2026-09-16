import type { Component } from 'vue'
import {
  IconApps,
  IconBook,
  IconBug,
  IconCamera,
  IconClockCircle,
  IconCode,
  IconDashboard,
  IconDelete,
  IconFile,
  IconFileImage,
  IconFolder,
  IconHistory,
  IconHome,
  IconIdcard,
  IconImage,
  IconInfoCircle,
  IconLink,
  IconLiveBroadcast,
  IconLock,
  IconMenu,
  IconMessage,
  IconNotification,
  IconPen,
  IconPublic,
  IconSafe,
  IconSchedule,
  IconSettings,
  IconTags,
  IconUnorderedList,
  IconUser,
  IconUserGroup,
  IconWifi
} from '@arco-design/web-vue/es/icon'

export interface MenuLike {
  path?: string
  name?: string
  icon?: string
}

/**
 * 菜单图标注册表。
 *
 * 后端菜单的 `icon` 字段是旧版 Element 图标字体名（`el-icon-myxxx`），
 * 在 Arco 下没有意义，因此这里以 **路由路径** 为准做显式映射：
 * 分组名（消息管理 / 系统管理 / 日志管理…）不会再因为
 * “submenu” 里含 “menu” 之类的巧合而退化成同一个图标。
 */
const PATH_ICONS: Record<string, Component> = {
  '/': IconHome,
  '/dashboard': IconDashboard,
	'/growth': IconNotification,

  '/article-submenu': IconBook,
  '/articles': IconPen,
  '/articles/*': IconPen,
  '/article-list': IconUnorderedList,
  '/categories': IconFolder,
  '/tags': IconTags,

  '/talk-submenu': IconMessage,
  '/talk-list': IconUnorderedList,
  '/talks': IconPen,
  '/talks/*': IconPen,

  '/monitor': IconLiveBroadcast,

  '/message-submenu': IconNotification,
  '/comments': IconMessage,

  '/users-submenu': IconUserGroup,
  '/users': IconUser,
  '/online/users': IconWifi,

  '/permission-submenu': IconLock,
  '/roles': IconSafe,
  '/menus': IconMenu,
  '/resources': IconCode,

  '/media': IconImage,

  '/system-submenu': IconSettings,
  '/website': IconPublic,
  '/quartz': IconSchedule,
  '/links': IconLink,
  '/about': IconInfoCircle,

  '/album-submenu': IconCamera,
  '/albums': IconUnorderedList,
  '/photos/delete': IconDelete,

  '/log-submenu': IconHistory,
  '/exception/log': IconBug,
  '/operation/log': IconFile,

  '/setting': IconIdcard,
  '/profile': IconIdcard
}

/** 末级资源页（带 id 的详情/编辑路由）。 */
const PATTERN_ICONS: Array<[RegExp, Component]> = [
  [/^\/articles\/[^/]+$/, IconPen],
  [/^\/talks\/[^/]+$/, IconPen],
  [/^\/albums\/[^/]+$/, IconFileImage],
  [/^\/quartz\/log\/[^/]+$/, IconClockCircle],
  [/^\/users\/[^/]+$/, IconUser]
]

/**
 * 未知路由的兜底：只对菜单名与去掉 `-submenu` 后的路径做关键词匹配。
 */
const KEYWORDS: Array<[string[], Component]> = [
  [['工作台', '首页', 'workplace', 'home'], IconHome],
  [['仪表', '概览', 'dashboard', 'overview'], IconDashboard],
  [['监控', 'monitor'], IconLiveBroadcast],
  [['异常', 'exception', 'error', '错误'], IconBug],
  [['文章', 'article', 'content', '内容'], IconBook],
  [['分类', 'category'], IconFolder],
  [['标签', 'tag'], IconTags],
  [['说说', 'talk'], IconMessage],
  [['评论', 'comment'], IconMessage],
  [['消息', 'message', 'notification', '通知'], IconNotification],
  [['相册', 'album'], IconCamera],
  [['照片', 'photo'], IconFileImage],
  [['图片', 'image', 'media', '媒体'], IconImage],
  [['在线', 'online'], IconWifi],
  [['角色', 'role'], IconSafe],
  [['权限', 'permission'], IconLock],
  [['接口', '资源', 'resource', 'api'], IconCode],
  [['用户', 'user'], IconUserGroup],
  [['日志', 'log'], IconHistory],
  [['任务', 'job', 'quartz', 'schedule'], IconSchedule],
  [['友链', 'link'], IconLink],
  [['网站', '站点', 'site', 'website'], IconPublic],
  [['关于', 'about'], IconInfoCircle],
  [['菜单', 'menu'], IconMenu],
  [['设置', 'setting', 'config', '配置'], IconSettings],
  [['个人', 'profile', 'account'], IconIdcard]
]

/** 去掉聚合用的 `-submenu` 后缀，避免它污染关键词匹配。 */
function normalizedPath(menu: MenuLike): string {
  return (menu.path || '').trim().toLowerCase().replace(/-submenu$/, '')
}

export function menuIconFor(menu: MenuLike): Component {
  const path = (menu.path || '').trim().toLowerCase()
  const exact = PATH_ICONS[path]
  if (exact) return exact

  for (const [pattern, icon] of PATTERN_ICONS) {
    if (pattern.test(path)) return icon
  }

  const haystack = `${(menu.name || '').toLowerCase()} ${normalizedPath(menu)}`
  for (const [keywords, icon] of KEYWORDS) {
    if (keywords.some((keyword) => haystack.includes(keyword))) return icon
  }
  return IconApps
}
