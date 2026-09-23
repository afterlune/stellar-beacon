<template>
  <div class="collection-detail-page">
    <p v-if="loading" class="collection-state">正在打开这本书单…</p>
    <p v-else-if="error" class="collection-state is-error">{{ error }}</p>
    <template v-else-if="detail">
      <header class="collection-hero">
        <div>
          <p>PUBLIC READING LIST</p>
          <h1>{{ detail.collection.title }}</h1>
          <span>{{ detail.collection.description || '这位读者还没有写书单简介。' }}</span>
          <div class="collection-hero__actions">
            <router-link v-if="detail.collection.owner" :to="`/u/${detail.collection.owner.handle}`">
              <img :src="detail.collection.owner.avatar || defaultAvatar" :alt="detail.collection.owner.nickname" />
              {{ detail.collection.owner.nickname || detail.collection.owner.handle }}
            </router-link>
            <router-link v-if="isOwner" :to="`/studio/collections/${detail.collection.id}/edit`">管理书单</router-link>
            <CollectionSubscribeButton v-else :collection-id="Number(detail.collection.id)" />
            <button type="button" class="collection-like" :class="{ 'is-active': liked }" :disabled="likeBusy" @click="toggleLike">
              {{ liked ? '已点赞' : '点赞' }} · {{ likeCount }}
            </button>
            <button type="button" class="collection-favorite" :class="{ 'is-active': favorited }" :disabled="favoriteBusy" @click="toggleFavorite">
              {{ favorited ? '已收藏' : '收藏' }} · {{ favoriteCount }}
            </button>
            <button type="button" @click="share">{{ copied ? '链接已复制' : '分享书单' }}</button>
          </div>
        </div>
        <aside><strong>{{ detail.items.length }}</strong><small>篇文章</small><small>{{ likeCount }} 赞 · {{ favoriteCount }} 收藏 · {{ detail.collection.commentCount || 0 }} 评论</small><em>{{ detail.collection.visibility === 'unlisted' ? '链接可见' : '公开书单' }}</em></aside>
      </header>
      <ol class="collection-items">
        <li v-for="(item, index) in detail.items" :key="item.articleId" :class="{ 'is-highlighted': Number(item.articleId) === highlightedArticleId }" :data-article-id="item.articleId">
          <span class="collection-items__index">{{ String(Number(index) + 1).padStart(2, '0') }}</span>
          <router-link v-if="item.article" :to="`/articles/${item.article.id}`" class="collection-items__main">
            <img v-if="item.article.articleCover" :src="item.article.articleCover" :alt="item.article.articleTitle" loading="lazy" />
            <div>
              <small>{{ item.article.categoryName || '未分类' }} · {{ formatDate(item.article.createTime) }}</small>
              <h2>{{ item.article.articleTitle }}</h2>
              <p>{{ item.note || excerpt(item.article.articleContent) }}</p>
            </div>
          </router-link>
        </li>
      </ol>
      <Comment />
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, nextTick, onMounted, onUnmounted, provide, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'
import { useCommentStore } from '@/stores/comment'
import { Comment } from '@/components/Comment'
import CollectionSubscribeButton from '@/components/CollectionSubscribeButton.vue'
import emitter from '@/utils/mitt'
import { pageCount, pageRecords } from '@/utils/page'

