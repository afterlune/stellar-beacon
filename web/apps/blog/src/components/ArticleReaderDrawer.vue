<template>
  <DialogSurface v-model="drawerVisible" variant="drawer-bottom" :title="t('reader.tocTitle')">
    <div class="article-reader-drawer__body" data-testid="article-reader-drawer">
      <nav v-if="tocItems.length" class="reader-toc" :aria-label="t('reader.tocTitle')">
        <button
          v-for="item in tocItems"
          :key="item.id"
          type="button"
          class="reader-toc__item"
          :class="{ 'is-active': item.id === activeHeadingId }"
          :style="{ paddingLeft: `${12 + (item.level - 1) * 18}px` }"
          :aria-current="item.id === activeHeadingId ? 'location' : undefined"
          @click="jumpToHeading(item.id)">
          {{ item.text }}
        </button>
      </nav>
      <p v-else class="reader-toc__empty">{{ t('reader.tocEmpty') }}</p>

      <section v-if="seriesContext" class="reader-series" data-testid="reader-series-progress">
        <div class="reader-series__head">
          <strong>{{ seriesContext.name }}</strong>
          <span>{{ t('series.progress', { current: seriesContext.currentIndex + 1, total: seriesContext.total }) }}</span>
        </div>
        <div class="reader-series__reading">
          <span>{{ t('series.readProgress', { completed: progress.completedCount, total: progress.total }) }}</span>
          <div
            class="reader-series__bar"
            role="progressbar"
            :aria-label="t('series.readProgress', { completed: progress.completedCount, total: progress.total })"
            :aria-valuenow="progress.percent"
            aria-valuemin="0"
            aria-valuemax="100">
            <i :style="{ width: `${progress.percent}%` }" />
          </div>
        </div>
        <div class="reader-series__nav">
          <router-link
            v-if="seriesContext.previous"
            class="reader-series__link"
            :to="'/articles/' + seriesContext.previous.id"
            @click="drawerVisible = false">
            <small>{{ t('series.previous') }}</small>
            <span>{{ seriesContext.previous.articleTitle }}</span>
          </router-link>
          <router-link
            v-if="seriesContext.next"
            class="reader-series__link reader-series__link--next"
            :to="'/articles/' + seriesContext.next.id"
            @click="drawerVisible = false">
            <small>{{ t('series.next') }}</small>
            <span>{{ seriesContext.next.articleTitle }}</span>
          </router-link>
        </div>
        <router-link class="reader-series__index" :to="'/series/' + seriesContext.id" @click="drawerVisible = false">
          {{ t('series.backToSeries') }}
        </router-link>
      </section>
    </div>
  </DialogSurface>
</template>

<script setup lang="ts">
import { computed, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'

import { useReaderStore } from '@/stores/reader'
import { scrollToArticleHeading } from '@/utils/article-reader'
import DialogSurface from '@/components/overlays/DialogSurface.vue'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [value: boolean] }>()
const { t } = useI18n()
const readerStore = useReaderStore()
const { tocItems, activeHeadingId, seriesContext } = storeToRefs(readerStore)

const drawerVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const progress = computed(() => readerStore.seriesSummary(
  Number(seriesContext.value?.id || 0),
  seriesContext.value?.articleIds || []
))

async function jumpToHeading(id: string): Promise<void> {
  drawerVisible.value = false
  await nextTick()
  if (scrollToArticleHeading(id)) readerStore.setActiveHeading(id)
}
</script>

<style scoped>
.article-reader-drawer__body {
  display: grid;
  gap: 22px;
  max-height: 68vh;
  overflow-y: auto;
}

.reader-toc {
  display: grid;
  gap: 4px;
}

.reader-toc__item {
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-normal);
  font: inherit;
  line-height: 1.45;
  text-align: left;
  cursor: pointer;
}

.reader-toc__item:hover,
.reader-toc__item.is-active {
  background: color-mix(in srgb, var(--color-ob) 12%, transparent);
  color: var(--color-ob);
}

.reader-toc__empty {
  margin: 0;
  color: var(--text-ob-dim);
}

.reader-series {
  padding: 14px;
  border: 1px solid var(--border-hairline);
  border-radius: 12px;
  background: color-mix(in srgb, var(--surface-solid) 76%, transparent);
}

.reader-series__head,
.reader-series__reading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.reader-series__head span,
.reader-series__reading > span,
.reader-series__link small {
  color: var(--text-ob-dim);
  font-size: 0.75rem;
}

.reader-series__reading {
  margin-top: 12px;
}

.reader-series__bar {
  width: 48%;
  height: 5px;
  overflow: hidden;
  border-radius: 999px;
  background: color-mix(in srgb, var(--text-ob-dim) 20%, transparent);
}

.reader-series__bar i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--brand-gradient);
  transition: width 0.25s ease;
}

.reader-series__nav {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 14px;
}

.reader-series__link {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--border-hairline);
  border-radius: 9px;
  color: inherit;
  text-decoration: none;
}

.reader-series__link span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reader-series__link--next {
  text-align: right;
}

.reader-series__index {
  display: inline-block;
  margin-top: 12px;
  color: var(--color-ob);
  font-size: 0.8rem;
}

@media (max-width: 420px) {
  .reader-series__nav {
    grid-template-columns: 1fr;
  }

  .reader-series__link--next {
    text-align: left;
  }
}
</style>
