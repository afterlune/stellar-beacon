<template>
  <el-drawer v-model="visible" direction="rtl" :with-header="false" :before-close="handleClose">
    <div class="account-center-header">
      <span>ACCOUNT / SETTINGS</span>
      <h2>账号设置</h2>
      <p>公开身份已移到创作台统一维护，这里只管理邮箱、订阅和通知。</p>
    </div>

    <template v-if="userInfo !== ''">
      <section class="account-identity">
        <img :src="userInfo.avatar || defaultAvatar" :alt="userInfo.nickname || '作者头像'" />
        <div>
          <strong>{{ userInfo.nickname || '未设置昵称' }}</strong>
          <small v-if="userInfo.handle">@{{ userInfo.handle }}</small>
        </div>
        <button type="button" @click="openStudioProfile">编辑公开资料</button>
      </section>

      <section class="account-section">
        <header><h3>邮箱与订阅</h3><span>用于验证码、订阅和评论邮件</span></header>
        <div class="account-row">
          <div><strong>邮箱</strong><small>{{ userInfo.email || '尚未绑定邮箱' }}</small></div>
          <button type="button" @click="emailDialogVisible = true">{{ userInfo.email ? '修改' : '绑定' }}</button>
        </div>
        <div class="account-row">
          <div><strong>文章订阅邮件</strong><small>{{ userInfo.email ? '新文章发布后的邮件通知' : '绑定邮箱后可用' }}</small></div>
          <el-switch
            :model-value="Number(userInfo.isSubscribe) === 1"
            :disabled="!userInfo.email || loading"
            @change="changeSubscribe" />
        </div>
      </section>

      <section class="account-section">
        <header><h3>互动通知</h3><span>分别控制站内互动和评论邮件</span></header>
        <div class="account-row">
          <div><strong>站内互动提醒</strong><small>评论、回复、点赞和收藏进入通知中心</small></div>
          <el-switch
            :model-value="Number(userInfo.notifyInteraction ?? 1) === 1"
            :disabled="loading"
            @change="changeInteractionNotice" />
        </div>
        <div class="account-row">
          <div><strong>话题订阅提醒</strong><small>订阅的话题有新文章时进入通知中心</small></div>
          <el-switch
            :model-value="Number(userInfo.notifyTopic ?? 1) === 1"
            :disabled="loading"
            @change="changeTopicNotice" />
        </div>
        <div class="account-row">
          <div><strong>评论邮件通知</strong><small>有人回复你时发送邮件提醒</small></div>
          <el-switch
            :model-value="Number(userInfo.notifyComment) === 1"
            :disabled="loading"
            @change="changeCommentNotice" />
        </div>
      </section>
    </template>
  </el-drawer>

  <el-dialog v-model="emailDialogVisible" width="30%">
    <el-form>
      <el-form-item class="mt-5">
        <el-input v-model.trim="email" placeholder="邮箱号" />
      </el-form-item>
      <el-form-item class="mt-8">
        <el-input v-model.trim="verificationCode" placeholder="验证码">
          <template #append>
            <button type="button" class="account-code-button" @click="sendCode">{{ message }}</button>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" size="large" class="mx-auto mt-3" @click="bindingEmail">保存邮箱</el-button>
      </el-form-item>
    </el-form>
  </el-dialog>
</template>

