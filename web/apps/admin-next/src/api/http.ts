import type { AxiosError, AxiosRequestConfig, AxiosResponse } from 'axios'
import { createApiClient } from '@stellar-beacon/api-client'

import { normalizePage, unwrapResult, type AdminAlbum, type AdminArticle, type AdminArticleView, type AdminContentAnalytics, type AdminDashboardAnalytics, type AdminFriendLink, type AdminJob, type AdminJobTarget, type AdminMediaAsset, type AdminPhoto, type AdminRole, type AdminTalk, type AdminUser, type CollectionSummary, type ContentAnalyticsRange, type ContentArticleAnalyticsDetail, type ContentArticlePerformance,
  type ContentAuditItem, type ContentAuditRecord, type ContentContinuationTarget, type DashboardRange, type GrowthSummaryItem, type JobRunOutcome, type NewsletterDelivery, type NewsletterHealth, type NewsletterSubscriber, type Page, type ResultVO, type UserMenu, type UserRole } from '@stellar-beacon/api-contract'
import { t } from '@/i18n'

export const AUTH_EXPIRED_EVENT = 'stellar-beacon-admin-auth-expired'

/** `GET admin/albums/options`（服务端 `PhotoAlbumDTO`）的可用字段。 */
export interface AdminAlbumOption {
  id: number
  albumName: string
  albumDesc?: string
  albumCover?: string
}

export const http = createApiClient({
  getToken: () => sessionStorage.getItem('token'),
  onUnauthorized: () => window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT))
})

function responseData<T>(response: AxiosResponse<ResultVO<T>>): T {
  return unwrapResult<T>(response.data)
}

export async function login(username: string, password: string): Promise<AdminUser & { token: string }> {
  const form = new URLSearchParams()
  form.set('username', username)
  form.set('password', password)
  const response = await http.post<ResultVO<AdminUser & { token: string }>>('auth/login', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
  return responseData(response)
}

export async function listUserMenus(config?: AxiosRequestConfig): Promise<UserMenu[]> {
  const response = await http.get<ResultVO<UserMenu[]>>('admin/me/menu', config)
  return responseData(response)
}

export async function getAdminDashboardAnalytics(
  range: DashboardRange = '7d',
  areaType: 'users' | 'visitors' = 'users',
  config?: AxiosRequestConfig
): Promise<AdminDashboardAnalytics> {
  const response = await http.get<ResultVO<AdminDashboardAnalytics>>('admin/dashboard/analytics', {
    ...config,
    params: { range, areaType }
  })
  return responseData(response)
}

export async function listNewsletterSubscribers(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<NewsletterSubscriber>> {
  return listAdminPage<NewsletterSubscriber>('admin/newsletter/subscribers', params, config)
}

export async function updateNewsletterSubscriberStatus(id: number, status: string): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/newsletter/subscribers/status', { id, status })
  responseData(response)
}

export async function resendNewsletterConfirmation(id: number): Promise<void> {
  const response = await http.post<ResultVO<unknown>>(`admin/newsletter/subscribers/${encodeURIComponent(id)}/resend`)
  responseData(response)
}

export async function listNewsletterDeliveries(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<NewsletterDelivery>> {
  return listAdminPage<NewsletterDelivery>('admin/newsletter/deliveries', params, config)
}

export async function retryNewsletterDelivery(id: number): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/newsletter/deliveries/retry', { id })
  responseData(response)
}

export async function retryFailedNewsletterDeliveries(): Promise<number> {
  const response = await http.post<ResultVO<{ count: number }>>('admin/newsletter/deliveries/retry-failed')
  return Number(responseData(response)?.count || 0)
}

export async function getNewsletterHealth(): Promise<NewsletterHealth> {
  const response = await http.get<ResultVO<NewsletterHealth>>('admin/newsletter/health')
  return responseData(response)
}

export async function getGrowthSummary(days = 30): Promise<GrowthSummaryItem[]> {
  const response = await http.get<ResultVO<GrowthSummaryItem[]>>('admin/growth/summary', { params: { days } })
  return responseData(response)
}

export async function listUserRoles(): Promise<UserRole[]> {
  const response = await http.get<ResultVO<UserRole[]>>('admin/users/roles')
  return responseData(response)
}

export async function updateAdminUser(payload: { userInfoId: number; nickname: string; roleIds: number[] }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/users/roles', payload)
  responseData(response)
}

export async function updateAdminUserDisable(id: number, isDisable: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/users/disable', { id, isDisable })
  responseData(response)
}

export async function listAdminRoles(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminRole>> {
  return listAdminPage<AdminRole>('admin/roles', params, config)
}

export async function listRoleMenus(): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/roles/menu-options')
}

export async function listRoleResources(): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/roles/resource-options')
}

