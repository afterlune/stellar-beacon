<template>
  <div class="feature-block">
    <div class="feature-head">
      <span class="feature-head-icon">
        <svg-icon icon-class="hot" />
      </span>
      <h2 class="feature-head-title">
        <span class="feature-head-kicker">EDITOR'S SELECTION</span>
        <span class="feature-head-label">{{ t('home.recommended') }}</span>
      </h2>
      <span class="feature-head-rule" />
    </div>

    <ul class="grid lg:grid-cols-2 gap-6">
      <template v-if="featuredArticles.length > 0">
        <li v-for="article in featuredArticles" :key="article.id">
          <ArticleCard class="home-featured-article" :data="article" />
        </li>
      </template>
      <template v-else>
        <li v-for="n in 2" :key="n">
          <ArticleCard :data="{}" />
        </li>
      </template>
    </ul>
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { useArticleStore } from '@/stores/article'
import { useI18n } from 'vue-i18n'
import { defineComponent, toRef } from 'vue'
import { ArticleCard } from '@/components/ArticleCard'

export default defineComponent({
  name: 'FeatureList',
  components: {
    ArticleCard
  },
  setup() {
    const articleStore = useArticleStore()
    const { t } = useI18n()
    return {
      featuredArticles: toRef(articleStore.$state, 'featuredArticles'),
      t
    }
  }
})
</script>

<style lang="scss">
/* Keep the featured section heading separate from the card grid. */
.feature-block {
  padding: 32px 0 8px;
}

.feature-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.feature-head-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background-image: var(--brand-gradient);
  color: #fff;
  flex: none;
}

.feature-head-title {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
}

.feature-head-kicker {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-dim);
}

.feature-head-label {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--text-bright);
}

.feature-head-rule {
  flex: 1;
  height: 1px;
  min-width: 24px;
  background: linear-gradient(90deg, var(--border-hairline), transparent);
}

@media (max-width: 767px) {
  .feature-block {
    padding-top: 24px;
  }
  .feature-head-kicker {
    display: none;
  }
}

.home-featured-article {
  .article-content {
    p {
      overflow: hidden;
      text-overflow: ellipsis;
      display: -webkit-box;
      -webkit-line-clamp: 4;
      -webkit-box-orient: vertical;
    }
    .article-footer {
      margin-top: 13px;
    }
  }
}
</style>
