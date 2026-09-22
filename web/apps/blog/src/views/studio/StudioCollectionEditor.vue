<template>
  <section class="collection-editor">
    <header class="collection-editor__head">
      <div><router-link to="/studio/collections">← 返回书单</router-link><h1>{{ detail?.collection?.title || '编辑书单' }}</h1></div>
      <div class="collection-editor__actions">
        <a v-if="publicSlug && detail?.collection?.visibility !== 'private'" :href="`/collections/${publicSlug}`" target="_blank" rel="noopener">查看公开页</a>
        <button type="button" :disabled="saving" @click="saveMetadata">{{ saving ? '保存中…' : '保存设置' }}</button>
      </div>
    </header>
    <p v-if="loading" class="editor-state">正在加载书单…</p>
    <template v-else-if="detail">
      <p v-if="detail.collection.moderationStatus === 'hidden'" class="moderation-banner">该公开书单已被隐藏：{{ detail.collection.moderationReason || '内容审核未通过' }}</p>
      <form class="collection-form" @submit.prevent="saveMetadata">
        <label><span>标题</span><input v-model.trim="form.title" maxlength="80" /></label>
        <label><span>简介</span><textarea v-model.trim="form.description" maxlength="500" rows="3" /></label>
        <label><span>可见性</span><select v-model="form.visibility"><option value="private">私有</option><option value="unlisted">链接可见</option><option value="public">公开</option></select></label>
      </form>

      <section class="collection-builder">
        <header><div><h2>收录文章</h2><span>拖动排序，推荐语保存后随书单公开。</span></div><small>{{ items.length }} / 500</small></header>
        <div class="collection-search">
          <input v-model.trim="keywords" placeholder="搜索要收录的公开文章…" @keyup.enter="searchArticles" />
          <button type="button" :disabled="searching" @click="searchArticles">{{ searching ? '搜索中…' : '搜索' }}</button>
        </div>
        <div v-if="searchResults.length" class="collection-search-results">
          <article v-for="item in searchResults" :key="item.id">
            <div><strong>{{ item.articleTitle }}</strong><small>{{ item.categoryName || '未分类' }}</small></div>
            <button type="button" @click="addArticle(item)">加入</button>
          </article>
        </div>
        <div v-if="items.length" class="collection-builder-list">
          <article v-for="(item, index) in items" :key="item.articleId" draggable="true" @dragstart="dragIndex = index" @dragover.prevent @drop="dropItem(index)">
            <span class="drag-handle">⋮⋮</span>
            <div class="collection-builder-main">
              <template v-if="item.article"><strong>{{ item.article.articleTitle }}</strong><small>{{ item.article.author?.nickname || item.article.author?.handle }} · {{ item.article.categoryName || '未分类' }}</small></template>
              <template v-else><strong>文章已不可公开</strong><small>作者可能已转为私有、删除或审核隐藏</small></template>
              <input v-model.trim="item.note" maxlength="280" placeholder="写一句推荐语…" @blur="saveNote(item)" />
            </div>
            <div class="collection-builder-actions"><button type="button" :disabled="index === 0" @click="move(index, -1)">↑</button><button type="button" :disabled="index === items.length - 1" @click="move(index, 1)">↓</button><button type="button" class="danger" @click="removeItem(item)">移除</button></div>
          </article>
        </div>
        <p v-else class="editor-state">还没有文章，先搜索加入。</p>
      </section>
    </template>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/api'

