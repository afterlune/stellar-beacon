<template>
  <div class="favorites-page">
    <PageHeader :title="t('reactions.favoritesTitle')" />
    <div v-if="!userToken" class="favorites-empty">
      <p>{{ t('reactions.loginRequired') }}</p>
      <button type="button" class="favorites-action" @click="openLogin">{{ t('settings.login') }}</button>
    </div>
    <template v-else>
      <p v-if="loading" class="favorites-hint">{{ t('reactions.loading') }}</p>
      <p v-else-if="articles.length === 0" class="favorites-hint">{{ t('reactions.favoritesEmpty') }}</p>
      <div v-else class="favorites-list">
        <ArticleCard v-for="article in articles" :key="article.id" :data="article" />
      </div>
      <div v-if="total > pageInfo.size" class="favorites-pager">
        <button type="button" :disabled="pageInfo.current <= 1" @click="goTo(pageInfo.current - 1)">
          {{ t('reactions.previous') }}
        </button>
        <span>{{ pageInfo.current }} / {{ Math.max(1, Math.ceil(total / pageInfo.size)) }}</span>
        <button type="button" :disabled="pageInfo.current >= Math.ceil(total / pageInfo.size)" @click="goTo(pageInfo.current + 1)">
          {{ t('reactions.next') }}
        </button>
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { ArticleCard } from '@/components/ArticleCard'
import { PageHeader } from '@/components/PageHeader'
import api from '@/api/api'

export default defineComponent({
  name: 'Favorites',
  components: { ArticleCard, PageHeader },
  setup() {
    const { t } = useI18n()
    const userStore = useUserStore()
    const userToken = computed(() => Boolean(userStore.token))
    const loading = ref(false)
    const articles = ref<any[]>([])
    const total = ref(0)
    const pageInfo = reactive({ current: 1, size: 10 })

    const openLogin = () => {
      userStore.userVisible = true
    }

    const fetchFavorites = () => {
      if (!userStore.token) {
        articles.value = []
        total.value = 0
        return
      }
      loading.value = true
      api
        .getMyArticleReactions({ reaction: 'favorite', current: pageInfo.current, size: pageInfo.size })
        .then(({ data }: any) => {
          const payload = data?.data || {}
          articles.value = Array.isArray(payload.records) ? payload.records : []
          total.value = Number(payload.count || 0)
        })
        .catch(() => {
          articles.value = []
          total.value = 0
        })
        .finally(() => {
          loading.value = false
        })
    }

    const goTo = (page: number) => {
      pageInfo.current = page
      fetchFavorites()
    }

    onMounted(fetchFavorites)

    return { t, userToken, loading, articles, total, pageInfo, openLogin, goTo }
  }
})
</script>

<style lang="scss" scoped>
.favorites-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 2rem 1rem 4rem;
}
.favorites-hint { opacity: .7; }
.favorites-list { display: flex; flex-direction: column; gap: 1rem; }
.favorites-empty { display: flex; flex-direction: column; align-items: flex-start; gap: .75rem; }
.favorites-action {
  padding: .45rem .95rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.favorites-action:hover { border-color: var(--color-ob); color: var(--color-ob); }
.favorites-pager { display: flex; align-items: center; justify-content: center; gap: 1rem; margin-top: 2rem; }
.favorites-pager button {
  padding: .4rem .9rem;
  border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.favorites-pager button:disabled { opacity: .5; cursor: not-allowed; }
</style>
