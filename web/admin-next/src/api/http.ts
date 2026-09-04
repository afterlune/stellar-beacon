import axios from 'axios'
import type { AxiosError, AxiosRequestConfig, AxiosResponse } from 'axios'

import { normalizePage, parseResult, unwrapResult, type AdminAlbum, type AdminFriendLink, type AdminJob, type AdminPhoto, type AdminRole, type AdminTalk, type AdminUser, type AgentMemoryAssertionPage, type AgentMemoryConflictPage, type AgentMemoryHistory, type AgentProfile, type AgentReviewPolicy, type AIReview, type AIVisionPreview, type AIWritingPreview, type JobRunOutcome, type Page, type ResultVO, type UserMenu, type UserRole } from '@shared/api-contract'

export const http = axios.create({
  baseURL: '/api',
  timeout: 15_000,
  headers: { Accept: 'application/json' }
})

export const AUTH_EXPIRED_EVENT = 'benetnasch-admin-auth-expired'

http.interceptors.request.use((config) => {
  const token = sessionStorage.getItem('token')
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    response.data = parseResult(response.data)
    return response
  },
  (error: AxiosError<unknown>) => {
    if (error.response) {
      const result = parseResult(error.response.data)
      error.response.data = result as never
      if (error.response.status === 401 || result.code === 40001 || result.code === 41000) {
        window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT))
      }
    }
    return Promise.reject(error)
  }
)

function responseData<T>(response: AxiosResponse<ResultVO<T>>): T {
  return unwrapResult<T>(response.data)
}

export function resultData<T>(response: AxiosResponse<ResultVO<T>>): T {
  return responseData(response)
}

export async function login(username: string, password: string): Promise<AdminUser & { token: string }> {
  const form = new URLSearchParams()
  form.set('username', username)
  form.set('password', password)
  const response = await http.post<ResultVO<AdminUser & { token: string }>>('users/login', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
  return responseData(response)
}

export async function listUserMenus(config?: AxiosRequestConfig): Promise<UserMenu[]> {
  const response = await http.get<ResultVO<UserMenu[]>>('admin/user/menus', config)
  return responseData(response)
}

export async function listUserRoles(): Promise<UserRole[]> {
  const response = await http.get<ResultVO<UserRole[]>>('admin/users/role')
  return responseData(response)
}

export async function updateAdminUser(payload: { userInfoId: number; nickname: string; roleIds: number[] }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/users/role', payload)
  responseData(response)
}

export async function updateAdminUserDisable(id: number, isDisable: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/users/disable', { id, isDisable })
  responseData(response)
}

export async function listAdminRoles(params: Record<string, string | number> = {}): Promise<Page<AdminRole>> {
  return listAdminPage<AdminRole>('admin/roles', params)
}

export async function listRoleMenus(): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/role/menus')
}

export async function listRoleResources(): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/role/resources')
}

export async function saveAdminRole(payload: { id?: number; roleName: string; resourceIds: number[]; menuIds: number[] }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/role', payload)
  responseData(response)
}

export async function deleteAdminRoles(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/roles', { data: ids })
  responseData(response)
}

