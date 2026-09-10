import type { Component } from 'vue'
import {
  IconApps,
  IconBook,
  IconCalendarClock,
  IconCode,
  IconDashboard,
  IconFile,
  IconFolder,
  IconHistory,
  IconImage,
  IconLink,
  IconLock,
  IconMenu,
  IconMessage,
  IconSafe,
  IconSettings,
  IconTags,
  IconUser,
  IconUserGroup
} from '@arco-design/web-vue/es/icon'

export interface MenuLike {
  path?: string
  name?: string
  icon?: string
}

/**
 * 根据菜单的 path/name/icon 推断语义图标。
 * 关键词按「精确语义优先」排序，避免「资源」同时命中「权限」与「接口」分支。
 */
export function menuIconFor(menu: MenuLike): Component {
  const key = `${menu.path || ''} ${menu.name || ''} ${menu.icon || ''}`.toLowerCase()
  if (key.includes('dashboard') || key.includes('workplace') || key.includes('工作台') || key.includes('仪表盘')) return IconDashboard
  if (key.includes('monitor') || key.includes('监控')) return IconDashboard
  if (key.includes('media') || key.includes('图片资源')) return IconImage
  if (key.includes('article') || key.includes('文章') || key.includes('content') || key.includes('内容')) return IconBook
  if (key.includes('category') || key.includes('分类')) return IconFolder
  if (key.includes('tag') || key.includes('标签')) return IconTags
  if (key.includes('comment') || key.includes('评论')) return IconMessage
  if (key.includes('talk') || key.includes('说说')) return IconMessage
  if (key.includes('album') || key.includes('photo') || key.includes('相册') || key.includes('照片')) return IconImage
  if (key.includes('resource') || key.includes('接口') || key.includes('资源')) return IconCode
  if (key.includes('role') || key.includes('permission') || key.includes('权限')) return IconLock
  if (key.includes('online') || key.includes('在线')) return IconUser
  if (key.includes('user') || key.includes('用户')) return IconUserGroup
  if (key.includes('menu') || key.includes('菜单')) return IconMenu
  if (key.includes('log') || key.includes('日志')) return IconHistory
  if (key.includes('quartz') || key.includes('job') || key.includes('任务')) return IconCalendarClock
  if (key.includes('link') || key.includes('友链')) return IconLink
  if (key.includes('website') || key.includes('站点') || key.includes('网站')) return IconDashboard
  if (key.includes('about') || key.includes('关于')) return IconFile
  if (key.includes('setting') || key.includes('设置') || key.includes('个人')) return IconSettings
  if (key.includes('safe') || key.includes('安全')) return IconSafe
  return IconApps
}
