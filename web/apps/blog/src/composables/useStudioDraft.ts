export interface StudioDraftSnapshot<T> {
  version: 1
  savedAt: string
  data: T
}

export function studioDraftKey(userID: number, kind: string, contentID: number): string {
  return `stellar-beacon:studio-draft:v1:${userID}:${kind}:${contentID > 0 ? contentID : 'new'}`
}

export function readStudioDraft<T>(key: string): StudioDraftSnapshot<T> | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return null
    const value = JSON.parse(raw) as StudioDraftSnapshot<T>
    if (value?.version !== 1 || !value.data) return null
    return value
  } catch {
    return null
  }
}

export function saveStudioDraft<T>(key: string, data: T): void {
  if (typeof localStorage === 'undefined') return
  const value: StudioDraftSnapshot<T> = {
    version: 1,
    savedAt: new Date().toISOString(),
    data
  }
  localStorage.setItem(key, JSON.stringify(value))
}

export function clearStudioDraft(key: string): void {
  if (typeof localStorage === 'undefined') return
  localStorage.removeItem(key)
}

export function stableSnapshot(value: unknown): string {
  const normalize = (input: any): any => {
    if (Array.isArray(input)) return input.map(normalize)
    if (input && typeof input === 'object') {
      return Object.keys(input).sort().reduce<Record<string, unknown>>((result, key) => {
        result[key] = normalize(input[key])
        return result
      }, {})
    }
    return input
  }
  return JSON.stringify(normalize(value))
}