export default defineComponent({
  name: 'StudioCollectionEditor',
  setup() {
    const route = useRoute()
    const collectionId = computed(() => Number(route.params.id || 0))
    const detail = ref<any>(null)
    const items = ref<any[]>([])
    const loading = ref(false)
    const saving = ref(false)
    const searching = ref(false)
    const keywords = ref('')
    const searchResults = ref<any[]>([])
    const dragIndex = ref(-1)
    const form = reactive({ title: '', description: '', visibility: 'private' })
    const publicSlug = computed(() => String(detail.value?.collection?.slug || ''))
    const load = async () => {
      loading.value = true
      try {
        const response = await api.getStudioCollection(collectionId.value)
        detail.value = response?.data?.data || null
        items.value = Array.isArray(detail.value?.items) ? detail.value.items : []
        form.title = String(detail.value?.collection?.title || '')
        form.description = String(detail.value?.collection?.description || '')
        form.visibility = String(detail.value?.collection?.visibility || 'private')
      } finally { loading.value = false }
    }
    const saveMetadata = async () => {
      if (!form.title.trim()) return
      saving.value = true
      try { const response = await api.updateStudioCollection(collectionId.value, { ...form }); detail.value.collection = response?.data?.data || detail.value.collection } finally { saving.value = false }
    }
    const searchArticles = async () => {
      if (!keywords.value) return
      searching.value = true
      try { const response = await api.searchArticles({ keywords: keywords.value, current: 1, size: 10 }); const data = response?.data?.data || {}; searchResults.value = Array.isArray(data.items) ? data.items : Array.isArray(data.records) ? data.records : [] } finally { searching.value = false }
    }
    const addArticle = async (article: any) => {
      const articleId = Number(article.id)
      const response = await api.addStudioCollectionItem(collectionId.value, articleId, '')
      if (!response?.data?.flag) return
      await load()
    }
    const removeItem = async (item: any) => {
      await api.removeStudioCollectionItem(collectionId.value, Number(item.articleId))
      items.value = items.value.filter((row) => row.articleId !== item.articleId)
      await saveOrder()
    }
    const saveNote = async (item: any) => { await api.addStudioCollectionItem(collectionId.value, Number(item.articleId), String(item.note || '')) }
    const saveOrder = async () => { if (items.value.length) await api.reorderStudioCollection(collectionId.value, items.value.map((item) => Number(item.articleId))) }
    const move = async (index: number, delta: number) => {
      const target = index + delta
      if (target < 0 || target >= items.value.length) return
      const next = [...items.value]
      ;[next[index], next[target]] = [next[target], next[index]]
      items.value = next
      await saveOrder()
    }
    const dropItem = async (index: number) => {
      if (dragIndex.value < 0 || dragIndex.value === index) return
      const next = [...items.value]
      const [moved] = next.splice(dragIndex.value, 1)
      next.splice(index, 0, moved)
      items.value = next
      dragIndex.value = -1
      await saveOrder()
    }
    onMounted(() => void load())
    return { detail, items, loading, saving, searching, keywords, searchResults, dragIndex, form, publicSlug, load, saveMetadata, searchArticles, addArticle, removeItem, saveNote, move, dropItem }
  }
})
</script>

<style scoped>
.collection-editor__head { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; padding: 22px; border: 1px solid var(--border-hairline); border-radius: 18px; }
.collection-editor__head a, .collection-editor__actions a { color: var(--color-ob); font-size: 12px; text-decoration: none; }
.collection-editor__head h1 { margin: 9px 0 0; }
.collection-editor__actions { display: flex; align-items: center; gap: 10px; }
.collection-editor__actions button, .collection-search button, .collection-builder-actions button, .collection-search-results button { min-height: 30px; padding: 6px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.collection-editor__actions button { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
.moderation-banner { margin: 14px 0 0; padding: 12px 15px; border: 1px solid rgba(239, 140, 127, .5); border-radius: 12px; color: #ef8c7f; font-size: 12px; }
.collection-form { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 2fr) 180px; gap: 14px; margin-top: 16px; padding: 18px; border: 1px solid var(--border-hairline); border-radius: 16px; }
.collection-form label span { display: block; margin-bottom: 7px; color: var(--text-ob-dim); font-size: 11px; }
.collection-form input, .collection-form textarea, .collection-form select, .collection-builder-main input, .collection-search input { width: 100%; box-sizing: border-box; padding: 9px 11px; border: 1px solid var(--border-hairline); border-radius: 10px; background: var(--background-primary); color: inherit; font: inherit; }
.collection-builder { margin-top: 16px; padding: 18px; border: 1px solid var(--border-hairline); border-radius: 16px; }
.collection-builder > header { display: flex; justify-content: space-between; }
.collection-builder h2 { margin: 0 0 4px; } .collection-builder header span, .collection-builder header small { color: var(--text-ob-dim); font-size: 11px; }
.collection-search { display: flex; gap: 8px; margin-top: 16px; } .collection-search input { flex: 1; }
.collection-search-results { display: grid; gap: 7px; margin-top: 10px; padding: 9px; border: 1px solid var(--border-hairline); border-radius: 12px; }
.collection-search-results article { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.collection-search-results strong, .collection-search-results small { display: block; } .collection-search-results small { color: var(--text-ob-dim); font-size: 10px; }
.collection-builder-list { display: grid; gap: 9px; margin-top: 14px; }
.collection-builder-list > article { display: grid; grid-template-columns: 28px minmax(0, 1fr) auto; gap: 10px; align-items: center; padding: 12px; border: 1px solid var(--border-hairline); border-radius: 13px; background: color-mix(in srgb, var(--background-primary-alt) 80%, transparent); }
.drag-handle { color: var(--text-ob-dim); cursor: grab; }
.collection-builder-main strong, .collection-builder-main small { display: block; } .collection-builder-main small { margin: 3px 0 8px; color: var(--text-ob-dim); font-size: 10px; }
.collection-builder-actions { display: flex; gap: 5px; } .collection-builder-actions button { min-width: 30px; padding: 5px 8px; } .collection-builder-actions .danger { color: #ef8c7f; }
.editor-state { padding: 50px 0; color: var(--text-ob-dim); text-align: center; }
@media (max-width: 760px) { .collection-form { grid-template-columns: 1fr; } .collection-editor__head { align-items: flex-start; flex-direction: column; } .collection-builder-list > article { grid-template-columns: 20px minmax(0, 1fr); } .collection-builder-actions { grid-column: 2; } }
</style>
