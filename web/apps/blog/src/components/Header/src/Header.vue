<template>
  <div class="header-container" :class="{ 'is-stuck': stuck }">
    <header class="site-header">
      <Logo />
      <Navigation />
      <Controls />
    </header>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Logo, Navigation, Controls } from '../index'

export default defineComponent({
  name: 'Header',
  components: {
    Logo,
    Navigation,
    Controls
  },
  props: {
    msg: String
  },
  setup() {
    // Use a solid surface after the banner scrolls away, or immediately on routes without a banner.
    const route = useRoute()
    const noBanner = computed(() => route.meta.hideBanner === true)
    const stuck = ref(false)
    const onScroll = () => {
      if (noBanner.value) {
        stuck.value = true
        return
      }
      const band = document.querySelector('.app-banner')
      const limit = band ? band.clientHeight - 72 : 0
      stuck.value = window.scrollY > limit
    }
    onMounted(() => {
      onScroll()
      window.addEventListener('scroll', onScroll, { passive: true })
    })
    onUnmounted(() => {
      window.removeEventListener('scroll', onScroll)
    })
    watch(noBanner, () => onScroll(), { immediate: true })
    return { stuck }
  }
})
</script>

<style lang="scss" scoped>
.header-container {
  position: sticky;
  top: 0;
  z-index: 50;
  /* White while floating over the brand band; theme ink once the bar has its
     own surface. Nav, logo and controls all read this. */
  --header-fg: #fff;

  &.is-stuck {
    --header-fg: var(--text-bright);
  }

  .site-header {
    max-width: var(--max-width);
    @apply relative flex z-50 my-0 mx-auto py-4;
    transition: background-color 250ms ease, border-color 250ms ease, backdrop-filter 250ms ease;
    border-bottom: 1px solid transparent;
  }

  &.is-stuck .site-header {
    background-color: var(--surface-1);
    border-bottom-color: var(--border-hairline);
  }

  @supports (backdrop-filter: blur(1px)) {
    &.is-stuck .site-header {
      background-color: color-mix(in srgb, var(--surface-1) 82%, transparent);
      backdrop-filter: blur(12px);
    }
  }
}
</style>