export default defineComponent({
  name: 'CollectionDetail',
  components: { Comment, CollectionSubscribeButton },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const commentStore = useCommentStore()
    const detail = ref<any>(null)
    const loading = ref(true)
    const error = ref('')
    const copied = ref(false)
    const highlightedArticleId = ref(0)
    const comments = ref<any[]>([])
    const haveMore = ref(false)
    const isReload = ref(false)
    const pageInfo = reactive({ current: 1, size: 7 })
    const liked = ref(false)
    const likeCount = ref(0)
    const likeBusy = ref(false)
    const favorited = ref(false)
    const favoriteCount = ref(0)
    const favoriteBusy = ref(false)
    const isOwner = computed(() => {
      const currentID = Number(userStore.userInfo?.userInfoId || userStore.userInfo?.id || 0)
      return currentID > 0 && currentID === Number(detail.value?.collection?.owner?.id || 0)
    })
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="72" height="72"%3E%3Crect width="72" height="72" rx="36" fill="%23172554"/%3E%3Ccircle cx="36" cy="27" r="13" fill="%239bb8ff"/%3E%3Cpath d="M12 67c4-17 12-25 24-25s20 8 24 25" fill="%239bb8ff"/%3E%3C/svg%3E'
    const focusComment = (commentID: number) => {
      if (commentID <= 0) return
      void nextTick(() => {
        const element = document.getElementById(`comment-${commentID}`)
        if (!element) return
        element.scrollIntoView({ behavior: 'smooth', block: 'center' })
        element.classList.remove('comment-focus')
        void element.offsetWidth
        element.classList.add('comment-focus')
        window.setTimeout(() => element.classList.remove('comment-focus'), 2500)
      })
    }
    const fetchComments = async () => {
      const collectionID = Number(detail.value?.collection?.id || 0)
      if (!collectionID) return
      commentStore.type = 6
      commentStore.topicId = String(collectionID)
      const commentID = Number(route.query.comment || 0)
      const params: any = { type: 6, topicId: String(collectionID), current: pageInfo.current, size: pageInfo.size }
      if (commentID > 0) params.focusCommentId = commentID
      const response = await api.getComments(params)
      const records = pageRecords(response?.data)
      const reloading = isReload.value
      if (reloading) {
        comments.value = records
        isReload.value = false
      } else {
        comments.value.push(...records)
      }
      if (detail.value?.collection) detail.value.collection.commentCount = pageCount(response?.data)
      haveMore.value = comments.value.length < pageCount(response?.data)
      if (commentID > 0 && reloading) focusComment(commentID)
    }
    const fetchReplies = async (commentID: number) => {
      const comment = comments.value.find((row: any) => Number(row?.id) === commentID)
      if (!comment?.id) return
      const response = await api.getRepliesByCommentId(comment.id)
      comment.replyDTOs = Array.isArray(response?.data?.data) ? response.data.data : []
    }
    const fetchReactionState = async () => {
      const collectionID = Number(detail.value?.collection?.id || 0)
      if (!collectionID || !userStore.token) return
      try {
        const response = await api.getCollectionReactionState(collectionID)
        liked.value = Boolean(response?.data?.data?.like)
        favorited.value = Boolean(response?.data?.data?.favorite)
      } catch {
        liked.value = false
        favorited.value = false
      }
    }
    const load = async () => {
      loading.value = true; error.value = ''
      try {
        const response = await api.getPublicCollection(String(route.params.slug || ''))
        detail.value = response?.data?.data || null
        if (!detail.value?.collection) throw new Error('missing collection')
        likeCount.value = Number(detail.value.collection.likeCount || 0)
        favoriteCount.value = Number(detail.value.collection.favoriteCount || 0)
        liked.value = false
        favorited.value = false
        pageInfo.current = 1
        isReload.value = true
        comments.value = []
        await Promise.allSettled([fetchComments(), fetchReactionState()])
        highlightedArticleId.value = Number(route.query.article || 0)
        if (highlightedArticleId.value > 0) {
          await nextTick()
          document.querySelector(`[data-article-id="${highlightedArticleId.value}"]`)?.scrollIntoView({ block: 'center' })
        }
      } catch { error.value = '没有找到这个公开书单。'; detail.value = null; comments.value = [] } finally { loading.value = false }
    }
    const toggleCollectionReaction = async (reaction: 'like' | 'favorite') => {
      const collectionID = Number(detail.value?.collection?.id || 0)
      if (!collectionID) return
      if (!userStore.userInfo) {
        await router.push({ path: route.path, query: { ...route.query, login: '1', redirect: route.fullPath } })
        return
      }
      const busy = reaction === 'favorite' ? favoriteBusy : likeBusy
      const active = reaction === 'favorite' ? favorited : liked
      busy.value = true
      try {
        const response = await api.setCollectionReaction({ collectionId: collectionID, reaction, active: !active.value })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '操作失败')
        if (reaction === 'favorite') favorited.value = Boolean(response.data.data?.active)
        else liked.value = Boolean(response.data.data?.active)
        likeCount.value = Number(response.data.data?.likeCount || 0)
        favoriteCount.value = Number(response.data.data?.favoriteCount || 0)
      } catch (reason: any) {
        ElMessage.error(reason?.response?.data?.message || reason?.message || '操作失败')
      } finally {
        busy.value = false
      }
    }
    const toggleLike = () => toggleCollectionReaction('like')
    const toggleFavorite = () => toggleCollectionReaction('favorite')
    const share = async () => {
      const url = window.location.href
      try {
        if (navigator.share) await navigator.share({ title: detail.value?.collection?.title, url })
        else { await navigator.clipboard.writeText(url); copied.value = true; window.setTimeout(() => { copied.value = false }, 1800) }
      } catch { /* user cancelled */ }
    }
    const excerpt = (value: string) => String(value || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 160)
    const formatDate = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value)) : ''
    provide('comments', computed(() => comments.value))
    provide('haveMore', computed(() => haveMore.value))
    provide('collectionId', () => Number(detail.value?.collection?.id || 0))
    provide('canModerate', () => isOwner.value)
    emitter.on('collectionFetchComment', () => { pageInfo.current = 1; isReload.value = true; void fetchComments() })
    emitter.on('collectionFetchReplies', (commentId: any) => { void fetchReplies(Number(commentId)) })
    emitter.on('collectionLoadMore', () => { if (haveMore.value) { pageInfo.current += 1; void fetchComments() } })
    onUnmounted(() => {
      emitter.off('collectionFetchComment')
      emitter.off('collectionFetchReplies')
      emitter.off('collectionLoadMore')
    })
    watch(() => route.params.slug, () => void load())
    onMounted(() => void load())
    return { detail, loading, error, copied, highlightedArticleId, isOwner, defaultAvatar, liked, likeCount, likeBusy, favorited, favoriteCount, favoriteBusy, toggleLike, toggleFavorite, share, excerpt, formatDate }
  }
})
</script>

