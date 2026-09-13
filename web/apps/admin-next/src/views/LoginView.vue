<template>
  <main class="admin-login-page">
    <section class="admin-login-layout" aria-label="Benetnasch 管理台登录">
      <aside class="admin-login-art">
        <div class="admin-login-art-copy">
          <div class="admin-login-brand">
            <BrandMark />
            <span>Benetnasch 编辑台</span>
          </div>
          <div class="admin-home-eyebrow">A QUIET PLACE TO CREATE</div>
          <h1>把博客，留在清爽的秩序里。</h1>
          <p>文章、评论、图片与站点设置，都在一个安静而清晰的工作台里。</p>
          <ul class="admin-login-highlights">
            <li v-for="highlight in highlights" :key="highlight" class="admin-login-highlight">
              <span class="admin-login-highlight-icon" aria-hidden="true"><IconCheckCircle /></span>
              {{ highlight }}
            </li>
          </ul>
        </div>
      </aside>

      <section class="admin-login-panel">
        <div class="admin-login-card" aria-labelledby="login-title">
          <div class="admin-login-heading">
            <div class="admin-page-eyebrow">WELCOME BACK</div>
            <h2 id="login-title">管理员登录</h2>
            <p>使用你注册的邮箱与密码登录编辑台。</p>
          </div>

          <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
          <a-alert v-if="menuWarning" type="warning" closable @close="menuWarning = ''">{{ menuWarning }}</a-alert>
          <a-alert v-if="redirectHint" type="info">{{ redirectHint }}</a-alert>

          <a-form ref="formRef" :model="form" layout="vertical" :disabled="auth.loading" @submit-success="submit">
            <a-form-item
              field="username"
              hide-label
              :rules="[
                { required: true, message: '请输入邮箱' },
                { type: 'email', message: '邮箱格式不正确' }
              ]">
              <div class="admin-login-control">
                <label class="admin-login-label" for="admin-username">邮箱</label>
                <a-input
                  v-model="form.username"
                  :input-attrs="{ id: 'admin-username', name: 'username', 'aria-label': '管理员邮箱' }"
                  data-testid="login-username"
                  autocomplete="username"
                  size="large"
                  placeholder="请输入管理员邮箱" />
              </div>
            </a-form-item>

            <a-form-item
              field="password"
              hide-label
              :rules="[{ required: true, message: '请输入密码' }]">
              <div class="admin-login-control">
                <label class="admin-login-label" for="admin-password">密码</label>
                <a-input-password
                  v-model="form.password"
                  :input-attrs="{ id: 'admin-password', name: 'password', 'aria-label': '密码' }"
                  data-testid="login-password"
                  autocomplete="current-password"
                  size="large"
                  placeholder="请输入密码" />
              </div>
            </a-form-item>

            <a-button
              html-type="submit"
              type="primary"
              long
              size="large"
              :loading="auth.loading"
              data-testid="login-submit">
              登录编辑台
              <template #icon><IconArrowRight /></template>
            </a-button>
          </a-form>

          <p class="admin-login-hint">
            忘记密码？请联系站点管理员在数据库或部署脚本中重置。
          </p>
          <div class="admin-login-footer">BENETNASCH / EDITORIAL ADMIN</div>
        </div>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconArrowRight, IconCheckCircle } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage } from '@/api/http'
import BrandMark from '@/components/BrandMark.vue'
import { resetMenuRoutes } from '@/router'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { resetSessionExpiredNotice } from '@/utils/session-notice'

const highlights = [
  '文章、分类、标签统一管理',
  '评论审核与友链维护',
  '图片资源与相册集中托管',
  '站点配置与访问数据一屏掌握'
]

const auth = useAuthStore()
const menuStore = useMenuStore()
const route = useRoute()
const router = useRouter()

const form = reactive({ username: '', password: '' })
const errorMessage = ref('')
const menuWarning = ref('')
const formRef = ref<{ validate: () => Promise<Record<string, unknown> | undefined> } | null>(null)

/** 因会话过期或未登录被挡回登录页时，告诉用户登录后会回到原来的页面。 */
const redirectHint = computed(() => (safeRedirect(route.query.redirect) === '/'
  ? ''
  : '登录成功后会返回你刚才访问的页面。'))

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
    errorMessage.value = apiErrorMessage(error, '登录失败，请检查账号和密码')
    return
  }

  resetMenuRoutes()
  menuStore.reset()

  // Phase 2: menu loading. The session is already valid, so a menu failure must
  // not masquerade as a bad password — navigate and let the shell retry.
  try {
    await menuStore.load()
  } catch (error) {
    menuWarning.value = apiErrorMessage(error, '登录成功，但菜单加载失败，可在管理台内重试')
  }

  Message.success('登录成功')
  await router.replace(safeRedirect(route.query.redirect))
}

function safeRedirect(value: unknown): string {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//') ? value : '/'
}
</script>

<style scoped>
.admin-login-highlights {
  display: grid;
  gap: 10px;
  margin: 26px 0 0;
  padding: 0;
  list-style: none;
}

.admin-login-hint {
  margin: 18px 0 0;
  color: var(--admin-subtle);
  font-size: 12px;
  line-height: 1.6;
  text-align: center;
}
</style>
