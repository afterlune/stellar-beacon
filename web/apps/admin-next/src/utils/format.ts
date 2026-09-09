export function formatCell(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'string' && value.includes('T')) return value.replace('T', ' ').replace(/\.\d+Z$/, '')
  return String(value)
}

export function formatTime(value: unknown): string {
  return typeof value === 'string' ? value.replace('T', ' ').replace(/\.\d+Z$/, '') : '—'
}
