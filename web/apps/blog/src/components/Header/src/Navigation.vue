<template>
  <nav class="items-center flex-1 hidden lg:flex">
    <ul class="nav-list flex flex-row list-none px-6">
      <li
        class="not-italic font-medium text-xs h-full relative flex flex-col items-center justify-center cursor-pointer text-center py-4 px-2"
        v-for="route in routes"
        :key="route.path">
        <div
          class="nav-link text-sm block px-1.5 py-0.5 rounded-md relative uppercase cursor-pointer"
          :class="{ 'nav-link-active': isActive(route.path) }"
          @click="pushPage(route.path)"
          v-if="route.children && route.children.length === 0"
          :data-menu="route.name"
          :aria-current="isActive(route.path) ? 'page' : undefined">
          <span class="relative z-50" v-if="$i18n.locale === 'cn' && route.i18n.cn">
            {{ route.i18n.cn }}
          </span>
          <span class="relative z-50" v-else-if="$i18n.locale === 'en' && route.i18n.en">
            {{ route.i18n.en }}
          </span>
          <span class="relative z-50" v-else>{{ route.name }}</span>
        </div>
        <Dropdown
          @command="pushPage"
          hover
          v-else
          class="nav-link text-sm block px-1.5 py-0.5 rounded-md relative uppercase">
          <span class="relative z-50" v-if="$i18n.locale === 'cn' && route.i18n.cn">
            {{ route.i18n.cn }}
          </span>
          <span class="relative z-50" v-else-if="$i18n.locale === 'en' && route.i18n.en">
            {{ route.i18n.en }}
          </span>
          <span class="relative z-50" v-else>{{ route.name }}</span>
          <DropdownMenu>
            <DropdownItem v-for="sub in route.children" :key="sub.path" :name="sub.path">
              <span class="relative z-50" v-if="$i18n.locale === 'cn' && sub.i18n.cn">
                {{ sub.i18n.cn }}
              </span>
              <span class="relative z-50" v-else-if="$i18n.locale === 'en' && sub.i18n.en">
                {{ sub.i18n.en }}
              </span>
              <span class="relative z-50" v-else>{{ sub.name }}</span>
            </DropdownItem>
          </DropdownMenu>
        </Dropdown>
      </li>
      <li
        class="not-italic font-medium text-xs h-full relative flex flex-col items-center justify-center cursor-pointer text-center py-4 px-2"
        data-menu="PhotoAlbums">
        <Dropdown hover class="nav-link text-sm block px-1.5 py-0.5 rounded-md relative uppercase">
          <span class="relative z-50" v-if="$i18n.locale === 'cn'"> 相册 </span>
          <span class="relative z-50" v-else-if="$i18n.locale === 'en'"> PhotoAlbums </span>
          <DropdownMenu>
            <template v-for="item in albums" :key="item.id">
              <DropdownItem @click="pushPage(`/photos/${item.id}`)" :name="item.albumName">
                <span class="relative z-50">{{ item.albumName }}</span>
              </DropdownItem>
            </template>
          </DropdownMenu>
        </Dropdown>
      </li>
    </ul>
  </nav>
</template>

<script lang="ts">
// @ts-nocheck
import { defineComponent, onMounted, reactive, toRef, toRefs } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Dropdown, DropdownMenu, DropdownItem } from '@/components/Dropdown'
import { isExternal } from '@/utils/validate'
import config from '@/config/config'
import api from '@/api/api'

export default defineComponent({
  name: 'Navigation',
  components: { Dropdown, DropdownMenu, DropdownItem },
  setup() {
    const { t, te } = useI18n()
    const router = useRouter()
    const currentRoute = useRoute()
    // There was no way to tell which page you were on; the nav only reacted to
    // the pointer.
    const isActive = (path: string): boolean => {
      if (!path) return false
      if (path === '/') return currentRoute.path === '/'
      return currentRoute.path === path || currentRoute.path.startsWith(path + '/')
    }
    const pushPage = (path: string): void => {
      if (!path) return
      if (isExternal(path)) {
        window.location.href = path
      } else {
        router.push({
          path: path
        })
      }
    }
    const reactiveData = reactive({
      albums: [] as any
    })
    onMounted(() => {
      api.getAlbums().then(({ data }) => {
        reactiveData.albums = data.data
      })
    })
    const openPhotoAlbum = (id: any): void => {
      router.push('/photos/' + id)
    }
    return {
      ...toRefs(reactiveData),
      routes: config.routes,
      pushPage,
      isActive,
      openPhotoAlbum,
      te,
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.nav-list {
  color: var(--header-fg);
  /* The bar floats on the aurora; the ink carries its own shadow. */
  text-shadow: 0 1px 10px rgba(0, 0, 0, 0.55);
}

.nav-link {
  font-size: 13px;
  letter-spacing: 0.04em;

  &:hover {
    color: var(--header-fg);
    &:before {
      opacity: 1;
    }
  }
  &:before {
    @apply absolute rounded-lg opacity-0 transition z-40;
    content: '';
    top: -4px;
    left: -4px;
    width: calc(100% + 8px);
    height: calc(100% + 8px);
    // Was filled with the card colour, which read as a dirty block over the
    // brand band. A translucent wash works on any backdrop.
    background-color: var(--surface-hover);
  }
}

/* The nav had no selected state at all. */
.nav-link-active {
  color: var(--header-fg);
  font-weight: 600;

  &:before {
    opacity: 1;
  }
  &:after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: -6px;
    height: 2px;
    border-radius: var(--radius-pill);
    background-image: var(--brand-gradient);
  }
}
</style>