<script lang="ts">
import { defineComponent, getCurrentInstance, reactive, toRef, toRefs } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'UserCenter',
  setup() {
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const router = useRouter()
    const userStore = useUserStore()
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="72" height="72"%3E%3Crect width="72" height="72" rx="36" fill="%23172554"/%3E%3Ccircle cx="36" cy="27" r="13" fill="%239bb8ff"/%3E%3Cpath d="M12 67c4-17 12-25 24-25s20 8 24 25" fill="%239bb8ff"/%3E%3C/svg%3E'
    const userInfo = toRef(userStore.$state, 'userInfo')
    const visible = toRef(userStore.$state, 'userVisible')
    const reactiveData = reactive({
      message: '发送',
      emailDialogVisible: false,
      email: '',
      verificationCode: '',
      loading: false
    })

    const handleClose = () => {
      userStore.userVisible = false
    }
    const openStudioProfile = () => {
      userStore.userVisible = false
      void router.push('/studio/profile')
    }
    const bindingEmail = async () => {
      try {
        const response = await api.bindingEmail({ email: reactiveData.email, code: reactiveData.verificationCode })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '邮箱保存失败')
        userStore.userInfo = { ...(userStore.userInfo || {}), email: reactiveData.email }
        reactiveData.emailDialogVisible = false
        proxy.$notify({ title: '成功', message: '邮箱已保存', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: reason?.response?.data?.message || reason?.message || '邮箱保存失败', type: 'error' })
      }
    }
    const changeSubscribe = async (value: boolean) => {
      reactiveData.loading = true
      try {
        const response = await api.updateUserSubscribe({ userId: userStore.userInfo.userInfoId, isSubscribe: value ? 1 : 0 })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '订阅设置保存失败')
        userStore.userInfo = { ...(userStore.userInfo || {}), isSubscribe: value ? 1 : 0 }
        proxy.$notify({ title: '成功', message: '订阅设置已更新', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: reason?.response?.data?.message || reason?.message || '订阅设置保存失败', type: 'error' })
      } finally {
        reactiveData.loading = false
      }
    }
    const changeCommentNotice = async (value: boolean) => {
      reactiveData.loading = true
      try {
        const notifyComment = value ? 1 : 0
        const response = await api.updateCommentNotice({ notifyComment })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '通知设置保存失败')
        userStore.userInfo = { ...(userStore.userInfo || {}), notifyComment }
        proxy.$notify({ title: '成功', message: '通知设置已更新', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: reason?.response?.data?.message || reason?.message || '通知设置保存失败', type: 'error' })
      } finally {
        reactiveData.loading = false
      }
    }
    const changeInteractionNotice = async (value: boolean) => {
      reactiveData.loading = true
      try {
        const notifyInteraction = value ? 1 : 0
        const response = await api.updateNotificationPreferences({
          notifyInteraction,
          notifyTopic: Number(userStore.userInfo?.notifyTopic ?? 1)
        })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '通知设置保存失败')
        userStore.userInfo = { ...(userStore.userInfo || {}), notifyInteraction }
        proxy.$notify({ title: '成功', message: '站内通知设置已更新', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: reason?.response?.data?.message || reason?.message || '通知设置保存失败', type: 'error' })
      } finally {
        reactiveData.loading = false
      }
    }
    const changeTopicNotice = async (value: boolean) => {
      reactiveData.loading = true
      try {
        const notifyTopic = value ? 1 : 0
        const response = await api.updateNotificationPreferences({
          notifyInteraction: Number(userStore.userInfo?.notifyInteraction ?? 1),
          notifyTopic
        })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '通知设置保存失败')
        userStore.userInfo = { ...(userStore.userInfo || {}), notifyTopic }
        proxy.$notify({ title: '成功', message: '话题订阅通知已更新', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: reason?.response?.data?.message || reason?.message || '通知设置保存失败', type: 'error' })
      } finally {
        reactiveData.loading = false
      }
    }
    const sendCode = async () => {
      try {
        const response = await api.sendValidationCode(reactiveData.email)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '验证码发送失败')
        reactiveData.message = '已发送'
        proxy.$notify({ title: '成功', message: '验证码已发送', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: reason?.response?.data?.message || reason?.message || '验证码发送失败', type: 'error' })
      }
    }

    return {
      userInfo, visible, defaultAvatar, ...toRefs(reactiveData), handleClose, openStudioProfile,
      bindingEmail, changeSubscribe, changeInteractionNotice, changeTopicNotice, changeCommentNotice, sendCode
    }
  }
})
</script>

<style lang="scss" scoped>
.account-center-header { padding: 6px 0 20px; border-bottom: 1px solid var(--border-hairline); }
.account-center-header span { color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.account-center-header h2 { margin: 8px 0; font-size: 1.8rem; }
.account-center-header p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.account-identity { display: grid; grid-template-columns: 58px minmax(0, 1fr) auto; gap: 12px; align-items: center; margin-top: 20px; padding: 15px; border: 1px solid var(--border-hairline); border-radius: 15px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.account-identity img { width: 58px; height: 58px; border-radius: 50%; object-fit: cover; }
.account-identity strong, .account-identity small { display: block; }
.account-identity small { margin-top: 3px; color: var(--color-ob); }
.account-identity button, .account-row button, .account-code-button { border: 0; background: transparent; color: var(--color-ob); cursor: pointer; }
.account-section { margin-top: 18px; padding: 18px; border: 1px solid var(--border-hairline); border-radius: 15px; }
.account-section h3 { margin: 0; }
.account-section header { margin-bottom: 10px; }
.account-section header span { color: var(--text-ob-dim); font-size: 11px; }
.account-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 13px 0; border-top: 1px solid var(--border-hairline); }
.account-row strong, .account-row small { display: block; }
.account-row small { margin-top: 3px; color: var(--text-ob-dim); font-size: 11px; }
@media (max-width: 620px) { .account-identity { grid-template-columns: 52px 1fr; } .account-identity button { grid-column: 1 / -1; text-align: left; } }
</style>