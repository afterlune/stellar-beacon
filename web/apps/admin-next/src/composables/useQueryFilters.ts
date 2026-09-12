import { onScopeDispose, watch, type Ref } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'

export interface QueryFilterSpec {
  /** URL 参数名。 */
  key: string
  /** 绑定的控件状态。 */
  ref: Ref<unknown>
  /** 默认值：等于默认值的取值不会写进 URL，保证默认态地址与改造前完全一致。缺省时取控件初值。 */
  defaultValue?: unknown
  /** 文本输入类筛选：写 URL 前做 300ms 防抖。 */
  debounce?: boolean
}

interface QueryFilterEntry extends QueryFilterSpec {
  fallback: unknown
}

const WRITE_DELAY = 300

/**
 * 把列表筛选条件同步到 URL query。
 *
 * 约定（E2E 契约要求）：**只写非默认值**。任何等于默认值、空串、`undefined`/`null`
 * 的取值都会从 query 中剔除，因此默认态的地址仍然是 `/article-list` 而不是
 * `/article-list?status=0`——测试里的 `toHaveURL(/\/article-list$/)` 之类的行尾锚点
 * 必须继续成立。
 *
 * 本组件不管理的 query（例如 `/quartz/log/:id` 这类由路由参数派生的键）一律原样保留。
 */
export function useQueryFilters(
  specs: QueryFilterSpec[],
  options: { onRestore?: () => void; onSearch?: () => void } = {}
): { syncNow: () => void } {
  const route = useRoute()
  const router = useRouter()

  // 控件初值即隐含默认值：调用方不用为每个筛选都重复写一遍 defaultValue。
  const entries: QueryFilterEntry[] = specs.map((spec) => ({
    ...spec,
    fallback: spec.defaultValue !== undefined ? spec.defaultValue : spec.ref.value
  }))

  let timer: number | undefined
  let lastWritten = ''

  // 1. 初始状态来自 URL：在首次请求发出前（onMounted）同步完成。
  applyFromQuery()

  // 2. 控件变化 → 写回 URL。
  for (const entry of entries) {
    watch(entry.ref, () => schedule(entry.debounce === true))
  }

  // 3. 浏览器前进/后退 → 回写控件，并通知调用方重新加载。
  //    自己写入的 query（`incoming === lastWritten`）不算外部变化，否则每次输入
  //    都会多打一次接口。
  watch(() => route.query, () => {
    const external = serialize(route.query) !== lastWritten
    applyFromQuery()
    lastWritten = serialize(desiredQuery())
    if (external) options.onRestore?.()
  })

  onScopeDispose(() => {
    if (timer !== undefined) window.clearTimeout(timer)
  })

  return { syncNow: write }

  function schedule(debounced: boolean): void {
    if (timer !== undefined) window.clearTimeout(timer)
    if (debounced) {
      // 文本输入：等用户停手再同时「写 URL + 查一次」，避免每敲一个字打一次接口。
      timer = window.setTimeout(() => {
        write()
        options.onSearch?.()
      }, WRITE_DELAY)
      return
    }
    // 下拉/单选类由模板的 @change 自己触发查询，这里只负责同步地址栏。
    write()
  }

  function write(): void {
    const next = desiredQuery()
    const serialized = serialize(next)
    if (serialized === serialize(route.query) || serialized === lastWritten) return
    lastWritten = serialized
    void router.replace({ query: next })
  }

  function desiredQuery(): LocationQuery {
    const next: LocationQuery = { ...route.query }
    for (const entry of entries) {
      const value = entry.ref.value
      if (isEmpty(value) || isDefault(value, entry.fallback)) delete next[entry.key]
      else next[entry.key] = String(value)
    }
    return next
  }

  function applyFromQuery(): void {
    for (const entry of entries) {
      const raw = first(route.query[entry.key])
      if (raw === undefined) {
        entry.ref.value = entry.fallback
        continue
      }
      const value = coerce(raw, entry.ref.value, entry.fallback)
      if (value !== undefined) entry.ref.value = value
    }
  }

  function serialize(query: LocationQuery): string {
    return Object.keys(query)
      .filter((key) => first(query[key]) !== undefined)
      .sort()
      .map((key) => `${key}=${String(first(query[key]))}`)
      .join('&')
  }
}

function first(value: unknown): string | undefined {
  if (Array.isArray(value)) return value.length > 0 ? String(value[0]) : undefined
  if (value === null || value === undefined) return undefined
  return String(value)
}

function isEmpty(value: unknown): boolean {
  return value === undefined || value === null || (typeof value === 'string' && value.trim() === '')
}

function isDefault(value: unknown, fallback: unknown): boolean {
  if (fallback === undefined || fallback === null) return false
  return String(value) === String(fallback)
}

/** 按控件当前类型（或默认值类型）还原 URL 中的字符串。 */
function coerce(raw: string, current: unknown, fallback: unknown): unknown {
  const template = !isEmpty(current) ? current : fallback
  if (typeof template === 'number') {
    const parsed = Number(raw)
    return Number.isFinite(parsed) ? parsed : undefined
  }
  if (typeof template === 'boolean') return raw === 'true'
  return raw
}
