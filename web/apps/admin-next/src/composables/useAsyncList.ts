import { onMounted, onScopeDispose, ref, type Ref } from 'vue'

import { apiErrorMessage } from '@/api/http'

/** 一页数据：与 `listAdminPage` 的返回结构兼容。 */
export interface AsyncListPage<T> {
  items: T[]
  total: number
}

export interface AsyncListContext {
  current: number
  pageSize: number
  signal: AbortSignal
}

export interface AsyncListOptions {
  /** 每页条数初值（会被持久化偏好覆盖，见 `useStoredPageSize`）。 */
  pageSize?: number
  /** 加载失败时的兜底文案。 */
  fallbackMessage?: string
  /** 是否在挂载时自动加载首页，默认 true。 */
  immediate?: boolean
}

/**
 * 列表页统一的异步状态机。
 *
 * 解决三个此前每个视图各自实现、且实现不一致的问题：
 *
 * 1. **竞态**：快速切换筛选/分页时，先发出的请求可能后返回并覆盖新数据。
 *    这里用自增 `seq` 只允许最新一次请求写入状态。
 * 2. **取消**：发起新请求或组件卸载时中止上一个请求，避免无谓的网络与状态写入。
 * 3. **错误态**：失败只写入 `error`，由视图渲染带「重试」的错误块；不再同时弹
 *    Message（重复报错且打断操作）。
 */
export function useAsyncList<T>(
  fetcher: (context: AsyncListContext) => Promise<AsyncListPage<T>>,
  options: AsyncListOptions = {}
) {
  const items = ref([]) as Ref<T[]>
  const total = ref(0)
  const current = ref(1)
  const pageSize = ref(Math.max(1, options.pageSize ?? 10))
  const loading = ref(false)
  const error = ref('')
  const hasLoaded = ref(false)

  const fallbackMessage = options.fallbackMessage ?? '数据加载失败'

  let seq = 0
  let controller: AbortController | null = null

  async function load(): Promise<void> {
    const token = ++seq
    controller?.abort()
    controller = new AbortController()
    const signal = controller.signal

    loading.value = true
    error.value = ''
    try {
      const page = await fetcher({ current: current.value, pageSize: pageSize.value, signal })
      if (token !== seq) return
      items.value = page.items
      total.value = page.total
      hasLoaded.value = true
    } catch (cause) {
      if (token !== seq || isAbort(cause)) return
      error.value = apiErrorMessage(cause, fallbackMessage)
    } finally {
      if (token === seq) loading.value = false
    }
  }

  /** 回到第一页并重新加载（筛选条件变化时的默认行为）。 */
  async function reload(): Promise<void> {
    current.value = 1
    await load()
  }

  function changePage(page: number): void {
    current.value = Math.max(1, page)
    void load()
  }

  function changePageSize(size: number): void {
    pageSize.value = Math.max(1, size)
    current.value = 1
    void load()
  }

  /** 清空当前数据（退出登录、切换资源上下文等场景）。 */
  function reset(): void {
    seq += 1
    controller?.abort()
    controller = null
    items.value = []
    total.value = 0
    current.value = 1
    error.value = ''
    loading.value = false
    hasLoaded.value = false
  }

  onMounted(() => {
    if (options.immediate !== false) void load()
  })

  onScopeDispose(() => {
    seq += 1
    controller?.abort()
    controller = null
  })

  return {
    items,
    total,
    current,
    pageSize,
    loading,
    error,
    hasLoaded,
    load,
    reload,
    changePage,
    changePageSize,
    reset
  }
}

/**
 * 非列表场景的「最新请求胜出」保护（例如仪表盘的区间切换）。
 *
 * 与 `useAsyncList` 共用同一套取消/序号语义，但不持有列表状态。
 */
export function useLatestRequest() {
  let seq = 0
  let controller: AbortController | null = null

  async function run<T>(task: (signal: AbortSignal) => Promise<T>): Promise<T | undefined> {
    const token = ++seq
    controller?.abort()
    controller = new AbortController()
    try {
      const value = await task(controller.signal)
      return token === seq ? value : undefined
    } catch (cause) {
      if (isAbort(cause)) return undefined
      throw cause
    }
  }

  onScopeDispose(() => {
    seq += 1
    controller?.abort()
    controller = null
  })

  return { run }
}

function isAbort(cause: unknown): boolean {
  return Boolean(cause && typeof cause === 'object' && 'code' in cause && (cause as { code?: string }).code === 'ERR_CANCELED')
}
