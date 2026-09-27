<template>
  <div class="flex flex-col">
    <PageHeader :title="t('menu.archives')" :current="t('menu.archives')" />
    <div class="archive-panel">
      <section
        class="archive-group"
        v-for="archive in archives"
        :key="t(`settings.months[${archive.time.split('-')[1]}]`) + '-' + archive.time.split('-')[0]">
        <h2 class="archive-group-title">
          <span>
            {{ t(`settings.months[${archive.time.split('-')[1] - 1}]`) }}&nbsp{{ archive.time.split('-')[0] }}
          </span>
          <span class="archive-group-count">{{ archive.articles.length }}</span>
        </h2>
        <ul class="archive-list">
          <li class="archive-entry" v-for="article in archive.articles" :key="article.id">
            <span class="archive-marker" />
            <div class="archive-entry-body">
              <router-link :to="'/articles/' + article.id" class="archive-entry-title">
                {{ article.articleTitle }}
              </router-link>
              <time class="archive-entry-date">
                {{ t(`settings.months[${new Date(article.createTime).getMonth()}]`) }}
                {{ new Date(article.createTime).getDate() }}, {{ new Date(article.createTime).getFullYear() }}
              </time>
              <p class="archive-entry-excerpt">
                {{ article.articleContent }}
              </p>
            </div>
          </li>
        </ul>
      </section>
      <Paginator
        :pageSize="pagination.size"
        :pageTotal="pagination.total"
        :page="pagination.current"
        @pageChange="pageChangeHanlder" />
    </div>
  </div>
</template>

<script lang="ts">
import { useArticleStore } from '@/stores/article'
import { useCommonStore } from '@/stores/common'
import { defineComponent, onMounted, onUnmounted, reactive, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { PageHeader } from '@/components/PageHeader'
import Paginator from '@/components/Paginator.vue'
import MarkdownIt from 'markdown-it'
import api from '@/api/api'

export default defineComponent({
  name: 'Archives',
  components: { PageHeader, Paginator },
  setup() {
    const commonStore = useCommonStore()
    const articleStore = useArticleStore()
    const { t } = useI18n()
    const md = new MarkdownIt({ html: true })
    const pagination = reactive({
      current: 1,
      total: 0,
      size: 12
    })
    onMounted(() => {
      toPageTop()
      fetchArchives()
    })
    onUnmounted(() => {
      commonStore.resetHeaderImage()
    })
    const fetchArchives = () => {
      articleStore.archives = ''
      api
        .getAllArchives({
          current: pagination.current,
          size: pagination.size
        })
        .then(({ data }) => {
          const page = data?.data || {}
          const archives = Array.isArray(page.items)
            ? page.items
            : Array.isArray(page.records)
              ? page.records
              : []
          archives.forEach((item: any) => {
            item.articles.forEach((article: any) => {
              article.articleContent = md
                .render(article.articleContent)
                .replace(/<\/?[^>]*>/g, '')
                .replace(/[|]*\n/, '')
                .replace(/&npsp;/gi, '')
            })
          })
          articleStore.archives = archives
          pagination.total = Number(page.total ?? page.count ?? archives.length)
        })
    }
    const pageChangeHanlder = (current: number) => {
      pagination.current = current
      toPageTop()
      fetchArchives()
    }
    const toPageTop = () => {
      window.scrollTo({
        top: 0
      })
    }
    return {
      pageChangeHanlder,
      pagination,
      archives: toRef(articleStore.$state, 'archives'),
      t
    }
  }
})
</script>

<style lang="scss" scoped>
/* The old timeline put the axis in the middle of a 1120px card, so reading a
   row meant a ~700px eye jump and half the width was dead space. The axis is
   now pinned left and the month is a group header, not a centred node. */
.archive-panel {
  @apply bg-ob-deep-800 rounded-2xl block;
  border: none;
  box-shadow: inset 0 1px 0 var(--glass-edge), var(--shadow-card);
  padding: 32px clamp(20px, 4vw, 56px) 40px;
}

.archive-group + .archive-group {
  margin-top: 8px;
}

.archive-group-title {
  position: sticky;
  top: calc(var(--header-h) + 12px);
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-dim);
  /* Opaque: a translucent sticky header would let the rows show through it. */
  background-color: var(--surface-solid);
  border-bottom: 1px solid var(--border-hairline);
}

.archive-group-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--radius-pill);
  background-color: var(--surface-2);
  border: none;
  font-size: 11px;
  letter-spacing: 0;
}

.archive-list {
  position: relative;
  padding-left: 20px;

  &::before {
    content: "";
    position: absolute;
    left: 3px;
    top: 6px;
    bottom: 6px;
    width: 1px;
    background: var(--border-hairline);
  }
}

.archive-entry {
  position: relative;
  padding: 0 0 22px 0;

  &:last-child {
    padding-bottom: 4px;
  }
}

.archive-marker {
  position: absolute;
  left: -20px;
  top: 7px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-accent);
  transition: box-shadow 200ms ease;
}

.archive-entry:hover .archive-marker {
  box-shadow: 0 0 0 4px var(--surface-hover);
}

.archive-entry-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: baseline;
  column-gap: 16px;
}

.archive-entry-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--text-bright);
  transition: color 200ms ease;

  &:hover {
    color: var(--text-accent);
  }
}

.archive-entry-date {
  font-size: 12px;
  color: var(--text-dim);
  white-space: nowrap;
}

.archive-entry-excerpt {
  grid-column: 1 / -1;
  margin-top: 4px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  word-wrap: break-word;
  word-break: break-word;
}

@media (max-width: 767px) {
  .archive-entry-body {
    grid-template-columns: minmax(0, 1fr);
  }
  .archive-entry-date {
    margin-top: 2px;
  }
}
</style>
