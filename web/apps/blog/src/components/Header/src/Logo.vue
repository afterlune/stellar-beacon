<template>
  <div class="flex items-start self-stretch relative" @click="handleClick">
    <div class="logo-mark flex items-center gap-3 relative py-4 z-10 cursor-pointer">
      <BrandMark :size="38" />
      <span class="logo-copy flex flex-col justify-center">
        <span class="logo-name text-2xl">
          {{ websiteConfig.name || 'LOADING' }}
        </span>
        <span class="logo-sub">
          {{ websiteConfig.englishName || 'BLOG' }}
        </span>
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { useAppStore } from '@/stores/app'
import { computed } from '@vue/reactivity'
import { defineComponent } from 'vue'
import { useRouter } from 'vue-router'
import { useCommonStore } from '@/stores/common'
import { useNavigatorStore } from '@/stores/navigator'
import BrandMark from '@/components/BrandMark.vue'

export default defineComponent({
  name: 'Logo',
  components: { BrandMark },
  setup() {
    const appStore = useAppStore()
    const commonStore = useCommonStore()
    const navigatorStore = useNavigatorStore()
    const router = useRouter()
    const handleClick = () => {
      router.push({ path: '/' })
      if (commonStore.isMobile && navigatorStore.openMenu === true) {
        navigatorStore.toggleMobileMenu()
      }
    }
    return {
      websiteConfig: computed(() => {
        return appStore.websiteConfig
      }),
      handleClick
    }
  }
})
</script>

<style lang="scss" scoped>
.logo-mark {
  color: var(--header-fg);
  transition: color 250ms ease;
}

.logo-copy {
  min-width: 0;
}

.logo-name {
  font-family: var(--font-display);
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: 0.01em;
  /* 浮在 hero 上时给文字一点暗底，压在浅色封面上也读得清。 */
  text-shadow: 0 2px 12px rgba(4, 5, 12, 0.5);
}

.logo-sub {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  opacity: 0.72;
}

/* 顶栏吸顶后不再压在 hero 上，去掉文字阴影。 */
:global(.is-stuck) .logo-name {
  text-shadow: none;
}

/* 窄屏给导航和控件让位。 */
@media (max-width: 1023px) {
  .logo-mark {
    gap: 8px;
  }
  .logo-mark :deep(.brand-mark) {
    width: 32px;
    height: 32px;
    border-radius: 8px;
  }
  .logo-name {
    font-size: 18px;
  }
  .logo-sub {
    font-size: 8.5px;
    letter-spacing: 0.16em;
  }
}
</style>
