export const AUTH_SESSION_STORAGE_KEY = 'stellar-beacon:auth-session'

export type AuthUserInfo = Record<string, any>

export type AuthSession = {
  token: string
  userInfo: AuthUserInfo
}

const normalizeUserInfo = (userInfo: unknown): AuthUserInfo => {
  if (!userInfo || typeof userInfo !== 'object' || Array.isArray(userInfo)) return {}
  const { token: _token, ...profile } = userInfo as Record<string, any>
  return profile
}

const isTokenExpired = (token: string): boolean => {
  const encodedPayload = token.split('.')[1]
  if (!encodedPayload) return false
  try {
    const base64 = encodedPayload.replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
    const bytes = Uint8Array.from(window.atob(padded), (character) => character.charCodeAt(0))
    const claims = JSON.parse(new TextDecoder().decode(bytes))
    const expiresAt = Number(claims?.exp)
    return Number.isFinite(expiresAt) && expiresAt * 1000 <= Date.now()
  } catch {
    return false
  }
}

export const parseAuthSession = (raw: string | null): AuthSession | null => {
  if (!raw) return null
  try {
    const value = JSON.parse(raw)
    if (typeof value?.token !== 'string' || !value.token || isTokenExpired(value.token)) return null
    return { token: value.token, userInfo: normalizeUserInfo(value.userInfo) }
  } catch {
    return null
  }
}

const readLegacyAuthSession = (): AuthSession | null => {
  if (typeof window === 'undefined') return null
  try {
    const stored = window.sessionStorage.getItem('userStore')
    const state = stored ? JSON.parse(stored) : null
    const token = window.sessionStorage.getItem('token') || state?.token || state?.userInfo?.token
    if (typeof token !== 'string' || !token || isTokenExpired(token)) return null
    return { token, userInfo: normalizeUserInfo(state?.userInfo) }
  } catch {
    return null
  }
}

export const writeAuthSession = (userInfo: unknown, token: string): void => {
  if (typeof window === 'undefined' || !token) return
  const session = { token, userInfo: normalizeUserInfo(userInfo) }
  try { window.localStorage.setItem(AUTH_SESSION_STORAGE_KEY, JSON.stringify(session)) } catch { /* persistent storage may be blocked */ }
  try { window.sessionStorage.setItem('token', token) } catch { /* per-tab storage may be blocked */ }
}

export const readAuthSession = (): AuthSession | null => {
  if (typeof window === 'undefined') return null
  let raw: string | null
  try {
    raw = window.localStorage.getItem(AUTH_SESSION_STORAGE_KEY)
  } catch {
    return readLegacyAuthSession()
  }
  const shared = parseAuthSession(raw)
  if (shared) {
    try { window.sessionStorage.setItem('token', shared.token) } catch { /* per-tab storage may be blocked */ }
    return shared
  }
  if (raw) {
    try { window.localStorage.removeItem(AUTH_SESSION_STORAGE_KEY) } catch { /* ignore unavailable storage */ }
  }

  // Migrate the current tab's old session once instead of signing the user out.
  const legacy = readLegacyAuthSession()
  if (!legacy) return null
  writeAuthSession(legacy.userInfo, legacy.token)
  return legacy
}

export const clearAuthSession = (): void => {
  if (typeof window === 'undefined') return
  try { window.localStorage.removeItem(AUTH_SESSION_STORAGE_KEY) } catch { /* persistent storage may be blocked */ }
  try { window.sessionStorage.removeItem('token') } catch { /* per-tab storage may be blocked */ }
}

export const getAuthToken = (): string | null => readAuthSession()?.token || null
