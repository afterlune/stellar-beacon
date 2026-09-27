<template>
  <div class="header-controls absolute top-10 right-0 flex flex-row">
    <button type="button" class="header-control header-control-menu" data-dia="menu" :aria-label="t('settings.tips-open-menu')" @click="handleOpenMenu"><svg-icon icon-class="nav-menu" /></button>
    <button type="button" class="ob-drop-shadow" data-dia="search" :aria-label="locale === 'cn' ? '搜索' : 'Search'" aria-keyshortcuts="Control+K Meta+K" @click="handleOpenModel(true)"><svg-icon icon-class="search" /></button>
    <button type="button" class="ob-drop-shadow" data-dia="reading" :aria-label="t('settings.tips-open-reading')" @click="openReading"><svg-icon icon-class="clock-outline" /></button>
    <Dropdown v-if="multiLanguage === 1" @command="handleClick">
      <button type="button" class="ob-drop-shadow" data-dia="language" ><svg-icon icon-class="globe" /><span v-if="locale == 'cn'">中文</span><span v-if="locale == 'en'">EN</span><span class="sr-only">{{ locale === 'cn' ? '切换语言' : 'Switch language' }}</span></button>
      <DropdownMenu>
        <DropdownItem name="en">English</DropdownItem>
        <DropdownItem name="cn">中文</DropdownItem>
      </DropdownMenu>
    </Dropdown>
    <SpaceSwitcher />
    <button v-if="userInfo !== ''" type="button" class="ob-drop-shadow header-notification" data-dia="notifications" @click="openNotifications"><svg-icon icon-class="notice" /><i v-if="unreadCount > 0">{{ unreadCount > 99 ? '99+' : unreadCount }}</i><span class="sr-only">{{ locale === 'cn' ? '通知中心' : 'Notifications' }}</span></button>
    <template v-if="userInfo === ''">
      <button type="button" class="header-login" data-dia="login" @click="openLoginDialog">{{ t('settings.login') }}</button>
    </template>
    <template v-if="userInfo !== ''">
      <Dropdown class="account-menu">
        <button type="button" class="header-avatar-trigger" :aria-label="locale === 'cn' ? '账号菜单' : 'Account menu'" aria-haspopup="true">
          <Avatar :url="userInfo.avatar" />
        </button>
        <DropdownMenu>
          <template v-if="!isMobile">
            <DropdownItem @click="openForYou">{{ locale === 'cn' ? '为你推荐' : 'For you' }}</DropdownItem>
            <DropdownItem @click="openPublicProfile" :disabled="!publicHandle">{{ locale === 'cn' ? '我的公开主页' : 'My public profile' }}</DropdownItem>
            <DropdownItem @click="openFollowing">{{ locale === 'cn' ? '关注动态' : 'Following' }}</DropdownItem>
            <DropdownItem @click="openNotifications">{{ locale === 'cn' ? '发布提醒' : 'Notifications' }}</DropdownItem>
            <DropdownItem @click="openStudioProfile">{{ locale === 'cn' ? '公开资料' : 'Public profile' }}</DropdownItem>
            <DropdownItem @click="openFavorites">{{ t('reactions.favorites') }}</DropdownItem>
          </template>
          <DropdownItem @click="openUserCenter">{{ t('settings.personal-center') }}</DropdownItem>
          <DropdownItem @click="logout">{{ t('settings.logout') }}</DropdownItem>
        </DropdownMenu>
      </Dropdown>
    </template>
    <span no-hover-effect class="ob-drop-shadow" data-dia="light-switch">
      <ThemeToggle />
    </span>
  </div>
  <DialogSurface v-model="loginDialogVisible" title="登录" @opened="handleLoginDialogOpened" @closed="handleLoginDialogClosed">
    <form class="auth-form" @submit.prevent="login">
      <div class="mt-5"><input ref="loginUsernameInput" v-model="loginInfo.username" class="auth-input" autocomplete="username" placeholder="邮箱" /></div>
      <div class="auth-input-group mt-8">
        <input v-model="loginInfo.password" class="auth-input" :type="showPassword.login ? 'text' : 'password'" autocomplete="current-password" placeholder="密码" />
        <button type="button" class="auth-link password-toggle" :aria-label="showPassword.login ? '隐藏密码' : '显示密码'" :aria-pressed="showPassword.login" @click="showPassword.login = !showPassword.login">{{ showPassword.login ? '隐藏' : '显示' }}</button>
      </div>
      <button type="submit" :disabled="authLoading.login" class="auth-submit">{{ authLoading.login ? '登录中…' : '登录' }}</button>
      <div class="mt-8">
        <button type="button" class="auth-link" @click="openRegisterDialog">立即注册</button>
        <button type="button" class="auth-link float-right" @click="openForgetPasswordDialog">忘记密码?</button>
      </div>
    </form>
  </DialogSurface>
  <DialogSurface v-model="registerDialogVisible" title="注册">
    <form class="auth-form" @submit.prevent="register">
      <div class="mt-5"><input v-model="loginInfo.username" class="auth-input" autocomplete="email" placeholder="邮箱" /></div>
      <div class="auth-input-group mt-8">
        <input v-model="loginInfo.code" class="auth-input" autocomplete="one-time-code" placeholder="验证码" />
        <button type="button" class="auth-link" :disabled="authLoading.code" @click="sendCode">{{ authLoading.code ? '发送中…' : '发送' }}</button>
      </div>
      <div class="auth-input-group mt-8">
        <input v-model="loginInfo.password" class="auth-input" :type="showPassword.register ? 'text' : 'password'" autocomplete="new-password" placeholder="密码" />
        <button type="button" class="auth-link password-toggle" :aria-label="showPassword.register ? '隐藏密码' : '显示密码'" :aria-pressed="showPassword.register" @click="showPassword.register = !showPassword.register">{{ showPassword.register ? '隐藏' : '显示' }}</button>
      </div>
      <button type="submit" :disabled="authLoading.register" class="auth-submit">{{ authLoading.register ? '注册中…' : '注册' }}</button>
      <button type="button" class="auth-link" @click="returnLoginDialog">已有帐号?登录</button>
    </form>
  </DialogSurface>
  <DialogSurface v-model="forgetPasswordDialogVisible" title="重置密码">
    <form class="auth-form" @submit.prevent="updatePassword">
      <div class="mt-5"><input v-model="loginInfo.username" class="auth-input" autocomplete="email" placeholder="邮箱" /></div>
      <div class="auth-input-group mt-8">
        <input v-model="loginInfo.code" class="auth-input" autocomplete="one-time-code" placeholder="验证码" />
        <button type="button" class="auth-link" :disabled="authLoading.code" @click="sendCode">{{ authLoading.code ? '发送中…' : '发送' }}</button>
      </div>
      <div class="auth-input-group mt-8">
        <input v-model="loginInfo.password" class="auth-input" :type="showPassword.reset ? 'text' : 'password'" autocomplete="new-password" placeholder="新密码" />
        <button type="button" class="auth-link password-toggle" :aria-label="showPassword.reset ? '隐藏密码' : '显示密码'" :aria-pressed="showPassword.reset" @click="showPassword.reset = !showPassword.reset">{{ showPassword.reset ? '隐藏' : '显示' }}</button>
      </div>
      <button type="submit" :disabled="authLoading.password" class="auth-submit">{{ authLoading.password ? '提交中…' : '确定' }}</button>
      <button type="button" class="auth-link" @click="returnLoginDialog">返回登录</button>
    </form>
  </DialogSurface>
  <DialogSurface v-model="articlePasswordDialogVisible" title="受保护文章">
    <form class="auth-form" @submit.prevent="accessArticle">
      <div class="mt-5"><input id="article-password-input" v-model="articlePassword" class="auth-input" placeholder="文章受密码保护,请输入密码" /></div>
      <button type="submit" :disabled="authLoading.article" class="auth-submit">{{ authLoading.article ? '校验中…' : '校验密码' }}</button>
    </form>
  </DialogSurface>
  <teleport to="body">
    <SearchModel />
  </teleport>
