<template>
  <section class="subscription-action">
    <div class="subscription-action__panel">
      <p class="subscription-action__eyebrow">STELLAR BEACON / {{ mode === 'confirm' ? 'CONFIRM' : 'UNSUBSCRIBE' }}</p>
      <h1>{{ mode === 'confirm' ? t('newsletter.confirmTitle') : t('newsletter.unsubscribeTitle') }}</h1>
      <p>{{ status === 'loading' ? t('newsletter.processing') : status === 'success' ? t('newsletter.done') : t('newsletter.failed') }}</p>
      <router-link to="/">{{ t('settings.tips-back-to-home') }}</router-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import api from '@/api/api'

const props = defineProps<{ mode: 'confirm' | 'unsubscribe' }>()
const { t } = useI18n()
const route = useRoute()
const status = ref<'loading' | 'success' | 'failed'>('loading')

onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) { status.value = 'failed'; return }
  try {
    const response = props.mode === 'confirm'
      ? await api.confirmNewsletter({ token })
      : await api.unsubscribeNewsletter({ token })
    status.value = response.data.flag ? 'success' : 'failed'
    if (response.data.flag) void api.trackGrowthEvent({ eventName: props.mode === 'confirm' ? 'subscribe_confirm' : 'unsubscribe', path: window.location.pathname })
  } catch {
    status.value = 'failed'
  }
})
</script>

<style scoped lang="scss">
.subscription-action { min-height: 60vh; display: grid; place-items: center; padding: 8rem 1rem 4rem; }
.subscription-action__panel { width: min(620px, 100%); padding: 3rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 24%, transparent); border-radius: 1.5rem; background: var(--background-primary); box-shadow: 0 24px 80px rgb(0 0 0 / 12%); }
.subscription-action__eyebrow { margin: 0 0 1rem; color: var(--color-ob); font-size: .72rem; font-weight: 700; letter-spacing: .18em; }
.subscription-action h1 { margin: 0 0 1rem; font-family: var(--font-display); font-size: clamp(2rem, 6vw, 4rem); }
.subscription-action p { margin: 0 0 2rem; opacity: .72; }
.subscription-action a { color: var(--color-ob); font-weight: 700; }
</style>
