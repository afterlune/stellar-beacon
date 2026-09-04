<template>
  <section>
    <a-card :title="isEditing ? '修改文章' : '发布文章'">
      <a-alert type="info" :show-icon="true" :closable="false">保存操作沿用后端文章接口；AI 生成物必须先经过审核，不会在此处绕过审核发布。</a-alert>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-form class="article-form" :model="form" layout="vertical" @submit-success="save">
        <a-form-item field="articleTitle" label="标题" :rules="[{ required: true, message: '标题不能为空' }]">
          <a-input v-model="form.articleTitle" :max-length="256" show-word-limit />
        </a-form-item>
        <div class="article-form-grid">
          <a-form-item field="categoryName" label="分类">
            <a-input v-model="form.categoryName" placeholder="分类名称" />
          </a-form-item>
          <a-form-item field="tagNames" label="标签">
            <a-input v-model="form.tagNames" placeholder="多个标签用逗号分隔" />
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
            </a-select>
          </a-form-item>
        </div>
        <a-form-item field="articleCover" label="封面 URL">
          <a-input v-model="form.articleCover" placeholder="可选" />
        </a-form-item>
        <a-form-item field="articleContent" label="正文" :rules="[{ required: true, message: '正文不能为空' }]">
          <a-textarea v-model="form.articleContent" :max-length="100000" show-word-limit :auto-size="{ minRows: 16, maxRows: 32 }" />
        </a-form-item>
        <a-space wrap>
          <a-checkbox v-model="form.isTop" :checked-value="1" :unchecked-value="0">置顶</a-checkbox>
          <a-checkbox v-model="form.isFeatured" :checked-value="1" :unchecked-value="0">精选</a-checkbox>
        </a-space>
        <a-space>
          <a-button type="primary" html-type="submit" :loading="saving">保存</a-button>
          <a-button @click="router.push('/article-list')">返回列表</a-button>
        </a-space>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRoute, useRouter } from 'vue-router'

import { apiErrorMessage, getAdminArticle, saveAdminArticle } from '@/api/http'

const route = useRoute()
const router = useRouter()
const saving = ref(false)
const errorMessage = ref('')
const form = reactive({ id: 0, articleTitle: '', articleContent: '', articleCover: '', categoryName: '', tagNames: '', status: 1, type: 1, isTop: 0, isFeatured: 0, password: '', originalUrl: '' })
const articleId = computed(() => typeof route.params.articleId === 'string' ? route.params.articleId : '')
const isEditing = computed(() => /^\d+$/.test(articleId.value))

onMounted(() => {
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
    await saveAdminArticle({ ...form, tagNames: form.tagNames.split(',').map((value) => value.trim()).filter(Boolean) })
    Message.success('文章已保存')
    await router.push('/article-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '文章保存失败')
  } finally {
    saving.value = false
  }
}

function normalizeTags(value: unknown): string {
  if (Array.isArray(value)) return value.map(String).join(', ')
  if (value && typeof value === 'object' && 'root' in value) return ''
  return typeof value === 'string' ? value : ''
}
</script>

<style scoped>
.article-form { margin-top: 18px; }
.article-form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
@media (max-width: 800px) { .article-form-grid { grid-template-columns: 1fr; } }
</style>
