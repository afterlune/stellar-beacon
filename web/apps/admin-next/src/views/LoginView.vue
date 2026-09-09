<template>
  <main class="admin-login-page">
    <section class="admin-login-layout" aria-label="Benetnasch 管理台登录">
      <aside class="admin-login-art">
        <div class="admin-login-art-copy">
          <div class="admin-login-brand">
            <span class="admin-brand-mark" aria-hidden="true" />
            <span>Benetnasch 编辑台</span>
          </div>
          <div class="admin-home-eyebrow">A QUIET PLACE TO CREATE</div>
          <h1>把博客，留在清爽的秩序里。</h1>
          <p>文章、评论、图片与站点设置，都在一个安静而清晰的工作台里。</p>
          <div class="admin-login-note"><IconCheckCircle /> 内容、评论、站点一处管理</div>
        </div>
      </aside>

      <section class="admin-login-panel">
        <div class="admin-login-card" aria-labelledby="login-title">
          <div class="admin-login-heading">
            <div class="admin-page-eyebrow">WELCOME BACK</div>
            <h2 id="login-title">管理员登录</h2>
            <p>登录后继续整理你的博客内容。</p>
          </div>
          <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
          <a-form :model="form" layout="vertical" :disabled="auth.loading" @submit-success="submit">
            <a-form-item field="username" hide-label :rules="[{ required: true, message: '请输入邮箱' }]">
              <div class="admin-login-control">
                <label class="admin-login-label" for="admin-username">邮箱</label>
                <a-input
                  v-model="form.username"
                  :input-attrs="{ id: 'admin-username', name: 'username', 'aria-label': '管理员邮箱' }"
                  data-testid="login-username"
                  autocomplete="username"
                  placeholder="请输入管理员邮箱" />
              </div>
            </a-form-item>
            <a-form-item field="password" hide-label :rules="[{ required: true, message: '请输入密码' }]">
              <div class="admin-login-control">
                <label class="admin-login-label" for="admin-password">密码</label>
                <a-input-password
                  v-model="form.password"
                  :input-attrs="{ id: 'admin-password', name: 'password', 'aria-label': '密码' }"
                  data-testid="login-password"
                  autocomplete="current-password"
                  placeholder="请输入密码" />
              </div>
            </a-form-item>
            <a-button html-type="submit" type="primary" long :loading="auth.loading" data-testid="login-submit">
              登录编辑台
              <template #icon><IconArrowRight /></template>
            </a-button>
          </a-form>
          <div class="admin-login-footer">BENETNASCH / EDITORIAL ADMIN</div>
        </div>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconArrowRight, IconCheckCircle } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage } from '@/api/http'
import { resetMenuRoutes } from '@/router'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'

const auth = useAuthStore()
const menuStore = useMenuStore()
const route = useRoute()
const router = useRouter()
const form = reactive({ username: '', password: '' })
const errorMessage = ref('')

async function submit(): Promise<void> {
  errorMessage.value = ''
  try {
    await auth.login(form.username, form.password)
    resetMenuRoutes()
    menuStore.reset()
    await menuStore.load()
    await router.replace(safeRedirect(route.query.redirect))
    Message.success('登录成功')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '登录失败，请检查账号和密码')
  }
}

function safeRedirect(value: unknown): string {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//') ? value : '/'
}
</script>
