<template>
  <div class="flex-shrink-0">
    <div class="rounded-full overflow-hidden w-9 xl:w-10">
      <template v-if="url"> <img class="avatar-img" :src="url" alt="" @error="handleImageError" /></template>
      <template v-else><img class="avatar-img" :src="default" alt="" /></template>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, toRefs } from 'vue'
import avatarPlaceholder from '@/assets/avatar-placeholder.svg'

export default defineComponent({
  name: 'Avatar',
  props: ['url'],
  setup(props) {
    const handleImageError = (event: Event) => {
      const image = event.target as HTMLImageElement
      if (image.dataset.fallbackApplied === 'true') return
      image.dataset.fallbackApplied = 'true'
      image.src = avatarPlaceholder
    }

    return {
      url: toRefs(props).url,
      default: avatarPlaceholder,
      handleImageError
    }
  }
})
</script>
<style lang="scss" scoped>
.avatar-img {
  transition-property: transform;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  transition-duration: 800ms;
  transform: rotate(-360deg);
}
.avatar-img:hover {
  transform: rotate(360deg);
}
</style>
