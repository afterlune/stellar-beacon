<template>
  <div class="studio-content">
    <header class="studio-page-head">
      <div><p>WORKSPACE / {{ pageIndex }}</p><h1>{{ pageTitle }}</h1><span>{{ pageHint }}</span></div>
      <button type="button" @click="openNew">新建{{ kindLabel }} +</button>
    </header>

    <section class="studio-toolbar">
      <div class="studio-toolbar__tabs">
        <button v-for="item in statusTabs" :key="item.value" type="button" :class="{ active: status === item.value }" @click="changeStatus(item.value)">
          {{ item.label }}
        </button>
      </div>
      <label><input v-model.trim="keywords" placeholder="搜索我的内容" @keyup.enter="loadList(true)" /><button type="button" @click="loadList(true)">搜索</button></label>
    </section>

    <p v-if="loading" class="studio-state">正在读取私有内容…</p>
    <p v-else-if="error" class="studio-state is-error">{{ error }}</p>
    <div v-else-if="records.length" class="studio-record-list">
      <article v-for="item in records" :key="item.id" class="studio-record">
        <template v-if="kind === 'article'">
          <div class="studio-record__main">
            <span class="studio-badge" :class="`status-${item.status}`">{{ statusLabel(item.status) }}</span>
            <span v-if="item.moderationStatus === 'hidden'" class="studio-badge is-danger">已隐藏</span>
            <h2>{{ item.articleTitle }}</h2>
            <p>{{ item.categoryName || '未分类' }} · {{ formatDate(item.createTime) }}</p>
            <small v-if="item.moderationReason">审核说明：{{ item.moderationReason }}</small>
          </div>
          <div class="studio-record__metrics"><span>{{ item.viewsCount || 0 }} 阅读</span><span>{{ item.likeCount || 0 }} 赞</span><span>{{ item.favoriteCount || 0 }} 收藏</span></div>
        </template>
        <template v-else-if="kind === 'talk'">
          <div class="studio-record__main">
            <span class="studio-badge" :class="`status-${item.status}`">{{ statusLabel(item.status) }}</span>
            <h2 class="is-talk">{{ item.content }}</h2>
            <p>{{ formatDate(item.createTime) }}</p>
            <small v-if="item.moderationReason">审核说明：{{ item.moderationReason }}</small>
          </div>
        </template>
        <template v-else>
          <div class="studio-record__main">
            <span class="studio-badge" :class="`status-${item.status}`">{{ statusLabel(item.status) }}</span>
            <h2>{{ item.seriesName }}</h2>
            <p>{{ item.seriesDesc || '暂无系列说明' }} · {{ item.articleCount }} 篇文章</p>
          </div>
        </template>
        <div class="studio-record__actions">
          <button type="button" @click="openEdit(item)">编辑</button>
          <button type="button" class="is-danger" @click="remove(item)">删除</button>
        </div>
      </article>
    </div>
    <p v-else class="studio-state">这里还没有内容。点击右上角开始创作。</p>

    <button v-if="records.length < total" type="button" class="studio-more" :disabled="loading" @click="loadMore">加载更多</button>

    <Teleport to="body">
    <div v-if="editorOpen" class="studio-editor" @click.self="editorOpen = false">
      <form @submit.prevent="save">
        <header>
          <div><p>{{ form.id ? 'EDIT' : 'CREATE' }}</p><h2>{{ form.id ? '编辑' : '新建' }}{{ kindLabel }}</h2></div>
          <button type="button" @click="editorOpen = false">×</button>
        </header>

        <div v-if="kind === 'article'" class="studio-editor__grid">
          <label class="wide">标题<input v-model.trim="form.articleTitle" required maxlength="150" /></label>
          <label class="wide">正文<textarea v-model="form.articleContent" rows="14" required placeholder="支持 Markdown / HTML 文本，发布后公开展示。" /></label>
          <label>封面地址<input v-model.trim="form.articleCover" placeholder="https://... 或上传后的地址" /></label>
          <label>可见性<select v-model="form.visibility"><option value="public">公开</option><option value="private">私有</option><option value="draft">草稿</option><option value="scheduled">定时公开</option></select></label>
          <label v-if="form.visibility === 'scheduled'">计划发布时间<input v-model="form.scheduledAt" type="datetime-local" required /></label>
          <label v-if="form.visibility === 'public'">访问密码<input v-model="form.password" type="password" placeholder="可留空" /></label>
          <label>分类<select v-model.number="form.categoryId"><option :value="0">未分类</option><option v-for="item in categories" :key="item.id" :value="item.id">{{ item.categoryName }}</option></select></label>
          <label>标签<select v-model="form.tagIds" multiple size="4"><option v-for="item in tags" :key="item.id" :value="item.id">#{{ item.tagName }}</option></select></label>
          <label>所属系列<select v-model.number="form.seriesId"><option :value="0">不属于系列</option><option v-for="item in series" :key="item.id" :value="item.id">{{ item.seriesName }}</option></select></label>
          <label>系列顺序<input v-model.number="form.seriesOrder" type="number" min="0" /></label>
          <label class="wide">原文链接<input v-model.trim="form.originalUrl" placeholder="转载时填写" /></label>
        </div>

        <div v-else-if="kind === 'talk'" class="studio-editor__grid">
          <label class="wide">随想内容<textarea v-model="form.content" rows="8" required maxlength="1000" /></label>
          <label class="wide">图片地址<textarea v-model="form.images" rows="3" placeholder="多个地址用英文逗号分隔" /></label>
          <label>可见性<select v-model="form.visibility"><option value="public">公开</option><option value="private">私有</option><option value="draft">草稿</option></select></label>
          <label class="is-check"><input v-model="form.isTop" type="checkbox" /> 置顶到我的公开主页</label>
        </div>

        <div v-else class="studio-editor__grid">
          <label class="wide">系列名称<input v-model.trim="form.seriesName" required maxlength="50" /></label>
          <label class="wide">系列说明<textarea v-model.trim="form.seriesDesc" rows="4" maxlength="255" /></label>
          <label>封面地址<input v-model.trim="form.cover" /></label>
          <label>可见性<select v-model="form.visibility"><option value="public">公开</option><option value="private">私有</option><option value="draft">草稿</option></select></label>
        </div>

        <footer><button type="button" @click="editorOpen = false">取消</button><button type="submit" :disabled="saving">{{ saving ? '保存中…' : '保存' }}</button></footer>
      </form>
    </div>
    </Teleport>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import api from '@/api/api'

