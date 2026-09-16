<template>
  <section class="newsletter-card" aria-labelledby="newsletter-title">
    <div class="newsletter-card__copy">
      <p class="newsletter-card__eyebrow">{{ t('newsletter.eyebrow') }}</p>
      <h2 id="newsletter-title">{{ t('newsletter.title') }}</h2>
      <p>{{ t('newsletter.description') }}</p>
    </div>
    <form class="newsletter-card__form" @submit.prevent="submit">
      <label class="sr-only" for="newsletter-email">{{ t('newsletter.placeholder') }}</label>
      <input id="newsletter-email" v-model="email" type="email" autocomplete="email" required :placeholder="t('newsletter.placeholder')" />
      <button type="submit" :disabled="loading">
        {{ loading ? t('newsletter.submitting') : t('newsletter.submit') }}
      </button>
      <p v-if="message" class="newsletter-card__message" :class="{ 'is-error': failed }">{{ message }}</p>
    </form>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api/api'

const { t } = useI18n()
const email = ref('')
const loading = ref(false)
const message = ref('')
const failed = ref(false)

async function submit(): Promise<void> {
  if (!email.value.trim() || loading.value) return
  loading.value = true
  message.value = ''
  failed.value = false
  void api.trackGrowthEvent({ eventName: 'subscribe_start', path: window.location.pathname })
  try {
    const response = await api.subscribeNewsletter({ email: email.value.trim() })
    const result = response.data
    if (result.flag) {
      message.value = t('newsletter.success')
      email.value = ''
    } else {
      failed.value = true
      message.value = result.message || t('newsletter.error')
    }
  } catch {
    failed.value = true
    message.value = t('newsletter.error')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.newsletter-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 0.9fr);
  gap: 2rem;
  align-items: end;
  margin: 3rem 0;
  padding: 2rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 22%, transparent);
  border-radius: 1.25rem;
  background: linear-gradient(135deg, color-mix(in srgb, var(--background-primary) 86%, #6de0d4), var(--background-primary));
}
.newsletter-card__eyebrow { margin: 0 0 .6rem; color: var(--color-ob); font-size: .72rem; font-weight: 700; letter-spacing: .18em; }
.newsletter-card h2 { margin: 0 0 .7rem; font-family: var(--font-display); font-size: clamp(1.35rem, 2vw, 2rem); }
.newsletter-card p { margin: 0; line-height: 1.7; opacity: .76; }
.newsletter-card__form { display: grid; grid-template-columns: 1fr auto; gap: .65rem; }
.newsletter-card__form input { min-width: 0; padding: .85rem 1rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 35%, transparent); border-radius: .75rem; background: color-mix(in srgb, var(--background-primary) 80%, transparent); color: inherit; }
.newsletter-card__form button { padding: .85rem 1.2rem; border: 0; border-radius: .75rem; background: var(--text-ob); color: var(--background-primary); font-weight: 700; cursor: pointer; }
.newsletter-card__form button:disabled { opacity: .55; cursor: wait; }
.newsletter-card__message { grid-column: 1 / -1; font-size: .85rem; }
.newsletter-card__message.is-error { color: #e66a6a; }
@media (max-width: 800px) { .newsletter-card { grid-template-columns: 1fr; } }
</style>
