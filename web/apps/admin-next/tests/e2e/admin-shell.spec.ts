import { expect, test } from '@playwright/test'

let cleanJobLogsCalled = false
let jobLogQueryJobId = ''
let runJobCalled = false
let photoRestoreCalled = false
let friendLinkSavedCalled = false
let friendLinkDeletedCalled = false
let websiteConfigUpdatedCalled = false
let aboutUpdatedCalled = false
let profileUpdatedCalled = false
let returnEmptyMenus = false
let articleImported = false
let articleImportHasFile = false
let articleExportedIds: number[] = []
let articleExported = false
let photoMovePayload: { photoIds?: number[]; albumId?: number } | null = null
let jobGroupsRequested = false
let jobLogGroupsRequested = false

const defaultWebsiteConfig = {
  name: 'Benetnasch',
  englishName: 'Benetnasch',
  author: '测试作者',
  logo: 'https://example.com/logo.png',
  github: 'https://github.com/example',
  gitee: 'https://gitee.com/example',
  notice: '欢迎来到 Benetnasch'
}
const defaultAboutContent = '关于 Benetnasch 的介绍'
let websiteConfig = { ...defaultWebsiteConfig }
let aboutContent = defaultAboutContent
let profile = { nickname: '测试管理员', intro: '保持公开资料边界', website: 'https://example.com/admin' }

