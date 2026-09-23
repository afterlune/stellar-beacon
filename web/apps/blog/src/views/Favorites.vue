<template>
  <div class="favorites-page">
    <PageHeader :title="t('reactions.favoritesTitle')" />
    <div v-if="!userToken" class="favorites-empty">
      <p>{{ t('reactions.loginRequired') }}</p>
      <button type="button" class="favorites-action" @click="openLogin">{{ t('settings.login') }}</button>
    </div>
    <template v-else>
      <nav class="favorites-tabs" aria-label="收藏类型">
        <button type="button" :class="{ active: tab === 'articles' }" @click="switchTab('articles')">文章收藏</button>
        <button type="button" :class="{ active: tab === 'collections' }" @click="switchTab('collections')">书单收藏</button>
      </nav>
      <p v-if="loading" class="favorites-hint">{{ t('reactions.loading') }}</p>
      <template v-else-if="tab === 'articles'">
        <p v-if="articles.length === 0" class="favorites-hint">{{ t('reactions.favoritesEmpty') }}</p>
        <div v-else class="favorites-list">
          <ArticleCard v-for="article in articles" :key="article.id" :data="article" />
        </div>
      </template>
      <template v-else>
        <p v-if="collections.length === 0" class="favorites-hint">还没有收藏书单。</p>
        <div v-else class="favorites-collections">
          <router-link v-for="item in collections" :key="item.id || item.slug" :to="`/collections/${item.slug}`" class="favorite-collection">
            <img v-if="item.cover" :src="item.cover" :alt="item.title" loading="lazy" />
            <span v-else>{{ String(item.title || 'LIST').slice(0, 1) }}</span>
            <div>
              <small>{{ item.articleCount }} 篇文章 · {{ item.likeCount || 0 }} 赞 · {{ item.commentCount || 0 }} 评论</small>
              <h2>{{ item.title }}</h2>
              <p>{{ item.description || '暂无书单简介' }}</p>
            </div>
          </router-link>
        </div>
      </template>
      <div v-if="total > pageInfo.size" class="favorites-pager">
        <button type="button" :disabled="pageInfo.current <= 1" @click="goTo(pageInfo.current - 1)">{{ t('reactions.previous') }}</button>
        <span>{{ pageInfo.current }} / {{ Math.max(1, Math.ceil(total / pageInfo.size)) }}</span>
        <button type="button" :disabled="pageInfo.current >= Math.ceil(total / pageInfo.size)" @click="goTo(pageInfo.current + 1)">{{ t('reactions.next') }}</button>
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { ArticleCard } from '@/components/ArticleCard'
import { PageHeader } from '@/components/PageHeader'
import api from '@/api/api'

type FavoriteTab = 'articles' | 'collections'

export default defineComponent({
  name: 'Favorites',
  components: { ArticleCard, PageHeader },
  setup() {
    const { t } = useI18n()
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const userToken = computed(() => Boolean(userStore.token))
    const tab = ref<FavoriteTab>(route.query.tab === 'collections' ? 'collections' : 'articles')
    const loading = ref(false)
    const articles = ref<any[]>([])
    const collections = ref<any[]>([])
    const total = ref(0)
    const pageInfo = reactive({ current: 1, size: 10 })

    const openLogin = () => { userStore.userVisible = true }
    const fetchFavorites = async () => {
      if (!userStore.token) {
        articles.value = []; collections.value = []; total.value = 0
        return
      }
      loading.value = true
      try {
        if (tab.value === 'collections') {
          const { data } = await api.getMyCollectionReactions({ reaction: 'favorite', current: pageInfo.current, size: pageInfo.size })
          const payload = data?.data || {}
          collections.value = Array.isArray(payload.records) ? payload.records : []
          total.value = Number(payload.count || 0)
        } else {
          const { data } = await api.getMyArticleReactions({ reaction: 'favorite', current: pageInfo.current, size: pageInfo.size })
          const payload = data?.data || {}
          articles.value = Array.isArray(payload.records) ? payload.records : []
          total.value = Number(payload.count || 0)
        }
      } catch {
        articles.value = []; collections.value = []; total.value = 0
      } finally {
        loading.value = false
      }
    }
    const switchTab = (next: FavoriteTab) => {
      if (tab.value === next) return
      tab.value = next; pageInfo.current = 1
      void router.replace({ query: { ...route.query, tab: next } })
      void fetchFavorites()
    }
    const goTo = (page: number) => { pageInfo.current = page; void fetchFavorites() }
    onMounted(fetchFavorites)

    return { t, userToken, tab, loading, articles, collections, total, pageInfo, openLogin, switchTab, goTo }
  }
})
</script>

<style lang="scss" scoped>
.favorites-page { max-width: 980px; margin: 0 auto; padding: 2rem 1rem 4rem; }
.favorites-tabs { display: flex; gap: 8px; margin: 0 0 1.25rem; }
.favorites-tabs button { padding: .45rem 1rem; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.favorites-tabs button.active { border-color: var(--color-ob); color: var(--color-ob); }
.favorites-hint { opacity: .7; }
.favorites-list { display: flex; flex-direction: column; gap: 1rem; }
.favorites-collections { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.favorite-collection { display: grid; grid-template-columns: 132px minmax(0, 1fr); gap: 1rem; overflow: hidden; padding: 1rem; border: 1px solid var(--border-hairline); border-radius: 18px; color: inherit; text-decoration: none; }
.favorite-collection > img, .favorite-collection > span { width: 132px; height: 100px; border-radius: 13px; object-fit: cover; }
.favorite-collection > span { display: grid; place-items: center; background: color-mix(in srgb, var(--color-ob) 12%, transparent); color: var(--color-ob); font-size: 2rem; font-weight: 800; }
.favorite-collection small, .favorite-collection p { color: var(--text-ob-dim); font-size: 11px; }
.favorite-collection h2 { margin: .35rem 0; font-size: 1rem; }
.favorite-collection p { margin: 0; line-height: 1.6; }
.favorites-empty { display: flex; flex-direction: column; align-items: flex-start; gap: .75rem; }
.favorites-action { padding: .45rem .95rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.favorites-action:hover { border-color: var(--color-ob); color: var(--color-ob); }
.favorites-pager { display: flex; align-items: center; justify-content: center; gap: 1rem; margin-top: 2rem; }
.favorites-pager button { padding: .4rem .9rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.favorites-pager button:disabled { opacity: .5; cursor: not-allowed; }
@media (max-width: 720px) { .favorites-collections { grid-template-columns: 1fr; } .favorite-collection { grid-template-columns: 96px minmax(0, 1fr); } .favorite-collection > img, .favorite-collection > span { width: 96px; height: 82px; } }
</style>
