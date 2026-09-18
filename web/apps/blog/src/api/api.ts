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
  trackContinuationEvent: (params: { articleId: number; eventType: string }) => {
    return http.post(`/public/articles/${encodeURIComponent(params.articleId)}/continuation-events`, { eventType: params.eventType })
  }
}
