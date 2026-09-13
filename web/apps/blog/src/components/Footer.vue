<template>
  <div id="footer" class="brand-gradient relative w-full pt-1">
    <span class="bg-ob-deep-900 flex justify-center">
      <div
        class="bg-ob-deep-900 rounded-lg max-w-10/12 lg:max-w-screen-2xl text-sm text-ob-normal w-full py-6 px-6 flex flex-col lg:flex-row items-center justify-between gap-6 h-auto lg:h-36 mx-auto">
        <div class="flex items-center gap-3 mx-auto lg:mx-0">
          <BrandMark :size="36" />
          <span class="flex flex-col">
            <b class="footer-name">{{ websiteConfig.name }}</b>
            <span class="footer-sub">{{ websiteConfig.englishName || 'BLOG' }}</span>
          </span>
        </div>

        <ul class="flex flex-col gap-2 text-center lg:text-right mx-auto lg:mx-0">
          <li>
            Copyright © 2022 - {{ currentYear }}
            <b class="font-extrabold">{{ websiteConfig.author }}</b>
          </li>
          <li v-if="websiteConfig.beianNumber != ''">
            <a href="https://beian.miit.gov.cn/" target="_blank">
              <b class="font-extrabold border-b-2 border-ob hover:text-ob"> {{ websiteConfig.beianNumber }} </b>
            </a>
          </li>
        </ul>
      </div>
    </span>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'
import { useAppStore } from '@/stores/app'
import { useI18n } from 'vue-i18n'
import BrandMark from '@/components/BrandMark.vue'

export default defineComponent({
  name: 'Footer',
  components: { BrandMark },
  setup() {
    const appStore = useAppStore()
    const { t } = useI18n()
    return {
      avatarClass: computed(() => {
        return {
          'footer-avatar': true,
          [appStore.themeConfig.profile_shape]: true
        }
      }),

      currentYear: computed(() => new Date().getUTCFullYear()),
      websiteConfig: computed(() => appStore.websiteConfig),
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.footer-name {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.footer-sub {
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  opacity: 0.62;
}
</style>
