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

      <section class="collection-builder governance">
        <header>
          <div><h2>评论治理</h2><span>勾选后可批量删除、置顶或恢复；批量置顶只保留最新的一条根评论，删除根评论会连带隐藏其回复。</span></div>
          <small>{{ commentTotal }} 条</small>
        </header>
        <div class="collection-search">
          <input v-model.trim="commentKeywords" placeholder="搜索评论内容…" @keyup.enter="loadComments" />
          <button type="button" :disabled="commentLoading" @click="loadComments">{{ commentLoading ? '加载中…' : '搜索' }}</button>
        </div>
        <div class="governance-actions">
          <span>已选 {{ selectedCommentIds.length }} 条</span>
          <button type="button" class="danger" :disabled="!selectedCommentIds.length || commentBusy" @click="runBatch('delete')">批量删除</button>
          <button type="button" :disabled="!selectedCommentIds.length || commentBusy" @click="runBatch('pin')">批量置顶</button>
          <button type="button" :disabled="!selectedCommentIds.length || commentBusy" @click="runBatch('unpin')">取消置顶</button>
          <button type="button" :disabled="!selectedCommentIds.length || commentBusy" @click="restoreSelected">恢复选中</button>
        </div>
        <p v-if="governanceMessage" class="governance-message" data-testid="governance-message">{{ governanceMessage }}</p>
        <div v-if="comments.length" class="governance-list">
          <article v-for="comment in comments" :key="comment.id" :class="{ 'is-deleted': Number(comment.isDelete) === 1 }">
            <input v-model="selectedCommentIds" type="checkbox" :value="Number(comment.id)" />
            <div class="governance-main">
              <strong>{{ comment.commentContent }}</strong>
              <small>{{ comment.nickname }} · {{ formatCommentTime(comment.createTime) }} · 回复 {{ comment.replyCount }} · 举报 {{ comment.reportCount }}</small>
            </div>
            <div class="governance-tags">
              <span v-if="Number(comment.isDelete) === 1" class="tag tag--deleted">已删除</span>
              <span v-if="Number(comment.isTop) === 1" class="tag tag--pinned">置顶</span>
            </div>
          </article>
        </div>
        <p v-else-if="!commentLoading" class="editor-state">还没有符合条件的评论。</p>
        <div v-if="commentTotal > commentsPageSize" class="governance-pager">
          <button type="button" :disabled="commentsPage <= 1" @click="changeCommentPage(-1)">上一页</button>
          <span>第 {{ commentsPage }} 页</span>
          <button type="button" :disabled="commentsPage * commentsPageSize >= commentTotal" @click="changeCommentPage(1)">下一页</button>
        </div>
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
    const comments = ref<any[]>([])
    const commentTotal = ref(0)
    const commentLoading = ref(false)
    const commentBusy = ref(false)
    const commentKeywords = ref('')
    const commentsPage = ref(1)
    const commentsPageSize = 20
    const selectedCommentIds = ref<number[]>([])
    const governanceMessage = ref('')
    const publicSlug = computed(() => String(detail.value?.collection?.slug || ''))
    const formatCommentTime = (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '')
    const loadComments = async () => {
      if (!collectionId.value) return
      commentLoading.value = true
      try {
        const response = await api.listOwnedCollectionComments(collectionId.value, {
          current: commentsPage.value,
          size: commentsPageSize,
          keywords: commentKeywords.value || undefined,
          includeDeleted: '1'
        })
        const data = response?.data?.data || {}
        comments.value = Array.isArray(data.items) ? data.items : Array.isArray(data.records) ? data.records : []
        commentTotal.value = Number(data.total || 0)
        selectedCommentIds.value = []
      } catch {
        comments.value = []
        commentTotal.value = 0
      } finally {
        commentLoading.value = false
      }
    }
    const changeCommentPage = async (delta: number) => {
      const next = commentsPage.value + delta
      if (next < 1) return
      if (delta > 0 && commentsPage.value * commentsPageSize >= commentTotal.value) return
      commentsPage.value = next
      await loadComments()
    }
    const summarizeGovernance = (result: any) => {
      const succeeded = Array.isArray(result?.succeeded) ? result.succeeded.length : 0
      const failures = Array.isArray(result?.failed) ? result.failed : []
      const reasons = Array.from(new Set(failures.map((row: any) => String(row?.message || '')).filter(Boolean)))
      return `成功 ${succeeded} 条` + (failures.length ? `，失败 ${failures.length} 条：${reasons.join('、')}` : '')
    }
    const governanceRequest = async (send: () => Promise<any>) => {
      commentBusy.value = true
      try {
        const response = await send()
        if (!response?.data?.flag) throw new Error(response?.data?.message || '操作失败')
        governanceMessage.value = summarizeGovernance(response.data.data)
        await loadComments()
      } catch (reason: any) {
        governanceMessage.value = reason?.response?.data?.message || reason?.message || '操作失败'
      } finally {
        commentBusy.value = false
      }
    }
    const selectedIds = () => selectedCommentIds.value.map(Number).filter((id) => id > 0)
    const runBatch = async (action: 'delete' | 'pin' | 'unpin') => {
      const ids = selectedIds()
      if (!ids.length) return
      if (action === 'pin' && ids.length > 1) {
        governanceMessage.value = '批量置顶只保留最新的一条根评论，其余会自动取消置顶。'
      }
      await governanceRequest(() => api.batchModerateCollectionComments(collectionId.value, action, ids))
    }
    const restoreSelected = async () => {
      const ids = selectedIds()
      if (!ids.length) return
      await governanceRequest(() => api.restoreOwnedCollectionComments(collectionId.value, ids))
    }
    const load = async () => {
      loading.value = true
      try {
        const response = await api.getStudioCollection(collectionId.value)
        detail.value = response?.data?.data || null
        items.value = Array.isArray(detail.value?.items) ? detail.value.items : []
        form.title = String(detail.value?.collection?.title || '')
        form.description = String(detail.value?.collection?.description || '')
        form.visibility = String(detail.value?.collection?.visibility || 'private')
        await loadComments()
      } finally {
        loading.value = false
      }
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
    return {
      detail, items, loading, saving, searching, keywords, searchResults, dragIndex, form, publicSlug,
      comments, commentTotal, commentLoading, commentBusy, commentKeywords, commentsPage, commentsPageSize, selectedCommentIds, governanceMessage,
      load, saveMetadata, searchArticles, addArticle, removeItem, saveNote, move, dropItem,
      loadComments, changeCommentPage, formatCommentTime, runBatch, restoreSelected
    }
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
.governance-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 12px; }
.governance-actions span { color: var(--text-ob-dim); font-size: 11px; }
.governance-actions button { min-height: 30px; padding: 6px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.governance-actions button:disabled { opacity: .5; cursor: not-allowed; }
.governance-actions .danger { color: #ef8c7f; border-color: rgba(239, 140, 127, .5); }
.governance-message { margin: 12px 0 0; padding: 9px 12px; border: 1px solid var(--border-hairline); border-radius: 10px; color: var(--color-ob); font-size: 11px; }
.governance-list { display: grid; gap: 8px; margin-top: 12px; }
.governance-list > article { display: grid; grid-template-columns: 22px minmax(0, 1fr) auto; gap: 10px; align-items: center; padding: 11px 12px; border: 1px solid var(--border-hairline); border-radius: 12px; background: color-mix(in srgb, var(--background-primary-alt) 80%, transparent); }
.governance-list > article.is-deleted { opacity: .62; }
.governance-main strong, .governance-main small { display: block; }
.governance-main strong { font-size: 12px; font-weight: 500; word-break: break-word; }
.governance-main small { margin-top: 4px; color: var(--text-ob-dim); font-size: 10px; }
.governance-tags { display: flex; gap: 6px; }
.governance-tags .tag { padding: 2px 8px; border-radius: 999px; font-size: 10px; }
.governance-tags .tag--deleted { background: rgba(239, 140, 127, .18); color: #ef8c7f; }
.governance-tags .tag--pinned { background: color-mix(in srgb, var(--color-ob) 22%, transparent); color: var(--color-ob); }
.governance-pager { display: flex; align-items: center; gap: 10px; margin-top: 12px; }
.governance-pager span { color: var(--text-ob-dim); font-size: 11px; }
.governance-pager button { min-height: 28px; padding: 5px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.governance-pager button:disabled { opacity: .5; cursor: not-allowed; }
.editor-state { padding: 50px 0; color: var(--text-ob-dim); text-align: center; }
@media (max-width: 760px) { .collection-form { grid-template-columns: 1fr; } .collection-editor__head { align-items: flex-start; flex-direction: column; } .collection-builder-list > article { grid-template-columns: 20px minmax(0, 1fr); } .collection-builder-actions { grid-column: 2; } }
</style>
