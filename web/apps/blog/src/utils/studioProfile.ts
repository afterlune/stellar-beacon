export interface StudioProfile {
  handle: string
  nickname: string
  avatar: string
  intro: string
  website: string
}

export interface StudioProfileCheck {
  key: 'avatar' | 'nickname' | 'intro' | 'handle'
  label: string
  done: boolean
}

export const DEFAULT_STUDIO_NICKNAME = '用户'

export function normalizeStudioHandle(value: unknown): string {
  return String(value || '').trim().toLowerCase()
}

export function isValidStudioHandle(value: string): boolean {
  return /^[a-z0-9][a-z0-9-]{2,39}$/.test(value)
}

export function isValidStudioWebsite(value: string): boolean {
  if (!value) return true
  try {
    const parsed = new URL(value)
    return (parsed.protocol === 'http:' || parsed.protocol === 'https:') && Boolean(parsed.host)
  } catch {
    return false
  }
}

export function studioProfileCompletion(profile: StudioProfile, siteDefaultAvatar = '') {
  const handle = normalizeStudioHandle(profile.handle)
  const nickname = String(profile.nickname || '').trim()
  const avatar = String(profile.avatar || '').trim()
  const intro = String(profile.intro || '').trim()
  const checks: StudioProfileCheck[] = [
    { key: 'avatar', label: '头像', done: Boolean(avatar) && avatar !== siteDefaultAvatar },
    { key: 'nickname', label: '昵称', done: Boolean(nickname) && nickname !== DEFAULT_STUDIO_NICKNAME },
    { key: 'intro', label: '简介', done: Boolean(intro) },
    { key: 'handle', label: 'Handle', done: isValidStudioHandle(handle) }
  ]
  const completed = checks.filter((item) => item.done).length
  return {
    completed,
    total: checks.length,
    checks,
    missing: checks.filter((item) => !item.done)
  }
}