export interface TablePagination {
  current: number
  pageSize: number
  total: number
  showTotal: boolean
  showJumper: boolean
  showPageSize: boolean
}

export function tablePagination(current: number, pageSize: number, total: number): TablePagination {
  return { current, pageSize, total, showTotal: true, showJumper: true, showPageSize: true }
}
