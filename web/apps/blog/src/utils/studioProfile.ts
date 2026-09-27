export interface StudioProfile {
  handle: string
  nickname: string
  avatar: string
  intro: string
  website: string
  about: string
  links: Array<{ label: string; url: string; description?: string }>
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
export type StudioActivationStepKey = 'identity' | 'content' | 'profile'

export interface StudioActivationStep {
  key: StudioActivationStepKey
  label: string
  description: string
  done: boolean
}

export interface StudioActivationDashboard {
  articleCount?: number
  talkCount?: number
}

export interface StudioActivationState {
  collapsed: boolean
  profileVisited: boolean
  startedAt: string
  completedAt: string
}

export const STUDIO_ACTIVATION_STORAGE_PREFIX = 'stellar-beacon:studio-activation:v1:'

export function studioActivationStorageKey(userId: unknown): string {
  const id = String(userId ?? '').trim()
  return STUDIO_ACTIVATION_STORAGE_PREFIX + (id || 'current')
}

export function readStudioActivationState(userId: unknown): StudioActivationState {
  const empty: StudioActivationState = { collapsed: false, profileVisited: false, startedAt: '', completedAt: '' }
  if (typeof localStorage === 'undefined') return empty
  try {
    const parsed = JSON.parse(localStorage.getItem(studioActivationStorageKey(userId)) || '{}')
    return {
      collapsed: Boolean(parsed?.collapsed),
      profileVisited: Boolean(parsed?.profileVisited),
      startedAt: String(parsed?.startedAt || ''),
      completedAt: String(parsed?.completedAt || '')
    }
  } catch {
    return empty
  }
}

export function saveStudioActivationState(userId: unknown, state: StudioActivationState): void {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(studioActivationStorageKey(userId), JSON.stringify({
    collapsed: Boolean(state.collapsed),
    profileVisited: Boolean(state.profileVisited),
    startedAt: String(state.startedAt || ''),
    completedAt: String(state.completedAt || '')
  }))
}

export function studioActivationProgress(
  profile: StudioProfile,
  dashboard: StudioActivationDashboard,
  profileVisited = false,
  siteDefaultAvatar = ''
) {
  const completion = studioProfileCompletion(profile, siteDefaultAvatar)
  const validHandle = isValidStudioHandle(normalizeStudioHandle(profile.handle))
  const contentCount = Number(dashboard.articleCount || 0) + Number(dashboard.talkCount || 0)
  const steps: StudioActivationStep[] = [
    {
      key: 'identity',
      label: '完成公开身份',
      description: '设置 Handle、昵称、头像和简介，让别人能认识你。',
      done: completion.completed === completion.total
    },
    {
      key: 'content',
      label: '写下第一条内容',
      description: '草稿、私有或公开文章与随想都会计入进度。',
      done: contentCount > 0
    },
    {
      key: 'profile',
      label: '预览公开主页',
      description: '检查你的主页在公共空间里的最终呈现。',
      done: validHandle && profileVisited
    }
  ]
  const completed = steps.filter((item) => item.done).length
  return {
    completed,
    total: steps.length,
    steps,
    isComplete: completed === steps.length,
    nextStep: steps.find((item) => !item.done) || null,
    profileComplete: completion.completed === completion.total,
    contentCount
  }
}
