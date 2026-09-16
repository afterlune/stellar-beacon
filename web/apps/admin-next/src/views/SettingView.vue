<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('account.title')" :description="t('account.description')">
      <template #actions>
        <a-button @click="router.push('/setting')">
          <template #icon><IconRefresh /></template>
          {{ t('account.reload') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <div class="profile-hero">
      <div class="profile-avatar-wrap">
        <a-avatar :size="88" :image-url="auth.user?.avatar">{{ avatarText }}</a-avatar>
        <input ref="avatarInput" type="file" accept="image/*" hidden @change="selectAvatar" />
        <a-button size="small" :loading="avatarUploading" @click="avatarInput?.click()">{{ t('account.changeAvatar') }}</a-button>
      </div>
      <div class="profile-hero-copy">
        <div class="admin-page-eyebrow">{{ t('account.eyebrow') }}</div>
        <h3>{{ auth.user?.nickname || auth.user?.username || t('shell.admin') }}</h3>
        <p>{{ form.intro || t('account.avatarHint') }}</p>
        <a-space wrap>
          <a-tag color="arcoblue">{{ auth.user?.username || t('account.adminAccount') }}</a-tag>
          <a-tag color="green">{{ t('account.roleAdmin') }}</a-tag>
          <a-tag v-if="auth.user?.website" color="orange">{{ auth.user.website }}</a-tag>
        </a-space>
      </div>
      <div class="profile-meta">
        <span>{{ t('account.createdAt') }}</span><strong>{{ formatDate(auth.user?.createTime) }}</strong>
        <span>{{ t('account.lastLogin') }}</span><strong>{{ formatDate(auth.user?.lastLoginTime) }}</strong>
        <span>{{ t('account.loginEmail') }}</span><strong>{{ auth.user?.email || auth.user?.username || t('account.unbound') }}</strong>
        <span>{{ t('account.loginIp') }}</span><strong>{{ auth.user?.ipAddress || t('account.unknownIp') }}</strong>
      </div>
    </div>

    <div class="profile-grid">
      <a-card class="admin-form-panel admin-form-card" :bordered="false" :title="t('account.public.title')">
        <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
        <a-form :model="form" layout="vertical" @submit-success="save">
          <a-form-item
            field="nickname"
            :label="t('account.public.nickname')"
            :rules="[{ required: true, message: t('account.public.nicknameRequired') }]">
            <a-input v-model="form.nickname" maxlength="30" show-word-limit :placeholder="t('account.public.nicknamePlaceholder')" />
          </a-form-item>
          <a-form-item field="intro" :label="t('account.public.intro')">
            <a-textarea
              v-model="form.intro"
              :max-length="200"
              show-word-limit
              :auto-size="{ minRows: 5, maxRows: 8 }"
              :placeholder="t('account.public.introPlaceholder')" />
          </a-form-item>
          <a-form-item
            field="website"
            :label="t('account.public.website')"
            :rules="websiteRules">
            <a-input v-model="form.website" placeholder="https://" />
            <template #help>{{ t('account.public.websiteHelp') }}</template>
          </a-form-item>
          <div class="admin-form-actions">
            <a-button type="primary" html-type="submit" :loading="saving">{{ t('account.public.save') }}</a-button>
            <a-button :disabled="saving" @click="resetForm">{{ t('account.public.revert') }}</a-button>
          </div>
        </a-form>
      </a-card>

      <a-card class="admin-form-panel admin-form-card" :bordered="false" :title="t('account.password.title')">
        <a-alert v-if="passwordMessage" :type="passwordError ? 'error' : 'success'" closable @close="passwordMessage = ''">
          {{ passwordMessage }}
        </a-alert>
        <a-form ref="passwordFormRef" :model="passwordForm" layout="vertical" @submit-success="changePassword">
          <a-form-item
            field="oldPassword"
            :label="t('account.password.current')"
            :rules="[{ required: true, message: t('account.password.currentRequired') }]">
            <a-input-password v-model="passwordForm.oldPassword" autocomplete="current-password" :placeholder="t('account.password.currentPlaceholder')" />
          </a-form-item>
          <a-form-item
            field="newPassword"
            :label="t('account.password.next')"
            :rules="[
              { required: true, message: t('account.password.nextRequired') },
              { minLength: 6, message: t('account.password.tooShort') }
            ]">
            <a-input-password v-model="passwordForm.newPassword" autocomplete="new-password" :placeholder="t('account.password.nextPlaceholder')" />
          </a-form-item>
          <a-form-item
            field="confirmPassword"
            :label="t('account.password.confirm')"
            :rules="[
              { required: true, message: t('account.password.confirmRequired') },
              { validator: validateConfirm, message: t('account.password.mismatch') }
            ]">
            <a-input-password v-model="passwordForm.confirmPassword" autocomplete="new-password" :placeholder="t('account.password.confirmPlaceholder')" />
          </a-form-item>
          <div class="admin-form-actions">
            <a-button html-type="submit" :loading="passwordSaving">{{ t('account.password.submit') }}</a-button>
          </div>
        </a-form>
        <div class="profile-security-note">
          <IconSafe aria-hidden="true" />
          <span>{{ t('account.password.note') }}</span>
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
import { t } from '@/i18n'
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
      if (!/^https?:\/\/.+/i.test(text)) callback(t('account.public.websiteInvalid'))
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
  if (String(value || '') !== passwordForm.newPassword) callback(t('account.password.mismatch'))
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
    Message.error(t('account.public.nicknameEmpty'))
    return
  }
  saving.value = true
  errorMessage.value = ''
  try {
    await updateUserProfile({ nickname, intro: form.intro.trim(), website: form.website.trim() })
    auth.updateUser({ nickname, intro: form.intro.trim(), website: form.website.trim() })
    Message.success(t('account.public.saved'))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('account.public.saveFailed'))
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
    passwordMessage.value = t('account.password.tooShort')
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    passwordError.value = true
    passwordMessage.value = t('account.password.mismatch')
    return
  }
  passwordSaving.value = true
  try {
    await changeAdminPassword({
      oldPassword: passwordForm.oldPassword,
      newPassword: passwordForm.newPassword
    })
    passwordMessage.value = t('account.password.updated')
    Message.success(t('account.password.updated'))
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
  } catch (error) {
    passwordError.value = true
    passwordMessage.value = apiErrorMessage(error, t('account.password.updateFailed'))
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
    Message.warning(t('account.avatar.wrongType'))
    return
  }
  avatarUploading.value = true
  try {
    const avatar = await uploadUserAvatar(file)
    auth.updateUser({ avatar })
    Message.success(t('account.avatar.updated'))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('account.avatar.failed'))
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
