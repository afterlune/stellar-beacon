<template>
  <nav
    ref="navRoot"
    class="site-navigation items-center flex-1 hidden lg:flex"
    :class="{ 'is-layout-ready': layoutReady }"
    :aria-label="locale === 'cn' ? '主导航' : 'Primary navigation'">
    <ul ref="navList" class="nav-list">
      <li
        v-for="entry in visibleEntries"
        :key="entry.key"
        class="nav-item"
        :data-menu="entry.route.name">
        <router-link
          v-if="entry.kind === 'route' && !hasChildren(entry.route)"
          class="nav-link"
          :class="{ 'nav-link-active': isEntryActive(entry) }"
          :to="entry.route.path"
          :data-menu="entry.route.name"
          :aria-current="isEntryActive(entry) ? 'page' : undefined">
          <span class="nav-index">{{ String(entry.index + 1).padStart(2, '0') }}</span>
          <svg-icon class="nav-icon" :icon-class="iconFor(entry.route.name)" />
          <span class="nav-label">{{ routeLabel(entry.route) }}</span>
        </router-link>
        <Dropdown
          v-else-if="entry.kind === 'route'"
          @command="pushPage"
          class="nav-link"
          :class="{ 'nav-link-active': isEntryActive(entry) }">
          <button type="button" aria-haspopup="true">
            <span class="nav-index">{{ String(entry.index + 1).padStart(2, '0') }}</span>
            <svg-icon class="nav-icon" :icon-class="iconFor(entry.route.name)" />
            <span class="nav-label">{{ routeLabel(entry.route) }}</span>
          </button>
          <DropdownMenu>
            <DropdownItem v-for="sub in entry.route.children" :key="sub.path" :name="sub.path">
              {{ routeLabel(sub) }}
            </DropdownItem>
          </DropdownMenu>
        </Dropdown>
      </li>

      <li v-if="overflowEntries.length" ref="moreRoot" class="nav-more-item">
        <button
          ref="moreTrigger"
          type="button"
          class="nav-link nav-more-trigger"
          :class="{ 'nav-link-active': hasActiveOverflowEntry }"
          :aria-expanded="moreOpen"
          aria-controls="header-nav-overflow"
          @click.stop="moreOpen = !moreOpen">
          <span class="nav-more-label">{{ moreLabel }}</span>
          <span class="nav-more-chevron" aria-hidden="true" />
        </button>
        <div v-if="moreOpen" id="header-nav-overflow" class="nav-overflow-panel">
          <ul class="nav-overflow-list">
            <template v-for="entry in overflowEntries" :key="entry.key">
              <li v-if="entry.kind === 'route' && hasChildren(entry.route)" class="nav-overflow-group">
                <span class="nav-overflow-heading">{{ routeLabel(entry.route) }}</span>
                <template v-for="sub in entry.route.children" :key="sub.path">
                  <a
                    v-if="isExternal(sub.path)"
                    class="nav-overflow-link"
                    :href="sub.path"
                    @click="closeMore">{{ routeLabel(sub) }}</a>
                  <router-link
                    v-else
                    class="nav-overflow-link"
                    :class="{ 'nav-overflow-link-active': isActive(sub.path) }"
                    :to="sub.path"
                    @click="closeMore">{{ routeLabel(sub) }}</router-link>
                </template>
              </li>
              <li v-else-if="entry.kind === 'route'" class="nav-overflow-item">
                <a
                  v-if="isExternal(entry.route.path)"
                  class="nav-overflow-link"
                  :href="entry.route.path"
                  @click="closeMore">{{ routeLabel(entry.route) }}</a>
                <router-link
                  v-else
                  class="nav-overflow-link"
                  :class="{ 'nav-overflow-link-active': isEntryActive(entry) }"
                  :to="entry.route.path"
                  :aria-current="isEntryActive(entry) ? 'page' : undefined"
                  @click="closeMore">{{ routeLabel(entry.route) }}</router-link>
              </li>
            </template>
          </ul>
        </div>
      </li>
    </ul>

    <div ref="measureRoot" class="nav-measure" aria-hidden="true">
      <span
        v-for="entry in entries"
        :key="entry.key"
        class="nav-link nav-measure-item"
        :class="{ 'nav-link-active': isEntryActive(entry) }"
        :data-entry-index="entry.index">
        <span class="nav-index">{{ String(entry.index + 1).padStart(2, '0') }}</span>
        <svg-icon class="nav-icon" :icon-class="iconFor(entry.route.name)" />
        <span class="nav-label">{{ routeLabel(entry) }}</span>
      </span>
      <span ref="measureMore" class="nav-link nav-more-trigger nav-measure-more">
        <span class="nav-more-label">{{ moreLabel }}</span>
        <span class="nav-more-chevron" aria-hidden="true" />
      </span>
    </div>
  </nav>
