<template>
  <main class="admin-login-page">
    <section class="admin-login-card" aria-labelledby="login-title">
      <div class="admin-login-heading">
        <h1 id="login-title">管理员登录</h1>
        <p>Benetnasch 内容与 Agent 控制台</p>
      </div>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-form :model="form" layout="vertical" :disabled="auth.loading" @submit-success="submit">
        <a-form-item field="username" label="邮箱" :rules="[{ required: true, message: '请输入邮箱' }]">
          <a-input
            v-model="form.username"
            data-testid="login-username"
            autocomplete="username"
            placeholder="请输入管理员邮箱" />
        </a-form-item>
        <a-form-item field="password" label="密码" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password
            v-model="form.password"
            data-testid="login-password"
            autocomplete="current-password"
            placeholder="请输入密码" />
        </a-form-item>
        <a-button html-type="submit" type="primary" long :loading="auth.loading" data-testid="login-submit">
          登录
        </a-button>
      </a-form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'

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