export async function listAdminJobs(params: Record<string, string | number> = {}): Promise<Page<AdminJob>> {
  return listAdminPage<AdminJob>('admin/jobs', params)
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
  misfirePolicy: number
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

export async function runAdminJob(id: number, jobGroup: string): Promise<JobRunOutcome> {
  const response = await http.put<ResultVO<JobRunOutcome>>('admin/jobs/run', { id, jobGroup })
  return responseData(response)
}

export async function getAdminAgentProfile(): Promise<AgentProfile> {
  const response = await http.get<ResultVO<AgentProfile>>('admin/ai/profile')
  return responseData(response)
}

export async function updateAdminAgentProfile(payload: {
  name: string
  promptVersion: string
  systemPrompt: string
  opening: string
  awakePrompt: string
  duskPrompt: string
  nightPrompt: string
  enabled: boolean
}): Promise<AgentProfile> {
  const response = await http.patch<ResultVO<AgentProfile>>('admin/ai/profile', payload)
  return responseData(response)
}

export async function getAdminAgentReviewPolicy(): Promise<AgentReviewPolicy> {
  const response = await http.get<ResultVO<AgentReviewPolicy>>('admin/ai/review-policy')
  return responseData(response)
}

export async function updateAdminAgentReviewPolicy(payload: {
  version: number
  reviewTtlSeconds: number
  maxCandidateRunes: number
  similarityThreshold: number
  dailyLimit: number
  perArticleLimit: number
  perActionLimit: number
  allowedActions: string[]
  sensitivePatterns: string[]
}): Promise<AgentReviewPolicy> {
  const response = await http.patch<ResultVO<AgentReviewPolicy>>('admin/ai/review-policy', payload)
  return responseData(response)
}

export async function listAdminAgentMemoryAssertions(params: Record<string, string | number> = {}): Promise<AgentMemoryAssertionPage> {
  const response = await http.get<ResultVO<AgentMemoryAssertionPage>>('admin/ai/memory/assertions', { params })
  return responseData(response)
}

export async function listAdminAgentMemoryHistory(id: string): Promise<AgentMemoryHistory> {
  const response = await http.get<ResultVO<AgentMemoryHistory>>(`admin/ai/memory/assertions/${encodeURIComponent(id)}/history`)
  return responseData(response)
}

export async function revokeAdminAgentMemoryAssertion(id: string): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/ai/memory/assertions/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminAgentMemoryConflicts(params: Record<string, string | number> = {}): Promise<AgentMemoryConflictPage> {
  const response = await http.get<ResultVO<AgentMemoryConflictPage>>('admin/ai/memory/conflicts', { params })
  return responseData(response)
}

export async function resolveAdminAgentMemoryConflict(id: string, winnerAssertionId: string, resolution = ''): Promise<void> {
  const response = await http.post<ResultVO<unknown>>(`admin/ai/memory/conflicts/${encodeURIComponent(id)}/resolve`, { winnerAssertionId, resolution })
  responseData(response)
}

export async function rejectAdminAgentMemoryConflict(id: string, resolution = ''): Promise<void> {
  const response = await http.post<ResultVO<unknown>>(`admin/ai/memory/conflicts/${encodeURIComponent(id)}/reject`, { resolution })
  responseData(response)
}

export async function listAdminAlbums(params: Record<string, string | number> = {}): Promise<Page<AdminAlbum>> {
  return listAdminPage<AdminAlbum>('admin/photos/albums', params)
}

export async function saveAdminAlbum(payload: { id?: number; albumName: string; albumDesc: string; albumCover: string; status: number }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/photos/albums', payload)
  responseData(response)
}

export async function deleteAdminAlbum(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/photos/albums/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminFriendLinks(params: Record<string, string | number> = {}): Promise<Page<AdminFriendLink>> {
  return listAdminPage<AdminFriendLink>('admin/links', params)
}

export async function saveAdminFriendLink(payload: { id?: number; linkName: string; linkAvatar: string; linkAddress: string; linkIntro: string }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/links', payload)
  responseData(response)
}

export async function deleteAdminFriendLinks(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/links', { data: ids })
  responseData(response)
}

export async function uploadAdminAlbumCover(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('admin/photos/albums/upload', form)
  return responseData(response)
}

export async function listAdminTalks(params: Record<string, string | number> = {}): Promise<Page<AdminTalk>> {
  return listAdminPage<AdminTalk>('admin/talks', params)
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

export async function getAdminAlbum(id: number): Promise<AdminAlbum> {
  const response = await http.get<ResultVO<AdminAlbum>>(`admin/photos/albums/${encodeURIComponent(id)}/info`)
  return responseData(response)
}

export async function listAdminAlbumOptions(): Promise<AdminAlbum[]> {
  const response = await http.get<ResultVO<AdminAlbum[]>>('admin/photos/albums/info')
  return responseData(response)
}

export async function listAdminPhotos(params: Record<string, string | number> = {}): Promise<Page<AdminPhoto>> {
  return listAdminPage<AdminPhoto>('admin/photos', params)
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
  const response = await http.put<ResultVO<unknown>>('admin/photos/delete', { ids, isDelete })
  responseData(response)
}

export async function deleteAdminPhotos(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/photos', { data: ids })
  responseData(response)
}

export async function listAdminMenus(params: Record<string, string | number> = {}): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/menus', params)
}

export async function saveAdminMenu(payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/menus', payload)
  responseData(response)
}

export async function updateAdminMenuHidden(id: number, isHidden: number): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/menus/isHidden', { id, isHidden })
  responseData(response)
}

export async function deleteAdminMenu(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/menus/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminResources(params: Record<string, string | number> = {}): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/resources', params)
}

export async function saveAdminResource(payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/resources', payload)
  responseData(response)
}

export async function deleteAdminResource(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/resources/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function deleteAdminLogs(kind: 'operation' | 'exception' | 'job', ids: number[]): Promise<void> {
  const endpoint = kind === 'operation' ? 'admin/operation/logs' : kind === 'exception' ? 'admin/exception/logs' : 'admin/jobLogs'
  const response = await http.delete<ResultVO<unknown>>(endpoint, { data: ids })
  responseData(response)
}

export async function cleanAdminJobLogs(): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/jobLogs/clean')
  responseData(response)
}

export async function logout(): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('users/logout')
  responseData(response)
}