</template>

<script lang="ts">
import { computed, defineComponent, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Dropdown, DropdownMenu, DropdownItem } from '@/components/Dropdown'
import { isExternal } from '@/utils/validate'
import config from '@/config/config'

type NavEntry = {
  key: string
  index: number
  kind: 'route'
  route: any
}

export default defineComponent({
  name: 'Navigation',
  components: { Dropdown, DropdownMenu, DropdownItem },
  setup() {
    const { locale } = useI18n()
    const router = useRouter()
    const currentRoute = useRoute()
    const routes = config.routes as any[]
    const navRoot = ref<HTMLElement | null>(null)
    const navList = ref<HTMLElement | null>(null)
    const measureRoot = ref<HTMLElement | null>(null)
    const measureMore = ref<HTMLElement | null>(null)
    const moreRoot = ref<HTMLElement | null>(null)
    const moreTrigger = ref<HTMLButtonElement | null>(null)
    const visibleCount = ref(0)
    const layoutReady = ref(false)
    const moreOpen = ref(false)
    let resizeObserver: ResizeObserver | undefined
    let measureQueued = false

    const entries = computed<NavEntry[]>(() => routes.map((route, index) => ({ key: route.path, index, kind: 'route' as const, route })))
    const visibleEntries = computed(() => entries.value.slice(0, visibleCount.value))
    const overflowEntries = computed(() => entries.value.slice(visibleCount.value))
    const moreLabel = computed(() => locale.value === 'cn' ? '更多' : 'More')

    const routeLabel = (routeOrEntry: any): string => {
      const route = routeOrEntry?.route || routeOrEntry
      if (locale.value === 'cn' && route?.i18n?.cn) return route.i18n.cn
      if (locale.value === 'en' && route?.i18n?.en) return route.i18n.en
      return route?.name || ''
    }
    const hasChildren = (route: any): boolean => Array.isArray(route?.children) && route.children.length > 0
    const isActive = (path: string): boolean => {
      if (!path) return false
      if (path === '/') return currentRoute.path === '/'
      return currentRoute.path === path || currentRoute.path.startsWith(path + '/')
    }
    const isEntryActive = (entry: NavEntry): boolean => {
      return isActive(entry.route.path) || (entry.route.children || []).some((sub: any) => isActive(sub.path))
    }
    const hasActiveOverflowEntry = computed(() => overflowEntries.value.some(isEntryActive))
    const pushPage = (path: string): void => {
      if (!path) return
      if (isExternal(path)) {
        window.location.href = path
      } else {
        void router.push({ path })
      }
    }
    const iconFor = (name: string): string => ({
      Home: 'nav-home',
      Topics: 'hot',
      Categories: 'category',
      Series: 'toc',
      Collections: 'folder',
      Talks: 'message',
      Authors: 'people',
      Following: 'people',
      Archives: 'date',
      Tags: 'tag'
    }[name] || 'article')

    const measureNavigation = (): void => {
      const root = navRoot.value
      const list = navList.value
      const ruler = measureRoot.value
      if (!root || !list || !ruler || root.clientWidth <= 0) return

      const widths = entries.value.map((entry) => {
        const item = ruler.querySelector<HTMLElement>(`[data-entry-index="${entry.index}"]`)
        return item?.getBoundingClientRect().width || 0
      })
      const moreWidth = measureMore.value?.getBoundingClientRect().width || 0
      const columnGap = Number.parseFloat(getComputedStyle(list).columnGap) || 0
      const available = root.clientWidth
      let count = 0
      let used = 0
      while (count < widths.length) {
        const next = used + (count > 0 ? columnGap : 0) + widths[count]
        if (next > available) break
        used = next
        count += 1
      }
      if (count < widths.length) {
        while (count > 0 && used + columnGap + moreWidth > available) {
          count -= 1
          used = widths.slice(0, count).reduce((sum, width) => sum + width, 0) + Math.max(0, count - 1) * columnGap
        }
      }
      visibleCount.value = count
      if (count === widths.length) moreOpen.value = false
      layoutReady.value = true
    }
    const scheduleMeasure = (): void => {
      if (measureQueued) return
      measureQueued = true
      void nextTick(() => {
        measureQueued = false
        measureNavigation()
      })
    }
    const closeMore = (): void => {
      moreOpen.value = false
    }
    const handleDocumentPointerDown = (event: PointerEvent): void => {
      if (moreOpen.value && !moreRoot.value?.contains(event.target as Node)) closeMore()
    }
    const handleDocumentKeydown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape' && moreOpen.value) {
        event.preventDefault()
        closeMore()
        void nextTick(() => moreTrigger.value?.focus())
      }
    }

    watch(locale, scheduleMeasure)
    watch(() => currentRoute.fullPath, closeMore)
    onMounted(() => {
      void nextTick(measureNavigation)
      if (typeof ResizeObserver !== 'undefined' && navRoot.value) {
        resizeObserver = new ResizeObserver(scheduleMeasure)
        resizeObserver.observe(navRoot.value)
      }
      window.addEventListener('resize', scheduleMeasure, { passive: true })
      document.addEventListener('pointerdown', handleDocumentPointerDown)
      document.addEventListener('keydown', handleDocumentKeydown)
      document.fonts?.ready.then(scheduleMeasure)
    })
    onUnmounted(() => {
      resizeObserver?.disconnect()
      window.removeEventListener('resize', scheduleMeasure)
      document.removeEventListener('pointerdown', handleDocumentPointerDown)
      document.removeEventListener('keydown', handleDocumentKeydown)
    })

    return {
      locale,
      currentRoute,
      entries,
      visibleEntries,
      overflowEntries,
      visibleCount,
      layoutReady,
      moreOpen,
      moreLabel,
      hasActiveOverflowEntry,
      navRoot,
      navList,
      measureRoot,
      measureMore,
      moreRoot,
      moreTrigger,
      routeLabel,
      hasChildren,
      closeMore,
      isActive,
      isEntryActive,
      pushPage,
      iconFor,
      isExternal
    }
  }
})
</script>