<style scoped>
.collection-detail-page { max-width: 1020px; margin: 0 auto; padding: 28px 0 96px; }
.collection-hero { display: grid; grid-template-columns: minmax(0, 1fr) 180px; gap: 24px; padding: clamp(28px, 5vw, 52px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 90% 0, rgba(96, 120, 220, .2), transparent 36%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.collection-hero p { margin: 0 0 10px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.collection-hero h1 { margin: 0 0 12px; font-size: clamp(2rem, 5vw, 3.8rem); }
.collection-hero > div > span { color: var(--text-ob-dim); font-size: 13px; line-height: 1.8; }
.collection-hero__actions { display: flex; align-items: center; gap: 12px; margin-top: 24px; }
.collection-hero__actions a, .collection-hero__actions button { display: inline-flex; align-items: center; min-height: 36px; padding: 7px 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; font-size: 12px; text-decoration: none; cursor: pointer; }
.collection-hero__actions .collection-like.is-active, .collection-hero__actions .collection-favorite.is-active { border-color: var(--color-ob); background: color-mix(in srgb, var(--color-ob) 12%, transparent); color: var(--color-ob); }
.collection-hero__actions button:disabled { opacity: .55; cursor: wait; }
.collection-hero__actions img { width: 25px; height: 25px; margin-right: 7px; border-radius: 50%; object-fit: cover; }
.collection-hero aside { display: grid; place-content: center; border: 1px solid var(--border-hairline); border-radius: 18px; text-align: center; }
.collection-hero aside strong { font-size: 2.4rem; }
.collection-hero aside small, .collection-hero aside em { color: var(--text-ob-dim); font-size: 11px; font-style: normal; }
.collection-hero aside small + small { margin-top: 6px; }
.collection-hero aside em { margin-top: 8px; color: var(--color-ob); }
.collection-items { display: grid; gap: 12px; margin: 22px 0 0; padding: 0; list-style: none; }
.collection-items li { display: grid; grid-template-columns: 42px minmax(0, 1fr); gap: 12px; align-items: start; padding: 15px; border: 1px solid var(--border-hairline); border-radius: 17px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.collection-items li.is-highlighted { border-color: var(--color-ob); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-ob) 12%, transparent); }
.collection-items__index { padding-top: 7px; color: var(--color-ob); font-family: ui-monospace, monospace; font-size: 11px; }
.collection-items__main { display: grid; grid-template-columns: minmax(0, 1fr) 180px; gap: 18px; color: inherit; text-decoration: none; }
.collection-items__main img { grid-column: 2; grid-row: 1; width: 100%; aspect-ratio: 16 / 9; border-radius: 12px; object-fit: cover; }
.collection-items__main small { color: var(--text-ob-dim); font-size: 10px; }
.collection-items__main h2 { margin: 7px 0; font-size: 1.1rem; }
.collection-items__main p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.collection-state { padding: 70px 0; color: var(--text-ob-dim); text-align: center; }
.collection-state.is-error { color: #df8177; }
@media (max-width: 680px) { .collection-hero { grid-template-columns: 1fr; } .collection-hero aside { min-height: 120px; } .collection-items__main { grid-template-columns: 1fr; } .collection-items__main img { grid-column: 1; grid-row: auto; } }
</style>
