import type { FollowContentType, NotificationCursor, NotificationGroup, StudioContentBatchDelete, StudioContentBatchPreview, StudioContentBatchStatus } from '@stellar-beacon/api-contract'
import { createApiClient } from '@stellar-beacon/api-client'

// The presentation layer still reads `flag` while the shared client is being
// adopted. The client converts the canonical server envelope at this one
// boundary, so no visual component needs to know about the transport change.
const http = createApiClient({
  legacyResponse: true,
  rejectBusinessErrors: false,
  getToken: () => sessionStorage.getItem('token'),
  onUnauthorized: () => {
    sessionStorage.removeItem('token')
    if (window.location.pathname !== '/') window.location.href = '/'
  }
})

// Transient platform failures (rate limits, restarts, upstream hiccups) are
// retried once for idempotent GETs before the caller sees an error.
const RETRYABLE_STATUS = new Set([429, 500, 502, 503, 504])
http.interceptors.response.use(undefined, async (error: any) => {
  const config = error?.config
  const status = Number(error?.response?.status)
  if (!config || config.__retried || String(config.method || 'get').toLowerCase() !== 'get') {
    return Promise.reject(error)
  }
  if (!RETRYABLE_STATUS.has(status)) return Promise.reject(error)
  config.__retried = true
  const retryAfter = Number(error?.response?.headers?.['retry-after'])
  const delay = Number.isFinite(retryAfter) && retryAfter > 0 ? Math.min(retryAfter * 1000, 5000) : 700
  await new Promise((resolve) => setTimeout(resolve, delay))
  return http.request(config)
})