type Kind = 'article' | 'talk' | 'series'

export default defineComponent({
  name: 'StudioContent',
  props: { kind: { type: String as () => Kind, required: true } },
  setup(props) {
    const route = useRoute()
    const router = useRouter()
    const records = ref<any[]>([])
    const categories = ref<any[]>([])
    const tags = ref<any[]>([])
    const series = ref<any[]>([])
    const loading = ref(false)
    const saving = ref(false)
    const error = ref('')
    const editorOpen = ref(false)
    const status = ref(0)
    const keywords = ref('')
    const page = ref(1)
    const total = ref(0)
    const pageSize = 12
    const form = reactive<any>(emptyForm(props.kind))

    const kindLabel = computed(() => props.kind === 'article' ? '文章' : props.kind === 'talk' ? '随想' : '系列')
    const pageTitle = computed(() => `${kindLabel.value}工作台`)
    const pageHint = computed(() => props.kind === 'article' ? '公开、私有、草稿与定时发布都由你决定。' : props.kind === 'talk' ? '发布轻量想法，也可只留给自己。' : '用自己的系列整理长期写作路径。')
    const pageIndex = computed(() => props.kind === 'article' ? '02' : props.kind === 'talk' ? '03' : '04')
    const statusTabs = [
      { value: 0, label: '全部' },
      { value: 1, label: '公开' },
      { value: 2, label: '私有' },
      { value: 3, label: '草稿' },
      { value: 4, label: '定时' }
    ]

    function emptyForm(kind: Kind) {
      if (kind === 'article') return { id: 0, articleTitle: '', articleContent: '', articleContentHtml: '', articleCover: '', categoryId: 0, tagIds: [] as number[], seriesId: 0, seriesOrder: 0, scheduledAt: '', visibility: 'draft', type: 1, password: '', originalUrl: '' }
      if (kind === 'talk') return { id: 0, content: '', images: '', isTop: 0, visibility: 'public' }
      return { id: 0, seriesName: '', seriesDesc: '', cover: '', visibility: 'draft' }
    }

    const dataOf = (response: any) => response?.data?.data || {}
    const listRequest = (params: any) => props.kind === 'article' ? api.getStudioArticles(params) : props.kind === 'talk' ? api.getStudioTalks(params) : api.getStudioSeries(params)

    const loadList = async (reset = false) => {
      if (reset) { page.value = 1; records.value = [] }
      loading.value = true
      error.value = ''
      try {
        const response = await listRequest({ current: page.value, size: pageSize, status: status.value || undefined, keywords: keywords.value || undefined })
        const data = dataOf(response)
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
      } catch (reason: any) {
        error.value = reason?.response?.data?.message || '内容加载失败'
      } finally {
        loading.value = false
      }
    }

    const loadOptions = async () => {
      if (props.kind !== 'article') return
      try {
        const [categoryResponse, tagResponse, seriesResponse] = await Promise.all([
          api.getStudioCategories(), api.getStudioTags(), api.getStudioSeries({ current: 1, size: 100 })
        ])
        categories.value = dataOf(categoryResponse) || []
        tags.value = dataOf(tagResponse) || []
        series.value = dataOf(seriesResponse).records || []
      } catch {
        categories.value = []; tags.value = []; series.value = []
      }
    }

    const resetForm = (value: any) => {
      Object.keys(form).forEach((key) => delete form[key])
      Object.assign(form, emptyForm(props.kind), value)
    }
    const openNew = () => { resetForm({}); editorOpen.value = true }
    const openEdit = async (item: any) => {
      try {
        const response = props.kind === 'article' ? await api.getStudioArticle(item.id) : props.kind === 'talk' ? await api.getStudioTalk(item.id) : await api.getStudioSeriesItem(item.id)
        const detail = dataOf(response)
        const visibility = ({ 1: 'public', 2: 'private', 3: 'draft', 4: 'scheduled' } as Record<number, string>)[Number(detail.status)] || 'draft'
        if (props.kind === 'article') {
          resetForm({ ...detail, visibility, tagIds: Array.isArray(detail.tagNames) ? detail.tagNames : [], scheduledAt: detail.scheduledAt ? String(detail.scheduledAt).slice(0, 16) : '' })
        } else if (props.kind === 'talk') {
          resetForm({ ...detail, visibility })
        } else {
          resetForm({ ...detail, visibility })
        }
        editorOpen.value = true
      } catch {
        ElMessage.error('内容详情加载失败')
      }
    }

    const save = async () => {
      saving.value = true
      try {
        let response: any
        if (props.kind === 'article') {
          const payload = {
            id: Number(form.id || 0), articleTitle: form.articleTitle, articleContent: form.articleContent,
            articleContentHtml: form.articleContentHtml || '', articleCover: form.articleCover, categoryId: Number(form.categoryId || 0),
            tagIds: (form.tagIds || []).map(Number), seriesId: Number(form.seriesId || 0), seriesOrder: Number(form.seriesOrder || 0),
            scheduledAt: form.scheduledAt ? new Date(form.scheduledAt).toISOString() : '', visibility: form.visibility,
            type: Number(form.type || 1), password: form.password, originalUrl: form.originalUrl
          }
          response = await api.saveStudioArticle(payload, payload.id)
        } else if (props.kind === 'talk') {
          const payload = { id: Number(form.id || 0), content: form.content, images: form.images, isTop: form.isTop ? 1 : 0, visibility: form.visibility }
          response = await api.saveStudioTalk(payload, payload.id)
        } else {
          const payload = { id: Number(form.id || 0), seriesName: form.seriesName, seriesDesc: form.seriesDesc, cover: form.cover, visibility: form.visibility }
          response = await api.saveStudioSeries(payload, payload.id)
        }
        if (!response?.data?.flag) throw new Error(response?.data?.message || 'save failed')
        ElMessage.success(`${kindLabel.value}已保存`)
        editorOpen.value = false
        await Promise.all([loadList(true), loadOptions()])
      } catch (reason: any) {
        ElMessage.error(reason?.response?.data?.message || reason?.message || '保存失败')
      } finally {
        saving.value = false
      }
    }

    const remove = async (item: any) => {
      try {
        await ElMessageBox.confirm(`确认删除这条${kindLabel.value}吗？`, '删除确认', { type: 'warning' })
        if (props.kind === 'article') await api.deleteStudioArticles([item.id])
        else if (props.kind === 'talk') await api.deleteStudioTalks([item.id])
        else await api.deleteStudioSeries(item.id)
        ElMessage.success('已删除')
        await loadList(true)
      } catch (reason: any) {
        if (reason !== 'cancel' && reason !== 'close') ElMessage.error('删除失败')
      }
    }

    const changeStatus = (value: number) => { status.value = value; void loadList(true) }
    const loadMore = () => { page.value += 1; void loadList(false) }
    const statusLabel = (value: number) => ({ 1: '公开', 2: '私有', 3: '草稿', 4: '定时' } as Record<number, string>)[value] || '未知'
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''

    watch(() => props.kind, () => {
      resetForm({})
      status.value = 0
      keywords.value = ''
      router.replace(route.path)
      void loadList(true)
      void loadOptions()
    })

    onMounted(async () => {
      await Promise.all([loadList(true), loadOptions()])
      if (route.query.new === '1') openNew()
    })

    return { records, categories, tags, series, loading, saving, error, editorOpen, status, statusTabs, keywords, page, total, form, kindLabel, pageTitle, pageHint, pageIndex, loadList, loadMore, openNew, openEdit, save, remove, changeStatus, statusLabel, formatDate }
  }
})
</script>

