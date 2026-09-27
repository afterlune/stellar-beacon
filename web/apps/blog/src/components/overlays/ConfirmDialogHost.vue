<template>
  <DialogSurface
    :model-value="pendingConfirms.length > 0"
    :title="current?.title || '确认操作'"
    @update:model-value="(visible) => !visible && finishConfirm(false)">
    <div v-if="current" class="confirm-dialog" :class="`confirm-dialog--${current.tone}`">
      <span class="confirm-dialog__icon" aria-hidden="true">{{ current.tone === 'info' ? 'i' : '!' }}</span>
      <p>{{ current.message }}</p>
      <footer>
        <button type="button" class="confirm-dialog__cancel" @click="finishConfirm(false)">{{ current.cancelText }}</button>
        <button type="button" class="confirm-dialog__accept" @click="finishConfirm(true)">{{ current.confirmText }}</button>
      </footer>
    </div>
  </DialogSurface>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DialogSurface from './DialogSurface.vue'
import { finishConfirm, pendingConfirms } from '@/services/confirm'

const current = computed(() => pendingConfirms[0])
</script>

<style>
.confirm-dialog__icon { display: grid; width: 38px; height: 38px; place-items: center; border-radius: 50%; background: rgba(222, 165, 63, .16); color: #b27a18; font-size: 21px; font-weight: 800; }
.confirm-dialog--info .confirm-dialog__icon { background: color-mix(in srgb, var(--color-ob) 18%, transparent); color: var(--color-ob); }
.confirm-dialog p { margin: 14px 0 22px; line-height: 1.7; white-space: pre-line; overflow-wrap: anywhere; }
.confirm-dialog footer { display: flex; justify-content: flex-end; gap: 9px; }
.confirm-dialog footer button { min-height: 38px; padding: 7px 15px; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 26%, transparent); border-radius: 999px; background: transparent; color: inherit; font: inherit; cursor: pointer; }
.confirm-dialog footer .confirm-dialog__accept { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
</style>