export async function saveAdminRole(payload: { id?: number; roleName: string; resourceIds: number[]; menuIds: number[] }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/roles', payload)
  responseData(response)
}

export async function deleteAdminRoles(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/roles', { data: ids })
  responseData(response)
}

export async function listAdminJobs(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminJob>> {
  return listAdminPage<AdminJob>('admin/jobs', params, config)
}

export async function getAdminJob(id: number): Promise<AdminJob> {
  const response = await http.get<ResultVO<AdminJob>>(`admin/jobs/${encodeURIComponent(id)}`)
  return responseData(response)
}

export async function saveAdminJob(payload: {
  id?: number
  jobName: string
  jobGroup: string
  invokeTarget: string
  cronExpression: string
  concurrent: number
  status: number
  remark: string
}): Promise<void> {
  const response = payload.id
    ? await http.put<ResultVO<unknown>>('admin/jobs', payload)
    : await http.post<ResultVO<unknown>>('admin/jobs', payload)
  responseData(response)
}

export async function deleteAdminJobs(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/jobs', { data: ids })
  responseData(response)
}

export async function updateAdminJobStatus(id: number, status: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/jobs/status', { id, status })
  responseData(response)
}

export async function runAdminJob(id: number): Promise<JobRunOutcome> {
  const response = await http.put<ResultVO<JobRunOutcome>>('admin/jobs/run', { id })
  return responseData(response)
}

export async function getAdminContentAnalytics(range: ContentAnalyticsRange = '7d'): Promise<AdminContentAnalytics> {
  const response = await http.get<ResultVO<AdminContentAnalytics>>('admin/content/analytics', { params: { range } })
  return responseData(response)
}

export async function listAdminContentArticles(
  range: ContentAnalyticsRange,
  sort: string,
  current: number,
  size: number
): Promise<Page<ContentArticlePerformance>> {
  return listAdminPage<ContentArticlePerformance>('admin/content/analytics/articles', { range, sort, current, size })
}

export async function getAdminContentArticleAnalytics(articleId: number, range: ContentAnalyticsRange = '7d'): Promise<ContentArticleAnalyticsDetail> {
  const response = await http.get<ResultVO<ContentArticleAnalyticsDetail>>(`admin/content/analytics/articles/${encodeURIComponent(articleId)}`, { params: { range } })
  return responseData(response)
}

export async function listAdminContinuationTargets(
  range: ContentAnalyticsRange,
  sort: string,
  current: number,
  size: number,
  sourceArticleId?: number
): Promise<Page<ContentContinuationTarget>> {
  return listAdminPage<ContentContinuationTarget>('admin/content/analytics/continuation-targets', {
    range,
    sort,
    current,
    size,
    ...(sourceArticleId ? { sourceArticleId } : {})
  })
}
export async function listAdminJobTargets(): Promise<AdminJobTarget[]> {
  const response = await http.get<ResultVO<AdminJobTarget[]>>('admin/jobs/targets')
  return responseData(response)
}

export async function listAdminAlbums(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminAlbum>> {
  return listAdminPage<AdminAlbum>('admin/albums', params, config)
}

export async function saveAdminAlbum(payload: { id?: number; albumName: string; albumDesc: string; albumCover: string; status: number }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/albums', payload)
  responseData(response)
}

export async function deleteAdminAlbum(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/albums/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminFriendLinks(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminFriendLink>> {
  return listAdminPage<AdminFriendLink>('admin/friend-links', params, config)
}

export async function saveAdminFriendLink(payload: { id?: number; linkName: string; linkAvatar: string; linkAddress: string; linkIntro: string }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/friend-links', payload)
  responseData(response)
}

export async function reviewAdminFriendLinks(ids: number[], status: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/friend-links/review', { ids, status })
  responseData(response)
}

export async function deleteAdminFriendLinks(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/friend-links', { data: ids })
  responseData(response)
}

/** 文章系列（合集）：列表、下拉选项与写操作。 */
export interface AdminSeries {
  id: number
  userId?: number
  authorHandle?: string
  authorNickname?: string
  authorAvatar?: string
  seriesName: string
  seriesDesc: string
  cover: string
  status?: number
  moderationStatus?: 'visible' | 'hidden' | string
  moderationReason?: string
  moderatedBy?: number
  moderatedAt?: string
  articleCount: number
  updateTime?: string
}

export async function getAdminSeries(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminSeries>> {
  return listAdminPage<AdminSeries>('admin/series', params, config)
}

export async function getAdminCollections(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<CollectionSummary>> {
  return listAdminPage<CollectionSummary>('admin/collections', params, config)
}

export async function listAdminSeriesOptions(): Promise<AdminSeries[]> {
  const response = await http.get<ResultVO<AdminSeries[]>>('admin/series/options')
  return responseData(response)
}

export async function saveAdminSeries(payload: { id?: number; seriesName: string; seriesDesc: string; cover: string }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/series', payload)
  responseData(response)
}

export async function deleteAdminSeries(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/series', { data: ids })
  responseData(response)
}

export async function uploadAdminAlbumCover(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('admin/albums/cover', form)
  return responseData(response)
}

export async function listAdminTalks(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminTalk>> {
  return listAdminPage<AdminTalk>('admin/talks', params, config)
}

export async function getAdminTalk(id: number): Promise<AdminTalk> {
  const response = await http.get<ResultVO<AdminTalk>>(`admin/talks/${encodeURIComponent(id)}`)
  return responseData(response)
}

export async function saveAdminTalk(payload: { id?: number; content: string; images: string; isTop: number; status: number }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/talks', payload)
  responseData(response)
}

export async function deleteAdminTalks(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/talks', { data: ids })
  responseData(response)
}

export async function uploadAdminTalkImage(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('admin/talks/images', form)
  return responseData(response)
}

export async function uploadAdminArticleImage(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('admin/articles/images', form)
  return responseData(response)
}

export async function listAdminCategories(keywords = ''): Promise<Array<Record<string, unknown>>> {
  return listAdminCollection<Record<string, unknown>>('admin/categories/search', keywords ? { keywords } : {})
}

export async function listAdminTags(keywords = ''): Promise<Array<Record<string, unknown>>> {
  return listAdminCollection<Record<string, unknown>>('admin/tags/search', keywords ? { keywords } : {})
}

export async function getAdminAlbum(id: number): Promise<AdminAlbum> {
  const response = await http.get<ResultVO<AdminAlbum>>(`admin/albums/${encodeURIComponent(id)}`)
  return responseData(response)
}

export async function listAdminPhotos(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminPhoto>> {
  return listAdminPage<AdminPhoto>('admin/photos', params, config)
}

export async function uploadAdminPhoto(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('admin/photos/upload', form)
  return responseData(response)
}

export async function saveAdminPhotos(albumId: number, photoUrls: string[]): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/photos', { albumId: String(albumId), photoUrls })
  responseData(response)
}

export async function updateAdminPhoto(payload: { id: number; photoName: string; photoDesc: string }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/photos', payload)
  responseData(response)
}

export async function updateAdminPhotoDelete(ids: number[], isDelete = 1): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/photos/trash', { ids, isDelete })
  responseData(response)
}

export async function deleteAdminPhotos(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/photos', { data: ids })
  responseData(response)
}

/** Move the selected photos into another album. */
export async function moveAdminPhotosToAlbum(photoIds: number[], albumId: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/photos/album', { photoIds, albumId })
  responseData(response)
}

/** Album id/name options used by the "move photos" picker. */
export async function listAdminAlbumOptions(): Promise<AdminAlbumOption[]> {
  return listAdminCollection<AdminAlbumOption>('admin/albums/options')
}

export async function listAdminMenus(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/menus', params, config)
}

export async function saveAdminMenu(payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/menus', payload)
  responseData(response)
}

export async function updateAdminMenuHidden(id: number, isHidden: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>(`admin/menus/${encodeURIComponent(id)}/visibility`, { id, isHidden })
  responseData(response)
}

export async function deleteAdminMenu(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/menus/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminResources(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/permissions', params, config)
}

export async function saveAdminResource(payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/permissions', payload)
  responseData(response)
}

export async function deleteAdminResource(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/permissions/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function deleteAdminLogs(kind: 'operation' | 'exception' | 'job', ids: number[]): Promise<void> {
  const endpoint = kind === 'operation' ? 'admin/logs/operations' : kind === 'exception' ? 'admin/logs/exceptions' : 'admin/logs/jobs'
  const response = await http.delete<ResultVO<unknown>>(endpoint, { data: ids })
  responseData(response)
}

export async function cleanAdminJobLogs(): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/logs/jobs/clean')
  responseData(response)
}

export async function logout(): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('auth/logout')
  responseData(response)
}

/**
 * `config` 用于透传 `AbortSignal` 等请求级选项：列表视图在筛选/分页快速变化时
 * 需要中止上一次还在飞的请求（见 `composables/useAsyncList.ts`）。
 */
export async function listAdminPage<T>(
  path: string,
  params: Record<string, string | number>,
  config?: AxiosRequestConfig
): Promise<Page<T>> {
  const response = await http.get<ResultVO<unknown>>(path, { ...config, params })
  return normalizePage<T>(responseData(response))
}

export async function listAdminCollection<T>(
  path: string,
  params: Record<string, string | number> = {},
  config?: AxiosRequestConfig
): Promise<T[]> {
  const response = await http.get<ResultVO<unknown>>(path, { ...config, params })
  const value = responseData(response)
  return Array.isArray(value) ? value as T[] : []
}

export async function getAdminArticle(id: string): Promise<AdminArticleView> {
  const response = await http.get<ResultVO<AdminArticleView>>(`admin/articles/${encodeURIComponent(id)}`)
  return responseData(response)
}

export async function saveAdminArticle(payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/articles', payload)
  responseData(response)
}

export async function moderateAdminContent(payload: { contentType: 'article' | 'talk' | 'series' | 'collection'; id: number; hidden: boolean; reason: string }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>(`admin/content/${payload.contentType}/${encodeURIComponent(payload.id)}/moderation`, payload)
  responseData(response)
}

export async function distributeAdminArticle(articleId: number, payload: { featured: boolean; newsletter: boolean }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>(`admin/content/articles/${encodeURIComponent(articleId)}/distribution`, payload)
  responseData(response)
}
/** Toggle the 置顶 / 精选 flags of one article. */
export async function updateAdminArticleFeatured(payload: { id: number; isTop: number; isFeatured: number }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/articles/featured', payload)
  responseData(response)
}

/** Move articles into (or out of) the recycle bin. `isDelete: 0` restores them. */
export async function updateAdminArticleTrash(ids: number[], isDelete = 1): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/articles/trash', { ids, isDelete })
  responseData(response)
}

/** Permanently delete articles. */
export async function deleteAdminArticles(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/articles/batch-delete', { data: ids })
  responseData(response)
}

/**
 * Import one Markdown/text file as a new draft article.
 *
 * The backend derives the title from the file name and stores the raw text as
 * the body with `status: 3` (draft); the response carries no id, so callers can
 * only refresh the list afterwards.
 */
export async function importAdminArticles(file: File): Promise<void> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<unknown>>('admin/articles/import', form)
  responseData(response)
}

/**
 * Export articles as Markdown.
 *
 * `POST admin/articles/export` binds a bare JSON array of ids (not an object),
 * and answers with the public URLs of the uploaded `.md` files.
 */
export async function exportAdminArticles(ids: number[]): Promise<string[]> {
  const response = await http.post<ResultVO<unknown>>('admin/articles/export', ids)
  const value = responseData(response)
  return Array.isArray(value) ? value.map(String) : []
}

export async function saveTaxonomy(kind: 'categories' | 'tags', payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>(`admin/${kind}`, payload)
  responseData(response)
}

export async function deleteTaxonomy(kind: 'categories' | 'tags', ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/${kind}`, { data: ids })
  responseData(response)
}

export async function reviewComment(id: number, isReview: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/comments/review', { ids: [id], isReview })
  responseData(response)
}

/** Review several comments in one request (`ReviewVO` accepts an id list). */
export async function reviewComments(ids: number[], isReview: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/comments/review', { ids, isReview })
  responseData(response)
}

export async function deleteComments(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/comments', { data: ids })
  responseData(response)
}

/** Row shape shared by the comment governance endpoints. */
export interface AdminCommentRow extends Record<string, unknown> {
  id: number
  type?: number
  isReview?: number
  isTop?: number
  isDelete?: number
  collectionId?: number
  reportCount?: number
  commentContent?: string
  articleTitle?: string
  createTime?: string
}

export async function deleteComment(id: number): Promise<void> {

  const response = await http.delete<ResultVO<unknown>>('admin/comments', { data: [id] })
  responseData(response)
}

/** Admin governance view of one reading list's comments, soft-deleted rows included. */
export async function listCollectionComments(
  collectionId: number,
  params: Record<string, string | number> = {},
  config?: AxiosRequestConfig
): Promise<Page<AdminCommentRow>> {
  return listAdminPage<AdminCommentRow>(`admin/collections/${encodeURIComponent(collectionId)}/comments`, params, config)
}

/** Administrators restore soft-deleted comments; pinning stays with the reading-list owner. */
export async function restoreComment(id: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>(`admin/comments/${encodeURIComponent(id)}/restore`)
  responseData(response)
}

export async function getWebsiteConfig(): Promise<Record<string, unknown>> {
  const response = await http.get<ResultVO<Record<string, unknown>>>('admin/site')
  return responseData(response)
}

export async function updateWebsiteConfig(payload: Record<string, unknown>): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/site', payload)
  responseData(response)
}

export async function getAbout(): Promise<Record<string, unknown>> {
  const response = await http.get<ResultVO<Record<string, unknown>>>('public/about')
  return responseData(response)
}

export async function updateAbout(content: string): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/about', { content })
  responseData(response)
}

export async function updateUserProfile(payload: { nickname: string; intro: string; website: string }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('auth/me', payload)
  responseData(response)
}

export async function uploadUserAvatar(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('auth/me/avatar', form)
  return responseData(response)
}

export async function changeAdminPassword(payload: { oldPassword: string; newPassword: string }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/users/password', payload)
  responseData(response)
}

export async function listAdminOnlineUsers(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminUser>> {
  return listAdminPage<AdminUser>('admin/users/online', params, config)
}

/** Force a single online session to log out. */
export async function removeAdminOnlineUser(userInfoId: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/users/${encodeURIComponent(userInfoId)}/online`)
  responseData(response)
}

export async function listAdminJobGroups(): Promise<string[]> {
  return listJobGroupOptions('admin/jobs/groups')
}

export async function listAdminJobLogGroups(): Promise<string[]> {
  return listJobGroupOptions('admin/logs/jobs/groups')
}

/**
 * Job group options.
 *
 * `admin/jobs/groups` answers with a JSON array, while `admin/logs/jobs/groups`
 * currently serialises a single string from the repository, so both shapes are
 * accepted and split loosely instead of trusting one contract.
 */
async function listJobGroupOptions(path: string): Promise<string[]> {
  const response = await http.get<ResultVO<unknown>>(path)
  return normalizeGroupList(responseData(response))
}

function normalizeGroupList(value: unknown): string[] {
  if (Array.isArray(value)) return value.map((item) => String(item).trim()).filter(Boolean)
  if (typeof value === 'string') return value.split(/[,\n]/).map((item) => item.trim()).filter(Boolean)
  return []
}

export async function listAdminOperationLogs(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<Record<string, unknown>>> {
  return listAdminPage<Record<string, unknown>>('admin/logs/operations', params, config)
}

export async function listAdminExceptionLogs(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<Record<string, unknown>>> {
  return listAdminPage<Record<string, unknown>>('admin/logs/exceptions', params, config)
}

export async function listAdminContentAudits(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<ContentAuditRecord>> {
  return listAdminPage<ContentAuditRecord>('admin/content/audits', params, config)
}

export async function listAdminContentAuditItems(auditId: number, params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<ContentAuditItem>> {
  return listAdminPage<ContentAuditItem>(`admin/content/audits/${encodeURIComponent(auditId)}/items`, params, config)
}

export async function listAdminMedia(params: Record<string, string | number> = {}, config?: AxiosRequestConfig): Promise<Page<AdminMediaAsset>> {
  return listAdminPage<AdminMediaAsset>('admin/media', params, config)
}

export async function uploadAdminMedia(file: File): Promise<AdminMediaAsset> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<AdminMediaAsset>>('admin/media/upload', form)
  return responseData(response)
}

export async function deleteAdminMedia(keys: string[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/media', { data: keys })
  responseData(response)
}

export function apiErrorMessage(error: unknown, fallback?: string): string {
  const axiosError = error as AxiosError<unknown>
  const body = axiosError?.response?.data
  if (body && typeof body === 'object' && 'message' in body && typeof body.message === 'string') {
    return body.message
  }
  if (error instanceof Error && error.message) return error.message
  // 兜底文案延迟到调用时取词条：这个模块在 i18n 初始化时就参与构建图，
  // 在模块作用域读 t() 会拿到未初始化的语言。
  return fallback ?? t('common.requestFailed')
}