test.beforeEach(async ({ page }) => {
  cleanJobLogsCalled = false
  jobLogQueryJobId = ''
  runJobCalled = false
  photoRestoreCalled = false
  friendLinkSavedCalled = false
  friendLinkDeletedCalled = false
  websiteConfigUpdatedCalled = false
  aboutUpdatedCalled = false
  profileUpdatedCalled = false
  returnEmptyMenus = false
  articleImported = false
  articleImportHasFile = false
  articleExportedIds = []
  articleExported = false
  photoMovePayload = null
  jobGroupsRequested = false
  jobLogGroupsRequested = false
  websiteConfig = { ...defaultWebsiteConfig }
  aboutContent = defaultAboutContent
  profile = { nickname: '测试管理员', intro: '保持公开资料边界', website: 'https://example.com/admin' }
  await page.route('**/*', async (route) => {
    const requestURL = new URL(route.request().url())
    if (!requestURL.pathname.startsWith('/api')) {
      await route.continue()
      return
    }

    if (requestURL.pathname === '/api/v1/auth/login') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          flag: true,
          code: 20000,
          message: '操作成功',
          data: { token: 'e2e-token', userInfoId: 1, username: 'admin@example.com', ...profile }
        })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/me/menu') {
      if (returnEmptyMenus) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: [] })
        })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          flag: true,
          code: 20000,
          message: '操作成功',
          data: [
            {
              name: '内容管理',
              path: '/content',
              component: 'Layout',
              icon: 'iconfont el-icon-menu',
              hidden: false,
              children: [
                {
                  name: '发布文章',
                  path: '/articles',
                  component: '/article/Article.vue',
                  hidden: false
                },
                {
                  name: '修改文章',
                  path: '/articles/*',
                  component: '/article/Article.vue',
                  hidden: true
                },
                {
                  name: '文章列表',
                  path: '/article-list',
                  component: '/article/ArticleList.vue',
                  icon: 'iconfont el-icon-document',
                  hidden: false
                },
                {
                  name: '分类管理',
                  path: '/categories',
                  component: '/category/Category.vue',
                  hidden: false
                },
                {
                  name: '标签管理',
                  path: '/tags',
                  component: '/tag/Tag.vue',
                  hidden: false
                },
                {
                  name: '评论管理',
                  path: '/comments',
                  component: '/comment/Comment.vue',
                  hidden: false
                },
                {
                  name: '用户管理',
                  path: '/users',
                  component: '/user/User.vue',
                  hidden: false
                },
                {
                  name: '角色管理',
                  path: '/roles',
                  component: '/role/Role.vue',
                  hidden: false
                },
                {
                  name: '操作日志',
                  path: '/operation/log',
                  component: '/log/OperationLog.vue',
                  hidden: false
                },
                {
                  name: '异常日志',
                  path: '/exception/log',
                  component: '/log/ExceptionLog.vue',
                  hidden: false
                },
                {
                  name: '任务日志',
                  path: '/quartz/log/:quartzId',
                  component: '/log/QuartzLog.vue',
                  hidden: true
                },
                {
                  name: '定时任务',
                  path: '/quartz',
                  component: '/quartz/Quartz.vue',
                  hidden: false
                },
                {
                  name: '相册管理',
                  path: '/albums',
                  component: '/album/Album.vue',
                  hidden: false
                },
                {
                  name: '照片管理',
                  path: '/albums/*',
                  component: '/album/Photo.vue',
                  hidden: true
                },
                {
                  name: '照片回收站',
                  path: '/photos/delete',
                  component: '/album/Delete.vue',
                  hidden: true
                },
                {
                  name: '说说管理',
                  path: '/talk-list',
                  component: '/talk/TalkList.vue',
                  hidden: false
                },
                {
                  name: '菜单管理',
                  path: '/menus',
                  component: '/menu/Menu.vue',
                  hidden: false
                },
                {
                  name: '资源管理',
                  path: '/resources',
                  component: '/resource/Resource.vue',
                  hidden: false
                },
                {
                  name: '发布说说',
                  path: '/talks',
                  component: '/talk/Talk.vue',
                  hidden: true
                },
                {
                  name: '编辑说说',
                  path: '/talks/*',
                  component: '/talk/Talk.vue',
                  hidden: true
                },
                {
                  name: '隐藏页面',
                  path: '/hidden',
                  component: '/article/Article.vue',
                  hidden: true
                },
              ]
            },
            {
              name: '个人中心',
              path: '/setting',
              component: '/setting/Setting.vue',
              hidden: false
            },
            {
              name: '友链管理',
              path: '/links',
              component: '/friendLink/FriendLink.vue',
              hidden: false
            },
            {
              name: '关于我',
              path: '/about',
              component: '/about/About.vue',
              hidden: false
            },
            {
              name: '网站管理',
              path: '/website',
              component: '/website/Website.vue',
              hidden: false
            },
            {
              name: '在线用户',
              path: '/online/users',
              component: '/user/Online.vue',
              hidden: false
            },
            {
              name: '无可见菜单',
              path: '/hidden-group',
              component: 'Layout',
              hidden: false,
              children: [
                {
                  name: '仅用于权限测试',
                  path: '/hidden-only',
                  component: '/article/Article.vue',
                  hidden: true
                }
              ]
            }
          ]
        })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/site') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: websiteConfig })
        })
      } else if (route.request().method() === 'PUT') {
        const payload = route.request().postDataJSON() as Record<string, unknown>
        websiteConfig = { ...websiteConfig, ...payload } as typeof websiteConfig
        websiteConfigUpdatedCalled = true
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      } else {
        await route.fulfill({ status: 405, contentType: 'application/json', body: JSON.stringify({ flag: false, code: 40500, message: '不支持的请求方法', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/public/about') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { content: aboutContent } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/about') {
      if (route.request().method() === 'PUT') {
        const payload = route.request().postDataJSON() as { content?: unknown }
        if (typeof payload.content === 'string') aboutContent = payload.content
        aboutUpdatedCalled = true
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      return
    }

    if (requestURL.pathname === '/api/v1/auth/me') {
      if (route.request().method() === 'PUT') {
        const payload = route.request().postDataJSON() as Partial<typeof profile>
        profile = { ...profile, ...payload }
        profileUpdatedCalled = true
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/articles') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          flag: true,
          code: 20000,
          message: '操作成功',
          data: { records: [{ id: 7, articleTitle: '传统后台路线', categoryName: '工程化', status: 1, viewsCount: 3, createTime: '2026-08-29T10:00:00Z' }], count: 1 }
        })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/articles/42') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { id: 42, articleTitle: '已存在文章', articleContent: '正文', categoryName: '工程化', tagNames: ['工程化'], status: 1, type: 1, isTop: 1, isFeatured: 0 } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/articles/import') {
      // 导入走 multipart，服务端读的是 `file` 字段。
      articleImported = true
      articleImportHasFile = (route.request().postData() || '').includes('name="file"')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/articles/export') {
      // 导出接口绑定的是裸的 id 数组，而不是对象。
      articleExportedIds = (route.request().postDataJSON() as number[] | null) || []
      articleExported = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          flag: true,
          code: 20000,
          message: '操作成功',
          data: ['https://example.com/export/传统后台路线.md']
        })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/photos/album') {
      photoMovePayload = route.request().postDataJSON() as { photoIds?: number[]; albumId?: number }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/albums/options') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          flag: true,
          code: 20000,
          message: '操作成功',
          data: [{ id: 5, albumName: '项目截图' }, { id: 6, albumName: '旅行相册' }]
        })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/jobs/groups') {
      jobGroupsRequested = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: ['默认', '清理'] })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/logs/jobs/groups') {
      // 该接口当前把分组序列化成一个字符串，前端必须兼容这种形状。
      jobLogGroupsRequested = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: '默认,清理' })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/categories' || requestURL.pathname === '/api/v1/admin/tags') {
      if (route.request().method() === 'GET') {
        const isCategory = requestURL.pathname.endsWith('categories')
        const row = isCategory
          ? { id: 1, categoryName: '工程化', articleCount: 2, createTime: '2026-08-29T10:00:00Z' }
          : { id: 2, tagName: '工程化', articleCount: 1, createTime: '2026-08-29T10:00:00Z' }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [row], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/comments') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 3, nickname: '访客', articleTitle: '传统后台路线', commentContent: '不错', isReview: 0, createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/users/roles') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: [{ id: 2, roleName: '编辑者' }] })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/users') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ userInfoId: 9, nickname: '测试用户', loginType: 1, roles: [{ id: 2, roleName: '编辑者' }], isDisable: 0, lastLoginTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/roles') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 2, roleName: '编辑者', menuIds: [1], resourceIds: [2], createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/logs/operations') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 31, optModule: '文章模块', optType: '新增或修改', optUri: '/admin/articles', requestMethod: 'POST', nickname: '测试管理员', createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/logs/exceptions') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 32, optUri: '/admin/articles', requestMethod: 'POST', optDesc: '保存文章', exceptionInfo: '模拟异常', createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/logs/jobs') {
      if (route.request().method() === 'GET') {
        jobLogQueryJobId = requestURL.searchParams.get('jobId') || ''
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 33, jobName: '定时任务', jobGroup: '默认', invokeTarget: 'article.cleanup', status: 1, startTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/logs/jobs/clean') {
      cleanJobLogsCalled = true
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/jobs/85') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { id: 85, jobName: '定时任务', jobGroup: '默认', invokeTarget: 'article.cleanup', cronExpression: '0 0 * * * ?', misfirePolicy: '2', concurrent: 0, status: 1, remark: '夜间执行' } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/jobs/status') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/jobs/run') {
      runJobCalled = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '任务已执行一次', data: { jobId: 85, target: 'article.cleanup', processed: true } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/jobs') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 85, jobName: '定时任务', jobGroup: '默认', invokeTarget: 'article.cleanup', cronExpression: '0 0 * * * ?', misfirePolicy: 2, concurrent: 0, status: 1, remark: '夜间执行', canRunOnce: true, createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/roles/menu-options' || requestURL.pathname === '/api/v1/admin/roles/resource-options') {
      const isMenu = requestURL.pathname.endsWith('/menu-options')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: [{ id: isMenu ? 1 : 2, label: isMenu ? '内容管理' : '文章资源', children: [{ id: isMenu ? 11 : 22, label: isMenu ? '文章列表' : '读取文章' }] }] })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/albums') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 5, albumName: '项目截图', albumDesc: '联调素材', albumCover: 'album-cover', photoCount: 2, status: 1 }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/albums/5') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { id: 5, albumName: '项目截图', albumDesc: '联调素材', albumCover: 'album-cover', photoCount: 1, status: 1 } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/photos/trash') {
      if (route.request().method() === 'PUT') {
        const body = route.request().postDataJSON() as { isDelete?: number }
        if (body.isDelete === 0) photoRestoreCalled = true
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/photos') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 13, photoName: '首页截图', photoDesc: '测试图片', photoSrc: 'photo-key' }], count: 1 } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/talks') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 7, nickname: '测试管理员', content: '一次完整的前后端联调', imgs: [], isTop: 0, status: 1, commentCount: 1, createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/talks/7') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { id: 7, content: '一次完整的前后端联调', images: '[]', imgs: [], isTop: 0, status: 1 } })
      })
      return
    }

    if (requestURL.pathname === '/api/v1/admin/friend-links') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: { records: [{ id: 6, linkName: '项目友链', linkAvatar: 'https://example.com/avatar.png', linkAddress: 'https://example.com', linkIntro: '工程化参考', createTime: '2026-08-29T10:00:00Z' }], count: 1 } })
        })
      } else if (route.request().method() === 'POST') {
        friendLinkSavedCalled = true
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      } else if (route.request().method() === 'DELETE') {
        friendLinkDeletedCalled = true
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      } else {
        await route.fulfill({ status: 405, contentType: 'application/json', body: JSON.stringify({ flag: false, code: 40500, message: '不支持的请求方法', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/menus') {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: [{ id: 1, name: '内容管理', path: '/content', component: 'Layout', orderNum: 1, parentId: 0, isHidden: 0, children: [{ id: 11, name: '文章列表', path: '/article-list', component: '/article/ArticleList.vue', orderNum: 1, parentId: 1, isHidden: 0 }] }] })
        })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null }) })
      }
      return
    }

    if (requestURL.pathname === '/api/v1/admin/permissions') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: [{ id: 2, resourceName: '文章读取', url: '/admin/articles', requestMethod: 'GET', parentId: 0, isAnonymous: 0, children: [] }] })
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ flag: true, code: 20000, message: '操作成功', data: null })
    })
  })
})

