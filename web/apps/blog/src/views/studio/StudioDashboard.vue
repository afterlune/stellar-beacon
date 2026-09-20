<template>
  <div class="studio-dashboard">
    <header class="studio-page-head">
      <div><p>WORKSPACE / 01</p><h1>创作总览</h1><span>管理公开作品，也保留只属于自己的内容。</span></div>
      <router-link to="/studio/articles/new">写一篇文章 →</router-link>
    </header>

    <section class="studio-stats">
      <article v-for="stat in stats" :key="stat.key">
        <span>{{ stat.index }}</span>
        <strong>{{ dashboard[stat.key] || 0 }}</strong>
        <small>{{ stat.label }}</small>
      </article>
    </section>

    <div class="studio-dashboard__grid">
      <section class="studio-panel studio-panel--profile">
        <header>
          <div><p>PUBLIC IDENTITY</p><h2>公开主页资料</h2></div>
          <span>{{ completion.completed }}/{{ completion.total }} 已完成</span>
        </header>
        <div class="studio-profile-summary">
          <img :src="profile.avatar || defaultAvatar" :alt="profile.nickname || '作者头像'" />
          <div class="studio-profile-summary__copy">
            <strong>{{ profile.nickname || '未设置昵称' }}</strong>
            <span>@{{ profile.handle || 'your-handle' }}</span>
            <p>{{ profile.intro || '还没有公开简介，补全后作者主页会更完整。' }}</p>
          </div>
          <div class="studio-profile-progress">
            <span><i :style="{ width: `${completion.completed * 25}%` }" /></span>
            <small>{{ completion.missing.length ? `还缺：${completion.missing.map((item) => item.label).join('、')}` : '公开身份已完整' }}</small>
          </div>
          <div class="studio-profile-actions">
            <router-link to="/studio/profile">{{ completion.missing.length ? '完善公开资料' : '编辑公开资料' }} →</router-link>
            <router-link v-if="validHandle" :to="`/u/${normalizedHandle}`">预览主页</router-link>
          </div>
        </div>
      </section>

      <section class="studio-panel">
        <header><div><p>QUICK START</p><h2>继续创作</h2></div></header>
        <div class="studio-quick">
          <router-link to="/studio/articles"><strong>文章工作台</strong><span>公开、私有、草稿与定时发布</span><em>01 →</em></router-link>
          <router-link to="/studio/talks"><strong>发布随想</strong><span>记录一条轻量的公开信号</span><em>02 →</em></router-link>
          <router-link to="/studio/series"><strong>组织系列</strong><span>把长期文章串成阅读路径</span><em>03 →</em></router-link>
        </div>
      </section>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api/api'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { isValidStudioHandle, normalizeStudioHandle, studioProfileCompletion, type StudioProfile } from '@/utils/studioProfile'

export default defineComponent({
  name: 'StudioDashboard',
  setup() {
    const userStore = useUserStore()
    const appStore = useAppStore()
    const dashboard = ref<Record<string, number>>({})
    const profile = reactive<StudioProfile>({
      handle: normalizeStudioHandle(userStore.userInfo?.handle),
      nickname: userStore.userInfo?.nickname || '',
      avatar: userStore.userInfo?.avatar || '',
      intro: userStore.userInfo?.intro || '',
      website: userStore.userInfo?.website || ''
    })
    const stats = [
      { key: 'articleCount', label: '全部文章', index: 'A' },
      { key: 'draftCount', label: '草稿', index: 'D' },
      { key: 'privateCount', label: '私有内容', index: 'P' },
      { key: 'talkCount', label: '随想', index: 'T' },
      { key: 'seriesCount', label: '系列', index: 'S' },
      { key: 'favoriteCount', label: '收藏', index: 'F' }
    ]
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'
    const completion = computed(() => studioProfileCompletion(profile, appStore.websiteConfig?.userAvatar || ''))
    const normalizedHandle = computed(() => normalizeStudioHandle(profile.handle))
    const validHandle = computed(() => isValidStudioHandle(normalizedHandle.value))

    const loadDashboard = async () => {
      try {
        const response = await api.getStudioDashboard()
        dashboard.value = response?.data?.data || {}
      } catch {
        ElMessage.error('创作数据加载失败')
      }
    }

    const loadProfile = async () => {
      try {
        const response = await api.getStudioProfile()
        if (!response?.data?.flag) return
        Object.assign(profile, response.data.data || {})
        userStore.userInfo = { ...(userStore.userInfo || {}), ...response.data.data }
      } catch {
        // Keep the cached identity visible when the profile request is unavailable.
      }
    }

    onMounted(() => {
      void loadDashboard()
      void loadProfile()
    })

    return { dashboard, profile, stats, defaultAvatar, completion, normalizedHandle, validHandle }
  }
})
</script>

