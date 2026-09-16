<template>
  <main class="admin-login-page">
    <div class="admin-login-controls">
      <AdminShellControls labelled />
    </div>

    <section class="admin-login-layout" :aria-label="t('login.pageLabel')">
      <aside class="admin-login-art">
        <!-- 这一栏只留品牌：标题、介绍、四条卖点都是界面上的噪音，
             版面交给星空与北斗，文字只剩必要的签名。 -->
        <div class="admin-login-art-copy">
          <div class="admin-login-brand">
            <BrandMark />
            <span>{{ t('login.brand') }}</span>
          </div>
        </div>
      </aside>

      <section class="admin-login-panel">
        <div class="admin-login-card" aria-labelledby="login-title">
          <div class="admin-login-heading">
            <h2 id="login-title">{{ t('login.title') }}</h2>
          </div>

          <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
          <a-alert v-if="menuWarning" type="warning" closable @close="menuWarning = ''">{{ menuWarning }}</a-alert>
          <a-alert v-if="redirectHint" type="info">{{ redirectHint }}</a-alert>

          <a-form ref="formRef" :model="form" layout="vertical" :disabled="auth.loading" @submit-success="submit">
            <a-form-item field="username" hide-label :rules="usernameRules">
              <div class="admin-login-control">
                <label class="admin-login-label" for="admin-username">{{ t('login.email') }}</label>
                <a-input
                  v-model="form.username"
                  :input-attrs="{ id: 'admin-username', name: 'username', 'aria-label': t('login.emailAria') }"
                  data-testid="login-username"
                  autocomplete="username"
                  size="large"
                  :placeholder="t('login.emailPlaceholder')" />
              </div>
            </a-form-item>

            <a-form-item field="password" hide-label :rules="passwordRules">
              <div class="admin-login-control">
                <label class="admin-login-label" for="admin-password">{{ t('login.password') }}</label>
                <a-input-password
                  v-model="form.password"
                  :input-attrs="{ id: 'admin-password', name: 'password', 'aria-label': t('login.password') }"
                  data-testid="login-password"
                  autocomplete="current-password"
                  size="large"
                  :placeholder="t('login.passwordPlaceholder')" />
              </div>
            </a-form-item>

            <a-button
              html-type="submit"
              type="primary"
              long
              size="large"
              :loading="auth.loading"
              data-testid="login-submit">
              {{ t('login.submit') }}
              <template #icon><IconArrowRight /></template>
            </a-button>
          </a-form>

          <p class="admin-login-hint">{{ t('login.forgot') }}</p>
        </div>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconArrowRight } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage } from '@/api/http'
import AdminShellControls from '@/components/AdminShellControls.vue'
import BrandMark from '@/components/BrandMark.vue'
import { t } from '@/i18n'
import { resetMenuRoutes } from '@/router'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { resetSessionExpiredNotice } from '@/utils/session-notice'

const auth = useAuthStore()
const menuStore = useMenuStore()
const route = useRoute()
const router = useRouter()

const form = reactive({ username: '', password: '' })
const errorMessage = ref('')
const menuWarning = ref('')
const formRef = ref<{ validate: () => Promise<Record<string, unknown> | undefined> } | null>(null)

// 校验提示由 t() 生成，语言切换后重新提交即生效（已渲染的旧提示会保留上一次语言）。
const usernameRules = computed(() => [
  { required: true, message: t('login.emailRequired') },
  { type: 'email' as const, message: t('login.emailInvalid') }
])
const passwordRules = computed(() => [{ required: true, message: t('login.passwordRequired') }])

/** 因会话过期或未登录被挡回登录页时，告诉用户登录后会回到原来的页面。 */
const redirectHint = computed(() => (safeRedirect(route.query.redirect) === '/'
  ? ''
  : t('login.redirectHint')))

async function submit(): Promise<void> {
  const errors = await formRef.value?.validate()
  if (errors) return

  errorMessage.value = ''
  menuWarning.value = ''

  // Phase 1: credential check. Failures here stay on the login page.
  try {
    await auth.login(form.username, form.password)
    // 新会话开始，允许下一次过期重新提示。
    resetSessionExpiredNotice()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('login.failed'))
    return
  }

  resetMenuRoutes()
  menuStore.reset()

  // Phase 2: menu loading. The session is already valid, so a menu failure must
  // not masquerade as a bad password — navigate and let the shell retry.
  try {
    await menuStore.load()
  } catch (error) {
    menuWarning.value = apiErrorMessage(error, t('login.menuFailed'))
  }

  Message.success(t('login.success'))
  await router.replace(safeRedirect(route.query.redirect))
}

function safeRedirect(value: unknown): string {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//') ? value : '/'
}
</script>

<style scoped>
.admin-login-hint {
  margin: 18px 0 0;
  color: var(--admin-subtle);
  font-size: 12px;
  line-height: 1.6;
  text-align: center;
}
</style>
