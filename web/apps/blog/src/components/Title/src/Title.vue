<template>
  <p :id="id" class="relative opacity-90 flex items-center pt-12 pb-2 mb-8 text-3xl text-ob-bright uppercase">
    <svg-icon v-if="icon" :icon-class="icon" class="inline-block mr-2" />
    {{ t(titleStr) }}
    <span class="brand-rule absolute bottom-0 w-24" />
  </p>
</template>

<script lang="ts">
import { defineComponent, toRefs } from 'vue'
import { useI18n } from 'vue-i18n'

export default defineComponent({
  name: 'Title',
  props: {
    title: {
      type: String,
      required: true
    },
    id: String,
    icon: String
  },
  setup(props) {
    const { t } = useI18n()
    const titleStr = toRefs(props).title
    return {
      titleStr,
      t
    }
  }
})
</script>
