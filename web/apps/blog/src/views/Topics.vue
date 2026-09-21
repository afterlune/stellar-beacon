<template>
  <div class="topics-page">
    <header class="topics-head">
      <p>TOPIC PLAZA</p>
      <h1>话题广场</h1>
      <span>按近 7 天的收藏、评论、点赞与阅读热度排列，找到此刻正在被讨论的分类、标签与系列。</span>
    </header>

    <p v-if="loading && !hasTopics" class="topics-state">正在汇合公共话题…</p>
    <p v-else-if="error" class="topics-state is-error">{{ error }}</p>
    <template v-else>
      <section v-for="group in groups" :key="group.key" class="topics-group">
        <header>
          <div>
            <p>{{ group.eyebrow }}</p>
            <h2>{{ group.title }}</h2>
          </div>
          <span>{{ group.items.length }} 个话题</span>
        </header>
        <div v-if="group.items.length" class="topics-grid">
          <router-link v-for="item in group.items" :key="`${group.key}-${item.id}`" :to="group.path(item)" class="topic-card">
            <strong>{{ item.name }}</strong>
            <p v-if="item.description">{{ item.description }}</p>
            <small>
              <span>{{ item.articleCount }} 篇文章</span>
              <em v-if="item.hotScore > 0">近 7 天热度 {{ item.hotScore }}</em>
              <em v-else>近期暂无互动</em>
            </small>
          </router-link>
        </div>
        <p v-else class="topics-state">这一类还没有公开话题。</p>
      </section>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'

export default defineComponent({
  name: 'Topics',
  setup() {
    const categories = ref<any[]>([])
    const tags = ref<any[]>([])
    const series = ref<any[]>([])
    const loading = ref(false)
    const error = ref('')

    const hasTopics = computed(() => categories.value.length + tags.value.length + series.value.length > 0)
    const groups = computed(() => [
      {
        key: 'categories',
        eyebrow: 'CATEGORIES',
        title: '分类',
        items: categories.value,
        path: (item: any) => `/categories/${item.id}?name=${encodeURIComponent(item.name || '')}`
      },
      {
        key: 'tags',
        eyebrow: 'TAGS',
        title: '标签',
        items: tags.value,
        path: (item: any) => `/tags/${item.id}?tagName=${encodeURIComponent(item.name || '')}`
      },
      {
        key: 'series',
        eyebrow: 'COLLECTIONS',
        title: '系列',
        items: series.value,
        path: (item: any) => `/series/${item.id}`
      }
    ])

    const load = async () => {
      loading.value = true
      error.value = ''
      try {
        const response = await api.getTopicOverview({ size: 12 })
        const data = response?.data?.data || {}
        categories.value = Array.isArray(data.categories) ? data.categories : []
        tags.value = Array.isArray(data.tags) ? data.tags : []
        series.value = Array.isArray(data.series) ? data.series : []
      } catch {
        error.value = '话题广场暂时无法加载，请稍后重试。'
      } finally {
        loading.value = false
      }
    }

    onMounted(() => { void load() })

    return { categories, tags, series, groups, hasTopics, loading, error }
  }
})
</script>

<style scoped>
.topics-page { max-width: 1120px; margin: 0 auto; padding: 28px 0 96px; }
.topics-head { margin-bottom: 24px; padding: clamp(26px, 5vw, 48px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 88% 6%, rgba(103, 72, 188, .22), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.topics-head p, .topics-group header p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.topics-head h1 { margin: 0 0 10px; font-size: clamp(2rem, 4vw, 3.4rem); letter-spacing: -.04em; }
.topics-head span { color: var(--text-ob-dim); line-height: 1.8; }
.topics-group { margin-top: 22px; padding: clamp(20px, 4vw, 32px); border: 1px solid var(--border-hairline); border-radius: 22px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.topics-group header { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.topics-group header h2 { margin: 0; font-size: clamp(1.35rem, 3vw, 1.9rem); }
.topics-group header > span { color: var(--text-ob-dim); font-size: 11px; }
.topics-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.topic-card { display: grid; gap: 8px; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 16px; color: inherit; text-decoration: none; transition: border-color .2s ease, transform .2s ease; }
.topic-card:hover { transform: translateY(-2px); border-color: color-mix(in srgb, var(--color-ob) 55%, transparent); }
.topic-card strong { font-size: 1.02rem; }
.topic-card p { overflow: hidden; margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.6; text-overflow: ellipsis; white-space: nowrap; }
.topic-card small { display: flex; flex-wrap: wrap; gap: 10px; color: var(--text-ob-dim); font-size: 10px; }
.topic-card em { color: var(--color-ob); font-style: normal; }
.topics-state { padding: 46px 0; color: var(--text-ob-dim); text-align: center; }
.topics-state.is-error { color: #df8177; }
@media (max-width: 900px) { .topics-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 620px) { .topics-grid { grid-template-columns: 1fr; } }
</style>