<style lang="scss" scoped>
.site-navigation {
  min-width: 0;
  visibility: hidden;
}

.site-navigation.is-layout-ready {
  visibility: visible;
}

.nav-list {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-width: 0;
  margin: 0;
  padding: 0;
  overflow: visible;
  color: var(--header-fg);
  list-style: none;
  text-shadow: 0 1px 10px rgba(0, 0, 0, 0.55);
}

.nav-item,
.nav-more-item {
  position: relative;
  flex: 0 0 auto;
  padding: 0;
}

.nav-link {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 6px 10px;
  border: 1px solid color-mix(in srgb, var(--header-fg) 14%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--header-fg) 6%, transparent);
  color: inherit;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  line-height: 1.2;
  text-decoration: none;
  white-space: nowrap;
  transition: background 180ms ease, border-color 180ms ease, color 180ms ease, box-shadow 180ms ease;
  backdrop-filter: blur(12px);

  &:hover {
    color: var(--header-fg);
    background: color-mix(in srgb, var(--header-fg) 11%, transparent);
  }

  &::before {
    position: absolute;
    inset: 0;
    z-index: 0;
    border-radius: inherit;
    background: color-mix(in srgb, var(--header-fg) 9%, transparent);
    content: '';
    opacity: 0;
    pointer-events: none;
    transition: opacity 180ms ease;
  }

  &:hover::before {
    opacity: 1;
  }

  &:focus-visible {
    outline: 2px solid var(--header-fg);
    outline-offset: 3px;
  }
}