test('logs in, installs backend menu routes, and avoids blank pages', async ({ page }) => {
  // This is an exhaustive write-capable mock flow across every migrated
  // module. Its route loop is intentionally longer than the normal per-test
  // budget; each navigation still has its own assertion timeout.
  test.setTimeout(120_000)
  const pageErrors = capturePageErrors(page)
  await page.goto('/login')
  await page.getByTestId('login-username').locator('input').fill('admin@example.com')
  await page.getByTestId('login-password').locator('input').fill('password')
  await page.getByTestId('login-submit').click()

  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('main').getByRole('button', { name: '发布文章', exact: true })).toBeVisible()
  await expect(page.locator('.admin-sider').getByText('内容管理', { exact: true })).toBeVisible()
  await expect(page.getByText('无可见菜单', { exact: true })).toHaveCount(0)
  await page.locator('.admin-sider').getByText('文章列表', { exact: true }).click()
  await expect(page).toHaveURL(/\/article-list$/)
  await expect(page.getByRole('main').getByText('文章列表')).toBeVisible()
  await page.locator('.admin-sider').getByText('发布文章', { exact: true }).click()
  await expect(page).toHaveURL(/\/articles$/)
  await expect(page.getByRole('main').getByText('发布文章')).toBeVisible()
  await page.goto('/articles/42')
  await expect(page).toHaveURL(/\/articles\/42$/)
  await expect(page.getByRole('main').getByText('修改文章')).toBeVisible()
  await expect(page.locator('input').first()).toHaveValue('已存在文章')
  await page.getByText('分类管理').click()
  await expect(page).toHaveURL(/\/categories$/)
  await expect(page.getByRole('main').getByText('工程化')).toBeVisible()
  await page.getByRole('button', { name: '新增' }).click()
  const dialog = page.locator('.arco-modal:visible')
  await dialog.locator('input').fill('新增分类')
  await dialog.getByRole('button', { name: '确定' }).click()
  await page.getByText('标签管理').click()
  await expect(page).toHaveURL(/\/tags$/)
  await expect(page.getByRole('main').getByText('工程化')).toBeVisible()
  await page.getByRole('button', { name: '编辑' }).click()
  const tagEditDialog = page.locator('.arco-modal:visible')
  await expect(tagEditDialog).toContainText('编辑')
  await tagEditDialog.locator('input').fill('工程化更新')
  await tagEditDialog.getByRole('button', { name: '确定' }).click()
  await page.getByRole('button', { name: '新增' }).click()
  const tagCreateDialog = page.locator('.arco-modal:visible')
  await tagCreateDialog.locator('input').fill('新标签')
  await tagCreateDialog.getByRole('button', { name: '确定' }).click()
  await page.reload()
  await expect(page).toHaveURL(/\/tags$/)
  await expect(page.getByRole('main').getByText('标签管理')).toBeVisible()
  await expect(page.getByRole('main').getByText('工程化')).toBeVisible()
  await page.getByText('评论管理').click()
  await expect(page).toHaveURL(/\/comments$/)
  await expect(page.getByText('不错')).toBeVisible()
  await expect(page.getByRole('button', { name: '通过审核' })).toBeVisible()
  await page.getByText('用户管理').click()
  await expect(page).toHaveURL(/\/users$/)
  await expect(page.getByText('测试用户')).toBeVisible()
  await page.getByRole('button', { name: '编辑' }).click()
  await expect(page.locator('.arco-modal:visible')).toContainText('修改用户')
  await page.getByRole('button', { name: '确定' }).click()
  await page.getByText('角色管理').click()
  await expect(page).toHaveURL(/\/roles$/)
  await expect(page.getByText('编辑者')).toBeVisible()
  await page.getByRole('button', { name: '新增' }).click()
  const roleDialog = page.locator('.arco-modal:visible')
  await roleDialog.locator('input[type="text"]').first().fill('审核员')
  await roleDialog.getByRole('button', { name: '确定' }).click()
  await page.getByText('定时任务').click()
  await expect(page).toHaveURL(/\/quartz$/)
  await expect(page.getByRole('main').getByRole('heading', { name: '定时任务' })).toBeVisible()
  await expect(page.getByRole('button', { name: '执行一次' })).toBeEnabled()
  await page.getByRole('button', { name: '执行一次' }).click()
  await page.locator('.arco-popconfirm:visible').getByRole('button', { name: '确定' }).click()
  await expect.poll(() => runJobCalled).toBe(true)
  await page.getByRole('button', { name: '新增' }).click()
  const jobDialog = page.locator('.arco-modal:visible')
  const jobInputs = jobDialog.locator('input[type="text"]')
  await jobInputs.nth(0).fill('新任务')
  await jobInputs.nth(1).fill('默认')
  await jobInputs.nth(2).fill('article.cleanup')
  await jobInputs.nth(3).fill('0 0 * * * ?')
  await jobDialog.getByRole('button', { name: '确定' }).click()
  await page.getByRole('button', { name: '编辑' }).click()
  await expect(page.locator('.arco-modal:visible')).toContainText('编辑任务')
  await page.locator('.arco-modal:visible').getByRole('button', { name: '确定' }).click()
  await page.getByText('操作日志').click()
  await expect(page).toHaveURL(/\/operation\/log$/)
  await expect(page.getByText('文章模块')).toBeVisible()
  await page.getByRole('button', { name: '详情' }).click()
  await expect(page.locator('.arco-modal:visible')).toContainText('新增或修改')
  await page.keyboard.press('Escape')
  await expect(page.locator('.arco-modal:visible')).toHaveCount(0)
  await page.getByText('异常日志').click()
  await expect(page).toHaveURL(/\/exception\/log$/)
  await expect(page.getByText('模拟异常')).toBeVisible()
  await page.goto('/quartz/log/85')
  await expect(page).toHaveURL(/\/quartz\/log\/\d+$/)
  await expect(page.getByRole('main').getByText('定时任务', { exact: true })).toBeVisible()
  await expect.poll(() => jobLogQueryJobId).toBe('85')
  await page.getByRole('button', { name: '清空任务日志' }).click()
  await page.locator('.arco-popconfirm:visible').getByRole('button', { name: '确定' }).click()
  await expect.poll(() => cleanJobLogsCalled).toBe(true)
  await page.getByText('相册管理').click()
  await expect(page).toHaveURL(/\/albums$/)
  await expect(page.getByText('项目截图')).toBeVisible()
  await page.getByRole('button', { name: '新增' }).click()
  const albumDialog = page.locator('.arco-modal:visible')
  const albumTextInputs = albumDialog.locator('input[type="text"]')
  await albumTextInputs.nth(0).fill('新相册')
  await albumDialog.locator('textarea').fill('新相册描述')
  await albumTextInputs.nth(1).fill('new-album-cover')
  await albumDialog.getByRole('button', { name: '确定' }).click()
  await page.getByRole('button', { name: '回收站' }).click()
  await expect(page).toHaveURL(/\/photos\/delete$/)
  await expect(page.getByRole('main').getByText('照片回收站')).toBeVisible()
  await expect(page.getByText('首页截图')).toBeVisible()
  await page.getByRole('row', { name: /首页截图/ }).getByRole('checkbox').locator('..').click()
  await expect(page.getByRole('button', { name: '批量恢复' })).toBeEnabled()
  await page.getByRole('button', { name: '批量恢复' }).click()
  await expect.poll(() => photoRestoreCalled).toBe(true)
  await page.reload()
  await expect(page.getByRole('main').getByText('照片回收站')).toBeVisible()
  await page.goto('/albums/5')
  await expect(page).toHaveURL(/\/albums\/5$/)
  await expect(page.getByRole('main').getByText('项目截图 · 照片')).toBeVisible()
  await expect(page.getByText('首页截图')).toBeVisible()
  await page.getByText('说说管理').click()
  await expect(page).toHaveURL(/\/talk-list$/)
  await expect(page.getByText('一次完整的前后端联调')).toBeVisible()
  await page.getByRole('button', { name: '编辑' }).click()
  await expect(page).toHaveURL(/\/talks\/7$/)
  await expect(page.getByRole('main').getByText('编辑说说')).toBeVisible()
  await page.getByText('菜单管理').click()
  await expect(page).toHaveURL(/\/menus$/)
  await expect(page.getByRole('table').getByText('文章列表', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '新增' }).click()
  const menuDialog = page.locator('.arco-modal:visible')
  await menuDialog.locator('input[type="text"]').nth(0).fill('新菜单')
  await menuDialog.locator('input[type="text"]').nth(1).fill('/new-menu')
  await menuDialog.locator('input[type="text"]').nth(2).fill('/article/ArticleList.vue')
  await menuDialog.getByRole('button', { name: '确定' }).click()
  await page.getByText('资源管理').click()
  await expect(page).toHaveURL(/\/resources$/)
  await expect(page.locator('.arco-table').getByText('文章读取', { exact: true })).toBeVisible()
  await page.getByText('在线用户').click()
  await expect(page).toHaveURL(/\/online\/users$/)
  await expect(page.getByRole('main').getByText('在线用户')).toBeVisible()
  await page.getByText('友链管理').click()
  await expect(page).toHaveURL(/\/links$/)
  await expect(page.getByRole('main').getByText('友链管理')).toBeVisible()
  await expect(page.getByText('项目友链', { exact: true })).toBeVisible()
  const friendLinkRow = page.getByRole('row', { name: /项目友链/ })
  await friendLinkRow.getByRole('button', { name: '编辑' }).click()
  const friendLinkEditDialog = page.locator('.arco-modal:visible')
  await expect(friendLinkEditDialog).toContainText('编辑友链')
  await expect(friendLinkEditDialog.locator('input').nth(0)).toHaveValue('项目友链')
  await friendLinkEditDialog.getByRole('button', { name: '确定' }).click()
  await expect.poll(() => friendLinkSavedCalled).toBe(true)
  friendLinkSavedCalled = false
  await page.getByRole('button', { name: '新增', exact: true }).click()
  const friendLinkCreateDialog = page.locator('.arco-modal:visible')
  const friendLinkInputs = friendLinkCreateDialog.locator('input')
  await friendLinkInputs.nth(0).fill('新友链')
  await friendLinkInputs.nth(1).fill('https://example.com/new-avatar.png')
  await friendLinkInputs.nth(2).fill('https://example.com/new')
  await friendLinkCreateDialog.locator('textarea').fill('新友链介绍')
  await friendLinkCreateDialog.getByRole('button', { name: '确定' }).click()
  await expect.poll(() => friendLinkSavedCalled).toBe(true)
  await page.reload()
  await expect(page).toHaveURL(/\/links$/)
  await expect(page.getByRole('main').getByText('友链管理')).toBeVisible()
  await friendLinkRow.getByRole('button', { name: '删除' }).click()
  await page.locator('.arco-popconfirm:visible').getByRole('button', { name: '确定' }).click()
  await expect.poll(() => friendLinkDeletedCalled).toBe(true)
  await page.getByText('关于我').click()
  await expect(page).toHaveURL(/\/about$/)
  const aboutMain = page.getByRole('main')
  await expect(aboutMain.getByText('关于我')).toBeVisible()
  const aboutTextarea = aboutMain.locator('textarea')
  await expect(aboutTextarea).toHaveValue(defaultAboutContent)
  await aboutTextarea.fill('关于 Benetnasch 的 E2E 更新')
  await aboutMain.getByRole('button', { name: '保存', exact: true }).click()
  await expect.poll(() => aboutUpdatedCalled).toBe(true)
  aboutUpdatedCalled = false
  await page.reload()
  await expect(aboutMain.locator('textarea')).toHaveValue('关于 Benetnasch 的 E2E 更新')

  await page.getByText('网站管理').click()
  await expect(page).toHaveURL(/\/website$/)
  const websiteMain = page.getByRole('main')
  await expect(websiteMain.getByText('网站配置')).toBeVisible()
  await expect(websiteMain.locator('input').nth(0)).toHaveValue('Benetnasch')
  await expect(websiteMain.locator('textarea')).toHaveValue('欢迎来到 Benetnasch')
  await websiteMain.locator('input').nth(0).fill('Benetnasch E2E')
  await websiteMain.getByRole('button', { name: '保存', exact: true }).click()
  await expect.poll(() => websiteConfigUpdatedCalled).toBe(true)
  websiteConfigUpdatedCalled = false
  await page.reload()
  await expect(websiteMain.locator('input').nth(0)).toHaveValue('Benetnasch E2E')

  await page.getByText('个人中心').click()
  await expect(page).toHaveURL(/\/setting$/)
  const settingMain = page.getByRole('main')
  const profileInputs = settingMain.locator('input:not([type="file"])')
  await expect(settingMain.getByText('个人中心')).toBeVisible()
  await expect(profileInputs.nth(0)).toHaveValue('测试管理员')
  await expect(settingMain.locator('textarea')).toHaveValue('保持公开资料边界')
  await expect(profileInputs.nth(1)).toHaveValue('https://example.com/admin')
  await profileInputs.nth(0).fill('测试管理员 E2E')
  await settingMain.locator('textarea').fill('个人中心刷新后仍保留')
  await profileInputs.nth(1).fill('https://example.com/admin-e2e')
  await settingMain.getByRole('button', { name: '保存资料', exact: true }).click()
  await expect.poll(() => profileUpdatedCalled).toBe(true)
  // 等待前端把更新写回 sessionStorage 后再 reload，避免与 mock 响应之间的竞态
  await expect.poll(() => page.evaluate(() => {
    const raw = sessionStorage.getItem('benetnasch.admin.user')
    if (!raw) return false
    try { return (JSON.parse(raw) as { nickname?: string }).nickname === '测试管理员 E2E' } catch { return false }
  })).toBe(true)
  await page.reload()
  await expect(page).toHaveURL(/\/setting$/)
  await expect(page.getByRole('main').locator('input:not([type="file"])').nth(0)).toHaveValue('测试管理员 E2E')
  await expect(page.getByRole('main').locator('textarea')).toHaveValue('个人中心刷新后仍保留')
  await expect(page.getByRole('main').locator('input:not([type="file"])').nth(1)).toHaveValue('https://example.com/admin-e2e')
  expect(pageErrors()).toEqual([])
})

