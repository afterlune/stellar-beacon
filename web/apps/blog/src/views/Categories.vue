<template>
  <div class="categories-page">
    <PageHeader :title="t('menu.categories')" :current="t('menu.categories')" />
    <section class="categories-page__panel">
      <p v-if="loading" class="categories-page__status">{{ t('reactions.loading') }}</p>
      <p v-else-if="categories.length === 0" class="categories-page__status">{{ t('categories.empty') }}</p>
      <div v-else class="categories-page__grid" data-testid="categories-grid">
        <router-link
          v-for="category in categories"
          :key="category.id"
          class="categories-page__card"
          :to="`/categories/${category.id}?name=${encodeURIComponent(category.categoryName)}`">
          <span>{{ t('categories.articleCount', { count: category.articleCount }) }}</span>
          <strong>{{ category.categoryName }}</strong>
          <em>{{ t('categories.browse') }} →</em>
        </router-link>
      </div>
    </section>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'

import { PageHeader } from '@/components/PageHeader'
import { useDiscoveryStore } from '@/stores/discovery'

export default defineComponent({
  name: 'Categories',
  components: { PageHeader },
  setup() {
    const { t } = useI18n()
    const discoveryStore = useDiscoveryStore()
    const { categories, loading } = storeToRefs(discoveryStore)
    const orderedCategories = computed(() => [...categories.value].sort((left, right) => (
      Number(right.articleCount || 0) - Number(left.articleCount || 0)
      || String(left.categoryName || '').localeCompare(String(right.categoryName || ''))
    )))

    onMounted(() => void discoveryStore.load())

    return { t, categories: orderedCategories, loading }
  }
})
</script>

<style scoped>
.categories-page__panel {
  padding: clamp(20px, 4vw, 42px);
  border: 1px solid var(--border-hairline);
  border-radius: 18px;
  background: color-mix(in srgb, var(--surface-solid) 86%, transparent);
  box-shadow: inset 0 1px 0 var(--glass-edge), var(--shadow-card);
}

.categories-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.categories-page__card {
  display: grid;
  gap: 8px;
  min-height: 150px;
  padding: 18px;
  border: 1px solid var(--border-hairline);
  border-radius: 14px;
  color: inherit;
  text-decoration: none;
  transition: border-color 0.2s ease, transform 0.2s ease;
}

.categories-page__card:hover {
  border-color: var(--color-ob);
  transform: translateY(-2px);
}

.categories-page__card span,
.categories-page__card em,
.categories-page__status {
  color: var(--text-ob-dim);
  font-size: 0.78rem;
  font-style: normal;
}

.categories-page__card strong {
  align-self: center;
  font-family: var(--font-display);
  font-size: 1.35rem;
}

.categories-page__status {
  margin: 0;
  text-align: center;
}
</style>