</template>

<script lang="ts">
import { computed, defineComponent, toRef, toRefs, reactive, ref, getCurrentInstance, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { Dropdown, DropdownMenu, DropdownItem } from '@/components/Dropdown'
import Avatar from '@/components/Avatar.vue'
import { useAppStore } from '@/stores/app'
import { useCommonStore } from '@/stores/common'
import { useUserStore } from '@/stores/user'
import { useSocialStore } from '@/stores/social'
import { useRoute, useRouter } from 'vue-router'
import SpaceSwitcher from './SpaceSwitcher.vue'
import ThemeToggle from '@/components/ToggleSwitch/ThemeToggle.vue'
import api from '@/api/api'
import SearchModel from '@/components/SearchModel.vue'
import DialogSurface from '@/components/overlays/DialogSurface.vue'
import { useSearchStore } from '@/stores/search'
import { useNavigatorStore } from '@/stores/navigator'
import { useI18n } from 'vue-i18n'
import emitter from '@/utils/mitt'

export default defineComponent({
  name: 'Controls',
  components: {
    Dropdown,
    DropdownMenu,
    DropdownItem,
    Avatar,
    ThemeToggle,
    SearchModel,
    DialogSurface,
    SpaceSwitcher
  },
  setup() {
    const { t, locale } = useI18n()
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const appStore = useAppStore()
    const commonStore = useCommonStore()
    const userStore = useUserStore()
    const socialStore = useSocialStore()
    const searchStore = useSearchStore()
    const navigatorStore = useNavigatorStore()
    const router = useRouter()
    const publicHandle = computed(() => String(userStore.userInfo?.handle || '').trim())
    const route = useRoute()
    const loginUsernameInput = ref<any>(null)
    const loginInfo = reactive({
      username: '' as any,
      password: '' as any,
      code: '' as any
    })
    const reactiveDate = reactive({
      loginDialogVisible: false,
      registerDialogVisible: false,
      forgetPasswordDialogVisible: false,
      articlePasswordDialogVisible: false,
      articlePassword: '',
      articleId: '',
      showPassword: { login: false, register: false, reset: false },
      authLoading: {
        login: false,
        code: false,
        register: false,
        password: false,
        article: false
      }
    })
    emitter.on('changeArticlePasswordDialogVisible', (articleId: any) => {
      reactiveDate.articlePasswordDialogVisible = true
      reactiveDate.articlePassword = ''
      reactiveDate.articleId = articleId
      nextTick(() => {
        document.getElementById('article-password-input')?.focus()
      })
    })
    let unreadTimer: number | undefined
    const refreshUnread = () => {
      if (userStore.userInfo) void socialStore.refreshUnread()
    }
    watch(() => route.query.login, () => {
      if (route.query.login === '1' && !userStore.userInfo) {
        reactiveDate.loginDialogVisible = true
      } else if (userStore.userInfo) {
        reactiveDate.loginDialogVisible = false
      }
    }, { immediate: true })
    watch(() => reactiveDate.loginDialogVisible, (visible) => {
      if (!visible) return
      window.setTimeout(() => loginUsernameInput.value?.focus?.(), 50)
    })
    watch(() => userStore.userInfo, (value) => {
      if (value) refreshUnread()
      else socialStore.reset()
    }, { immediate: true })
    const handleSearchShortcut = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        reactiveDate.loginDialogVisible = false
        searchStore.setOpenModal(true)
      }
    }
    onMounted(() => {
      window.addEventListener('focus', refreshUnread)
      window.addEventListener('keydown', handleSearchShortcut)
      unreadTimer = window.setInterval(refreshUnread, 60000)
    })
    onUnmounted(() => {
      window.removeEventListener('focus', refreshUnread)
      window.removeEventListener('keydown', handleSearchShortcut)
      if (unreadTimer) window.clearInterval(unreadTimer)
    })
    const handleClick = (name: string): void => {
      appStore.changeLocale(name)
    }
    const errorMessage = (reason: any, fallback: string) => reason?.response?.data?.message || reason?.message || fallback
    const login = async () => {
      if (reactiveDate.authLoading.login) return
      if (loginInfo.username.trim().length == 0 || loginInfo.password.trim().length == 0) {
        proxy.$notify({ title: '提示', message: '账号或者密码不能为空', type: 'warning' })
        return
      }
      reactiveDate.authLoading.login = true
      const params = new URLSearchParams()
      params.append('username', loginInfo.username)
      params.append('password', loginInfo.password)
      try {
        const { data } = await api.login(params)
        if (!data.flag) throw new Error(data.message || '登录失败')
        userStore.setAuthSession({ userInfo: data.data, token: data.data.token })
        proxy.$notify({ title: '成功', message: '登录成功', type: 'success' })
        const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
        if (redirect) await router.replace(redirect)
        reactiveDate.loginDialogVisible = false
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: errorMessage(reason, '登录失败'), type: 'error' })
      } finally {
        reactiveDate.authLoading.login = false
      }
    }
    const handleLoginDialogOpened = () => {
      void nextTick(() => loginUsernameInput.value?.focus?.())
    }
    const handleLoginDialogClosed = () => {
      if (route.query.login !== '1' && typeof route.query.redirect !== 'string') return
      const query = { ...route.query }
      delete query.login
      delete query.redirect
      void router.replace({ path: route.path, query })
    }
    const logout = () => {
      api.logout().then(({ data }) => {
        if (data.flag) {
          userStore.clearSession()
          proxy.$notify({
            title: '成功',
            message: '登出成功',
            type: 'success'
          })
        } else {
          proxy.$notify({
            title: '错误',
            message: data.message,
            type: 'error'
          })
        }
      })
    }
    const openUserCenter = () => {
      userStore.userVisible = true
    }
    const openFavorites = () => {
      router.push({ path: '/studio/library/favorites' })
    }
    const openStudio = () => {
      router.push({ path: '/studio' })
    }
    const openNotifications = () => {
      router.push({ path: '/notifications' })
    }
    const openFollowing = () => {
      router.push({ path: '/following' })
    }
    const openForYou = () => {
      router.push({ path: '/for-you' })
    }
    const openPublicProfile = () => { if (publicHandle.value) router.push({ path: `/u/${publicHandle.value}` }) }
    const openStudioProfile = () => {
      router.push({ path: '/studio/profile' })
    }
    const openReading = () => {
      router.push({ path: '/studio/library/reading' })
    }
    const openLoginDialog = () => {
      reactiveDate.loginDialogVisible = true
    }
    const openRegisterDialog = () => {
      loginInfo.code = ''
      reactiveDate.loginDialogVisible = false
      reactiveDate.registerDialogVisible = true
    }
    const returnLoginDialog = () => {
      reactiveDate.registerDialogVisible = false
      reactiveDate.forgetPasswordDialogVisible = false
      reactiveDate.loginDialogVisible = true
    }
    const openForgetPasswordDialog = () => {
      loginInfo.code = ''
      reactiveDate.loginDialogVisible = false
      reactiveDate.forgetPasswordDialogVisible = true
    }
    const sendCode = async () => {
      if (reactiveDate.authLoading.code) return
      if (loginInfo.username.trim().length == 0) {
        proxy.$notify({ title: '提示', message: '请先填写邮箱', type: 'warning' })
        return
      }
      reactiveDate.authLoading.code = true
      try {
        const { data } = await api.sendValidationCode(loginInfo.username)
        if (!data.flag) throw new Error(data.message || '验证码发送失败')
        proxy.$notify({ title: '成功', message: '验证码已发送', type: 'success' })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: errorMessage(reason, '验证码发送失败'), type: 'error' })
      } finally {
        reactiveDate.authLoading.code = false
      }
    }
    const register = async () => {
      if (reactiveDate.authLoading.register) return
      if (!loginInfo.username.trim() || !loginInfo.code.trim() || !loginInfo.password.trim()) {
        proxy.$notify({ title: '提示', message: '邮箱、验证码和密码不能为空', type: 'warning' })
        return
      }
      reactiveDate.authLoading.register = true
      try {
        const { data } = await api.register({ code: loginInfo.code, username: loginInfo.username, password: loginInfo.password })
        if (!data.flag) throw new Error(data.message || '注册失败')
        proxy.$notify({ title: '成功', message: '注册成功', type: 'success' })
        reactiveDate.registerDialogVisible = false
        reactiveDate.loginDialogVisible = true
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: errorMessage(reason, '注册失败'), type: 'error' })
      } finally {
        reactiveDate.authLoading.register = false
      }
    }
    const handleOpenModel = (status: boolean) => {
      searchStore.setOpenModal(status)
    }

    // The mobile drawer was only reachable from the floating navigator button in
    // the bottom-right corner; a hamburger belongs in the bar.
    const handleOpenMenu = () => {
      navigatorStore.toggleMobileMenu()
    }

    const updatePassword = async () => {
      if (reactiveDate.authLoading.password) return
      if (!loginInfo.username.trim() || !loginInfo.code.trim() || !loginInfo.password.trim()) {
        proxy.$notify({ title: '提示', message: '邮箱、验证码和新密码不能为空', type: 'warning' })
        return
      }
      reactiveDate.authLoading.password = true
      try {
        const { data } = await api.updatePassword(loginInfo)
        if (!data.flag) throw new Error(data.message || '密码修改失败')
        proxy.$notify({ title: '成功', message: '密码已更新，请重新登录', type: 'success' })
        reactiveDate.forgetPasswordDialogVisible = false
        reactiveDate.loginDialogVisible = true
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: errorMessage(reason, '密码修改失败'), type: 'error' })
      } finally {
        reactiveDate.authLoading.password = false
      }
    }
    const accessArticle = async () => {
      if (reactiveDate.authLoading.article) return
      if (reactiveDate.articlePassword.trim().length == 0) {
        proxy.$notify({ title: '提示', message: '密码不能为空', type: 'warning' })
        return
      }
      reactiveDate.authLoading.article = true
      try {
        const { data } = await api.accessArticle({ articleId: reactiveDate.articleId, articlePassword: reactiveDate.articlePassword })
        if (!data.flag) throw new Error(data.message || '文章密码校验失败')
        reactiveDate.articlePasswordDialogVisible = false
        userStore.accessArticles.push(reactiveDate.articleId)
        await router.push({ path: '/articles/' + reactiveDate.articleId })
      } catch (reason: any) {
        proxy.$notify({ title: '错误', message: errorMessage(reason, '文章密码校验失败'), type: 'error' })
      } finally {
        reactiveDate.authLoading.article = false
      }
    }
    return {
      handleOpenModel,
      handleOpenMenu,
      handleLoginDialogClosed,
      handleLoginDialogOpened,
      loginInfo,
      loginUsernameInput,
      ...toRefs(reactiveDate),
      userInfo: toRef(userStore.$state, 'userInfo'),
      isMobile: toRef(commonStore.$state, 'isMobile'),
      login,
      logout,
      handleClick,
      openUserCenter,
      openFavorites,
      openStudio,
      openPublicProfile,
      publicHandle,
      openNotifications,
      openFollowing,
      openForYou,
      unreadCount: computed(() => socialStore.unreadCount),
      openStudioProfile,
      openReading,
      openLoginDialog,
      openRegisterDialog,
      returnLoginDialog,
      sendCode,
      register,
      updatePassword,
      openForgetPasswordDialog,
      accessArticle,
      multiLanguage: computed(() => {
        let websiteConfig: any = appStore.websiteConfig
        return websiteConfig.multiLanguage
      }),
      locale,
      t
    }
  }
})
</script>
<style lang="scss" scoped>
.text {
  color: var(--text-normal);
  cursor: pointer;
}
.auth-link {
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  padding: 6px 4px;
  border: 0;
  background: transparent;
  color: var(--color-ob);
  font: inherit;
  cursor: pointer;
}
.auth-link:disabled {
  cursor: not-allowed;
  opacity: .55;
}
.auth-submit {
  display: block;
  width: 100%;
  min-height: 42px;
  padding: 9px 14px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ob);
  color: #081127;
  font: inherit;
  font-weight: 700;
  margin-top: 12px;
  cursor: pointer;
}
.auth-submit:disabled { opacity: .6; cursor: progress; }
.auth-form { display: grid; gap: 12px; }
.auth-input {
  box-sizing: border-box;
  display: block;
  width: 100%;
  min-height: 42px;
  padding: 9px 12px;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 25%, transparent);
  border-radius: 9px;
  outline: none;
  background: var(--background-primary-alt);
  color: var(--text-normal);
  font: inherit;
}
.auth-input:focus { border-color: var(--color-ob); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-ob) 18%, transparent); }
.auth-input-group { display: flex; align-items: center; gap: 12px; padding-right: 12px; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 25%, transparent); border-radius: 9px; background: var(--background-primary-alt); }
.auth-input-group .auth-input { border: 0; background: transparent; box-shadow: none; }
.auth-input-group .auth-link { flex: 0 0 auto; }
.auth-input-group .password-toggle { min-width: 38px; justify-content: center; }
.auth-link:disabled { cursor: progress; }
.auth-form > .auth-link { justify-self: start; }
.auth-form .mt-8 { margin-top: .5rem; }
.auth-form .mt-5 { margin-top: .25rem; }
@media (max-width: 640px) {
  .auth-form { gap: 9px; }
}
.header-controls > button:focus-visible,
.header-controls .ob-dropdown > button:focus-visible,
.auth-link:focus-visible {
  outline: 2px solid var(--color-ob);
  outline-offset: 2px;
}
#submit-button {
  outline: none;
  background: var(--text-accent);
}
.header-notification {
  position: relative;
}
.header-notification i {
  position: absolute;
  top: -7px;
  right: -8px;
  min-width: 17px;
  height: 17px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--color-ob);
  color: #081127;
  font-size: 10px;
  font-style: normal;
  font-weight: 800;
  line-height: 17px;
  text-align: center;
}
.header-controls {
  gap: 2px;
  > span,
  > button,
  .ob-dropdown > button {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 8px;
    height: 34px;
    padding: 0 8px;
    border: 0;
    background: transparent;
    border-radius: var(--radius-md);
    color: var(--header-fg);
    font: inherit;
    font-size: 13px;
    cursor: pointer;
    transition: background-color 200ms ease, color 250ms ease;
    &[no-hover-effect] {
      &:hover {
        background-color: transparent;
      }
    }
    &:hover {
      background-color: var(--surface-hover);
    }
    .svg-icon {
      stroke: var(--header-fg);
      height: 1.25rem;
      width: 1.25rem;
      margin-right: 0;
      pointer-events: none;
      transition: stroke 250ms ease;
    }
  }
  /* The side drawer replaces desktop navigation below the 1280px breakpoint. */
  .header-control-menu {
    display: flex;
  }
  @media (min-width: 1280px) {
    .header-control-menu {
      display: none;
    }
  }
  .search-bar {
    @apply bg-transparent flex flex-row px-0 mr-2 rounded-full;
    opacity: 0;
    width: 0;
    transition: 300ms all ease-out;
    &.active {
      @apply bg-ob-deep-800;
      opacity: 0.95;
      width: 200px;
      imput {
        width: initial;
      }
    }
    &:focus {
      appearance: none;
      outline: none;
    }
    input {
      @apply flex flex-1 bg-transparent text-ob-normal px-6 box-border;
      width: 0;
      appearance: none;
      outline: none;
    }
    svg {
      @apply float-right;
    }
  }
}
.avatar-img {
  transition-property: transform;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  transition-duration: 800ms;
  transform: rotate(-360deg);
}
.avatar-img:hover {
  transform: rotate(360deg);
}
</style>
