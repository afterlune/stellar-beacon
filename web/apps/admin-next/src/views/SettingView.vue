<template>
  <section class="admin-page">
    <AdminPageHeader title="个人中心" description="维护你的后台身份信息和对外简介。" />
    <div class="profile-hero">
      <div class="profile-avatar-wrap">
        <a-avatar :size="88" :image-url="auth.user?.avatar">{{ avatarText }}</a-avatar>
        <input ref="avatarInput" type="file" accept="image/*" hidden @change="selectAvatar" />
        <a-button size="small" :loading="avatarUploading" @click="avatarInput?.click()">更换头像</a-button>
      </div>
      <div class="profile-hero-copy">
        <div class="admin-page-eyebrow">ACCOUNT / PROFILE</div>
        <h3>{{ auth.user?.nickname || auth.user?.username || '管理员' }}</h3>
        <p>{{ form.intro || '还没有填写个人简介。补充一点信息，让个人中心真正成为你的后台身份名片。' }}</p>
        <a-space wrap>
          <a-tag color="arcoblue">{{ auth.user?.username || '管理员账号' }}</a-tag>
          <a-tag color="green">后台管理员</a-tag>
          <a-tag v-if="auth.user?.website" color="orange">{{ auth.user.website }}</a-tag>
        </a-space>
      </div>
      <div class="profile-meta">
        <span>注册时间</span><strong>{{ formatDate(auth.user?.createTime) }}</strong>
        <span>最近登录</span><strong>{{ formatDate(auth.user?.lastLoginTime) }}</strong>
        <span>登录邮箱</span><strong>{{ auth.user?.email || '未绑定' }}</strong>
      </div>
    </div>

    <div class="profile-grid">
      <a-card class="admin-form-panel admin-form-card" :bordered="false" title="公开资料">
        <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
        <a-form :model="form" layout="vertical" @submit-success="save">
          <a-form-item field="nickname" label="昵称"><a-input v-model="form.nickname" /></a-form-item>
          <a-form-item field="intro" label="个人简介"><a-textarea v-model="form.intro" :max-length="200" show-word-limit :auto-size="{ minRows: 5, maxRows: 8 }" /></a-form-item>
          <a-form-item field="website" label="个人网站"><a-input v-model="form.website" placeholder="https://" /></a-form-item>
          <div class="admin-form-actions"><a-button type="primary" html-type="submit" :loading="saving">保存资料</a-button></div>
        </a-form>
      </a-card>

      <a-card class="admin-form-panel admin-form-card" :bordered="false" title="账号安全">
        <a-alert v-if="passwordMessage" :type="passwordError ? 'error' : 'success'" closable @close="passwordMessage = ''">{{ passwordMessage }}</a-alert>
        <a-form :model="passwordForm" layout="vertical" @submit-success="changePassword">
          <a-form-item field="oldPassword" label="当前密码" required><a-input-password v-model="passwordForm.oldPassword" /></a-form-item>
          <a-form-item field="newPassword" label="新密码" required><a-input-password v-model="passwordForm.newPassword" /></a-form-item>
          <a-form-item field="confirmPassword" label="确认新密码" required><a-input-password v-model="passwordForm.confirmPassword" /></a-form-item>
          <div class="admin-form-actions"><a-button html-type="submit" :loading="passwordSaving">更新密码</a-button></div>
        </a-form>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, changeAdminPassword, updateUserProfile, uploadUserAvatar } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import AdminPageHeader from '@/components/AdminPageHeader.vue'

const auth = useAuthStore()
const form = reactive({ nickname: auth.user?.nickname || '', intro: auth.user?.intro || '', website: auth.user?.website || '' })
const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const saving = ref(false)
const passwordSaving = ref(false)
const avatarUploading = ref(false)
const errorMessage = ref('')
const passwordMessage = ref('')
const passwordError = ref(false)
const avatarInput = ref<HTMLInputElement | null>(null)
const avatarText = computed(() => (auth.user?.nickname || auth.user?.username || '管').slice(0, 1).toUpperCase())
async function save(): Promise<void> {
  saving.value = true
  try {
    await updateUserProfile(form)
    auth.updateUser(form)
    Message.success('个人信息已保存')
  } catch (error) { errorMessage.value = apiErrorMessage(error, '个人信息保存失败') }
  finally { saving.value = false }
}

async function changePassword(): Promise<void> {
  passwordMessage.value = ''
  passwordError.value = false
  if (passwordForm.newPassword.length < 6) { passwordError.value = true; passwordMessage.value = '新密码至少需要 6 位'; return }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) { passwordError.value = true; passwordMessage.value = '两次输入的新密码不一致'; return }
  passwordSaving.value = true
  try { await changeAdminPassword({ oldPassword: passwordForm.oldPassword, newPassword: passwordForm.newPassword }); passwordMessage.value = '密码已更新'; passwordForm.oldPassword = ''; passwordForm.newPassword = ''; passwordForm.confirmPassword = '' }
  catch (error) { passwordError.value = true; passwordMessage.value = apiErrorMessage(error, '密码更新失败') }
  finally { passwordSaving.value = false }
}

async function selectAvatar(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  avatarUploading.value = true
  try { const avatar = await uploadUserAvatar(file); auth.updateUser({ avatar }); Message.success('头像已更新') }
  catch (error) { errorMessage.value = apiErrorMessage(error, '头像上传失败') }
  finally { avatarUploading.value = false }
}

function formatDate(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.profile-hero { display: grid; grid-template-columns: auto minmax(0, 1fr) minmax(190px, .65fr); align-items: center; gap: 26px; margin-bottom: 18px; padding: 28px; border: 1px solid #dfe5f7; border-radius: 20px; background: linear-gradient(125deg, #f0f3ff, #f8fbff 58%, #eef8f5); box-shadow: var(--admin-shadow-card); }.profile-avatar-wrap { display: grid; justify-items: center; gap: 10px; }.profile-hero-copy h3 { margin: 0; font-size: 27px; }.profile-hero-copy p { max-width: 620px; margin: 9px 0 14px; color: var(--admin-muted); line-height: 1.7; }.profile-meta { display: grid; grid-template-columns: 1fr auto; gap: 12px 18px; padding-left: 22px; border-left: 1px solid rgb(88 113 216 / 18%); }.profile-meta span { color: var(--admin-muted); font-size: 12px; }.profile-meta strong { overflow: hidden; color: var(--admin-ink); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }.profile-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }
@media (max-width: 900px) { .profile-hero { grid-template-columns: auto minmax(0, 1fr); }.profile-meta { grid-column: 1 / -1; padding-top: 18px; padding-left: 0; border-top: 1px solid rgb(88 113 216 / 18%); border-left: 0; } .profile-grid { grid-template-columns: 1fr; } }
@media (max-width: 520px) { .profile-hero { grid-template-columns: 1fr; justify-items: center; text-align: center; }.profile-meta { width: 100%; text-align: left; }.profile-hero-copy { min-width: 0; } }
</style>
