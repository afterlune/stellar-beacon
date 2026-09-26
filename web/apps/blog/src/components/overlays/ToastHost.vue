<template>
  <Teleport to="body">
    <div class="toast-stack" aria-live="polite" aria-relevant="additions text">
      <article v-for="notice in notices" :key="notice.id" class="toast-card" :class="`toast-card--${notice.tone}`" :role="notice.tone === 'error' ? 'alert' : 'status'">
        <span class="toast-card__mark" aria-hidden="true">{{ marks[notice.tone] }}</span>
        <div class="toast-card__copy">
          <strong v-if="notice.title">{{ notice.title }}</strong>
          <p v-if="notice.message">{{ notice.message }}</p>
        </div>
        <button type="button" aria-label="关闭提示" @click="dismissNotice(notice.id)">×</button>
      </article>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useNotices, type NoticeTone } from '@/services/notifications'

const { notices, dismissNotice } = useNotices()
const marks: Record<NoticeTone, string> = { success: '✓', info: 'i', warning: '!', error: '×' }
</script>

<style>
.toast-stack {
  position: fixed;
  z-index: 2147483000;
  top: max(18px, env(safe-area-inset-top));
  right: max(18px, env(safe-area-inset-right));
  display: grid;
  width: min(380px, calc(100vw - 32px));
  gap: 10px;
  pointer-events: none;
}
.toast-card {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) 26px;
  align-items: start;
  gap: 10px;
  padding: 13px 12px;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 25%, transparent);
  border-radius: 14px;
  background: var(--background-primary);
  color: var(--text-normal);
  box-shadow: 0 12px 42px rgba(7, 12, 28, .2);
  pointer-events: auto;
  animation: toast-appear 160ms ease-out;
}
.toast-card__mark { display: grid; width: 26px; height: 26px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--color-ob) 20%, transparent); color: var(--color-ob); font-weight: 800; }
.toast-card--success .toast-card__mark { color: #31a66a; background: rgba(49, 166, 106, .13); }
.toast-card--warning .toast-card__mark { color: #bb7d13; background: rgba(221, 163, 56, .16); }
.toast-card--error .toast-card__mark { color: #d25050; background: rgba(210, 80, 80, .14); }
.toast-card__copy { min-width: 0; }
.toast-card__copy strong { display: block; margin: 1px 0 3px; font-size: 13px; }
.toast-card__copy p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.5; overflow-wrap: anywhere; }
.toast-card > button { width: 26px; height: 26px; border: 0; border-radius: 50%; background: transparent; color: inherit; font-size: 19px; cursor: pointer; }
.toast-card > button:hover { background: color-mix(in srgb, var(--text-ob-dim) 15%, transparent); }
@keyframes toast-appear { from { transform: translateY(-6px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
@media (prefers-reduced-motion: reduce) { .toast-card { animation: none; } }
</style>
