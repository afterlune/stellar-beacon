<template>
  <div class="authors-page">
    <header class="authors-head">
      <p>AUTHOR DIRECTORY</p>
      <h1>发现作者</h1>
      <span>关注持续写作的人，他们的新文章和随想会进入你的关注动态。</span>
    </header>
    <p v-if="loading && !records.length" class="authors-state">正在加载作者…</p>
    <p v-else-if="error" class="authors-state is-error">{{ error }}</p>
    <div v-else class="authors-grid">
      <article v-for="author in records" :key="author.id" class="author-card">
        <router-link :to="`/u/${author.handle}`"><img :src="author.avatar || defaultAvatar" :alt="author.nickname || author.handle" /></router-link>
        <div class="author-card__copy">
          <router-link :to="`/u/${author.handle}`"><strong>{{ author.nickname || author.handle }}</strong></router-link>
          <span>@{{ author.handle }}</span>
          <p>{{ author.intro || '这位作者还没有写下简介。' }}</p>
          <small>{{ author.followerCount || 0 }} 关注者 · {{ author.articleCount || 0 }} 篇文章 · {{ author.talkCount || 0 }} 条随想</small>
        </div>
        <FollowButton :author-id="author.id" :following="Boolean(author.isFollowing)" compact @changed="(value: boolean) => changed(author, value)" />
      </article>
    </div>
    <button v-if="records.length < total" type="button" class="authors-more" :disabled="loading" @click="loadMore">{{ loading ? '加载中…' : '加载更多' }}</button>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'
import FollowButton from '@/components/FollowButton.vue'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'Authors',
  components: { FollowButton },
  setup() {
    const records = ref<any[]>([])
    const loading = ref(false)
    const error = ref('')
    const page = ref(1)
    const pageSize = 12
    const total = ref(0)
    const load = async (reset = false) => {
      if (reset) {
        page.value = 1
        records.value = []
      }
      loading.value = true
      error.value = ''
      try {
        const response = await api.getPlatformAuthors({ current: page.value, size: pageSize })
        const data = response?.data?.data || {}
        const next = Array.isArray(data.records) ? data.records : []
        records.value = reset ? next : records.value.concat(next)
        total.value = Number(data.count || 0)
      } catch {
        error.value = '作者目录加载失败'
      } finally {
        loading.value = false
      }
    }
    const loadMore = () => {
      page.value += 1
      void load(false)
    }
    const changed = (author: any, following: boolean) => {
      author.isFollowing = following
      author.followerCount = Math.max(0, Number(author.followerCount || 0) + (following ? 1 : -1))
    }
    onMounted(() => { void load(true) })
    return { records, loading, error, total, defaultAvatar, loadMore, changed }
  }
})
</script>

<style scoped>
.authors-page { max-width: 1120px; margin: 0 auto; padding: 32px 0 96px; }
.authors-head { margin-bottom: 24px; padding: clamp(26px, 5vw, 48px); border: 1px solid var(--border-hairline); border-radius: 22px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .2), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.authors-head p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.authors-head h1 { margin: 0 0 10px; font-size: clamp(2rem, 4vw, 3.4rem); }
.authors-head span { color: var(--text-ob-dim); }
.authors-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.author-card { display: grid; grid-template-columns: 76px minmax(0, 1fr) auto; gap: 16px; align-items: center; padding: 18px; border: 1px solid var(--border-hairline); border-radius: 17px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.author-card > a img { width: 76px; height: 76px; border-radius: 50%; object-fit: cover; }
.author-card__copy { min-width: 0; }
.author-card__copy strong { color: inherit; font-size: 1.08rem; text-decoration: none; }
.author-card__copy > span, .author-card__copy p, .author-card__copy small { display: block; color: var(--text-ob-dim); }
.author-card__copy > span { margin-top: 3px; font-size: 11px; }
.author-card__copy p { overflow: hidden; margin: 10px 0; line-height: 1.55; text-overflow: ellipsis; white-space: nowrap; }
.author-card__copy small { font-size: 10px; }
.authors-state { padding: 60px 0; color: var(--text-ob-dim); text-align: center; }
.authors-state.is-error { color: #df8177; }
.authors-more { display: block; margin: 26px auto 0; padding: 9px 20px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
@media (max-width: 760px) { .authors-grid { grid-template-columns: 1fr; } .author-card { grid-template-columns: 64px minmax(0, 1fr); } .author-card > a img { width: 64px; height: 64px; } .author-card > .follow-button { grid-column: 2; justify-self: start; } }
</style>