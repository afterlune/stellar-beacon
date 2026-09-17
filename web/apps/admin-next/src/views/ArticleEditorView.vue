<template>
  <section class="admin-page">
    <AdminPageHeader
      :title="isEditing ? t('articles.editor.editTitle') : t('articles.actions.publish')"
      :description="isEditing ? t('articles.editor.editDescription') : t('articles.editor.createDescription')">
      <template #actions>
        <a-button type="primary" :loading="saving" @click="submit">{{ t('common.save') }}</a-button>
        <a-button @click="router.push('/article-list')">{{ t('articles.editor.backToList') }}</a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!editorReady" class="article-editor-loading" :tip="t('articles.editor.loading')" />
      <a-form v-else ref="formRef" class="article-form" :model="form" layout="vertical">
        <a-form-item
          field="articleTitle"
          :label="t('common.title')"
          :rules="[{ required: true, message: t('articles.editor.titleRequired') }]">
          <a-input v-model="form.articleTitle" :max-length="256" show-word-limit :placeholder="t('articles.editor.titlePlaceholder')" />
        </a-form-item>

        <div class="article-form-grid">
          <a-form-item field="categoryName" :label="t('articles.category')">
            <a-select
              v-model="form.categoryName"
              allow-search
              allow-clear
              :loading="taxonomyLoading"
              :placeholder="t('articles.editor.categoryPlaceholder')"
              @search="searchCategories">
              <a-option v-for="category in categories" :key="category" :value="category">{{ category }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="tagNames" :label="t('articles.tags')">
            <a-select
              v-model="form.tagNames"
              multiple
              allow-search
              allow-clear
              :loading="taxonomyLoading"
              :placeholder="t('articles.editor.tagPlaceholder')"
              @search="searchTags">
              <a-option v-for="tag in tags" :key="tag" :value="tag">{{ tag }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="status" :label="t('common.status')">
            <a-select v-model="form.status">
              <a-option :value="1">{{ t('status.published') }}</a-option>
              <a-option :value="2">{{ t('status.private') }}</a-option>
              <a-option :value="3">{{ t('status.draft') }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="type" :label="t('common.type')">
            <a-select v-model="form.type">
              <a-option :value="1">{{ t('status.original') }}</a-option>
              <a-option :value="2">{{ t('status.reprint') }}</a-option>
              <a-option :value="3">{{ t('status.translate') }}</a-option>
            </a-select>
          </a-form-item>
        </div>

        <!-- password / originalUrl render only when relevant so the article form
             keeps a stable, predictable field set for keyboard and E2E flows. -->
        <a-form-item v-if="form.status === 2" field="password" :label="t('articles.editor.password')">
          <a-input-password v-model="form.password" maxlength="255" :placeholder="t('articles.editor.passwordPlaceholder')" />
          <template #help>{{ t('articles.editor.passwordHelp') }}</template>
        </a-form-item>
        <a-form-item
          v-if="form.type === 2 || form.type === 3"
          field="originalUrl"
          :label="t('articles.editor.originalUrl')"
          :rules="[{ required: true, message: t('articles.editor.originalUrlRequired') }]">
          <a-input v-model="form.originalUrl" maxlength="255" :placeholder="t('articles.editor.originalUrlPlaceholder')" />
        </a-form-item>

        <a-form-item field="articleCover" :label="t('articles.editor.cover')">
          <div class="article-cover-control">
            <AdminImagePreview
              v-if="form.articleCover"
              :src="form.articleCover"
              :alt="t('articles.editor.cover')"
              :width="160"
              :height="104" />
            <div class="article-cover-actions">
              <a-input v-model="form.articleCover" :placeholder="t('articles.editor.coverPlaceholder')" />
              <a-space wrap>
                <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
                <a-button :loading="coverUploading" @click="coverInput?.click()">{{ t('articles.editor.uploadCover') }}</a-button>
                <a-button @click="mediaPickerVisible = true">{{ t('articles.editor.pickFromLibrary') }}</a-button>
                <a-button :disabled="!form.articleCover" @click="form.articleCover = ''">{{ t('articles.editor.clearCover') }}</a-button>
              </a-space>
            </div>
          </div>
        </a-form-item>

        <a-form-item
          field="articleContent"
          :label="t('articles.editor.content')"
          :rules="[{ required: true, message: t('articles.editor.contentRequired') }]">
          <div class="article-editor-head">
            <a-radio-group :model-value="contentMode" type="button" size="small" @change="switchContentMode">
              <a-radio value="visual">{{ t('articles.editor.visualMode') }}</a-radio>
              <a-radio value="source">{{ t('articles.editor.sourceMode') }}</a-radio>
            </a-radio-group>
            <a-button
              v-if="contentMode === 'source' && form.articleContent.trim()"
              type="text"
              size="small"
              @click="convertSourceToVisual">
              {{ t('articles.editor.convertToRich') }}
            </a-button>
            <span class="admin-field-hint">{{ t('articles.editor.contentHint') }}</span>
          </div>

          <div v-if="contentMode === 'visual'" class="rich-editor-shell">
            <Toolbar :editor="editorInstance" :default-config="toolbarConfig" mode="default" />
            <Editor
              v-model="form.articleContentHtml"
              class="rich-editor"
              mode="default"
              :default-config="editorConfig"
              @on-created="handleEditorCreated"
              @on-change="handleEditorChange" />
          </div>

          <div v-else class="article-source-shell">
            <a-textarea
              v-model="form.articleContent"
              class="article-content-editor"
              :max-length="100000"
              show-word-limit
              :auto-size="{ minRows: 16, maxRows: 32 }"
              :placeholder="t('articles.editor.sourcePlaceholder')" />
            <div class="article-source-preview">
              <div class="article-source-preview-head">{{ t('articles.editor.preview') }}</div>
              <div class="post-html" v-html="sourcePreviewHtml" />
            </div>
          </div>
        </a-form-item>

        <div class="article-meta-row">
          <span class="article-meta-label">{{ t('articles.editor.displayOptions') }}</span>
          <AdminFlagCheckbox v-model="form.isTop">{{ t('status.pinned') }}</AdminFlagCheckbox>
          <AdminFlagCheckbox v-model="form.isFeatured">{{ t('articles.featured') }}</AdminFlagCheckbox>
          <span class="article-meta-summary">{{ summary }}</span>
        </div>

        <div class="admin-form-actions">
          <a-button type="primary" :loading="saving" @click="submit">{{ t('common.save') }}</a-button>
          <a-button :disabled="saving" @click="router.push('/article-list')">{{ t('common.cancel') }}</a-button>
        </div>
      </a-form>
    </a-card>

    <AdminMediaPicker v-model="mediaPickerVisible" @select="selectMedia" />

    <AdminLeaveGuard
      :visible="leaveVisible"
      :title="t('common.unsavedTitle')"
      :ok-text="t('common.unsavedLeave')"
      :cancel-text="t('common.unsavedStay')"
      @ok="confirmLeave"
      @cancel="cancelLeave">
      {{ t('articles.editor.leaveGuard') }}
    </AdminLeaveGuard>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef } from 'vue'
import { Message } from '@arco-design/web-vue'
import { Editor, Toolbar } from '@wangeditor-next/editor-for-vue'
import type { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor-next/editor'
import '@wangeditor-next/editor/dist/css/style.css'
import { useRoute, useRouter } from 'vue-router'

import {
  apiErrorMessage,
  getAdminArticle,
  listAdminCategories,
  listAdminTags,
  saveAdminArticle,
  uploadAdminArticleImage
} from '@/api/http'
import AdminFlagCheckbox from '@/components/AdminFlagCheckbox.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminLeaveGuard from '@/components/AdminLeaveGuard.vue'
import AdminMediaPicker from '@/components/AdminMediaPicker.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useUnsavedGuard } from '@/composables/useUnsavedGuard'
import { t } from '@/i18n'
import { plainText } from '@/utils/format'
import { markdownToHtml, sanitizePreviewHtml } from '@/utils/markdown'

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024
const route = useRoute()
const router = useRouter()

const saving = ref(false)
const coverUploading = ref(false)
const editorReady = ref(false)
const errorMessage = ref('')
const taxonomyLoading = ref(false)
const mediaPickerVisible = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const formRef = ref<{ validate: () => Promise<Record<string, unknown> | undefined> } | null>(null)
const editorInstance = shallowRef<IDomEditor>()
const contentMode = ref<'visual' | 'source'>('visual')

const form = reactive({
  id: 0,
  articleTitle: '',
  articleContent: '',
  articleContentHtml: '',
  articleCover: '',
  categoryName: '',
  tagNames: [] as string[],
  status: 1,
  type: 1,
  isTop: 0,
  isFeatured: 0,
  password: '',
  originalUrl: ''
})

const categories = ref<string[]>([])
const tags = ref<string[]>([])
const articleId = computed(() => (typeof route.params.articleId === 'string' ? route.params.articleId : ''))
const isEditing = computed(() => /^\d+$/.test(articleId.value))

/**
 * 只比较真正会被保存的字段：加载与保存都会重置基线，因此「打开就离开」不会被拦截。
 */
const { visible: leaveVisible, markClean, confirmLeave, cancelLeave } = useUnsavedGuard(() => JSON.stringify([
  form.articleTitle,
  form.articleContent,
  form.articleContentHtml,
  contentMode.value,
  form.articleCover,
  form.categoryName,
  form.tagNames,
  form.status,
  form.type,
  form.isTop,
  form.isFeatured,
  form.password,
  form.originalUrl
]))

const wordCount = computed(() => plainText(form.articleContent).replace(/\s/g, '').length)
const sourcePreviewHtml = computed(() => sanitizePreviewHtml(markdownToHtml(form.articleContent)))
const summary = computed(() => {
  const parts = [t('articles.editor.wordCount', { count: wordCount.value })]
  if (form.categoryName) parts.push(form.categoryName)
  if (form.tagNames.length) parts.push(t('articles.editor.tagCount', { count: form.tagNames.length }))
  return parts.join(' · ')
})

onMounted(() => {
  void loadTaxonomy()
  if (isEditing.value) void load()
  else {
    editorReady.value = true
    markClean()
  }
})

const toolbarConfig: Partial<IToolbarConfig> = {
  excludeKeys: ['group-video']
}

const editorConfig: Partial<IEditorConfig> = {
  placeholder: t('articles.editor.richPlaceholder'),
  maxLength: 100000,
  sanitizeHtml: (html) => sanitizePreviewHtml(html),
  MENU_CONF: {
    uploadImage: {
      maxFileSize: MAX_UPLOAD_BYTES,
      allowedFileTypes: ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'],
      customUpload(file, insertFn) {
        void uploadEditorImage(file, insertFn)
      }
    }
  }
}

function handleEditorCreated(editor: IDomEditor): void {
  editorInstance.value = editor
}

/**
 * wangEditor 把空文档规范成 `<p><br></p>`；原样写回表单会让「刚打开就离开」
 * 也被未保存守卫拦截，所以空文档仍然按空内容保存。
 */
function normalizeEditorHtml(html: string): string {
  return /^(?:<p><br\s*\/?><\/p>|\s)*$/i.test(html) ? '' : html
}

function handleEditorChange(editor: IDomEditor): void {
  const html = normalizeEditorHtml(editor.getHtml())
  form.articleContentHtml = html
  form.articleContent = html
}

async function uploadEditorImage(file: File, insertFn: (url: string, poster?: string, alt?: string) => void): Promise<void> {
  if (!file.type.startsWith('image/')) {
    Message.error(t('articles.editor.imageTypeInvalid'))
    return
  }
  if (file.size > MAX_UPLOAD_BYTES) {
    Message.error(t('articles.editor.imageTooLarge'))
    return
  }
  try {
    const url = await uploadAdminArticleImage(file)
    insertFn(url, '', file.name)
  } catch (error) {
    Message.error(apiErrorMessage(error, t('articles.editor.imageUploadFailed')))
  }
}

function switchContentMode(value: string | number | boolean): void {
  const next = String(value) as 'visual' | 'source'
  if (next === contentMode.value) return
  if (next === 'source') {
    form.articleContent = form.articleContentHtml || form.articleContent
    form.articleContentHtml = ''
  } else {
    form.articleContentHtml = sanitizePreviewHtml(markdownToHtml(form.articleContent))
    form.articleContent = form.articleContentHtml
  }
  contentMode.value = next
}

function convertSourceToVisual(): void {
  form.articleContentHtml = sanitizePreviewHtml(markdownToHtml(form.articleContent))
  form.articleContent = form.articleContentHtml
  contentMode.value = 'visual'
  Message.success(t('articles.editor.convertedToRich'))
}

async function load(): Promise<void> {
  try {
    const article = await getAdminArticle(articleId.value)
    form.id = Number(article.id || article.articleId || 0)
    form.articleTitle = String(article.articleTitle || '')
    form.articleContent = String(article.articleContent || '')
    form.articleContentHtml = String(article.articleContentHtml || '')
    if (form.articleContentHtml) {
      form.articleContent = form.articleContentHtml
      contentMode.value = 'visual'
    } else {
      contentMode.value = 'source'
    }
    form.articleCover = String(article.articleCover || '')
    form.categoryName = String(article.categoryName || '')
    form.tagNames = normalizeTags(article.tagNames)
    form.status = Number(article.status || 1)
    form.type = Number(article.type || 1)
    form.isTop = Number(article.isTop || 0)
    form.isFeatured = Number(article.isFeatured || 0)
    form.password = String(article.password || '')
    form.originalUrl = String(article.originalUrl || '')
    // Ensure the loaded taxonomy value is always selectable in its dropdown.
    if (form.categoryName && !categories.value.includes(form.categoryName)) {
      categories.value = [form.categoryName, ...categories.value]
    }
    const missingTags = form.tagNames.filter((tag) => !tags.value.includes(tag))
    if (missingTags.length) tags.value = [...missingTags, ...tags.value]
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('articles.editor.loadFailed'))
    Message.error(errorMessage.value)
  } finally {
    editorReady.value = true
    markClean()
  }
}

/** The header save button drives the same validated submit as the form button. */
async function submit(): Promise<void> {
  if (saving.value) return
  const errors = await formRef.value?.validate()
  if (errors) {
    Message.error(t('articles.editor.fixErrors'))
    return
  }
  await save()
}

async function save(): Promise<void> {
  saving.value = true
  errorMessage.value = ''
  try {
    await saveAdminArticle({
      // `id: 0` must never be sent: the backend treats its presence as an update.
      id: form.id || undefined,
      articleTitle: form.articleTitle.trim(),
      articleContent: form.articleContent,
      ...(contentMode.value === 'visual' ? { articleContentHtml: form.articleContentHtml } : {}),
      articleCover: form.articleCover.trim(),
      categoryName: form.categoryName,
      tagNames: form.tagNames,
      status: form.status,
      type: form.type,
      isTop: form.isTop,
      isFeatured: form.isFeatured,
      password: form.status === 2 ? form.password : '',
      originalUrl: form.type === 1 ? '' : form.originalUrl.trim()
    })
    Message.success(isEditing.value ? t('articles.editor.saved') : t('articles.editor.published'))
    markClean()
    await router.push('/article-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('articles.editor.saveFailed'))
    Message.error(errorMessage.value)
  } finally {
    saving.value = false
  }
}

async function loadTaxonomy(): Promise<void> {
  taxonomyLoading.value = true
  try {
    const [categoryItems, tagItems] = await Promise.all([listAdminCategories(), listAdminTags()])
    categories.value = mergeOptions(categoryItems.map(taxonomyName), categories.value)
    tags.value = mergeOptions(tagItems.map(taxonomyName), tags.value)
  } catch (error) {
    Message.warning(apiErrorMessage(error, t('articles.editor.taxonomyLoadFailed')))
  } finally {
    taxonomyLoading.value = false
  }
}

async function searchCategories(value: string): Promise<void> {
  try {
    categories.value = mergeOptions((await listAdminCategories(value)).map(taxonomyName), categories.value)
  } catch (error) {
    // Keep the previous options but tell the author why the list may be stale.
    Message.warning(apiErrorMessage(error, t('articles.editor.categorySearchFailed')))
  }
}

async function searchTags(value: string): Promise<void> {
  try {
    tags.value = mergeOptions((await listAdminTags(value)).map(taxonomyName), tags.value)
  } catch (error) {
    Message.warning(apiErrorMessage(error, t('articles.editor.tagSearchFailed')))
  }
}

async function selectCover(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    Message.warning(t('articles.editor.imageTypeInvalid'))
    return
  }
  if (file.size > MAX_UPLOAD_BYTES) {
    Message.warning(t('articles.editor.imageTooLarge'))
    return
  }
  coverUploading.value = true
  try {
    form.articleCover = await uploadAdminArticleImage(file)
    Message.success(t('articles.editor.coverUploaded'))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('articles.editor.coverUploadFailed'))
    Message.error(errorMessage.value)
  } finally {
    coverUploading.value = false
  }
}

function selectMedia(asset: { url: string }): void {
  form.articleCover = asset.url
  Message.success(t('articles.editor.coverSelected'))
}

/** Union of the freshly searched options and any value already present. */
function mergeOptions(incoming: string[], existing: string[]): string[] {
  const values = new Set(incoming.filter(Boolean))
  for (const value of existing) if (value) values.add(value)
  return [...values]
}

function taxonomyName(value: Record<string, unknown>): string {
  return String(value.categoryName || value.tagName || value.name || '').trim()
}

function normalizeTags(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String).map((item) => item.trim()).filter(Boolean)
  if (value && typeof value === 'object' && 'root' in value) return []
  if (typeof value === 'string') {
    try {
      const parsed: unknown = JSON.parse(value)
      if (Array.isArray(parsed)) return parsed.map(String).map((item) => item.trim()).filter(Boolean)
    } catch {
      // Legacy comma-separated values are handled below.
    }
    return value.split(',').map((item) => item.trim()).filter(Boolean)
  }
  return []
}

