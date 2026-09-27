<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="app-dialog"
      :class="`app-dialog--${props.variant}`"
      :aria-label="props.hideHeader ? props.ariaLabel || props.title : undefined"
      :aria-labelledby="props.hideHeader ? undefined : headingId"
      @cancel="onCancel"
      @close="onClose"
      @click="onDialogClick">
      <header v-if="!props.hideHeader" class="app-dialog__header">
        <slot name="header">
          <h2 :id="headingId">{{ props.title }}</h2>
        </slot>
        <button class="app-dialog__close" type="button" :aria-label="props.closeLabel" @click="requestClose">×</button>
      </header>
      <div class="app-dialog__content">
        <slot />
      </div>
    </dialog>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  variant?: 'modal' | 'drawer-right' | 'drawer-bottom'
  hideHeader?: boolean
  ariaLabel?: string
  closeLabel?: string
}>(), {
  variant: 'modal',
  hideHeader: false,
  ariaLabel: '',
  closeLabel: '关闭'
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  opened: []
  closed: []
}>()

const dialog = ref<HTMLDialogElement>()
const headingId = `dialog-title-${Math.random().toString(36).slice(2)}`
let bodyLocked = false
let activeBodyLocks = 0
let savedOverflow = ''
let savedPaddingRight = ''

function lockBodyScroll() {
  if (bodyLocked || typeof document === 'undefined') return
  bodyLocked = true
  activeBodyLocks += 1
  if (activeBodyLocks === 1) {
    savedOverflow = document.body.style.overflow
    savedPaddingRight = document.body.style.paddingRight
    const scrollbarGap = Math.max(0, window.innerWidth - document.documentElement.clientWidth)
    if (scrollbarGap) {
      const existingPadding = Number.parseFloat(savedPaddingRight) || 0
      document.body.style.paddingRight = `${existingPadding + scrollbarGap}px`
    }
    document.body.style.overflow = 'hidden'
  }
}

function unlockBodyScroll() {
  if (!bodyLocked || typeof document === 'undefined') return
  bodyLocked = false
  activeBodyLocks = Math.max(0, activeBodyLocks - 1)
  if (activeBodyLocks === 0) {
    document.body.style.overflow = savedOverflow
    document.body.style.paddingRight = savedPaddingRight
  }
}

function focusDialogContent() {
  const node = dialog.value
  const target = node?.querySelector<HTMLElement>('[autofocus]')
    || node?.querySelector<HTMLElement>('.app-dialog__content input:not([type="hidden"]), .app-dialog__content select, .app-dialog__content textarea, .app-dialog__content button, .app-dialog__content a[href], .app-dialog__content [tabindex]:not([tabindex="-1"])')
    || node?.querySelector<HTMLElement>('.app-dialog__close')
  target?.focus({ preventScroll: true })
}

watch(() => props.modelValue, async (visible) => {
  await nextTick()
  const node = dialog.value
  if (!node) return
  if (visible && !node.open) {
    lockBodyScroll()
    node.showModal()
    emit('opened')
    await nextTick()
    focusDialogContent()
  } else if (!visible && node.open) {
    node.close()
    unlockBodyScroll()
  }
}, { immediate: true, flush: 'post' })

function requestClose() {
  emit('update:modelValue', false)
}

function onCancel(event: Event) {
  event.preventDefault()
  requestClose()
}

function onClose() {
  unlockBodyScroll()
  emit('closed')
  if (props.modelValue) emit('update:modelValue', false)
}

function onDialogClick(event: MouseEvent) {
  if (event.target === dialog.value) requestClose()
}

onBeforeUnmount(() => {
  unlockBodyScroll()
})
</script>

<style>
dialog.app-dialog {
  position: fixed;
  inset: 0;
  width: min(460px, calc(100vw - 32px));
  max-width: none;
  max-height: min(88dvh, 900px);
  margin: auto;
  padding: 0;
  overflow: auto;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent);
  border-radius: 22px;
  background: var(--background-primary);
  color: var(--text-normal);
  box-shadow: 0 24px 90px rgba(7, 12, 28, .34);
  overscroll-behavior: contain;
}

dialog.app-dialog::backdrop {
  background: rgba(9, 15, 32, .48);
  backdrop-filter: blur(5px);
}

.app-dialog__header {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 58px;
  padding: 15px 20px;
  border-bottom: 1px solid color-mix(in srgb, var(--text-ob-dim) 16%, transparent);
  background: var(--background-primary);
}

.app-dialog__header h2 {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
}

.app-dialog__close {
  display: inline-grid;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 0;
  border-radius: 50%;
  background: color-mix(in srgb, var(--text-ob-dim) 12%, transparent);
  color: inherit;
  font-size: 23px;
  line-height: 1;
  cursor: pointer;
}

.app-dialog__close:hover { background: color-mix(in srgb, var(--text-ob-dim) 22%, transparent); }
.app-dialog__content { padding: 20px; }

dialog.app-dialog--drawer-right {
  inset: 0 0 0 auto;
  width: min(430px, 100vw);
  height: 100dvh;
  max-height: 100dvh;
  margin: 0;
  border-radius: 22px 0 0 22px;
  animation: app-drawer-right-in 180ms ease-out;
}

dialog.app-dialog--drawer-right .app-dialog__content { min-height: 100%; }

dialog.app-dialog--drawer-bottom {
  inset: auto 0 0;
  width: 100%;
  height: min(78dvh, 900px);
  max-height: 90dvh;
  margin: 0 auto;
  border-radius: 22px 22px 0 0;
  animation: app-drawer-bottom-in 180ms ease-out;
}

@keyframes app-drawer-right-in { from { transform: translateX(28px); opacity: .8; } to { transform: translateX(0); opacity: 1; } }
@keyframes app-drawer-bottom-in { from { transform: translateY(28px); opacity: .8; } to { transform: translateY(0); opacity: 1; } }

@media (max-width: 640px) {
  dialog.app-dialog--modal { width: 100vw; max-height: 100dvh; border-radius: 18px 18px 0 0; }
  dialog.app-dialog--drawer-right { width: 100vw; border-radius: 0; }
}

@media (prefers-reduced-motion: reduce) {
  dialog.app-dialog--drawer-right, dialog.app-dialog--drawer-bottom { animation: none; }
}
</style>