export default {
  getTopAndFeaturedArticles: () => {
    return http.get('/public/articles/featured')
  },
  getArticles: (params: any) => {
    return http.get('/public/articles', { params })
  },
  getArticlesByCategoryId: (params: any) => {
    return http.get('/public/articles/by-category', { params })
  },
  getArticeById: (articleId: any) => {
    return http.get('/public/articles/' + encodeURIComponent(articleId))
  },
  getAllCategories: () => {
    return http.get('/public/categories')
  },
  getAllTags: () => {
    return http.get('/public/tags')
  },
  getTopTenTags: () => {
    return http.get('/public/tags/top')
  },
  getArticlesByTagId: (params: any) => {
    return http.get('/public/articles/by-tag', { params })
  },
  getAllArchives: (params: any) => {
    return http.get('/public/archives', { params })
  },
  login: (params: any) => {
    return http.post('/auth/login', params)
  },
  saveComment: (params: any) => {
    return http.post('/public/comments', params)
  },
  getComments: (params: any) => {
    return http.get('/public/comments', { params })
  },
  getTopSixComments: () => {
    return http.get('/public/comments/top')
  },
  getAbout: () => {
    return http.get('/public/about')
  },
  getFriendLink: () => {
    return http.get('/public/links')
  },
  submitUserInfo: (params: any) => {
    return http.put('/auth/me', params)
  },
  getUserInfoById: (id: any) => {
    return http.get('/public/users/' + encodeURIComponent(id))
  },
  updateUserSubscribe: (params: any) => {
    return http.put('/auth/me/subscription', params)
  },
  sendValidationCode: (username: any) => {
    return http.get('/auth/verification-code', {
      params: { username }
    })
  },
  bindingEmail: (params: any) => {
    return http.put('/auth/me/email', params)
  },
  register: (params: any) => {
    return http.post('/auth/register', params)
  },
  searchArticles: (params: any) => {
    return http.get('/public/articles/search', { params })
  },
  getAlbums: () => {
    return http.get('/public/albums')
  },
  getPhotosBuAlbumId: (albumId: any, params: any) => {
    return http.get('/public/albums/' + encodeURIComponent(albumId) + '/photos', { params })
  },
  getWebsiteConfig: () => {
    return http.get('/public/')
  },
  report: () => {
    return http.post('/public/reports/visit')
  },
  getTalks: (params: any) => {
    return http.get('/public/talks', { params })
  },
  getTalkById: (id: any) => {
    return http.get('/public/talks/' + encodeURIComponent(id))
  },
  logout: () => {
    return http.post('/auth/logout')
  },
  getRepliesByCommentId: (commentId: any) => {
    return http.get(`/public/comments/${encodeURIComponent(commentId)}/replies`)
  },
  updatePassword: (params: any) => {
    return http.put('/auth/password', params)
  },
  accessArticle: (params: any) => {
    return http.post('/public/articles/' + encodeURIComponent(params.articleId) + '/access', params)
  },
  subscribeNewsletter: (params: { email: string }) => {
    return http.post('/public/subscriptions', params)
  },
  confirmNewsletter: (params: { token: string }) => {
    return http.post('/public/subscriptions/confirm', params)
  },
  unsubscribeNewsletter: (params: { token: string }) => {
    return http.post('/public/subscriptions/unsubscribe', params)
  },
  trackGrowthEvent: (params: { eventName: string; articleId?: number; path?: string }) => {
    return http.post('/public/growth/events', params)
  },
  toggleArticleReaction: (params: { articleId: number; reaction: string; active: boolean }) => {
    return http.put('/auth/me/reactions', params)
  },
  getMyArticleReactions: (params: { reaction: string; current: number; size: number }) => {
    return http.get('/auth/me/reactions', { params })
  },
  getArticleReactionStates: (articleIds: number[]) => {
    return http.get('/auth/me/reactions/state', { params: { articleIds: articleIds.join(',') } })
  },
  updateCommentNotice: (params: { notifyComment: number }) => {
    return http.put('/auth/me/notifications', params)
  },
  applyFriendLink: (params: any) => {
    return http.post('/public/links/applications', params)
  },
  getSeriesList: () => {
    return http.get('/public/series')
  },
  getSeriesDetail: (seriesId: any) => {
    return http.get('/public/series/' + encodeURIComponent(seriesId))
  },
  trackContinuationEvent: (params: { articleId: number; eventType: string; targetType?: string; targetId?: number; placement?: string; position?: number }) => {
    return http.post(`/public/articles/${encodeURIComponent(params.articleId)}/continuation-events`, {
      eventType: params.eventType,
      targetType: params.targetType,
      targetId: params.targetId,
      placement: params.placement,
      position: params.position
    })
  },
  getPlatformFeed: (params: any) => {
    return http.get('/public/feed', { params })
  },
  getPlatformAuthors: (params: any) => {
    return http.get('/public/authors', { params })
  },
  getTopicOverview: (params?: { size?: number }) => {
    return http.get('/public/topics', { params })
  },
  getAuthorByHandle: (handle: string) => {
    return http.get('/public/authors/' + encodeURIComponent(handle))
  },
  getMyFollowing: (params: { current?: number; size?: number }) => {
    return http.get('/auth/me/following', { params })
  },
  getMyFollowers: (params: { current?: number; size?: number }) => {
    return http.get('/auth/me/followers', { params })
  },
  getFollowingFeed: (params: { type?: FollowContentType | 'all'; current?: number; size?: number }) => {
    return http.get('/auth/me/following-feed', { params })
  },
  getFollowNotifications: (params: { group?: NotificationGroup; current?: number; size?: number }) => {
    return http.get('/auth/me/notifications', { params })
  },
  getFollowNotificationUnreadCount: () => {
    return http.get('/auth/me/notifications/unread-count')
  },
  markFollowNotificationsRead: (cursor: NotificationCursor) => {
    return http.post('/auth/me/notifications/read', cursor)
  },
  updateNotificationPreferences: (params: { notifyInteraction: number; notifyTopic: number }) => {
    return http.put('/auth/me/notification-preferences', params)
  },
  subscribeTopic: (topicType: string, topicKey: string) => {
    return http.put(`/auth/me/topic-subscriptions/${encodeURIComponent(topicType)}/${encodeURIComponent(topicKey)}`)
  },
  unsubscribeTopic: (topicType: string, topicKey: string) => {
    return http.delete(`/auth/me/topic-subscriptions/${encodeURIComponent(topicType)}/${encodeURIComponent(topicKey)}`)
  },
  muteTopicSubscription: (topicType: string, topicKey: string, muted: number) => {
    return http.put(`/auth/me/topic-subscriptions/${encodeURIComponent(topicType)}/${encodeURIComponent(topicKey)}/mute`, { muted })
  },
  getTopicSubscriptions: (params: { current?: number; size?: number }) => {
    return http.get('/auth/me/topic-subscriptions', { params })
  },
  getTopicFeed: (params: { current?: number; size?: number }) => {
    return http.get('/auth/me/topic-feed', { params })
  },
  followAuthor: (authorId: number) => {
    return http.put('/auth/me/following/' + encodeURIComponent(authorId))
  },
  unfollowAuthor: (authorId: number) => {
    return http.delete('/auth/me/following/' + encodeURIComponent(authorId))
  },
  getAuthorArticles: (handle: string, params: any) => {
    return http.get('/public/authors/' + encodeURIComponent(handle) + '/articles', { params })
  },
  getAuthorTalks: (handle: string, params: any) => {
    return http.get('/public/authors/' + encodeURIComponent(handle) + '/talks', { params })
  },
  getAuthorSeries: (handle: string, params: any) => {
    return http.get('/public/authors/' + encodeURIComponent(handle) + '/series', { params })
  },
  getTopicArticles: (topic: string, slug: string, params: any) => {
    return http.get('/public/topics/' + encodeURIComponent(topic) + '/' + encodeURIComponent(slug) + '/articles', { params })
  },
  getStudioDashboard: () => {
    return http.get('/studio/dashboard')
  },
  getStudioAnalytics: (range: string) => {
    return http.get('/studio/analytics', { params: { range } })
  },
  getStudioCalendar: (start: string, end: string) => {
    return http.get('/studio/calendar', { params: { start, end } })
  },
  getStudioProfile: () => {
    return http.get('/studio/profile')
  },
  saveStudioProfile: (params: any) => {
    return http.put('/studio/profile', params)
  },
  uploadUserAvatar: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return http.post('/auth/me/avatar', form)
  },
  getStudioArticles: (params: any) => {
    return http.get('/studio/articles', { params })
  },
  getStudioArticle: (articleId: number) => {
    return http.get('/studio/articles/' + encodeURIComponent(articleId))
  },
  saveStudioArticle: (params: any, articleId?: number) => {
    return articleId ? http.put('/studio/articles/' + encodeURIComponent(articleId), params) : http.post('/studio/articles', params)
  },
  deleteStudioArticles: (ids: number[]) => {
    return http.delete('/studio/articles', { data: ids })
  },
  getStudioTalks: (params: any) => {
    return http.get('/studio/talks', { params })
  },
  getStudioTalk: (talkId: number) => {
    return http.get('/studio/talks/' + encodeURIComponent(talkId))
  },
  saveStudioTalk: (params: any, talkId?: number) => {
    return talkId ? http.put('/studio/talks/' + encodeURIComponent(talkId), params) : http.post('/studio/talks', params)
  },
  deleteStudioTalks: (ids: number[]) => {
    return http.delete('/studio/talks', { data: ids })
  },
  getStudioSeries: (params: any) => {
    return http.get('/studio/series', { params })
  },
  getStudioSeriesItem: (seriesId: number) => {
    return http.get('/studio/series/' + encodeURIComponent(seriesId))
  },
  saveStudioSeries: (params: any, seriesId?: number) => {
    return seriesId ? http.put('/studio/series/' + encodeURIComponent(seriesId), params) : http.post('/studio/series', params)
  },
  deleteStudioSeries: (seriesId: number) => {
    return http.delete('/studio/series/' + encodeURIComponent(seriesId))
  },
  previewStudioContent: (params: StudioContentBatchPreview) => {
    return http.post('/studio/content/batch-preview', params)
  },
  batchUpdateStudioContentStatus: (params: StudioContentBatchStatus) => {
    return http.put('/studio/content/batch-status', params)
  },
  batchDeleteStudioContent: (params: StudioContentBatchDelete) => {
    return http.delete('/studio/content/batch', { data: params })
  },
  retryStudioArticlePublication: (articleId: number) => {
    return http.post('/studio/articles/' + encodeURIComponent(articleId) + '/publish-retry')
  },
  getStudioCategories: () => {
    return http.get('/studio/categories')
  },
  saveStudioCategory: (params: any, categoryId?: number) => {
    return categoryId ? http.put('/studio/categories/' + encodeURIComponent(categoryId), params) : http.post('/studio/categories', params)
  },
  deleteStudioCategory: (categoryId: number) => {
    return http.delete('/studio/categories/' + encodeURIComponent(categoryId))
  },
  getStudioTags: () => {
    return http.get('/studio/tags')
  },
  saveStudioTag: (params: any, tagId?: number) => {
    return tagId ? http.put('/studio/tags/' + encodeURIComponent(tagId), params) : http.post('/studio/tags', params)
  },
  deleteStudioTag: (tagId: number) => {
    return http.delete('/studio/tags/' + encodeURIComponent(tagId))
  },
  uploadStudioAsset: (file: File, kind: string) => {
    const form = new FormData()
    form.append('file', file)
    return http.post('/studio/uploads?kind=' + encodeURIComponent(kind), form)
  }}
