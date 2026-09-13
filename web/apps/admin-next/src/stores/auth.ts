import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { apiErrorMessage, login as loginRequest, logout as logoutRequest } from '@/api/http'
import type { AdminUser } from '@stellar-beacon/api-contract'

const USER_KEY = 'stellar-beacon.admin.user'
const LEGACY_USER_KEY = 'benetnasch.admin.user'

export const useAuthStore = defineStore('admin-auth', () => {
  const token = ref(sessionStorage.getItem('token') || '')
  const user = ref<AdminUser | null>(readUser())
  const loading = ref(false)

  const isAuthenticated = computed(() => token.value.length > 0)

  function restore(): void {
    token.value = sessionStorage.getItem('token') || ''
    user.value = readUser()
  }

  async function login(username: string, password: string): Promise<AdminUser> {
    loading.value = true
    try {
      const result = await loginRequest(username.trim(), password)
      token.value = result.token
      user.value = result
      sessionStorage.setItem('token', result.token)
      sessionStorage.setItem(USER_KEY, JSON.stringify(result))
      return result
    } finally {
      loading.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      if (token.value) await logoutRequest()
    } catch (error) {
      // A stale/expired token should not prevent the local session from being cleared.
      void apiErrorMessage(error)
    } finally {
      clear()
    }
  }

  function clear(): void {
    token.value = ''
    user.value = null
    sessionStorage.removeItem('token')
    sessionStorage.removeItem(USER_KEY)
    sessionStorage.removeItem(LEGACY_USER_KEY)
  }

  function updateUser(patch: Partial<AdminUser>): void {
    if (!user.value) return
    Object.assign(user.value, patch)
    sessionStorage.setItem(USER_KEY, JSON.stringify(user.value))
  }

  return { token, user, loading, isAuthenticated, restore, login, logout, clear, updateUser }
})

function readUser(): AdminUser | null {
  let raw = sessionStorage.getItem(USER_KEY)
  if (raw === null) {
    raw = sessionStorage.getItem(LEGACY_USER_KEY)
    if (raw !== null) sessionStorage.setItem(USER_KEY, raw)
  }
  if (!raw) return null
  try {
    const value: unknown = JSON.parse(raw)
    return value && typeof value === 'object' ? (value as AdminUser) : null
  } catch {
    sessionStorage.removeItem(USER_KEY)
    return null
  }
}
