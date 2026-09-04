import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { listUserMenus } from '@/api/http'
import { normalizeMenus, type NormalizedMenu, visibleChildren } from '@/types'

export const useMenuStore = defineStore('admin-menu', () => {
  const menus = ref<NormalizedMenu[]>([])
  const loaded = ref(false)
  const loading = ref(false)
  const error = ref('')
  let request: Promise<NormalizedMenu[]> | null = null

  const visibleMenus = computed(() => menus.value.filter((menu) => {
    if (menu.hidden) return false
    return menu.children.length === 0 || visibleChildren(menu).length > 0
  }))

  async function load(force = false): Promise<NormalizedMenu[]> {
    if (loaded.value && !force) return menus.value
    if (request) return request
    loading.value = true
    error.value = ''
    request = listUserMenus()
      .then((result) => {
        menus.value = normalizeMenus(result)
        loaded.value = true
        return menus.value
      })
      .catch((cause: unknown) => {
        error.value = cause instanceof Error ? cause.message : '菜单加载失败'
        throw cause
      })
      .finally(() => {
        loading.value = false
        request = null
      })
    return request
  }

  function reset(): void {
    menus.value = []
    loaded.value = false
    loading.value = false
    error.value = ''
    request = null
  }

  return { menus, visibleMenus, loaded, loading, error, load, reset }
})
