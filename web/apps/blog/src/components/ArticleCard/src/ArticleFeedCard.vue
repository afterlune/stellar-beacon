<template>
  <article class="feed-card" :class="{ 'is-compact': compact }" data-testid="article-feed-card">
    <router-link v-if="showCover" class="feed-card__cover" :to="articlePath" :aria-label="data?.articleTitle || '打开文章'">
      <img v-if="cover && !imageFailed" :src="cover" :alt="data?.articleTitle || ''" loading="lazy" @error="imageFailed = true" />
      <span v-else class="feed-card__placeholder">{{ fallbackLabel }}</span>
      <em v-if="data?.isFeatured === 1">精选</em>
    </router-link>
    <div class="feed-card__body">
      <div v-if="authorName" class="feed-card__author">
        <img :src="author?.avatar || defaultAvatar" :alt="authorName" loading="lazy" />
        <router-link v-if="author?.handle" :to="`/u/${author.handle}`">{{ authorName }}</router-link>
        <span v-else>{{ authorName }}</span>
        <time v-if="data?.createTime">{{ formatDate(data.createTime) }}</time>
      </div>
      <router-link class="feed-card__title" :to="articlePath">
        <span v-if="data?.highlightedTitle" v-html="safeSearchHighlight(data.highlightedTitle)" />
        <span v-else>{{ data?.articleTitle || '未命名文章' }}</span>
      </router-link>
      <p class="feed-card__excerpt">
        <span v-if="data?.highlightedContent" v-html="safeSearchHighlight(data.highlightedContent)" />
        <span v-else>{{ excerpt(data?.articleContent || data?.note || '') }}</span>
      </p>
      <footer class="feed-card__meta">
        <slot name="meta">
          <span>{{ data?.categoryName || '未分类' }}</span>
          <span v-if="data?.likeCount">{{ data.likeCount }} 赞</span>
          <span v-if="data?.favoriteCount">{{ data.favoriteCount }} 收藏</span>
        </slot>
      </footer>
      <slot name="actions" />
    </div>
  </article>
</template>

<script lang="ts">
import { computed, defineComponent, ref } from 'vue'
import { safeSearchHighlight } from '@/utils/search'

const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="80" height="80"%3E%3Crect width="80" height="80" rx="40" fill="%23172554"/%3E%3Ccircle cx="40" cy="30" r="14" fill="%239bb8ff"/%3E%3Cpath d="M15 72c3-18 14-27 25-27s22 9 25 27" fill="%239bb8ff"/%3E%3C/svg%3E'

export default defineComponent({
  name: 'ArticleFeedCard',
  props: {
    data: { type: Object, required: true },
    compact: { type: Boolean, default: false },
    showCover: { type: Boolean, default: true },
    excerptLength: { type: Number, default: 180 }
  },
  setup(props) {
    const imageFailed = ref(false)
    const author = computed(() => (props.data as any)?.author || {})
    const authorName = computed(() => author.value.nickname || author.value.handle || '')
    const cover = computed(() => String((props.data as any)?.articleCover || '').trim())
    const fallbackLabel = computed(() => String((props.data as any)?.categoryName || (props.data as any)?.articleTitle || 'S').trim().charAt(0) || 'S')
    const articlePath = computed(() => `/articles/${Number((props.data as any)?.id || 0)}`)
    const excerpt = (value: unknown): string => {
      const text = String(value || '')
        .replace(/<[^>]*>/g, ' ')
        .replace(/\s+/g, ' ')
        .trim()
      if (text.length <= props.excerptLength) return text
      return text.slice(0, props.excerptLength).trimEnd() + '…'
    }
    const formatDate = (value: string | number | Date): string => {
      const date = new Date(value)
      if (Number.isNaN(date.getTime())) return ''
      return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric', year: 'numeric' }).format(date)
    }
    return { imageFailed, author, authorName, cover, fallbackLabel, articlePath, excerpt, formatDate, defaultAvatar, safeSearchHighlight }
  }
})
</script>

<style scoped>
.feed-card {
  display: flex;
  min-width: 0;
  height: 100%;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--border-hairline);
  border-radius: 18px;
  background: color-mix(in srgb, var(--background-primary) 94%, transparent);
  transition: border-color 180ms ease, transform 180ms ease, box-shadow 180ms ease;
}

.feed-card:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--color-ob) 45%, var(--border-hairline));
  box-shadow: 0 16px 36px rgba(7, 14, 36, .16);
}

.feed-card__cover {
  position: relative;
  display: block;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  background: linear-gradient(135deg, #152250, #3c2c70);
}

.feed-card__cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 450ms ease;
}

.feed-card:hover .feed-card__cover img { transform: scale(1.035); }

.feed-card__cover em {
  position: absolute;
  top: 12px;
  left: 12px;
  padding: 4px 9px;
  border-radius: 999px;
  background: rgba(7, 14, 36, .72);
  color: #dbe6ff;
  font-size: 10px;
  font-style: normal;
  backdrop-filter: blur(8px);
}

.feed-card__placeholder {
  display: grid;
  height: 100%;
  place-items: center;
  color: rgba(218, 228, 255, .85);
  font-size: 3.5rem;
  font-weight: 800;
}

.feed-card__body {
  display: flex;
  flex: 1;
  flex-direction: column;
  padding: 16px;
}

.feed-card__author {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  margin-bottom: 14px;
  color: var(--text-ob-dim);
  font-size: 12px;
}

.feed-card__author img {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--background-primary);
}

.feed-card__author a { display: inline-flex; min-height: 24px; align-items: center; padding: 0 2px; color: inherit; text-decoration: none; }
.feed-card__author a:hover { color: var(--color-ob); }
.feed-card__author time { margin-left: auto; white-space: nowrap; }

.feed-card__title {
  display: -webkit-box;
  min-height: 2.9em;
  overflow: hidden;
  color: inherit;
  font-size: 17px;
  font-weight: 700;
  line-height: 1.45;
  text-decoration: none;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.feed-card__title:hover { color: var(--color-ob); }

.feed-card__excerpt {
  display: -webkit-box;
  min-height: 5.1em;
  margin: 12px 0;
  overflow: hidden;
  color: var(--text-ob-dim);
  font-size: 13px;
  line-height: 1.7;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.feed-card__meta {
  display: flex;
  gap: 12px;
  margin-top: auto;
  padding-top: 14px;
  color: var(--text-ob-dim);
  font-size: 11px;
}

.feed-card :deep(mark) {
  border-radius: .2em;
  background: color-mix(in srgb, var(--color-ob) 28%, transparent);
  color: inherit;
}

.feed-card.is-compact .feed-card__cover { aspect-ratio: 16 / 8; }
.feed-card.is-compact .feed-card__body { padding: 14px; }
.feed-card.is-compact .feed-card__title { min-height: 24px; font-size: 15px; -webkit-line-clamp: 1; }
.feed-card.is-compact .feed-card__excerpt { min-height: 0; margin: 8px 0; -webkit-line-clamp: 2; }
</style>
