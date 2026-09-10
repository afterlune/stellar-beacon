<template>
  <section class="admin-page">
    <a-card class="admin-form-panel admin-form-card" :bordered="false" :title="isEditing ? '修改文章' : '发布文章'">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-form class="article-form" :model="form" layout="vertical" @submit-success="save">
        <a-form-item field="articleTitle" label="标题" :rules="[{ required: true, message: '标题不能为空' }]">
          <a-input v-model="form.articleTitle" :max-length="256" show-word-limit />
        </a-form-item>
        <div class="article-form-grid">
          <a-form-item field="categoryName" label="分类">
            <a-select v-model="form.categoryName" allow-search :loading="taxonomyLoading" placeholder="从分类表选择" @search="searchCategories">
              <a-option v-for="category in categories" :key="category" :value="category">{{ category }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="tagNames" label="标签">
            <a-select v-model="form.tagNames" multiple allow-search :loading="taxonomyLoading" placeholder="从标签表选择" @search="searchTags">
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
        <a-form-item field="articleCover" label="文章封面">
          <div class="article-cover-control">
            <AdminImagePreview v-if="form.articleCover" :src="form.articleCover" alt="文章封面" :width="160" :height="104" />
            <div class="article-cover-actions">
              <a-input v-model="form.articleCover" placeholder="可选，也可以从图片资源选择" />
              <a-space>
                <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
                <a-button @click="coverInput?.click()">上传封面</a-button>
                <a-button @click="mediaPickerVisible = true">选择资源</a-button>
              </a-space>
            </div>
          </div>
        </a-form-item>
        <a-form-item field="articleContent" label="正文" :rules="[{ required: true, message: '正文不能为空' }]">
          <a-textarea v-model="form.articleContent" class="article-content-editor" :max-length="100000" show-word-limit :auto-size="{ minRows: 16, maxRows: 32 }" />
        </a-form-item>
        <a-space wrap>
          <a-checkbox v-model="form.isTop" :checked-value="1" :unchecked-value="0">置顶</a-checkbox>
          <a-checkbox v-model="form.isFeatured" :checked-value="1" :unchecked-value="0">精选</a-checkbox>
        </a-space>
        <a-space class="admin-form-actions">
          <a-button type="primary" html-type="submit" :loading="saving">保存</a-button>
          <a-button @click="router.push('/article-list')">返回列表</a-button>
        </a-space>
      </a-form>
    </a-card>
    <AdminMediaPicker v-model="mediaPickerVisible" @select="selectMedia" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRoute, useRouter } from 'vue-router'

import { apiErrorMessage, getAdminArticle, listAdminCategories, listAdminTags, saveAdminArticle, uploadAdminArticleImage } from '@/api/http'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminMediaPicker from '@/components/AdminMediaPicker.vue'

const route = useRoute()
const router = useRouter()
const saving = ref(false)
const errorMessage = ref('')
const form = reactive({ id: 0, articleTitle: '', articleContent: '', articleCover: '', categoryName: '', tagNames: [] as string[], status: 1, type: 1, isTop: 0, isFeatured: 0, password: '', originalUrl: '' })
const categories = ref<string[]>([])
const tags = ref<string[]>([])
const taxonomyLoading = ref(false)
const mediaPickerVisible = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const articleId = computed(() => typeof route.params.articleId === 'string' ? route.params.articleId : '')
const isEditing = computed(() => /^\d+$/.test(articleId.value))

onMounted(() => {
  void loadTaxonomy()
  if (isEditing.value) void load()
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
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '文章加载失败')
  }
}

async function save(): Promise<void> {
  saving.value = true
  errorMessage.value = ''
  try {
    await saveAdminArticle({ ...form, tagNames: form.tagNames })
    Message.success('文章已保存')
    await router.push('/article-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '文章保存失败')
  } finally {
    saving.value = false
  }
}

async function loadTaxonomy(): Promise<void> {
  taxonomyLoading.value = true
  try {
    const [categoryItems, tagItems] = await Promise.all([listAdminCategories(), listAdminTags()])
    categories.value = categoryItems.map(taxonomyName).filter(Boolean)
    tags.value = tagItems.map(taxonomyName).filter(Boolean)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '分类和标签加载失败')
  } finally {
    taxonomyLoading.value = false
  }
}

async function searchCategories(value: string): Promise<void> {
  try { categories.value = (await listAdminCategories(value)).map(taxonomyName).filter(Boolean) }
  catch { /* keep the last successful options visible */ }
}

async function searchTags(value: string): Promise<void> {
  try { tags.value = (await listAdminTags(value)).map(taxonomyName).filter(Boolean) }
  catch { /* keep the last successful options visible */ }
}

async function selectCover(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  try { form.articleCover = await uploadAdminArticleImage(file); Message.success('封面上传成功') }
  catch (error) { errorMessage.value = apiErrorMessage(error, '封面上传失败') }
}

function selectMedia(asset: { url: string }): void {
  form.articleCover = asset.url
}

function taxonomyName(value: Record<string, unknown>): string {
  return String(value.categoryName || value.tagName || value.name || '').trim()
}

function normalizeTags(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String).map(item => item.trim()).filter(Boolean)
  if (value && typeof value === 'object' && 'root' in value) return []
  if (typeof value === 'string') {
    try {
      const parsed: unknown = JSON.parse(value)
      if (Array.isArray(parsed)) return parsed.map(String).map(item => item.trim()).filter(Boolean)
    } catch { /* legacy comma-separated values are handled below */ }
    return value.split(',').map(item => item.trim()).filter(Boolean)
  }
  return []
}
</script>

<style scoped>
.article-form { margin-top: 22px; }
.article-form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 20px; }
.article-cover-control { display: flex; align-items: flex-start; gap: 16px; }.article-cover-actions { display: grid; flex: 1; gap: 10px; min-width: 0; }
@media (max-width: 800px) { .article-form-grid { grid-template-columns: 1fr; } }
@media (max-width: 560px) { .article-cover-control { align-items: stretch; flex-direction: column; } }
</style>
