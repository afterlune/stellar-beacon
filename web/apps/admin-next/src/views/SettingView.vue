<template>
  <section class="admin-page">
    <AdminPageHeader title="个人中心" description="维护你的后台身份信息和对外简介。">
      <template #actions>
        <a-button @click="router.push('/setting')">
          <template #icon><IconRefresh /></template>
          重新载入
        </a-button>
      </template>
    </AdminPageHeader>

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
        <span>登录邮箱</span><strong>{{ auth.user?.email || auth.user?.username || '未绑定' }}</strong>
        <span>登录 IP</span><strong>{{ auth.user?.ipAddress || '未知' }}</strong>
      </div>
    </div>

    <div class="profile-grid">
      <a-card class="admin-form-panel admin-form-card" :bordered="false" title="公开资料">
        <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
        <a-form :model="form" layout="vertical" @submit-success="save">
          <a-form-item
            field="nickname"
            label="昵称"
            :rules="[{ required: true, message: '请输入昵称' }]">
            <a-input v-model="form.nickname" maxlength="30" show-word-limit placeholder="展示在后台与公开页面的名字" />
          </a-form-item>
          <a-form-item field="intro" label="个人简介">
            <a-textarea
              v-model="form.intro"
              :max-length="200"
              show-word-limit
              :auto-size="{ minRows: 5, maxRows: 8 }"
              placeholder="一两句话介绍自己" />
          </a-form-item>
          <a-form-item
            field="website"
            label="个人网站"
            :rules="websiteRules">
            <a-input v-model="form.website" placeholder="https://" />
            <template #help>留空表示不展示个人网站。</template>
          </a-form-item>
          <div class="admin-form-actions">
            <a-button type="primary" html-type="submit" :loading="saving">保存资料</a-button>
            <a-button :disabled="saving" @click="resetForm">还原修改</a-button>
          </div>
        </a-form>
      </a-card>

      <a-card class="admin-form-panel admin-form-card" :bordered="false" title="账号安全">
        <a-alert v-if="passwordMessage" :type="passwordError ? 'error' : 'success'" closable @close="passwordMessage = ''">
          {{ passwordMessage }}
        </a-alert>
        <a-form ref="passwordFormRef" :model="passwordForm" layout="vertical" @submit-success="changePassword">
          <a-form-item
            field="oldPassword"
            label="当前密码"
            :rules="[{ required: true, message: '请输入当前密码' }]">
            <a-input-password v-model="passwordForm.oldPassword" autocomplete="current-password" placeholder="用于验证身份" />
          </a-form-item>
          <a-form-item
            field="newPassword"
            label="新密码"
            :rules="[
              { required: true, message: '请输入新密码' },
              { minLength: 6, message: '新密码至少需要 6 位' }
            ]">
            <a-input-password v-model="passwordForm.newPassword" autocomplete="new-password" placeholder="至少 6 位" />
          </a-form-item>
          <a-form-item
            field="confirmPassword"
            label="确认新密码"
            :rules="[
              { required: true, message: '请再次输入新密码' },
              { validator: validateConfirm, message: '两次输入的新密码不一致' }
            ]">
            <a-input-password v-model="passwordForm.confirmPassword" autocomplete="new-password" placeholder="再次输入新密码" />
          </a-form-item>
          <div class="admin-form-actions">
            <a-button html-type="submit" :loading="passwordSaving">更新密码</a-button>
          </div>
        </a-form>
        <div class="profile-security-note">
          <IconSafe aria-hidden="true" />
          <span>修改密码后当前会话保持有效；如怀疑账号泄露，请同时更换其他站点的同款密码。</span>
        </div>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh, IconSafe } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, changeAdminPassword, updateUserProfile, uploadUserAvatar } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatDate, initialOf } from '@/utils/format'

const router = useRouter()
const auth = useAuthStore()

const form = reactive({
  nickname: auth.user?.nickname || '',
  intro: auth.user?.intro || '',
  website: auth.user?.website || ''
})
const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const saving = ref(false)
const passwordSaving = ref(false)
const avatarUploading = ref(false)
const errorMessage = ref('')
const passwordMessage = ref('')
const passwordError = ref(false)
const avatarInput = ref<HTMLInputElement | null>(null)

const avatarText = computed(() => initialOf(auth.user?.nickname || auth.user?.username))
const websiteRules = computed(() => [
  {
    validator: (value: unknown, callback: (error?: string) => void) => {
      const text = String(value || '').trim()
      if (!text) {
        callback()
        return
      }
      if (!/^https?:\/\/.+/i.test(text)) callback('请输入以 http:// 或 https:// 开头的完整地址')
      else callback()
    }
  }
])

// Keep the form in sync when the stored profile changes (avatar upload, reload).
watch(() => auth.user, (user) => {
  if (!user) return
  form.nickname = user.nickname || ''
  form.intro = user.intro || ''
  form.website = user.website || ''
}, { deep: true })

function validateConfirm(value: unknown, callback: (error?: string) => void): void {
  if (String(value || '') !== passwordForm.newPassword) callback('两次输入的新密码不一致')
  else callback()
}

function resetForm(): void {
  form.nickname = auth.user?.nickname || ''
  form.intro = auth.user?.intro || ''
  form.website = auth.user?.website || ''
}

async function save(): Promise<void> {
  const nickname = form.nickname.trim()
  if (!nickname) {
    Message.error('昵称不能为空')
    return
  }
  saving.value = true
  errorMessage.value = ''
  try {
    await updateUserProfile({ nickname, intro: form.intro.trim(), website: form.website.trim() })
    auth.updateUser({ nickname, intro: form.intro.trim(), website: form.website.trim() })
    Message.success('个人信息已保存')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '个人信息保存失败')
    Message.error(errorMessage.value)
  } finally {
    saving.value = false
  }
}

async function changePassword(): Promise<void> {
  passwordMessage.value = ''
  passwordError.value = false
  if (passwordForm.newPassword.length < 6) {
    passwordError.value = true
    passwordMessage.value = '新密码至少需要 6 位'
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    passwordError.value = true
    passwordMessage.value = '两次输入的新密码不一致'
    return
  }
  passwordSaving.value = true
  try {
    await changeAdminPassword({
      oldPassword: passwordForm.oldPassword,
      newPassword: passwordForm.newPassword
    })
    passwordMessage.value = '密码已更新'
    Message.success('密码已更新')
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
  } catch (error) {
    passwordError.value = true
    passwordMessage.value = apiErrorMessage(error, '密码更新失败')
    Message.error(passwordMessage.value)
  } finally {
    passwordSaving.value = false
  }
}

async function selectAvatar(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    Message.warning('请选择图片文件作为头像')
    return
  }
  avatarUploading.value = true
  try {
    const avatar = await uploadUserAvatar(file)
    auth.updateUser({ avatar })
    Message.success('头像已更新')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '头像上传失败')
    Message.error(errorMessage.value)
  } finally {
    avatarUploading.value = false
  }
}
</script>

<style scoped>
.profile-security-note {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin-top: 4px;
  padding: 12px 14px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  color: var(--admin-muted);
  background: var(--admin-surface-soft);
  font-size: 12px;
  line-height: 1.65;
}

.profile-security-note svg {
  flex: 0 0 auto;
  margin-top: 2px;
  color: var(--admin-brand);
  font-size: 15px;
}
</style>
