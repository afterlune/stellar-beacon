<template>
  <div class="flex flex-col">
    <PageHeader :title="t('menu.tags')" :current="t('menu.tags')" />
    <div class="surface-panel px-14 py-16 rounded-2xl block">
      <TagList>
        <template v-if="tags != '' && tags.length > 0">
          <TagItem
            v-for="tag in orderedTags"
            :key="tag.id"
            :id="tag.id"
            :name="tag.tagName"
            :count="tag.count"
            :size="sizeFor(Number(tag.count) || 0)" />
        </template>
      </TagList>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, onUnmounted, toRef } from 'vue'
import { PageHeader } from '@/components/PageHeader'
import { useI18n } from 'vue-i18n'
import { useTagStore } from '@/stores/tag'
import { TagList, TagItem } from '@/components/Tag'
import { useCommonStore } from '@/stores/common'
import api from '@/api/api'

export default defineComponent({
  name: 'Tag',
  components: { PageHeader, TagList, TagItem },
  setup() {
    const commonStore = useCommonStore()
    const { t } = useI18n()
    const tagStore = useTagStore()
    onMounted(() => {
      fetchTags()
    })
    onUnmounted(() => {
      commonStore.resetHeaderImage()
    })
    const fetchTags = () => {
      api.getAllTags().then(({ data }) => {
        tagStore.tags = data.data
      })
    }
    // Index pages rank by volume instead of flattening every tag to one size.
    const orderedTags = computed(() => [...(tagStore.tags || [])]
      .sort((left: any, right: any) => Number(right.count || 0) - Number(left.count || 0)
        || String(left.tagName || '').localeCompare(String(right.tagName || ''))))
    const sizeFor = (count: number): string => {
      if (count >= 5) return 'xl'
      if (count >= 2) return 'lg'
      if (count >= 1) return 'md'
      return 'sm'
    }
    return {
      tags: toRef(tagStore.$state, 'tags'),
      orderedTags,
      sizeFor,
      t
    }
  }
})
</script>

<style lang="scss" scoped></style>
