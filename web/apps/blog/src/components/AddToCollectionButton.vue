<template>
  <button type="button" class="collection-add-button" @click="open" data-testid="add-to-collection">加入书单</button>
  <el-dialog v-model="visible" title="加入书单" width="460px">
    <p class="collection-add-hint">选择收藏集，或将这篇文章加入新书单。</p>
    <div v-if="loading" class="collection-add-state">加载中…</div>
    <div v-else class="collection-add-list">
      <button v-for="item in collections" :key="item.id" type="button" :class="{ active: added.has(Number(item.id)) }" @click="add(item)">
        <span><strong>{{ item.title }}</strong><small>{{ visibilityLabel(item.visibility) }} · {{ item.articleCount }} 篇</small></span>
        <em>{{ added.has(Number(item.id)) ? '已加入' : '加入' }}</em>
      </button>
    </div>
    <div class="collection-add-create">
      <input v-model.trim="newTitle" placeholder="新书单标题" maxlength="80" @keyup.enter="createAndAdd" />
      <select v-model="newVisibility"><option value="private">私有</option><option value="unlisted">链接可见</option><option value="public">公开</option></select>
      <button type="button" :disabled="!newTitle || creating" @click="createAndAdd">{{ creating ? '创建中…' : '新建并加入' }}</button>
    </div>
  </el-dialog>
</template>

<script lang="ts">
import { defineComponent, getCurrentInstance, onMounted, ref } from 'vue'
import api from '@/api/api'

export default defineComponent({
  name: 'AddToCollectionButton',
  props: { articleId: { type: Number, required: true } },
  setup(props) {
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const visible = ref(false)
    const loading = ref(false)
    const creating = ref(false)
    const collections = ref<any[]>([])
    const added = ref<Set<number>>(new Set())
    const newTitle = ref('')
    const newVisibility = ref('private')
    const load = async () => {
      loading.value = true
      try { const response = await api.getStudioCollections({ current: 1, size: 100 }); const data = response?.data?.data || {}; collections.value = Array.isArray(data.items) ? data.items : [] } catch { collections.value = [] } finally { loading.value = false }
    }
    const open = () => { visible.value = true; void load() }
    const add = async (item: any) => {
      const response = await api.addStudioCollectionItem(Number(item.id), props.articleId, '')
      if (!response?.data?.flag) return
      added.value = new Set([...added.value, Number(item.id)])
      proxy?.$notify?.({ title: '成功', message: `已加入「${item.title}」`, type: 'success' })
    }
    const createAndAdd = async () => {
      if (!newTitle.value) return
      creating.value = true
      try {
        const created = await api.createStudioCollection({ title: newTitle.value, description: '', visibility: newVisibility.value })
        const id = Number(created?.data?.data?.id || 0)
        if (id > 0) { await api.addStudioCollectionItem(id, props.articleId, ''); newTitle.value = ''; await load(); added.value = new Set([...added.value, id]) }
      } finally { creating.value = false }
    }
    const visibilityLabel = (value: string) => value === 'public' ? '公开' : value === 'unlisted' ? '链接可见' : '私有'
    onMounted(() => undefined)
    return { visible, loading, creating, collections, added, newTitle, newVisibility, open, add, createAndAdd, visibilityLabel }
  }
})
</script>

<style scoped>
.collection-add-button { min-height: 34px; padding: 7px 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.collection-add-hint { color: var(--text-ob-dim); font-size: 12px; }
.collection-add-list { display: grid; gap: 7px; max-height: 280px; overflow: auto; }
.collection-add-list button { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 12px; border: 1px solid var(--border-hairline); border-radius: 11px; background: transparent; color: inherit; text-align: left; cursor: pointer; }
.collection-add-list button.active { border-color: var(--color-ob); color: var(--color-ob); }
.collection-add-list strong, .collection-add-list small { display: block; } .collection-add-list small { margin-top: 3px; color: var(--text-ob-dim); font-size: 10px; } .collection-add-list em { font-size: 11px; font-style: normal; }
.collection-add-create { display: grid; grid-template-columns: minmax(0, 1fr) 90px auto; gap: 7px; margin-top: 15px; padding-top: 15px; border-top: 1px solid var(--border-hairline); }
.collection-add-create input, .collection-add-create select { min-width: 0; padding: 8px 9px; border: 1px solid var(--border-hairline); border-radius: 9px; background: var(--background-primary); color: inherit; }
.collection-add-create button { border: 1px solid var(--border-hairline); border-radius: 999px; background: var(--color-ob); color: #081127; cursor: pointer; }
.collection-add-state { padding: 30px 0; color: var(--text-ob-dim); text-align: center; }
@media (max-width: 520px) { .collection-add-create { grid-template-columns: 1fr; } }
</style>
