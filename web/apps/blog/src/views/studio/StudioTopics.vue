<template>
  <div class="studio-topics">
    <header class="studio-page-head">
      <div><p>WORKSPACE / 05</p><h1>分类与标签</h1><span>这些词汇只属于你的空间，用于组织文章与公开主页。</span></div>
    </header>

    <div class="studio-topics__grid">
      <section class="topic-panel">
        <header><div><p>CATEGORIES</p><h2>分类</h2></div><span>{{ categories.length }}</span></header>
        <form @submit.prevent="saveCategory"><input v-model.trim="categoryName" placeholder="新分类名称" /><button type="submit">{{ categoryId ? '更新' : '添加' }}</button><button v-if="categoryId" type="button" @click="resetCategory">取消</button></form>
        <ul>
          <li v-for="item in categories" :key="item.id"><span>{{ item.categoryName }}<small>{{ item.articleCount || 0 }} 篇文章</small></span><div><button type="button" @click="editCategory(item)">编辑</button><button type="button" class="danger" @click="removeCategory(item)">删除</button></div></li>
        </ul>
      </section>

      <section class="topic-panel">
        <header><div><p>TAGS</p><h2>标签</h2></div><span>{{ tags.length }}</span></header>
        <form @submit.prevent="saveTag"><input v-model.trim="tagName" placeholder="新标签名称" /><button type="submit">{{ tagId ? '更新' : '添加' }}</button><button v-if="tagId" type="button" @click="resetTag">取消</button></form>
        <ul>
          <li v-for="item in tags" :key="item.id"><span>#{{ item.tagName }}<small>{{ item.count || item.articleCount || 0 }} 次使用</small></span><div><button type="button" @click="editTag(item)">编辑</button><button type="button" class="danger" @click="removeTag(item)">删除</button></div></li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api/api'

export default defineComponent({
  name: 'StudioTopics',
  setup() {
    const categories = ref<any[]>([])
    const tags = ref<any[]>([])
    const categoryName = ref('')
    const categoryId = ref(0)
    const tagName = ref('')
    const tagId = ref(0)
    const dataOf = (response: any) => response?.data?.data || []

    const load = async () => {
      try {
        const [categoryResponse, tagResponse] = await Promise.all([api.getStudioCategories(), api.getStudioTags()])
        categories.value = Array.isArray(dataOf(categoryResponse)) ? dataOf(categoryResponse) : []
        tags.value = Array.isArray(dataOf(tagResponse)) ? dataOf(tagResponse) : []
      } catch {
        ElMessage.error('分类与标签加载失败')
      }
    }
    const resetCategory = () => { categoryId.value = 0; categoryName.value = '' }
    const resetTag = () => { tagId.value = 0; tagName.value = '' }
    const editCategory = (item: any) => { categoryId.value = item.id; categoryName.value = item.categoryName }
    const editTag = (item: any) => { tagId.value = item.id; tagName.value = item.tagName }
    const saveCategory = async () => {
      if (!categoryName.value) return
      try {
        await api.saveStudioCategory({ id: categoryId.value, name: categoryName.value }, categoryId.value || undefined)
        resetCategory(); await load(); ElMessage.success('分类已保存')
      } catch (reason: any) { ElMessage.error(reason?.response?.data?.message || '分类保存失败') }
    }
    const saveTag = async () => {
      if (!tagName.value) return
      try {
        await api.saveStudioTag({ id: tagId.value, name: tagName.value }, tagId.value || undefined)
        resetTag(); await load(); ElMessage.success('标签已保存')
      } catch (reason: any) { ElMessage.error(reason?.response?.data?.message || '标签保存失败') }
    }
    const removeCategory = async (item: any) => {
      try {
        await ElMessageBox.confirm(`确认删除分类“${item.categoryName}”？`, '删除确认', { type: 'warning' })
        await api.deleteStudioCategory(item.id); await load()
      } catch (reason: any) { if (reason !== 'cancel' && reason !== 'close') ElMessage.error(reason?.response?.data?.message || '删除失败') }
    }
    const removeTag = async (item: any) => {
      try {
        await ElMessageBox.confirm(`确认删除标签“#${item.tagName}”？`, '删除确认', { type: 'warning' })
        await api.deleteStudioTag(item.id); await load()
      } catch (reason: any) { if (reason !== 'cancel' && reason !== 'close') ElMessage.error(reason?.response?.data?.message || '删除失败') }
    }

    onMounted(load)
    return { categories, tags, categoryName, categoryId, tagName, tagId, resetCategory, resetTag, editCategory, editTag, saveCategory, saveTag, removeCategory, removeTag }
  }
})
</script>

<style lang="scss" scoped>
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 28px 30px; border: 1px solid var(--border-hairline); border-radius: 20px; background: radial-gradient(circle at 86% 0, rgba(97, 73, 184, .18), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p, .topic-panel header p { margin: 0 0 7px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 6px; font-size: clamp(1.8rem, 4vw, 3rem); letter-spacing: -.05em; }
.studio-page-head span { color: var(--text-ob-dim); font-size: 12px; }
.studio-topics__grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; margin-top: 18px; }
.topic-panel { padding: 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.topic-panel > header { display: flex; justify-content: space-between; margin-bottom: 16px; }
.topic-panel h2 { margin: 0; }
.topic-panel header > span { color: var(--text-ob-dim); }
.topic-panel form { display: flex; gap: 7px; margin-bottom: 14px; }
.topic-panel input { min-width: 0; flex: 1; padding: 10px 12px; border: 1px solid var(--border-hairline); border-radius: 10px; outline: none; background: var(--background-primary); color: inherit; }
.topic-panel button { padding: 8px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.topic-panel form button[type='submit'] { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
.topic-panel ul { display: grid; gap: 7px; margin: 0; padding: 0; list-style: none; }
.topic-panel li { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 12px; border: 1px solid var(--border-hairline); border-radius: 11px; }
.topic-panel li span, .topic-panel li small { display: block; }
.topic-panel li small { margin-top: 3px; color: var(--text-ob-dim); font-size: 10px; }
.topic-panel li div { display: flex; gap: 5px; }
.topic-panel li button { padding: 5px 8px; font-size: 10px; }
.topic-panel li button.danger { color: #df8177; }
@media (max-width: 760px) { .studio-topics__grid { grid-template-columns: 1fr; } }
</style>