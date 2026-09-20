/**
 * 词典装配点。
 *
 * 每个业务域一个文件，zh-CN 与 en-US 各一份，键集合必须一一对应。
 * 新增页面时：在两边各加一个同名模块，然后在这里挂上。
 */
import commonZh from './zh-CN/common'
import shellZh from './zh-CN/shell'
import loginZh from './zh-CN/login'
import dashboardZh from './zh-CN/dashboard'
import articlesZh from './zh-CN/articles'
import taxonomyZh from './zh-CN/taxonomy'
import commentsZh from './zh-CN/comments'
import mediaZh from './zh-CN/media'
import rbacZh from './zh-CN/rbac'
import logsZh from './zh-CN/logs'
import siteZh from './zh-CN/site'
import accountZh from './zh-CN/account'
import growthZh from './zh-CN/growth'
import seriesZh from './zh-CN/series'
import contentPerformanceZh from './zh-CN/content-performance'
import moderationZh from './zh-CN/moderation'

import commonEn from './en-US/common'
import shellEn from './en-US/shell'
import loginEn from './en-US/login'
import dashboardEn from './en-US/dashboard'
import articlesEn from './en-US/articles'
import taxonomyEn from './en-US/taxonomy'
import commentsEn from './en-US/comments'
import mediaEn from './en-US/media'
import rbacEn from './en-US/rbac'
import logsEn from './en-US/logs'
import siteEn from './en-US/site'
import accountEn from './en-US/account'
import growthEn from './en-US/growth'
import seriesEn from './en-US/series'
import contentPerformanceEn from './en-US/content-performance'
import moderationEn from './en-US/moderation'

export const messages = {
  'zh-CN': {
    ...commonZh,
    ...shellZh,
    ...loginZh,
    ...dashboardZh,
    ...articlesZh,
    ...taxonomyZh,
    ...commentsZh,
    ...mediaZh,
    ...rbacZh,
    ...logsZh,
    ...siteZh,
    ...accountZh,
    ...growthZh,
    ...seriesZh,
    ...contentPerformanceZh,
    ...moderationZh
  },
  'en-US': {
    ...commonEn,
    ...shellEn,
    ...loginEn,
    ...dashboardEn,
    ...articlesEn,
    ...taxonomyEn,
    ...commentsEn,
    ...mediaEn,
    ...rbacEn,
    ...logsEn,
    ...siteEn,
    ...accountEn,
    ...growthEn,
    ...seriesEn,
    ...contentPerformanceEn,
    ...moderationEn
  }
} as const

export type MessageKey = keyof (typeof messages)['zh-CN']
