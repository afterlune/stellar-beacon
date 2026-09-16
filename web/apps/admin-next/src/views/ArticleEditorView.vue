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
          <a-textarea
            v-model="form.articleContent"
            class="article-content-editor"
            :max-length="100000"
            show-word-limit
            :auto-size="{ minRows: 16, maxRows: 32 }"
            :placeholder="t('articles.editor.contentPlaceholder')" />
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
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
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

const form = reactive({
  id: 0,
  articleTitle: '',
  articleContent: '',
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

async function load(): Promise<void> {
  try {
    const article = await getAdminArticle(articleId.value)
    form.id = Number(article.id || article.articleId || 0)
    form.articleTitle = String(article.articleTitle || '')
    form.articleContent = String(article.articleContent || '')
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
</script>

<style scoped>
.article-editor-loading {
  display: flex;
  justify-content: center;
  padding: 72px 0;
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
