export function pageData(data: any): any {
  return data && data.data && typeof data.data === 'object' ? data.data : {}
}

export function pageRecords(data: any): any[] {
  const page = pageData(data)
  return Array.isArray(page.records) ? page.records : []
}

export function pageCount(data: any): number {
  const count = Number(pageData(data).count)
  return Number.isFinite(count) ? count : 0
}
