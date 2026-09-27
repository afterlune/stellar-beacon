import { computed, onBeforeUnmount, onMounted, ref, type ComputedRef, type Ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'

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
    event.returnValue = ''
  }

  onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))
  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', onBeforeUnload)
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