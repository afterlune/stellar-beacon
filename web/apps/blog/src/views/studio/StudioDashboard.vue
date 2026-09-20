<template>
  <div class="studio-dashboard">
    <header class="studio-page-head">
      <div><p>WORKSPACE / 01</p><h1>创作总览</h1><span>管理公开作品，也保留只属于自己的内容。</span></div>
      <router-link to="/studio/articles?new=1">写一篇文章 →</router-link>
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
        <header><div><p>PUBLIC IDENTITY</p><h2>公开主页资料</h2></div><span>handle 决定主页地址</span></header>
        <form @submit.prevent="saveProfile">
          <label>Handle<input v-model.trim="profile.handle" maxlength="32" placeholder="your-handle" /></label>
          <label>昵称<input v-model.trim="profile.nickname" maxlength="30" placeholder="作者昵称" /></label>
          <label class="wide">个人简介<textarea v-model.trim="profile.intro" rows="3" maxlength="255" placeholder="介绍你的关注领域与写作方向" /></label>
          <label class="wide">个人网站<input v-model.trim="profile.website" placeholder="https://example.com" /></label>
          <div class="studio-form-actions wide">
            <button type="submit" :disabled="saving">{{ saving ? '保存中…' : '保存资料' }}</button>
            <router-link v-if="profile.handle" :to="`/u/${profile.handle}`">预览主页</router-link>
          </div>
        </form>
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
import { defineComponent, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'StudioDashboard',
  setup() {
    const userStore = useUserStore()
    const dashboard = ref<Record<string, number>>({})
    const saving = ref(false)
    const profile = reactive({
      handle: userStore.userInfo?.handle || '',
      nickname: userStore.userInfo?.nickname || '',
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

    const loadDashboard = async () => {
      try {
        const response = await api.getStudioDashboard()
        dashboard.value = response?.data?.data || {}
      } catch {
        ElMessage.error('创作数据加载失败')
      }
    }

    const saveProfile = async () => {
      saving.value = true
      try {
        const response = await api.updateStudioProfile(profile)
        if (!response?.data?.flag) throw new Error(response?.data?.message || 'save failed')
        userStore.userInfo = { ...(userStore.userInfo || {}), ...profile }
        ElMessage.success('公开资料已更新')
      } catch (error: any) {
        ElMessage.error(error?.response?.data?.message || error?.message || '保存失败')
      } finally {
        saving.value = false
      }
    }

    onMounted(loadDashboard)
    return { dashboard, profile, saving, stats, saveProfile }
  }
})
</script>

<style lang="scss" scoped>
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 32px; border: 1px solid var(--border-hairline); border-radius: 20px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .22), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p, .studio-panel header p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 8px; font-size: clamp(2rem, 4vw, 3.4rem); letter-spacing: -.05em; }
.studio-page-head span, .studio-panel header > span { color: var(--text-ob-dim); font-size: 12px; }
.studio-page-head > a { padding: 10px 16px; border-radius: 999px; background: var(--color-ob); color: #081127; font-size: 12px; font-weight: 700; text-decoration: none; }
.studio-stats { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 10px; margin: 18px 0; }
.studio-stats article { position: relative; padding: 18px; overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 16px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.studio-stats article > span { position: absolute; top: 10px; right: 11px; color: var(--text-ob-dim); font-size: 9px; }
.studio-stats strong { display: block; font-size: 1.8rem; }
.studio-stats small { color: var(--text-ob-dim); font-size: 11px; }
.studio-dashboard__grid { display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(280px, .65fr); gap: 18px; }
.studio-panel { padding: 24px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.studio-panel header { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 20px; }
.studio-panel h2 { margin: 0; font-size: 1.25rem; }
.studio-panel form { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.studio-panel label { display: grid; gap: 7px; color: var(--text-ob-dim); font-size: 11px; }
.studio-panel label.wide, .studio-form-actions.wide { grid-column: 1 / -1; }
.studio-panel input, .studio-panel textarea { width: 100%; padding: 10px 12px; border: 1px solid var(--border-hairline); border-radius: 10px; outline: none; background: var(--background-primary); color: inherit; font: inherit; resize: vertical; }
.studio-panel input:focus, .studio-panel textarea:focus { border-color: var(--color-ob); }
.studio-form-actions { display: flex; align-items: center; gap: 14px; }
.studio-form-actions button { padding: 9px 16px; border: 0; border-radius: 999px; background: var(--color-ob); color: #081127; font-weight: 700; cursor: pointer; }
.studio-form-actions a { color: var(--text-ob-dim); font-size: 12px; }
.studio-quick { display: grid; gap: 10px; }
.studio-quick a { display: grid; grid-template-columns: 1fr auto; gap: 4px 12px; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 14px; color: inherit; text-decoration: none; }
.studio-quick a:hover { border-color: var(--color-ob); }
.studio-quick strong, .studio-quick span { display: block; }
.studio-quick span { color: var(--text-ob-dim); font-size: 11px; }
.studio-quick em { grid-column: 2; grid-row: 1 / 3; align-self: center; color: var(--color-ob); font-size: 11px; font-style: normal; }
@media (max-width: 1100px) { .studio-stats { grid-template-columns: repeat(3, 1fr); } .studio-dashboard__grid { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .studio-page-head, .studio-panel header { align-items: flex-start; flex-direction: column; } .studio-stats { grid-template-columns: repeat(2, 1fr); } .studio-panel form { grid-template-columns: 1fr; } }
</style>