.nav-link-active {
  border-color: color-mix(in srgb, var(--accent) 68%, transparent);
  background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 20%, transparent), color-mix(in srgb, var(--header-fg) 8%, transparent));
  color: var(--header-fg);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--accent) 18%, transparent), 0 0 20px color-mix(in srgb, var(--accent) 20%, transparent);

  &::after {
    position: absolute;
    right: 12px;
    bottom: 1px;
    left: 12px;
    height: 1px;
    border-radius: 999px;
    background: linear-gradient(90deg, transparent, var(--accent), transparent);
    content: '';
  }
}

.nav-index {
  position: relative;
  z-index: 1;
  color: color-mix(in srgb, var(--header-fg) 52%, transparent);
  font-size: 8px;
  letter-spacing: 0.08em;
}

.nav-icon {
  position: relative;
  z-index: 1;
  width: 14px;
  height: 14px;
  color: currentColor;
}

.nav-label,
.nav-more-label {
  position: relative;
  z-index: 1;
}

.nav-more-trigger {
  appearance: none;
  cursor: pointer;
}

.nav-more-chevron {
  position: relative;
  z-index: 1;
  width: 7px;
  height: 7px;
  margin-top: -3px;
  border-right: 1px solid currentColor;
  border-bottom: 1px solid currentColor;
  transform: rotate(45deg);
}

.ob-dropdown button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  white-space: nowrap;
}

.nav-overflow-panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  z-index: 90;
  width: max-content;
  min-width: 220px;
  max-width: min(320px, calc(100vw - 32px));
  max-height: min(70vh, 480px);
  overflow: auto;
  padding: 8px;
  border: 1px solid var(--border-hairline);
  border-radius: 12px;
  background: var(--surface-solid);
  box-shadow: 0 16px 42px rgba(0, 0, 0, 0.24);
  color: var(--text-bright);
  text-shadow: none;
}

.nav-overflow-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.nav-overflow-group + .nav-overflow-item,
.nav-overflow-item + .nav-overflow-group,
.nav-overflow-group + .nav-overflow-group {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--border-hairline);
}

.nav-overflow-heading {
  display: block;
  padding: 7px 10px 4px;
  color: var(--text-dim);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.nav-overflow-link {
  display: block;
  padding: 8px 10px;
  border-radius: 7px;
  color: var(--text-bright);
  font-size: 13px;
  line-height: 1.35;
  text-decoration: none;

  &:hover,
  &:focus-visible {
    outline: none;
    background: var(--surface-hover);
    color: var(--text-bright);
  }
}

.nav-overflow-link-active {
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  color: var(--accent);
}

.nav-overflow-empty {
  display: block;
  padding: 8px 10px;
  color: var(--text-dim);
  font-size: 12px;
}

.nav-measure {
  position: fixed;
  top: 0;
  left: -10000px;
  display: flex;
  gap: 6px;
  width: max-content;
  visibility: hidden;
  pointer-events: none;
  white-space: nowrap;
}

.nav-measure-item,
.nav-measure-more {
  flex: 0 0 auto;
}

@media (min-width: 1024px) and (max-width: 1535px) {
  .nav-label { display: none; }
  .nav-link { padding-inline: 9px; }
}
</style>
