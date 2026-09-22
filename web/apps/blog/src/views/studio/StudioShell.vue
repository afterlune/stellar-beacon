<template>
  <div class="studio-shell">
    <header class="studio-shell__top">
      <router-link to="/" class="studio-shell__brand">
        <span>SB</span>
        <div><strong>Stellar Studio</strong><small>私有创作空间</small></div>
      </router-link>
      <div class="studio-shell__user">
        <router-link v-if="userInfo?.handle" :to="`/u/${userInfo.handle}`">查看公开主页</router-link>
        <router-link to="/studio/profile" class="studio-shell__identity">
          <img :src="userInfo?.avatar || defaultAvatar" :alt="userInfo?.nickname || 'author'" />
          <span><strong>{{ userInfo?.nickname || userInfo?.username || '创作者' }}</strong><small v-if="userInfo?.handle">@{{ userInfo.handle }}</small></span>
        </router-link>
        <button type="button" @click="logout">退出</button>
      </div>
    </header>

    <div class="studio-shell__body">
      <aside class="studio-nav">
        <router-link v-for="item in navigation" :key="item.path" :to="item.path" :class="{ active: isActive(item.path) }">
          <span>{{ item.index }}</span>
          <div><strong>{{ item.label }}</strong><small>{{ item.hint }}</small></div>
        </router-link>
      </aside>
      <main class="studio-main">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import api from '@/api/api'

export default defineComponent({
  name: 'StudioShell',
  setup() {
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const userInfo = computed(() => userStore.userInfo || {})
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="64" height="64"%3E%3Crect width="64" height="64" rx="32" fill="%23172554"/%3E%3Ccircle cx="32" cy="24" r="11" fill="%239bb8ff"/%3E%3Cpath d="M11 59c3-15 11-22 21-22s18 7 21 22" fill="%239bb8ff"/%3E%3C/svg%3E'
    const navigation = [
      { path: '/studio/dashboard', index: '01', label: '总览', hint: '数据与下一步' },
      { path: '/studio/articles', index: '02', label: '文章', hint: '公开、私有与草稿' },
      { path: '/studio/talks', index: '03', label: '随想', hint: '轻量内容' },
      { path: '/studio/series', index: '04', label: '系列', hint: '组织长期主题' },
      { path: '/studio/collections', index: '05', label: '书单', hint: '公开文章收藏集' },
      { path: '/studio/topics', index: '06', label: '分类与标签', hint: '私有词汇表' },
      { path: '/studio/library/reading', index: '07', label: '阅读记录', hint: '最近读过' },
      { path: '/studio/library/favorites', index: '08', label: '我的收藏', hint: '稍后阅读' },
      { path: '/studio/profile', index: '09', label: '公开资料', hint: '主页身份与头像' }
    ]
    const isActive = (path: string) => route.path === path || route.path.startsWith(path + '/')
    const logout = async () => {
      try { await api.logout() } catch { /* local session still clears */ }
      userStore.userInfo = ''
      userStore.token = ''
      userStore.accessArticles = []
      sessionStorage.removeItem('token')
      await router.replace('/')
    }
    return { userInfo, defaultAvatar, navigation, isActive, logout }
  }
})
</script>

<style lang="scss" scoped>
.studio-shell { max-width: 1320px; margin: 0 auto; padding: 18px 0 90px; }
.studio-shell__top { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 15px 18px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-shell__brand { display: flex; align-items: center; gap: 12px; color: inherit; text-decoration: none; }
.studio-shell__brand > span { display: grid; width: 42px; height: 42px; place-items: center; border-radius: 12px; background: var(--color-ob); color: #081127; font-weight: 900; letter-spacing: -.05em; }
.studio-shell__brand strong, .studio-shell__brand small, .studio-shell__user strong, .studio-shell__user small { display: block; }
.studio-shell__brand small, .studio-shell__user small { margin-top: 2px; color: var(--text-ob-dim); font-size: 10px; letter-spacing: .08em; }
.studio-shell__user { display: flex; align-items: center; gap: 11px; }
.studio-shell__user > a, .studio-shell__user button, .studio-shell__identity { padding: 7px 11px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); font: inherit; font-size: 11px; text-decoration: none; cursor: pointer; }
.studio-shell__user img { width: 38px; height: 38px; border-radius: 50%; object-fit: cover; }
.studio-shell__user span { font-size: 12px; }
.studio-shell__identity { display: flex; align-items: center; gap: 9px; color: inherit; text-decoration: none; }
.studio-shell__identity:hover { border-color: color-mix(in srgb, var(--color-ob) 48%, transparent); }
.studio-shell__body { display: grid; grid-template-columns: 230px minmax(0, 1fr); gap: 22px; margin-top: 22px; }
.studio-nav { position: sticky; top: 18px; align-self: start; display: grid; gap: 6px; padding: 10px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 86%, transparent); }
.studio-nav a { display: grid; grid-template-columns: 34px 1fr; gap: 10px; align-items: center; padding: 10px; border-radius: 12px; color: var(--text-ob-dim); text-decoration: none; transition: background .2s ease, color .2s ease; }
.studio-nav a > span { color: color-mix(in srgb, var(--text-ob-dim) 60%, transparent); font-size: 10px; }
.studio-nav a strong, .studio-nav a small { display: block; }
.studio-nav a strong { color: inherit; font-size: 13px; }
.studio-nav a small { margin-top: 2px; font-size: 10px; opacity: .65; }
.studio-nav a:hover, .studio-nav a.active { background: color-mix(in srgb, var(--color-ob) 12%, transparent); color: var(--color-ob); }
.studio-main { min-width: 0; }
@media (max-width: 860px) { .studio-shell__top { align-items: flex-start; flex-direction: column; } .studio-shell__body { grid-template-columns: 1fr; } .studio-nav { position: static; display: flex; overflow-x: auto; } .studio-nav a { min-width: 150px; } }
</style>
