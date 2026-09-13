import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'

const STORAGE_PREFIX = 'stellar-beacon.admin.table.'
const LEGACY_STORAGE_PREFIX = 'benetnasch.admin.table.'

interface TablePreferences {
  pageSize?: number
  hiddenColumns?: string[]
}

/**
 * 表格偏好（每页条数、隐藏列）落在 localStorage。
 *
 * 与 `stores/theme.ts` 相同的容错策略：隐私模式或损坏的 JSON 都不能让页面报错，
 * 读取失败一律回退到默认值。
 */
function read(viewKey: string): TablePreferences {
  try {
    const key = STORAGE_PREFIX + viewKey
    let raw = localStorage.getItem(key)
    if (raw === null) {
      raw = localStorage.getItem(LEGACY_STORAGE_PREFIX + viewKey)
      if (raw !== null) localStorage.setItem(key, raw)
    }
    if (!raw) return {}
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') return {}
    const value = parsed as TablePreferences
    return {
      pageSize: Number.isFinite(Number(value.pageSize)) ? Number(value.pageSize) : undefined,
      hiddenColumns: Array.isArray(value.hiddenColumns) ? value.hiddenColumns.map(String) : undefined
    }
  } catch {
    return {}
  }
}

function write(viewKey: string, patch: Partial<TablePreferences>): void {
  try {
    const next = { ...read(viewKey), ...patch }
    localStorage.setItem(STORAGE_PREFIX + viewKey, JSON.stringify(next))
  } catch {
    /* 隐私模式：只保留内存中的偏好 */
  }
}

/** 同步读取持久化的每页条数（在首次请求发出前调用）。 */
export function readStoredPageSize(viewKey: string, fallback = 10): number {
  const stored = read(viewKey).pageSize
  return stored && stored > 0 ? stored : fallback
}

/** 把 `useAsyncList` 的 `pageSize` 与持久化偏好双向同步。 */
export function useStoredPageSize(viewKey: string, pageSize: Ref<number>): void {
  const stored = read(viewKey).pageSize
  if (stored && stored > 0) pageSize.value = stored
  watch(pageSize, (value) => write(viewKey, { pageSize: value }))
}

export interface ColumnPreferences {
  hiddenKeys: Ref<string[]>
  hiddenCount: ComputedRef<number>
  isVisible: (key: string) => boolean
  toggle: (key: string, visible: boolean) => void
  reset: () => void
}

/**
 * 列显示偏好。
 *
 * 约束：任何时刻至少保留一列可见——否则表格会退化成空白，且 `:columns` 为空时
 * Arco 会渲染出自相矛盾的骨架，用户很难自己恢复。
 */
export function useColumnPrefs(viewKey: string, keys: string[], protectedKeys: string[] = ['actions']): ColumnPreferences {
  const hiddenKeys = ref<string[]>(normalize(read(viewKey).hiddenColumns))

  const visibleCount = computed(() => keys.filter((key) => !hiddenKeys.value.includes(key)).length)
  const hiddenCount = computed(() => hiddenKeys.value.filter((key) => keys.includes(key)).length)

  watch(hiddenKeys, (value) => write(viewKey, { hiddenColumns: normalize(value) }), { deep: true })

  return {
    hiddenKeys,
    hiddenCount,
    isVisible,
    toggle,
    reset
  }

  function isVisible(key: string): boolean {
    return !hiddenKeys.value.includes(key)
  }

  function toggle(key: string, visible: boolean): void {
    if (protectedKeys.includes(key)) return
    if (visible) {
      hiddenKeys.value = hiddenKeys.value.filter((value) => value !== key)
      return
    }
    if (visibleCount.value <= 1) return
    if (!hiddenKeys.value.includes(key)) hiddenKeys.value = [...hiddenKeys.value, key]
  }

  function reset(): void {
    hiddenKeys.value = []
  }

  /** 只保留仍然存在于当前列集合里的键，避免改版后残留脏数据。 */
  function normalize(values: string[] | undefined): string[] {
    if (!values) return []
    return values.filter((value) => keys.includes(value) && !protectedKeys.includes(value))
  }
}
