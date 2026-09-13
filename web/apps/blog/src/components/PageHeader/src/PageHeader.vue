<template>
  <div class="post-header">
    <Breadcrumb v-if="current" :current="current" />
    <h1 v-if="!loading" class="post-title text-white uppercase">{{ title }}</h1>
    <ob-skeleton
      v-else
      class="post-title text-white uppercase"
      :width="skeletonWidth"
      :height="skeletonHeight" />
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import Breadcrumb from '@/components/Breadcrumb.vue'

// Shared heading, breadcrumb, and loading skeleton for inner pages.
export default defineComponent({
  name: 'PageHeader',
  components: { Breadcrumb },
  props: {
    title: {
      type: String,
      default: ''
    },
    current: {
      type: String,
      default: ''
    },
    loading: {
      type: Boolean,
      default: false
    },
    skeletonWidth: {
      type: String,
      default: '30%'
    },
    skeletonHeight: {
      type: String,
      default: 'clamp(1.75rem, calc(1.2rem + 2.2vw), 2.75rem)'
    }
  }
})
</script>
