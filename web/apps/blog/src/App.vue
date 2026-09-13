<template>
  <div id="App-Wrapper" :class="[appWrapperClass, theme]">
    <div
      id="App-Container"
      class="app-container px-3 lg:px-8"
      @keydown.meta.k.stop.prevent=""
      tabindex="-1">
      <HeaderMain />
      <template v-if="!hideBanner">
        <div class="app-banner app-banner-base" />
        <div class="app-banner app-banner-image" :style="headerImage" />
        <div class="app-banner app-banner-aurora" />
        <div class="app-banner app-banner-grain" />
        <div class="app-banner app-banner-veil" />
        <div class="app-banner-rule" />
        <div class="app-banner-spill" />
      </template>
      <div class="relative z-10">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide-y" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </div>
    <div id="loading-bar-wrapper" :class="loadingBarClass"></div>
  </div>
  <Footer id="footer" />
  <div class="App-Mobile-sidebar" v-if="isMobile">
    <div id="App-Mobile-Profile" class="App-Mobile-wrapper">
      <MobileMenu />
    </div>
  </div>
  <AuroraNavigator />
  <Dia v-if="!isMobile" />
  <UserCenter />
  <teleport to="head">
    <title>{{ title }}</title>
  </teleport>
</template>

<script lang="ts">
import { computed, defineComponent, onBeforeMount, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useCommonStore } from '@/stores/common'
import { useMetaStore } from '@/stores/meta'
import HeaderMain from '@/components/Header/src/Header.vue'
import Footer from '@/components/Footer.vue'
import MobileMenu from '@/components/MobileMenu.vue'
import Dia from '@/components/Dia.vue'
import AuroraNavigator from '@/components/AuroraNavigator.vue'
import UserCenter from '@/components/UserCenter.vue'
import api from './api/api'
export default defineComponent({
  name: 'App',
  components: {
    HeaderMain,
    Footer,
    Dia,
    AuroraNavigator,
    MobileMenu,
    UserCenter
  },
  setup() {
    const appStore = useAppStore()
    const commonStore = useCommonStore()
    const metaStore = useMetaStore()
    const route = useRoute()
    const MOBILE_WITH = 996
    const appWrapperClass = 'app-wrapper'
    const loadingBarClass = ref({
      'nprogress-custom-parent': false
    })
    const isMobile = computed(() => {
      return commonStore.isMobile
    })
    onBeforeMount(() => {
      initialApp()
    })
    onUnmounted(() => {
      document.removeEventListener('copy', copyEventHandler)
      window.removeEventListener('resize', resizeHander)
    })
    const initialApp = () => {
      initResizeEvent()
      intialCopy()
      initWindowOnload()
      fetchWebsiteConfig()
      appStore.initializeTheme(appStore.themeConfig.theme)
    }
    const fetchWebsiteConfig = () => {
      api.getWebsiteConfig().then(({ data }) => {
        appStore.viewCount = data.data.viewCount
        appStore.articleCount = data.data.articleCount
        appStore.talkCount = data.data.talkCount
        appStore.categoryCount = data.data.categoryCount
        appStore.tagCount = data.data.tagCount
        appStore.websiteConfig = data.data.websiteConfigDTO
      })
    }
    const copyEventHandler = (event: any) => {
      if (document.getSelection() instanceof Selection) {
        if (document.getSelection()?.toString() !== '' && event.clipboardData) {
          event.clipboardData.setData('text', document.getSelection())
          event.preventDefault()
        }
      }
    }
    const intialCopy = () => {
      document.addEventListener('copy', copyEventHandler)
    }
    const resizeHander = () => {
      const rect = document.body.getBoundingClientRect()
      const mobileState = rect.width - 1 < MOBILE_WITH
      if (isMobile.value !== mobileState) commonStore.changeMobileState(mobileState)
    }
    const initResizeEvent = () => {
      resizeHander()
      window.addEventListener('resize', resizeHander)
    }
    const initWindowOnload = () => {
      window.onload = () => {
        setTimeout(() => {
          window.scrollTo({
            top: 0
          })
        }, 10)
      }
    }
    return {
      title: metaStore.title,
      theme: computed(() => appStore.themeConfig.theme),
      hideBanner: computed(() => route.meta.hideBanner === true),
      headerImage: computed(() => {
        // The cover is the ground, the aurora is the light: the image stays well
        // under the glow so the sky always reads as sky.
        return {
          backgroundImage: commonStore.headerImage !== '' ? `url(${commonStore.headerImage})` : 'none',
          opacity: commonStore.headerImage !== '' ? 0.42 : 0
        }
      }),

      isMobile: computed(() => commonStore.isMobile),
      appWrapperClass,
      loadingBarClass
    }
  }
})
</script>

