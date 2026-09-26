<template>
  <div class="studio-content">
    <header class="studio-page-head">
      <div>
        <p>WORKSPACE / {{ pageIndex }}</p>
        <h1>{{ pageTitle }}</h1>
        <span>{{ pageHint }}</span>
      </div>
      <button type="button" @click="openNew">新建{{ kindLabel }} +</button>
    </header>

    <section class="studio-toolbar">
      <div class="studio-toolbar__tabs">
        <button
          v-for="item in statusTabs"
          :key="item.value"
          type="button"
          :class="{ active: status === item.value }"
          @click="changeStatus(item.value)">
          {{ item.label }}
        </button>
      </div>
      <label>
        <input v-model.trim="keywords" placeholder="搜索我的内容" @keyup.enter="loadList(true)" />
        <button type="button" @click="loadList(true)">搜索</button>
      </label>
    </section>

    <section v-if="records.length" class="studio-batch-bar">
      <label>
        <input type="checkbox" :checked="allLoadedSelected" :aria-label="`全选当前已加载的${kindLabel}`" @change="toggleAllLoaded" />
        全选当前已加载
      </label>
      <span>已选 {{ selectionCount }} 项</span>
      <button v-if="canSelectAllMatching" type="button" class="is-plain" @click="selectAllMatchingItems">选择全部 {{ total }} 条</button>
      <div>
        <select v-model.number="batchStatus" aria-label="批量状态">
          <option :value="1">公开</option>
          <option :value="2">私有</option>
          <option :value="3">草稿</option>
        </select>
        <button type="button" :disabled="!selectionCount" @click="applyBatchStatus">应用状态</button>
        <button type="button" class="is-danger" :disabled="!selectionCount" @click="batchDelete">批量删除</button>
        <button v-if="selectionCount" type="button" class="is-plain" @click="clearSelection">取消选择</button>
      </div>
    </section>

    <p v-if="loading && !records.length" class="studio-state">正在读取创作内容…</p>
    <p v-else-if="error" class="studio-state is-error">{{ error }}</p>
    <div v-else-if="records.length" class="studio-record-list">
      <article v-for="item in records" :key="item.id" class="studio-record" :class="{ 'is-selected': isSelected(item.id) }">
        <label class="studio-record__select">
          <input type="checkbox" :checked="isSelected(item.id)" :aria-label="`选择${item.articleTitle || item.content || item.seriesName}`" @change="toggleSelection(item.id)" />
        </label>

        <template v-if="kind === 'article'">
          <div class="studio-record__main">
            <span class="studio-badge" :class="`status-${item.status}`">{{ statusLabel(item.status) }}</span>
            <span v-if="item.moderationStatus === 'hidden'" class="studio-badge is-danger">已隐藏</span>
            <h2>{{ item.articleTitle }}</h2>
            <p>{{ item.categoryName || '未分类' }} · {{ formatDate(item.createTime) }}</p>
            <p v-if="item.status === 4 && item.scheduledAt" class="studio-schedule">
              计划发布：{{ formatDateTime(item.scheduledAt) }}
            </p>
            <small v-if="item.moderationReason">审核说明：{{ item.moderationReason }}</small>
          </div>
          <div class="studio-record__metrics">
            <span>{{ item.viewsCount || 0 }} 阅读</span>
            <span>{{ item.likeCount || 0 }} 赞</span>
            <span>{{ item.favoriteCount || 0 }} 收藏</span>
          </div>
        </template>
        <template v-else-if="kind === 'talk'">
          <div class="studio-record__main">
            <span class="studio-badge" :class="`status-${item.status}`">{{ statusLabel(item.status) }}</span>
            <span v-if="item.moderationStatus === 'hidden'" class="studio-badge is-danger">已隐藏</span>
            <h2 class="is-talk">{{ item.content }}</h2>
            <p>{{ formatDate(item.createTime) }}</p>
            <small v-if="item.moderationReason">审核说明：{{ item.moderationReason }}</small>
          </div>
        </template>
        <template v-else>
          <div class="studio-record__main">
            <span class="studio-badge" :class="`status-${item.status}`">{{ statusLabel(item.status) }}</span>
            <span v-if="item.moderationStatus === 'hidden'" class="studio-badge is-danger">已隐藏</span>
            <h2>{{ item.seriesName }}</h2>
            <p>{{ item.seriesDesc || '暂无系列说明' }} · {{ item.articleCount }} 篇文章</p>
            <small v-if="item.moderationReason">审核说明：{{ item.moderationReason }}</small>
          </div>
        </template>

        <div class="studio-record__actions">
          <button type="button" @click="openPreview(item)">预览</button>
          <button v-if="canShare(item)" type="button" @click="copyPublicLink(item)">复制链接</button>
          <button type="button" @click="openEdit(item)">编辑</button>
          <button type="button" class="is-danger" @click="remove(item)">删除</button>
        </div>
      </article>
    </div>
    <p v-else class="studio-state">这里还没有内容。点击右上角开始创作。</p>

    <button v-if="records.length < total" type="button" class="studio-more" :disabled="loading" @click="loadMore">
      {{ loading ? '加载中…' : '加载更多' }}
    </button>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { StudioBatchScope } from '@stellar-beacon/api-contract'
