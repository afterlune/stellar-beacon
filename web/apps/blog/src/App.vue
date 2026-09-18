<template>
  <div id="App-Wrapper" :class="[appWrapperClass, theme]">
    <AmbientGrid />
    <div
      id="App-Container"
      class="app-container px-3 lg:px-8"
      @keydown.meta.k.stop.prevent=""
      tabindex="-1">
      <HeaderMain />
      <template v-if="showBanner">
        <div class="app-banner future-banner" :style="headerImage" aria-hidden="true">
          <span class="future-banner__scanline" />
          <span class="future-banner__index">SB / 01</span>
          <span class="future-banner__signal">SIGNAL ONLINE</span>
        </div>
      </template>
      <div class="app-content">
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
  <div class="App-Mobile-sidebar" :class="{ 'is-open': navigatorStore.openMenu }" v-if="isMobile">
    <div id="App-Mobile-Profile" class="App-Mobile-wrapper">
      <MobileMenu />
    </div>
  </div>
  <AuroraNavigator />
  <Dia v-if="!isMobile" />
  <UserCenter />
</template>

<script lang="ts">
import { computed, defineComponent, onBeforeMount, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useCommonStore } from '@/stores/common'
import { useNavigatorStore } from '@/stores/navigator'
import HeaderMain from '@/components/Header/src/Header.vue'
import Footer from '@/components/Footer.vue'
import MobileMenu from '@/components/MobileMenu.vue'
import Dia from '@/components/Dia.vue'
import AuroraNavigator from '@/components/AuroraNavigator.vue'
import UserCenter from '@/components/UserCenter.vue'
import AmbientGrid from '@/components/AmbientGrid.vue'
import api from './api/api'
import { useSeoMeta } from '@/composables/useSeoMeta'
export default defineComponent({
  name: 'App',
  components: {
    HeaderMain,
    Footer,
    Dia,
    AuroraNavigator,
    MobileMenu,
    UserCenter,
    AmbientGrid
  },
  setup() {
    const appStore = useAppStore()
    const commonStore = useCommonStore()
    const navigatorStore = useNavigatorStore()
    const route = useRoute()
    const { setSeo } = useSeoMeta()
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
      applyRouteSeo()
    })
    watch(() => route.fullPath, applyRouteSeo)
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
    function applyRouteSeo(): void {
      if (route.path.startsWith('/articles/')) return
      setSeo({
        title: route.path === '/' ? 'Stellar Beacon · 技术与思考' : 'Stellar Beacon · ' + String(route.name || ''),
        description: '记录后端工程、系统实践与仍在发生的思考。',
        canonical: window.location.href
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
      theme: computed(() => appStore.themeConfig.theme),
      hideBanner: computed(() => route.meta.hideBanner === true),
      // The cover is a homepage treatment. Inner pages use a clear heading
      // band so the sticky navigation never sits on top of their first row.
      showBanner: computed(() => route.path === '/' && route.meta.hideBanner !== true),
      headerImage: computed(() => {
        // Keep the cover image as a quiet texture under the new editorial shell.
        return {
          backgroundImage: commonStore.headerImage !== '' ? `url(${commonStore.headerImage})` : 'none',
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          opacity: 1
        }
      }),

      isMobile: computed(() => commonStore.isMobile),
      navigatorStore,
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

#app {
  @apply relative min-w-full min-h-screen h-full;
  display: flex;
  flex-direction: column;
  font-family: var(--font-sans);
  .app-wrapper {
    @apply min-w-full pb-12;
    background-color: var(--background-primary);
    // Keep the ambient background fixed while the page scrolls.
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
    // The drawer stays mounted while closed; keep it from swallowing taps
    // meant for the page underneath it.
    pointer-events: none;

    &.is-open {
      pointer-events: auto;
    }
  }
  .App-Mobile-wrapper {
    @apply relative overflow-y-auto h-full -mr-4 pr-6 pl-4 pt-8 opacity-0;
    transition: all 0.85s cubic-bezier(0, 1.8, 1, 1.2);
    transform: translateY(-20%);
    width: 280px;
  }
}

/* Banner layers and blend effects. */
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

/* Blend the aurora over the cover image. */
.app-banner-aurora {
  z-index: 3;
  background-image: var(--hero-aurora);
  mix-blend-mode: screen;
}

/* Noise reduces visible banding in the gradient. */
.app-banner-grain {
  z-index: 4;
  background-image: var(--hero-grain);
  opacity: 0.05;
  mix-blend-mode: overlay;
}

/* Preserve contrast for navigation and page titles. */
.app-banner-veil {
  z-index: 5;
  background-image: var(--hero-veil);
}

/* Star field. Sits above the veil so the veil cannot dim it. */
.app-banner-stars {
  z-index: 6;
  background-image: var(--hero-star);
  background-size: 420px 420px;
  background-repeat: repeat;
  opacity: 0.9;
}

/* The Big Dipper. Alkaid, at the end of the handle, is the lit star.
   Only the band above the content card is free, so the asterism is sized to
   finish inside it rather than run under the card or off the top edge.
   The radial mask keeps it from ending on a hard edge. */
.app-banner-dipper {
  z-index: 7;
  background-image: var(--dipper);
  background-repeat: no-repeat;
  background-position: right 3% top 104px;
  background-size: min(22vw, 196px) auto;
  opacity: 0.94;
  -webkit-mask-image: radial-gradient(64% 90% at 72% 74%, #000 0%, rgba(0, 0, 0, 0) 100%);
  mask-image: radial-gradient(64% 90% at 72% 74%, #000 0%, rgba(0, 0, 0, 0) 100%);
}

/* Brand divider below the banner. */
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

/* Fade the banner light into the page content. */
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
  /* The short banner has no room for the full asterism next to the title. */
  .app-banner-dipper {
    background-size: min(34vw, 132px) auto;
    background-position: right 3% top 96px;
    opacity: 0.75;
  }
}
#footer {
  flex: none;
  margin-top: auto;
}
</style>
