<template>
  <template v-if="loggedIn">
    <div v-if="!isMobile" class="space-switcher" data-dia="space-switcher" aria-label="空间切换">
      <button type="button" :class="{ active: !isPrivate }" data-space="public" @click="goPublic">
        {{ cn ? '公共空间' : 'Public' }}
      </button>
      <button type="button" :class="{ active: isPrivate }" data-space="private" @click="goPrivate">
        {{ cn ? '我的空间' : 'My space' }}
      </button>
    </div>
    <Dropdown v-else @command="handleCommand">
      <span class="space-switcher-mobile" data-dia="space-switcher-mobile">
        {{ cn ? '空间' : 'Space' }}
      </span>
      <DropdownMenu>
        <DropdownItem name="public">{{ cn ? '公共空间' : 'Public space' }}</DropdownItem>
        <DropdownItem name="private">{{ cn ? '我的空间' : 'My space' }}</DropdownItem>
        <DropdownItem name="profile" :disabled="!publicHandle">{{ cn ? '我的公开主页' : 'My public profile' }}</DropdownItem>
      </DropdownMenu>
    </Dropdown>
  </template>
</template>

<script lang="ts">
import { computed, defineComponent, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { Dropdown, DropdownItem, DropdownMenu } from '@/components/Dropdown'
import { useCommonStore } from '@/stores/common'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'SpaceSwitcher',
  components: { Dropdown, DropdownItem, DropdownMenu },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const commonStore = useCommonStore()
    const { locale } = useI18n()
    const loggedIn = computed(() => Boolean(userStore.userInfo))
    const publicHandle = computed(() => String(userStore.userInfo?.handle || '').trim())
    const isPrivate = computed(() => route.path === '/studio' || route.path.startsWith('/studio/'))
    const cn = computed(() => String(locale.value || 'cn') !== 'en')

    const goPublic = () => void router.push('/')
    const goPrivate = () => void router.push('/studio/dashboard')
    const goProfile = () => {
      if (publicHandle.value) void router.push(`/u/${publicHandle.value}`)
    }
    const handleCommand = (command: string) => {
      if (command === 'public') goPublic()
      else if (command === 'private') goPrivate()
      else if (command === 'profile') goProfile()
    }

    return {
      loggedIn,
      isMobile: toRef(commonStore.$state, 'isMobile'),
      publicHandle,
      isPrivate,
      cn,
      goPublic,
      goPrivate,
      handleCommand
    }
  }
})
</script>

<style lang="scss" scoped>
.space-switcher {
  display: inline-flex;
  align-items: center;
  height: 34px;
  padding: 2px;
  border: 1px solid color-mix(in srgb, var(--header-fg) 24%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--header-fg) 7%, transparent);
}

.space-switcher button {
  height: 28px;
  padding: 0 10px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--header-fg);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  opacity: .72;
}

.space-switcher button.active {
  background: color-mix(in srgb, var(--header-fg) 16%, transparent);
  opacity: 1;
  font-weight: 700;
}

.space-switcher-mobile {
  display: flex;
  align-items: center;
  height: 32px;
  padding: 0 8px;
  border-radius: var(--radius-md);
  color: var(--header-fg);
  font-size: 12px;
  cursor: pointer;
}
</style>
