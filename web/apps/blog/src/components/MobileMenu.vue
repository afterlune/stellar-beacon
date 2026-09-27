<template>
  <div class="mobile-menu-brand flex flex-col justify-center items-center">
    <BrandMark :size="76" />
    <h2 class="text-center pt-4 text-3xl font-semibold text-ob-bright">{{ websiteConfig.name || 'Stellar Beacon' }}</h2>
    <span class="brand-rule w-14 mt-2" />
    <p class="pt-5 px-5 w-full text-sm text-center text-ob-dim">{{ t('platform.description') }}</p>
    <ul class="grid grid-cols-3 pt-4 w-full px-2 text-lg">
      <li class="col-span-1 text-center">
        <span class="text-ob-bright">{{ articleCount }}</span>
        <p class="text-base text-ob-dim">{{ t('settings.articles') }}</p>
      </li>
      <li class="col-span-1 text-center">
        <span class="text-ob-bright">{{ categoryCount }}</span>
        <p class="text-base text-ob-dim">{{ t('settings.categories') }}</p>
      </li>
      <li class="col-span-1 text-center">
        <span class="text-ob-bright">{{ tagCount }}</span>
        <p class="text-base text-ob-dim">{{ t('settings.tags') }}</p>
      </li>
    </ul>
  </div>
  <ul class="flex flex-col justify-center items-center mt-8 w-full list-none text-ob-bright">
    <li class="pb-2 cursor-pointer" v-for="route in routes" :key="route.path">
      <button type="button" class="mobile-menu-route text-sm block px-1.5 py-0.5 rounded-md relative uppercase" @click="pushPage(route.path)">
        <span class="relative z-50" v-if="locale === 'cn' && route.i18n.cn">
          {{ route.i18n.cn }}
        </span>
        <span class="relative z-50" v-else-if="locale === 'en' && route.i18n.en">
          {{ route.i18n.en }}
        </span>
        <span class="relative z-50" v-else>{{ route.name }}</span>
      </button>
    </li>
  </ul>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'
import { useAppStore } from '@/stores/app'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useNavigatorStore } from '@/stores/navigator'
import BrandMark from '@/components/BrandMark.vue'
import config from '@/config/config'

export default defineComponent({
  name: 'ObMobileMenu',
  components: { BrandMark },
  setup() {
    const appStore = useAppStore()
    const router = useRouter()
    const navigatorStore = useNavigatorStore()
    const { t, locale } = useI18n()
    const routes = config.routes as Array<{ name: string; path: string; i18n: { cn: string; en: string } }>
    const pushPage = (path: string): void => {
      console.log(path)
      if (!path) return
      navigatorStore.toggleMobileMenu()
      navigatorStore.setOpenNavigator(false)
      if (path.match(/(http:\/\/|https:\/\/)((\w|=|\?|\.|\/|&|-)+)/g)) {
        window.location.href = path
      } else {
        router.push({
          path: path
        })
      }
    }
    return {
      routes,
      pushPage,
      locale,
      websiteConfig: computed(() => appStore.websiteConfig),
      articleCount: computed(() => appStore.articleCount),
      talkCount: computed(() => appStore.talkCount),
      categoryCount: computed(() => appStore.categoryCount),
      tagCount: computed(() => appStore.tagCount),
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.mobile-menu-route {
  min-height: 44px;
  padding: 8px 12px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: center;
  cursor: pointer;
}

.mobile-menu-route:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 3px;
}
</style>
