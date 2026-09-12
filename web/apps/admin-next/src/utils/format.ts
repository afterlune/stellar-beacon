/**
 * Display formatting helpers.
 *
 * The API emits a mix of Go `time.Time` (RFC3339 with a `T` separator) and
 * pre-formatted strings. These helpers keep rendering deterministic so table
 * columns, cards and detail panels never leak `2026-08-29T10:00:00Z` to users.
 */

/** Matches `2026-08-29T10:00:00`, optionally followed by `.123` and/or `Z`/offset. */
const TIMESTAMP_PATTERN = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}(?::\d{2})?)(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?$/

const EMPTY = '—'

function parseTimestamp(value: unknown): { date: string; time: string } | null {
  if (typeof value !== 'string') return null
  const match = TIMESTAMP_PATTERN.exec(value.trim())
  if (!match) return null
  return { date: match[1], time: match[2] }
}

/**
 * Human-readable date + time for table cells and cross-references.
 * Falls back to the raw value when it is not a timestamp, and to `—` when empty.
 */
export function formatDateTime(value: unknown): string {
  if (value === null || value === undefined || value === '') return EMPTY
  const parsed = parseTimestamp(value)
  if (parsed) return `${parsed.date} ${parsed.time}`
  return String(value)
}

/** Date-only rendering, used where the time of day adds no information. */
export function formatDate(value: unknown): string {
  if (value === null || value === undefined || value === '') return EMPTY
  const parsed = parseTimestamp(value)
  if (parsed) return parsed.date
  return String(value)
}

/** Backwards-compatible alias: several legacy cells still call `formatTime`. */
export const formatTime = formatDateTime

/** Generic cell rendering: empty values become an em dash, timestamps are softened. */
export function formatCell(value: unknown): string {
  if (value === null || value === undefined || value === '') return EMPTY
  return formatDateTime(value)
}

/** Thousands-separated integer/float rendering for counters. */
export function formatNumber(value: unknown): string {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return EMPTY
  return new Intl.NumberFormat('zh-CN').format(numeric)
}

/** Byte sizes for the media library. */
export function formatFileSize(value: unknown): string {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return '未知大小'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

/** Truncates long values for compact cells while keeping the full text available. */
export function truncate(value: unknown, max = 80): string {
  const text = String(value ?? '').trim()
  if (text.length <= max) return text
  return `${text.slice(0, max - 1)}…`
}

/** Strips HTML so feed-like content renders as readable plain text. */
export function plainText(value: unknown): string {
  return String(value ?? '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

/** Only absolute http(s) URLs are treated as renderable remote images. */
export function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && /^https?:\/\//i.test(value)
}

/** First grapheme-ish character of a display name, for avatar fallbacks. */
export function initialOf(value: unknown, fallback = '管'): string {
  return String(value || fallback).trim().slice(0, 1).toUpperCase() || fallback
}
