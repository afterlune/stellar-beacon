import { computed, onBeforeUnmount, onMounted, ref, type ComputedRef, type Ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'

/**
 * 编辑器「未保存修改」保护。
 *
 * 需要调用方在数据加载完成后（以及保存成功后）调用 `markClean()` 建立基线：
 * 只有用户真正改过内容才会拦截，否则像 E2E 那样只读打开再离开也会被弹窗打断。
 *
 * 两个拦截入口：
 * - `onBeforeRouteLeave`：站内跳转（点侧边栏、点返回列表）。
 * - `beforeunload`：刷新/关闭标签页。浏览器只在用户确实改过东西时提示。
 */
export function useUnsavedGuard(serialize: () => string): {
  dirty: ComputedRef<boolean>
  visible: Ref<boolean>
  markClean: () => void
  confirmLeave: () => void
  cancelLeave: () => void
} {
  const baseline = ref('')
  const ready = ref(false)
  const visible = ref(false)
  const dirty = computed(() => ready.value && serialize() !== baseline.value)

  let resolveLeave: ((value: boolean) => void) | null = null

  function markClean(): void {
    baseline.value = serialize()
    ready.value = true
  }

  function ask(): Promise<boolean> {
    visible.value = true
    return new Promise<boolean>((resolve) => {
      resolveLeave = resolve
    })
  }

  function settle(value: boolean): void {
    visible.value = false
    const resolve = resolveLeave
    resolveLeave = null
    resolve?.(value)
  }

  onBeforeRouteLeave(async () => {
    if (!dirty.value) return true
    return await ask()
  })

  function onBeforeUnload(event: BeforeUnloadEvent): void {
    if (!dirty.value) return
    event.preventDefault()
    // Chrome 需要 returnValue 才能显示原生确认框。
    event.returnValue = ''
  }

  onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))
  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', onBeforeUnload)
    // 组件卸载时若还挂着等待中的确认框，直接放行，避免路由守卫永远悬停。
    settle(true)
  })

  return {
    dirty,
    visible,
    markClean,
    confirmLeave: () => settle(true),
    cancelLeave: () => settle(false)
  }
}
