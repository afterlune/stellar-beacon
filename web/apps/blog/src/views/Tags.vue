<template>
  <div class="flex flex-col">
    <PageHeader :title="t('menu.tags')" :current="t('menu.tags')" />
    <div class="surface-panel px-14 py-16 rounded-2xl block">
      <TagList>
        <template v-if="tags != '' && tags.length > 0">
          <TagItem v-for="tag in tags" :key="tag.id" :id="tag.id" :name="tag.tagName" :count="tag.count" size="xl" />
        </template>
      </TagList>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, onUnmounted, toRef } from 'vue'
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
    return {
      tags: toRef(tagStore.$state, 'tags'),
      t
    }
  }
})
</script>

<style lang="scss" scoped></style>
