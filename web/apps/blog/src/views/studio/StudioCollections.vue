<template>
  <section class="studio-collections">
    <header class="studio-page-head">
      <div><p>READING LISTS</p><h1>书单与收藏集</h1><span>整理公开文章，设置私有、链接可见或公开分享。</span></div>
      <button type="button" @click="openCreate">新建书单</button>
    </header>
    <p v-if="loading" class="studio-state">正在加载书单…</p>
    <div v-else-if="records.length" class="collection-admin-grid">
      <article v-for="item in records" :key="item.id">
        <router-link :to="`/studio/collections/${item.id}/edit`" class="collection-admin-cover">
          <img v-if="item.cover" :src="item.cover" :alt="item.title" loading="lazy" />
          <span v-else>{{ String(item.title).slice(0, 1) }}</span>
        </router-link>
        <div>
          <div class="collection-admin-meta"><span :class="`visibility-${item.visibility}`">{{ visibilityLabel(item.visibility) }}</span><small>{{ item.articleCount }} 篇</small><em v-if="item.moderationStatus === 'hidden'">审核隐藏</em></div>
          <h2>{{ item.title }}</h2>
          <p>{{ item.description || '暂无简介' }}</p>
          <footer><small>更新于 {{ formatDate(item.updatedAt) }}</small><router-link :to="`/studio/collections/${item.id}/edit`">编辑 →</router-link></footer>
        </div>
      </article>
    </div>
    <p v-else class="studio-state">还没有书单，先整理一组公开文章吧。</p>

    <DialogSurface v-model="createVisible" title="新建书单">
      <form class="collection-create-form" @submit.prevent="create">
        <label>标题<input v-model="form.title" maxlength="80" required /></label>
        <label>简介<textarea v-model="form.description" rows="3" maxlength="500" /></label>
        <label>可见性
          <select v-model="form.visibility">
            <option value="private">私有：仅自己可见</option>
            <option value="unlisted">链接可见：不公开列出</option>
            <option value="public">公开：进入主页与发现页</option>
          </select>
        </label>
        <footer><button type="button" class="dialog-secondary" @click="createVisible = false">取消</button><button type="submit" class="dialog-primary" :disabled="creating">{{ creating ? '创建中…' : '创建并编辑' }}</button></footer>
      </form>
    </DialogSurface>
  </section>
</template>

<script lang="ts">
import { defineComponent, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api/api'
import DialogSurface from '@/components/overlays/DialogSurface.vue'

export default defineComponent({
  name: 'StudioCollections',
  components: { DialogSurface },
  setup() {
    const router = useRouter()
    const records = ref<any[]>([])
    const loading = ref(false)
    const creating = ref(false)
    const createVisible = ref(false)
    const form = reactive({ title: '', description: '', visibility: 'private' })
    const load = async () => {
      loading.value = true
      try { const response = await api.getStudioCollections({ current: 1, size: 100 }); const data = response?.data?.data || {}; records.value = Array.isArray(data.items) ? data.items : [] } catch { records.value = [] } finally { loading.value = false }
    }
    const openCreate = () => { form.title = ''; form.description = ''; form.visibility = 'private'; createVisible.value = true }
    const create = async () => {
      if (!form.title.trim()) return
      creating.value = true
      try {
        const response = await api.createStudioCollection({ ...form })
        const item = response?.data?.data || {}
        createVisible.value = false
        await router.push(`/studio/collections/${item.id}/edit`)
      } finally { creating.value = false }
    }
    const visibilityLabel = (value: string) => value === 'public' ? '公开' : value === 'unlisted' ? '链接可见' : '私有'
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''
    onMounted(() => void load())
    return { records, loading, creating, createVisible, form, load, openCreate, create, visibilityLabel, formatDate }
  }
})
</script>

<style scoped>
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; padding: 24px; border: 1px solid var(--border-hairline); border-radius: 20px; background: color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 7px; font-size: 2rem; }
.studio-page-head span { color: var(--text-ob-dim); font-size: 12px; }
.studio-page-head button, .dialog-primary, .dialog-secondary { min-height: 34px; padding: 7px 14px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-page-head button, .dialog-primary { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
.collection-admin-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; margin-top: 18px; }
.collection-admin-grid > article { display: grid; grid-template-columns: 150px minmax(0, 1fr); overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 16px; background: color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.collection-admin-cover { display: block; min-height: 180px; background: linear-gradient(135deg, #152250, #3c2c70); color: rgba(218, 228, 255, .9); text-decoration: none; }
.collection-admin-cover img { width: 100%; height: 100%; object-fit: cover; }
.collection-admin-cover span { display: grid; place-items: center; height: 100%; font-size: 3rem; font-weight: 900; }
.collection-admin-grid article > div { padding: 16px; }
.collection-admin-meta { display: flex; align-items: center; gap: 8px; font-size: 10px; }
.collection-admin-meta span, .collection-admin-meta em { padding: 3px 7px; border-radius: 999px; font-style: normal; }
.visibility-public { color: #b9dd70; } .visibility-unlisted { color: #7eb9ff; } .visibility-private { color: var(--text-ob-dim); } .collection-admin-meta em { color: #ef8c7f; }
.collection-admin-grid h2 { margin: 9px 0; font-size: 1.1rem; }
.collection-admin-grid p { min-height: 36px; color: var(--text-ob-dim); font-size: 11px; line-height: 1.6; }
.collection-admin-grid footer { display: flex; justify-content: space-between; margin-top: 15px; color: var(--text-ob-dim); font-size: 10px; }
.collection-admin-grid footer a { display: inline-flex; align-items: center; min-height: 24px; padding: 2px 0; color: var(--color-ob); text-decoration: none; }
.studio-state { padding: 60px 0; color: var(--text-ob-dim); text-align: center; }
.collection-create-form { display: grid; gap: 14px; }
.collection-create-form label { display: grid; gap: 6px; color: var(--text-ob-dim); font-size: 12px; }
.collection-create-form input, .collection-create-form textarea, .collection-create-form select { width: 100%; box-sizing: border-box; padding: 9px 11px; border: 1px solid var(--border-hairline); border-radius: 9px; background: var(--background-primary-alt); color: var(--text-normal); font: inherit; }
.collection-create-form footer { display: flex; justify-content: flex-end; gap: 9px; }
@media (max-width: 760px) { .collection-admin-grid { grid-template-columns: 1fr; } }
</style>