<style lang="scss">
.arrow-left > .icon,
.arrow-right > .icon {
  display: inline !important;
}
.img-error {
  display: none !important;
}
.el-drawer {
  background-color: var(--background-primary) !important;
}
.el-dialog {
  background-color: var(--background-primary) !important;
}
body {
  background: var(--background-primary-alt);
}

*:focus {
  outline: none;
}

#app {
  @apply relative min-w-full min-h-screen h-full;
  display: flex;
  flex-direction: column;
  font-family: var(--font-sans);
  .app-wrapper {
    @apply min-w-full pb-12;
    background-color: var(--background-primary);
    // The aurora's afterglow, so content never floats on flat grey. Pinned to
    // the viewport: it reads as ambient light rather than a texture that scrolls.
    background-image: var(--ambient);
    background-repeat: no-repeat;
    background-attachment: fixed;
    flex: 1 0 auto;
    min-height: 100vh;
    min-height: 100dvh;
    transition-property: transform, border-radius;
    transition-duration: 350ms;
    transition-timing-function: ease;
    transform-origin: 0 42%;
    .app-container {
      color: var(--text-normal);
      margin: 0 auto;
      max-width: var(--page-max);
    }
  }

  .header-wave {
    position: absolute;
    top: 100px;
    left: 0;
    z-index: 1;
  }

  .App-Mobile-sidebar {
    @apply fixed top-0 bottom-0 left-0;
  }
  .App-Mobile-wrapper {
    @apply relative overflow-y-auto h-full -mr-4 pr-6 pl-4 pt-8 opacity-0;
    transition: all 0.85s cubic-bezier(0, 1.8, 1, 1.2);
    transform: translateY(-20%);
    width: 280px;
  }
}

/* ============================================================
   HERO — an aurora, not a gradient bar
   A night sky with three saturated light cores composited with
   `screen`, so overlapping glows ADD light instead of averaging
   into the grey mud a 3-stop linear ramp produces. Five stacked
   layers, then a brand rule on the horizon.
   ============================================================ */
.app-banner {
  display: block;
  height: var(--banner-h);
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  z-index: 1;
  pointer-events: none;
}

.app-banner-base {
  background-color: var(--hero-base);
}

.app-banner-image {
  z-index: 2;
  background-size: cover;
  background-position: center;
  filter: saturate(1.05) contrast(0.95);
  opacity: 0;
  transition: ease-in-out opacity 300ms;
}

/* The light itself. `screen` is what makes it read as light rather than paint. */
.app-banner-aurora {
  z-index: 3;
  background-image: var(--hero-aurora);
  mix-blend-mode: screen;
}

/* Grain kills the banding that large soft gradients always produce. */
.app-banner-grain {
  z-index: 4;
  background-image: var(--hero-grain);
  opacity: 0.05;
  mix-blend-mode: overlay;
}

/* Top and bottom scrims: the nav sits high, page titles sit low. */
.app-banner-veil {
  z-index: 5;
  background-image: var(--hero-veil);
}

/* The horizon: the brand rule lives where the sky meets the page. */
.app-banner-rule {
  position: absolute;
  top: var(--banner-h);
  left: 0;
  right: 0;
  height: 3px;
  z-index: 6;
  pointer-events: none;
  background-image: var(--brand-gradient);
}

/* Light spills past the horizon and fades into the page. Without this the hero
   stops being light and becomes a painted rectangle. */
.app-banner-spill {
  position: absolute;
  top: var(--banner-h);
  left: 0;
  right: 0;
  height: 200px;
  z-index: 5;
  pointer-events: none;
  background-image: var(--hero-spill);
  filter: blur(26px);
  opacity: 0.6;
  -webkit-mask-image: linear-gradient(180deg, #000 0%, rgba(0, 0, 0, 0) 100%);
  mask-image: linear-gradient(180deg, #000 0%, rgba(0, 0, 0, 0) 100%);
}

@media (max-width: 1023px) {
  .app-banner {
    height: var(--banner-h-sm);
  }
  .app-banner-rule {
    top: var(--banner-h-sm);
  }
  .app-banner-spill {
    top: var(--banner-h-sm);
    height: 130px;
  }
}
#footer {
  flex: none;
  margin-top: auto;
}
</style>