test('imports and exports articles, moves photos, and keeps filters in the URL', async ({ page }) => {
  test.setTimeout(120_000)
  const pageErrors = capturePageErrors(page)
  await page.goto('/login')
  await page.getByTestId('login-username').locator('input').fill('admin@example.com')
  await page.getByTestId('login-password').locator('input').fill('password')
  await page.getByTestId('login-submit').click()
  await expect(page).toHaveURL(/\/$/)

  await page.locator('.admin-sider').getByText('文章列表', { exact: true }).click()
  await expect(page).toHaveURL(/\/article-list$/)
  await expect(page.locator('.arco-table').getByText('传统后台路线')).toBeVisible()
  // 默认态不写任何 query，行尾锚点才能继续成立。
  expect(new URL(page.url()).search).toBe('')

  // 导入文章：multipart 上传，服务端按文件名建草稿。
  await page.locator('input[type="file"]').setInputFiles({
    name: '导入的文章.md',
    mimeType: 'text/markdown',
    buffer: Buffer.from('# 导入的文章')
  })
  await expect.poll(() => articleImported).toBe(true)
  expect(articleImportHasFile).toBe(true)

  // 批量导出：请求体必须是裸的 id 数组，返回的 Markdown 链接要展示出来。
  await page.getByRole('row', { name: /传统后台路线/ }).getByRole('checkbox').locator('..').click()
  await expect(page.getByRole('button', { name: '导出 Markdown' })).toBeVisible()
  await page.getByRole('button', { name: '导出 Markdown' }).click()
  await expect.poll(() => articleExported).toBe(true)
  expect(articleExportedIds).toEqual([7])
  await expect(page.locator('.arco-modal:visible')).toContainText('传统后台路线.md')
  await page.keyboard.press('Escape')
  await expect(page.locator('.arco-modal:visible')).toHaveCount(0)

  // 搜索防抖：停止输入后既查一次接口，也把非默认条件写进地址栏。
  const search = page.locator('.arco-input-search').getByRole('textbox')
  const filtered = page.waitForResponse((response) => {
    const responseURL = new URL(response.url())
    return responseURL.pathname === '/api/v1/admin/articles' &&
      response.request().method() === 'GET' &&
      responseURL.searchParams.get('keywords') === '联调'
  })
  await search.fill('联调')
  await filtered
  await expect(page).toHaveURL(/keywords=/)
  // 第一页是默认值，不应该出现在地址栏里。
  expect(new URL(page.url()).searchParams.has('page')).toBe(false)

  // 刷新后筛选条件仍然生效，说明 URL 是唯一事实来源。
  await page.reload()
  await expect(page.locator('.arco-input-search').getByRole('textbox')).toHaveValue('联调')

  // 照片批量移动相册：PUT 的请求体要同时带 photoIds 与 albumId。
  await page.goto('/albums/5')
  await expect(page.getByRole('main').getByText('项目截图 · 照片')).toBeVisible()
  await page.getByRole('row', { name: /首页截图/ }).getByRole('checkbox').locator('..').click()
  await page.getByRole('button', { name: '移动到相册' }).click()
  const moveDialog = page.locator('.arco-modal:visible')
  await expect(moveDialog).toContainText('目标相册')
  await moveDialog.locator('.arco-select').click()
  await page.locator('.arco-select-option').filter({ hasText: '旅行相册' }).click()
  await moveDialog.getByRole('button', { name: '确定' }).click()
  await expect.poll(() => photoMovePayload?.albumId).toBe(6)
  expect(photoMovePayload?.photoIds).toEqual([13])

  // 任务分组：列表接口返回数组，日志接口返回字符串，两者都要能被解析成选项。
  await page.goto('/quartz')
  await expect.poll(() => jobGroupsRequested).toBe(true)
  await page.goto('/quartz/log/85')
  await expect.poll(() => jobLogGroupsRequested).toBe(true)
  await page.getByRole('main').locator('.arco-select').first().click()
  await expect(page.locator('.arco-select-option').filter({ hasText: '清理' })).toBeVisible()
  await page.keyboard.press('Escape')

  expect(pageErrors()).toEqual([])
})

