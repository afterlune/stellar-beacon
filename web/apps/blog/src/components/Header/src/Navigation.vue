<template>
  <nav
    class="site-navigation items-center flex-1"
    :class="{ 'is-english': locale === 'en' }"
    :aria-label="locale === 'cn' ? '主导航' : 'Primary navigation'">
    <ul class="nav-list">
      <li
        v-for="entry in entries"
        :key="entry.key"
        class="nav-item"
        :data-menu="entry.route.name">
        <router-link
          v-if="entry.kind === 'route' && !hasChildren(entry.route)"
          class="nav-link"
          :class="{ 'nav-link-active': isEntryActive(entry) }"
          :to="entry.route.path"
          :data-menu="entry.route.name"
          :title="routeLabel(entry.route)"
          :aria-label="routeLabel(entry.route)"
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
          <button type="button" aria-haspopup="true" :title="routeLabel(entry.route)" :aria-label="routeLabel(entry.route)">
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
    </ul>
  </nav>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'
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

    const entries = computed<NavEntry[]>(() => routes.map((route, index) => ({ key: route.path, index, kind: 'route' as const, route })))

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

    return {
      locale,
      entries,
      routeLabel,
      hasChildren,
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
  display: none;
  min-width: 0;
  container-name: primary-navigation;
  container-type: inline-size;
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

.nav-item {
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

.nav-label {
  position: relative;
  z-index: 1;
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

@media (min-width: 1280px) {
  .site-navigation { display: flex; }
  .nav-label { display: none; }
  .nav-index { display: none; }
  .nav-link { gap: 0; padding-inline: 7px; }
}
</style>
