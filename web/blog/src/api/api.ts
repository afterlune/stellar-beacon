import axios, { type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { app } from '@/main'
import {
  parseResult,
  type Album,
  type AgentFeatureFlags,
  type AgentVitals,
  type Archive,
  type Article,
  type ArticleDetail,
  type Category,
  type Capsule,
  type Comment,
  type FriendLink,
  type GalaxyPoint,
  type Page,
  type Photo,
  type RadioPage,
  type ResultVO,
  type LoginResponse,
  type Tag,
  type Talk,
  type TopAndFeaturedArticles,
  type VideoPage,
  type WebsiteSummary
} from '@shared/api-contract'

export const API_BASE_PATH = '/api'
export const apiPath = (path = '') => `${API_BASE_PATH}${path ? `/${path.replace(/^\/+/, '')}` : ''}`

type QueryParams = Record<string, unknown>
type RouteParam = number | string | string[]
export type APIResponse<T> = Promise<AxiosResponse<ResultVO<T>>>

const get = <T>(path: string, config?: AxiosRequestConfig): APIResponse<T> =>
  axios.get<ResultVO<T>>(apiPath(path), config)
const post = <T>(path: string, data?: unknown, config?: AxiosRequestConfig): APIResponse<T> =>
  axios.post<ResultVO<T>>(apiPath(path), data, config)
const put = <T>(path: string, data?: unknown, config?: AxiosRequestConfig): APIResponse<T> =>
  axios.put<ResultVO<T>>(apiPath(path), data, config)

axios.interceptors.request.use((config: any) => {
  config.headers = config.headers || {}
  config.headers.Authorization = 'Bearer ' + sessionStorage.getItem('token')
  return config
})

axios.interceptors.response.use(
  (response: AxiosResponse<unknown>) => {
    const result = parseResult(response.data)
    response.data = result
    if (result.code === 41000) {
      sessionStorage.clear()
      app.config.globalProperties.$notify({
        title: 'Warning',
        message: '登录已过期',
        type: 'warning'
      })
      location.href = '/'
      return Promise.reject(new Error('登录已过期'))
    }
    switch (result.code) {
      case 50000:
        app.config.globalProperties.$notify({
          title: 'Error',
          message: '系统异常,请联系管理员',
          type: 'error'
        })
        break
      case 40001:
        app.config.globalProperties.$notify({
          title: 'Error',
          message: '用户未登录',
          type: 'error'
        })
        break
    }
    return response as AxiosResponse<ResultVO<unknown>>
  },
  (error) => Promise.reject(error)
)

export default {
  getTopAndFeaturedArticles: () => get<TopAndFeaturedArticles>('articles/topAndFeatured'),
  getArticles: (params: QueryParams) => get<Page<Article>>('articles/all', { params }),
  getArticlesByCategoryId: (params: QueryParams) => get<Page<Article>>('articles/categoryId', { params }),
  getArticeById: (articleId: RouteParam) => get<ArticleDetail>(`articles/${encodeURIComponent(String(articleId))}`),
  getAllCategories: () => get<Category[]>('categories/all'),
  getAllTags: () => get<Tag[]>('tags/all'),
  getTopTenTags: () => get<Tag[]>('tags/topTen'),
  getArticlesByTagId: (params: QueryParams) => get<Page<Article>>('articles/tagId', { params }),
  getAllArchives: (params: QueryParams) => get<Page<Archive>>('archives/all', { params }),
  login: (params: unknown) => post<LoginResponse>('users/login', params),
  saveComment: (params: unknown) => post<unknown>('comments/save', params),
  getComments: (params: QueryParams) => get<Page<Comment>>('comments', { params }),
  getTopSixComments: () => get<Comment[]>('comments/topSix'),
  getAbout: () => get<{ content: string }>('about'),
  getFriendLink: () => get<FriendLink[]>('links'),
  submitUserInfo: (params: unknown) => put<unknown>('users/info', params),
  getUserInfoById: (id: RouteParam) => get<unknown>(`users/info/${encodeURIComponent(String(id))}`),
  updateUserSubscribe: (params: unknown) => put<unknown>('users/subscribe', params),
  sendValidationCode: (username: string) => get<unknown>('users/code', { params: { username } }),
  bindingEmail: (params: unknown) => put<unknown>('users/email', params),
  register: (params: unknown) => post<unknown>('users/register', params),
  searchArticles: (params: QueryParams) => get<Page<Article>>('articles/search', { params }),
  getAlbums: () => get<Album[]>('photos/albums'),
  getPhotosBuAlbumId: (albumId: RouteParam, params: QueryParams) =>
    get<Page<Photo>>(`albums/${encodeURIComponent(String(albumId))}/photos`, { params }),
  getWebsiteConfig: () => get<WebsiteSummary>(''),
  getAgentFeatures: () => get<AgentFeatureFlags>('agent/features'),
  getAgentVitals: () => get<AgentVitals>('agent/vitals'),
  getAgentGalaxy: (params: QueryParams) => get<Page<GalaxyPoint>>('galaxy', { params }),
  getDreams: (params: QueryParams) => get<Page<unknown>>('dreams', { params }),
  getRadio: () => get<RadioPage>('radio'),
  getVideos: (params: QueryParams) => get<VideoPage>('videos', { params }),
  createTimeCapsule: (data: unknown) => post<Capsule>('capsules', data),
  getTimeCapsule: (id: string) => get<Capsule>(`capsules/${encodeURIComponent(id)}`),
  sealTimeCapsule: (id: string) => post<Capsule>(`capsules/${encodeURIComponent(id)}/seal`),
  report: () => {
    void post<unknown>('report')
  },
  getTalks: (params: QueryParams) => get<Page<Talk>>('talks', { params }),
  getTalkById: (id: RouteParam) => get<Talk>(`talks/${encodeURIComponent(String(id))}`),
  logout: () => post<unknown>('users/logout'),
  getRepliesByCommentId: (commentId: RouteParam) =>
    get<Comment[]>(`comments/${encodeURIComponent(String(commentId))}/replies`),
  updatePassword: (params: unknown) => put<unknown>('users/password', params),
  accessArticle: (params: unknown) => post<unknown>('articles/access', params)
}
