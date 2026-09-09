import type { AxiosError, AxiosRequestConfig, AxiosResponse } from 'axios'
import { createApiClient } from '@benetnasch/api-client'

import { normalizePage, unwrapResult, type AdminAlbum, type AdminFriendLink, type AdminJob, type AdminPhoto, type AdminRole, type AdminTalk, type AdminUser, type JobRunOutcome, type Page, type ResultVO, type UserMenu, type UserRole } from '@benetnasch/api-contract'

export const AUTH_EXPIRED_EVENT = 'benetnasch-admin-auth-expired'

export const http = createApiClient({
  getToken: () => sessionStorage.getItem('token'),
  onUnauthorized: () => window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT))
})

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
  const response = await http.post<ResultVO<AdminUser & { token: string }>>('auth/login', form, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
  return responseData(response)
}

export async function listUserMenus(config?: AxiosRequestConfig): Promise<UserMenu[]> {
  const response = await http.get<ResultVO<UserMenu[]>>('admin/me/menu', config)
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

export async function listAdminRoles(params: Record<string, string | number> = {}): Promise<Page<AdminRole>> {
  return listAdminPage<AdminRole>('admin/roles', params)
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

export async function listAdminAlbums(params: Record<string, string | number> = {}): Promise<Page<AdminAlbum>> {
  return listAdminPage<AdminAlbum>('admin/albums', params)
}

export async function saveAdminAlbum(payload: { id?: number; albumName: string; albumDesc: string; albumCover: string; status: number }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/albums', payload)
  responseData(response)
}

export async function deleteAdminAlbum(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/albums/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminFriendLinks(params: Record<string, string | number> = {}): Promise<Page<AdminFriendLink>> {
  return listAdminPage<AdminFriendLink>('admin/friend-links', params)
}

export async function saveAdminFriendLink(payload: { id?: number; linkName: string; linkAvatar: string; linkAddress: string; linkIntro: string }): Promise<void> {
  const response = await http.post<ResultVO<unknown>>('admin/friend-links', payload)
  responseData(response)
}

export async function deleteAdminFriendLinks(ids: number[]): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>('admin/friend-links', { data: ids })
  responseData(response)
}

export async function uploadAdminAlbumCover(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const response = await http.post<ResultVO<string>>('admin/albums/cover', form)
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
  const response = await http.get<ResultVO<AdminAlbum>>(`admin/albums/${encodeURIComponent(id)}`)
  return responseData(response)
}

export async function listAdminAlbumOptions(): Promise<AdminAlbum[]> {
  const response = await http.get<ResultVO<AdminAlbum[]>>('admin/albums/options')
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
  const response = await http.put<ResultVO<unknown>>('admin/photos/trash', { ids, isDelete })
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
  const response = await http.put<ResultVO<unknown>>(`admin/menus/${encodeURIComponent(id)}/visibility`, { id, isHidden })
  responseData(response)
}

export async function deleteAdminMenu(id: number): Promise<void> {
  const response = await http.delete<ResultVO<unknown>>(`admin/menus/${encodeURIComponent(id)}`)
  responseData(response)
}

export async function listAdminResources(params: Record<string, string | number> = {}): Promise<unknown[]> {
  return listAdminCollection<unknown>('admin/permissions', params)
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

export function apiErrorMessage(error: unknown, fallback = '请求失败，请稍后再试'): string {
  const axiosError = error as AxiosError<unknown>
  const body = axiosError?.response?.data
  if (body && typeof body === 'object' && 'message' in body && typeof body.message === 'string') {
    return body.message
  }
  if (error instanceof Error && error.message) return error.message
  return fallback
}
