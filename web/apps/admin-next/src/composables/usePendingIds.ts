import { ref, type Ref } from 'vue'

/**
 * 行内/批量动作的进行中状态。
 *
 * 同一个 id 只允许一个动作在飞：重复点击不会重复提交（此前每个视图各自用
 * `busyId`/`pendingTopId` 等单个数字跟踪，无法覆盖批量动作与并发行）。
 */
export function usePendingIds(): {
  pending: Ref<number[]>
  isPending: (id: unknown) => boolean
  begin: (id: unknown) => void
  end: (id: unknown) => void
  withPending: <T>(id: unknown, task: () => Promise<T>) => Promise<T | undefined>
  clear: () => void
} {
  const pending = ref<number[]>([])

  return { pending, isPending, begin, end, withPending, clear }

  function normalize(id: unknown): number {
    const value = Number(id)
    return Number.isFinite(value) ? value : Number.NaN
  }

  function isPending(id: unknown): boolean {
    const value = normalize(id)
    return Number.isFinite(value) && pending.value.includes(value)
  }

  function begin(id: unknown): void {
    const value = normalize(id)
    if (!Number.isFinite(value) || pending.value.includes(value)) return
    pending.value = [...pending.value, value]
  }

  function end(id: unknown): void {
    const value = normalize(id)
    pending.value = pending.value.filter((item) => item !== value)
  }

  function clear(): void {
    pending.value = []
  }

  /** 已有同 id 动作在飞时直接跳过，返回 `undefined`。 */
  async function withPending<T>(id: unknown, task: () => Promise<T>): Promise<T | undefined> {
    if (isPending(id)) return undefined
    begin(id)
    try {
      return await task()
    } finally {
      end(id)
    }
  }
}