test('keeps a direct unknown route on an explicit 404 page', async ({ page }) => {
  const pageErrors = capturePageErrors(page)
  await page.addInitScript(() => sessionStorage.setItem('token', 'e2e-token'))
  await page.goto('/unknown-page')
  await expect(page.getByText('页面不存在')).toBeVisible()
  expect(pageErrors()).toEqual([])
})

test('explains when an authenticated account has no visible menu', async ({ page }) => {
  const pageErrors = capturePageErrors(page)
  returnEmptyMenus = true
  await page.goto('/login')
  await page.getByTestId('login-username').locator('input').fill('admin@example.com')
  await page.getByTestId('login-password').locator('input').fill('password')
  await page.getByTestId('login-submit').click()

  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByText('当前账号没有可见菜单，请联系管理员分配权限。')).toBeVisible()
  expect(pageErrors()).toEqual([])
})

test('all migrated routes keep their menu permission and survive a refresh', async ({ page }) => {
  // Validate every route plus a hard refresh. Keep the suite-level timeout
  // short for ordinary cases, but allow this deliberate full matrix to finish
  // on a cold Vite server.
  test.setTimeout(120_000)
  const pageErrors = capturePageErrors(page)
  await page.goto('/login')
  await page.getByTestId('login-username').locator('input').fill('admin@example.com')
  await page.getByTestId('login-password').locator('input').fill('password')
  await page.getByTestId('login-submit').click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('main').getByRole('button', { name: '发布文章', exact: true })).toBeVisible()

  await expect(page.locator('.admin-sider').getByText('内容管理', { exact: true })).toBeVisible()
  await expect(page.getByText('隐藏页面', { exact: true })).toHaveCount(0)
  await expect(page.getByText('仅用于权限测试', { exact: true })).toHaveCount(0)

  const routes = [
    '/',
    '/article-list',
    '/articles',
    '/articles/42',
    '/categories',
    '/tags',
    '/comments',
    '/users',
    '/roles',
    '/operation/log',
    '/exception/log',
    '/quartz',
    '/quartz/log/85',
    '/albums',
    '/albums/5',
    '/photos/delete',
    '/talk-list',
    '/talks/7',
    '/menus',
    '/resources',
    '/links',
    '/about',
    '/website',
    '/online/users',
    '/setting'
  ]

  for (const route of routes) {
    const response = await page.goto(route, { waitUntil: 'domcontentloaded' })
    expect(response?.status(), route).toBe(200)
    await expect(page.locator('.admin-shell'), route).toBeVisible()
    await expect(page.locator('.admin-content'), route).toBeVisible()
    await expect(page.locator('.admin-content'), route).not.toContainText('页面不存在')

    const refreshed = await page.reload({ waitUntil: 'domcontentloaded' })
    expect(refreshed?.status(), `${route} refresh`).toBe(200)
    await expect(page.locator('.admin-shell'), `${route} refresh`).toBeVisible()
    await expect(page.locator('.admin-content'), `${route} refresh`).toBeVisible()
    await expect(page.locator('.admin-content'), `${route} refresh`).not.toContainText('页面不存在')
  }

  expect(pageErrors()).toEqual([])
})

test('login first screen stays within the browser baseline budget', async ({ page }) => {
  const response = await page.goto('/login', { waitUntil: 'load' })
  expect(response?.status()).toBe(200)
  const timing = await page.evaluate(() => {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined
    const paint = performance.getEntriesByName('first-contentful-paint')[0]
    return {
      domContentLoaded: navigation?.domContentLoadedEventEnd ?? 0,
      loadEvent: navigation?.loadEventEnd ?? 0,
      firstContentfulPaint: paint?.startTime ?? 0
    }
  })
  expect(timing.domContentLoaded).toBeGreaterThan(0)
  expect(timing.domContentLoaded).toBeLessThan(5_000)
  expect(timing.loadEvent).toBeGreaterThan(0)
  expect(timing.loadEvent).toBeLessThan(5_000)
  if (timing.firstContentfulPaint > 0) expect(timing.firstContentfulPaint).toBeLessThan(5_000)
})

function capturePageErrors(page: import('@playwright/test').Page): () => string[] {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.stack || error.message))
  page.on('console', (message) => {
    if (message.type() === 'error') errors.push(message.text())
  })
  return () => errors
}
