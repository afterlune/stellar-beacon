<template>
  <div class="studio-editor-page">
    <header class="studio-editor-head">
      <button type="button" class="studio-editor-back" @click="backToList">← 返回{{ kindLabel }}列表</button>
      <div>
        <p>{{ isNew ? 'CREATE' : 'EDIT' }} / {{ kindLabel }}</p>
        <h1>{{ isNew ? `新建${kindLabel}` : `编辑${kindLabel}` }}</h1>
      </div>
      <span class="studio-editor-save-state">{{ saveState }}</span>
    </header>

    <p v-if="loading" class="studio-editor-state">正在载入编辑内容…</p>
    <p v-else-if="error" class="studio-editor-state is-error">{{ error }}</p>

    <form v-else class="studio-editor-form" @submit.prevent="save">
      <div class="studio-editor-layout">
        <main class="studio-editor-main">
          <template v-if="kind === 'article'">
            <label class="studio-field studio-field--title">
              <span>标题</span>
              <input v-model.trim="form.articleTitle" required maxlength="50" placeholder="写下一个清晰的标题" />
              <small>{{ form.articleTitle.length }}/50</small>
            </label>

            <div class="studio-editor-modebar">
              <div>
                <button type="button" :class="{ active: workspaceMode === 'write' }" @click="workspaceMode = 'write'">编辑</button>
                <button type="button" :class="{ active: workspaceMode === 'preview' }" @click="workspaceMode = 'preview'">预览</button>
              </div>
              <div v-if="workspaceMode === 'write'">
                <button type="button" :class="{ active: contentMode === 'visual' }" @click="switchContentMode('visual')">富文本</button>
                <button type="button" :class="{ active: contentMode === 'source' }" @click="switchContentMode('source')">Markdown / HTML</button>
              </div>
            </div>

            <div v-if="workspaceMode === 'preview'" class="studio-article-preview" v-html="articlePreviewHtml"></div>
            <div v-else-if="contentMode === 'visual'" class="studio-rich-editor">
              <Toolbar :editor="editorRef" :default-config="toolbarConfig" mode="default" />
              <Editor
                v-model="form.articleContentHtml"
                :default-config="editorConfig"
                mode="default"
                @on-created="handleEditorCreated"
                @on-change="handleEditorChange"
                @on-destroyed="handleEditorDestroyed" />
            </div>
            <label v-else class="studio-field studio-field--source">
              <span>Markdown / HTML 源码</span>
              <textarea v-model="form.articleContent" rows="20" placeholder="支持 Markdown 或 HTML，切换到富文本后会转换并安全预览。" />
            </label>
          </template>

          <template v-else-if="kind === 'talk'">
            <label class="studio-field studio-field--title">
              <span>随想内容</span>
              <textarea v-model.trim="form.content" required maxlength="2000" rows="8" placeholder="记录此刻的想法…" />
              <small>{{ form.content.length }}/2000</small>
            </label>
            <section class="studio-talk-images">
              <header><strong>图片</strong><span>{{ form.talkImages.length }}/9</span></header>
              <div v-if="form.talkImages.length" class="studio-talk-images__grid">
                <article v-for="(image, index) in form.talkImages" :key="`${image}-${index}`">
                  <img :src="image" alt="" />
                  <div>
                    <button type="button" :disabled="Number(index) === 0" @click="moveTalkImage(Number(index), -1)">←</button>
                    <button type="button" :disabled="Number(index) === form.talkImages.length - 1" @click="moveTalkImage(Number(index), 1)">→</button>
                    <button type="button" class="danger" @click="removeTalkImage(Number(index))">删除</button>
                  </div>
                </article>
              </div>
              <button v-if="form.talkImages.length < 9" type="button" class="studio-upload-button" :disabled="uploadingTalkImages" @click="talkImageInput?.click()">
                {{ uploadingTalkImages ? '上传中…' : '添加图片' }}
              </button>
              <input ref="talkImageInput" type="file" accept="image/*" multiple hidden @change="handleTalkImages" />
            </section>
          </template>

          <template v-else>
            <label class="studio-field studio-field--title">
              <span>系列名称</span>
              <input v-model.trim="form.seriesName" required maxlength="50" placeholder="给长期主题一个名字" />
              <small>{{ form.seriesName.length }}/50</small>
            </label>
            <label class="studio-field">
              <span>系列简介</span>
              <textarea v-model.trim="form.seriesDesc" maxlength="255" rows="5" placeholder="说明这个系列关注什么。" />
              <small>{{ form.seriesDesc.length }}/255</small>
            </label>
          </template>
        </main>

        <aside class="studio-editor-sidebar">
          <section v-if="kind === 'article'">
            <h2>文章设置</h2>
            <label class="studio-field">
              <span>可见性</span>
              <select v-model="form.visibility">
                <option value="public">公开</option>
                <option value="private">私有</option>
                <option value="draft">草稿</option>
                <option value="scheduled">定时公开</option>
              </select>
            </label>
            <label v-if="form.visibility === 'scheduled'" class="studio-field">
              <span>计划发布时间</span>
              <input v-model="form.scheduledAt" type="datetime-local" required />
            </label>
            <label v-if="form.visibility === 'public'" class="studio-field">
              <span>访问密码</span>
              <input v-model="form.password" maxlength="255" type="password" placeholder="可留空" />
            </label>
            <label class="studio-field">
              <span>分类</span>
              <select v-model.number="form.categoryId">
                <option :value="0">未分类</option>
                <option v-for="item in categories" :key="item.id" :value="item.id">{{ item.categoryName }}</option>
              </select>
            </label>
            <label class="studio-field">
              <span>标签</span>
              <select v-model="form.tagIds" multiple size="5">
                <option v-for="item in tags" :key="item.id" :value="item.id">#{{ item.tagName }}</option>
              </select>
            </label>
            <label class="studio-field">
              <span>所属系列</span>
              <select v-model.number="form.seriesId">
                <option :value="0">不属于系列</option>
                <option v-for="item in series" :key="item.id" :value="item.id">{{ item.seriesName }}</option>
              </select>
            </label>
            <label class="studio-field">
              <span>系列顺序</span>
              <input v-model.number="form.seriesOrder" type="number" min="0" max="9999" :disabled="!form.seriesId" />
            </label>
            <label class="studio-field">
              <span>文章类型</span>
              <select v-model.number="form.type">
                <option :value="1">原创</option>
                <option :value="2">转载</option>
                <option :value="3">翻译</option>
              </select>
            </label>
            <label class="studio-field">
              <span>原文链接</span>
              <input v-model.trim="form.originalUrl" maxlength="255" placeholder="转载或翻译时填写" />
            </label>
          </section>

          <section v-else>
            <h2>{{ kind === 'talk' ? '随想设置' : '系列设置' }}</h2>
            <label class="studio-field">
              <span>可见性</span>
              <select v-model="form.visibility">
                <option value="public">公开</option>
                <option value="private">私有</option>
                <option value="draft">草稿</option>
              </select>
            </label>
            <label v-if="kind === 'talk'" class="studio-checkbox">
              <input v-model="form.isTop" type="checkbox" />
              <span>置顶这条随想</span>
            </label>
            <section v-else class="studio-cover-field">
              <span>系列封面</span>
              <img v-if="form.cover" :src="form.cover" alt="系列封面预览" />
              <button type="button" class="studio-upload-button" :disabled="uploadingCover" @click="seriesCoverInput?.click()">
                {{ uploadingCover ? '上传中…' : form.cover ? '更换封面' : '上传封面' }}
              </button>
              <button v-if="form.cover" type="button" class="studio-text-button" @click="form.cover = ''">移除</button>
              <input v-model.trim="form.cover" maxlength="1024" placeholder="也可以粘贴图片地址" />
              <input ref="seriesCoverInput" type="file" accept="image/*" hidden @change="handleSeriesCover" />
            </section>
          </section>

          <section v-if="kind === 'article'" class="studio-cover-field">
            <h2>文章封面</h2>
            <img v-if="form.articleCover" :src="form.articleCover" alt="文章封面预览" />
            <button type="button" class="studio-upload-button" :disabled="uploadingCover" @click="articleCoverInput?.click()">
              {{ uploadingCover ? '上传中…' : form.articleCover ? '更换封面' : '上传封面' }}
            </button>
            <button v-if="form.articleCover" type="button" class="studio-text-button" @click="form.articleCover = ''">移除</button>
            <input v-model.trim="form.articleCover" maxlength="1024" placeholder="也可以粘贴图片地址" />
            <input ref="articleCoverInput" type="file" accept="image/*" hidden @change="handleArticleCover" />
          </section>
        </aside>
      </div>

      <footer class="studio-editor-footer">
        <span>{{ draftStatus }}</span>
        <div>
          <a v-if="publicLink" :href="publicLink" target="_blank" rel="noopener noreferrer" class="studio-public-link">查看公开页</a>
          <button type="button" class="studio-secondary-button" @click="backToList">返回列表</button>
          <button type="submit" class="studio-primary-button" :disabled="saving">{{ saving ? '保存中…' : '保存内容' }}</button>
        </div>
      </footer>
    </form>

    <div v-if="leaveVisible" class="studio-leave-mask">
      <section>
        <h2>还有未保存的修改</h2>
        <p>内容已自动保存在当前浏览器，离开后再次打开可以恢复。</p>
        <div>
          <button type="button" @click="cancelLeave">继续编辑</button>
          <button type="button" class="primary" @click="leaveWithDraft">保存到本机并离开</button>
        </div>
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Editor, Toolbar } from '@wangeditor-next/editor-for-vue'
import type { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor-next/editor'
import '@wangeditor-next/editor/dist/css/style.css'

import api from '@/api/api'
import { useUnsavedGuard } from '@/composables/useUnsavedGuard'
import { clearStudioDraft, readStudioDraft, saveStudioDraft, stableSnapshot, studioDraftKey } from '@/composables/useStudioDraft'
import { useUserStore } from '@/stores/user'
import markdownToHtml, { sanitizePreviewHtml } from '@/utils/markdown'
import { MAX_STUDIO_UPLOAD_BYTES, parseTalkImages, uploadStudioImage } from '@/utils/studioUpload'

type Kind = 'article' | 'talk' | 'series'
type ContentMode = 'visual' | 'source'
type WorkspaceMode = 'write' | 'preview'

const props = defineProps<{ kind: Kind }>()
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const saveState = ref('')
const draftStatus = ref('')
const workspaceMode = ref<WorkspaceMode>('write')
const contentMode = ref<ContentMode>('visual')
const uploadingCover = ref(false)
const uploadingTalkImages = ref(false)
const categories = ref<any[]>([])
const tags = ref<any[]>([])
const series = ref<any[]>([])
const articleCoverInput = ref<HTMLInputElement>()
const seriesCoverInput = ref<HTMLInputElement>()
const talkImageInput = ref<HTMLInputElement>()
const editorRef = shallowRef<IDomEditor>()
let routeReloadSkipped = false
let draftTimer: number | undefined
let suppressDraftAutosave = false

const contentID = computed(() => Number(route.params.id || 0))
const isNew = computed(() => contentID.value <= 0)
const kindLabel = computed(() => props.kind === 'article' ? '文章' : props.kind === 'talk' ? '随想' : '系列')
const kindPath = computed(() => props.kind === 'article' ? 'articles' : props.kind === 'talk' ? 'talks' : 'series')
const currentUserID = computed(() => Number(userStore.userInfo?.userInfoId || userStore.userInfo?.id || 0))
const draftKey = computed(() => studioDraftKey(currentUserID.value, props.kind, contentID.value))

const emptyForm = () => {
  if (props.kind === 'article') {
    return {
      id: 0, articleTitle: '', articleContent: '', articleContentHtml: '', articleCover: '', categoryId: 0,
      tagIds: [] as number[], seriesId: 0, seriesOrder: 0, scheduledAt: '', visibility: 'draft', type: 1,
      password: '', originalUrl: ''
    }
  }
  if (props.kind === 'talk') {
    return { id: 0, content: '', talkImages: [] as string[], isTop: 0, visibility: 'public' }
  }
  return { id: 0, seriesName: '', seriesDesc: '', cover: '', visibility: 'draft' }
}

const form = reactive<Record<string, any>>(emptyForm())
const stableForm = () => stableSnapshot({ ...form, id: Number(form.id || 0) })
const { visible: leaveVisible, markClean, confirmLeave, cancelLeave } = useUnsavedGuard(stableForm)
let cleanFormSnapshot = ''
function markFormClean(): void {
  markClean()
  cleanFormSnapshot = stableForm()
}

const publicLink = computed(() => {
  if (!form.id || form.visibility !== 'public') return ''
  if (props.kind === 'article') return `/articles/${form.id}`
  if (props.kind === 'talk') return `/talks/${form.id}`
  return `/series/${form.id}`
})

const articlePreviewHtml = computed(() => {
  const source = contentMode.value === 'visual' ? String(form.articleContentHtml || '') : markdownToHtml(String(form.articleContent || ''))
  return sanitizePreviewHtml(source)
})

const toolbarConfig: Partial<IToolbarConfig> = {
  excludeKeys: ['group-video', 'insertVideo', 'uploadVideo', 'fullScreen']
}
const editorConfig: Partial<IEditorConfig> = {
  placeholder: '从这里开始写作…',
  maxLength: 100000,
  sanitizeHtml: (html) => sanitizePreviewHtml(html),
  MENU_CONF: {
    uploadImage: {
      maxFileSize: MAX_STUDIO_UPLOAD_BYTES,
      allowedFileTypes: ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'],
      customUpload(file: File, insertFn: (url: string, alt?: string, href?: string) => void) {
        void uploadStudioImage(file, 'article-inline')
          .then((url) => insertFn(url, file.name))
          .catch((reason) => ElMessage.error(reason?.message || '图片上传失败'))
      }
    }
  }
}

function normalizeEditorHtml(html: string): string {
  return /^(?:<p><br\s*\/?><\/p>|\s)*$/i.test(html) ? '' : html
}

function handleEditorCreated(editor: IDomEditor): void {
  editorRef.value = editor
}

function handleEditorChange(editor: IDomEditor): void {
  form.articleContentHtml = normalizeEditorHtml(editor.getHtml())
  form.articleContent = form.articleContentHtml
}

function handleEditorDestroyed(): void {
  editorRef.value = undefined
}

function switchContentMode(next: ContentMode): void {
  if (contentMode.value === next) return
  if (next === 'source') {
    form.articleContent = form.articleContentHtml || form.articleContent || ''
    form.articleContentHtml = ''
  } else {
    form.articleContentHtml = sanitizePreviewHtml(markdownToHtml(String(form.articleContent || '')))
    form.articleContent = form.articleContentHtml
  }
  contentMode.value = next
}

function resetForm(): void {
  Object.keys(form).forEach((key) => delete form[key])
  Object.assign(form, emptyForm())
}

function normalizeStatus(value: number): string {
  return ({ 1: 'public', 2: 'private', 3: 'draft', 4: 'scheduled' } as Record<number, string>)[value] || 'draft'
}

function applyArticle(detail: any): void {
  Object.assign(form, {
    id: Number(detail.id || 0), articleTitle: String(detail.articleTitle || ''),
    articleContent: String(detail.articleContent || ''), articleContentHtml: String(detail.articleContentHtml || ''),
    articleCover: String(detail.articleCover || ''), categoryId: Number(detail.categoryId || 0),
    tagIds: Array.isArray(detail.tagNames) ? detail.tagNames.map(Number) : [],
    seriesId: Number(detail.seriesId || 0), seriesOrder: Number(detail.seriesOrder || 0),
    scheduledAt: detail.scheduledAt ? new Date(detail.scheduledAt).toISOString().slice(0, 16) : '',
    visibility: normalizeStatus(Number(detail.status)), type: Number(detail.type || 1),
    password: String(detail.password || ''), originalUrl: String(detail.originalUrl || '')
  })
  contentMode.value = form.articleContentHtml ? 'visual' : 'source'
}

function applyTalk(detail: any): void {
  Object.assign(form, {
    id: Number(detail.id || 0), content: String(detail.content || ''),
    talkImages: parseTalkImages(detail.images ?? detail.imgs), isTop: Number(detail.isTop || 0),
    visibility: normalizeStatus(Number(detail.status))
  })
}

function applySeries(detail: any): void {
  Object.assign(form, {
    id: Number(detail.id || 0), seriesName: String(detail.seriesName || ''),
    seriesDesc: String(detail.seriesDesc || ''), cover: String(detail.cover || ''),
    visibility: normalizeStatus(Number(detail.status))
  })
}

async function loadOptions(): Promise<void> {
  if (props.kind !== 'article') return
  try {
    const [categoryResponse, tagResponse, seriesResponse] = await Promise.all([
      api.getStudioCategories(), api.getStudioTags(), api.getStudioSeries({ current: 1, size: 100 })
    ])
    categories.value = categoryResponse?.data?.data || []
    tags.value = tagResponse?.data?.data || []
    const seriesData = seriesResponse?.data?.data || {}
    series.value = Array.isArray(seriesData.records) ? seriesData.records : Array.isArray(seriesData.items) ? seriesData.items : []
  } catch {
    categories.value = []
    tags.value = []
    series.value = []
  }
}

async function loadDetail(): Promise<void> {
  if (isNew.value) {
    resetForm()
    return
  }
  const response = props.kind === 'article'
    ? await api.getStudioArticle(contentID.value)
    : props.kind === 'talk'
      ? await api.getStudioTalk(contentID.value)
      : await api.getStudioSeriesItem(contentID.value)
  const detail = response?.data?.data
  if (!detail) throw new Error('内容不存在')
  if (props.kind === 'article') applyArticle(detail)
  else if (props.kind === 'talk') applyTalk(detail)
  else applySeries(detail)
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  saveState.value = ''
  try {
    await Promise.all([loadDetail(), loadOptions()])
    markFormClean()
    const draft = readStudioDraft<Record<string, any>>(draftKey.value)
    if (draft) {
      try {
        await ElMessageBox.confirm('发现这个内容在本机自动保存的草稿，是否恢复？', '恢复本地草稿', {
          confirmButtonText: '恢复草稿', cancelButtonText: '丢弃草稿', distinguishCancelAndClose: true, type: 'info'
        })
        Object.assign(form, draft.data)
        if (props.kind === 'article' && !form.articleContentHtml) contentMode.value = 'source'
        saveState.value = `已恢复 ${new Date(draft.savedAt).toLocaleString('zh-CN')} 的本地草稿`
      } catch {
        clearStudioDraft(draftKey.value)
      }
    }
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '内容加载失败'
  } finally {
    loading.value = false
  }
}

function persistDraft(): void {
  if (loading.value || !currentUserID.value || stableForm() === cleanFormSnapshot) return
  saveStudioDraft(draftKey.value, JSON.parse(JSON.stringify(form)))
  draftStatus.value = `已自动保存到本机 · ${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
}

watch(form, () => {
  if (loading.value || suppressDraftAutosave) return
  if (draftTimer) window.clearTimeout(draftTimer)
  draftTimer = window.setTimeout(persistDraft, 800)
}, { deep: true })

watch(contentID, async () => {
  if (routeReloadSkipped) {
    routeReloadSkipped = false
    return
  }
  await load()
})

function backToList(): void {
  void router.push(`/studio/${kindPath.value}`)
}

function leaveWithDraft(): void {
  persistDraft()
  confirmLeave()
}

function validateArticle(): string {
  const title = String(form.articleTitle || '').trim()
  if (!title) return '文章标题不能为空'
  if ([...title].length > 50) return '文章标题不能超过 50 字'
  const source = contentMode.value === 'visual' ? String(form.articleContentHtml || '') : String(form.articleContent || '').trim()
  if (!source) return '文章内容不能为空'
  if ([...source].length > 100000) return '文章内容不能超过 100000 字'
  if (String(form.articleCover || '').length > 1024) return '文章封面地址过长'
  if (String(form.originalUrl || '').length > 255) return '原文链接不能超过 255 字'
  if (String(form.password || '').length > 255) return '访问密码不能超过 255 字'
  return ''
}

function articlePayload(): Record<string, unknown> {
  const html = contentMode.value === 'visual' ? String(form.articleContentHtml || '') : ''
  const source = contentMode.value === 'visual' ? html : String(form.articleContent || '')
  return {
    id: Number(form.id || 0), articleTitle: String(form.articleTitle || '').trim(), articleContent: source,
    articleContentHtml: html, articleCover: String(form.articleCover || '').trim(), categoryId: Number(form.categoryId || 0),
    tagIds: (form.tagIds || []).map(Number), seriesId: Number(form.seriesId || 0), seriesOrder: Number(form.seriesOrder || 0),
    scheduledAt: form.scheduledAt ? new Date(form.scheduledAt).toISOString() : '', visibility: form.visibility,
    type: Number(form.type || 1), password: String(form.password || '').trim(), originalUrl: String(form.originalUrl || '').trim()
  }
}

async function save(): Promise<void> {
  const validation = props.kind === 'article' ? validateArticle() : ''
  if (validation) {
    ElMessage.warning(validation)
    return
  }
  saving.value = true
  saveState.value = '正在保存…'
  try {
    let response: any
    if (props.kind === 'article') {
      const payload = articlePayload()
      response = await api.saveStudioArticle(payload, Number(form.id || 0) || undefined)
    } else if (props.kind === 'talk') {
      const payload = {
        id: Number(form.id || 0), content: String(form.content || '').trim(),
        images: JSON.stringify(form.talkImages || []), isTop: form.isTop ? 1 : 0, visibility: form.visibility
      }
      response = await api.saveStudioTalk(payload, Number(form.id || 0) || undefined)
    } else {
      const payload = {
        id: Number(form.id || 0), seriesName: String(form.seriesName || '').trim(),
        seriesDesc: String(form.seriesDesc || '').trim(), cover: String(form.cover || '').trim(), visibility: form.visibility
      }
      response = await api.saveStudioSeries(payload, Number(form.id || 0) || undefined)
    }
    if (!response?.data?.flag) throw new Error(response?.data?.message || '保存失败')
    const savedID = Number(response.data.data?.id || form.id || 0)
    const wasNew = isNew.value
    if (wasNew && savedID > 0) {
      suppressDraftAutosave = true
      if (draftTimer) {
        window.clearTimeout(draftTimer)
        draftTimer = undefined
      }
    }
    clearStudioDraft(draftKey.value)
    form.id = savedID
    markFormClean()
    draftStatus.value = '内容已保存'
    saveState.value = `已保存 · ${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
    ElMessage.success(`${kindLabel.value}已保存`)
    if (wasNew && savedID > 0) {
      routeReloadSkipped = true
      await router.replace(`/studio/${kindPath.value}/${savedID}/edit`)
      await nextTick()
      clearStudioDraft(studioDraftKey(currentUserID.value, props.kind, savedID))
      suppressDraftAutosave = false
    }
  } catch (reason: any) {
    suppressDraftAutosave = false
    saveState.value = '保存失败'
    ElMessage.error(reason?.response?.data?.message || reason?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleArticleCover(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploadingCover.value = true
  try {
    form.articleCover = await uploadStudioImage(file, 'article-cover')
    ElMessage.success('封面已上传')
  } catch (reason: any) {
    ElMessage.error(reason?.message || '封面上传失败')
  } finally {
    uploadingCover.value = false
  }
}

async function handleSeriesCover(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploadingCover.value = true
  try {
    form.cover = await uploadStudioImage(file, 'series-cover')
    ElMessage.success('封面已上传')
  } catch (reason: any) {
    ElMessage.error(reason?.message || '封面上传失败')
  } finally {
    uploadingCover.value = false
  }
}

async function handleTalkImages(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return
  const available = 9 - form.talkImages.length
  if (available <= 0) {
    ElMessage.warning('最多只能添加 9 张图片')
    return
  }
  uploadingTalkImages.value = true
  try {
    for (const file of files.slice(0, available)) {
      form.talkImages.push(await uploadStudioImage(file, 'talk-image'))
    }
    if (files.length > available) ElMessage.warning('超出数量的图片已忽略')
  } catch (reason: any) {
    ElMessage.error(reason?.message || '图片上传失败')
  } finally {
    uploadingTalkImages.value = false
  }
}

function moveTalkImage(index: number, offset: number): void {
  const target = index + offset
  if (target < 0 || target >= form.talkImages.length) return
  const [image] = form.talkImages.splice(index, 1)
  form.talkImages.splice(target, 0, image)
}

function removeTalkImage(index: number): void {
  form.talkImages.splice(index, 1)
}

onMounted(() => {
  void load()
  window.addEventListener('beforeunload', persistDraft)
})
onBeforeUnmount(() => {
  if (draftTimer) window.clearTimeout(draftTimer)
  window.removeEventListener('beforeunload', persistDraft)
  editorRef.value?.destroy()
})
</script>
<style lang="scss" scoped>
.studio-editor-page { min-width: 0; padding-bottom: 80px; }
.studio-editor-head { display: grid; grid-template-columns: auto 1fr auto; gap: 18px; align-items: center; padding: 18px 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-editor-head p { margin: 0 0 4px; color: var(--color-ob); font-size: 10px; letter-spacing: .16em; }
.studio-editor-head h1 { margin: 0; font-size: 1.5rem; }
.studio-editor-back, .studio-secondary-button, .studio-primary-button { padding: 8px 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-editor-save-state { color: var(--text-ob-dim); font-size: 11px; }
.studio-editor-state { margin: 38px 0; color: var(--text-ob-dim); text-align: center; }
.studio-editor-state.is-error { color: #df8177; }
.studio-editor-form { margin-top: 18px; }
.studio-editor-layout { display: grid; grid-template-columns: minmax(0, 1fr) 310px; gap: 18px; align-items: start; }
.studio-editor-main, .studio-editor-sidebar section { padding: 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.studio-editor-main { min-height: 620px; }
.studio-editor-sidebar { position: sticky; top: 18px; display: grid; gap: 14px; }
.studio-editor-sidebar h2 { margin: 0 0 15px; font-size: 1rem; }
.studio-field { position: relative; display: grid; gap: 7px; margin-bottom: 14px; color: var(--text-ob-dim); font-size: 11px; }
.studio-field input, .studio-field select, .studio-field textarea, .studio-cover-field input { width: 100%; padding: 10px 12px; border: 1px solid var(--border-hairline); border-radius: 10px; outline: none; background: var(--background-primary); color: inherit; font: inherit; }
.studio-field input:focus, .studio-field select:focus, .studio-field textarea:focus, .studio-cover-field input:focus { border-color: var(--color-ob); }
.studio-field textarea { resize: vertical; line-height: 1.7; }
.studio-field small { justify-self: end; color: var(--text-ob-dim); font-size: 10px; }
.studio-field--title input { font-size: 1.45rem; font-weight: 700; }
.studio-field--source textarea { min-height: 520px; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
.studio-editor-modebar { display: flex; justify-content: space-between; gap: 12px; margin: 8px 0 12px; }
.studio-editor-modebar > div { display: flex; gap: 6px; padding: 4px; border: 1px solid var(--border-hairline); border-radius: 999px; }
.studio-editor-modebar button { padding: 7px 12px; border: 0; border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.studio-editor-modebar button.active { background: var(--color-ob); color: #081127; }
.studio-rich-editor { overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 12px; background: var(--background-primary); }
.studio-rich-editor :deep(.w-e-toolbar) { border-bottom: 1px solid var(--border-hairline); background: var(--background-primary-alt); }
.studio-rich-editor :deep(.w-e-text-container) { min-height: 520px; background: var(--background-primary); color: var(--text-primary); }
.studio-article-preview { min-height: 520px; padding: 24px; border: 1px solid var(--border-hairline); border-radius: 12px; background: var(--background-primary); line-height: 1.85; }
.studio-article-preview :deep(img) { max-width: 100%; height: auto; border-radius: 10px; }
.studio-cover-field { display: grid; gap: 9px; }
.studio-cover-field img { width: 100%; aspect-ratio: 16 / 9; border-radius: 10px; object-fit: cover; }
.studio-upload-button { padding: 9px 13px; border: 1px solid var(--border-hairline); border-radius: 10px; background: transparent; color: inherit; cursor: pointer; }
.studio-text-button { border: 0; background: transparent; color: #df8177; cursor: pointer; }
.studio-checkbox { display: flex; align-items: center; gap: 8px; color: var(--text-ob-dim); font-size: 12px; }
.studio-talk-images header { display: flex; justify-content: space-between; margin-bottom: 10px; }
.studio-talk-images__grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; margin-bottom: 12px; }
.studio-talk-images__grid article { overflow: hidden; border: 1px solid var(--border-hairline); border-radius: 10px; }
.studio-talk-images__grid img { width: 100%; aspect-ratio: 1; object-fit: cover; }
.studio-talk-images__grid article > div { display: flex; justify-content: center; gap: 3px; padding: 5px; }
.studio-talk-images__grid button { border: 0; background: transparent; color: inherit; cursor: pointer; }
.studio-talk-images__grid button.danger { color: #df8177; }
.studio-editor-footer { position: sticky; bottom: 12px; display: flex; justify-content: space-between; align-items: center; gap: 14px; margin-top: 18px; padding: 13px 16px; border: 1px solid var(--border-hairline); border-radius: 16px; background: color-mix(in srgb, var(--background-primary-alt) 96%, transparent); backdrop-filter: blur(16px); }
.studio-editor-footer span { color: var(--text-ob-dim); font-size: 11px; }
.studio-editor-footer > div { display: flex; align-items: center; gap: 8px; }
.studio-public-link { color: var(--color-ob); font-size: 12px; text-decoration: none; }
.studio-primary-button { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
.studio-primary-button:disabled, .studio-upload-button:disabled { opacity: .55; cursor: wait; }
.studio-leave-mask { position: fixed; z-index: 3000; inset: 0; display: grid; place-items: center; padding: 20px; background: rgba(2, 6, 20, .72); backdrop-filter: blur(8px); }
.studio-leave-mask section { width: min(440px, 100%); padding: 26px; border: 1px solid var(--border-hairline); border-radius: 18px; background: var(--background-primary-alt); }
.studio-leave-mask h2 { margin: 0 0 10px; }
.studio-leave-mask p { color: var(--text-ob-dim); line-height: 1.7; }
.studio-leave-mask section > div { display: flex; justify-content: flex-end; gap: 8px; }
.studio-leave-mask button { padding: 9px 14px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-leave-mask button.primary { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
@media (max-width: 1000px) { .studio-editor-layout { grid-template-columns: 1fr; } .studio-editor-sidebar { position: static; grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 680px) { .studio-editor-head { grid-template-columns: 1fr; } .studio-editor-save-state { justify-self: start; } .studio-editor-modebar { align-items: flex-start; flex-direction: column; } .studio-editor-sidebar { grid-template-columns: 1fr; } .studio-talk-images__grid { grid-template-columns: repeat(2, 1fr); } .studio-editor-footer { align-items: stretch; flex-direction: column; } .studio-editor-footer > div { display: grid; grid-template-columns: 1fr 1fr; } }
</style>