onBeforeUnmount(() => {
  editorInstance.value?.destroy()
  editorInstance.value = undefined
})
</script>

<style scoped>
.article-editor-loading {
  display: flex;
  justify-content: center;
  padding: 72px 0;
}

.article-editor-head {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.rich-editor-shell,
.article-source-shell {
  overflow: hidden;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface);
}

.rich-editor-shell :deep(.w-e-toolbar) {
  border-color: var(--admin-border);
  background: var(--admin-surface-soft);
}

.rich-editor-shell :deep(.w-e-text-container) {
  min-height: 430px;
  border-color: var(--admin-border);
  background: var(--admin-surface);
}

.rich-editor-shell :deep(.w-e-text-container [data-slate-editor]) {
  min-height: 390px;
  padding: 18px 20px;
}

.article-source-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.article-source-shell .article-content-editor {
  min-height: 470px;
  border: 0;
  border-right: 1px solid var(--admin-border);
  border-radius: 0;
}

.article-source-preview {
  min-height: 470px;
  padding: 14px 18px;
  overflow: auto;
  background: var(--admin-surface-soft);
}

.article-source-preview-head {
  margin-bottom: 14px;
  color: var(--admin-subtle);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}

@media (max-width: 900px) {
  .article-source-shell {
    grid-template-columns: 1fr;
  }

  .article-source-shell .article-content-editor {
    border-right: 0;
    border-bottom: 1px solid var(--admin-border);
  }
}

.article-meta-row {
  display: flex;
  align-items: center;
  gap: var(--admin-gap);
  flex-wrap: wrap;
}

.article-meta-summary {
  margin-left: auto;
  color: var(--admin-subtle);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
</style>