import { notify } from '@/services/notifications'
import { confirm } from '@/services/confirm'
import api from '@/api/api'

type Kind = 'article' | 'talk' | 'series'

export default defineComponent({
  name: 'StudioContent',
  props: {
    kind: { type: String as () => Kind, required: true }
  },
  setup(props) {
    const route = useRoute()
    const router = useRouter()
    const records = ref<any[]>([])
    const loading = ref(false)
    const error = ref('')
    const status = ref(0)
    const keywords = ref('')
    const page = ref(1)
    const total = ref(0)
    const selectedIds = ref<number[]>([])
    const matchingSelection = ref<{ maxId: number; count: number; status: number; keywords: string; seriesId: number; excludeIds: number[]; hiddenCount: number } | null>(null)
    const batchStatus = ref(1)
    const pageSize = 12

    const kindLabel = computed(() => props.kind === 'article' ? '文章' : props.kind === 'talk' ? '随想' : '系列')
    const pageTitle = computed(() => `${kindLabel.value}工作台`)
    const pageHint = computed(() => props.kind === 'article' ? '公开、私有、草稿与定时发布都由你决定。' : props.kind === 'talk' ? '发布轻量想法，也可只留给自己。' : '用自己的系列整理长期写作路径。')
    const pageIndex = computed(() => props.kind === 'article' ? '02' : props.kind === 'talk' ? '03' : '04')
    const kindPath = computed(() => props.kind === 'article' ? 'articles' : props.kind === 'talk' ? 'talks' : 'series')
    const statusTabs = computed(() => {
      const tabs = [
        { value: 0, label: '全部' },
        { value: 1, label: '公开' },
        { value: 2, label: '私有' },
        { value: 3, label: '草稿' }
      ]
      if (props.kind === 'article') tabs.push({ value: 4, label: '定时' })
      return tabs
    })
    const selectedRecords = computed(() => records.value.filter((item) => isSelected(item.id)))
    const allLoadedSelected = computed(() => records.value.length > 0 && records.value.every((item) => isSelected(item.id)))
    const selectionCount = computed(() => matchingSelection.value ? matchingSelection.value.count - matchingSelection.value.excludeIds.length : selectedIds.value.length)
    const canSelectAllMatching = computed(() => total.value > 0 && matchingSelection.value === null)

    const dataOf = (response: any) => response?.data?.data || {}
    const listRequest = (params: any) => props.kind === 'article' ? api.getStudioArticles(params) : props.kind === 'talk' ? api.getStudioTalks(params) : api.getStudioSeries(params)

    const loadList = async (reset = false) => {
      if (reset) {
        page.value = 1
        records.value = []
        clearSelection()
      }
      loading.value = true
      error.value = ''
      try {
        const response = await listRequest({
          current: page.value,
          size: pageSize,
          status: status.value || undefined,
          keywords: keywords.value || undefined
        })
        const data = dataOf(response)
        const next = Array.isArray(data.records) ? data.records : Array.isArray(data.items) ? data.items : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count ?? data.total ?? 0)
      } catch (reason: any) {
        error.value = reason?.response?.data?.message || '内容加载失败'
      } finally {
        loading.value = false
      }
    }

    const openNew = () => router.push(`/studio/${kindPath.value}/new`)
    const openEdit = (item: any) => router.push(`/studio/${kindPath.value}/${item.id}/edit`)
    const openPreview = (item: any) => router.push(`/studio/${kindPath.value}/${item.id}/preview`)
    const publicPath = (item: any) => props.kind === 'article' ? `/articles/${item.id}` : props.kind === 'talk' ? `/talks/${item.id}` : `/series/${item.id}`
    const canShare = (item: any) => Number(item.status) === 1 && item.moderationStatus !== 'hidden'

    const copyPublicLink = async (item: any) => {
      const url = new URL(publicPath(item), window.location.origin).toString()
      try {
        if (navigator.clipboard?.writeText) {
          await navigator.clipboard.writeText(url)
        } else {
          const input = document.createElement('input')
          input.value = url
          document.body.appendChild(input)
          input.select()
          document.execCommand('copy')
          input.remove()
        }
        notify.success('公开链接已复制')
      } catch {
        notify.error('复制失败，请手动打开公开页')
      }
    }

    const remove = async (item: any) => {
      if (!await confirm({ title: '删除确认', message: `确认删除这条${kindLabel.value}吗？`, confirmText: '确认删除' })) return
      try {
        const response = await api.batchDeleteStudioContent({ kind: props.kind, scope: { mode: 'ids', ids: [item.id] } })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '删除失败')
        notify.success('已删除')
        await loadList(true)
      } catch (reason: any) { notify.error(reason?.response?.data?.message || '删除失败') }
    }

    const changeStatus = (value: number) => {
      if (status.value === value) return
      status.value = value
      const query = { ...route.query, status: value ? String(value) : undefined }
      void router.replace({ path: route.path, query })
      void loadList(true)
    }
    const loadMore = () => {
      page.value += 1
      void loadList(false)
    }
    const statusLabel = (value: number) => ({ 1: '公开', 2: '私有', 3: '草稿', 4: '定时' } as Record<number, string>)[value] || '未知'
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''
    const formatDateTime = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : ''
    const isSelected = (id: number) => {
      const value = Number(id)
      return matchingSelection.value ? !matchingSelection.value.excludeIds.includes(value) : selectedIds.value.includes(value)
    }
    const toggleSelection = (id: number) => {
      const value = Number(id)
      if (matchingSelection.value) {
        const excluded = matchingSelection.value.excludeIds
        matchingSelection.value.excludeIds = excluded.includes(value) ? excluded.filter((item) => item !== value) : [...excluded, value]
        return
      }
      selectedIds.value = isSelected(value) ? selectedIds.value.filter((item) => item !== value) : [...selectedIds.value, value]
    }
    const toggleAllLoaded = () => {
      if (matchingSelection.value || !allLoadedSelected.value) {
        matchingSelection.value = null
        selectedIds.value = records.value.map((item) => Number(item.id))
        return
      }
      clearSelection()
    }
    const clearSelection = () => {
      selectedIds.value = []
      matchingSelection.value = null
    }
    const currentFilter = () => ({ status: status.value, keywords: keywords.value, seriesId: 0 })
    const selectAllMatchingItems = async () => {
      try {
        const response = await api.previewStudioContent({ kind: props.kind, ...currentFilter() })
        const preview = dataOf(response)
        if (!response?.data?.flag || Number(preview.count || 0) <= 0) {
          notify.warning('当前筛选条件下没有可操作内容')
          return
        }
        selectedIds.value = []
        matchingSelection.value = {
          maxId: Number(preview.maxId),
          count: Number(preview.count),
          ...currentFilter(),
          excludeIds: [],
          hiddenCount: Number(preview.hiddenCount || 0)
        }
      } catch (reason: any) {
        notify.error(reason?.response?.data?.message || '无法读取全部匹配内容')
      }
    }
    const batchVisibility = (value: number): 'public' | 'private' | 'draft' => value === 2 ? 'private' : value === 3 ? 'draft' : 'public'
    const selectionScope = (): StudioBatchScope => matchingSelection.value
      ? {
          mode: 'filter',
          status: matchingSelection.value.status || 0,
          keywords: matchingSelection.value.keywords || undefined,
          seriesId: matchingSelection.value.seriesId || undefined,
          maxId: matchingSelection.value.maxId,
          excludeIds: matchingSelection.value.excludeIds,
          expectedCount: matchingSelection.value.count - matchingSelection.value.excludeIds.length
        }
      : { mode: 'ids', ids: selectedIds.value }
    const titleOf = (item: any) => item.articleTitle || item.content || item.seriesName || `#${item.id}`

    const applyBatchStatus = async () => {
      if (!selectionCount.value) return
      const targetCount = selectionCount.value
      const hiddenCount = matchingSelection.value
        ? matchingSelection.value.hiddenCount
        : selectedRecords.value.filter((item) => item.moderationStatus === 'hidden').length
      if (batchStatus.value === 1) {
        const preview = matchingSelection.value
          ? `将按当前筛选条件处理 ${targetCount} 条内容。`
          : selectedRecords.value.slice(0, 5).map((item) => `· ${titleOf(item)}`).join('\n')
        const moderation = hiddenCount ? `\n其中 ${hiddenCount} 条已被审核隐藏，改状态后仍不会公开可见。` : ''
        if (!await confirm({ title: '确认批量公开', message: `将 ${targetCount} 条内容批量公开：\n${preview}${moderation}`, confirmText: '确认公开' })) return
      } else {
        if (!await confirm({ title: '批量修改状态', message: `确认将 ${targetCount} 条内容改为“${statusLabel(batchStatus.value)}”？` })) return
      }
      try {
        const response = await api.batchUpdateStudioContentStatus({
          kind: props.kind,
          scope: selectionScope(),
          visibility: batchVisibility(batchStatus.value)
        })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '批量修改失败')
        notify.success(`已更新 ${Number(response.data.data?.affected ?? targetCount)} 条内容`)
        clearSelection()
        await loadList(true)
      } catch (reason: any) {
        notify.error(reason?.response?.data?.message || reason?.message || '批量修改失败')
      }
    }

    const batchDelete = async () => {
      if (!selectionCount.value) return
      const targetCount = selectionCount.value
      if (!await confirm({ title: '批量删除', message: `确认删除已选择的 ${targetCount} 条${kindLabel.value}吗？此操作不可撤销。`, confirmText: '确认删除' })) return
      try {
        const response = await api.batchDeleteStudioContent({ kind: props.kind, scope: selectionScope() })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '批量删除失败')
        notify.success(`已删除 ${Number(response.data.data?.affected ?? targetCount)} 条内容`)
        clearSelection()
        await loadList(true)
      } catch (reason: any) { notify.error(reason?.response?.data?.message || reason?.message || '批量删除失败') }
    }
    watch(() => props.kind, () => {
      status.value = Number(route.query.status || 0)
      keywords.value = ''
      void loadList(true)
    })
    watch(() => route.query.status, (value) => {
      const next = Number(value || 0)
      if (next === status.value) return
      status.value = next
      void loadList(true)
    })

    onMounted(() => {
      const queryStatus = Number(route.query.status || 0)
      status.value = statusTabs.value.some((item) => item.value === queryStatus) ? queryStatus : 0
      void loadList(true)
    })
    onBeforeUnmount(clearSelection)

    return {
      records, loading, error, status, statusTabs, keywords, page, total, batchStatus, selectedIds, selectionCount, canSelectAllMatching, allLoadedSelected,
      kindLabel, pageTitle, pageHint, pageIndex, kindPath, loadList, loadMore, openNew, openEdit, openPreview,
      copyPublicLink, canShare, remove, changeStatus, statusLabel, formatDate, formatDateTime,
      isSelected, toggleSelection, toggleAllLoaded, selectAllMatchingItems, clearSelection, applyBatchStatus, batchDelete
    }
  }
})
</script>

