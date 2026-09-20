export interface NormalizedArticleTag {
  id: string
  tagName: string
}

/**
 * Article payloads have been observed in three shapes:
 * `["python"]`, `"nginx"` and `[{ id, tagName }]`.
 * Normalize them into a stable list so cards and headers never render a bare `#`.
 */
export function normalizeArticleTags(value: unknown): NormalizedArticleTag[] {
  const source = Array.isArray(value) ? value : typeof value === 'string' ? [value] : []
  const seen = new Set<string>()
  const result: NormalizedArticleTag[] = []
  for (const item of source) {
    let name = ''
    let id = ''
    if (typeof item === 'string') {
      name = item
    } else if (item && typeof item === 'object') {
      const record = item as Record<string, unknown>
      name = String(record.tagName ?? record.name ?? '')
      id = String(record.id ?? record.tagId ?? '')
    }
    name = name.trim()
    if (!name) continue
    const key = name.toLocaleLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    result.push({ id: id || name, tagName: name })
  }
  return result
}
