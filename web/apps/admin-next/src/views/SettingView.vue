<template>
  <section class="admin-page">
    <AdminPageHeader title="个人中心" description="维护你的后台身份信息和对外简介。" />
    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-form :model="form" layout="vertical" @submit-success="save">
        <a-form-item field="nickname" label="昵称"><a-input v-model="form.nickname" /></a-form-item>
        <a-form-item field="intro" label="简介"><a-textarea v-model="form.intro" :auto-size="{ minRows: 3, maxRows: 8 }" /></a-form-item>
        <a-form-item field="website" label="个人网站"><a-input v-model="form.website" /></a-form-item>
        <div class="admin-form-actions"><a-button type="primary" html-type="submit" :loading="saving">保存</a-button></div>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, updateUserProfile } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import AdminPageHeader from '@/components/AdminPageHeader.vue'

const auth = useAuthStore()
const form = reactive({ nickname: auth.user?.nickname || '', intro: auth.user?.intro || '', website: auth.user?.website || '' })
const saving = ref(false)
const errorMessage = ref('')
async function save(): Promise<void> {
  saving.value = true
  try {
    await updateUserProfile(form)
    auth.updateUser(form)
    Message.success('个人信息已保存')
  } catch (error) { errorMessage.value = apiErrorMessage(error, '个人信息保存失败') }
  finally { saving.value = false }
}
</script>
