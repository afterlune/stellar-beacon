<template>
  <section class="admin-page">
    <AdminPageHeader
      :title="isEditing ? '修改文章' : '发布文章'"
      :description="isEditing ? '调整内容、分类与发布状态，保存后立即生效。' : '写下新的内容，保存后会出现在文章列表中。'">
      <template #actions>
        <a-button type="primary" :loading="saving" @click="submit">保存</a-button>
        <a-button @click="router.push('/article-list')">返回列表</a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!editorReady" class="article-editor-loading" tip="正在加载文章…" />
      <a-form v-else ref="formRef" class="article-form" :model="form" layout="vertical">
        <a-form-item field="articleTitle" label="标题" :rules="[{ required: true, message: '标题不能为空' }]">
          <a-input v-model="form.articleTitle" :max-length="256" show-word-limit placeholder="一句话说清这篇文章讲什么" />
        </a-form-item>

        <div class="article-form-grid">
          <a-form-item field="categoryName" label="分类">
            <a-select
              v-model="form.categoryName"
              allow-search
              allow-clear
              :loading="taxonomyLoading"
              placeholder="从分类表选择"
              @search="searchCategories">
              <a-option v-for="category in categories" :key="category" :value="category">{{ category }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="tagNames" label="标签">
            <a-select
              v-model="form.tagNames"
              multiple
              allow-search
              allow-clear
              :loading="taxonomyLoading"
              placeholder="从标签表选择"
              @search="searchTags">
              <a-option v-for="tag in tags" :key="tag" :value="tag">{{ tag }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="status" label="状态">
            <a-select v-model="form.status">
              <a-option :value="1">公开</a-option>
              <a-option :value="2">私密</a-option>
              <a-option :value="3">草稿</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="type" label="类型">
            <a-select v-model="form.type">
              <a-option :value="1">原创</a-option>
              <a-option :value="2">转载</a-option>
              <a-option :value="3">翻译</a-option>
            </a-select>
          </a-form-item>
        </div>

        <!-- password / originalUrl render only when relevant so the article form
             keeps a stable, predictable field set for keyboard and E2E flows. -->
        <a-form-item v-if="form.status === 2" field="password" label="访问密码">
          <a-input-password v-model="form.password" maxlength="255" placeholder="留空表示不加访问限制" />
          <template #help>状态为「私密」时生效，读者需要输入该密码才能查看正文。</template>
        </a-form-item>
        <a-form-item
          v-if="form.type === 2 || form.type === 3"
          field="originalUrl"
          label="原文链接"
          :rules="[{ required: true, message: '转载或翻译的文章需要填写原文链接' }]">
          <a-input v-model="form.originalUrl" maxlength="255" placeholder="https://原文地址" />
        </a-form-item>

        <a-form-item field="articleCover" label="文章封面">
          <div class="article-cover-control">
            <AdminImagePreview v-if="form.articleCover" :src="form.articleCover" alt="文章封面" :width="160" :height="104" />
            <div class="article-cover-actions">
              <a-input v-model="form.articleCover" placeholder="可选，也可以粘贴 HTTPS 图片地址" />
              <a-space wrap>
                <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
                <a-button :loading="coverUploading" @click="coverInput?.click()">上传封面</a-button>
                <a-button @click="mediaPickerVisible = true">从资源库选择</a-button>
                <a-button :disabled="!form.articleCover" @click="form.articleCover = ''">清除封面</a-button>
              </a-space>
            </div>
          </div>
        </a-form-item>

        <a-form-item field="articleContent" label="正文" :rules="[{ required: true, message: '正文不能为空' }]">
          <a-textarea
            v-model="form.articleContent"
            class="article-content-editor"
            :max-length="100000"
            show-word-limit
            :auto-size="{ minRows: 16, maxRows: 32 }"
            placeholder="支持 Markdown / HTML，保存后由前台渲染" />
        </a-form-item>

        <div class="article-meta-row">
          <span class="article-meta-label">展示选项</span>
          <AdminFlagCheckbox v-model="form.isTop">置顶</AdminFlagCheckbox>
          <AdminFlagCheckbox v-model="form.isFeatured">精选</AdminFlagCheckbox>
          <span class="article-meta-summary">{{ summary }}</span>
        </div>

        <div class="admin-form-actions">
          <a-button type="primary" :loading="saving" @click="submit">保存</a-button>
          <a-button :disabled="saving" @click="router.push('/article-list')">取消</a-button>
        </div>
      </a-form>
    </a-card>

    <AdminMediaPicker v-model="mediaPickerVisible" @select="selectMedia" />

    <AdminLeaveGuard :visible="leaveVisible" @ok="confirmLeave" @cancel="cancelLeave">
      当前页面还有未保存的修改，离开后这些内容会丢失。
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
  const parts = [`约 ${wordCount.value} 字`]
  if (form.categoryName) parts.push(form.categoryName)
  if (form.tagNames.length) parts.push(`${form.tagNames.length} 个标签`)
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
    errorMessage.value = apiErrorMessage(error, '文章加载失败')
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
    Message.error('请先修正表单中标红的字段')
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
    Message.success(isEditing.value ? '文章已更新' : '文章已发布')
    markClean()
    await router.push('/article-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '文章保存失败')
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
    Message.warning(apiErrorMessage(error, '分类和标签加载失败，可稍后重试'))
  } finally {
    taxonomyLoading.value = false
  }
}

async function searchCategories(value: string): Promise<void> {
  try {
    categories.value = mergeOptions((await listAdminCategories(value)).map(taxonomyName), categories.value)
  } catch (error) {
    // Keep the previous options but tell the author why the list may be stale.
    Message.warning(apiErrorMessage(error, '分类搜索失败，仍显示上一次的结果'))
  }
}

async function searchTags(value: string): Promise<void> {
  try {
    tags.value = mergeOptions((await listAdminTags(value)).map(taxonomyName), tags.value)
  } catch (error) {
    Message.warning(apiErrorMessage(error, '标签搜索失败，仍显示上一次的结果'))
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
    Message.success('封面上传成功')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '封面上传失败')
    Message.error(errorMessage.value)
  } finally {
    coverUploading.value = false
  }
}

function selectMedia(asset: { url: string }): void {
  form.articleCover = asset.url
  Message.success('已选择封面')
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