<style lang="scss" scoped>
.studio-content { min-width: 0; }
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 28px 30px; border: 1px solid var(--border-hairline); border-radius: 20px; background: radial-gradient(circle at 86% 0, rgba(97, 73, 184, .18), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p { margin: 0 0 7px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 6px; font-size: clamp(1.8rem, 4vw, 3rem); letter-spacing: -.05em; }
.studio-page-head span { color: var(--text-ob-dim); font-size: 12px; }
.studio-page-head button, .studio-toolbar label button { padding: 9px 15px; border: 1px solid var(--border-hairline); border-radius: 999px; background: var(--color-ob); color: #081127; font-weight: 700; cursor: pointer; }
.studio-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin: 16px 0; }
.studio-toolbar__tabs { display: flex; gap: 7px; flex-wrap: wrap; }
.studio-toolbar__tabs button { padding: 7px 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.studio-toolbar__tabs button.active { border-color: var(--color-ob); color: var(--color-ob); }
.studio-toolbar label { display: flex; min-width: min(340px, 100%); }
.studio-toolbar input { min-width: 0; flex: 1; padding: 9px 12px; border: 1px solid var(--border-hairline); border-right: 0; border-radius: 999px 0 0 999px; outline: none; background: var(--background-primary); color: inherit; }
.studio-toolbar label button { border-radius: 0 999px 999px 0; }
.studio-batch-bar { display: flex; align-items: center; gap: 16px; margin: 0 0 12px; padding: 10px 14px; border: 1px solid var(--border-hairline); border-radius: 12px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); font-size: 12px; }
.studio-batch-bar > label { display: flex; align-items: center; gap: 7px; }
.studio-batch-bar > span { color: var(--text-ob-dim); }
.studio-batch-bar > div { display: flex; gap: 7px; margin-left: auto; }
.studio-batch-bar select, .studio-batch-bar button { padding: 7px 10px; border: 1px solid var(--border-hairline); border-radius: 9px; background: transparent; color: inherit; font: inherit; }
.studio-batch-bar button { cursor: pointer; }
.studio-batch-bar button:disabled { opacity: .4; cursor: not-allowed; }
.studio-batch-bar button.is-danger { color: #df8177; }
.studio-batch-bar button.is-plain { color: var(--text-ob-dim); }
.studio-state { margin: 32px 0; color: var(--text-ob-dim); text-align: center; }
.studio-state.is-error { color: #df8177; }
.studio-record-list { display: grid; gap: 10px; }
.studio-record { display: grid; grid-template-columns: 24px minmax(0, 1fr) auto auto; gap: 16px; align-items: center; padding: 18px 20px; border: 1px solid var(--border-hairline); border-radius: 15px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.studio-record.is-selected { border-color: color-mix(in srgb, var(--color-ob) 55%, transparent); background: color-mix(in srgb, var(--color-ob) 6%, var(--background-primary-alt)); }
.studio-record__select { align-self: start; padding-top: 4px; }
.studio-record__main { min-width: 0; }
.studio-record h2 { margin: 8px 0 5px; overflow: hidden; font-size: 1.05rem; text-overflow: ellipsis; white-space: nowrap; }
.studio-record h2.is-talk { white-space: normal; }
.studio-record p, .studio-record small { color: var(--text-ob-dim); font-size: 11px; }
.studio-record small, .studio-schedule { display: block; margin-top: 6px; }
.studio-record small { color: #df8177; }
.studio-schedule { color: var(--color-ob) !important; font-weight: 600; }
.studio-badge { display: inline-block; padding: 3px 8px; border: 1px solid var(--border-hairline); border-radius: 999px; color: var(--text-ob-dim); font-size: 10px; }
.studio-badge.status-1 { border-color: rgba(98, 201, 177, .55); color: #78d0bb; }
.studio-badge.status-2 { border-color: rgba(231, 166, 82, .55); color: #e7a652; }
.studio-badge.status-4 { border-color: rgba(137, 108, 220, .55); color: #bca0ef; }
.studio-badge.is-danger { margin-left: 6px; border-color: rgba(223, 129, 119, .55); color: #df8177; }
.studio-record__metrics { display: flex; gap: 10px; color: var(--text-ob-dim); font-size: 10px; }
.studio-record__actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 6px; }
.studio-record__actions button { padding: 7px 10px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-record__actions button.is-danger { color: #df8177; }
.studio-more { display: block; margin: 24px auto 0; padding: 8px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 900px) { .studio-record { grid-template-columns: 24px minmax(0, 1fr) auto; } .studio-record__metrics { grid-column: 2 / -1; grid-row: 2; } }
@media (max-width: 760px) { .studio-page-head, .studio-toolbar, .studio-batch-bar { align-items: stretch; flex-direction: column; } .studio-batch-bar > div { display: grid; grid-template-columns: 1fr 1fr; margin-left: 0; } .studio-record { grid-template-columns: 24px minmax(0, 1fr); } .studio-record__metrics, .studio-record__actions { grid-column: 2; }
  .studio-record__actions { justify-content: flex-start; }
}
</style>