<style lang="scss" scoped>
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 32px; border: 1px solid var(--border-hairline); border-radius: 20px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .22), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p, .studio-panel header p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 8px; font-size: clamp(2rem, 4vw, 3.4rem); letter-spacing: -.05em; }
.studio-page-head span, .studio-panel header > span { color: var(--text-ob-dim); font-size: 12px; }
.studio-page-head > a { padding: 10px 16px; border-radius: 999px; background: var(--color-ob); color: #081127; font-weight: 700; text-decoration: none; }
.studio-stats { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 10px; margin: 18px 0; }
.studio-stats article { padding: 16px; border: 1px solid var(--border-hairline); border-radius: 14px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.studio-stats span, .studio-stats small { display: block; color: var(--text-ob-dim); font-size: 10px; }
.studio-stats strong { display: block; margin: 8px 0 4px; font-size: 1.45rem; }
.studio-dashboard__grid { display: grid; grid-template-columns: minmax(0, 1.25fr) minmax(280px, .75fr); gap: 16px; }
.studio-panel { padding: 24px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.studio-panel > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.studio-panel h2 { margin: 0; }
.studio-profile-summary { display: grid; grid-template-columns: 84px minmax(0, 1fr); gap: 16px; align-items: center; }
.studio-profile-summary > img { width: 84px; height: 84px; border: 1px solid color-mix(in srgb, var(--color-ob) 45%, transparent); border-radius: 50%; object-fit: cover; }
.studio-profile-summary__copy strong, .studio-profile-summary__copy span { display: block; }
.studio-profile-summary__copy strong { font-size: 1.2rem; }
.studio-profile-summary__copy span { margin-top: 3px; color: var(--color-ob); font-size: 11px; }
.studio-profile-summary__copy p { margin: 9px 0 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.65; }
.studio-profile-progress { grid-column: 1 / -1; display: grid; gap: 8px; }
.studio-profile-progress > span { height: 5px; overflow: hidden; border-radius: 999px; background: color-mix(in srgb, var(--text-ob-dim) 18%, transparent); }
.studio-profile-progress i { display: block; height: 100%; border-radius: inherit; background: var(--color-ob); transition: width .2s ease; }
.studio-profile-progress small { color: var(--text-ob-dim); font-size: 11px; }
.studio-profile-actions { grid-column: 1 / -1; display: flex; flex-wrap: wrap; gap: 10px; }
.studio-profile-actions a { color: var(--color-ob); font-size: 12px; text-decoration: none; }
.studio-quick { display: grid; gap: 10px; }
.studio-quick a { display: grid; grid-template-columns: 1fr auto; gap: 4px 12px; padding: 15px; border: 1px solid var(--border-hairline); border-radius: 13px; color: inherit; text-decoration: none; }
.studio-quick strong, .studio-quick span { display: block; }
.studio-quick span { grid-column: 1; color: var(--text-ob-dim); font-size: 11px; }
.studio-quick em { grid-column: 2; grid-row: 1 / span 2; align-self: center; color: var(--color-ob); font-style: normal; }
@media (max-width: 980px) { .studio-stats { grid-template-columns: repeat(3, minmax(0, 1fr)); } .studio-dashboard__grid { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .studio-page-head { align-items: stretch; flex-direction: column; } .studio-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); } .studio-profile-summary { grid-template-columns: 1fr; } .studio-profile-summary > img { width: 72px; height: 72px; } .studio-profile-progress, .studio-profile-actions { grid-column: auto; } }
</style>