export async function listAdminPage<T>(path: string, params: Record<string, string | number>): Promise<Page<T>> {
  const response = await http.get<ResultVO<unknown>>(path, { params })
  return normalizePage<T>(responseData(response))
}

export async function listAdminCollection<T>(path: string, params: Record<string, string | number> = {}): Promise<T[]> {
  const response = await http.get<ResultVO<unknown>>(path, { params })
  const value = responseData(response)
  return Array.isArray(value) ? value as T[] : []
}

export async function getAdminArticle(id: string): Promise<Record<string, unknown>> {
  const response = await http.get<ResultVO<Record<string, unknown>>>(`admin/articles/${encodeURIComponent(id)}`)
  return responseData(response)
}

export async function saveAdminArticle(payload: Record<string, unknown>): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/articles', payload)
  responseData(response)
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

export async function deleteComment(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/comments', { data: [id] })
  responseData(response)
}

export async function getWebsiteConfig(): Promise<Record<string, unknown>> {
  const response = await http.get<ResultVO<Record<string, unknown>>>('admin/website/config')
  return responseData(response)
}

export async function updateWebsiteConfig(payload: Record<string, unknown>): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/website/config', payload)
  responseData(response)
}

export async function getAbout(): Promise<Record<string, unknown>> {
  const response = await http.get<ResultVO<Record<string, unknown>>>('about')
  return responseData(response)
}

export async function updateAbout(content: string): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('admin/about', { content })
  responseData(response)
}

export async function updateUserProfile(payload: { nickname: string; intro: string; website: string }): Promise<void> {
  const response = await http.put<ResultVO<unknown>>('users/info', payload)
  responseData(response)
}

export async function previewWriting(payload: Record<string, unknown>): Promise<AIWritingPreview> {
	const response = await http.post<ResultVO<AIWritingPreview>>('admin/ai/writing/preview', payload)
	return responseData(response)
}

export async function previewVision(payload: Record<string, unknown>): Promise<AIVisionPreview> {
	const response = await http.post<ResultVO<AIVisionPreview>>('admin/ai/vision/preview', payload)
	return responseData(response)
}

export async function listAIReviews(current: number, size: number): Promise<Page<AIReview>> {
  const response = await http.get<ResultVO<unknown>>('admin/ai/reviews', { params: { current, size } })
  return normalizePage<AIReview>(responseData(response))
}

export async function reviewAction(id: string, action: 'approve' | 'partial' | 'reject' | 'regenerate' | 'expire', payload: Record<string, unknown> = {}): Promise<void> {
  const response = await http.post<ResultVO<unknown>>(`admin/ai/reviews/${encodeURIComponent(id)}/${action}`, payload)
  responseData(response)
}

export function apiErrorMessage(error: unknown, fallback = '请求失败，请稍后再试'): string {
  const axiosError = error as AxiosError<unknown>
  const body = axiosError?.response?.data
  if (body && typeof body === 'object' && 'message' in body && typeof body.message === 'string') {
    return body.message
  }
  if (error instanceof Error && error.message) return error.message
  return fallback
}
