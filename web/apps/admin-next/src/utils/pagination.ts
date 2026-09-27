export interface TablePagination {
  current: number
  pageSize: number
  total: number
  showTotal: boolean
  showJumper: boolean
  showPageSize: boolean
  pageSizeOptions: number[]
}

const DEFAULT_PAGE_SIZES = [10, 20, 50, 100]

/**
 * Shared pagination contract for every admin table.
 *
 * Keeping this in one place means the page-size options, jumper and total
 * display stay identical across views instead of being re-invented per table.
 */
export function tablePagination(
  current: number,
  pageSize: number,
  total: number,
  pageSizeOptions: number[] = DEFAULT_PAGE_SIZES
): TablePagination {
  return {
    current: Math.max(1, current),
    pageSize: Math.max(1, pageSize),
    total: Math.max(0, total),
    showTotal: true,
    showJumper: true,
    showPageSize: true,
    pageSizeOptions
  }
}
