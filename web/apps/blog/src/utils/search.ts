import type { ArticleSearchResult, Page } from '@stellar-beacon/api-contract'

export function normalizeSearchPage(value: unknown): Page<ArticleSearchResult> {
  const source = value && typeof value === 'object' ? value as Record<string, unknown> : {}
  const items = Array.isArray(source.items)
    ? source.items as ArticleSearchResult[]
    : Array.isArray(source.records)
      ? source.records as ArticleSearchResult[]
      : []
  const total = Number(source.total ?? source.count ?? items.length)
  const page = Math.max(1, Number(source.page ?? 1) || 1)
  const pageSize = Math.max(0, Number(source.pageSize ?? items.length) || 0)
  return {
    items,
    total: Number.isFinite(total) ? total : 0,
    page,
    pageSize,
    records: items,
    count: Number.isFinite(total) ? total : 0
  }
}

export function safeSearchHighlight(value: unknown): string {
  const escaped = String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
  return escaped
    .replace(/&lt;mark&gt;/g, '<mark>')
    .replace(/&lt;\/mark&gt;/g, '</mark>')
}