<style lang="scss" scoped>
.studio-content { min-width: 0; }
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 28px 30px; border: 1px solid var(--border-hairline); border-radius: 20px; background: radial-gradient(circle at 86% 0, rgba(97, 73, 184, .18), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p { margin: 0 0 7px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 6px; font-size: clamp(1.8rem, 4vw, 3rem); letter-spacing: -.05em; }
.studio-page-head span { color: var(--text-ob-dim); font-size: 12px; }
.studio-page-head button { padding: 10px 16px; border: 0; border-radius: 999px; background: var(--color-ob); color: #081127; font-weight: 700; cursor: pointer; }
.studio-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin: 16px 0; }
.studio-toolbar__tabs { display: flex; gap: 7px; flex-wrap: wrap; }
.studio-toolbar__tabs button { padding: 7px 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.studio-toolbar__tabs button.active { border-color: var(--color-ob); color: var(--color-ob); background: color-mix(in srgb, var(--color-ob) 11%, transparent); }
.studio-toolbar > label { display: flex; overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 999px; background: var(--background-primary-alt); }
.studio-toolbar input { width: 210px; padding: 8px 12px; border: 0; outline: 0; background: transparent; color: inherit; }
.studio-toolbar label button { border: 0; background: transparent; color: var(--color-ob); cursor: pointer; }
.studio-record-list { display: grid; gap: 10px; }
.studio-record { display: flex; align-items: center; gap: 18px; padding: 18px; border: 1px solid var(--border-hairline); border-radius: 15px; background: color-mix(in srgb, var(--background-primary-alt) 88%, transparent); }
.studio-record__main { min-width: 0; flex: 1; }
.studio-record h2 { margin: 8px 0 5px; overflow: hidden; font-size: 1rem; text-overflow: ellipsis; white-space: nowrap; }
.studio-record h2.is-talk { white-space: normal; line-height: 1.65; }
.studio-record p, .studio-record small { color: var(--text-ob-dim); font-size: 11px; }
.studio-record small { display: block; margin-top: 5px; color: #df8b63; }
.studio-record__metrics { display: flex; gap: 14px; color: var(--text-ob-dim); font-size: 11px; }
.studio-record__actions { display: flex; gap: 7px; }
.studio-record__actions button { padding: 7px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; font-size: 11px; cursor: pointer; }
.studio-record__actions button:hover { border-color: var(--color-ob); color: var(--color-ob); }
.studio-record__actions button.is-danger:hover { border-color: #d86d62; color: #e3887e; }
.studio-badge { display: inline-block; padding: 3px 8px; border-radius: 999px; background: rgba(92, 118, 177, .18); color: #9fb7ed; font-size: 10px; }
.studio-badge.status-1 { background: rgba(60, 166, 119, .16); color: #7ed5a5; }
.studio-badge.status-2 { background: rgba(183, 133, 62, .16); color: #e2b878; }
.studio-badge.status-3 { background: rgba(130, 130, 150, .16); color: #b8b8c8; }
.studio-badge.status-4 { background: rgba(115, 76, 191, .18); color: #bca0ef; }
.studio-badge.is-danger { margin-left: 5px; background: rgba(203, 78, 70, .16); color: #e98b82; }
.studio-state { padding: 56px 0; color: var(--text-ob-dim); text-align: center; }
.studio-state.is-error { color: #e2776c; }
.studio-more { display: block; margin: 24px auto 0; padding: 8px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-editor { position: fixed; z-index: 1000; inset: 0; display: grid; place-items: center; padding: 20px; background: rgba(3, 7, 20, .74); backdrop-filter: blur(9px); }
.studio-editor form { width: min(860px, 100%); max-height: calc(100vh - 40px); overflow-y: auto; padding: 26px; border: 1px solid var(--border-hairline); border-radius: 20px; background: var(--background-primary-alt); box-shadow: 0 30px 100px rgba(0, 0, 0, .45); }
.studio-editor form > header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 22px; }
.studio-editor header p { margin: 0 0 5px; color: var(--color-ob); font-size: 9px; letter-spacing: .18em; }
.studio-editor h2 { margin: 0; }
.studio-editor header button { border: 0; background: transparent; color: var(--text-ob-dim); font-size: 25px; cursor: pointer; }
.studio-editor__grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.studio-editor label { display: grid; gap: 7px; color: var(--text-ob-dim); font-size: 11px; }
.studio-editor label.wide { grid-column: 1 / -1; }
.studio-editor input, .studio-editor textarea, .studio-editor select { width: 100%; padding: 10px 12px; border: 1px solid var(--border-hairline); border-radius: 10px; outline: none; background: var(--background-primary); color: inherit; font: inherit; }
.studio-editor textarea { resize: vertical; line-height: 1.65; }
.studio-editor input:focus, .studio-editor textarea:focus, .studio-editor select:focus { border-color: var(--color-ob); }
.studio-editor label.is-check { display: flex; align-items: center; align-self: end; padding-bottom: 11px; }
.studio-editor label.is-check input { width: auto; }
.studio-editor footer { display: flex; justify-content: flex-end; gap: 10px; margin-top: 22px; }
.studio-editor footer button { padding: 9px 18px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-editor footer button[type='submit'] { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
@media (max-width: 720px) { .studio-toolbar { align-items: stretch; flex-direction: column; } .studio-record { align-items: flex-start; flex-direction: column; } .studio-record__metrics { flex-wrap: wrap; } .studio-editor__grid { grid-template-columns: 1fr; } .studio-editor label.wide { grid-column: auto; } }
</style>